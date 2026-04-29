package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/omar/distributed-cracker/proto"
)

// WorkerServer implements the CrackerService defined in the proto
type WorkerServer struct {
	proto.UnimplementedCrackerServiceServer
}

// Helper to check a password against a hash
func checkPassword(password string, targetHash string) bool {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:]) == targetHash
}

func (s *WorkerServer) ProcessTask(ctx context.Context, req *proto.TaskRequest) (*proto.TaskResponse, error) {
	fmt.Printf("[WORKER] Cracking hash: %s\n", req.TargetHash)

	// Brute force 5-letter passwords within the range
	characters := "abcdefghijklmnopqrstuvwxyz"
	
	for i := 0; i < len(characters); i++ {
		c1 := string(characters[i])
		// Check if the first letter is within the manager's assigned range
		if c1 < req.StartRange || c1 > req.EndRange {
			continue
		}

		for j := 0; j < len(characters); j++ {
			for k := 0; k < len(characters); k++ {
				testPass := c1 + string(characters[j]) + string(characters[k])
				
				if checkPassword(testPass, req.TargetHash) {
					fmt.Printf(">>> [WORKER] SUCCESS! Found: %s\n", testPass)
					return &proto.TaskResponse{
						Found:    true,
						Password: testPass,
					}, nil
				}
			}
		}
	}

	return &proto.TaskResponse{Found: false}, nil
}
