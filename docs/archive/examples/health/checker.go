package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	StatusHealthy  HealthStatus = "healthy"
	StatusDegraded HealthStatus = "degraded"
	StatusCritical HealthStatus = "critical"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name         string        `json:"name"`
	Status       HealthStatus  `json:"status"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
	Details      interface{}   `json:"details,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status     HealthStatus                `json:"status"`
	RegionID   string                      `json:"region_id"`
	Timestamp  time.Time                   `json:"timestamp"`
	Components map[string]*ComponentHealth `json:"components"`
	Score      float64                     `json:"score"`
	Summary    string                      `json:"summary"`
}

// HealthChecker manages health checks for multiple components
type HealthChecker struct {
	regionID   string
	components map[string]HealthCheck
	results    map[string]*ComponentHealth
	config     Config
	logger     *log.Logger
	mu         sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// HealthCheck defines the interface for component health checks
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) error
	Timeout() time.Duration
	Interval() time.Duration
}

// Config holds configuration for the health checker
type Config struct {
	RegionID        string        `json:"region_id"`
	CheckInterval   time.Duration `json:"check_interval"`
	DefaultTimeout  time.Duration `json:"default_timeout"`
	HealthyScore    float64       `json:"healthy_score"`  // >= 0.8
	DegradedScore   float64       `json:"degraded_score"` // >= 0.5
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// DefaultConfig returns a default configuration
func DefaultConfig(regionID string) Config {
	return Config{
		RegionID:        regionID,
		CheckInterval:   5 * time.Second,
		DefaultTimeout:  2 * time.Second,
		HealthyScore:    0.8,
		DegradedScore:   0.5,
		EnableMetrics:   true,
		MetricsInterval: 30 * time.Second,
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config Config, logger *log.Logger) *HealthChecker {
	if logger == nil {
		logger = log.New(log.Writer(), "[HealthChecker] ", log.LstdFlags)
	}

	return &HealthChecker{
		regionID:   config.RegionID,
		components: make(map[string]HealthCheck),
		results:    make(map[string]*ComponentHealth),
		config:     config,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
}

// RegisterCheck registers a health check for a component
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	name := check.Name()
	hc.components[name] = check
	hc.results[name] = &ComponentHealth{
		Name:      name,
		Status:    StatusCritical,
		LastCheck: time.Time{},
		Error:     "Not checked yet",
	}

	hc.logger.Printf("Registered health check: %s", name)
}

// Start begins running health checks
func (hc *HealthChecker) Start() error {
	hc.logger.Printf("Starting health checker for region: %s", hc.regionID)

	// Start health check routines for each component
	for name, check := range hc.components {
		hc.wg.Add(1)
		go hc.runHealthCheck(name, check)
	}

	// Start metrics collection if enabled
	if hc.config.EnableMetrics {
		hc.wg.Add(1)
		go hc.runMetricsCollection()
	}

	return nil
}

// Stop stops all health checks
func (hc *HealthChecker) Stop() {
	hc.logger.Printf("Stopping health checker")
	close(hc.stopCh)
	hc.wg.Wait()
}

// GetSystemHealth returns the current system health status
func (hc *HealthChecker) GetSystemHealth() *SystemHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Copy current results
	components := make(map[string]*ComponentHealth)
	for name, result := range hc.results {
		// Create a copy to avoid race conditions
		components[name] = &ComponentHealth{
			Name:         result.Name,
			Status:       result.Status,
			LastCheck:    result.LastCheck,
			ResponseTime: result.ResponseTime,
			Error:        result.Error,
			Details:      result.Details,
		}
	}

	// Calculate overall health score and status
	score := hc.calculateHealthScore(components)
	status := hc.determineOverallStatus(score)
	summary := hc.generateSummary(components, status)

	return &SystemHealth{
		Status:     status,
		RegionID:   hc.regionID,
		Timestamp:  time.Now(),
		Components: components,
		Score:      score,
		Summary:    summary,
	}
}

// GetComponentHealth returns health status for a specific component
func (hc *HealthChecker) GetComponentHealth(componentName string) (*ComponentHealth, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, exists := hc.results[componentName]
	if !exists {
		return nil, fmt.Errorf("component %s not found", componentName)
	}

	// Return a copy
	return &ComponentHealth{
		Name:         result.Name,
		Status:       result.Status,
		LastCheck:    result.LastCheck,
		ResponseTime: result.ResponseTime,
		Error:        result.Error,
		Details:      result.Details,
	}, nil
}

// runHealthCheck runs health checks for a specific component
func (hc *HealthChecker) runHealthCheck(name string, check HealthCheck) {
	defer hc.wg.Done()

	interval := check.Interval()
	if interval == 0 {
		interval = hc.config.CheckInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial check
	hc.performCheck(name, check)

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.performCheck(name, check)
		}
	}
}

