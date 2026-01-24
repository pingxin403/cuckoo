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

	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "9093"
	}

	// Initialize Redis client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
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

	// Create gateway service with default config
	config := service.DefaultGatewayConfig()
	gateway := service.NewGatewayService(
		authClient,
		registryClient,
		imClient,
		redisClient,
		config,
	)

	// TODO: Start gateway service with Kafka config
	// kafkaConfig := service.KafkaConfig{...}
	// if err := gateway.Start(kafkaConfig); err != nil {
	//     log.Fatalf("Failed to start gateway service: %v", err)
	// }

	// Setup HTTP server with timeouts
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", gateway.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
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
		log.Printf("im-gateway-service listening on port %s", port)
		log.Println("WebSocket endpoint: /ws")
		log.Println("Health endpoint: /health")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown gateway service
	if err := gateway.Shutdown(shutdownCtx); err != nil {
		log.Printf("Gateway shutdown error: %v", err)
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("im-gateway-service shutdown complete")
}
