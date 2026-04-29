package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/omar/distributed-cracker/proto"
	"google.golang.org/grpc"
)

var crackedPassword string

func main() {
	hostname, _ := os.Hostname()
	config := memberlist.DefaultLocalConfig()
	// The timestamp here acts as our "seniority"
	config.Name = fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano()%1000000)

	// Check for a custom bind port (so we can run multiple on one machine)
	bindPort := os.Getenv("BIND_PORT")
	if bindPort != "" {
		var p int
		fmt.Sscanf(bindPort, "%d", &p)
		config.BindPort = p
	}

	list, err := memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	local := list.LocalNode()
	fmt.Printf("Node [%s] is alive at %s:%d\n", local.Name, local.Addr, local.Port)

	// Look for an environment variable called JOIN_ADDR
	joinAddr := os.Getenv("JOIN_ADDR")
	if joinAddr != "" {
		// Split multiple addresses if provided (comma separated)
		parts := strings.Split(joinAddr, ",")
		_, err := list.Join(parts)
		if err != nil {
			fmt.Printf("Failed to join cluster: %v\n", err)
		}
	}

	// Calculate a gRPC port based on the BIND_PORT to avoid conflicts
	grpcPort := 50051
	if bindPort != "" {
		fmt.Sscanf(bindPort, "%d", &grpcPort)
		grpcPort = grpcPort - 7946 + 50051 // Offset based on gossip port
	}

	// Start gRPC server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
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
			if crackedPassword != "" {
				fmt.Printf("--- TASK COMPLETE! Password is: %s ---\n", crackedPassword)
				time.Sleep(10 * time.Second)
				continue
			}

			// Get only the Workers (exclude the Manager)
			var workers []*memberlist.Node
			for _, m := range members {
				if m.Name != local.Name {
					workers = append(workers, m)
				}
			}

			numWorkers := len(workers)
			if numWorkers > 0 {
				fmt.Printf("ROLE: [ MANAGER ] | Splitting work between %d workers\n", numWorkers)

				alphabet := "abcdefghijklmnopqrstuvwxyz"
				chunkSize := len(alphabet) / numWorkers

				for i, w := range workers {
					// Divide the work into ranges of the alphabet based on the number of workers
					startIdx := i * chunkSize
					endIdx := (i + 1) * chunkSize
					if i == numWorkers-1 {
						endIdx = len(alphabet)
					} // Catch the remainder

					startRange := string(alphabet[startIdx])
					endRange := string(alphabet[endIdx-1])

					fmt.Printf(">>> Sending Range [%s-%s] to Worker %s\n", startRange, endRange, w.Name)

					// Dispatch in background
					go func(addr string, port int, s string, e string) {
						res := sendTask(addr, port, "900150983cd24fb0d6963f7d28e17f72", s, e)
						if res != "" {
							crackedPassword = res // 4. Update State
						}
					}(w.Addr.String(), int(w.Port), startRange, endRange)
				}
			}
		} else {
			fmt.Printf("ROLE: [ WORKER ]  | Manager is: %s\n", manager.Name)
		}

		time.Sleep(5 * time.Second)
	}
}

func sendTask(workerAddr string, workerPort int, targetHash, start, end string) string {
	// Calculate the worker's gRPC port based on their Gossip port
	// (Matching the logic that was used to start the server)
	grpcPort := workerPort - 7946 + 50051

	// Dial the worker
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", workerAddr, grpcPort), grpc.WithInsecure())
	if err != nil {
		return ""
	}
	defer conn.Close()

	client := proto.NewCrackerServiceClient(conn)

	// Send a test task
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := client.ProcessTask(ctx, &proto.TaskRequest{
		TargetHash: targetHash,
		StartRange: start,
		EndRange:   end,
	})

	if err == nil && resp.Found {
		return resp.Password
	}
	return ""
}
