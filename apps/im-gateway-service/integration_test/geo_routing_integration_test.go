package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-gateway-service/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeoRouterIntegration tests the geo router component
func TestGeoRouterIntegration(t *testing.T) {
	config := routing.DefaultGeoRouterConfig()
	config.PeerRegions = map[string]string{
		"region-b": "http://region-b.example.com",
	}
	config.HealthCheckInterval = 5 * time.Second

	router := routing.NewGeoRouter("region-a", config, nil)

	t.Run("start and stop router", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		err = router.Stop()
		require.NoError(t, err)
	})

	t.Run("route request to local region", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Create test request
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-a")

		decision := router.RouteRequest(req)

		assert.Equal(t, "region-a", decision.TargetRegion)
		assert.Equal(t, "local", decision.Reason)
	})

	t.Run("route request based on region hint", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Request with region-b hint
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")

		decision := router.RouteRequest(req)

		// Should route to region-b if healthy
		assert.NotEmpty(t, decision.TargetRegion)
	})

	t.Run("fallback to local region when peer unhealthy", func(t *testing.T) {
		// Create router with unhealthy peer
		config := routing.DefaultGeoRouterConfig()
		config.PeerRegions = map[string]string{
			"region-b": "http://invalid-endpoint:9999",
		}
		config.HealthCheckInterval = 1 * time.Second

		router := routing.NewGeoRouter("region-a", config, nil)
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Wait for health check to fail
		time.Sleep(2 * time.Second)

		// Request with region-b hint
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")

		decision := router.RouteRequest(req)

		// Should fallback to local region
		assert.Equal(t, "region-a", decision.TargetRegion)
		assert.Contains(t, decision.Reason, "fallback")
	})
}

// TestGeoRouterHealthChecks tests health check functionality
func TestGeoRouterHealthChecks(t *testing.T) {
	// Create mock health check server
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer healthServer.Close()

	config := routing.DefaultGeoRouterConfig()
	config.PeerRegions = map[string]string{
		"region-b": healthServer.URL,
	}
	config.HealthCheckInterval = 1 * time.Second

	router := routing.NewGeoRouter("region-a", config, nil)

	t.Run("detect healthy peer", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Wait for health check
		time.Sleep(2 * time.Second)

		// Request to healthy peer
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")

		decision := router.RouteRequest(req)

		// Should route to region-b
		assert.Equal(t, "region-b", decision.TargetRegion)
	})

	t.Run("detect unhealthy peer after server stops", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Wait for initial health check
		time.Sleep(2 * time.Second)

		// Stop health server
		healthServer.Close()

		// Wait for health check to fail
		time.Sleep(3 * time.Second)

		// Request to now-unhealthy peer
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")

		decision := router.RouteRequest(req)

		// Should fallback to local
		assert.Equal(t, "region-a", decision.TargetRegion)
	})
}

// TestGeoRouterConcurrency tests concurrent routing decisions
func TestGeoRouterConcurrency(t *testing.T) {
	config := routing.DefaultGeoRouterConfig()
	config.PeerRegions = map[string]string{
		"region-b": "http://region-b.example.com",
	}

	router := routing.NewGeoRouter("region-a", config, nil)
	err := router.Start()
	require.NoError(t, err)
	defer router.Stop()

	// Make concurrent routing decisions
	done := make(chan routing.RoutingDecision, 100)
	for i := 0; i < 100; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/ws", nil)
			decision := router.RouteRequest(req)
			done <- decision
		}()
	}

	// Collect all decisions
	decisions := make([]routing.RoutingDecision, 100)
	for i := 0; i < 100; i++ {
		decisions[i] = <-done
	}

	// All decisions should be valid
	for _, decision := range decisions {
		assert.NotEmpty(t, decision.TargetRegion)
		assert.NotEmpty(t, decision.Reason)
	}
}

// TestGeoRouterFailover tests failover scenarios
func TestGeoRouterFailover(t *testing.T) {
	// Create two mock servers
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverB.Close()

	config := routing.DefaultGeoRouterConfig()
	config.PeerRegions = map[string]string{
		"region-b": serverB.URL,
	}
	config.HealthCheckInterval = 1 * time.Second
	config.FailoverEnabled = true

	router := routing.NewGeoRouter("region-a", config, nil)

	t.Run("automatic failover when peer becomes unhealthy", func(t *testing.T) {
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		// Wait for initial health check
		time.Sleep(2 * time.Second)

		// Verify region-b is healthy
		req := httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")
		decision := router.RouteRequest(req)
		assert.Equal(t, "region-b", decision.TargetRegion)

		// Simulate region-b failure
		serverB.Close()

		// Wait for health check to detect failure
		time.Sleep(3 * time.Second)

		// Should now failover to region-a
		req = httptest.NewRequest("GET", "/ws", nil)
		req.Header.Set("X-Region-Hint", "region-b")
		decision = router.RouteRequest(req)
		assert.Equal(t, "region-a", decision.TargetRegion)
		assert.Contains(t, decision.Reason, "fallback")
	})
}

// TestGeoRouterMetrics tests that routing metrics are recorded
func TestGeoRouterMetrics(t *testing.T) {
	config := routing.DefaultGeoRouterConfig()
	router := routing.NewGeoRouter("region-a", config, nil)

	err := router.Start()
	require.NoError(t, err)
	defer router.Stop()

	// Make multiple routing decisions
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/ws", nil)
		_ = router.RouteRequest(req)
	}

	// In a real test, you'd verify metrics were recorded
	// This would require access to the metrics registry
}
