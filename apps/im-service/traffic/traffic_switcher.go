package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TrafficSwitcher manages traffic distribution between regions
type TrafficSwitcher struct {
	redis  *redis.Client
	logger *log.Logger
	mu     sync.RWMutex

	// Current traffic configuration
	config *TrafficConfig

	// Event logging
	eventLog []TrafficEvent
}

// TrafficConfig defines traffic distribution rules
type TrafficConfig struct {
	RegionWeights map[string]int `json:"region_weights"` // region_id -> weight percentage
	DefaultRegion string         `json:"default_region"`
	LastUpdated   time.Time      `json:"last_updated"`
	UpdatedBy     string         `json:"updated_by"`
	Version       int64          `json:"version"`
}

// TrafficEvent represents a traffic switching event
type TrafficEvent struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // "proportional", "full_switch", "rollback"
	Timestamp  time.Time              `json:"timestamp"`
	FromConfig *TrafficConfig         `json:"from_config"`
	ToConfig   *TrafficConfig         `json:"to_config"`
	Reason     string                 `json:"reason"`
	Operator   string                 `json:"operator"`
	Duration   time.Duration          `json:"duration"`
	Status     string                 `json:"status"` // "started", "completed", "failed"
	Metadata   map[string]interface{} `json:"metadata"`
}

// TrafficSwitchRequest represents a traffic switching request
type TrafficSwitchRequest struct {
	Type          string         `json:"type"` // "proportional", "full_switch"
	RegionWeights map[string]int `json:"region_weights,omitempty"`
	TargetRegion  string         `json:"target_region,omitempty"`
	Reason        string         `json:"reason"`
	Operator      string         `json:"operator"`
	DryRun        bool           `json:"dry_run"`
}

// TrafficSwitchResponse represents the response to a traffic switch request
type TrafficSwitchResponse struct {
	Success           bool           `json:"success"`
	EventID           string         `json:"event_id"`
	Message           string         `json:"message"`
	OldConfig         *TrafficConfig `json:"old_config"`
	NewConfig         *TrafficConfig `json:"new_config"`
	EstimatedDuration time.Duration  `json:"estimated_duration"`
}

const (
	TrafficConfigKey = "traffic:config"
	TrafficEventsKey = "traffic:events"
	TrafficLockKey   = "traffic:lock"
)

// NewTrafficSwitcher creates a new traffic switcher
func NewTrafficSwitcher(redisClient *redis.Client, logger *log.Logger) *TrafficSwitcher {
	if logger == nil {
		logger = log.New(log.Writer(), "[TrafficSwitcher] ", log.LstdFlags|log.Lshortfile)
	}

	ts := &TrafficSwitcher{
		redis:    redisClient,
		logger:   logger,
		eventLog: make([]TrafficEvent, 0),
	}

	// Initialize with default configuration
	ts.initializeDefaultConfig()

	return ts
}

// initializeDefaultConfig sets up the default traffic configuration
func (ts *TrafficSwitcher) initializeDefaultConfig() {
	defaultConfig := &TrafficConfig{
		RegionWeights: map[string]int{
			"region-a": 100,
			"region-b": 0,
		},
		DefaultRegion: "region-a",
		LastUpdated:   time.Now(),
		UpdatedBy:     "system",
		Version:       1,
	}

	ts.config = defaultConfig

	// Try to load existing config from Redis
	if err := ts.loadConfigFromRedis(); err != nil {
		ts.logger.Printf("Failed to load config from Redis, using default: %v", err)
		// Save default config to Redis
		if err := ts.saveConfigToRedis(defaultConfig); err != nil {
			ts.logger.Printf("Failed to save default config to Redis: %v", err)
		}
	}
}

