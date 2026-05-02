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
    fmt.Printf("[WORKER] Started cracking: %s\n", req.TargetHash)

    // Check lengths 1 through 5
    for length := 1; length <= 5; length++ {

        // This checks if the Manager sent a "STOP" signal (CancelFunc)
        if ctx.Err() != nil { 
            fmt.Printf("[WORKER] Stopping work on %s (cancelled by manager)\n", req.TargetHash)
            return &proto.TaskResponse{Found: false}, nil 
        }

        password := make([]byte, length)
        found := s.recursiveCrack(0, length, password, req.Alphabet, req)
        if found != "" {
            fmt.Printf(">>> [WORKER] SUCCESS: %s\n", found)
            return &proto.TaskResponse{Found: true, Password: found}, nil
        }
    }

    return &proto.TaskResponse{Found: false}, nil
}

func (s *WorkerServer) recursiveCrack(depth, maxDepth int, current []byte, alphabet string, req *proto.TaskRequest) string {
    if depth == maxDepth {
        if checkPassword(string(current), req.TargetHash) { return string(current) }
        return ""
    }

    for i := 0; i < len(alphabet); i++ {
        char := alphabet[i]
        
        // Ensure the character is correctly within the assigned range
        if depth == 0 {
            charStr := string(char)
            if charStr < req.StartRange || charStr > req.EndRange { 
                continue 
            }
        }

        current[depth] = char
        result := s.recursiveCrack(depth+1, maxDepth, current, alphabet, req)
        if result != "" { return result }
    }
    return ""
}
