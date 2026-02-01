package failover

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/multi-region/arbiter"
	"github.com/cuckoo-org/cuckoo/examples/multi-region/health"
)

// MockHealthChecker implements a mock health checker for testing
type MockHealthChecker struct {
	systemHealth *health.SystemHealth
	components   map[string]*health.ComponentHealth
}

func NewMockHealthChecker(regionID string) *MockHealthChecker {
	components := map[string]*health.ComponentHealth{
		"im-service": {
			Name:         "im-service",
			Status:       health.StatusHealthy,
			LastCheck:    time.Now(),
			ResponseTime: 50 * time.Millisecond,
		},
		"redis": {
			Name:         "redis",
			Status:       health.StatusHealthy,
			LastCheck:    time.Now(),
			ResponseTime: 10 * time.Millisecond,
		},
		"database": {
			Name:         "database",
			Status:       health.StatusHealthy,
			LastCheck:    time.Now(),
			ResponseTime: 100 * time.Millisecond,
		},
	}

	return &MockHealthChecker{
		components: components,
		systemHealth: &health.SystemHealth{
			Status:     health.StatusHealthy,
			RegionID:   regionID,
			Timestamp:  time.Now(),
			Components: components,
			Score:      1.0,
			Summary:    "All systems operational",
		},
	}
}

func (m *MockHealthChecker) GetSystemHealth() *health.SystemHealth {
	// Update timestamp
	m.systemHealth.Timestamp = time.Now()
	return m.systemHealth
}

func (m *MockHealthChecker) SetComponentHealth(name string, status health.HealthStatus, err string) {
	if component, exists := m.components[name]; exists {
		component.Status = status
		component.LastCheck = time.Now()
		component.Error = err
	}

	// Recalculate overall health
	m.recalculateHealth()
}

func (m *MockHealthChecker) SetAllComponentsUnhealthy() {
	for _, component := range m.components {
		component.Status = health.StatusCritical
		component.Error = "Simulated failure"
		component.LastCheck = time.Now()
	}
	m.recalculateHealth()
}

func (m *MockHealthChecker) recalculateHealth() {
	totalScore := 0.0
	for _, component := range m.components {
		switch component.Status {
		case health.StatusHealthy:
			totalScore += 1.0
		case health.StatusDegraded:
			totalScore += 0.5
		case health.StatusCritical:
			totalScore += 0.0
		}
	}

	score := totalScore / float64(len(m.components))
	m.systemHealth.Score = score

	if score >= 0.8 {
		m.systemHealth.Status = health.StatusHealthy
		m.systemHealth.Summary = "All systems operational"
	} else if score >= 0.5 {
		m.systemHealth.Status = health.StatusDegraded
		m.systemHealth.Summary = "Some systems degraded"
	} else {
		m.systemHealth.Status = health.StatusCritical
		m.systemHealth.Summary = "Critical issues detected"
	}
}

// MockArbiterClient implements a mock arbiter client for testing
type MockArbiterClient struct {
	currentLeader string
	healthReports map[string]map[string]bool
	electionCount int
}

func NewMockArbiterClient() *MockArbiterClient {
	return &MockArbiterClient{
		currentLeader: "region-a",
		healthReports: make(map[string]map[string]bool),
		electionCount: 0,
	}
}

func (m *MockArbiterClient) ElectPrimary(ctx context.Context, healthStatus map[string]bool) (*arbiter.ElectionResult, error) {
	m.electionCount++

	// Simple election logic: if current region is healthy, keep it; otherwise switch
	isHealthy := true
	for _, healthy := range healthStatus {
		if !healthy {
			isHealthy = false
			break
		}
	}

	var newLeader string
	var reason string

	if isHealthy && m.currentLeader != "" {
		newLeader = m.currentLeader
		reason = "current_leader_healthy"
	} else {
		// Switch to the other region
		if m.currentLeader == "region-a" {
			newLeader = "region-b"
		} else {
			newLeader = "region-a"
		}
		reason = "failover_election"
	}

	m.currentLeader = newLeader

	return &arbiter.ElectionResult{
		Leader:    newLeader,
		IsPrimary: false, // Will be set by caller
		Timestamp: time.Now(),
		TTL:       30,
		Reason:    reason,
	}, nil
}

