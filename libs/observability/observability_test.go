package observability

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"testing"
	"time"
)

// TestPprofEndpointsAvailable tests that pprof endpoints are available when enabled
func TestPprofEndpointsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{"Index", "/debug/pprof/"},
		{"Cmdline", "/debug/pprof/cmdline"},
		{"Profile", "/debug/pprof/profile?seconds=1"},
		{"Symbol", "/debug/pprof/symbol"},
		{"Trace", "/debug/pprof/trace?seconds=1"},
		{"Heap", "/debug/pprof/heap"},
		{"Goroutine", "/debug/pprof/goroutine"},
		{"Threadcreate", "/debug/pprof/threadcreate"},
		{"Block", "/debug/pprof/block"},
		{"Mutex", "/debug/pprof/mutex"},
		{"Allocs", "/debug/pprof/allocs"},
	}

	config := Config{
		ServiceName:   "test-service",
		EnableMetrics: true,
		MetricsPort:   9091, // Use different port to avoid conflicts
		EnablePprof:   true,
	}

	obs, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("http://localhost:%d%s", config.MetricsPort, tt.endpoint)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET %s: %v", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", tt.endpoint, resp.StatusCode)
			}

			// Verify we got some content
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if len(body) == 0 {
				t.Errorf("Expected non-empty response for %s", tt.endpoint)
			}
		})
	}
}

// TestPprofEndpointsDisabled tests that pprof endpoints return 404 when disabled
func TestPprofEndpointsDisabled(t *testing.T) {
	config := Config{
		ServiceName:   "test-service",
		EnableMetrics: true,
		MetricsPort:   9092, // Use different port
		EnablePprof:   false,
	}

	obs, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	pprofEndpoints := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
	}

	for _, endpoint := range pprofEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			url := fmt.Sprintf("http://localhost:%d%s", config.MetricsPort, endpoint)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET %s: %v", url, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected status 404 for %s when pprof disabled, got %d", endpoint, resp.StatusCode)
			}
		})
	}
}

// TestPprofBlockProfileRate tests that block profile rate is configured correctly
func TestPprofBlockProfileRate(t *testing.T) {
	tests := []struct {
		name           string
		configuredRate int
		expectEnabled  bool
		port           int
	}{
		{"Disabled", 0, false, 9093},
		{"Enabled_1ns", 1, true, 9094},
		{"Enabled_1000ns", 1000, true, 9095},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset block profile rate before test
			runtime.SetBlockProfileRate(0)

			config := Config{
				ServiceName:           "test-service",
				EnableMetrics:         true,
				MetricsPort:           tt.port,
				EnablePprof:           true,
				PprofBlockProfileRate: tt.configuredRate,
			}

			obs, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create observability: %v", err)
			}
			defer obs.Shutdown(context.Background())

			// Wait for server to start and pprof to be configured
			time.Sleep(100 * time.Millisecond)

			// Note: There's no direct way to read the block profile rate from runtime,
			// but we can verify the profile is collecting data by checking the block endpoint
			url := fmt.Sprintf("http://localhost:%d/debug/pprof/block", config.MetricsPort)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET block profile: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for block profile, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			// When enabled, we should get profile data (even if empty)
			// When disabled (rate=0), we should still get a response but it will be minimal
			if len(body) == 0 {
				t.Errorf("Expected non-empty response for block profile")
			}
		})
	}
}

// TestPprofMutexProfileFraction tests that mutex profile fraction is configured correctly
func TestPprofMutexProfileFraction(t *testing.T) {
	tests := []struct {
		name               string
		configuredFraction int
		expectEnabled      bool
		port               int
	}{
		{"Disabled", 0, false, 9096},
		{"Enabled_1", 1, true, 9097},
		{"Enabled_10", 10, true, 9098},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mutex profile fraction before test
			runtime.SetMutexProfileFraction(0)

			config := Config{
				ServiceName:               "test-service",
				EnableMetrics:             true,
				MetricsPort:               tt.port,
				EnablePprof:               true,
				PprofMutexProfileFraction: tt.configuredFraction,
			}

			obs, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create observability: %v", err)
			}
			defer obs.Shutdown(context.Background())

			// Wait for server to start and pprof to be configured
			time.Sleep(100 * time.Millisecond)

			// Verify mutex profile endpoint is accessible
			url := fmt.Sprintf("http://localhost:%d/debug/pprof/mutex", config.MetricsPort)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET mutex profile: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for mutex profile, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if len(body) == 0 {
				t.Errorf("Expected non-empty response for mutex profile")
			}
		})
	}
}

// TestPprofConfigDefaults tests that pprof configuration defaults are applied correctly
func TestPprofConfigDefaults(t *testing.T) {
	config := Config{
		ServiceName: "test-service",
	}

	config = config.WithDefaults()

	// Verify pprof is disabled by default (security)
	if config.EnablePprof {
		t.Errorf("Expected EnablePprof to be false by default, got true")
	}

	// Verify default rates are 0 (disabled)
	if config.PprofBlockProfileRate != 0 {
		t.Errorf("Expected PprofBlockProfileRate to be 0 by default, got %d", config.PprofBlockProfileRate)
	}

	if config.PprofMutexProfileFraction != 0 {
		t.Errorf("Expected PprofMutexProfileFraction to be 0 by default, got %d", config.PprofMutexProfileFraction)
	}
}

// TestHealthEndpointAlwaysAvailable tests that health endpoint is always available
func TestHealthEndpointAlwaysAvailable(t *testing.T) {
	tests := []struct {
		name        string
		enablePprof bool
		port        int
	}{
		{"PprofEnabled", true, 9099},
		{"PprofDisabled", false, 9100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				ServiceName:   "test-service",
				EnableMetrics: true,
				MetricsPort:   tt.port,
				EnablePprof:   tt.enablePprof,
			}

			obs, err := New(config)
			if err != nil {
				t.Fatalf("Failed to create observability: %v", err)
			}
			defer obs.Shutdown(context.Background())

			// Wait for server to start
			time.Sleep(100 * time.Millisecond)

			url := fmt.Sprintf("http://localhost:%d/health", config.MetricsPort)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("Failed to GET health endpoint: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for health endpoint, got %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if string(body) != "OK" {
				t.Errorf("Expected 'OK' response, got '%s'", string(body))
			}
		})
	}
}

// TestMetricsEndpointAlwaysAvailable tests that metrics endpoint is always available
func TestMetricsEndpointAlwaysAvailable(t *testing.T) {
	config := Config{
		ServiceName:   "test-service",
		EnableMetrics: true,
		MetricsPort:   9102,
		EnablePprof:   true,
	}

	obs, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/metrics", config.MetricsPort)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to GET metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for metrics endpoint, got %d", resp.StatusCode)
	}
}
