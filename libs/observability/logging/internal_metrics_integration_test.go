package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestOTelLogger_InternalMetrics(t *testing.T) {
	config := OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "debug",
		Insecure:       true,
	}

	logger, err := NewOTelLogger(config)
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Initially, all metrics should be zero
	metrics := logger.GetInternalMetrics()
	assert.Equal(t, int64(0), metrics["total_logs"])
	assert.Equal(t, int64(0), metrics["debug_logs"])
	assert.Equal(t, int64(0), metrics["info_logs"])
	assert.Equal(t, int64(0), metrics["warn_logs"])
	assert.Equal(t, int64(0), metrics["error_logs"])

	// Log at different levels
	ctx := context.Background()
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")

	// Check that logs are tracked
	metrics = logger.GetInternalMetrics()
	assert.Equal(t, int64(4), metrics["total_logs"])
	assert.Equal(t, int64(1), metrics["debug_logs"])
	assert.Equal(t, int64(1), metrics["info_logs"])
	assert.Equal(t, int64(1), metrics["warn_logs"])
	assert.Equal(t, int64(1), metrics["error_logs"])
}

func TestOTelLogger_TraceCorrelationMetrics(t *testing.T) {
	config := OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	}

	logger, err := NewOTelLogger(config)
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Create a tracer provider for testing
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")

	// Log without trace context
	logger.Info(context.Background(), "message without trace")

	metrics := logger.GetInternalMetrics()
	assert.Equal(t, int64(1), metrics["total_logs"])
	assert.Equal(t, int64(0), metrics["logs_with_trace"])

	// Log with trace context
	ctx, span := tracer.Start(context.Background(), "test-span")
	logger.Info(ctx, "message with trace")
	span.End()

	metrics = logger.GetInternalMetrics()
	assert.Equal(t, int64(2), metrics["total_logs"])
	assert.Equal(t, int64(1), metrics["logs_with_trace"])
}

func TestOTelLogger_ChildLoggerMetrics(t *testing.T) {
	config := OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	}

	logger, err := NewOTelLogger(config)
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Create child logger
	childLogger := logger.With("component", "test")

	// Log with both parent and child
	ctx := context.Background()
	logger.Info(ctx, "parent message")
	childLogger.Info(ctx, "child message")

	// Both should share the same internal metrics
	parentMetrics := logger.GetInternalMetrics()
	childMetrics := childLogger.(*OTelLogger).GetInternalMetrics()

	assert.Equal(t, int64(2), parentMetrics["total_logs"])
	assert.Equal(t, int64(2), childMetrics["total_logs"])
	assert.Equal(t, int64(2), parentMetrics["info_logs"])
	assert.Equal(t, int64(2), childMetrics["info_logs"])
}

func TestOTelLogger_InternalMetrics_LevelFiltering(t *testing.T) {
	config := OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "warn", // Only warn and error
		Insecure:       true,
	}

	logger, err := NewOTelLogger(config)
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	ctx := context.Background()

	// These should be filtered out
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")

	// These should be logged
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message")

	metrics := logger.GetInternalMetrics()
	assert.Equal(t, int64(2), metrics["total_logs"]) // Only warn and error
	assert.Equal(t, int64(0), metrics["debug_logs"])
	assert.Equal(t, int64(0), metrics["info_logs"])
	assert.Equal(t, int64(1), metrics["warn_logs"])
	assert.Equal(t, int64(1), metrics["error_logs"])
}

func TestOTelLogger_ConcurrentLogging(t *testing.T) {
	config := OTelConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Level:          "info",
		Insecure:       true,
	}

	logger, err := NewOTelLogger(config)
	require.NoError(t, err)
	defer logger.Shutdown(context.Background())

	// Perform concurrent logging
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			ctx := context.Background()
			for j := 0; j < 100; j++ {
				logger.Info(ctx, "concurrent message")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check that all logs were tracked
	metrics := logger.GetInternalMetrics()
	assert.Equal(t, int64(1000), metrics["total_logs"]) // 10 goroutines * 100 iterations
	assert.Equal(t, int64(1000), metrics["info_logs"])
}