func (m *MockArbiterClient) GetCurrentLeader(ctx context.Context) (string, error) {
	return m.currentLeader, nil
}

func (m *MockArbiterClient) WatchLeaderChanges(ctx context.Context, callback func(leader string)) error {
	// For testing, we'll just call the callback once with current leader
	callback(m.currentLeader)

	// Block until context is cancelled
	<-ctx.Done()
	return ctx.Err()
}

func (m *MockArbiterClient) ReportHealth(services map[string]bool) error {
	// Store health report (not used in current tests)
	return nil
}

func (m *MockArbiterClient) IsHealthy() bool {
	return true
}

func (m *MockArbiterClient) GetHealthStatus() map[string]bool {
	return map[string]bool{
		"im-service": true,
		"redis":      true,
		"database":   true,
	}
}

func (m *MockArbiterClient) Close() error {
	return nil
}

// Test helper functions

func createTestFailoverManager(regionID string) (*FailoverManager, *MockHealthChecker, *MockArbiterClient) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	mockHealthChecker := NewMockHealthChecker(regionID)
	mockArbiterClient := NewMockArbiterClient()

	config := DefaultConfig(regionID)
	config.HealthCheckInterval = 100 * time.Millisecond // Faster for testing
	config.FailureThreshold = 2                         // Lower threshold for testing
	config.FailoverTimeout = 5 * time.Second            // Shorter timeout for testing
	config.CooldownPeriod = 1 * time.Second             // Shorter cooldown for testing

	fm := NewFailoverManager(regionID, mockHealthChecker, mockArbiterClient, config, logger)

	return fm, mockHealthChecker, mockArbiterClient
}

// Test cases

func TestFailoverManager_Creation(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	if fm.regionID != "region-a" {
		t.Errorf("Expected regionID 'region-a', got '%s'", fm.regionID)
	}

	if fm.currentState != StateStandby {
		t.Errorf("Expected initial state 'standby', got '%s'", fm.currentState)
	}

	if fm.isPrimary {
		t.Errorf("Expected isPrimary to be false initially")
	}
}

func TestFailoverManager_InitialElection(t *testing.T) {
	fm, _, mockArbiter := createTestFailoverManager("region-a")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set this region as the leader in mock arbiter
	mockArbiter.currentLeader = "region-a"

	err := fm.performInitialElection(ctx)
	if err != nil {
		t.Fatalf("Initial election failed: %v", err)
	}

	state, isPrimary := fm.GetCurrentState()
	if !isPrimary {
		t.Errorf("Expected to be primary after election")
	}

	if state != StateActive {
		t.Errorf("Expected state 'active', got '%s'", state)
	}
}

func TestFailoverManager_HealthMonitoring(t *testing.T) {
	fm, mockHealth, _ := createTestFailoverManager("region-a")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start the failover manager
	err := fm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start failover manager: %v", err)
	}

	// Make this region primary
	fm.mu.Lock()
	fm.isPrimary = true
	fm.currentState = StateActive
	fm.mu.Unlock()

	// Simulate health failures
	mockHealth.SetAllComponentsUnhealthy()

	// Wait for health monitoring to detect failures
	time.Sleep(300 * time.Millisecond)

	// Check that consecutive failures are being tracked
	fm.mu.RLock()
	failures := fm.consecutiveFailures
	fm.mu.RUnlock()

	if failures == 0 {
		t.Errorf("Expected consecutive failures to be tracked, got %d", failures)
	}

	fm.Stop()
}

func TestFailoverManager_ManualFailover(t *testing.T) {
	fm, _, mockArbiter := createTestFailoverManager("region-a")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Make this region primary
	fm.mu.Lock()
	fm.isPrimary = true
	fm.currentState = StateActive
	fm.mu.Unlock()

	// Set up arbiter to switch to region-b
	mockArbiter.currentLeader = "region-a"

	// Start the failover manager
	err := fm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start failover manager: %v", err)
	}

	// Trigger manual failover
	err = fm.TriggerManualFailover("Testing manual failover")
	if err != nil {
		t.Fatalf("Failed to trigger manual failover: %v", err)
	}

	// Wait for failover to process
	time.Sleep(500 * time.Millisecond)

	// Check failover history
	history := fm.GetFailoverHistory()
	if len(history) == 0 {
		t.Errorf("Expected failover event in history")
	} else {
		event := history[0]
		if event.Type != "automatic_failover" {
			t.Errorf("Expected event type 'automatic_failover', got '%s'", event.Type)
		}
		if event.Trigger.Type != TriggerManual {
			t.Errorf("Expected trigger type 'manual', got '%s'", event.Trigger.Type)
		}
	}

	fm.Stop()
}

