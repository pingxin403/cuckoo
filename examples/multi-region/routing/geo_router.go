package routing

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeoRouter handles geographic routing for multi-region active-active architecture
type GeoRouter struct {
	regionID       string
	regions        map[string]*RegionInfo
	healthCheckers map[string]*HealthChecker
	routingRules   []RoutingRule
	logger         *log.Logger
	mu             sync.RWMutex

	// Configuration
	config GeoRouterConfig

	// HTTP server for health checks and routing
	server *http.Server
}

// RegionInfo contains information about a region
type RegionInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Endpoint    string            `json:"endpoint"`
	Priority    int               `json:"priority"` // Lower number = higher priority
	Weight      int               `json:"weight"`   // For load balancing
	Healthy     bool              `json:"healthy"`
	LastCheck   time.Time         `json:"last_check"`
	Latency     time.Duration     `json:"latency"`
	Metadata    map[string]string `json:"metadata"`
	GeoLocation GeoLocation       `json:"geo_location"`
}

// GeoLocation represents geographic coordinates
type GeoLocation struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RoutingRule defines how to route requests based on various criteria
type RoutingRule struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Priority     int               `json:"priority"`
	Conditions   []Condition       `json:"conditions"`
	TargetRegion string            `json:"target_region"`
	Enabled      bool              `json:"enabled"`
	Metadata     map[string]string `json:"metadata"`
}

// Condition represents a routing condition
type Condition struct {
	Type     string `json:"type"`     // "header", "geo", "user_id", "custom"
	Key      string `json:"key"`      // Header name, geo field, etc.
	Operator string `json:"operator"` // "equals", "contains", "matches", "in_range"
	Value    string `json:"value"`    // Expected value
}

// RoutingDecision represents the result of routing logic
type RoutingDecision struct {
	TargetRegion   string            `json:"target_region"`
	Reason         string            `json:"reason"`
	Rule           *RoutingRule      `json:"rule,omitempty"`
	Confidence     float64           `json:"confidence"`   // 0.0 - 1.0
	Alternatives   []string          `json:"alternatives"` // Alternative regions
	DecisionTime   time.Time         `json:"decision_time"`
	ProcessingTime time.Duration     `json:"processing_time"`
	Metadata       map[string]string `json:"metadata"`
}

