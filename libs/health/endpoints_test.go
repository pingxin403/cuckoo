package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthzHandler(t *testing.T) {
	tests := []struct {
		name           string
		isLive         bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "service alive",
			isLive:         true,
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "service not alive",
			isLive:         false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "NOT ALIVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:    "test-service",
				CheckInterval:  1 * time.Second,
				DefaultTimeout: 100 * time.Millisecond,
			}, nil)

			// Start liveness probe
			hc.Start()
			defer hc.Stop()

			// Set liveness state
			if !tt.isLive {
				// Simulate heartbeat timeout
				hc.livenessProbe.lastHeartbeat.Store(time.Now().Add(-20 * time.Second))
			}

			// Create handler
			handler := HealthzHandler(hc)

			// Create test request
			req := httptest.NewRequest("GET", "/healthz", nil)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestReadyzHandler(t *testing.T) {
	tests := []struct {
		name           string
		isReady        bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "service ready",
			isReady:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   "READY",
		},
		{
			name:           "service not ready",
			isReady:        false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "NOT READY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:    "test-service",
				CheckInterval:  1 * time.Second,
				DefaultTimeout: 100 * time.Millisecond,
			}, nil)

			// Set readiness state
			if tt.isReady {
				hc.readinessProbe.isReady.Store(1)
			} else {
				hc.readinessProbe.isReady.Store(0)
			}

			// Create handler
			handler := ReadyzHandler(hc)

			// Create test request
			req := httptest.NewRequest("GET", "/readyz", nil)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupHealth    func(*HealthChecker)
		expectedStatus int
		validateBody   func(*testing.T, *HealthResponse)
	}{
		{
			name: "healthy service",
			setupHealth: func(hc *HealthChecker) {
				// Register a healthy check
				hc.RegisterCheck(&testHealthCheck{
					name:     "test-check",
					checkFn:  func(ctx context.Context) error { return nil },
					critical: true,
				})
				hc.Start()
				time.Sleep(100 * time.Millisecond) // Wait for first check
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, resp *HealthResponse) {
				if resp.Status != "healthy" {
					t.Errorf("expected status 'healthy', got %q", resp.Status)
				}
				if resp.Service != "test-service" {
					t.Errorf("expected service 'test-service', got %q", resp.Service)
				}
				if resp.Score < 0.8 {
					t.Errorf("expected score >= 0.8, got %.2f", resp.Score)
				}
			},
		},
		{
			name: "degraded service",
			setupHealth: func(hc *HealthChecker) {
				// Register a slow check (will be marked as degraded)
				hc.RegisterCheck(&testHealthCheck{
					name: "slow-check",
					checkFn: func(ctx context.Context) error {
						time.Sleep(60 * time.Millisecond) // Slow but not failing
						return nil
					},
					timeout:  100 * time.Millisecond,
					critical: true,
				})
				hc.Start()
				time.Sleep(200 * time.Millisecond) // Wait for check
			},
			expectedStatus: http.StatusOK, // Degraded still serves traffic
			validateBody: func(t *testing.T, resp *HealthResponse) {
				if resp.Status != "degraded" {
					t.Errorf("expected status 'degraded', got %q", resp.Status)
				}
			},
		},
		{
			name: "critical service",
			setupHealth: func(hc *HealthChecker) {
				// Register a failing check
				hc.RegisterCheck(&testHealthCheck{
					name:     "failing-check",
					checkFn:  func(ctx context.Context) error { return &testError{msg: "check failed"} },
					critical: true,
				})
				hc.Start()
				// Wait for failure threshold to be reached
				time.Sleep(time.Duration(hc.config.FailureThreshold+1) * hc.config.CheckInterval)
			},
			expectedStatus: http.StatusServiceUnavailable,
			validateBody: func(t *testing.T, resp *HealthResponse) {
				if resp.Status != "critical" {
					t.Errorf("expected status 'critical', got %q", resp.Status)
				}
				if resp.Score >= 0.5 {
					t.Errorf("expected score < 0.5, got %.2f", resp.Score)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:      "test-service",
				CheckInterval:    100 * time.Millisecond,
				DefaultTimeout:   100 * time.Millisecond,
				FailureThreshold: 3,
			}, nil)
			defer hc.Stop()

			// Setup health state
			if tt.setupHealth != nil {
				tt.setupHealth(hc)
			}

			// Create handler
			handler := HealthHandler(hc)

			// Create test request
			req := httptest.NewRequest("GET", "/health", nil)
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify content type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			// Parse response
			var response HealthResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Validate response body
			if tt.validateBody != nil {
				tt.validateBody(t, &response)
			}
		})
	}
}

