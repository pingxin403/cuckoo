package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-gateway-service/config"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/metrics"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = redisClient.Close() }()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Connected to Redis")
	}

	// TODO: Initialize actual clients (auth, registry, IM service)
	// For now, using nil clients - these should be replaced with real implementations
	var authClient service.AuthServiceClient
	var registryClient service.RegistryClient
	var imClient service.IMServiceClient

	// Initialize observability with OpenTelemetry metrics
	obs, err := observability.New(observability.Config{
		ServiceName:         cfg.Observability.ServiceName,
		ServiceVersion:      cfg.Observability.ServiceVersion,
		Environment:         cfg.Observability.Environment,
		EnableMetrics:       cfg.Observability.EnableMetrics,
		UseOTelMetrics:      true,                                     // Use OpenTelemetry metrics
		PrometheusEnabled:   true,                                     // Enable Prometheus exporter
		MetricsPort:         cfg.Observability.MetricsPort,            // Separate port for metrics
		OTLPMetricsEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), // OTLP endpoint for metrics
		OTLPInsecure:        true,                                     // Use insecure connection for development
		EnableTracing:       false,
		LogLevel:            cfg.Observability.LogLevel,
		LogFormat:           cfg.Observability.LogFormat,
	})
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := obs.Shutdown(shutdownCtx); err != nil {
			log.Printf("Observability shutdown error: %v", err)
		}
	}()

	obs.Logger().Info(ctx, "Observability initialized",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
		"metrics_port", cfg.Observability.MetricsPort,
		"otel_metrics", true,
	)

	// Initialize health checker
	healthChecker := health.NewHealthChecker(health.Config{
		ServiceName:      cfg.Observability.ServiceName,
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Create metrics instance with observability
	gatewayMetrics := metrics.NewMetrics(obs)

	// Create gateway service with default config
	gatewayConfig := service.DefaultGatewayConfig()
	gateway := service.NewGatewayService(
		authClient,
		registryClient,
		imClient,
		redisClient,
		gatewayConfig,
	)

	// Register health checks
	// 1. Redis health check (critical)
	healthChecker.RegisterCheck(health.NewRedisCheck("redis", redisClient))

	// 2. Downstream im-service health check (critical)
	// Get im-service address from config
	imServiceHealthURL := fmt.Sprintf("http://%s/healthz", cfg.ServiceDiscovery.IMServiceAddr)
	healthChecker.RegisterCheck(health.NewHTTPCheck("im-service", imServiceHealthURL, true))

	// 3. WebSocket connection health check (non-critical)
	healthChecker.RegisterCheck(NewWebSocketHealthCheck(gateway))

	// Start health checker
	if err := healthChecker.Start(); err != nil {
		log.Fatalf("Failed to start health checker: %v", err)
	}
	defer healthChecker.Stop()

	obs.Logger().Info(ctx, "Health checker started",
		"checks", []string{"redis", "im-service", "websocket-connections"},
	)

	// TODO: Integrate metrics with gateway service
	// gateway.SetMetrics(gatewayMetrics)

	// TODO: Start gateway service with Kafka config
	// kafkaConfig := service.KafkaConfig{...}
	// if err := gateway.Start(kafkaConfig); err != nil {
	//     log.Fatalf("Failed to start gateway service: %v", err)
	// }

	// Setup HTTP server with timeouts
	mux := http.NewServeMux()
	
	// Health endpoints (no middleware - always accessible)
	mux.HandleFunc("/healthz", health.HealthzHandler(healthChecker))
	mux.HandleFunc("/readyz", health.ReadyzHandler(healthChecker))
	mux.HandleFunc("/health", health.HealthHandler(healthChecker))
	
	// WebSocket endpoint with readiness middleware
	wsHandler := health.ReadinessMiddleware(healthChecker)(http.HandlerFunc(gateway.HandleWebSocket))
	mux.Handle("/ws", wsHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "Starting HTTP server",
			"port", cfg.Server.HTTPPort,
			"websocket_endpoint", "/ws",
			"health_endpoints", []string{"/healthz", "/readyz", "/health"},
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().Error(ctx, "HTTP server error", "error", err)
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal", "signal", sig.String())

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown gateway service
	if err := gateway.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "Gateway shutdown error", "error", err)
	}

	// Shutdown metrics
	if err := gatewayMetrics.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "Metrics shutdown error", "error", err)
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(shutdownCtx, "HTTP server shutdown error", "error", err)
	}

	obs.Logger().Info(shutdownCtx, "Shutdown complete")
}
