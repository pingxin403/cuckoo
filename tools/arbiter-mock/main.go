package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ArbiterMock simulates a distributed arbiter service for split-brain prevention
type ArbiterMock struct {
	mu              sync.RWMutex
	currentPrimary  string
	regionHealth    map[string]RegionHealth
	electionHistory []ElectionEvent

	// Metrics
	electionCount prometheus.Counter
	healthChecks  prometheus.CounterVec
}

type RegionHealth struct {
	RegionID      string          `json:"region_id"`
	IsHealthy     bool            `json:"is_healthy"`
	LastHeartbeat time.Time       `json:"last_heartbeat"`
	Services      map[string]bool `json:"services"`
}

type ElectionEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	Winner       string    `json:"winner"`
	Participants []string  `json:"participants"`
	Reason       string    `json:"reason"`
}

type ElectionRequest struct {
	RegionID  string          `json:"region_id"`
	Services  map[string]bool `json:"services"`
	Timestamp time.Time       `json:"timestamp"`
}

type ElectionResponse struct {
	Winner    string    `json:"winner"`
	IsPrimary bool      `json:"is_primary"`
	Timestamp time.Time `json:"timestamp"`
	TTL       int       `json:"ttl_seconds"`
}

type HealthCheckRequest struct {
	RegionID  string          `json:"region_id"`
	Services  map[string]bool `json:"services"`
	Timestamp time.Time       `json:"timestamp"`
}

func NewArbiterMock() *ArbiterMock {
	return &ArbiterMock{
		regionHealth:    make(map[string]RegionHealth),
		electionHistory: make([]ElectionEvent, 0),
		electionCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "arbiter_elections_total",
			Help: "Total number of primary elections conducted",
		}),
		healthChecks: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "arbiter_health_checks_total",
			Help: "Total number of health checks received",
		}, []string{"region", "status"}),
	}
}

func (a *ArbiterMock) registerMetrics() {
	prometheus.MustRegister(a.electionCount)
	prometheus.MustRegister(a.healthChecks)
}

// Health check endpoint
func (a *ArbiterMock) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "arbiter-mock",
		"version":   "1.0.0",
	})
}

// Primary election endpoint
func (a *ArbiterMock) electPrimaryHandler(w http.ResponseWriter, r *http.Request) {
	var req ElectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Update region health
	a.regionHealth[req.RegionID] = RegionHealth{
		RegionID:      req.RegionID,
		IsHealthy:     a.isRegionHealthy(req.Services),
		LastHeartbeat: req.Timestamp,
		Services:      req.Services,
	}

	// Determine primary based on health and current state
	winner := a.determinePrimary()

	// Record election event
	participants := make([]string, 0, len(a.regionHealth))
	for regionID := range a.regionHealth {
		participants = append(participants, regionID)
	}

	event := ElectionEvent{
		Timestamp:    time.Now(),
		Winner:       winner,
		Participants: participants,
		Reason:       "election_request",
	}

	a.electionHistory = append(a.electionHistory, event)
	a.currentPrimary = winner
	a.electionCount.Inc()

	// Prepare response
	response := ElectionResponse{
		Winner:    winner,
		IsPrimary: winner == req.RegionID,
		Timestamp: time.Now(),
		TTL:       30, // 30 seconds TTL
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Election completed: winner=%s, requester=%s, is_primary=%v",
		winner, req.RegionID, response.IsPrimary)
}

// Health check reporting endpoint
func (a *ArbiterMock) reportHealthHandler(w http.ResponseWriter, r *http.Request) {
	var req HealthCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	isHealthy := a.isRegionHealthy(req.Services)

	// Update region health
	a.regionHealth[req.RegionID] = RegionHealth{
		RegionID:      req.RegionID,
		IsHealthy:     isHealthy,
		LastHeartbeat: req.Timestamp,
		Services:      req.Services,
	}

	// Record metrics
	status := "healthy"
	if !isHealthy {
		status = "unhealthy"
	}
	a.healthChecks.WithLabelValues(req.RegionID, status).Inc()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"region_id":  req.RegionID,
		"is_healthy": isHealthy,
		"timestamp":  time.Now(),
		"ttl":        60, // 60 seconds TTL
	})

	log.Printf("Health check received: region=%s, healthy=%v, services=%v",
		req.RegionID, isHealthy, req.Services)
}

// Get current status endpoint
func (a *ArbiterMock) statusHandler(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Clean up stale health records (older than 2 minutes)
	cutoff := time.Now().Add(-2 * time.Minute)
	for regionID, health := range a.regionHealth {
		if health.LastHeartbeat.Before(cutoff) {
			delete(a.regionHealth, regionID)
		}
	}

	status := map[string]interface{}{
		"current_primary":  a.currentPrimary,
		"region_health":    a.regionHealth,
		"election_history": a.electionHistory[max(0, len(a.electionHistory)-10):], // Last 10 events
		"timestamp":        time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Determine primary region based on health and election rules
func (a *ArbiterMock) determinePrimary() string {
	healthyRegions := make([]string, 0)

	// Find all healthy regions
	for regionID, health := range a.regionHealth {
		if health.IsHealthy && time.Since(health.LastHeartbeat) < 30*time.Second {
			healthyRegions = append(healthyRegions, regionID)
		}
	}

	// Election rules:
	// 1. If current primary is healthy, keep it
	// 2. If no healthy regions, return empty (read-only mode)
	// 3. If multiple healthy regions, prefer region-a (deterministic)
	// 4. If only one healthy region, elect it

	if len(healthyRegions) == 0 {
		log.Println("No healthy regions found, entering read-only mode")
		return ""
	}

	// Check if current primary is still healthy
	for _, regionID := range healthyRegions {
		if regionID == a.currentPrimary {
			return a.currentPrimary
		}
	}

	// Deterministic election: prefer region-a, then region-b
	for _, preferred := range []string{"region-a", "region-b"} {
		for _, regionID := range healthyRegions {
			if regionID == preferred {
				return regionID
			}
		}
	}

	// Fallback: return first healthy region
	return healthyRegions[0]
}

// Check if a region is healthy based on its services
func (a *ArbiterMock) isRegionHealthy(services map[string]bool) bool {
	// A region is healthy if all critical services are up
	criticalServices := []string{"im-service", "redis", "database"}

	for _, service := range criticalServices {
		if healthy, exists := services[service]; !exists || !healthy {
			return false
		}
	}

	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9999"
	}

	arbiter := NewArbiterMock()
	arbiter.registerMetrics()

	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/health", arbiter.healthHandler).Methods("GET")
	r.HandleFunc("/elect", arbiter.electPrimaryHandler).Methods("POST")
	r.HandleFunc("/report-health", arbiter.reportHealthHandler).Methods("POST")
	r.HandleFunc("/status", arbiter.statusHandler).Methods("GET")

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// CORS middleware for development
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

	log.Printf("Arbiter Mock Service starting on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  GET  /health - Health check")
	log.Printf("  POST /elect - Primary election")
	log.Printf("  POST /report-health - Health reporting")
	log.Printf("  GET  /status - Current status")
	log.Printf("  GET  /metrics - Prometheus metrics")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
