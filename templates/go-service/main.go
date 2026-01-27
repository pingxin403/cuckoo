package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"{{MODULE_PATH}}/gen/{{PROTO_PACKAGE}}"
	"{{MODULE_PATH}}/service"
	"{{MODULE_PATH}}/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:       "{{SERVICE_NAME}}",
		ServiceVersion:    getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:       getEnv("DEPLOYMENT_ENVIRONMENT", "development"),
		EnableMetrics:     true,
		MetricsPort:       9090,
		PrometheusEnabled: true,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize observability: %v\n", err)
		// Continue with no-op observability - service should still work
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if obs != nil {
			if err := obs.Shutdown(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Observability shutdown error: %v\n", err)
			}
		}
	}()

	// Get port from environment variable or use default
	port := getEnv("PORT", "{{GRPC_PORT}}")

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		obs.Logger().Error(context.Background(), "Failed to listen",
			"port", port,
			"error", err,
		)
		os.Exit(1)
	}

	// Initialize storage
	store := storage.NewMemoryStore()
	obs.Logger().Info(context.Background(), "Initialized in-memory store")

	// Create service with observability
	svc := service.New{{ServiceName}}ServiceServer(store, obs)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	{{PROTO_PACKAGE}}.Register{{ServiceName}}ServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		obs.Logger().Info(context.Background(), "Service started",
			"service", "{{SERVICE_NAME}}",
			"port", port,
		)
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(context.Background(), "Failed to serve",
				"error", err,
			)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(context.Background(), "Received shutdown signal",
		"signal", sig.String(),
	)

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
		obs.Logger().Info(context.Background(), "Server stopped gracefully")
	case <-shutdownCtx.Done():
		obs.Logger().Warn(context.Background(), "Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	obs.Logger().Info(context.Background(), "Service shutdown complete",
		"service", "{{SERVICE_NAME}}",
	)
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
