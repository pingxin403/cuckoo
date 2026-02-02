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

	"github.com/pingxin403/cuckoo/api/gen/go/shortenerpb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/analytics"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/config"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/service"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
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
	obs.Logger().Info(ctx, "Starting shortener-service",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
	)

	// Initialize health checker with default config
	healthConfig := health.DefaultConfig(cfg.Observability.ServiceName)
	healthConfig.CheckInterval = 5 * time.Second
	healthConfig.DefaultTimeout = 100 * time.Millisecond
	healthConfig.FailureThreshold = 3
	hc := health.NewHealthChecker(healthConfig, obs)
	obs.Logger().Info(ctx, "Initialized health checker")

	// Create TCP listener for gRPC
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GRPCPort))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", cfg.Server.GRPCPort, "error", err)
		os.Exit(1)
	}

	// Initialize MySQL storage
	store, err := storage.NewMySQLStore(storage.MySQLConfig{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		Database:     cfg.Database.Database,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
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
	if cfg.Redis != nil && cfg.Redis.Addr != "" {
		l2Config := cache.L2CacheConfig{
			Addrs:    []string{cfg.Redis.Addr},
			PoolSize: cfg.Redis.PoolSize,
		}
		l2Cache, err = cache.NewL2Cache(l2Config)
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

	// Register health checks
	// Database health check (MySQL) - critical
	hc.RegisterCheck(health.NewDatabaseCheck("database", store.DB()))
	obs.Logger().Info(ctx, "Registered database health check")

	// Redis health check - critical if Redis is configured
	if l2Cache != nil {
		hc.RegisterCheck(health.NewRedisCheck("redis", l2Cache.Client()))
		obs.Logger().Info(ctx, "Registered Redis health check")
	}

	// Start health checker
	if err := hc.Start(); err != nil {
		obs.Logger().Error(ctx, "Failed to start health checker", "error", err)
		os.Exit(1)
	}
	obs.Logger().Info(ctx, "Started health checker")

	// Initialize analytics writer (Kafka) - optional
	// Requirements: 7.1, 7.2
	var analyticsWriter *analytics.AnalyticsWriter
	if cfg.Kafka != nil && len(cfg.Kafka.Brokers) > 0 {
		analyticsConfig := analytics.Config{
			KafkaBrokers: cfg.Kafka.Brokers,
			Topic:        cfg.Kafka.Topic,
			NumWorkers:   4,
			BufferSize:   10000,
		}
		analyticsWriter = analytics.NewAnalyticsWriter(analyticsConfig, obs)
		obs.Logger().Info(ctx, "Initialized analytics writer", "brokers", cfg.Kafka.Brokers)
	} else {
		obs.Logger().Info(ctx, "Kafka not configured, running without analytics")
	}

	// Create gRPC service
	svc := service.NewShortenerServiceImpl(store, idGenerator, urlValidator, cacheManager, obs)

	// Initialize rate limiter (100 requests per minute per IP)
	// Requirements: 6.1, 6.2
	rateLimiter := service.NewRateLimiter(cfg.RateLimiter.RequestsPerMinute)
	obs.Logger().Info(ctx, "Initialized rate limiter", "requests_per_minute", cfg.RateLimiter.RequestsPerMinute)

	// Start rate limiter cleanup goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rateLimiter.StartCleanup(ctx)

	// Create HTTP redirect handler
	redirectHandler := service.NewRedirectHandler(cacheManager, store, analyticsWriter, obs)
	httpRouter := redirectHandler.SetupRouter()

	// Create main HTTP mux to handle both health endpoints and application routes
	mainMux := http.NewServeMux()

	// Register health check endpoints (without readiness middleware)
	mainMux.HandleFunc("/healthz", health.HealthzHandler(hc))
	mainMux.HandleFunc("/readyz", health.ReadyzHandler(hc))
	mainMux.HandleFunc("/health", health.HealthHandler(hc))
	obs.Logger().Info(ctx, "Registered health check endpoints")

	// Wrap redirect handler with readiness middleware
	readinessWrappedHandler := health.ReadinessMiddleware(hc)(httpRouter)

	// Mount the redirect handler at root (it will handle all other routes)
	mainMux.Handle("/", readinessWrappedHandler)

	// Wrap HTTP router with rate limiter middleware
	// Requirements: 6.1, 6.2, 6.5
	httpRouterWithRateLimit := rateLimiter.HTTPMiddleware(mainMux)

	// Create gRPC server with rate limiter interceptor
	// Requirements: 6.1, 6.2, 6.5
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(rateLimiter.UnaryServerInterceptor()),
	)

	// Register gRPC service
	shortenerpb.RegisterShortenerServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTPPort),
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
		obs.Logger().Info(ctx, "gRPC server listening", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		obs.Logger().Info(ctx, "HTTP redirect server listening", "port", cfg.Server.HTTPPort)
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

	// Stop health checker
	hc.Stop()
	obs.Logger().Info(ctx, "Health checker stopped")

	obs.Logger().Info(ctx, "shortener-service shutdown complete")
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
