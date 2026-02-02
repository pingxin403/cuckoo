//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestHealthEndpoints verifies that all health check endpoints are working
func TestHealthEndpoints(t *testing.T) {
	// Wait a bit for the service to start
	time.Sleep(2 * time.Second)

	baseURL := "http://localhost:8080"

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		checkBody      bool
	}{
		{
			name:           "Liveness endpoint",
			endpoint:       "/healthz",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "Readiness endpoint",
			endpoint:       "/readyz",
			expectedStatus: http.StatusOK,
			checkBody:      true,
		},
		{
			name:           "Health detail endpoint",
			endpoint:       "/health",
			expectedStatus: http.StatusOK,
			checkBody:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + tt.endpoint)
			if err != nil {
				t.Fatalf("Failed to call %s: %v", tt.endpoint, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkBody {
				// For simple endpoints, just verify we got a response
				if resp.ContentLength == 0 {
					t.Error("Expected non-empty response body")
				}
			}

			t.Logf("✓ %s returned status %d", tt.endpoint, resp.StatusCode)
		})
	}
}

// TestHealthDetailedResponse verifies the detailed health endpoint returns proper JSON
func TestHealthDetailedResponse(t *testing.T) {
	time.Sleep(2 * time.Second)

	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Failed to call /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify Content-Type is JSON
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse JSON response
	var healthStatus map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthStatus); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	// Verify required fields exist
	requiredFields := []string{"status", "service", "timestamp", "score"}
	for _, field := range requiredFields {
		if _, ok := healthStatus[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify status is one of the expected values
	status, ok := healthStatus["status"].(string)
	if !ok {
		t.Error("Status field is not a string")
	} else {
		validStatuses := map[string]bool{
			"healthy":  true,
			"degraded": true,
			"critical": true,
		}
		if !validStatuses[status] {
			t.Errorf("Invalid status value: %s", status)
		}
	}

	// Verify service name
	service, ok := healthStatus["service"].(string)
	if !ok {
		t.Error("Service field is not a string")
	} else if service != "todo-service" {
		t.Errorf("Expected service 'todo-service', got '%s'", service)
	}

	// Verify score is a number between 0 and 1
	score, ok := healthStatus["score"].(float64)
	if !ok {
		t.Error("Score field is not a number")
	} else if score < 0 || score > 1 {
		t.Errorf("Score should be between 0 and 1, got %f", score)
	}

	t.Logf("✓ Health endpoint returned valid JSON with status: %s, score: %.2f", status, score)
}

// TestHealthEndpointsUnderLoad verifies health endpoints work under concurrent load
func TestHealthEndpointsUnderLoad(t *testing.T) {
	time.Sleep(2 * time.Second)

	numRequests := 50
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- err
				return
			}

			results <- nil
		}()
	}

	// Collect results
	failures := 0
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			failures++
			t.Logf("Request failed: %v", err)
		}
	}

	if failures > 0 {
		t.Errorf("%d out of %d requests failed", failures, numRequests)
	} else {
		t.Logf("✓ All %d concurrent health check requests succeeded", numRequests)
	}
}
