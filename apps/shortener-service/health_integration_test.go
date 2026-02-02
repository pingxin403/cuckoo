package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoints verifies that health check endpoints are properly configured
func TestHealthEndpoints(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "shortener-service-test",
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
func TestHealthChecksWithDatabase(t *testing.T) {
	// Skip if no database is available
	t.Skip("Skipping database integration test - requires MySQL")

	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "shortener-service-test",
		CheckInterval:    1 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Create a test database connection
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/test")
	require.NoError(t, err)
	defer func() {
		_ = db.Close()
	}()

	// Register database health check
	hc.RegisterCheck(health.NewDatabaseCheck("database", db))

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for initial health check
	time.Sleep(2 * time.Second)

	// Verify database check is registered
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Contains(t, systemHealth.Components, "database")
}

// TestReadinessMiddleware verifies that readiness middleware works correctly
func TestReadinessMiddleware(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "shortener-service-test",
		CheckInterval:    500 * time.Millisecond,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 1,
		HealthyScore:     0.8,
		DegradedScore:    0.5,
	}, obs)

	// Start health checker (starts as ready with no checks)
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with readiness middleware
	wrappedHandler := health.ReadinessMiddleware(hc)(testHandler)

	// Test that requests pass through when ready
	t.Run("Middleware allows requests when ready", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})
}

// TestStorageDBMethod verifies that storage exposes DB() method for health checks
func TestStorageDBMethod(t *testing.T) {
	// This test verifies the API exists, actual DB connection testing is done in integration tests
	t.Run("MySQLStore has DB() method", func(t *testing.T) {
		// We can't actually create a store without a database, but we can verify the method exists
		// by checking the interface
		var _ interface {
			DB() *sql.DB
		} = (*storage.MySQLStore)(nil)
	})
}
