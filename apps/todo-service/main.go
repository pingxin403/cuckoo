package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/todopb"
	"github.com/pingxin403/cuckoo/apps/todo-service/service"
	"github.com/pingxin403/cuckoo/apps/todo-service/storage"
	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    getEnv("SERVICE_NAME", "todo-service"),
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("DEPLOYMENT_ENVIRONMENT", "development"),
		EnableMetrics:  getEnvBool("ENABLE_METRICS", true),
		MetricsPort:    getEnvInt("METRICS_PORT", 9090),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "json"),
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
	obs.Logger().Info(ctx, "Starting todo-service",
		"service", "todo-service",
		"version", getEnv("SERVICE_VERSION", "1.0.0"),
	)

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      getEnv("SERVICE_NAME", "todo-service"),
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)
	obs.Logger().Info(ctx, "Initialized health checker")

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "9091"
	}

	// Get HTTP port from environment variable or use default
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	// Initialize storage
	store := storage.NewMemoryStore()
	obs.Logger().Info(ctx, "Initialized in-memory TODO store")

	// Note: todo-service uses in-memory storage, so no database health check needed
	// Only liveness checks will be performed
	
	// Start health checker
	if err := hc.Start(); err != nil {
		obs.Logger().Error(ctx, "Failed to start health checker", "error", err)
		os.Exit(1)
	}
	obs.Logger().Info(ctx, "Started health checker")

	// Create TODO service
	todoService := service.NewTodoServiceServer(store, obs)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register TODO service
	todopb.RegisterTodoServiceServer(grpcServer, todoService)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Create HTTP server for health endpoints
	httpMux := http.NewServeMux()
	
	// Register health check endpoints (without readiness middleware)
	httpMux.HandleFunc("/healthz", health.HealthzHandler(hc))
	httpMux.HandleFunc("/readyz", health.ReadyzHandler(hc))
	httpMux.HandleFunc("/health", health.HealthHandler(hc))
	obs.Logger().Info(ctx, "Registered health check endpoints")

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      httpMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "gRPC server listening", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "HTTP health server listening", "port", httpPort)
		obs.Logger().Info(ctx, "Service ready to accept requests")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal, initiating graceful shutdown", "signal", sig.String())

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(ctx, "HTTP server shutdown error", "error", err)
	}

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

	// Stop health checker
	hc.Stop()
	obs.Logger().Info(shutdownCtx, "Health checker stopped")

	obs.Logger().Info(shutdownCtx, "todo-service shutdown complete")
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
