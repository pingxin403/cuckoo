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

	"github.com/pingxin403/cuckoo/apps/im-service/config"
	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/readreceipt"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"github.com/pingxin403/cuckoo/apps/im-service/worker"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize observability first
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
			log.Printf("Observability shutdown error: %v", err)
		}
	}()

	ctx := context.Background()
	obs.Logger().Info(ctx, "Starting IM Service",
		"service", cfg.Observability.ServiceName,
		"version", cfg.Observability.ServiceVersion,
		"grpc_port", cfg.Server.GRPCPort,
		"http_port", cfg.Server.HTTPPort,
	)

	// Initialize shared dependencies
	obs.Logger().Info(ctx, "Initializing shared dependencies")

	// Create storage
	store, err := storage.NewOfflineStore(storage.Config{
		DSN:             cfg.GetDatabaseDSN(),
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		obs.Logger().Error(ctx, "Failed to create offline store", "error", err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()
	obs.Logger().Info(ctx, "Connected to database")

	// Create dedup service
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
		TTL:           cfg.OfflineWorker.MessageTTL,
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

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		obs.Logger().Error(ctx, "Failed to listen", "port", cfg.Server.GRPCPort, "error", err)
		os.Exit(1)
	}

	go func() {
		obs.Logger().Info(ctx, "gRPC server listening", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			obs.Logger().Error(ctx, "Failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	// Start offline worker (background component)
	var offlineWorker *worker.OfflineWorker
	if cfg.OfflineWorker.Enabled {
		obs.Logger().Info(ctx, "Starting offline worker component")
		offlineWorker, err = worker.NewOfflineWorker(
			worker.WorkerConfig{
				KafkaBrokers:  cfg.Kafka.Brokers,
				ConsumerGroup: cfg.Kafka.ConsumerGroup,
				Topic:         cfg.Kafka.Topic,
				BatchSize:     cfg.OfflineWorker.BatchSize,
				BatchTimeout:  cfg.OfflineWorker.BatchTimeout,
				MaxRetries:    cfg.OfflineWorker.MaxRetries,
				RetryBackoff:  cfg.OfflineWorker.RetryBackoff,
				MessageTTL:    cfg.OfflineWorker.MessageTTL,
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
	if cfg.ReadReceipt.KafkaEnabled {
		readReceiptService = readreceipt.NewReadReceiptServiceWithKafka(
			store.GetDB(),
			cfg.Kafka.Brokers,
			cfg.ReadReceipt.Topic,
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
	go startHTTPServer(obs, offlineWorker, readReceiptHandler, cfg.Server.HTTPPort)

	obs.Logger().Info(ctx, "IM Service started successfully",
		"grpc_port", cfg.Server.GRPCPort,
		"http_port", cfg.Server.HTTPPort,
		"offline_worker_enabled", cfg.OfflineWorker.Enabled,
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
