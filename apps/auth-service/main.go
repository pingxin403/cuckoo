package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/apps/auth-service/gen/authpb"
	"github.com/pingxin403/cuckoo/apps/auth-service/service"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    getEnv("SERVICE_NAME", "auth-service"),
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
	obs.Logger().Info(ctx, "Starting auth-service",
		"service", "auth-service",
		"version", getEnv("SERVICE_VERSION", "1.0.0"),
	)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "9095"
	}

	// Get JWT secret from environment variable
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		obs.Logger().Error(ctx, "JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", port, "error", err)
		os.Exit(1)
	}

	// Create service
	svc := service.NewAuthServiceServer(jwtSecret, obs)
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
		obs.Logger().Info(ctx, "auth-service listening", "port", port)
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
