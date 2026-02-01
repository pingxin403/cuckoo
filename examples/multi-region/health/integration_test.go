package health

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
	_ "github.com/mattn/go-sqlite3"
)

// TestHealthChecker_Integration tests the health checker with real components
func TestHealthChecker_Integration(t *testing.T) {
	// Create test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create test storage
	storageConfig := storage.Config{
		RegionID:   "test-region",
		MemoryMode: true,
		TTL:        24 * time.Hour,
	}

	store, err := storage.NewLocalStore(storageConfig)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}
	defer store.Close()

	// Create test queue
	queueConfig := queue.DefaultConfig("test-region")
	logger := log.New(os.Stdout, "[TestQueue] ", log.LstdFlags)

	testQueue, err := queue.NewLocalQueue(queueConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create test queue: %v", err)
	}
	defer testQueue.Close()

	// Create health checker
	config := DefaultConfig("test-region")
	config.CheckInterval = 1 * time.Second
	config.EnableMetrics = false // Disable for test

	checker := NewHealthChecker(config, nil)

	// Register health checks
	mysqlCheck := NewMySQLHealthCheck("mysql", db)
	checker.RegisterCheck(mysqlCheck)

	redisCheck := NewRedisHealthCheck("redis", store)
	checker.RegisterCheck(redisCheck)

	kafkaCheck := NewKafkaHealthCheck("kafka", testQueue)
	checker.RegisterCheck(kafkaCheck)

	// Start health checking
	err = checker.Start()
	if err != nil {
		t.Fatalf("Failed to start health checker: %v", err)
	}
	defer checker.Stop()

	// Wait for initial checks to complete
	time.Sleep(2 * time.Second)

	// Verify system health
	health := checker.GetSystemHealth()

	if health.RegionID != "test-region" {
		t.Errorf("Expected region ID 'test-region', got '%s'", health.RegionID)
	}

	if len(health.Components) != 3 {
		t.Errorf("Expected 3 components, got %d", len(health.Components))
	}

	// All components should be healthy
	expectedComponents := []string{"mysql", "redis", "kafka"}
	for _, componentName := range expectedComponents {
		component, exists := health.Components[componentName]
		if !exists {
			t.Errorf("Expected component '%s' not found", componentName)
			continue
		}

		if component.Status != StatusHealthy {
			t.Errorf("Expected component '%s' to be healthy, got %s: %s",
				componentName, component.Status, component.Error)
		}

		if component.LastCheck.IsZero() {
			t.Errorf("Expected component '%s' to have been checked", componentName)
		}
	}

	// Overall system should be healthy
	if health.Status != StatusHealthy {
		t.Errorf("Expected system status to be healthy, got %s: %s",
			health.Status, health.Summary)
	}

	// Health score should be 1.0 (all healthy)
	if health.Score != 1.0 {
		t.Errorf("Expected health score 1.0, got %.2f", health.Score)
	}
}

// TestHealthChecker_FailureScenarios tests various failure scenarios
func TestHealthChecker_FailureScenarios(t *testing.T) {
	config := DefaultConfig("test-region")
	config.CheckInterval = 500 * time.Millisecond
	config.EnableMetrics = false

	checker := NewHealthChecker(config, nil)

	// Register a failing check
	failingCheck := &mockHealthCheck{
		name: "failing-component",
		checkFn: func(ctx context.Context) error {
			return fmt.Errorf("simulated failure")
		},
	}
	checker.RegisterCheck(failingCheck)

	// Register a healthy check
	healthyCheck := &mockHealthCheck{
		name: "healthy-component",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}
	checker.RegisterCheck(healthyCheck)

	// Start health checking
	err := checker.Start()
	if err != nil {
		t.Fatalf("Failed to start health checker: %v", err)
	}
	defer checker.Stop()

	// Wait for checks to complete
	time.Sleep(1 * time.Second)

	// Verify system health
	health := checker.GetSystemHealth()

	// Check failing component
	failingComponent := health.Components["failing-component"]
	if failingComponent.Status != StatusCritical {
		t.Errorf("Expected failing component to be critical, got %s", failingComponent.Status)
	}

	if failingComponent.Error == "" {
		t.Error("Expected failing component to have error message")
	}

	// Check healthy component
	healthyComponent := health.Components["healthy-component"]
	if healthyComponent.Status != StatusHealthy {
		t.Errorf("Expected healthy component to be healthy, got %s", healthyComponent.Status)
	}

	// Overall system should be degraded (score = 0.5)
	expectedScore := 0.5
	if health.Score != expectedScore {
		t.Errorf("Expected health score %.1f, got %.1f", expectedScore, health.Score)
	}

	if health.Status != StatusDegraded {
		t.Errorf("Expected system status to be degraded, got %s", health.Status)
	}
}

