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

	"github.com/pingxin403/cuckoo/api/gen/go/authpb"
	"github.com/pingxin403/cuckoo/apps/auth-service/config"
	"github.com/pingxin403/cuckoo/apps/auth-service/service"
	"github.com/pingxin403/cuckoo/libs/health"
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

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      cfg.Observability.ServiceName,
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)
	obs.Logger().Info(ctx, "Initialized health checker")

	// Get HTTP port for health endpoints from environment variable or use default
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Create TCP listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// Create service
	svc := service.NewAuthServiceServer(cfg.JWT.Secret, obs)
	obs.Logger().Info(ctx, "Initialized auth service")

	// Note: auth-service is currently stateless with no database or Redis dependencies
	// If dependencies are added in the future, register health checks here:
	// hc.RegisterCheck(health.NewDatabaseCheck("database", db))
	// hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))

	// Start health checker
	if err := hc.Start(); err != nil {
		obs.Logger().Error(ctx, "Failed to start health checker", "error", err)
		os.Exit(1)
	}
	obs.Logger().Info(ctx, "Started health checker")

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	authpb.RegisterAuthServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Setup HTTP server for health endpoints
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", health.HealthzHandler(hc))
	httpMux.HandleFunc("/readyz", health.ReadyzHandler(hc))
	httpMux.HandleFunc("/health", health.HealthHandler(hc))
	obs.Logger().Info(ctx, "Registered health check endpoints")

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

	// Start server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "auth-service gRPC listening", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server for health endpoints in a goroutine
	go func() {
		obs.Logger().Info(ctx, "auth-service HTTP health server listening", "port", httpPort)
		obs.Logger().Info(ctx, "Service ready to accept requests")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal", "signal", sig.String())

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
	obs.Logger().Info(ctx, "Health checker stopped")

	obs.Logger().Info(shutdownCtx, "auth-service shutdown complete")
}
