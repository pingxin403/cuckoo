package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
	_ "github.com/mattn/go-sqlite3"
)

// ExampleHealthChecker demonstrates how to set up and use the health checker
func ExampleHealthChecker() {
	// Create logger
	logger := log.New(os.Stdout, "[HealthChecker] ", log.LstdFlags)

	// Create health checker configuration
	config := DefaultConfig("region-a")
	config.CheckInterval = 3 * time.Second
	config.DefaultTimeout = 2 * time.Second
	config.EnableMetrics = true
	config.MetricsInterval = 10 * time.Second

	// Create health checker
	checker := NewHealthChecker(config, logger)

	// Setup components for health checking
	setupHealthChecks(checker, logger)

	// Start health checking
	if err := checker.Start(); err != nil {
		log.Fatalf("Failed to start health checker: %v", err)
	}
	defer checker.Stop()

	// Start HTTP server for health endpoints
	startHealthServer(checker, 8080)

	// Simulate running for a while
	fmt.Println("Health checker running... Check http://localhost:8080/health")
	time.Sleep(30 * time.Second)
}

// setupHealthChecks configures all health checks for the multi-region system
func setupHealthChecks(checker *HealthChecker, logger *log.Logger) {
	// 1. Setup MySQL/SQLite health check
	setupDatabaseHealthCheck(checker, logger)

	// 2. Setup Redis/Storage health check
	setupStorageHealthCheck(checker, logger)

	// 3. Setup Kafka/Queue health check
	setupQueueHealthCheck(checker, logger)

	// 4. Setup Network health checks
	setupNetworkHealthChecks(checker)

	// 5. Setup HTTP service health checks
	setupHTTPHealthChecks(checker)
}

// setupDatabaseHealthCheck creates a database health check
func setupDatabaseHealthCheck(checker *HealthChecker, logger *log.Logger) {
	// Create SQLite database for demonstration
	db, err := sql.Open("sqlite3", "./health_test.db")
	if err != nil {
		logger.Printf("Failed to create database: %v", err)
		return
	}

	// Create a simple table for testing
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS health_test (
			id INTEGER PRIMARY KEY,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		logger.Printf("Failed to create test table: %v", err)
		return
	}

	// Register MySQL health check
	mysqlCheck := NewMySQLHealthCheck("mysql", db)
	checker.RegisterCheck(mysqlCheck)

	logger.Printf("Registered MySQL health check")
}

// setupStorageHealthCheck creates a storage health check
func setupStorageHealthCheck(checker *HealthChecker, logger *log.Logger) {
	// Create local storage for demonstration
	config := storage.Config{
		RegionID:   "region-a",
		MemoryMode: true, // Use memory mode for demo
		TTL:        24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		logger.Printf("Failed to create storage: %v", err)
		return
	}

	// Register Redis health check (using storage as Redis simulation)
	redisCheck := NewRedisHealthCheck("redis", store)
	checker.RegisterCheck(redisCheck)

	logger.Printf("Registered Redis health check")
}

// setupQueueHealthCheck creates a queue health check
func setupQueueHealthCheck(checker *HealthChecker, logger *log.Logger) {
	// Create local queue for demonstration
	config := queue.DefaultConfig("region-a")
	config.BufferSize = 1000

	testQueue, err := queue.NewLocalQueue(config, logger)
	if err != nil {
		logger.Printf("Failed to create queue: %v", err)
		return
	}

	// Register Kafka health check (using local queue as Kafka simulation)
	kafkaCheck := NewKafkaHealthCheck("kafka", testQueue)
	checker.RegisterCheck(kafkaCheck)

	logger.Printf("Registered Kafka health check")
}

// setupNetworkHealthChecks creates network connectivity health checks
func setupNetworkHealthChecks(checker *HealthChecker) {
	// Check connectivity to remote region (simulated)
	// In production, this would be the actual remote region endpoints
	networkChecks := []struct {
		name string
		host string
		port int
	}{
		{"network-region-b", "127.0.0.1", 8081}, // Simulated remote region
		{"network-dns", "8.8.8.8", 53},          // DNS connectivity
	}

	for _, nc := range networkChecks {
		networkCheck := NewNetworkHealthCheck(nc.name, nc.host, nc.port)
		checker.RegisterCheck(networkCheck)
	}
}

// setupHTTPHealthChecks creates HTTP service health checks
func setupHTTPHealthChecks(checker *HealthChecker) {
	// Check HTTP services
	httpChecks := []struct {
		name string
		url  string
	}{
		{"http-local-api", "http://localhost:8080/health"},
		{"http-external", "https://httpbin.org/status/200"},
	}

	for _, hc := range httpChecks {
		httpCheck := NewHTTPHealthCheck(hc.name, hc.url)
		checker.RegisterCheck(httpCheck)
	}
}