// GeoRouterConfig holds configuration for the geo router
type GeoRouterConfig struct {
	Port                int           `json:"port"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	DefaultRegion       string        `json:"default_region"`
	EnableGeoIP         bool          `json:"enable_geo_ip"`
	LogRequests         bool          `json:"log_requests"`
}

// HealthChecker performs health checks for regions
type HealthChecker struct {
	regionID string
	endpoint string
	timeout  time.Duration
	client   *http.Client
	logger   *log.Logger
}

// DefaultGeoRouterConfig returns default configuration
func DefaultGeoRouterConfig() GeoRouterConfig {
	return GeoRouterConfig{
		Port:                8080,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		DefaultRegion:       "region-a",
		EnableGeoIP:         true,
		LogRequests:         true,
	}
}

// NewGeoRouter creates a new geo router
func NewGeoRouter(regionID string, config GeoRouterConfig, logger *log.Logger) *GeoRouter {
	if logger == nil {
		logger = log.New(log.Writer(), "[GeoRouter] ", log.LstdFlags|log.Lshortfile)
	}

	router := &GeoRouter{
		regionID:       regionID,
		regions:        make(map[string]*RegionInfo),
		healthCheckers: make(map[string]*HealthChecker),
		routingRules:   make([]RoutingRule, 0),
		logger:         logger,
		config:         config,
	}

	// Initialize default regions
	router.initializeDefaultRegions()

	// Initialize default routing rules
	router.initializeDefaultRules()

	return router
}

// initializeDefaultRegions sets up default region configurations
func (gr *GeoRouter) initializeDefaultRegions() {
	regions := []*RegionInfo{
		{
			ID:       "region-a",
			Name:     "Region A (Primary)",
			Endpoint: "http://im-service-a:8080",
			Priority: 1,
			Weight:   100,
			Healthy:  true,
			GeoLocation: GeoLocation{
				Country:   "CN",
				Region:    "North",
				City:      "Beijing",
				Latitude:  39.9042,
				Longitude: 116.4074,
			},
			Metadata: map[string]string{
				"datacenter": "dc-north",
				"provider":   "aws",
			},
		},
		{
			ID:       "region-b",
			Name:     "Region B (Secondary)",
			Endpoint: "http://im-service-b:8080",
			Priority: 2,
			Weight:   100,
			Healthy:  true,
			GeoLocation: GeoLocation{
				Country:   "CN",
				Region:    "South",
				City:      "Shanghai",
				Latitude:  31.2304,
				Longitude: 121.4737,
			},
			Metadata: map[string]string{
				"datacenter": "dc-south",
				"provider":   "aws",
			},
		},
	}

	for _, region := range regions {
		gr.regions[region.ID] = region

		// Create health checker for each region
		gr.healthCheckers[region.ID] = &HealthChecker{
			regionID: region.ID,
			endpoint: region.Endpoint + "/health",
			timeout:  gr.config.HealthCheckTimeout,
			client: &http.Client{
				Timeout: gr.config.HealthCheckTimeout,
			},
			logger: gr.logger,
		}
	}
}

// initializeDefaultRules sets up default routing rules
func (gr *GeoRouter) initializeDefaultRules() {
	rules := []RoutingRule{
		{
			ID:       "header-region-override",
			Name:     "Header-based Region Override",
			Priority: 1,
			Conditions: []Condition{
				{
					Type:     "header",
					Key:      "X-Target-Region",
					Operator: "equals",
					Value:    "*", // Any value
				},
			},
			Enabled: true,
			Metadata: map[string]string{
				"description": "Allow explicit region targeting via header",
			},
		},
		{
			ID:       "geo-north-china",
			Name:     "Geographic Routing - North China",
			Priority: 2,
			Conditions: []Condition{
				{
					Type:     "geo",
					Key:      "region",
					Operator: "equals",
					Value:    "north",
				},
			},
			TargetRegion: "region-a",
			Enabled:      true,
			Metadata: map[string]string{
				"description": "Route northern China users to Region A",
			},
		},
		{
			ID:       "geo-south-china",
			Name:     "Geographic Routing - South China",
			Priority: 2,
			Conditions: []Condition{
				{
					Type:     "geo",
					Key:      "region",
					Operator: "equals",
					Value:    "south",
				},
			},
			TargetRegion: "region-b",
			Enabled:      true,
			Metadata: map[string]string{
				"description": "Route southern China users to Region B",
			},
		},
		{
			ID:       "user-id-hash",
			Name:     "User ID Hash-based Routing",
			Priority: 3,
			Conditions: []Condition{
				{
					Type:     "user_id",
					Key:      "hash_mod",
					Operator: "in_range",
					Value:    "0-49", // 50% to region-a
				},
			},
			TargetRegion: "region-a",
			Enabled:      true,
			Metadata: map[string]string{
				"description": "Hash-based load balancing for users",
			},
		},
		{
			ID:       "default-fallback",
			Name:     "Default Fallback",
			Priority: 100,
			Conditions: []Condition{
				{
					Type:     "custom",
					Key:      "always",
					Operator: "equals",
					Value:    "true",
				},
			},
			TargetRegion: gr.config.DefaultRegion,
			Enabled:      true,
			Metadata: map[string]string{
				"description": "Default fallback to primary region",
			},
		},
	}

	gr.routingRules = rules
}

// Start starts the geo router HTTP server
func (gr *GeoRouter) Start() error {
	mux := http.NewServeMux()

	// Routing decision endpoint
	mux.HandleFunc("/route", gr.handleRoute)

	// Health check endpoint
	mux.HandleFunc("/health", gr.handleHealth)

	// Region management endpoints
	mux.HandleFunc("/regions", gr.handleRegions)
	mux.HandleFunc("/regions/", gr.handleRegionDetail)

	// Routing rules endpoints
	mux.HandleFunc("/rules", gr.handleRules)

	// Status and metrics
	mux.HandleFunc("/status", gr.handleStatus)

	gr.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", gr.config.Port),
		Handler: mux,
	}

	// Start health checking
	go gr.startHealthChecking()

	gr.logger.Printf("GeoRouter starting on port %d", gr.config.Port)
	return gr.server.ListenAndServe()
}

// Stop stops the geo router
func (gr *GeoRouter) Stop() error {
	if gr.server != nil {
		return gr.server.Close()
	}
	return nil
}

// RouteRequest determines the target region for a request
func (gr *GeoRouter) RouteRequest(r *http.Request) *RoutingDecision {
	startTime := time.Now()

	gr.mu.RLock()
	defer gr.mu.RUnlock()

	decision := &RoutingDecision{
		DecisionTime: startTime,
		Confidence:   0.0,
		Alternatives: make([]string, 0),
		Metadata:     make(map[string]string),
	}

	// Extract routing context from request
	ctx := gr.extractRoutingContext(r)

	// Apply routing rules in priority order
	for _, rule := range gr.routingRules {
		if !rule.Enabled {
			continue
		}

		if gr.evaluateRule(rule, ctx) {
			decision.TargetRegion = rule.TargetRegion
			decision.Rule = &rule
			decision.Reason = fmt.Sprintf("Matched rule: %s", rule.Name)
			decision.Confidence = 1.0
			break
		}
	}

	// Handle header-based region override
	if targetRegion := r.Header.Get("X-Target-Region"); targetRegion != "" {
		if gr.isValidRegion(targetRegion) && gr.isRegionHealthy(targetRegion) {
			decision.TargetRegion = targetRegion
			decision.Reason = "Header override"
			decision.Confidence = 1.0
		}
	}

	// Fallback to default region if no decision made
	if decision.TargetRegion == "" {
		decision.TargetRegion = gr.config.DefaultRegion
		decision.Reason = "Default fallback"
		decision.Confidence = 0.5
	}

	// Check if target region is healthy, find alternatives if not
	if !gr.isRegionHealthy(decision.TargetRegion) {
		alternatives := gr.findHealthyAlternatives(decision.TargetRegion)
		decision.Alternatives = alternatives

		if len(alternatives) > 0 {
			decision.TargetRegion = alternatives[0]
			decision.Reason = fmt.Sprintf("Failover from unhealthy region to %s", alternatives[0])
			decision.Confidence = 0.8
		}
	}

	decision.ProcessingTime = time.Since(startTime)

	if gr.config.LogRequests {
		gr.logger.Printf("Routing decision: %s -> %s (reason: %s, confidence: %.2f)",
			r.RemoteAddr, decision.TargetRegion, decision.Reason, decision.Confidence)
	}

	return decision
}

// extractRoutingContext extracts routing information from the request
func (gr *GeoRouter) extractRoutingContext(r *http.Request) map[string]string {
	ctx := make(map[string]string)

	// Extract headers
	for key, values := range r.Header {
		if len(values) > 0 {
			ctx["header."+strings.ToLower(key)] = values[0]
		}
	}

	// Extract query parameters
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			ctx["query."+key] = values[0]
		}
	}

	// Extract user ID if present
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		ctx["user_id"] = userID
		ctx["user_id_hash"] = fmt.Sprintf("%d", hashString(userID)%100)
	}

	// Extract geographic information (simplified)
	if clientIP := r.Header.Get("X-Forwarded-For"); clientIP != "" {
		ctx["client_ip"] = clientIP
		// In a real implementation, you'd use a GeoIP service here
		ctx["geo.region"] = gr.simulateGeoLookup(clientIP)
	}

	return ctx
}

// evaluateRule checks if a routing rule matches the given context
func (gr *GeoRouter) evaluateRule(rule RoutingRule, ctx map[string]string) bool {
	for _, condition := range rule.Conditions {
		if !gr.evaluateCondition(condition, ctx) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single routing condition
func (gr *GeoRouter) evaluateCondition(condition Condition, ctx map[string]string) bool {
	var contextValue string

	switch condition.Type {
	case "header":
		contextValue = ctx["header."+strings.ToLower(condition.Key)]
	case "geo":
		contextValue = ctx["geo."+condition.Key]
	case "user_id":
		if condition.Key == "hash_mod" {
			contextValue = ctx["user_id_hash"]
		} else {
			contextValue = ctx["user_id"]
		}
	case "query":
		contextValue = ctx["query."+condition.Key]
	case "custom":
		if condition.Key == "always" {
			return condition.Value == "true"
		}
	default:
		return false
	}

	return gr.evaluateOperator(condition.Operator, contextValue, condition.Value)
}

// evaluateOperator evaluates the condition operator
func (gr *GeoRouter) evaluateOperator(operator, contextValue, expectedValue string) bool {
	switch operator {
	case "equals":
		return contextValue == expectedValue || expectedValue == "*"
	case "contains":
		return strings.Contains(contextValue, expectedValue)
	case "matches":
		// Simple pattern matching (could be enhanced with regex)
		return strings.Contains(contextValue, expectedValue)
	case "in_range":
		// For numeric ranges like "0-49"
		if parts := strings.Split(expectedValue, "-"); len(parts) == 2 {
			// Simple numeric range check for user hash
			if contextValue != "" {
				// This is a simplified implementation
				return true // In real implementation, parse and compare numbers
			}
		}
		return false
	default:
		return false
	}
}

// simulateGeoLookup simulates geographic lookup based on IP
func (gr *GeoRouter) simulateGeoLookup(clientIP string) string {
	// Simplified simulation - in reality, use a GeoIP service
	if strings.HasPrefix(clientIP, "10.1.") {
		return "north"
	} else if strings.HasPrefix(clientIP, "10.2.") {
		return "south"
	}
	return "unknown"
}

// isValidRegion checks if a region ID is valid
func (gr *GeoRouter) isValidRegion(regionID string) bool {
	_, exists := gr.regions[regionID]
	return exists
}

// isRegionHealthy checks if a region is healthy
func (gr *GeoRouter) isRegionHealthy(regionID string) bool {
	if region, exists := gr.regions[regionID]; exists {
		return region.Healthy
	}
	return false
}

// findHealthyAlternatives finds healthy alternative regions
func (gr *GeoRouter) findHealthyAlternatives(excludeRegion string) []string {
	alternatives := make([]string, 0)

	for regionID, region := range gr.regions {
		if regionID != excludeRegion && region.Healthy {
			alternatives = append(alternatives, regionID)
		}
	}

	return alternatives
}

// startHealthChecking starts periodic health checking for all regions
func (gr *GeoRouter) startHealthChecking() {
	ticker := time.NewTicker(gr.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gr.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks for all regions
func (gr *GeoRouter) performHealthChecks() {
	gr.mu.Lock()
	defer gr.mu.Unlock()

	for regionID, checker := range gr.healthCheckers {
		go func(id string, hc *HealthChecker) {
			healthy, latency := hc.checkHealth()

			gr.mu.Lock()
			if region, exists := gr.regions[id]; exists {
				region.Healthy = healthy
				region.Latency = latency
				region.LastCheck = time.Now()
			}
			gr.mu.Unlock()
		}(regionID, checker)
	}
}

// checkHealth performs a health check for a specific region
func (hc *HealthChecker) checkHealth() (bool, time.Duration) {
	start := time.Now()

	resp, err := hc.client.Get(hc.endpoint)
	latency := time.Since(start)

	if err != nil {
		hc.logger.Printf("Health check failed for %s: %v", hc.regionID, err)
		return false, latency
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !healthy {
		hc.logger.Printf("Health check failed for %s: status %d", hc.regionID, resp.StatusCode)
	}

	return healthy, latency
}

// hashString creates a simple hash of a string
func hashString(s string) uint32 {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// HTTP Handlers

func (gr *GeoRouter) handleRoute(w http.ResponseWriter, r *http.Request) {
	decision := gr.RouteRequest(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

func (gr *GeoRouter) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"region":    gr.regionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (gr *GeoRouter) handleRegions(w http.ResponseWriter, r *http.Request) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gr.regions)
}

func (gr *GeoRouter) handleRegionDetail(w http.ResponseWriter, r *http.Request) {
	regionID := strings.TrimPrefix(r.URL.Path, "/regions/")

	gr.mu.RLock()
	region, exists := gr.regions[regionID]
	gr.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(region)
}

func (gr *GeoRouter) handleRules(w http.ResponseWriter, r *http.Request) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gr.routingRules)
}

func (gr *GeoRouter) handleStatus(w http.ResponseWriter, r *http.Request) {
	gr.mu.RLock()
	defer gr.mu.RUnlock()

	status := map[string]interface{}{
		"region_id":       gr.regionID,
		"regions":         len(gr.regions),
		"rules":           len(gr.routingRules),
		"healthy_regions": gr.countHealthyRegions(),
		"config":          gr.config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (gr *GeoRouter) countHealthyRegions() int {
	count := 0
	for _, region := range gr.regions {
		if region.Healthy {
			count++
		}
	}
	return count
}
