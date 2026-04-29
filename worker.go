package main

import (
	"context"
	"fmt"

	"github.com/omar/distributed-cracker/proto"
)

// WorkerServer implements the CrackerService defined in the proto
type WorkerServer struct {
	proto.UnimplementedCrackerServiceServer
}

func (s *WorkerServer) ProcessTask(ctx context.Context, req *proto.TaskRequest) (*proto.TaskResponse, error) {
	fmt.Printf("Received Task: Crack %s in range %s-%s\n", req.TargetHash, req.StartRange, req.EndRange)

	// For now pretend to work
	return &proto.TaskResponse{
		Found:    false,
		Password: "",
	}, nil
}
