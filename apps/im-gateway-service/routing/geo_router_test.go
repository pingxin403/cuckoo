package routing

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewGeoRouter(t *testing.T) {
	config := DefaultGeoRouterConfig()
	router := NewGeoRouter("region-a", config, nil)

	if router.regionID != "region-a" {
		t.Errorf("Expected region ID 'region-a', got '%s'", router.regionID)
	}

	if len(router.regions) != 2 {
		t.Errorf("Expected 2 regions, got %d", len(router.regions))
	}

	if len(router.routingRules) == 0 {
		t.Error("Expected routing rules to be initialized")
	}
}

func TestRouteRequest_HeaderOverride(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Target-Region", "region-b")

	decision := router.RouteRequest(req)

	if decision.TargetRegion != "region-b" {
		t.Errorf("Expected target region 'region-b', got '%s'", decision.TargetRegion)
	}

	if decision.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", decision.Confidence)
	}
}

func TestRouteRequest_DefaultFallback(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	req := httptest.NewRequest("GET", "/test", nil)
	decision := router.RouteRequest(req)

	if decision.TargetRegion != "region-a" {
		t.Errorf("Expected default target region 'region-a', got '%s'", decision.TargetRegion)
	}

	if !strings.Contains(decision.Reason, "fallback") {
		t.Errorf("Expected fallback reason, got '%s'", decision.Reason)
	}
}

func TestHealthChecking(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	if !router.isRegionHealthy("region-a") {
		t.Error("Region A should be healthy initially")
	}

	router.mu.Lock()
	router.regions["region-b"].Healthy = false
	router.mu.Unlock()

	if router.isRegionHealthy("region-b") {
		t.Error("Region B should be unhealthy after marking")
	}

	alternatives := router.findHealthyAlternatives("region-b")
	if len(alternatives) == 0 {
		t.Error("Should find healthy alternatives")
	}
}

func TestRouteRequest_Failover(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	router.mu.Lock()
	router.regions["region-b"].Healthy = false
	router.mu.Unlock()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Target-Region", "region-b")

	decision := router.RouteRequest(req)

	if decision.TargetRegion == "region-b" {
		t.Error("Should not route to unhealthy region")
	}

	if !strings.Contains(decision.Reason, "Failover") {
		t.Errorf("Expected failover reason, got '%s'", decision.Reason)
	}
}

func BenchmarkRouteRequest(b *testing.B) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.RouteRequest(req)
	}
}
