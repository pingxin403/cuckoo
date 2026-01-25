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
	"github.com/pingxin403/cuckoo/apps/im-service/readreceipt"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"github.com/pingxin403/cuckoo/apps/im-service/worker"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
)

func main() {
	// Initialize observability first
	obs, err := observability.New(observability.Config{
		ServiceName:    getEnv("SERVICE_NAME", "im-service"),
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
			log.Printf("Observability shutdown error: %v", err)
		}
	}()

	ctx := context.Background()
	obs.Logger().Info(ctx, "Starting IM Service",
		"service", "im-service",
		"version", getEnv("SERVICE_VERSION", "1.0.0"),
	)

	// Load configuration from environment
	config := loadConfig()

	// Initialize shared dependencies
	obs.Logger().Info(ctx, "Initializing shared dependencies")

	// Create storage
	store, err := storage.NewOfflineStore(storage.Config{
		DSN:             config.DatabaseDSN,
		MaxOpenConns:    config.DBMaxOpenConns,
		MaxIdleConns:    config.DBMaxIdleConns,
		ConnMaxLifetime: config.DBConnMaxLifetime,
	})
	if err != nil {
		obs.Logger().Error(ctx, "Failed to create offline store", "error", err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()
	obs.Logger().Info(ctx, "Connected to database")

	// Create dedup service
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr:     config.RedisAddr,
		RedisPassword: config.RedisPassword,
		RedisDB:       config.RedisDB,
		TTL:           config.MessageTTL,
	})
	defer func() { _ = dedupService.Close() }()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := dedupService.Ping(ctx); err != nil {
		obs.Logger().Error(ctx, "Failed to connect to Redis", "error", err)
		cancel()
		os.Exit(1)
	}
	cancel()
	obs.Logger().Info(ctx, "Connected to Redis")

	// Start gRPC server for message routing
	grpcServer := grpc.NewServer()
	// TODO: Register IM Service gRPC handlers here when Task 9 is implemented
	// pb.RegisterIMServiceServer(grpcServer, imService)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.GRPCPort))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", config.GRPCPort, "error", err)
		os.Exit(1)
	}

	go func() {
		obs.Logger().Info(ctx, "gRPC server listening", "port", config.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start offline worker (background component)
	var offlineWorker *worker.OfflineWorker
	if config.OfflineWorkerEnabled {
		obs.Logger().Info(ctx, "Starting offline worker component")
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
			obs.Logger().Error(ctx, "Failed to create offline worker", "error", err)
			os.Exit(1)
		}

		if err := offlineWorker.Start(); err != nil {
			obs.Logger().Error(ctx, "Failed to start offline worker", "error", err)
			os.Exit(1)
		}
		obs.Logger().Info(ctx, "Offline worker started")
	} else {
		obs.Logger().Warn(ctx, "Offline worker disabled", "reason", "OFFLINE_WORKER_ENABLED=false")
	}

	// Initialize read receipt service
	obs.Logger().Info(ctx, "Initializing read receipt service")
	var readReceiptService *readreceipt.ReadReceiptService
	if config.ReadReceiptKafkaEnabled {
		readReceiptService = readreceipt.NewReadReceiptServiceWithKafka(
			store.GetDB(),
			config.KafkaBrokers,
			config.ReadReceiptTopic,
		)
		obs.Logger().Info(ctx, "Read receipt service initialized with Kafka support")
	} else {
		readReceiptService = readreceipt.NewReadReceiptService(store.GetDB())
		obs.Logger().Info(ctx, "Read receipt service initialized", "kafka_enabled", false)
	}
	defer func() { _ = readReceiptService.Close() }()

	readReceiptHandler := readreceipt.NewHTTPHandler(readReceiptService)
	obs.Logger().Info(ctx, "Read receipt HTTP handler initialized")

	// Start HTTP server for health checks, metrics, and read receipts
	go startHTTPServer(obs, offlineWorker, readReceiptHandler, config.HTTPPort)

	obs.Logger().Info(ctx, "IM Service started successfully",
		"grpc_port", config.GRPCPort,
		"http_port", config.HTTPPort,
		"offline_worker_enabled", config.OfflineWorkerEnabled,
	)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	obs.Logger().Info(ctx, "Shutting down IM Service")

	// Graceful shutdown
	if offlineWorker != nil {
		obs.Logger().Info(ctx, "Stopping offline worker")
		if err := offlineWorker.Stop(); err != nil {
			obs.Logger().Error(ctx, "Error stopping worker", "error", err)
		}
	}

	obs.Logger().Info(ctx, "Stopping gRPC server")
	grpcServer.GracefulStop()

	obs.Logger().Info(ctx, "IM Service stopped")
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

	// Read Receipt
	ReadReceiptKafkaEnabled bool
	ReadReceiptTopic        string
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

		// Read Receipt defaults
		ReadReceiptKafkaEnabled: getEnvBool("READ_RECEIPT_KAFKA_ENABLED", true),
		ReadReceiptTopic:        getEnv("READ_RECEIPT_TOPIC", "read_receipt_events"),
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

// startHTTPServer starts HTTP server for health checks, metrics, and read receipts
func startHTTPServer(obs observability.Observability, w *worker.OfflineWorker, readReceiptHandler *readreceipt.HTTPHandler, port int) {
	ctx := context.Background()
	mux := http.NewServeMux()

	// Middleware to track HTTP metrics
	metricsMiddleware := func(path string, handler http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrappedWriter := &responseWriter{ResponseWriter: rw, statusCode: http.StatusOK}

			// Call the actual handler
			handler(wrappedWriter, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := fmt.Sprintf("%d", wrappedWriter.statusCode)

			obs.Metrics().IncrementCounter("http_requests_total", map[string]string{
				"path":   path,
				"status": status,
			})
			obs.Metrics().RecordHistogram("http_request_duration_seconds", duration, map[string]string{
				"path": path,
			})
		}
	}

	// Health check endpoint
	mux.HandleFunc("/health", metricsMiddleware("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("OK"))
	}))

	// Readiness check endpoint
	mux.HandleFunc("/ready", metricsMiddleware("/ready", func(rw http.ResponseWriter, r *http.Request) {
		// Check if worker is processing messages (if enabled)
		if w != nil {
			stats := w.GetStats()
			if stats.Errors > 0 && stats.MessagesProcessed == 0 {
				rw.WriteHeader(http.StatusServiceUnavailable)
				_, _ = rw.Write([]byte("NOT READY"))
				return
			}
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("READY"))
	}))

	// Stats endpoint
	mux.HandleFunc("/stats", metricsMiddleware("/stats", func(rw http.ResponseWriter, r *http.Request) {
		if w == nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{"offline_worker_enabled": false}`))
			return
		}

		stats := w.GetStats()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(rw, `{
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
	}))

	// Metrics endpoint (Prometheus format) - use observability library's handler
	// Note: The observability library already exposes metrics on the configured MetricsPort (9090)
	// This endpoint provides backward compatibility and worker-specific metrics
	mux.HandleFunc("/metrics", metricsMiddleware("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)

		// Worker-specific metrics (if enabled)
		if w != nil {
			stats := w.GetStats()
			// Use observability library metrics instead of custom implementation
			obs.Metrics().SetGauge("offline_worker_enabled", 1, nil)
			obs.Metrics().SetGauge("messages_processed", float64(stats.MessagesProcessed), nil)
			obs.Metrics().SetGauge("messages_deduplicated", float64(stats.MessagesDeduplicated), nil)
			obs.Metrics().SetGauge("messages_persisted", float64(stats.MessagesPersisted), nil)
			obs.Metrics().SetGauge("batch_writes", float64(stats.BatchWrites), nil)
			obs.Metrics().SetGauge("errors", float64(stats.Errors), nil)
			obs.Metrics().SetGauge("batch_size_avg", stats.AvgBatchSize, nil)
		} else {
			obs.Metrics().SetGauge("offline_worker_enabled", 0, nil)
		}

		// Redirect to observability library's metrics endpoint
		_, _ = rw.Write([]byte("# Metrics are available on the observability metrics port (default: 9090)\n"))
		_, _ = rw.Write([]byte("# Worker metrics have been recorded and will be exported via the observability library\n"))
	}))

	// Read Receipt API endpoints
	mux.HandleFunc("/api/v1/messages/read", metricsMiddleware("/api/v1/messages/read", readReceiptHandler.HandleMarkAsRead))
	mux.HandleFunc("/api/v1/messages/unread/count", metricsMiddleware("/api/v1/messages/unread/count", readReceiptHandler.HandleGetUnreadCount))
	mux.HandleFunc("/api/v1/messages/unread", metricsMiddleware("/api/v1/messages/unread", readReceiptHandler.HandleGetUnreadMessages))
	mux.HandleFunc("/api/v1/messages/receipts", metricsMiddleware("/api/v1/messages/receipts", readReceiptHandler.HandleGetReadReceipts))
	mux.HandleFunc("/api/v1/conversations/read", metricsMiddleware("/api/v1/conversations/read", readReceiptHandler.HandleMarkConversationAsRead))

	addr := fmt.Sprintf(":%d", port)
	obs.Logger().Info(ctx, "HTTP server listening", "address", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		obs.Logger().Error(ctx, "HTTP server failed", "error", err)
		os.Exit(1)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
