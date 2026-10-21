package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngo/precedent_mcp/gateway"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/clients"
	"github.com/ngo/precedent_mcp/internal/env"
)

func main() {
	if err := env.LoadEnv(); err != nil {
		log.Printf("warn: %v", err)
	}
	if err := gateway.CORSConfig().Validate(); err != nil {
		log.Fatalf("invalid %s %q: %v", env.AllowedOrigins, env.AllowedOrigins.GetValue(), err)
	}
	conn := clients.GetGRPCConnectionInstance()
	g := gateway.NewGateway(
		pb.NewCapabilityServiceClient(conn),
		pb.NewToolServiceClient(conn),
		pb.NewResourceServiceClient(conn),
		pb.NewSkillServiceClient(conn),
	)
	router, err := g.Router()
	if err != nil {
		log.Fatalf("gateway router: %v", err)
	}
	srv := &http.Server{Addr: ":" + env.HTTPPort.GetValue(), Handler: router}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("gateway shutdown: %v", err)
		}
	}()
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
