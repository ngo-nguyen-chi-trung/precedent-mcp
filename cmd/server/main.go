package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":"+env.GRPCPort.GetValue())
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterCapabilityServiceServer(srv, capabilityService)
	pb.RegisterToolServiceServer(srv, toolService)
	pb.RegisterResourceServiceServer(srv, resourceService)
	pb.RegisterSkillServiceServer(srv, skillService)

	log.Printf("gRPC listening on :%s", env.GRPCPort.GetValue())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		srv.GracefulStop()
	}()
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
