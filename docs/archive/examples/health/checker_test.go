package health

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
	_ "github.com/mattn/go-sqlite3"
)

// Mock health check for testing
type mockHealthCheck struct {
	name     string
	checkFn  func(ctx context.Context) error
	timeout  time.Duration
	interval time.Duration
}

func (m *mockHealthCheck) Name() string            { return m.name }
func (m *mockHealthCheck) Timeout() time.Duration  { return m.timeout }
func (m *mockHealthCheck) Interval() time.Duration { return m.interval }
func (m *mockHealthCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

func TestNewHealthChecker(t *testing.T) {
	config := DefaultConfig("test-region")
	logger := log.New(os.Stdout, "[Test] ", log.LstdFlags)

	checker := NewHealthChecker(config, logger)

	if checker == nil {
		t.Fatal("Expected non-nil health checker")
	}

	if checker.regionID != "test-region" {
		t.Errorf("Expected region ID 'test-region', got '%s'", checker.regionID)
	}

	if len(checker.components) != 0 {
		t.Errorf("Expected 0 components, got %d", len(checker.components))
	}
}

func TestHealthChecker_RegisterCheck(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	mockCheck := &mockHealthCheck{
		name:     "test-component",
		timeout:  1 * time.Second,
		interval: 5 * time.Second,
	}

	checker.RegisterCheck(mockCheck)

	if len(checker.components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(checker.components))
	}

	if _, exists := checker.components["test-component"]; !exists {
		t.Error("Expected component 'test-component' to be registered")
	}

	if _, exists := checker.results["test-component"]; !exists {
		t.Error("Expected result for 'test-component' to be initialized")
	}
}

func TestHealthChecker_GetSystemHealth(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	// Register healthy check
	healthyCheck := &mockHealthCheck{
		name: "healthy-component",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}
	checker.RegisterCheck(healthyCheck)

	// Register failing check
	failingCheck := &mockHealthCheck{
		name: "failing-component",
		checkFn: func(ctx context.Context) error {
			return fmt.Errorf("component failure")
		},
	}
	checker.RegisterCheck(failingCheck)

	// Perform checks manually
	checker.performCheck("healthy-component", healthyCheck)
	checker.performCheck("failing-component", failingCheck)

	health := checker.GetSystemHealth()

	if health.RegionID != "test-region" {
		t.Errorf("Expected region ID 'test-region', got '%s'", health.RegionID)
	}

	if len(health.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(health.Components))
	}

	healthyComponent := health.Components["healthy-component"]
	if healthyComponent.Status != StatusHealthy {
		t.Errorf("Expected healthy status, got %s", healthyComponent.Status)
	}

	failingComponent := health.Components["failing-component"]
	if failingComponent.Status != StatusCritical {
		t.Errorf("Expected critical status, got %s", failingComponent.Status)
	}

	// Health score should be 0.5 (1 healthy + 0 critical) / 2
	expectedScore := 0.5
	if health.Score != expectedScore {
		t.Errorf("Expected health score %.1f, got %.1f", expectedScore, health.Score)
	}

	// Overall status should be degraded (score >= 0.5 but < 0.8)
	if health.Status != StatusDegraded {
		t.Errorf("Expected degraded status, got %s", health.Status)
	}
}

func TestHealthChecker_CalculateHealthScore(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	tests := []struct {
		name       string
		components map[string]*ComponentHealth
		expected   float64
	}{
		{
			name:       "empty components",
			components: map[string]*ComponentHealth{},
			expected:   0.0,
		},
		{
			name: "all healthy",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusHealthy},
				"comp2": {Status: StatusHealthy},
			},
			expected: 1.0,
		},
		{
			name: "all critical",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusCritical},
				"comp2": {Status: StatusCritical},
			},
			expected: 0.0,
		},
		{
			name: "mixed status",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusHealthy},  // 1.0
				"comp2": {Status: StatusDegraded}, // 0.5
				"comp3": {Status: StatusCritical}, // 0.0
			},
			expected: 0.5, // (1.0 + 0.5 + 0.0) / 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := checker.calculateHealthScore(tt.components)
			if score != tt.expected {
				t.Errorf("Expected score %.1f, got %.1f", tt.expected, score)
			}
		})
	}
}