// startHealthServer starts an HTTP server with health endpoints
func startHealthServer(checker *HealthChecker, port int) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		health := checker.GetSystemHealth()

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status based on health status
		switch health.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // Still OK, but degraded
		case StatusCritical:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(health)
	})

	// Component-specific health endpoint
	mux.HandleFunc("/health/", func(w http.ResponseWriter, r *http.Request) {
		componentName := r.URL.Path[len("/health/"):]
		if componentName == "" {
			http.Error(w, "Component name required", http.StatusBadRequest)
			return
		}

		component, err := checker.GetComponentHealth(componentName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status based on component health
		switch component.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK)
		case StatusCritical:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(component)
	})

	// Readiness probe endpoint (Kubernetes-style)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		health := checker.GetSystemHealth()

		// Ready if not critical
		if health.Status != StatusCritical {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	})

	// Liveness probe endpoint (Kubernetes-style)
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		// Always return OK if the service is running
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("alive"))
	})

	// Metrics endpoint (Prometheus-style)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		health := checker.GetSystemHealth()

		w.Header().Set("Content-Type", "text/plain")

		// Export metrics in Prometheus format
		fmt.Fprintf(w, "# HELP health_status Overall system health status (0=critical, 1=degraded, 2=healthy)\n")
		fmt.Fprintf(w, "# TYPE health_status gauge\n")

		var statusValue int
		switch health.Status {
		case StatusCritical:
			statusValue = 0
		case StatusDegraded:
			statusValue = 1
		case StatusHealthy:
			statusValue = 2
		}
		fmt.Fprintf(w, "health_status{region=\"%s\"} %d\n", health.RegionID, statusValue)

		fmt.Fprintf(w, "# HELP health_score Overall system health score (0.0 to 1.0)\n")
		fmt.Fprintf(w, "# TYPE health_score gauge\n")
		fmt.Fprintf(w, "health_score{region=\"%s\"} %.2f\n", health.RegionID, health.Score)

		// Component-specific metrics
		for name, component := range health.Components {
			fmt.Fprintf(w, "# HELP component_status Component health status (0=critical, 1=degraded, 2=healthy)\n")
			fmt.Fprintf(w, "# TYPE component_status gauge\n")

			var componentStatusValue int
			switch component.Status {
			case StatusCritical:
				componentStatusValue = 0
			case StatusDegraded:
				componentStatusValue = 1
			case StatusHealthy:
				componentStatusValue = 2
			}
			fmt.Fprintf(w, "component_status{region=\"%s\",component=\"%s\"} %d\n",
				health.RegionID, name, componentStatusValue)

			fmt.Fprintf(w, "# HELP component_response_time_ms Component response time in milliseconds\n")
			fmt.Fprintf(w, "# TYPE component_response_time_ms gauge\n")
			fmt.Fprintf(w, "component_response_time_ms{region=\"%s\",component=\"%s\"} %d\n",
				health.RegionID, name, component.ResponseTime.Milliseconds())
		}
	})

	// Start server in background
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		log.Printf("Starting health server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()
}

// DemoHealthChecker runs a demonstration of the health checker
func DemoHealthChecker() {
	fmt.Println("=== Multi-Region Health Checker Demo ===")

	// Create health checker
	config := DefaultConfig("region-a")
	config.CheckInterval = 2 * time.Second
	config.EnableMetrics = true
	config.MetricsInterval = 5 * time.Second

	logger := log.New(os.Stdout, "[Demo] ", log.LstdFlags)
	checker := NewHealthChecker(config, logger)

	// Register some demo health checks
	registerDemoChecks(checker)

	// Start health checking
	if err := checker.Start(); err != nil {
		log.Fatalf("Failed to start health checker: %v", err)
	}
	defer checker.Stop()

	// Start health server
	startHealthServer(checker, 8080)

	// Monitor health for a while
	fmt.Println("\nMonitoring system health...")
	fmt.Println("Health API available at:")
	fmt.Println("  - http://localhost:8080/health (full system health)")
	fmt.Println("  - http://localhost:8080/health/mysql (component health)")
	fmt.Println("  - http://localhost:8080/ready (readiness probe)")
	fmt.Println("  - http://localhost:8080/live (liveness probe)")
	fmt.Println("  - http://localhost:8080/metrics (Prometheus metrics)")
	fmt.Println()

	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)

		health := checker.GetSystemHealth()
		fmt.Printf("[%s] System Status: %s (Score: %.2f) - %s\n",
			health.Timestamp.Format("15:04:05"),
			health.Status,
			health.Score,
			health.Summary)

		// Show component details
		for name, component := range health.Components {
			fmt.Printf("  └─ %s: %s (%dms)",
				name,
				component.Status,
				component.ResponseTime.Milliseconds())
			if component.Error != "" {
				fmt.Printf(" - %s", component.Error)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

// registerDemoChecks registers demonstration health checks
func registerDemoChecks(checker *HealthChecker) {
	// Healthy check
	healthyCheck := &demoHealthCheck{
		name:     "demo-healthy",
		interval: 2 * time.Second,
		timeout:  1 * time.Second,
		checkFn: func(ctx context.Context) error {
			// Simulate some work
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}
	checker.RegisterCheck(healthyCheck)

	// Slow check (will be marked as degraded)
	slowCheck := &demoHealthCheck{
		name:     "demo-slow",
		interval: 3 * time.Second,
		timeout:  1 * time.Second,
		checkFn: func(ctx context.Context) error {
			// Simulate slow response
			time.Sleep(800 * time.Millisecond)
			return nil
		},
	}
	checker.RegisterCheck(slowCheck)

	// Intermittent failing check
	failCounter := 0
	intermittentCheck := &demoHealthCheck{
		name:     "demo-intermittent",
		interval: 2 * time.Second,
		timeout:  1 * time.Second,
		checkFn: func(ctx context.Context) error {
			failCounter++
			if failCounter%3 == 0 {
				return fmt.Errorf("simulated intermittent failure")
			}
			return nil
		},
	}
	checker.RegisterCheck(intermittentCheck)

	// External service check (will likely fail)
	externalCheck := NewHTTPHealthCheck("demo-external", "https://httpbin.org/delay/1")
	checker.RegisterCheck(externalCheck)
}

// demoHealthCheck is a simple implementation for demonstration
type demoHealthCheck struct {
	name     string
	checkFn  func(ctx context.Context) error
	timeout  time.Duration
	interval time.Duration
}

func (m *demoHealthCheck) Name() string            { return m.name }
func (m *demoHealthCheck) Timeout() time.Duration  { return m.timeout }
func (m *demoHealthCheck) Interval() time.Duration { return m.interval }
func (m *demoHealthCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}
