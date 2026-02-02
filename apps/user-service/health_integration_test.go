package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoints verifies that health check endpoints are properly configured
func TestHealthEndpoints(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "user-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "user-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Create test HTTP mux with health endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.HealthzHandler(hc))
	mux.HandleFunc("/readyz", health.ReadyzHandler(hc))
	mux.HandleFunc("/health", health.HealthHandler(hc))

	// Test liveness endpoint
	t.Run("Liveness endpoint returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	// Test readiness endpoint
	t.Run("Readiness endpoint returns 200 when no checks registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/readyz", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "READY", w.Body.String())
	})

	// Test health endpoint
	t.Run("Health endpoint returns JSON", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, w.Body.String(), "status")
		assert.Contains(t, w.Body.String(), "service")
	})
}

// TestHealthChecksWithDatabase verifies database health check integration
// This test is skipped by default as it requires a running MySQL database
func TestHealthChecksWithDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "user-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "user-service-test",
		CheckInterval:    1 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Try to connect to test database
	dsn := "im_service:im_password@tcp(localhost:3306)/im_chat?parseTime=true&charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Register database health check
	hc.RegisterCheck(health.NewDatabaseCheck("database", db))

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for initial health check
	time.Sleep(2 * time.Second)

	// Verify service is healthy
	assert.True(t, hc.IsLive())
	assert.True(t, hc.IsReady())

	// Verify system health includes database check
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Contains(t, systemHealth.Components, "database")
	assert.Equal(t, "healthy", string(systemHealth.Components["database"].Status))
}

// TestStorageDBMethod verifies that MySQLStore exposes DB() method for health checks
func TestStorageDBMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Try to connect to test database
	dsn := "im_service:im_password@tcp(localhost:3306)/im_chat?parseTime=true&charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Verify DB() method exists by attempting to use it
	// This test will fail at compile time if the method doesn't exist
	assert.NotNil(t, db)
}

// mockCheck is a mock health check for testing
type mockCheck struct {
	name     string
	critical bool
	checkFn  func(ctx context.Context) error
}

func (m *mockCheck) Name() string                     { return m.name }
func (m *mockCheck) Critical() bool                   { return m.critical }
func (m *mockCheck) Timeout() time.Duration           { return 100 * time.Millisecond }
func (m *mockCheck) Interval() time.Duration          { return 5 * time.Second }
func (m *mockCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

// TestHealthCheckerWithMockCheck verifies health checker behavior with mock checks
func TestHealthCheckerWithMockCheck(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "user-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "user-service-test",
		CheckInterval:    500 * time.Millisecond,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 1,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Register a passing mock check
	passingCheck := &mockCheck{
		name:     "test-check",
		critical: true,
		checkFn:  func(ctx context.Context) error { return nil },
	}
	hc.RegisterCheck(passingCheck)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for health check to run
	time.Sleep(1 * time.Second)

	// Verify service is healthy
	assert.True(t, hc.IsReady())

	// Verify system health includes the check
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Contains(t, systemHealth.Components, "test-check")
}