// GetCurrentConfig returns the current traffic configuration
func (ts *TrafficSwitcher) GetCurrentConfig() *TrafficConfig {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Create a copy to avoid race conditions
	config := &TrafficConfig{
		RegionWeights: make(map[string]int),
		DefaultRegion: ts.config.DefaultRegion,
		LastUpdated:   ts.config.LastUpdated,
		UpdatedBy:     ts.config.UpdatedBy,
		Version:       ts.config.Version,
	}

	for region, weight := range ts.config.RegionWeights {
		config.RegionWeights[region] = weight
	}

	return config
}

// SwitchTrafficProportional switches traffic with specified proportions
func (ts *TrafficSwitcher) SwitchTrafficProportional(ctx context.Context, regionWeights map[string]int, reason, operator string, dryRun bool) (*TrafficSwitchResponse, error) {
	// Validate weights
	if err := ts.validateRegionWeights(regionWeights); err != nil {
		return &TrafficSwitchResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid region weights: %v", err),
		}, err
	}

	// Acquire lock to prevent concurrent modifications
	lockAcquired, err := ts.acquireLock(ctx, 30*time.Second)
	if err != nil {
		return &TrafficSwitchResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to acquire lock: %v", err),
		}, err
	}
	if !lockAcquired {
		return &TrafficSwitchResponse{
			Success: false,
			Message: "Another traffic switch operation is in progress",
		}, fmt.Errorf("lock acquisition failed")
	}
	defer ts.releaseLock(ctx)

	oldConfig := ts.GetCurrentConfig()

	// Create new configuration
	newConfig := &TrafficConfig{
		RegionWeights: regionWeights,
		DefaultRegion: ts.findPrimaryRegion(regionWeights),
		LastUpdated:   time.Now(),
		UpdatedBy:     operator,
		Version:       oldConfig.Version + 1,
	}

	// Create event
	event := TrafficEvent{
		ID:         fmt.Sprintf("switch_%d", time.Now().UnixNano()),
		Type:       "proportional",
		Timestamp:  time.Now(),
		FromConfig: oldConfig,
		ToConfig:   newConfig,
		Reason:     reason,
		Operator:   operator,
		Status:     "started",
		Metadata: map[string]interface{}{
			"dry_run": dryRun,
		},
	}

	if dryRun {
		event.Status = "dry_run"
		return &TrafficSwitchResponse{
			Success:           true,
			EventID:           event.ID,
			Message:           "Dry run completed successfully",
			OldConfig:         oldConfig,
			NewConfig:         newConfig,
			EstimatedDuration: ts.estimateSwitchDuration(oldConfig, newConfig),
		}, nil
	}

	// Record event start
	ts.recordEvent(event)

	// Apply the configuration
	start := time.Now()
	if err := ts.applyTrafficConfig(ctx, newConfig); err != nil {
		event.Status = "failed"
		event.Duration = time.Since(start)
		event.Metadata["error"] = err.Error()
		ts.recordEvent(event)

		return &TrafficSwitchResponse{
			Success: false,
			EventID: event.ID,
			Message: fmt.Sprintf("Failed to apply traffic configuration: %v", err),
		}, err
	}

	// Update event with completion
	event.Status = "completed"
	event.Duration = time.Since(start)
	ts.recordEvent(event)

	ts.logger.Printf("Traffic switched proportionally: %v (operator: %s, reason: %s)",
		regionWeights, operator, reason)

	return &TrafficSwitchResponse{
		Success:           true,
		EventID:           event.ID,
		Message:           "Traffic switched successfully",
		OldConfig:         oldConfig,
		NewConfig:         newConfig,
		EstimatedDuration: event.Duration,
	}, nil
}