// performCheck executes a single health check
func (hc *HealthChecker) performCheck(name string, check HealthCheck) {
	start := time.Now()

	timeout := check.Timeout()
	if timeout == 0 {
		timeout = hc.config.DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := check.Check(ctx)
	responseTime := time.Since(start)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	result := hc.results[name]
	result.LastCheck = start
	result.ResponseTime = responseTime

	if err != nil {
		result.Status = StatusCritical
		result.Error = err.Error()
		hc.logger.Printf("Health check failed for %s: %v (took %v)", name, err, responseTime)
	} else {
		// Determine status based on response time
		if responseTime > timeout/2 {
			result.Status = StatusDegraded
			result.Error = fmt.Sprintf("Slow response: %v", responseTime)
		} else {
			result.Status = StatusHealthy
			result.Error = ""
		}
		hc.logger.Printf("Health check passed for %s (took %v)", name, responseTime)
	}
}

// calculateHealthScore calculates overall health score (0.0 to 1.0)
func (hc *HealthChecker) calculateHealthScore(components map[string]*ComponentHealth) float64 {
	if len(components) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, component := range components {
		switch component.Status {
		case StatusHealthy:
			totalScore += 1.0
		case StatusDegraded:
			totalScore += 0.5
		case StatusCritical:
			totalScore += 0.0
		}
	}

	return totalScore / float64(len(components))
}

// determineOverallStatus determines overall system status based on score
func (hc *HealthChecker) determineOverallStatus(score float64) HealthStatus {
	if score >= hc.config.HealthyScore {
		return StatusHealthy
	} else if score >= hc.config.DegradedScore {
		return StatusDegraded
	} else {
		return StatusCritical
	}
}

// generateSummary generates a human-readable summary
func (hc *HealthChecker) generateSummary(components map[string]*ComponentHealth, status HealthStatus) string {
	healthy := 0
	degraded := 0
	critical := 0

	for _, component := range components {
		switch component.Status {
		case StatusHealthy:
			healthy++
		case StatusDegraded:
			degraded++
		case StatusCritical:
			critical++
		}
	}

	total := len(components)

	switch status {
	case StatusHealthy:
		return fmt.Sprintf("All systems operational (%d/%d healthy)", healthy, total)
	case StatusDegraded:
		return fmt.Sprintf("Some systems degraded (%d healthy, %d degraded, %d critical)", healthy, degraded, critical)
	case StatusCritical:
		return fmt.Sprintf("Critical issues detected (%d healthy, %d degraded, %d critical)", healthy, degraded, critical)
	default:
		return fmt.Sprintf("Unknown status (%d components)", total)
	}
}

// runMetricsCollection periodically logs health metrics
func (hc *HealthChecker) runMetricsCollection() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.logMetrics()
		}
	}
}

// logMetrics logs current health metrics
func (hc *HealthChecker) logMetrics() {
	health := hc.GetSystemHealth()

	metricsData := map[string]interface{}{
		"region_id":       health.RegionID,
		"overall_status":  health.Status,
		"health_score":    health.Score,
		"component_count": len(health.Components),
		"timestamp":       health.Timestamp.Unix(),
	}

	// Add component-specific metrics
	for name, component := range health.Components {
		metricsData[fmt.Sprintf("component_%s_status", name)] = component.Status
		metricsData[fmt.Sprintf("component_%s_response_time_ms", name)] = component.ResponseTime.Milliseconds()
	}

	metricsJSON, _ := json.Marshal(metricsData)
	hc.logger.Printf("Health metrics: %s", string(metricsJSON))
}

// Built-in health checks

// MySQLHealthCheck checks MySQL database connectivity
type MySQLHealthCheck struct {
	name     string
	db       *sql.DB
	timeout  time.Duration
	interval time.Duration
}

func NewMySQLHealthCheck(name string, db *sql.DB) *MySQLHealthCheck {
	return &MySQLHealthCheck{
		name:     name,
		db:       db,
		timeout:  2 * time.Second,
		interval: 5 * time.Second,
	}
}

