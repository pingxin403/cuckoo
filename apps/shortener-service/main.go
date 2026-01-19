package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/service"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
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
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	// Initialize MySQL storage
	store, err := storage.NewMySQLStore()
	if err != nil {
		log.Fatalf("Failed to initialize MySQL store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Error closing store: %v", err)
		}
	}()
	log.Println("Initialized MySQL store")

	// Initialize ID generator
	idGenerator := idgen.NewRandomIDGenerator(store)
	log.Println("Initialized ID generator")

	// Initialize URL validator
	urlValidator := service.NewURLValidator()
	log.Println("Initialized URL validator")

	// Initialize L1 cache (Ristretto)
	l1Cache, err := cache.NewL1Cache()
	if err != nil {
		log.Fatalf("Failed to initialize L1 cache: %v", err)
	}
	log.Println("Initialized L1 cache")

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
			log.Printf("Warning: Failed to initialize L2 cache (Redis): %v. Continuing without Redis.", err)
			l2Cache = nil
		} else {
			log.Println("Initialized L2 cache (Redis)")
		}
	} else {
		log.Println("Redis not configured, running without L2 cache")
	}

	// Initialize cache manager
	cacheManager := cache.NewCacheManager(l1Cache, l2Cache, &cacheStorageAdapter{store: store})
	log.Println("Initialized cache manager")

	// Create gRPC service
	svc := service.NewShortenerServiceImpl(store, idGenerator, urlValidator, cacheManager)

	// Create HTTP redirect handler
	redirectHandler := service.NewRedirectHandler(cacheManager, store)
	httpRouter := redirectHandler.SetupRouter()

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register gRPC service
	shortener_servicepb.RegisterShortenerServiceServer(grpcServer, svc)

	// Register reflection service for debugging (e.g., with grpcurl)
	reflection.Register(grpcServer)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      httpRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("gRPC server listening on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP redirect server listening on port %s", httpPort)
		log.Println("Service ready to accept requests")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop gRPC server
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
