package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"context"
	"net"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"github.com/omar/distributed-cracker/proto"
)

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
			fmt.Printf("ROLE: [ MANAGER ] | Nodes in cluster: %d\n", len(members))
			
			// If there are other nodes, then send them a test task
			for _, m := range members {
				if m.Name != local.Name {
					fmt.Printf(">>> Dispatching task to Worker: %s\n", m.Name)
					go sendTask(m.Addr.String(), int(m.Port), "5d41402abc4b2a76b9719d911017c592") // this is md5 for 'hello'
				}
			}
		} else {
			fmt.Printf("ROLE: [ WORKER ]  | Manager is: %s\n", manager.Name)
		}

		time.Sleep(5 * time.Second)
	}
}

func sendTask(workerAddr string, workerPort int, targetHash string) {
	// Calculate the worker's gRPC port based on their Gossip port
	// (Matching the logic that was used to start the server)
	grpcPort := workerPort - 7946 + 50051

	// Dial the worker
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", workerAddr, grpcPort), grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Could not connect to worker %s: %v\n", workerAddr, err)
		return
	}
	defer conn.Close()

	client := proto.NewCrackerServiceClient(conn)

	// Send a test task
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.ProcessTask(ctx, &proto.TaskRequest{
		TargetHash: targetHash,
		StartRange: "a",
		EndRange:   "z",
	})

	if err != nil {
		fmt.Printf("Failed to send task to %s: %v\n", workerAddr, err)
	} else {
		fmt.Printf("Worker response: Found=%v, Password=%s\n", resp.Found, resp.Password)
	}
}