func (m *MySQLHealthCheck) Name() string            { return m.name }
func (m *MySQLHealthCheck) Timeout() time.Duration  { return m.timeout }
func (m *MySQLHealthCheck) Interval() time.Duration { return m.interval }

func (m *MySQLHealthCheck) Check(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Simple ping check
	if err := m.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test query
	var result int
	err := m.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected query result: %d", result)
	}

	return nil
}

// RedisHealthCheck checks Redis connectivity (simulated with storage)
type RedisHealthCheck struct {
	name     string
	store    *storage.LocalStore
	timeout  time.Duration
	interval time.Duration
}

func NewRedisHealthCheck(name string, store *storage.LocalStore) *RedisHealthCheck {
	return &RedisHealthCheck{
		name:     name,
		store:    store,
		timeout:  1 * time.Second,
		interval: 5 * time.Second,
	}
}

func (r *RedisHealthCheck) Name() string            { return r.name }
func (r *RedisHealthCheck) Timeout() time.Duration  { return r.timeout }
func (r *RedisHealthCheck) Interval() time.Duration { return r.interval }

func (r *RedisHealthCheck) Check(ctx context.Context) error {
	if r.store == nil {
		return fmt.Errorf("storage connection is nil")
	}

	// Test storage operation
	stats, err := r.store.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("storage stats failed: %w", err)
	}

	// Verify we got some stats
	if len(stats) == 0 {
		return fmt.Errorf("no storage stats returned")
	}

	return nil
}

// KafkaHealthCheck checks Kafka connectivity (simulated with queue)
type KafkaHealthCheck struct {
	name     string
	queue    *queue.LocalQueue
	timeout  time.Duration
	interval time.Duration
}

func NewKafkaHealthCheck(name string, queue *queue.LocalQueue) *KafkaHealthCheck {
	return &KafkaHealthCheck{
		name:     name,
		queue:    queue,
		timeout:  2 * time.Second,
		interval: 10 * time.Second,
	}
}

func (k *KafkaHealthCheck) Name() string            { return k.name }
func (k *KafkaHealthCheck) Timeout() time.Duration  { return k.timeout }
func (k *KafkaHealthCheck) Interval() time.Duration { return k.interval }

func (k *KafkaHealthCheck) Check(ctx context.Context) error {
	if k.queue == nil {
		return fmt.Errorf("queue connection is nil")
	}

	// Test queue operation
	stats := k.queue.GetStats()
	if stats == nil {
		return fmt.Errorf("no queue stats returned")
	}

	// Check if queue is responsive
	regionID, exists := stats["region_id"]
	if !exists {
		return fmt.Errorf("queue region_id not found in stats")
	}

	if regionID == "" {
		return fmt.Errorf("queue region_id is empty")
	}

	return nil
}

// NetworkHealthCheck checks network connectivity to remote region
type NetworkHealthCheck struct {
	name       string
	remoteHost string
	remotePort int
	timeout    time.Duration
	interval   time.Duration
}

func NewNetworkHealthCheck(name, remoteHost string, remotePort int) *NetworkHealthCheck {
	return &NetworkHealthCheck{
		name:       name,
		remoteHost: remoteHost,
		remotePort: remotePort,
		timeout:    3 * time.Second,
		interval:   3 * time.Second,
	}
}

func (n *NetworkHealthCheck) Name() string            { return n.name }
func (n *NetworkHealthCheck) Timeout() time.Duration  { return n.timeout }
func (n *NetworkHealthCheck) Interval() time.Duration { return n.interval }

func (n *NetworkHealthCheck) Check(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", n.remoteHost, n.remotePort)

	conn, err := net.DialTimeout("tcp", address, n.timeout)
	if err != nil {
		return fmt.Errorf("network connection failed to %s: %w", address, err)
	}
	defer conn.Close()

	return nil
}

// HTTPHealthCheck checks HTTP endpoint health
type HTTPHealthCheck struct {
	name     string
	url      string
	timeout  time.Duration
	interval time.Duration
	client   *http.Client
}

func NewHTTPHealthCheck(name, url string) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		name:     name,
		url:      url,
		timeout:  2 * time.Second,
		interval: 5 * time.Second,
		client:   &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *HTTPHealthCheck) Name() string            { return h.name }
func (h *HTTPHealthCheck) Timeout() time.Duration  { return h.timeout }
func (h *HTTPHealthCheck) Interval() time.Duration { return h.interval }

func (h *HTTPHealthCheck) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP health check failed with status: %d", resp.StatusCode)
	}

	return nil
}