func TestFailoverManager_AutoFailover(t *testing.T) {
	fm, mockHealth, mockArbiter := createTestFailoverManager("region-a")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Make this region primary
	fm.mu.Lock()
	fm.isPrimary = true
	fm.currentState = StateActive
	fm.mu.Unlock()

	// Set up arbiter to switch to region-b on failure
	mockArbiter.currentLeader = "region-a"

	// Start the failover manager
	err := fm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start failover manager: %v", err)
	}

	// Simulate critical health failures
	mockHealth.SetAllComponentsUnhealthy()

	// Wait for auto-failover to trigger
	time.Sleep(500 * time.Millisecond)

	// Check that failover was triggered
	history := fm.GetFailoverHistory()
	if len(history) == 0 {
		t.Errorf("Expected auto-failover to be triggered")
	} else {
		event := history[0]
		if event.Trigger.Type != TriggerHealthCheck {
			t.Errorf("Expected trigger type 'health_check', got '%s'", event.Trigger.Type)
		}
	}

	fm.Stop()
}

func TestFailoverManager_CooldownPeriod(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	// Set a very short cooldown for testing
	fm.config.CooldownPeriod = 100 * time.Millisecond

	// Make this region primary
	fm.mu.Lock()
	fm.isPrimary = true
	fm.currentState = StateActive
	fm.lastFailoverTime = time.Now() // Set recent failover
	fm.mu.Unlock()

	// Try to trigger failover during cooldown
	err := fm.TriggerManualFailover("Testing cooldown")
	if err != nil {
		t.Fatalf("Failed to trigger failover: %v", err)
	}

	// Process should ignore the trigger due to cooldown
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	trigger := FailoverTrigger{
		Type:      TriggerManual,
		Reason:    "Testing cooldown",
		Severity:  SeverityCritical,
		Timestamp: time.Now(),
		Source:    "test",
	}

	err = fm.processFailoverTrigger(ctx, trigger)
	if err != nil {
		t.Errorf("Expected no error during cooldown processing, got: %v", err)
	}

	// Should have no failover events due to cooldown
	history := fm.GetFailoverHistory()
	if len(history) > 0 {
		t.Errorf("Expected no failover events during cooldown, got %d", len(history))
	}
}

func TestFailoverManager_StateTransitions(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	// Test initial state
	state, isPrimary := fm.GetCurrentState()
	if state != StateStandby || isPrimary {
		t.Errorf("Expected initial state (standby, false), got (%s, %v)", state, isPrimary)
	}

	// Test becoming primary
	fm.mu.Lock()
	fm.isPrimary = true
	fm.currentState = StateActive
	fm.mu.Unlock()

	state, isPrimary = fm.GetCurrentState()
	if state != StateActive || !isPrimary {
		t.Errorf("Expected active primary state (active, true), got (%s, %v)", state, isPrimary)
	}

	// Test failover state
	fm.mu.Lock()
	fm.currentState = StateFailover
	fm.mu.Unlock()

	state, isPrimary = fm.GetCurrentState()
	if state != StateFailover {
		t.Errorf("Expected failover state, got %s", state)
	}
}

