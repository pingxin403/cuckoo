package routing

import (
	"net/http"
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

	// Create request with region header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Target-Region", "region-b")

	decision := router.RouteRequest(req)

	if decision.TargetRegion != "region-b" {
		t.Errorf("Expected target region 'region-b', got '%s'", decision.TargetRegion)
	}

	if decision.Reason != "Header override" {
		t.Errorf("Expected reason 'Header override', got '%s'", decision.Reason)
	}

	if decision.Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %f", decision.Confidence)
	}
}

func TestRouteRequest_GeoRouting(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	// Test northern region routing
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.1.1.1") // Simulates north region IP

	decision := router.RouteRequest(req)

	// Should route to region-a for northern IPs
	if decision.TargetRegion != "region-a" {
		t.Errorf("Expected target region 'region-a' for northern IP, got '%s'", decision.TargetRegion)
	}

	// Test southern region routing
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Forwarded-For", "10.2.1.1") // Simulates south region IP

	decision2 := router.RouteRequest(req2)

	// Should route to region-b for southern IPs
	if decision2.TargetRegion != "region-b" {
		t.Errorf("Expected target region 'region-b' for southern IP, got '%s'", decision2.TargetRegion)
	}
}

func TestRouteRequest_DefaultFallback(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	// Create request with no special headers
	req := httptest.NewRequest("GET", "/test", nil)

	decision := router.RouteRequest(req)

	// Should fall back to default region
	if decision.TargetRegion != "region-a" {
		t.Errorf("Expected default target region 'region-a', got '%s'", decision.TargetRegion)
	}

	if !strings.Contains(decision.Reason, "fallback") {
		t.Errorf("Expected fallback reason, got '%s'", decision.Reason)
	}
}

func TestRouteRequest_UserIDHashing(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	// Test with user ID
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user123")

	decision := router.RouteRequest(req)

	// Should route based on user ID hash
	if decision.TargetRegion == "" {
		t.Error("Expected a target region for user ID routing")
	}

	// Test consistency - same user should always get same region
	decision2 := router.RouteRequest(req)
	if decision.TargetRegion != decision2.TargetRegion {
		t.Error("User ID routing should be consistent")
	}
}

func TestHealthChecking(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	// Initially all regions should be healthy
	if !router.isRegionHealthy("region-a") {
		t.Error("Region A should be healthy initially")
	}

	if !router.isRegionHealthy("region-b") {
		t.Error("Region B should be healthy initially")
	}

	// Test marking region as unhealthy
	router.mu.Lock()
	router.regions["region-b"].Healthy = false
	router.mu.Unlock()

	if router.isRegionHealthy("region-b") {
		t.Error("Region B should be unhealthy after marking")
	}

	// Test finding alternatives
	alternatives := router.findHealthyAlternatives("region-b")
	if len(alternatives) == 0 {
		t.Error("Should find healthy alternatives")
	}

	if alternatives[0] != "region-a" {
		t.Errorf("Expected alternative 'region-a', got '%s'", alternatives[0])
	}
}

func TestRouteRequest_Failover(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	// Mark region-b as unhealthy
	router.mu.Lock()
	router.regions["region-b"].Healthy = false
	router.mu.Unlock()

	// Request targeting unhealthy region
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Target-Region", "region-b")

	decision := router.RouteRequest(req)

	// Should failover to healthy region
	if decision.TargetRegion == "region-b" {
		t.Error("Should not route to unhealthy region")
	}

	if len(decision.Alternatives) == 0 {
		t.Error("Should provide alternatives for unhealthy region")
	}

	if !strings.Contains(decision.Reason, "Failover") {
		t.Errorf("Expected failover reason, got '%s'", decision.Reason)
	}
}

func TestEvaluateCondition(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	ctx := map[string]string{
		"header.x-target-region": "region-b",
		"geo.region":             "north",
		"user_id":                "user123",
		"user_id_hash":           "45",
	}

	tests := []struct {
		condition Condition
		expected  bool
	}{
		{
			condition: Condition{Type: "header", Key: "X-Target-Region", Operator: "equals", Value: "region-b"},
			expected:  true,
		},
		{
			condition: Condition{Type: "header", Key: "X-Target-Region", Operator: "equals", Value: "region-a"},
			expected:  false,
		},
		{
			condition: Condition{Type: "geo", Key: "region", Operator: "equals", Value: "north"},
			expected:  true,
		},
		{
			condition: Condition{Type: "geo", Key: "region", Operator: "equals", Value: "south"},
			expected:  false,
		},
		{
			condition: Condition{Type: "header", Key: "X-Target-Region", Operator: "equals", Value: "*"},
			expected:  true,
		},
	}

	for i, test := range tests {
		result := router.evaluateCondition(test.condition, ctx)
		if result != test.expected {
			t.Errorf("Test %d: expected %v, got %v for condition %+v", i, test.expected, result, test.condition)
		}
	}
}

func TestHTTPHandlers(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	tests := []struct {
		path           string
		expectedStatus int
	}{
		{"/health", http.StatusOK},
		{"/regions", http.StatusOK},
		{"/regions/region-a", http.StatusOK},
		{"/regions/nonexistent", http.StatusNotFound},
		{"/rules", http.StatusOK},
		{"/status", http.StatusOK},
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.path, nil)
		w := httptest.NewRecorder()

		switch test.path {
		case "/health":
			router.handleHealth(w, req)
		case "/regions":
			router.handleRegions(w, req)
		case "/rules":
			router.handleRules(w, req)
		case "/status":
			router.handleStatus(w, req)
		default:
			if strings.HasPrefix(test.path, "/regions/") {
				router.handleRegionDetail(w, req)
			}
		}

		if w.Code != test.expectedStatus {
			t.Errorf("Path %s: expected status %d, got %d", test.path, test.expectedStatus, w.Code)
		}
	}
}

func TestRouteRequestHandler(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	req := httptest.NewRequest("GET", "/route", nil)
	req.Header.Set("X-Target-Region", "region-b")
	w := httptest.NewRecorder()

	router.handleRoute(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check that response body contains JSON
	body := w.Body.String()
	if !strings.Contains(body, "target_region") {
		t.Error("Response should contain target_region field")
	}
}

func TestSimulateGeoLookup(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	tests := []struct {
		ip       string
		expected string
	}{
		{"10.1.1.1", "north"},
		{"10.2.1.1", "south"},
		{"192.168.1.1", "unknown"},
	}

	for _, test := range tests {
		result := router.simulateGeoLookup(test.ip)
		if result != test.expected {
			t.Errorf("IP %s: expected %s, got %s", test.ip, test.expected, result)
		}
	}
}

func TestHashString(t *testing.T) {
	// Test that hash function is deterministic
	hash1 := hashString("user123")
	hash2 := hashString("user123")

	if hash1 != hash2 {
		t.Error("Hash function should be deterministic")
	}

	// Test that different strings produce different hashes
	hash3 := hashString("user456")
	if hash1 == hash3 {
		t.Error("Different strings should produce different hashes")
	}
}

func TestProcessingTime(t *testing.T) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)

	req := httptest.NewRequest("GET", "/test", nil)
	decision := router.RouteRequest(req)

	if decision.ProcessingTime <= 0 {
		t.Error("Processing time should be recorded")
	}

	if decision.DecisionTime.IsZero() {
		t.Error("Decision time should be recorded")
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

func BenchmarkRouteRequestWithGeo(b *testing.B) {
	router := NewGeoRouter("region-a", DefaultGeoRouterConfig(), nil)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.1.1.1")
	req.Header.Set("X-User-ID", "user123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.RouteRequest(req)
	}
}