func TestHealthHandler_ResponseFormat(t *testing.T) {
	// Create health checker with a check
	hc := NewHealthChecker(Config{
		ServiceName:    "test-service",
		CheckInterval:  1 * time.Second,
		DefaultTimeout: 100 * time.Millisecond,
	}, nil)

	// Register a check
	hc.RegisterCheck(&testHealthCheck{
		name:     "database",
		checkFn:  func(ctx context.Context) error { return nil },
		critical: true,
	})

	hc.Start()
	defer hc.Stop()

	// Wait for first check
	time.Sleep(100 * time.Millisecond)

	// Create handler
	handler := HealthHandler(hc)

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Parse response
	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response structure
	if response.Status == "" {
		t.Error("status field is empty")
	}
	if response.Service == "" {
		t.Error("service field is empty")
	}
	if response.Timestamp == "" {
		t.Error("timestamp field is empty")
	}
	if response.Summary == "" {
		t.Error("summary field is empty")
	}
	if response.Components == nil {
		t.Error("components field is nil")
	}

	// Verify component structure
	if comp, exists := response.Components["database"]; exists {
		if comp.Name != "database" {
			t.Errorf("expected component name 'database', got %q", comp.Name)
		}
		if comp.Status == "" {
			t.Error("component status is empty")
		}
		if comp.LastCheck == "" {
			t.Error("component last_check is empty")
		}
		// ResponseTimeMs should be >= 0
		if comp.ResponseTimeMs < 0 {
			t.Errorf("expected response_time_ms >= 0, got %.2f", comp.ResponseTimeMs)
		}
	} else {
		t.Error("database component not found in response")
	}
}

func TestRegisterHealthEndpoints(t *testing.T) {
	// Create health checker
	hc := NewHealthChecker(Config{
		ServiceName:    "test-service",
		CheckInterval:  1 * time.Second,
		DefaultTimeout: 100 * time.Millisecond,
	}, nil)

	hc.Start()
	defer hc.Stop()

	// Create mux and register endpoints
	mux := http.NewServeMux()
	RegisterHealthEndpoints(mux, hc)

	// Test each endpoint
	endpoints := []struct {
		path           string
		expectedStatus int
	}{
		{"/healthz", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/health", http.StatusOK},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", ep.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != ep.expectedStatus {
				t.Errorf("expected status %d for %s, got %d", ep.expectedStatus, ep.path, rec.Code)
			}
		})
	}
}

func TestFormatHealthResponse(t *testing.T) {
	// Create system health
	now := time.Now()
	systemHealth := &SystemHealth{
		Status:    StatusHealthy,
		Service:   "test-service",
		Timestamp: now,
		Score:     0.85,
		Summary:   "All systems operational",
		Components: map[string]*ComponentHealth{
			"database": {
				Name:         "database",
				Status:       StatusHealthy,
				LastCheck:    now,
				ResponseTime: 15 * time.Millisecond,
				Error:        "",
			},
			"redis": {
				Name:         "redis",
				Status:       StatusDegraded,
				LastCheck:    now,
				ResponseTime: 75 * time.Millisecond,
				Error:        "slow response: 75ms",
			},
		},
	}

	// Format response
	response := formatHealthResponse(systemHealth)

	// Verify response
	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", response.Status)
	}
	if response.Service != "test-service" {
		t.Errorf("expected service 'test-service', got %q", response.Service)
	}
	if response.Score != 0.85 {
		t.Errorf("expected score 0.85, got %.2f", response.Score)
	}
	if response.Summary != "All systems operational" {
		t.Errorf("expected summary 'All systems operational', got %q", response.Summary)
	}

	// Verify timestamp format (RFC3339)
	if _, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
		t.Errorf("timestamp not in RFC3339 format: %v", err)
	}

	// Verify components
	if len(response.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(response.Components))
	}

	// Verify database component
	if db, exists := response.Components["database"]; exists {
		if db.Name != "database" {
			t.Errorf("expected name 'database', got %q", db.Name)
		}
		if db.Status != "healthy" {
			t.Errorf("expected status 'healthy', got %q", db.Status)
		}
		if db.ResponseTimeMs != 15.0 {
			t.Errorf("expected response time 15.0ms, got %.2f", db.ResponseTimeMs)
		}
		if db.Error != "" {
			t.Errorf("expected empty error, got %q", db.Error)
		}
	} else {
		t.Error("database component not found")
	}

	// Verify redis component
	if redis, exists := response.Components["redis"]; exists {
		if redis.Status != "degraded" {
			t.Errorf("expected status 'degraded', got %q", redis.Status)
		}
		if redis.ResponseTimeMs != 75.0 {
			t.Errorf("expected response time 75.0ms, got %.2f", redis.ResponseTimeMs)
		}
		if redis.Error != "slow response: 75ms" {
			t.Errorf("expected error 'slow response: 75ms', got %q", redis.Error)
		}
	} else {
		t.Error("redis component not found")
	}
}

// testHealthCheck for testing endpoints
type testHealthCheck struct {
	name     string
	checkFn  func(context.Context) error
	timeout  time.Duration
	interval time.Duration
	critical bool
}

func (m *testHealthCheck) Name() string {
	return m.name
}

func (m *testHealthCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

func (m *testHealthCheck) Timeout() time.Duration {
	if m.timeout > 0 {
		return m.timeout
	}
	return 100 * time.Millisecond
}

func (m *testHealthCheck) Interval() time.Duration {
	if m.interval > 0 {
		return m.interval
	}
	return 5 * time.Second
}

func (m *testHealthCheck) Critical() bool {
	return m.critical
}

// testError for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