// TestHealthChecker_ComponentHealthRetrieval tests individual component health retrieval
func TestHealthChecker_ComponentHealthRetrieval(t *testing.T) {
	checker := NewHealthChecker(DefaultConfig("test-region"), nil)

	// Register a test component
	testCheck := &mockHealthCheck{
		name: "test-component",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}
	checker.RegisterCheck(testCheck)

	// Perform initial check
	checker.performCheck("test-component", testCheck)

	// Test getting existing component
	component, err := checker.GetComponentHealth("test-component")
	if err != nil {
		t.Errorf("Expected no error getting existing component, got: %v", err)
	}

	if component.Name != "test-component" {
		t.Errorf("Expected component name 'test-component', got '%s'", component.Name)
	}

	if component.Status != StatusHealthy {
		t.Errorf("Expected component to be healthy, got %s", component.Status)
	}

	// Test getting non-existent component
	_, err = checker.GetComponentHealth("non-existent")
	if err == nil {
		t.Error("Expected error getting non-existent component")
	}
}

// TestHealthChecker_ConfigurableThresholds tests configurable health thresholds
func TestHealthChecker_ConfigurableThresholds(t *testing.T) {
	// Create config with custom thresholds
	config := Config{
		RegionID:        "test-region",
		CheckInterval:   1 * time.Second,
		DefaultTimeout:  1 * time.Second,
		HealthyScore:    0.9, // Higher threshold for healthy
		DegradedScore:   0.7, // Higher threshold for degraded
		EnableMetrics:   false,
		MetricsInterval: 30 * time.Second,
	}

	checker := NewHealthChecker(config, nil)

	// Create components with mixed health (score = 0.75)
	components := map[string]*ComponentHealth{
		"comp1": {Status: StatusHealthy},  // 1.0
		"comp2": {Status: StatusHealthy},  // 1.0
		"comp3": {Status: StatusDegraded}, // 0.5
		"comp4": {Status: StatusCritical}, // 0.0
	}
	// Total score: (1.0 + 1.0 + 0.5 + 0.0) / 4 = 0.625

	score := checker.calculateHealthScore(components)
	expectedScore := 0.625
	if score != expectedScore {
		t.Errorf("Expected score %.3f, got %.3f", expectedScore, score)
	}

	// With custom thresholds, score 0.625 should be critical (< 0.7)
	status := checker.determineOverallStatus(score)
	if status != StatusCritical {
		t.Errorf("Expected status critical with custom thresholds, got %s", status)
	}

	// Test with default thresholds for comparison
	defaultChecker := NewHealthChecker(DefaultConfig("test-region"), nil)
	defaultStatus := defaultChecker.determineOverallStatus(score)
	if defaultStatus != StatusDegraded {
		t.Errorf("Expected status degraded with default thresholds, got %s", defaultStatus)
	}
}

// BenchmarkHealthChecker_Integration benchmarks the integrated health checker
func BenchmarkHealthChecker_Integration(b *testing.B) {
	// Setup components
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	store, _ := storage.NewLocalStore(storage.Config{
		RegionID:   "bench-region",
		MemoryMode: true,
		TTL:        24 * time.Hour,
	})
	defer store.Close()

	testQueue, _ := queue.NewLocalQueue(queue.DefaultConfig("bench-region"), nil)
	defer testQueue.Close()

	// Setup health checker
	checker := NewHealthChecker(DefaultConfig("bench-region"), nil)
	checker.RegisterCheck(NewMySQLHealthCheck("mysql", db))
	checker.RegisterCheck(NewRedisHealthCheck("redis", store))
	checker.RegisterCheck(NewKafkaHealthCheck("kafka", testQueue))

	// Perform initial checks
	for name, check := range checker.components {
		checker.performCheck(name, check)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.GetSystemHealth()
	}
}