// SwitchTrafficFull performs a full traffic switch to a single region
func (ts *TrafficSwitcher) SwitchTrafficFull(ctx context.Context, targetRegion, reason, operator string, dryRun bool) (*TrafficSwitchResponse, error) {
	// Validate target region
	if !ts.isValidRegion(targetRegion) {
		return &TrafficSwitchResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid target region: %s", targetRegion),
		}, fmt.Errorf("invalid region: %s", targetRegion)
	}

	// Create weights for full switch (100% to target, 0% to others)
	regionWeights := map[string]int{
		"region-a": 0,
		"region-b": 0,
	}
	regionWeights[targetRegion] = 100

	// Use proportional switch with full weights
	response, err := ts.SwitchTrafficProportional(ctx, regionWeights, reason, operator, dryRun)
	if response != nil {
		// Update event type to indicate full switch
		if event := ts.getEventByID(response.EventID); event != nil {
			event.Type = "full_switch"
			event.Metadata["target_region"] = targetRegion
		}
	}

	return response, err
}

// GetTrafficEvents returns recent traffic switching events
func (ts *TrafficSwitcher) GetTrafficEvents(limit int) []TrafficEvent {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if limit <= 0 || limit > len(ts.eventLog) {
		limit = len(ts.eventLog)
	}

	// Return the most recent events
	start := len(ts.eventLog) - limit
	if start < 0 {
		start = 0
	}

	events := make([]TrafficEvent, limit)
	copy(events, ts.eventLog[start:])

	return events
}

// RouteRequest determines the target region for a request based on current traffic config
func (ts *TrafficSwitcher) RouteRequest(userID string) string {
	config := ts.GetCurrentConfig()

	// Simple hash-based routing with weights
	hash := ts.hashString(userID)

	// Calculate cumulative weights
	totalWeight := 0
	for _, weight := range config.RegionWeights {
		totalWeight += weight
	}

	if totalWeight == 0 {
		return config.DefaultRegion
	}

	// Determine target based on hash and weights
	hashMod := int(hash % uint32(totalWeight))
	cumulative := 0

	for region, weight := range config.RegionWeights {
		cumulative += weight
		if hashMod < cumulative {
			return region
		}
	}

	return config.DefaultRegion
}

// validateRegionWeights validates that region weights are valid
func (ts *TrafficSwitcher) validateRegionWeights(weights map[string]int) error {
	if len(weights) == 0 {
		return fmt.Errorf("no region weights specified")
	}

	totalWeight := 0
	for region, weight := range weights {
		if !ts.isValidRegion(region) {
			return fmt.Errorf("invalid region: %s", region)
		}
		if weight < 0 || weight > 100 {
			return fmt.Errorf("invalid weight for region %s: %d (must be 0-100)", region, weight)
		}
		totalWeight += weight
	}

	if totalWeight != 100 {
		return fmt.Errorf("total weight must equal 100, got %d", totalWeight)
	}

	return nil
}

// isValidRegion checks if a region is valid
func (ts *TrafficSwitcher) isValidRegion(region string) bool {
	validRegions := []string{"region-a", "region-b"}
	for _, validRegion := range validRegions {
		if region == validRegion {
			return true
		}
	}
	return false
}

// findPrimaryRegion finds the region with the highest weight
func (ts *TrafficSwitcher) findPrimaryRegion(weights map[string]int) string {
	maxWeight := -1
	primaryRegion := "region-a" // default

	for region, weight := range weights {
		if weight > maxWeight {
			maxWeight = weight
			primaryRegion = region
		}
	}

	return primaryRegion
}

// estimateSwitchDuration estimates how long a traffic switch will take
func (ts *TrafficSwitcher) estimateSwitchDuration(oldConfig, newConfig *TrafficConfig) time.Duration {
	// Simple estimation based on configuration complexity
	// In a real implementation, this would consider factors like:
	// - Number of active connections
	// - DNS propagation time
	// - Load balancer update time
	return 5 * time.Second
}

// applyTrafficConfig applies the new traffic configuration
func (ts *TrafficSwitcher) applyTrafficConfig(ctx context.Context, config *TrafficConfig) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Save to Redis
	if err := ts.saveConfigToRedis(config); err != nil {
		return fmt.Errorf("failed to save config to Redis: %w", err)
	}

	// Update in-memory config
	ts.config = config

	ts.logger.Printf("Applied traffic configuration: %+v", config)
	return nil
}

