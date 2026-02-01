package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/example/im-system/arbiter"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ArbiterService struct {
	client  *arbiter.ArbiterClient
	logger  *log.Logger
	metrics *Metrics
}

type Metrics struct {
	electionCount    prometheus.Counter
	electionDuration prometheus.Histogram
	healthCheckCount prometheus.CounterVec
	leadershipStatus prometheus.Gauge
}

func newMetrics() *Metrics {
	m := &Metrics{
		electionCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arbiter_elections_total",
			Help: "Total number of elections performed",
		}),
		electionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "arbiter_election_duration_seconds",
			Help:    "Duration of election operations",
			Buckets: prometheus.DefBuckets,
		}),
		healthCheckCount: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "arbiter_health_checks_total",
			Help: "Total number of health checks performed",
		}, []string{"service", "status"}),
		leadershipStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "arbiter_is_leader",
			Help: "Whether this region is currently the leader (1) or not (0)",
		}),
	}

	prometheus.MustRegister(m.electionCount)
	prometheus.MustRegister(m.electionDuration)
	prometheus.MustRegister(m.healthCheckCount)
	prometheus.MustRegister(m.leadershipStatus)

	return m
}

func main() {
	logger := log.New(os.Stdout, "[ARBITER-SERVICE] ", log.LstdFlags)

	// Parse configuration from environment
	config, err := parseConfig()
	if err != nil {
		logger.Fatalf("Failed to parse configuration: %v", err)
	}

	config.Logger = logger

	// Create arbiter client
	client, err := arbiter.NewArbiterClient(config)
	if err != nil {
		logger.Fatalf("Failed to create arbiter client: %v", err)
	}
	defer client.Close()

	// Create service
	service := &ArbiterService{
		client:  client,
		logger:  logger,
		metrics: newMetrics(),
	}

	// Start HTTP server
	router := service.setupRoutes()
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start background health monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go service.healthMonitorLoop(ctx)
	go service.leadershipMonitorLoop(ctx)

	// Start HTTP server in goroutine
	go func() {
		logger.Printf("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Println("Shutting down...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("HTTP server shutdown error: %v", err)
	}

	cancel() // Stop background goroutines
	logger.Println("Shutdown complete")
}

func parseConfig() (arbiter.Config, error) {
	config := arbiter.Config{
		SessionTimeout: 10 * time.Second,
		ElectionTTL:    30 * time.Second,
	}

	// Parse Zookeeper hosts
	zkHosts := os.Getenv("ZOOKEEPER_HOSTS")
	if zkHosts == "" {
		zkHosts = "localhost:2181"
	}
	config.ZookeeperHosts = strings.Split(zkHosts, ",")

	// Parse region ID
	config.RegionID = os.Getenv("REGION_ID")
	if config.RegionID == "" {
		return config, fmt.Errorf("REGION_ID environment variable is required")
	}

	// Parse session timeout
	if timeoutStr := os.Getenv("SESSION_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.SessionTimeout = timeout
		}
	}

	// Parse election TTL
	if ttlStr := os.Getenv("ELECTION_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			config.ElectionTTL = ttl
		}
	}

	return config, nil
}

func (s *ArbiterService) setupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Arbiter API endpoints
	r.HandleFunc("/api/v1/elect", s.electHandler).Methods("POST")
	r.HandleFunc("/api/v1/leader", s.leaderHandler).Methods("GET")
	r.HandleFunc("/api/v1/status", s.statusHandler).Methods("GET")
	r.HandleFunc("/api/v1/health-report", s.healthReportHandler).Methods("POST")

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	return r
}

func (s *ArbiterService) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "arbiter-service",
		"version":   "1.0.0",
	})
}

