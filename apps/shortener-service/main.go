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
	"github.com/pingxin403/cuckoo/apps/shortener-service/logger"
	"github.com/pingxin403/cuckoo/apps/shortener-service/service"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	isDev := os.Getenv("ENVIRONMENT") != "production"
	if err := logger.InitLogger(isDev); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("Starting shortener-service")

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
		logger.Log.Fatal("Failed to listen", zap.String("port", grpcPort), zap.Error(err))
	}

	// Initialize MySQL storage
	store, err := storage.NewMySQLStore()
	if err != nil {
		logger.Log.Fatal("Failed to initialize MySQL store", zap.Error(err))
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Log.Error("Error closing store", zap.Error(err))
		}
	}()
	logger.Log.Info("Initialized MySQL store")

	// Initialize ID generator
	idGenerator := idgen.NewRandomIDGenerator(store)
	logger.Log.Info("Initialized ID generator")

	// Initialize URL validator
	urlValidator := service.NewURLValidator()
	logger.Log.Info("Initialized URL validator")

	// Initialize L1 cache (Ristretto)
	l1Cache, err := cache.NewL1Cache()
	if err != nil {
		logger.Log.Fatal("Failed to initialize L1 cache", zap.Error(err))
	}
	logger.Log.Info("Initialized L1 cache")

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
			logger.Log.Warn("Failed to initialize L2 cache (Redis), continuing without Redis", zap.Error(err))
			l2Cache = nil
		} else {
			logger.Log.Info("Initialized L2 cache (Redis)")
		}
	} else {
		logger.Log.Info("Redis not configured, running without L2 cache")
	}

	// Initialize cache manager
	cacheManager := cache.NewCacheManager(l1Cache, l2Cache, &cacheStorageAdapter{store: store})
	logger.Log.Info("Initialized cache manager")

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
		analyticsWriter = analytics.NewAnalyticsWriter(analyticsConfig)
		logger.Log.Info("Initialized analytics writer", zap.String("brokers", kafkaBrokers))
	} else {
		logger.Log.Info("Kafka not configured, running without analytics")
	}

	// Create gRPC service
	svc := service.NewShortenerServiceImpl(store, idGenerator, urlValidator, cacheManager)

	// Initialize rate limiter (100 requests per minute per IP)
	// Requirements: 6.1, 6.2
	rateLimiter := service.NewRateLimiter(100)
	logger.Log.Info("Initialized rate limiter", zap.Int("requests_per_minute", 100))

	// Start rate limiter cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rateLimiter.StartCleanup(ctx)

	// Create HTTP redirect handler
	redirectHandler := service.NewRedirectHandler(cacheManager, store, analyticsWriter)
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

	// Get metrics port from environment variable or use default
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}

	// Create metrics server
	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", metricsPort),
		Handler:      promhttp.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC server in a goroutine
	go func() {
		logger.Log.Info("gRPC server listening", zap.String("port", grpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		logger.Log.Info("HTTP redirect server listening", zap.String("port", httpPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	// Start metrics server in a goroutine
	go func() {
		logger.Log.Info("Metrics server listening", zap.String("port", metricsPort))
		logger.Log.Info("Service ready to accept requests")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to serve metrics", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Log.Info("Received shutdown signal, initiating graceful shutdown", zap.String("signal", sig.String()))

	// Cancel rate limiter cleanup context
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Metrics server shutdown error", zap.Error(err))
	}

	// Stop gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Log.Info("Server stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Log.Warn("Shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	// Close analytics writer if initialized
	if analyticsWriter != nil {
		if err := analyticsWriter.Close(); err != nil {
			logger.Log.Error("Analytics writer shutdown error", zap.Error(err))
		}
	}

	logger.Log.Info("shortener-service shutdown complete")
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
