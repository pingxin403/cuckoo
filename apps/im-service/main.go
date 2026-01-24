package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"github.com/pingxin403/cuckoo/apps/im-service/worker"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting IM Service...")

	// Load configuration from environment
	config := loadConfig()

	// Initialize shared dependencies
	log.Println("Initializing shared dependencies...")

	// Create storage
	store, err := storage.NewOfflineStore(storage.Config{
		DSN:             config.DatabaseDSN,
		MaxOpenConns:    config.DBMaxOpenConns,
		MaxIdleConns:    config.DBMaxIdleConns,
		ConnMaxLifetime: config.DBConnMaxLifetime,
	})
	if err != nil {
		log.Fatalf("Failed to create offline store: %v", err)
	}
	defer store.Close()
	log.Println("✓ Connected to database")

	// Create dedup service
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr:     config.RedisAddr,
		RedisPassword: config.RedisPassword,
		RedisDB:       config.RedisDB,
		TTL:           config.MessageTTL,
	})
	defer dedupService.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := dedupService.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	cancel()
	log.Println("✓ Connected to Redis")

	// Start gRPC server for message routing
	grpcServer := grpc.NewServer()
	// TODO: Register IM Service gRPC handlers here when Task 9 is implemented
	// pb.RegisterIMServiceServer(grpcServer, imService)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", config.GRPCPort, err)
	}

	go func() {
		log.Printf("✓ gRPC server listening on port %d", config.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start offline worker (background component)
	var offlineWorker *worker.OfflineWorker
	if config.OfflineWorkerEnabled {
		log.Println("Starting offline worker component...")
		offlineWorker, err = worker.NewOfflineWorker(
			worker.WorkerConfig{
				KafkaBrokers:  config.KafkaBrokers,
				ConsumerGroup: config.ConsumerGroup,
				Topic:         config.Topic,
				BatchSize:     config.BatchSize,
				BatchTimeout:  config.BatchTimeout,
				MaxRetries:    config.MaxRetries,
				RetryBackoff:  config.RetryBackoff,
				MessageTTL:    config.MessageTTL,
			},
			store,
			dedupService,
		)
		if err != nil {
			log.Fatalf("Failed to create offline worker: %v", err)
		}

		if err := offlineWorker.Start(); err != nil {
			log.Fatalf("Failed to start offline worker: %v", err)
		}
		log.Println("✓ Offline worker started")
	} else {
		log.Println("⚠ Offline worker disabled (OFFLINE_WORKER_ENABLED=false)")
	}

	// Start HTTP server for health checks and metrics
	go startHTTPServer(offlineWorker, config.HTTPPort)

	log.Println("✓ IM Service started successfully")
	log.Printf("  - gRPC server: :%d", config.GRPCPort)
	log.Printf("  - HTTP server: :%d", config.HTTPPort)
	log.Printf("  - Offline worker: %v", config.OfflineWorkerEnabled)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down IM Service...")

	// Graceful shutdown
	if offlineWorker != nil {
		log.Println("Stopping offline worker...")
		if err := offlineWorker.Stop(); err != nil {
			log.Printf("Error stopping worker: %v", err)
		}
	}

	log.Println("Stopping gRPC server...")
	grpcServer.GracefulStop()

	log.Println("IM Service stopped")
}

// Config holds application configuration
type Config struct {
	// gRPC server
	GRPCPort int

	// HTTP server
	HTTPPort int

	// Kafka
	KafkaBrokers  []string
	ConsumerGroup string
	Topic         string

	// Database
	DatabaseDSN       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Offline Worker
	OfflineWorkerEnabled bool
	BatchSize            int
	BatchTimeout         time.Duration
	MaxRetries           int
	RetryBackoff         []time.Duration
	MessageTTL           time.Duration
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	config := Config{
		// gRPC server defaults
		GRPCPort: getEnvInt("GRPC_PORT", 9094),

		// HTTP server defaults
		HTTPPort: getEnvInt("HTTP_PORT", 8080),

		// Kafka defaults
		KafkaBrokers:  parseStringSlice(getEnv("KAFKA_BROKERS", "localhost:9092")),
		ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "im-service-offline-workers"),
		Topic:         getEnv("KAFKA_TOPIC", "offline_msg"),

		// Database defaults
		DatabaseDSN:       buildDatabaseDSN(),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),

		// Redis defaults
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 2),

		// Offline Worker defaults
		OfflineWorkerEnabled: getEnvBool("OFFLINE_WORKER_ENABLED", true),
		BatchSize:            getEnvInt("BATCH_SIZE", 100),
		BatchTimeout:         getEnvDuration("BATCH_TIMEOUT", 5*time.Second),
		MaxRetries:           getEnvInt("MAX_RETRIES", 5),
		RetryBackoff:         parseRetryBackoff(getEnv("RETRY_BACKOFF", "1s,2s,4s,8s,16s")),
		MessageTTL:           getEnvDuration("MESSAGE_TTL", 7*24*time.Hour),
	}

	return config
}

