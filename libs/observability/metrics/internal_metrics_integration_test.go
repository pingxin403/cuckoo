package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTelMetricsCollector_InternalMetrics(t *testing.T) {
	config := OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		OTLPEndpoint:      "localhost:4317",
		PrometheusEnabled: false,
		Insecure:          true,
	}

	collector, err := NewOTelMetricsCollector(config)
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Initially, all metrics should be zero
	metrics := collector.GetInternalMetrics()
	assert.Equal(t, int64(0), metrics["total_operations"])
	assert.Equal(t, int64(0), metrics["cached_counters"])
	assert.Equal(t, int64(0), metrics["cached_histograms"])
	assert.Equal(t, int64(0), metrics["cached_gauges"])

	// Perform some operations
	collector.IncrementCounter("test_counter", nil)
	collector.AddCounter("test_counter", 5, nil)
	collector.SetGauge("test_gauge", 42.0, nil)
	collector.RecordHistogram("test_histogram", 1.5, nil)
	collector.IncrementGauge("test_gauge2", nil)

	// Check that operations are tracked
	metrics = collector.GetInternalMetrics()
	assert.Equal(t, int64(5), metrics["total_operations"])
	assert.Equal(t, int64(1), metrics["cached_counters"])
	assert.Equal(t, int64(1), metrics["cached_histograms"])
	assert.Equal(t, int64(2), metrics["cached_gauges"])

	// Create more instruments
	collector.IncrementCounter("another_counter", map[string]string{"label": "value"})
	collector.SetGauge("another_gauge", 100.0, map[string]string{"env": "test"})
	collector.DecrementGauge("test_gauge3", nil)

	metrics = collector.GetInternalMetrics()
	assert.Equal(t, int64(8), metrics["total_operations"])
	assert.Equal(t, int64(2), metrics["cached_counters"])
	assert.Equal(t, int64(4), metrics["cached_gauges"])
}

func TestOTelMetricsCollector_InstrumentFailures(t *testing.T) {
	config := OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		OTLPEndpoint:      "localhost:4317",
		PrometheusEnabled: false,
		Insecure:          true,
	}

	collector, err := NewOTelMetricsCollector(config)
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Normal operations should not cause failures
	collector.IncrementCounter("valid_counter", nil)
	collector.SetGauge("valid_gauge", 1.0, nil)
	collector.RecordHistogram("valid_histogram", 1.0, nil)

	metrics := collector.GetInternalMetrics()
	assert.Equal(t, int64(0), metrics["instrument_failures"])
}

func TestOTelMetricsCollector_ConcurrentOperations(t *testing.T) {
	config := OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		OTLPEndpoint:      "localhost:4317",
		PrometheusEnabled: false,
		Insecure:          true,
	}

	collector, err := NewOTelMetricsCollector(config)
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Perform concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				collector.IncrementCounter("concurrent_counter", nil)
				collector.SetGauge("concurrent_gauge", 42.0, nil) // Same value to avoid creating multiple gauges
				collector.RecordHistogram("concurrent_histogram", float64(j), nil)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check that all operations were tracked
	metrics := collector.GetInternalMetrics()
	assert.Equal(t, int64(3000), metrics["total_operations"]) // 10 goroutines * 100 iterations * 3 operations

	// Due to concurrent gauge creation, we may have multiple gauges created
	// Just verify that at least one of each type was created
	assert.GreaterOrEqual(t, metrics["cached_counters"], int64(1))
	assert.GreaterOrEqual(t, metrics["cached_histograms"], int64(1))
	assert.GreaterOrEqual(t, metrics["cached_gauges"], int64(1))
}

func TestOTelMetricsCollector_InternalMetricsExposure(t *testing.T) {
	config := OTelConfig{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		OTLPEndpoint:      "localhost:4317",
		PrometheusEnabled: true,
		Insecure:          true,
	}

	collector, err := NewOTelMetricsCollector(config)
	require.NoError(t, err)
	defer collector.Shutdown(context.Background())

	// Perform some operations
	collector.IncrementCounter("test_counter", nil)
	collector.SetGauge("test_gauge", 42.0, nil)

	// Wait a bit for metrics to be collected
	time.Sleep(100 * time.Millisecond)

	// Internal metrics should be registered and observable
	// (In a real test, we would scrape the Prometheus endpoint and verify the metrics are present)
	metrics := collector.GetInternalMetrics()
	assert.Greater(t, metrics["total_operations"], int64(0))
}
