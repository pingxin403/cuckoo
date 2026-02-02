package main

import (
	"context"
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
		ServiceName:    "auth-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "auth-service-test",
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

// TestHealthCheckerIntegration verifies health checker integration with auth-service
func TestHealthCheckerIntegration(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "auth-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "auth-service-test",
		CheckInterval:    1 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Note: auth-service is stateless with no dependencies
	// If dependencies are added, register health checks here

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for initial health check
	time.Sleep(2 * time.Second)

	// Verify service is healthy (no dependencies to fail)
	assert.True(t, hc.IsLive())
	assert.True(t, hc.IsReady())

	// Verify system health
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Equal(t, "healthy", string(systemHealth.Status))
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
		ServiceName:    "auth-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "auth-service-test",
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