// buildDatabaseDSN builds MySQL DSN from environment variables
func buildDatabaseDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "im_service")
	password := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "im_chat")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, password, host, port, dbName)
}

// startHTTPServer starts HTTP server for health checks and metrics
func startHTTPServer(w *worker.OfflineWorker, port int) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(rw http.ResponseWriter, r *http.Request) {
		// Check if worker is processing messages (if enabled)
		if w != nil {
			stats := w.GetStats()
			if stats.Errors > 0 && stats.MessagesProcessed == 0 {
				rw.WriteHeader(http.StatusServiceUnavailable)
				rw.Write([]byte("NOT READY"))
				return
			}
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("READY"))
	})

	// Stats endpoint
	mux.HandleFunc("/stats", func(rw http.ResponseWriter, r *http.Request) {
		if w == nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"offline_worker_enabled": false}`))
			return
		}

		stats := w.GetStats()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		fmt.Fprintf(rw, `{
			"offline_worker_enabled": true,
			"messages_processed": %d,
			"messages_deduplicated": %d,
			"messages_persisted": %d,
			"batch_writes": %d,
			"errors": %d,
			"avg_batch_size": %.2f
		}`, stats.MessagesProcessed, stats.MessagesDeduplicated,
			stats.MessagesPersisted, stats.BatchWrites,
			stats.Errors, stats.AvgBatchSize)
	})

	// Metrics endpoint (Prometheus format)
	mux.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)

		if w == nil {
			fmt.Fprintf(rw, "# HELP im_service_offline_worker_enabled Whether offline worker is enabled\n")
			fmt.Fprintf(rw, "# TYPE im_service_offline_worker_enabled gauge\n")
			fmt.Fprintf(rw, "im_service_offline_worker_enabled 0\n")
			return
		}

		stats := w.GetStats()
		fmt.Fprintf(rw, "# HELP im_service_offline_worker_enabled Whether offline worker is enabled\n")
		fmt.Fprintf(rw, "# TYPE im_service_offline_worker_enabled gauge\n")
		fmt.Fprintf(rw, "im_service_offline_worker_enabled 1\n")
		fmt.Fprintf(rw, "# HELP im_service_messages_processed_total Total messages consumed from Kafka\n")
		fmt.Fprintf(rw, "# TYPE im_service_messages_processed_total counter\n")
		fmt.Fprintf(rw, "im_service_messages_processed_total %d\n", stats.MessagesProcessed)
		fmt.Fprintf(rw, "# HELP im_service_messages_deduplicated_total Total messages skipped due to duplicates\n")
		fmt.Fprintf(rw, "# TYPE im_service_messages_deduplicated_total counter\n")
		fmt.Fprintf(rw, "im_service_messages_deduplicated_total %d\n", stats.MessagesDeduplicated)
		fmt.Fprintf(rw, "# HELP im_service_messages_persisted_total Total messages written to database\n")
		fmt.Fprintf(rw, "# TYPE im_service_messages_persisted_total counter\n")
		fmt.Fprintf(rw, "im_service_messages_persisted_total %d\n", stats.MessagesPersisted)
		fmt.Fprintf(rw, "# HELP im_service_batch_writes_total Total batch write operations\n")
		fmt.Fprintf(rw, "# TYPE im_service_batch_writes_total counter\n")
		fmt.Fprintf(rw, "im_service_batch_writes_total %d\n", stats.BatchWrites)
		fmt.Fprintf(rw, "# HELP im_service_errors_total Total errors encountered\n")
		fmt.Fprintf(rw, "# TYPE im_service_errors_total counter\n")
		fmt.Fprintf(rw, "im_service_errors_total %d\n", stats.Errors)
		fmt.Fprintf(rw, "# HELP im_service_batch_size_avg Average number of messages per batch\n")
		fmt.Fprintf(rw, "# TYPE im_service_batch_size_avg gauge\n")
		fmt.Fprintf(rw, "im_service_batch_size_avg %.2f\n", stats.AvgBatchSize)
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("✓ HTTP server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// Helper functions

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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseStringSlice(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseRetryBackoff(s string) []time.Duration {
	parts := parseStringSlice(s)
	result := make([]time.Duration, 0, len(parts))
	for _, part := range parts {
		if duration, err := time.ParseDuration(part); err == nil {
			result = append(result, duration)
		}
	}
	return result
}