func (s *ArbiterService) electHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Services map[string]bool `json:"services"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	start := time.Now()
	result, err := s.client.ElectPrimary(r.Context(), req.Services)
	duration := time.Since(start)

	s.metrics.electionCount.Inc()
	s.metrics.electionDuration.Observe(duration.Seconds())

	if err != nil {
		s.logger.Printf("Election failed: %v", err)
		http.Error(w, fmt.Sprintf("Election failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Update leadership metric
	if result.IsPrimary {
		s.metrics.leadershipStatus.Set(1)
	} else {
		s.metrics.leadershipStatus.Set(0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	s.logger.Printf("Election completed: leader=%s, is_primary=%v, reason=%s",
		result.Leader, result.IsPrimary, result.Reason)
}

func (s *ArbiterService) leaderHandler(w http.ResponseWriter, r *http.Request) {
	leader, err := s.client.GetCurrentLeader(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get leader: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"leader":    leader,
		"timestamp": time.Now(),
	})
}

func (s *ArbiterService) statusHandler(w http.ResponseWriter, r *http.Request) {
	leader, _ := s.client.GetCurrentLeader(r.Context())
	health := s.client.GetHealthStatus()
	isHealthy := s.client.IsHealthy()

	history, _ := s.client.GetElectionHistory(r.Context(), 5)

	status := map[string]interface{}{
		"current_leader":   leader,
		"health_status":    health,
		"is_healthy":       isHealthy,
		"election_history": history,
		"timestamp":        time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *ArbiterService) healthReportHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Services map[string]bool `json:"services"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := s.client.ReportHealth(req.Services)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to report health: %v", err), http.StatusInternalServerError)
		return
	}

	// Update health check metrics
	for service, healthy := range req.Services {
		status := "healthy"
		if !healthy {
			status = "unhealthy"
		}
		s.metrics.healthCheckCount.WithLabelValues(service, status).Inc()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "reported",
		"timestamp": time.Now(),
	})
}

func (s *ArbiterService) healthMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check health of local services
			health := s.checkLocalServices()

			// Report health to arbiter
			if err := s.client.ReportHealth(health); err != nil {
				s.logger.Printf("Failed to report health: %v", err)
			}

			// Update metrics
			for service, healthy := range health {
				status := "healthy"
				if !healthy {
					status = "unhealthy"
				}
				s.metrics.healthCheckCount.WithLabelValues(service, status).Inc()
			}
		}
	}
}

func (s *ArbiterService) leadershipMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Perform election with current health status
			health := s.checkLocalServices()
			result, err := s.client.ElectPrimary(ctx, health)
			if err != nil {
				s.logger.Printf("Leadership check failed: %v", err)
				s.metrics.leadershipStatus.Set(-1) // Unknown state
				continue
			}

			// Update leadership metric
			if result.IsPrimary {
				s.metrics.leadershipStatus.Set(1)
				s.logger.Printf("Leadership confirmed: PRIMARY")
			} else {
				s.metrics.leadershipStatus.Set(0)
				s.logger.Printf("Leadership confirmed: SECONDARY (leader: %s)", result.Leader)
			}

			s.metrics.electionCount.Inc()
		}
	}
}

func (s *ArbiterService) checkLocalServices() map[string]bool {
	health := make(map[string]bool)

	// Check IM service
	health["im-service"] = s.checkHTTPService("http://localhost:8080/health")

	// Check Redis
	health["redis"] = s.checkRedis()

	// Check Database
	health["database"] = s.checkDatabase()

	return health
}

func (s *ArbiterService) checkHTTPService(url string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *ArbiterService) checkRedis() bool {
	// In a real implementation, this would connect to Redis and ping
	// For now, we'll check if Redis port is accessible
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Simple TCP connection check
	conn, err := net.DialTimeout("tcp", redisAddr, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *ArbiterService) checkDatabase() bool {
	// In a real implementation, this would connect to the database
	// For now, we'll check if the database file exists (SQLite case)
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		return true // Assume healthy if no path specified
	}

	_, err := os.Stat(dbPath)
	return err == nil
}
