package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOTelMetricsIntegration tests OTel Metrics integration
func TestOTelMetricsIntegration(t *testing.T) {
	config := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		OTLPEndpoint:      "localhost:4317",
		PrometheusEnabled: false,
		OTLPInsecure:      true,
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify metrics collector is available
	assert.NotNil(t, obs.Metrics())

	// Record some metrics
	obs.Metrics().IncrementCounter("test_counter", nil)
	obs.Metrics().SetGauge("test_gauge", 42.0, nil)
	obs.Metrics().RecordHistogram("test_histogram", 1.5, nil)

	// No assertions - just verify no panics
}

// TestOTelLogsIntegration tests OTel Logs integration
func TestOTelLogsIntegration(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		UseOTelLogs:    true,
		OTLPEndpoint:   "localhost:4317",
		LogLevel:       "info",
		OTLPInsecure:   true,
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify logger is available
	assert.NotNil(t, obs.Logger())

	// Log some messages
	ctx := context.Background()
	obs.Logger().Info(ctx, "test info message", "key", "value")
	obs.Logger().Debug(ctx, "test debug message")
	obs.Logger().Warn(ctx, "test warn message")
	obs.Logger().Error(ctx, "test error message")

	// No assertions - just verify no panics
}

// TestMixedMode tests mixed mode (OTel traces + Prometheus metrics)
func TestMixedMode(t *testing.T) {
	config := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    false, // Use Prometheus
		EnableTracing:     true,
		TracingEndpoint:   "localhost:4317",
		TracingSampleRate: 1.0,
		LogLevel:          "info",
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify all components are available
	assert.NotNil(t, obs.Metrics())
	assert.NotNil(t, obs.Tracer())
	assert.NotNil(t, obs.Logger())

	// Use all components
	obs.Metrics().IncrementCounter("test_counter", nil)
	ctx, span := obs.Tracer().StartSpan(context.Background(), "test-span")
	obs.Logger().Info(ctx, "test message in span")
	span.End()
}

// TestGracefulFallback tests graceful fallback to no-op on initialization failure
func TestGracefulFallback(t *testing.T) {
	// This test verifies that invalid OTLP endpoints don't crash the service
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		UseOTelLogs:    true,
		OTLPEndpoint:   "invalid-endpoint:9999",
		LogLevel:       "info",
		OTLPInsecure:   true,
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify logger is still available (fallback to structured logger)
	assert.NotNil(t, obs.Logger())

	// Should be able to log without errors
	obs.Logger().Info(context.Background(), "test message")
}

// TestShutdownCoordination tests that shutdown properly coordinates all components
func TestShutdownCoordination(t *testing.T) {
	config := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		EnableTracing:     true,
		UseOTelLogs:       true,
		OTLPEndpoint:      "localhost:4317",
		TracingSampleRate: 1.0,
		LogLevel:          "info",
		OTLPInsecure:      true,
	}

	obs, err := New(config)
	require.NoError(t, err)

	// Use all components
	obs.Metrics().IncrementCounter("test_counter", nil)
	ctx, span := obs.Tracer().StartSpan(context.Background(), "test-span")
	obs.Logger().Info(ctx, "test message")
	span.End()

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = obs.Shutdown(shutdownCtx)
	// Shutdown may return errors if OTLP collector is not available, but should not panic
	if err != nil {
		t.Logf("Shutdown returned error (expected if no collector): %v", err)
	}
}

// TestBackwardCompatibility tests that existing Prometheus mode still works
func TestBackwardCompatibility(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  true,
		UseOTelMetrics: false, // Use legacy Prometheus
		LogLevel:       "info",
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify metrics collector is available
	assert.NotNil(t, obs.Metrics())

	// Record metrics using legacy interface
	obs.Metrics().IncrementCounter("test_counter", nil)
	obs.Metrics().SetGauge("test_gauge", 42.0, nil)
	obs.Metrics().RecordHistogram("test_histogram", 1.5, nil)
}

// TestDualExport tests dual export (OTLP + Prometheus)
func TestDualExport(t *testing.T) {
	config := Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true, // Enable both OTLP and Prometheus
		OTLPEndpoint:      "localhost:4317",
		MetricsPort:       9103,
		OTLPInsecure:      true,
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Verify metrics collector is available
	assert.NotNil(t, obs.Metrics())

	// Record metrics
	obs.Metrics().IncrementCounter("test_counter", nil)
	obs.Metrics().SetGauge("test_gauge", 42.0, nil)

	// Metrics should be exported to both OTLP and Prometheus
	// (No direct way to verify without mock collector, but should not panic)
}
