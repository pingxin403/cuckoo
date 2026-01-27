package observability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: observability-otel-enhancement, Property 31: Thread Safety
// Validates: Requirements 11.1-11.7
//
// Property: For any observability operation (metrics recording, logging, span creation),
// concurrent calls from multiple goroutines SHALL execute safely without data races.
//
// This test validates that the observability library is thread-safe by running
// concurrent operations from multiple goroutines. The race detector (-race flag)
// will catch any data race issues.
func TestProperty_ThreadSafety(t *testing.T) {
	// Skip if short mode (property tests can be slow)
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(5, 15).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(10, 30).Draw(t, "operationsPerGoroutine")
		metricName := rapid.StringMatching(`[a-z_]+`).Draw(t, "metricName")
		logMessage := rapid.String().Draw(t, "logMessage")
		spanName := rapid.StringMatching(`[a-z_]+`).Draw(t, "spanName")

		// Create observability instance with OTel implementations
		// Note: OTLP endpoints will fail to connect, but that's OK for thread safety testing
		config := Config{
			ServiceName:         "test-service",
			ServiceVersion:      "1.0.0",
			Environment:         "test",
			EnableMetrics:       true,
			UseOTelMetrics:      true,
			OTLPMetricsEndpoint: "localhost:4317",
			PrometheusEnabled:   false,
			EnableTracing:       true,
			TracingEndpoint:     "localhost:4317",
			UseOTelLogs:         true,
			OTLPLogsEndpoint:    "localhost:4317",
			LogLevel:            "info",
			OTLPInsecure:        true,
		}

		obs, err := New(config)
		require.NoError(t, err)

		// Use a short shutdown timeout to avoid hanging on export failures
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		// Run concurrent operations
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 3) // 3 types of operations per goroutine

		// Concurrent metrics operations (Requirement 11.1, 11.4)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Test concurrent calls to Metrics()
					metrics := obs.Metrics()
					assert.NotNil(t, metrics)

					// Test concurrent metric recording operations
					metrics.IncrementCounter(metricName, nil)
					metrics.AddCounter(metricName, float64(j), nil)
					metrics.SetGauge(metricName+"_gauge", float64(j), nil)
					metrics.RecordHistogram(metricName+"_hist", float64(j), nil)
					metrics.IncrementGauge(metricName+"_gauge2", nil)
					metrics.DecrementGauge(metricName+"_gauge3", nil)
				}
			}()
		}

		// Concurrent logging operations (Requirement 11.2, 11.5)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Test concurrent calls to Logger()
					logger := obs.Logger()
					assert.NotNil(t, logger)

					// Test concurrent logging operations
					logger.Debug(ctx, logMessage, "id", id, "iteration", j)
					logger.Info(ctx, logMessage, "id", id, "iteration", j)
					logger.Warn(ctx, logMessage, "id", id, "iteration", j)
					logger.Error(ctx, logMessage, "id", id, "iteration", j)

					// Test concurrent child logger creation
					childLogger := logger.With("component", "test", "id", id)
					childLogger.Info(ctx, logMessage)
				}
			}(i)
		}

		// Concurrent tracing operations (Requirement 11.3, 11.6)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Test concurrent calls to Tracer()
					tracer := obs.Tracer()
					assert.NotNil(t, tracer)

					// Test concurrent span creation operations
					spanCtx, span := tracer.StartSpan(ctx, spanName)
					assert.NotNil(t, spanCtx)
					assert.NotNil(t, span)

					// Test concurrent span operations
					span.SetAttribute("id", id)
					span.SetAttribute("iteration", j)
					span.SetAttributes(map[string]interface{}{
						"goroutine": id,
						"count":     j,
					})

					span.End()
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// If we reach here without data races or panics, the test passes
		// The race detector will catch any data race issues
	})
}

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Metrics Focus)
// Validates: Requirements 11.1, 11.4
//
// Property: Concurrent metric recording operations SHALL execute safely without data races.
func TestProperty_ThreadSafety_Metrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(10, 30).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(50, 100).Draw(t, "operationsPerGoroutine")

		// Create observability instance
		config := Config{
			ServiceName:         "test-service",
			ServiceVersion:      "1.0.0",
			Environment:         "test",
			EnableMetrics:       true,
			UseOTelMetrics:      true,
			OTLPMetricsEndpoint: "localhost:4317",
			PrometheusEnabled:   false,
			OTLPInsecure:        true,
		}

		obs, err := New(config)
		require.NoError(t, err)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		metrics := obs.Metrics()

		// Run concurrent metric operations
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Mix different metric operations
					switch j % 6 {
					case 0:
						metrics.IncrementCounter("concurrent_counter", map[string]string{"id": string(rune(id))})
					case 1:
						metrics.AddCounter("concurrent_counter", float64(j), nil)
					case 2:
						metrics.SetGauge("concurrent_gauge", float64(j), map[string]string{"id": string(rune(id))})
					case 3:
						metrics.IncrementGauge("concurrent_gauge_inc", nil)
					case 4:
						metrics.DecrementGauge("concurrent_gauge_dec", nil)
					case 5:
						metrics.RecordHistogram("concurrent_histogram", float64(j), nil)
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Logging Focus)
// Validates: Requirements 11.2, 11.5
//
// Property: Concurrent logging operations SHALL execute safely without data races.
func TestProperty_ThreadSafety_Logging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(10, 30).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(50, 100).Draw(t, "operationsPerGoroutine")

		// Create observability instance
		config := Config{
			ServiceName:      "test-service",
			ServiceVersion:   "1.0.0",
			Environment:      "test",
			UseOTelLogs:      true,
			OTLPLogsEndpoint: "localhost:4317",
			LogLevel:         "debug",
			OTLPInsecure:     true,
		}

		obs, err := New(config)
		require.NoError(t, err)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		logger := obs.Logger()

		// Run concurrent logging operations
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Mix different log levels
					switch j % 4 {
					case 0:
						logger.Debug(ctx, "debug message", "id", id, "iteration", j)
					case 1:
						logger.Info(ctx, "info message", "id", id, "iteration", j)
					case 2:
						logger.Warn(ctx, "warn message", "id", id, "iteration", j)
					case 3:
						logger.Error(ctx, "error message", "id", id, "iteration", j)
					}

					// Test child logger creation concurrently
					if j%10 == 0 {
						childLogger := logger.With("component", "test", "id", id)
						childLogger.Info(ctx, "child logger message")
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Tracing Focus)
// Validates: Requirements 11.3, 11.6
//
// Property: Concurrent span creation operations SHALL execute safely without data races.
func TestProperty_ThreadSafety_Tracing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(10, 30).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(50, 100).Draw(t, "operationsPerGoroutine")

		// Create observability instance
		config := Config{
			ServiceName:       "test-service",
			ServiceVersion:    "1.0.0",
			Environment:       "test",
			EnableTracing:     true,
			TracingEndpoint:   "localhost:4317",
			TracingSampleRate: 1.0,
		}

		obs, err := New(config)
		require.NoError(t, err)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		tracer := obs.Tracer()

		// Run concurrent tracing operations
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				for j := 0; j < operationsPerGoroutine; j++ {
					// Create spans concurrently
					spanCtx, span := tracer.StartSpan(ctx, "concurrent_span")

					// Perform concurrent span operations
					span.SetAttribute("id", id)
					span.SetAttribute("iteration", j)
					span.SetAttributes(map[string]interface{}{
						"goroutine": id,
						"count":     j,
						"timestamp": time.Now().Unix(),
					})

					// Nested spans
					if j%5 == 0 {
						_, childSpan := tracer.StartSpan(spanCtx, "child_span")
						childSpan.SetAttribute("parent_id", id)
						childSpan.End()
					}

					span.End()
				}
			}(i)
		}

		wg.Wait()
	})
}

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Mixed Operations)
// Validates: Requirements 11.1-11.7
//
// Property: Concurrent mixed operations (metrics, logging, tracing) SHALL execute safely.
func TestProperty_ThreadSafety_MixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(10, 20).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(20, 50).Draw(t, "operationsPerGoroutine")

		// Create observability instance with all features enabled
		config := Config{
			ServiceName:         "test-service",
			ServiceVersion:      "1.0.0",
			Environment:         "test",
			EnableMetrics:       true,
			UseOTelMetrics:      true,
			OTLPMetricsEndpoint: "localhost:4317",
			PrometheusEnabled:   false,
			EnableTracing:       true,
			TracingEndpoint:     "localhost:4317",
			TracingSampleRate:   1.0,
			UseOTelLogs:         true,
			OTLPLogsEndpoint:    "localhost:4317",
			LogLevel:            "info",
			OTLPInsecure:        true,
		}

		obs, err := New(config)
		require.NoError(t, err)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		// Run concurrent mixed operations
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()

				for j := 0; j < operationsPerGoroutine; j++ {
					// Mix all types of operations
					operation := j % 9

					switch operation {
					case 0, 1, 2:
						// Metrics operations
						metrics := obs.Metrics()
						metrics.IncrementCounter("mixed_counter", nil)
						metrics.SetGauge("mixed_gauge", float64(j), nil)
						metrics.RecordHistogram("mixed_histogram", float64(j), nil)

					case 3, 4, 5:
						// Logging operations
						logger := obs.Logger()
						logger.Info(ctx, "mixed operation", "id", id, "iteration", j)
						childLogger := logger.With("component", "mixed")
						childLogger.Debug(ctx, "child logger")

					case 6, 7, 8:
						// Tracing operations
						tracer := obs.Tracer()
						spanCtx, span := tracer.StartSpan(ctx, "mixed_span")
						span.SetAttribute("id", id)

						// Log within span context (trace-log correlation)
						logger := obs.Logger()
						logger.Info(spanCtx, "log with trace", "id", id)

						// Record metrics within span
						metrics := obs.Metrics()
						metrics.IncrementCounter("span_counter", nil)

						span.End()
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Shared State)
// Validates: Requirements 11.7
//
// Property: Shared state SHALL be protected with appropriate synchronization primitives.
func TestProperty_ThreadSafety_SharedState(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numGoroutines := rapid.IntRange(20, 50).Draw(t, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(100, 200).Draw(t, "operationsPerGoroutine")

		// Create observability instance
		config := Config{
			ServiceName:         "test-service",
			ServiceVersion:      "1.0.0",
			Environment:         "test",
			EnableMetrics:       true,
			UseOTelMetrics:      true,
			OTLPMetricsEndpoint: "localhost:4317",
			PrometheusEnabled:   false,
			OTLPInsecure:        true,
		}

		obs, err := New(config)
		require.NoError(t, err)

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}()

		metrics := obs.Metrics()

		// Test concurrent access to the same metric (shared state)
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					// All goroutines access the same metric
					metrics.IncrementCounter("shared_counter", nil)
					metrics.SetGauge("shared_gauge", float64(j), nil)
					metrics.IncrementGauge("shared_gauge_inc", nil)
					metrics.DecrementGauge("shared_gauge_dec", nil)
				}
			}()
		}

		wg.Wait()

		// Verify that the shared state is consistent
		// (The race detector will catch any synchronization issues)
	})
}
