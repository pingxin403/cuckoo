package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/authpb"
	"github.com/pingxin403/cuckoo/apps/auth-service/config"
	"github.com/pingxin403/cuckoo/apps/auth-service/service"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    cfg.Observability.ServiceName,
		ServiceVersion: cfg.Observability.ServiceVersion,
		Environment:    cfg.Observability.Environment,
		EnableMetrics:  cfg.Observability.EnableMetrics,
		MetricsPort:    cfg.Observability.MetricsPort,
		LogLevel:       cfg.Observability.LogLevel,
		LogFormat:      cfg.Observability.LogFormat,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Observability shutdown error: %v\n", err)
		}
	}()

	ctx := context.Background()
	obs.Logger().Info(ctx, "Starting auth-service",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
		"port", cfg.Server.Port,
	)

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// Create service
	svc := service.NewAuthServiceServer(cfg.JWT.Secret, obs)
	obs.Logger().Info(ctx, "Initialized auth service")

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	authpb.RegisterAuthServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "auth-service listening", "port", cfg.Server.Port)
		obs.Logger().Info(ctx, "Service ready to accept requests")
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal", "signal", sig.String())

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
		obs.Logger().Info(shutdownCtx, "Server stopped gracefully")
	case <-shutdownCtx.Done():
		obs.Logger().Warn(shutdownCtx, "Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	obs.Logger().Info(shutdownCtx, "auth-service shutdown complete")
}