// loadConfigFromRedis loads traffic configuration from Redis
func (ts *TrafficSwitcher) loadConfigFromRedis() error {
	ctx := context.Background()

	data, err := ts.redis.Get(ctx, TrafficConfigKey).Result()
	if err != nil {
		return err
	}

	var config TrafficConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return err
	}

	ts.mu.Lock()
	ts.config = &config
	ts.mu.Unlock()

	return nil
}

// saveConfigToRedis saves traffic configuration to Redis
func (ts *TrafficSwitcher) saveConfigToRedis(config *TrafficConfig) error {
	ctx := context.Background()

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return ts.redis.Set(ctx, TrafficConfigKey, data, 0).Err()
}

// acquireLock acquires a distributed lock for traffic switching
func (ts *TrafficSwitcher) acquireLock(ctx context.Context, timeout time.Duration) (bool, error) {
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	result, err := ts.redis.SetNX(ctx, TrafficLockKey, lockValue, timeout).Result()
	if err != nil {
		return false, err
	}

	return result, nil
}

// releaseLock releases the distributed lock
func (ts *TrafficSwitcher) releaseLock(ctx context.Context) error {
	return ts.redis.Del(ctx, TrafficLockKey).Err()
}

// recordEvent records a traffic switching event
func (ts *TrafficSwitcher) recordEvent(event TrafficEvent) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Add to in-memory log
	ts.eventLog = append(ts.eventLog, event)

	// Keep only the last 100 events in memory
	if len(ts.eventLog) > 100 {
		ts.eventLog = ts.eventLog[1:]
	}

	// Save to Redis for persistence
	ctx := context.Background()
	eventData, _ := json.Marshal(event)
	ts.redis.LPush(ctx, TrafficEventsKey, eventData)
	ts.redis.LTrim(ctx, TrafficEventsKey, 0, 999) // Keep last 1000 events
}

// getEventByID retrieves an event by ID
func (ts *TrafficSwitcher) getEventByID(eventID string) *TrafficEvent {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for i := range ts.eventLog {
		if ts.eventLog[i].ID == eventID {
			return &ts.eventLog[i]
		}
	}
	return nil
}

// hashString creates a simple hash of a string
func (ts *TrafficSwitcher) hashString(s string) uint32 {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// HTTPHandler provides HTTP endpoints for traffic switching
type HTTPHandler struct {
	switcher *TrafficSwitcher
}

// NewHTTPHandler creates a new HTTP handler for traffic switching
func NewHTTPHandler(switcher *TrafficSwitcher) *HTTPHandler {
	return &HTTPHandler{switcher: switcher}
}

// HandleGetConfig returns the current traffic configuration
func (h *HTTPHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.switcher.GetCurrentConfig()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// HandleSwitchTraffic handles traffic switching requests
func (h *HTTPHandler) HandleSwitchTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TrafficSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var response *TrafficSwitchResponse
	var err error

	switch req.Type {
	case "proportional":
		response, err = h.switcher.SwitchTrafficProportional(ctx, req.RegionWeights, req.Reason, req.Operator, req.DryRun)
	case "full_switch":
		response, err = h.switcher.SwitchTrafficFull(ctx, req.TargetRegion, req.Reason, req.Operator, req.DryRun)
	default:
		http.Error(w, fmt.Sprintf("Invalid switch type: %s", req.Type), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetEvents returns recent traffic switching events
func (h *HTTPHandler) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	events := h.switcher.GetTrafficEvents(50) // Last 50 events

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// HandleRouteRequest handles routing requests for testing
func (h *HTTPHandler) HandleRouteRequest(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter required", http.StatusBadRequest)
		return
	}

	targetRegion := h.switcher.RouteRequest(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":       userID,
		"target_region": targetRegion,
		"config":        h.switcher.GetCurrentConfig(),
	})
}
