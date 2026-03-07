package health

import (
	"context"
	"testing"
	"time"
)

// MockCheck is a mock implementation of the Check interface
type MockCheck struct {
	name     string
	checkFn  func(ctx context.Context) error
	timeout  time.Duration
	interval time.Duration
	critical bool
}

func (m *MockCheck) Name() string                      { return m.name }
func (m *MockCheck) Check(ctx context.Context) error   { return m.checkFn(ctx) }
func (m *MockCheck) Timeout() time.Duration            { return m.timeout }
func (m *MockCheck) Interval() time.Duration           { return m.interval }
func (m *MockCheck) Critical() bool                    { return m.critical }

func TestNewHealthChecker(t *testing.T) {
	config := DefaultConfig("test-service")

	hc := NewHealthChecker(config, nil)

	if hc == nil {
		t.Fatal("NewHealthChecker returned nil")
	}

	if hc.config.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got %s", hc.config.ServiceName)
	}

	if hc.livenessProbe == nil {
		t.Error("Liveness probe not initialized")
	}

	if hc.readinessProbe == nil {
		t.Error("Readiness probe not initialized")
	}

	if hc.checks == nil {
		t.Error("Checks map not initialized")
	}

	if hc.results == nil {
		t.Error("Results map not initialized")
	}

	if hc.stopCh == nil {
		t.Error("Stop channel not initialized")
	}
}

func TestNewHealthChecker_InvalidConfig(t *testing.T) {
	config := Config{
		ServiceName: "", // Invalid - empty service name
	}

	// Should not panic, should use defaults
	hc := NewHealthChecker(config, nil)

	if hc == nil {
		t.Fatal("NewHealthChecker returned nil")
	}

	// Should have used default config
	if hc.config.CheckInterval != 5*time.Second {
		t.Errorf("Expected default check interval, got %v", hc.config.CheckInterval)
	}
}

func TestHealthChecker_RegisterCheck(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	check := &MockCheck{
		name:     "test-check",
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}

	hc.RegisterCheck(check)

	if len(hc.checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(hc.checks))
	}

	if _, exists := hc.checks["test-check"]; !exists {
		t.Error("Check not registered")
	}

	if len(hc.readinessProbe.checks) != 1 {
		t.Errorf("Expected 1 check in readiness probe, got %d", len(hc.readinessProbe.checks))
	}

	if _, exists := hc.results["test-check"]; !exists {
		t.Error("Check result not initialized")
	}
}

func TestHealthChecker_RegisterCheck_Nil(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Should not panic
	hc.RegisterCheck(nil)

	if len(hc.checks) != 0 {
		t.Errorf("Expected 0 checks, got %d", len(hc.checks))
	}
}

func TestHealthChecker_RegisterCheck_EmptyName(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	check := &MockCheck{
		name:     "", // Empty name
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}

	// Should not panic
	hc.RegisterCheck(check)

	if len(hc.checks) != 0 {
		t.Errorf("Expected 0 checks, got %d", len(hc.checks))
	}
}

