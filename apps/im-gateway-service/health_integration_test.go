package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/pingxin403/cuckoo/libs/health"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoints verifies all health endpoints are working
func TestHealthEndpoints(t *testing.T) {
	// Create mock observability
	obs, err := observability.New(observability.Config{
		ServiceName:   "im-gateway-service-test",
		EnableMetrics: false,
		EnableTracing: false,
		LogLevel:      "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Create health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-gateway-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkJSON      bool
	}{
		{
			name:           "Liveness endpoint",
			endpoint:       "/healthz",
			expectedStatus: http.StatusOK,
			checkJSON:      false,
		},
		{
			name:           "Readiness endpoint",
			endpoint:       "/readyz",
			expectedStatus: http.StatusOK,
			checkJSON:      false,
		},
		{
			name:           "Detailed health endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.endpoint, nil)
			w := httptest.NewRecorder()

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

			if tt.checkJSON {
				var healthResp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &healthResp)
				assert.NoError(t, err)
				assert.Contains(t, healthResp, "status")
				assert.Contains(t, healthResp, "service")
			}
		})
	}
}

// TestHealthChecksWithDependencies tests health checks with actual dependencies
func TestHealthChecksWithDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock observability
	obs, err := observability.New(observability.Config{
		ServiceName:   "im-gateway-service-test",
		EnableMetrics: false,
		EnableTracing: false,
		LogLevel:      "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Create health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-gateway-service-test",
		CheckInterval:    5 * time.Second,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 3,
	}, obs)

	// Create Redis client (will fail if Redis not available)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer func() { _ = redisClient.Close() }()

	// Register Redis health check
	hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))

	// Create mock HTTP server for im-service
	mockIMService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockIMService.Close()

	// Register HTTP health check for im-service
	hc.RegisterCheck(health.NewHTTPCheck("im-service", mockIMService.URL+"/healthz", true))

	// Create gateway service
	gatewayConfig := service.DefaultGatewayConfig()
	gateway := service.NewGatewayService(
		nil, // authClient
		nil, // registryClient
		nil, // imClient
		redisClient,
		gatewayConfig,
	)

	// Register WebSocket health check
	hc.RegisterCheck(NewWebSocketHealthCheck(gateway))

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Wait for health checks to run
	time.Sleep(1 * time.Second)

	// Check health status
	systemHealth := hc.GetSystemHealth()
	assert.NotNil(t, systemHealth)
	assert.Contains(t, systemHealth.Components, "redis")
	assert.Contains(t, systemHealth.Components, "im-service")
	assert.Contains(t, systemHealth.Components, "websocket-connections")
}

// TestReadinessMiddleware verifies readiness middleware behavior
func TestReadinessMiddleware(t *testing.T) {
	// Create mock observability
	obs, err := observability.New(observability.Config{
		ServiceName:   "im-gateway-service-test",
		EnableMetrics: false,
		EnableTracing: false,
		LogLevel:      "error",
	})
	require.NoError(t, err)
	defer func() { _ = obs.Shutdown(context.Background()) }()

	// Create health checker
	hc := health.NewHealthChecker(health.Config{
		ServiceName:      "im-gateway-service-test",
		CheckInterval:    500 * time.Millisecond,
		DefaultTimeout:   100 * time.Millisecond,
		FailureThreshold: 1,
	}, obs)

	// Start health checker
	err = hc.Start()
	require.NoError(t, err)
	defer hc.Stop()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Handler called"))
	})

	// Wrap with readiness middleware
	wrappedHandler := health.ReadinessMiddleware(hc)(testHandler)

	t.Run("Service ready - handler called", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Handler called")
	})

	// Note: Testing "not ready" scenario is complex due to anti-flapping logic
	// The health library requires multiple consecutive failures before marking not ready
	// This is tested in the health library's own test suite
}

// TestWebSocketHealthCheck tests the custom WebSocket health check
func TestWebSocketHealthCheck(t *testing.T) {
	// Create gateway service
	gatewayConfig := service.DefaultGatewayConfig()
	gateway := service.NewGatewayService(
		nil, // authClient
		nil, // registryClient
		nil, // imClient
		nil, // redisClient
		gatewayConfig,
	)

	// Create WebSocket health check
	wsCheck := NewWebSocketHealthCheck(gateway)

	assert.Equal(t, "websocket-connections", wsCheck.Name())
	assert.Equal(t, 100*time.Millisecond, wsCheck.Timeout())
	assert.Equal(t, 5*time.Second, wsCheck.Interval())
	assert.False(t, wsCheck.Critical())

	// Test health check
	ctx := context.Background()
	err := wsCheck.Check(ctx)
	assert.NoError(t, err)

	// Verify connection stats
	stats := gateway.GetConnectionStats()
	assert.GreaterOrEqual(t, stats.TotalConnections, int64(0))
	assert.GreaterOrEqual(t, stats.ActiveDevices, int64(0))
}

// TestGatewayGetConnectionStats verifies GetConnectionStats method
func TestGatewayGetConnectionStats(t *testing.T) {
	// Create gateway service
	gatewayConfig := service.DefaultGatewayConfig()
	gateway := service.NewGatewayService(
		nil, // authClient
		nil, // registryClient
		nil, // imClient
		nil, // redisClient
		gatewayConfig,
	)

	// Get connection stats
	stats := gateway.GetConnectionStats()

	// Verify stats structure
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalConnections)
	assert.Equal(t, int64(0), stats.ActiveDevices)
	assert.Equal(t, int64(0), stats.ErrorCount)
}
