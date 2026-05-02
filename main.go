package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"
	"os/signal" 
    "syscall"   

	"github.com/hashicorp/memberlist"
	"github.com/omar/distributed-cracker/proto"
	"google.golang.org/grpc"
)

type ManagerNode struct {
    CurrentHash   string
    FoundPassword string
    IsProcessing  bool
    CancelFunc    context.CancelFunc // Add this
}

var globalManager = &ManagerNode{}
var apiStarted = false

func main() {
	hostname, _ := os.Hostname()
	config := memberlist.DefaultLocalConfig()
	role := os.Getenv("ROLE")

	globalManager.FoundPassword = loadResult()

	if role == "MANAGER" {
		// Giving it a prefix of 00 ensures it is always first in the sorted list
		config.Name = fmt.Sprintf("00-manager-%s", hostname)
	} else {
		config.Name = fmt.Sprintf("worker-%s-%d", hostname, time.Now().UnixNano()%1000)
	}

	// Check for a custom bind port
	bindPort := os.Getenv("BIND_PORT")
	if bindPort != "" {
		var p int
		fmt.Sscanf(bindPort, "%d", &p)
		config.BindPort = p
	}

	list, err := memberlist.Create(config)
    if err != nil { panic(err) }

    // Shutdown logic
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Println("\n[SHUTDOWN] Leaving cluster and stopping gRPC...")
        list.Leave(time.Second * 5)
        list.Shutdown()
        os.Exit(0)
    }()

	local := list.LocalNode()
	fmt.Printf("Node [%s] is alive at %s:%d\n", local.Name, local.Addr, local.Port)

	// Look for an environment variable called JOIN_ADDR
	joinAddr := os.Getenv("JOIN_ADDR")
	if joinAddr != "" {
		// Split multiple addresses if provided
		parts := strings.Split(joinAddr, ",")
		_, err := list.Join(parts)
		if err != nil {
			fmt.Printf("Failed to join cluster: %v\n", err)
		}
	}

	// Calculate a gRPC port based on the BIND_PORT to avoid conflicts
	grpcPort := 50051 // Default gRPC port
	apiPort := 8080   // Static port for all containers

	// Start gRPC server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			fmt.Printf("gRPC failed to listen: %v\n", err)
			return
		}
		s := grpc.NewServer()
		proto.RegisterCrackerServiceServer(s, &WorkerServer{})
		fmt.Printf("gRPC Server listening on port %d\n", grpcPort)
		if err := s.Serve(lis); err != nil {
			fmt.Printf("gRPC failed to serve: %v\n", err)
		}
	}()

	fmt.Println("Waiting for cluster to stabilize...")
	time.Sleep(5 * time.Second)
	for {
		// Get all members and sort them alphabetically by Name
		members := list.Members()
		sort.Slice(members, func(i, j int) bool {
			return members[i].Name < members[j].Name
		})

		// The first member in the sorted list is the Manager
		manager := members[0]

		fmt.Println("--- Cluster Status ---")
		if manager.Name == local.Name {
			if !apiStarted {
				fmt.Printf("ROLE: [ LEADER ] | Starting API on :%d\n", apiPort)
				go StartAPIServer(globalManager, apiPort)
				apiStarted = true
			}
			// Only proceed if the API has given a hash and it's not solved yet
			// Only start work if we have a hash AND we aren't already working on it
			if globalManager.CurrentHash != "" && globalManager.FoundPassword == "" && !globalManager.IsProcessing {

				globalManager.IsProcessing = true
				
				// Create a Master Context for this specific hash job
				// This allows to kill all worker requests at once
				ctx, cancel := context.WithCancel(context.Background())
				globalManager.CancelFunc = cancel 

				workers := getWorkers(members, local.Name)
				numWorkers := len(workers)

				if numWorkers > 0 {
					fmt.Printf("ROLE: [ MANAGER ] | Dispatching task to %d workers...\n", numWorkers)
					alphabet := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
					totalChars := len(alphabet)
					chunkSize := totalChars / numWorkers

					for i, w := range workers {
						startIdx := i * chunkSize
						endIdx := (i + 1) * chunkSize
						if i == numWorkers-1 {
							endIdx = totalChars
						}

						startRange := string(alphabet[startIdx])
						endRange := string(alphabet[endIdx-1])

						fmt.Printf("[MANAGER] Worker %d (%s) assigned range: %s to %s\n", i, w.Addr.String(), startRange, endRange)

						// Launch workers
						go func(addr string, s, e, alph string) {
							res := sendTask(ctx, addr, globalManager.CurrentHash, s, e, alph)
							
							if res != "" {
								fmt.Printf("!!! PASSWORD FOUND: %s\n", res)
								globalManager.FoundPassword = res
								
								// STOP ALL OTHER WORKERS
								globalManager.CancelFunc() 

								saveResult(res) 
							}
						}(w.Addr.String(), startRange, endRange, alphabet)
					}
				}
			} else if globalManager.FoundPassword != "" {
				fmt.Printf("--- TASK COMPLETE! Password is: %s ---\n", globalManager.FoundPassword)
			} else {
				fmt.Println("ROLE: [ MANAGER ] | Waiting for hash via API...")
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func sendTask(ctx context.Context, workerAddr string, targetHash, start, end, alphabet string) string {
    grpcPort := 50051
    // Use DialContext for better integration with the cancellation signal
    conn, err := grpc.Dial(fmt.Sprintf("%s:%d", workerAddr, grpcPort), grpc.WithInsecure())
    if err != nil {
        return ""
    }
    defer conn.Close()

    client := proto.NewCrackerServiceClient(conn)

    // Use the ctx passed from the manager. 
    // If globalManager.CancelFunc() is called, this ProcessTask will return immediately.
    resp, err := client.ProcessTask(ctx, &proto.TaskRequest{
        TargetHash: targetHash,
        StartRange: start,
        EndRange:   end,
        Alphabet:   alphabet,
    })

    if err != nil {
        // Only print error if it wasn't a intentional cancellation
        if ctx.Err() == nil {
            fmt.Printf("[ERROR] gRPC call failed: %v\n", err)
        }
        return ""
    }

    if resp.Found {
        return resp.Password
    }
    return ""
}

// getWorkers filters the memberlist to return only nodes that are NOT the manager
func getWorkers(members []*memberlist.Node, localName string) []*memberlist.Node {
	var workers []*memberlist.Node
	for _, m := range members {
		// If the node name doesn't match the manager's name, it's a worker
		if m.Name != localName {
			workers = append(workers, m)
		}
	}
	return workers
}

func saveResult(password string) {
    // Use the full path inside the container to avoid confusion
    filename := "/root/result.json" 
    err := os.WriteFile(filename, []byte(password), 0644)
    if err != nil {
        fmt.Printf("[ERROR] Failed to write to %s: %v\n", filename, err)
    } else {
        fmt.Printf("[SUCCESS] Password '%s' written to %s\n", password, filename)
    }
}

func loadResult() string {
    // Use the full path inside the container to avoid confusion
    filename := "/root/result.json"
    data, err := os.ReadFile(filename)
    if err != nil { return "" }
    return string(data)
}