func TestHealthChecker_RegisterCheck_Duplicate(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	check1 := &MockCheck{
		name:     "test-check",
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}

	check2 := &MockCheck{
		name:     "test-check", // Same name
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  200 * time.Millisecond,
		interval: 10 * time.Second,
		critical: false,
	}

	hc.RegisterCheck(check1)
	hc.RegisterCheck(check2)

	// Should have only one check (replaced)
	if len(hc.checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(hc.checks))
	}

	// Should be the second check
	registeredCheck := hc.checks["test-check"]
	if registeredCheck.Timeout() != 200*time.Millisecond {
		t.Error("Check was not replaced")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Start health checker
	err := hc.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Should be alive
	if !hc.IsLive() {
		t.Error("Expected health checker to be alive")
	}

	// Should be ready (no checks registered)
	if !hc.IsReady() {
		t.Error("Expected health checker to be ready")
	}

	// Stop health checker
	hc.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Should still report status (doesn't change on stop)
	// The liveness check might fail after stop, which is expected
}

func TestHealthChecker_IsLive(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	hc.Start()
	defer hc.Stop()

	// Give heartbeat time to start
	time.Sleep(100 * time.Millisecond)

	if !hc.IsLive() {
		t.Error("Expected health checker to be alive")
	}
}

func TestHealthChecker_IsReady(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Initially ready (no checks)
	if !hc.IsReady() {
		t.Error("Expected health checker to be ready initially")
	}

	// Register a passing check
	passingCheck := &MockCheck{
		name:     "passing-check",
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}
	hc.RegisterCheck(passingCheck)

	hc.Start()
	defer hc.Stop()

	// Wait for first check
	time.Sleep(200 * time.Millisecond)

	// Should still be ready
	if !hc.IsReady() {
		t.Error("Expected health checker to be ready with passing check")
	}
}

func TestHealthChecker_GetSystemHealth(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	health := hc.GetSystemHealth()

	if health == nil {
		t.Fatal("GetSystemHealth returned nil")
	}

	if health.Service != "test-service" {
		t.Errorf("Expected service 'test-service', got %s", health.Service)
	}

	if health.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", health.Status)
	}

	if health.Score != 1.0 {
		t.Errorf("Expected score 1.0 (no checks), got %f", health.Score)
	}

	if health.Components == nil {
		t.Error("Components map is nil")
	}
}

func TestHealthChecker_GetSystemHealth_WithChecks(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Register a passing check
	passingCheck := &MockCheck{
		name:     "passing-check",
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}
	hc.RegisterCheck(passingCheck)

	hc.Start()
	defer hc.Stop()

	// Wait for first check
	time.Sleep(200 * time.Millisecond)

	health := hc.GetSystemHealth()

	if health.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", health.Status)
	}

	if len(health.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(health.Components))
	}

	component, exists := health.Components["passing-check"]
	if !exists {
		t.Error("Component not found")
	}

	if component.Status != StatusHealthy {
		t.Errorf("Expected component status healthy, got %s", component.Status)
	}
}

func TestHealthChecker_GetComponentHealth(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Register a check
	check := &MockCheck{
		name:     "test-check",
		checkFn:  func(ctx context.Context) error { return nil },
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
	}
	hc.RegisterCheck(check)

	hc.Start()
	defer hc.Stop()

	// Wait for first check
	time.Sleep(200 * time.Millisecond)

	component := hc.GetComponentHealth("test-check")
	if component == nil {
		t.Fatal("GetComponentHealth returned nil")
	}

	if component.Name != "test-check" {
		t.Errorf("Expected component name 'test-check', got %s", component.Name)
	}

	// Non-existent component
	nonExistent := hc.GetComponentHealth("non-existent")
	if nonExistent != nil {
		t.Error("Expected nil for non-existent component")
	}
}

func TestHealthChecker_HealthScore_Calculation(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Manually set results to test score calculation
	hc.results["healthy1"] = &CheckResult{
		Name:   "healthy1",
		Status: StatusHealthy,
	}
	hc.results["healthy2"] = &CheckResult{
		Name:   "healthy2",
		Status: StatusHealthy,
	}
	hc.results["degraded"] = &CheckResult{
		Name:   "degraded",
		Status: StatusDegraded,
	}
	hc.results["critical"] = &CheckResult{
		Name:   "critical",
		Status: StatusCritical,
	}

	health := hc.GetSystemHealth()

	// Score = (1.0 + 1.0 + 0.5 + 0.0) / 4 = 0.625
	expectedScore := 0.625
	if health.Score != expectedScore {
		t.Errorf("Expected score %f, got %f", expectedScore, health.Score)
	}

	// Score 0.625 is between degraded (0.5) and healthy (0.8), so should be degraded
	if health.Status != StatusDegraded {
		t.Errorf("Expected status degraded, got %s", health.Status)
	}
}

func TestHealthChecker_MultipleStartStop(t *testing.T) {
	config := DefaultConfig("test-service")
	hc := NewHealthChecker(config, nil)

	// Start multiple times
	hc.Start()
	hc.Start()
	hc.Start()

	time.Sleep(100 * time.Millisecond)

	// Stop multiple times
	hc.Stop()
	hc.Stop()
	hc.Stop()

	// Should not panic
}
