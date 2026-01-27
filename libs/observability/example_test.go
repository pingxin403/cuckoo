package observability_test

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// Example demonstrates basic usage of the observability library
func Example() {
	// Initialize observability with logging disabled for example
	obs, err := observability.New(observability.Config{
		ServiceName:    "example-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		EnableMetrics:  false, // Disable metrics server for example
		EnableTracing:  false,
		LogLevel:       "error", // Only log errors
		LogFormat:      "json",
		LogOutput:      "stderr", // Send logs to stderr
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = obs.Shutdown(ctx)
	}()

	ctx := context.Background()

	// Use structured logging (won't output due to error level)
	obs.Logger().Info(ctx, "Service started", "port", 8080)

	// Record metrics
	obs.Metrics().IncrementCounter("requests_total", map[string]string{
		"method": "GET",
		"path":   "/api/users",
	})

	obs.Metrics().SetGauge("active_connections", 42, nil)

	obs.Metrics().RecordDuration("request_duration_seconds",
		150*time.Millisecond,
		map[string]string{"method": "GET"},
	)

	// Use logger with additional fields
	logger := obs.Logger().With("request_id", "abc123")
	logger.Info(ctx, "Processing request", "user_id", "user456")

	// Metrics are automatically exposed when EnableMetrics is true
	fmt.Println("Observability library initialized successfully")
	// Output: Observability library initialized successfully
}

// Example_configuration demonstrates configuration options
func Example_configuration() {
	// Create observability with custom configuration
	obs, err := observability.New(observability.Config{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		Environment:    "production",

		// Metrics configuration
		EnableMetrics:    true,
		MetricsPort:      9090,
		MetricsPath:      "/metrics",
		MetricsNamespace: "myapp",

		// Logging configuration
		LogLevel:  "info",
		LogFormat: "json",
		LogOutput: "stdout",

		// Tracing configuration (future)
		EnableTracing:     false,
		TracingEndpoint:   "",
		TracingSampleRate: 0.1,
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	// Suppress output for example
	_ = obs
	fmt.Println("Configuration example complete")
	// Output: Configuration example complete
}

// Example_metrics demonstrates metric recording
func Example_metrics() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "metrics-example",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	defer obs.Shutdown(context.Background())

	// Counter: always increasing
	obs.Metrics().IncrementCounter("requests_total", map[string]string{
		"method": "GET",
		"status": "200",
	})

	// Gauge: can go up or down
	obs.Metrics().SetGauge("active_connections", 42, nil)
	obs.Metrics().IncrementGauge("queue_size", nil)
	obs.Metrics().DecrementGauge("queue_size", nil)

	// Histogram: for latency/duration
	obs.Metrics().RecordDuration("request_duration_seconds",
		150*time.Millisecond,
		map[string]string{"endpoint": "/api/users"},
	)

	fmt.Println("Metrics recorded successfully")
	// Output: Metrics recorded successfully
}

// Example_logging demonstrates structured logging
func Example_logging() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "logging-example",
		EnableMetrics: false,
		LogLevel:      "error", // Only errors to avoid output
		LogFormat:     "json",
	})
	defer obs.Shutdown(context.Background())

	ctx := context.Background()

	// Different log levels (won't output due to error level)
	obs.Logger().Debug(ctx, "Debug message", "key", "value")
	obs.Logger().Info(ctx, "Info message", "key", "value")
	obs.Logger().Warn(ctx, "Warning message", "key", "value")

	// Child logger with additional fields
	logger := obs.Logger().With("request_id", "abc123", "user_id", "user456")
	logger.Info(ctx, "Processing request")

	fmt.Println("Logging example complete")
	// Output: Logging example complete
}
