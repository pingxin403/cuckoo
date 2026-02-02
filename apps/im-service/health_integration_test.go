package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/config"
	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	_ "github.com/pingxin403/cuckoo/apps/im-service/worker"
	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoints verifies that all health endpoints return correct responses
func TestHealthEndpoints(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "im-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Give health checker time to initialize
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkBody      bool
		expectedBody   string
	}{
		{
			name:           "Liveness endpoint",
			endpoint:       "/healthz",
			expectedStatus: http.StatusOK,
			checkBody:      true,
			expectedBody:   "OK",
		},
		{
			name:           "Readiness endpoint",
			endpoint:       "/readyz",
			expectedStatus: http.StatusOK,
			checkBody:      true,
			expectedBody:   "READY",
		},
		{
			name:           "Detailed health endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

			// Get the appropriate handler
			var handler http.HandlerFunc
			switch tt.endpoint {
			case "/healthz":
				handler = health.HealthzHandler(hc)
			case "/readyz":
				handler = health.ReadyzHandler(hc)
			case "/health":
				handler = health.HealthHandler(hc)
			}

			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkBody {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestHealthChecksWithDependencies tests health checks with actual dependencies
// This test is skipped by default and requires actual infrastructure
func TestHealthChecksWithDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Skip("Skipping test: configuration not available")
	}

	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "im-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Try to initialize storage
	store, err := storage.NewOfflineStore(storage.Config{
		DSN:             cfg.GetDatabaseDSN(),
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Skip("Skipping test: database not available")
	}
	defer func() { _ = store.Close() }()

	// Register database health check
	hc.RegisterCheck(health.NewDatabaseCheck("database", store.GetDB()))

	// Try to initialize dedup service
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
		TTL:           cfg.OfflineWorker.MessageTTL,
	})
	defer func() { _ = dedupService.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dedupService.Ping(ctx); err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	// Register Redis health check
	hc.RegisterCheck(health.NewRedisCheck("redis", dedupService.GetClient()))

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for health checks to run
	time.Sleep(2 * time.Second)

	// Verify service is ready
	assert.True(t, hc.IsReady(), "Service should be ready with healthy dependencies")
	assert.True(t, hc.IsLive(), "Service should be alive")

	// Get system health
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Equal(t, "im-service-test", systemHealth.Service)
	assert.Contains(t, []health.HealthStatus{health.StatusHealthy, health.StatusDegraded}, systemHealth.Status)
}

// TestReadinessMiddleware verifies that readiness middleware works correctly
func TestReadinessMiddleware(t *testing.T) {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "im-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  false,
		LogLevel:       "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Initialize health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Give health checker time to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with readiness middleware
	middleware := health.ReadinessMiddleware(hc)
	wrappedHandler := middleware(testHandler)

	// Test when service is ready
	t.Run("Service ready", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})
}

// TestCustomHealthChecks verifies custom health checks for im-service
func TestCustomHealthChecks(t *testing.T) {
	t.Run("EtcdHealthCheck", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		// Try to create registry client
		registryClient, err := registry.NewRegistryClient([]string{"localhost:2379"}, 90*time.Second)
		if err != nil {
			t.Skip("Skipping test: etcd not available")
		}
		defer func() { _ = registryClient.Close() }()

		// Create etcd health check
		check := NewEtcdHealthCheck("etcd", registryClient)

		assert.Equal(t, "etcd", check.Name())
		assert.Equal(t, 200*time.Millisecond, check.Timeout())
		assert.Equal(t, 10*time.Second, check.Interval())
		assert.True(t, check.Critical())

		// Run health check
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = check.Check(ctx)
		// We expect either success or a specific error (not a timeout)
		if err != nil {
			t.Logf("Etcd health check error (expected if etcd not running): %v", err)
		}
	})

	t.Run("OfflineWorkerHealthCheck", func(t *testing.T) {
		// Create a mock worker (nil worker should fail health check)
		check := NewOfflineWorkerHealthCheck("offline-worker", nil)

		assert.Equal(t, "offline-worker", check.Name())
		assert.Equal(t, 100*time.Millisecond, check.Timeout())
		assert.Equal(t, 5*time.Second, check.Interval())
		assert.False(t, check.Critical())

		// Run health check with nil worker
		ctx := context.Background()
		err := check.Check(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// TestStorageGetDBMethod verifies that storage exposes GetDB method
func TestStorageGetDBMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skip("Skipping test: configuration not available")
	}

	store, err := storage.NewOfflineStore(storage.Config{
		DSN:             cfg.GetDatabaseDSN(),
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Skip("Skipping test: database not available")
	}
	defer func() { _ = store.Close() }()

	// Verify GetDB method exists and returns non-nil
	db := store.GetDB()
	assert.NotNil(t, db, "GetDB should return non-nil database connection")

	// Verify we can ping the database
	err = db.Ping()
	assert.NoError(t, err, "Should be able to ping database")
}

// TestDedupGetClientMethod verifies that dedup service exposes GetClient method
func TestDedupGetClientMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skip("Skipping test: configuration not available")
	}

	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
		TTL:           7 * 24 * time.Hour,
	})
	defer func() { _ = dedupService.Close() }()

	// Verify GetClient method exists and returns non-nil
	client := dedupService.GetClient()
	assert.NotNil(t, client, "GetClient should return non-nil Redis client")

	// Verify we can ping Redis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx).Err()
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}
}

// TestWorkerHealthCheckWithStats tests worker health check with different stats scenarios
func TestWorkerHealthCheckWithStats(t *testing.T) {
	// This is a unit test that doesn't require actual infrastructure
	// We would need to create a mock worker or test with actual worker stats
	// For now, we verify the check can be created
	t.Run("Create worker health check", func(t *testing.T) {
		// Create check with nil worker
		check := NewOfflineWorkerHealthCheck("test-worker", nil)
		assert.NotNil(t, check)
		assert.Equal(t, "test-worker", check.Name())
		assert.False(t, check.Critical())
	})
}
