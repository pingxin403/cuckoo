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

	"github.com/pingxin403/cuckoo/apps/shortener-service/analytics"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/service"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    getEnv("SERVICE_NAME", "shortener-service"),
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
	obs.Logger().Info(ctx, "Starting shortener-service",
		"service", "shortener-service",
		"version", getEnv("SERVICE_VERSION", "1.0.0"),
	)

	// Get gRPC port from environment variable or use default
	grpcPort := os.Getenv("PORT")
	if grpcPort == "" {
		grpcPort = "9092"
	}

	// Get HTTP port from environment variable or use default
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	// Create TCP listener for gRPC
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	// Initialize MySQL storage
	store, err := storage.NewMySQLStore()
	if err != nil {
		obs.Logger().Error(ctx, "Failed to initialize MySQL store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			obs.Logger().Error(ctx, "Error closing store", "error", err)
		}
	}()
	obs.Logger().Info(ctx, "Initialized MySQL store")

	// Initialize ID generator
	idGenerator := idgen.NewRandomIDGenerator(store)
	obs.Logger().Info(ctx, "Initialized ID generator")

	// Initialize URL validator
	urlValidator := service.NewURLValidator()
	obs.Logger().Info(ctx, "Initialized URL validator")

	// Initialize L1 cache (Ristretto)
	l1Cache, err := cache.NewL1Cache()
	if err != nil {
		obs.Logger().Error(ctx, "Failed to initialize L1 cache", "error", err)
		os.Exit(1)
	}
	obs.Logger().Info(ctx, "Initialized L1 cache")

	// Initialize L2 cache (Redis) - optional, can be nil
	var l2Cache *cache.L2Cache
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		config := cache.L2CacheConfig{
			Addrs:    []string{redisAddr},
			PoolSize: 10,
		}
		l2Cache, err = cache.NewL2Cache(config)
		if err != nil {
			obs.Logger().Warn(ctx, "Failed to initialize L2 cache (Redis), continuing without Redis", "error", err)
			l2Cache = nil
		} else {
			obs.Logger().Info(ctx, "Initialized L2 cache (Redis)")
		}
	} else {
		obs.Logger().Info(ctx, "Redis not configured, running without L2 cache")
	}

	// Initialize cache manager
	cacheManager := cache.NewCacheManager(l1Cache, l2Cache, &cacheStorageAdapter{store: store}, obs)
	obs.Logger().Info(ctx, "Initialized cache manager")

	// Initialize analytics writer (Kafka) - optional
	// Requirements: 7.1, 7.2
	var analyticsWriter *analytics.AnalyticsWriter
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers != "" {
		analyticsConfig := analytics.Config{
			KafkaBrokers: []string{kafkaBrokers},
			Topic:        "url-clicks",
			NumWorkers:   4,
			BufferSize:   10000,
		}
		analyticsWriter = analytics.NewAnalyticsWriter(analyticsConfig, obs)
		obs.Logger().Info(ctx, "Initialized analytics writer", "brokers", kafkaBrokers)
	} else {
		obs.Logger().Info(ctx, "Kafka not configured, running without analytics")
	}

	// Create gRPC service
	svc := service.NewShortenerServiceImpl(store, idGenerator, urlValidator, cacheManager, obs)

	// Initialize rate limiter (100 requests per minute per IP)
	// Requirements: 6.1, 6.2
	rateLimiter := service.NewRateLimiter(100)
	obs.Logger().Info(ctx, "Initialized rate limiter", "requests_per_minute", 100)

	// Start rate limiter cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rateLimiter.StartCleanup(ctx)

	// Create HTTP redirect handler
	redirectHandler := service.NewRedirectHandler(cacheManager, store, analyticsWriter, obs)
	httpRouter := redirectHandler.SetupRouter()

	// Wrap HTTP router with rate limiter middleware
	// Requirements: 6.1, 6.2, 6.5
	httpRouterWithRateLimit := rateLimiter.HTTPMiddleware(httpRouter)

	// Create gRPC server with rate limiter interceptor
	// Requirements: 6.1, 6.2, 6.5
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(rateLimiter.UnaryServerInterceptor()),
	)

	// Register gRPC service
	shortener_servicepb.RegisterShortenerServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      httpRouterWithRateLimit,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "gRPC server listening", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "HTTP redirect server listening", "port", httpPort)
		obs.Logger().Info(ctx, "Service ready to accept requests")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			obs.Logger().Error(ctx, "Failed to serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	obs.Logger().Info(ctx, "Received shutdown signal, initiating graceful shutdown", "signal", sig.String())

	// Cancel rate limiter cleanup context
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		obs.Logger().Error(ctx, "HTTP server shutdown error", "error", err)
	}

	// Stop gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		obs.Logger().Info(ctx, "Server stopped gracefully")
	case <-shutdownCtx.Done():
		obs.Logger().Warn(ctx, "Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	// Close analytics writer if initialized
	if analyticsWriter != nil {
		if err := analyticsWriter.Close(); err != nil {
			obs.Logger().Error(ctx, "Analytics writer shutdown error", "error", err)
		}
	}

	obs.Logger().Info(ctx, "shortener-service shutdown complete")
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

// cacheStorageAdapter adapts storage.Storage to cache.Storage interface
type cacheStorageAdapter struct {
	store storage.Storage
}

func (a *cacheStorageAdapter) Get(ctx context.Context, shortCode string) (*cache.StorageMapping, error) {
	mapping, err := a.store.Get(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	return &cache.StorageMapping{
		ShortCode: mapping.ShortCode,
		LongURL:   mapping.LongURL,
		CreatedAt: mapping.CreatedAt,
		ExpiresAt: mapping.ExpiresAt,
		CreatorIP: mapping.CreatorIP,
	}, nil
}

func (a *cacheStorageAdapter) Exists(ctx context.Context, shortCode string) (bool, error) {
	return a.store.Exists(ctx, shortCode)
}