func TestHealthChecker_DetermineOverallStatus(t *testing.T) {
	config := DefaultConfig("test-region")
	checker := NewHealthChecker(config, nil)

	tests := []struct {
		score    float64
		expected HealthStatus
	}{
		{1.0, StatusHealthy},
		{0.9, StatusHealthy},
		{0.8, StatusHealthy},
		{0.7, StatusDegraded},
		{0.5, StatusDegraded},
		{0.4, StatusCritical},
		{0.0, StatusCritical},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%.1f", tt.score), func(t *testing.T) {
			status := checker.determineOverallStatus(tt.score)
			if status != tt.expected {
				t.Errorf("Expected status %s for score %.1f, got %s", tt.expected, tt.score, status)
			}
		})
	}
}

func TestMySQLHealthCheck(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	check := NewMySQLHealthCheck("test-mysql", db)

	if check.Name() != "test-mysql" {
		t.Errorf("Expected name 'test-mysql', got '%s'", check.Name())
	}

	ctx := context.Background()
	err = check.Check(ctx)
	if err != nil {
		t.Errorf("Expected healthy check, got error: %v", err)
	}

	// Test with closed database
	db.Close()
	err = check.Check(ctx)
	if err == nil {
		t.Error("Expected error with closed database")
	}

	// Test with nil database
	nilCheck := NewMySQLHealthCheck("nil-mysql", nil)
	err = nilCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with nil database")
	}
}

func TestRedisHealthCheck(t *testing.T) {
	// Create test storage
	config := storage.Config{
		RegionID:   "test-region",
		MemoryMode: true,
		TTL:        24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}
	defer store.Close()

	check := NewRedisHealthCheck("test-redis", store)

	if check.Name() != "test-redis" {
		t.Errorf("Expected name 'test-redis', got '%s'", check.Name())
	}

	ctx := context.Background()
	err = check.Check(ctx)
	if err != nil {
		t.Errorf("Expected healthy check, got error: %v", err)
	}

	// Test with nil storage
	nilCheck := NewRedisHealthCheck("nil-redis", nil)
	err = nilCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with nil storage")
	}
}

func TestKafkaHealthCheck(t *testing.T) {
	// Create test queue
	config := queue.DefaultConfig("test-region")
	logger := log.New(os.Stdout, "[TestQueue] ", log.LstdFlags)

	testQueue, err := queue.NewLocalQueue(config, logger)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Close()

	check := NewKafkaHealthCheck("test-kafka", testQueue)

	if check.Name() != "test-kafka" {
		t.Errorf("Expected name 'test-kafka', got '%s'", check.Name())
	}

	ctx := context.Background()
	err = check.Check(ctx)
	if err != nil {
		t.Errorf("Expected healthy check, got error: %v", err)
	}

	// Test with nil queue
	nilCheck := NewKafkaHealthCheck("nil-kafka", nil)
	err = nilCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with nil queue")
	}
}

func TestNetworkHealthCheck(t *testing.T) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer listener.Close()

	// Get the actual port
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	check := NewNetworkHealthCheck("test-network", "127.0.0.1", port)

	if check.Name() != "test-network" {
		t.Errorf("Expected name 'test-network', got '%s'", check.Name())
	}

	ctx := context.Background()
	err = check.Check(ctx)
	if err != nil {
		t.Errorf("Expected healthy check, got error: %v", err)
	}

	// Test with unreachable host
	unreachableCheck := NewNetworkHealthCheck("unreachable", "192.0.2.1", 12345)
	err = unreachableCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with unreachable host")
	}
}