func TestFailoverManager_Metrics(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	// Add some mock failover events
	fm.mu.Lock()
	fm.failoverHistory = []FailoverEvent{
		{
			ID:        "test-1",
			Status:    StatusCompleted,
			Duration:  15 * time.Second, // Within RTO
			StartTime: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "test-2",
			Status:    StatusCompleted,
			Duration:  45 * time.Second, // Exceeds RTO
			StartTime: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "test-3",
			Status:    StatusFailed,
			Duration:  10 * time.Second,
			StartTime: time.Now().Add(-10 * time.Minute),
		},
	}
	fm.mu.Unlock()

	metrics := fm.GetMetrics()

	if metrics["total_failovers"] != 3 {
		t.Errorf("Expected 3 total failovers, got %v", metrics["total_failovers"])
	}

	if metrics["successful_failovers"] != 2 {
		t.Errorf("Expected 2 successful failovers, got %v", metrics["successful_failovers"])
	}

	if metrics["failed_failovers"] != 1 {
		t.Errorf("Expected 1 failed failover, got %v", metrics["failed_failovers"])
	}

	if metrics["rto_violations"] != 1 {
		t.Errorf("Expected 1 RTO violation, got %v", metrics["rto_violations"])
	}

	successRate := metrics["success_rate"].(float64)
	if successRate < 0.66 || successRate > 0.67 {
		t.Errorf("Expected success rate ~0.67, got %v", successRate)
	}
}

func TestFailoverManager_Status(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	status := fm.GetStatus()

	expectedFields := []string{
		"region_id", "current_state", "is_primary", "last_failover_time",
		"consecutive_failures", "failover_count", "auto_failover_enabled",
		"rto_target", "rpo_target",
	}

	for _, field := range expectedFields {
		if _, exists := status[field]; !exists {
			t.Errorf("Expected status field '%s' not found", field)
		}
	}

	if status["region_id"] != "region-a" {
		t.Errorf("Expected region_id 'region-a', got %v", status["region_id"])
	}

	if status["auto_failover_enabled"] != true {
		t.Errorf("Expected auto_failover_enabled true, got %v", status["auto_failover_enabled"])
	}
}

func TestFailoverManager_EnableDisableAutoFailover(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	// Test initial state
	if !fm.config.EnableAutoFailover {
		t.Errorf("Expected auto-failover to be enabled by default")
	}

	// Disable auto-failover
	fm.EnableAutoFailover(false)
	if fm.config.EnableAutoFailover {
		t.Errorf("Expected auto-failover to be disabled")
	}

	// Re-enable auto-failover
	fm.EnableAutoFailover(true)
	if !fm.config.EnableAutoFailover {
		t.Errorf("Expected auto-failover to be re-enabled")
	}
}

func TestFailoverManager_Callbacks(t *testing.T) {
	fm, _, _ := createTestFailoverManager("region-a")

	var startCalled, completeCalled, trafficSwitchCalled bool

	fm.SetCallbacks(
		func(event FailoverEvent) error {
			startCalled = true
			return nil
		},
		func(event FailoverEvent) error {
			completeCalled = true
			return nil
		},
		func(from, to string) error {
			trafficSwitchCalled = true
			return nil
		},
	)

	// Verify callbacks are set
	if fm.onFailoverStart == nil {
		t.Errorf("Expected onFailoverStart callback to be set")
	}
	if fm.onFailoverComplete == nil {
		t.Errorf("Expected onFailoverComplete callback to be set")
	}
	if fm.onTrafficSwitch == nil {
		t.Errorf("Expected onTrafficSwitch callback to be set")
	}

	// Test that callbacks would be called (we can't easily test the actual calls
	// without running a full failover, which is complex in unit tests)
	if startCalled || completeCalled || trafficSwitchCalled {
		t.Errorf("Callbacks should not be called just by setting them")
	}
}

// Benchmark tests

func BenchmarkFailoverManager_GetStatus(b *testing.B) {
	fm, _, _ := createTestFailoverManager("region-a")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fm.GetStatus()
	}
}

func BenchmarkFailoverManager_GetMetrics(b *testing.B) {
	fm, _, _ := createTestFailoverManager("region-a")

	// Add some history for more realistic benchmark
	fm.mu.Lock()
	for i := 0; i < 50; i++ {
		fm.failoverHistory = append(fm.failoverHistory, FailoverEvent{
			ID:       fmt.Sprintf("bench-%d", i),
			Status:   StatusCompleted,
			Duration: time.Duration(i) * time.Second,
		})
	}
	fm.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fm.GetMetrics()
	}
}

// Integration test with real components (commented out as it requires external dependencies)
/*
func TestFailoverManager_Integration(t *testing.T) {
	// This test would require:
	// 1. Real Zookeeper instance
	// 2. Real health checker with actual services
	// 3. Real traffic switching mechanism

	// Skip for now as it requires complex setup
	t.Skip("Integration test requires external dependencies")
}
*/
