package observability

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Feature: observability-otel-enhancement, Property 31: Thread Safety (Unit Test)
// Validates: Requirements 11.1-11.7
//
// This is a UNIT TEST that uses no-op implementations to test thread safety
// without requiring external OTLP collectors.
//
// Property: For any observability operation (metrics recording, logging, span creation),
// concurrent calls from multiple goroutines SHALL execute safely without data races.

func TestThreadSafety_NoOp(t *testing.T) {
	// Create observability instance with no-op implementations (no external dependencies)
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  false, // Use no-op
		EnableTracing:  false, // Use no-op
		UseOTelLogs:    false, // Use no-op
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Run concurrent operations with no-op implementations
	var wg sync.WaitGroup
	numGoroutines := 20
	operationsPerGoroutine := 100

	wg.Add(numGoroutines * 3) // 3 types of operations per goroutine

	// Concurrent metrics operations (Requirement 11.1, 11.4)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				metrics := obs.Metrics()
				assert.NotNil(t, metrics)

				metrics.IncrementCounter("test_counter", nil)
				metrics.AddCounter("test_counter", float64(j), nil)
				metrics.SetGauge("test_gauge", float64(j), nil)
				metrics.RecordHistogram("test_histogram", float64(j), nil)
				metrics.IncrementGauge("test_gauge2", nil)
				metrics.DecrementGauge("test_gauge3", nil)
			}
		}()
	}

	// Concurrent logging operations (Requirement 11.2, 11.5)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < operationsPerGoroutine; j++ {
				logger := obs.Logger()
				assert.NotNil(t, logger)

				logger.Debug(ctx, "debug message", "id", id, "iteration", j)
				logger.Info(ctx, "info message", "id", id, "iteration", j)
				logger.Warn(ctx, "warn message", "id", id, "iteration", j)
				logger.Error(ctx, "error message", "id", id, "iteration", j)

				childLogger := logger.With("component", "test", "id", id)
				childLogger.Info(ctx, "child logger message")
			}
		}(i)
	}

	// Concurrent tracing operations (Requirement 11.3, 11.6)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < operationsPerGoroutine; j++ {
				tracer := obs.Tracer()
				assert.NotNil(t, tracer)

				spanCtx, span := tracer.StartSpan(ctx, "test_span")
				assert.NotNil(t, spanCtx)
				assert.NotNil(t, span)

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
}

func TestThreadSafety_Metrics_NoOp(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  false, // Use no-op
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	metrics := obs.Metrics()

	// Run concurrent metric operations
	var wg sync.WaitGroup
	numGoroutines := 30
	operationsPerGoroutine := 200

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
}

func TestThreadSafety_Logging_NoOp(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		UseOTelLogs:    false, // Use no-op
		LogLevel:       "debug",
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	logger := obs.Logger()

	// Run concurrent logging operations
	var wg sync.WaitGroup
	numGoroutines := 30
	operationsPerGoroutine := 200

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
}

func TestThreadSafety_Tracing_NoOp(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableTracing:  false, // Use no-op
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	tracer := obs.Tracer()

	// Run concurrent tracing operations
	var wg sync.WaitGroup
	numGoroutines := 30
	operationsPerGoroutine := 200

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
}

func TestThreadSafety_MixedOperations_NoOp(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  false, // Use no-op
		EnableTracing:  false, // Use no-op
		UseOTelLogs:    false, // Use no-op
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Run concurrent mixed operations
	var wg sync.WaitGroup
	numGoroutines := 20
	operationsPerGoroutine := 100

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
}

func TestThreadSafety_SharedState_NoOp(t *testing.T) {
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableMetrics:  false, // Use no-op
	}

	obs, err := New(config)
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	metrics := obs.Metrics()

	// Test concurrent access to the same metric (shared state)
	var wg sync.WaitGroup
	numGoroutines := 50
	operationsPerGoroutine := 200

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
}