func TestHTTPHealthCheck(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	check := NewHTTPHealthCheck("test-http", server.URL)

	if check.Name() != "test-http" {
		t.Errorf("Expected name 'test-http', got '%s'", check.Name())
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("Expected healthy check, got error: %v", err)
	}

	// Test with failing server
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	failingCheck := NewHTTPHealthCheck("failing-http", failingServer.URL)
	err = failingCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with failing HTTP server")
	}

	// Test with unreachable URL
	unreachableCheck := NewHTTPHealthCheck("unreachable-http", "http://192.0.2.1:12345/health")
	err = unreachableCheck.Check(ctx)
	if err == nil {
		t.Error("Expected error with unreachable HTTP server")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	// Register a mock check
	mockCheck := &mockHealthCheck{
		name:     "test-component",
		interval: 100 * time.Millisecond,
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}
	checker.RegisterCheck(mockCheck)

	// Start the checker
	err := checker.Start()
	if err != nil {
		t.Fatalf("Failed to start health checker: %v", err)
	}

	// Wait a bit for checks to run
	time.Sleep(200 * time.Millisecond)

	// Verify that checks have been performed
	health := checker.GetSystemHealth()
	component := health.Components["test-component"]
	if component.Status != StatusHealthy {
		t.Errorf("Expected healthy status after start, got %s", component.Status)
	}

	if component.LastCheck.IsZero() {
		t.Error("Expected LastCheck to be set after start")
	}

	// Stop the checker
	checker.Stop()

	// Verify that it stops gracefully (no panic or hanging)
}

func TestHealthChecker_GetComponentHealth(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	// Test non-existent component
	_, err := checker.GetComponentHealth("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent component")
	}

	// Register and test existing component
	mockCheck := &mockHealthCheck{name: "test-component"}
	checker.RegisterCheck(mockCheck)

	component, err := checker.GetComponentHealth("test-component")
	if err != nil {
		t.Errorf("Expected no error for existing component, got: %v", err)
	}

	if component.Name != "test-component" {
		t.Errorf("Expected component name 'test-component', got '%s'", component.Name)
	}
}

func TestHealthChecker_GenerateSummary(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	tests := []struct {
		name       string
		components map[string]*ComponentHealth
		status     HealthStatus
		expected   string
	}{
		{
			name: "all healthy",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusHealthy},
				"comp2": {Status: StatusHealthy},
			},
			status:   StatusHealthy,
			expected: "All systems operational (2/2 healthy)",
		},
		{
			name: "mixed status",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusHealthy},
				"comp2": {Status: StatusDegraded},
				"comp3": {Status: StatusCritical},
			},
			status:   StatusDegraded,
			expected: "Some systems degraded (1 healthy, 1 degraded, 1 critical)",
		},
		{
			name: "all critical",
			components: map[string]*ComponentHealth{
				"comp1": {Status: StatusCritical},
				"comp2": {Status: StatusCritical},
			},
			status:   StatusCritical,
			expected: "Critical issues detected (0 healthy, 0 degraded, 2 critical)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := checker.generateSummary(tt.components, tt.status)
			if summary != tt.expected {
				t.Errorf("Expected summary '%s', got '%s'", tt.expected, summary)
			}
		})
	}
}

// Benchmark tests
func BenchmarkHealthChecker_PerformCheck(b *testing.B) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	mockCheck := &mockHealthCheck{
		name: "benchmark-component",
		checkFn: func(ctx context.Context) error {
			// Simulate some work
			time.Sleep(1 * time.Millisecond)
			return nil
		},
	}

	checker.RegisterCheck(mockCheck)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.performCheck("benchmark-component", mockCheck)
	}
}

func BenchmarkHealthChecker_GetSystemHealth(b *testing.B) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	// Register multiple components
	for i := 0; i < 10; i++ {
		mockCheck := &mockHealthCheck{
			name: fmt.Sprintf("component-%d", i),
			checkFn: func(ctx context.Context) error {
				return nil
			},
		}
		checker.RegisterCheck(mockCheck)
		checker.performCheck(mockCheck.name, mockCheck)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.GetSystemHealth()
	}
}
