package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/service"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "9092"
	}

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Initialize storage
	store := storage.NewMemoryStore()
	log.Println("Initialized in-memory store")

	// Create service
	svc := service.NewUshortenerUserviceServiceServer(store)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	shortener_servicepb.RegisterUshortenerUserviceServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("shortener-service listening on port %s", port)
		log.Println("Service ready to accept requests")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new connections and wait for existing ones to finish
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Server stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	log.Println("shortener-service shutdown complete")
}
