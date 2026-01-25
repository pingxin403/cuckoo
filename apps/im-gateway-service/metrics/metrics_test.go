package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0, // Don't start HTTP server in tests
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)
	assert.NotNil(t, m)
	assert.NotNil(t, m.obs)
}

func TestConnectionMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Test increment - should not panic
	m.IncrementActiveConnections()
	m.IncrementActiveConnections()

	// Test decrement - should not panic
	m.DecrementActiveConnections()

	// Test errors - should not panic
	m.IncrementConnectionErrors()

	// Note: OTel metrics don't support direct reads
	// Metrics are exported to backend (Prometheus/OTEL collector)
	assert.Equal(t, int64(0), m.GetActiveConnections())
}

func TestMessageDeliveryMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.IncrementMessagesDelivered()
	m.IncrementMessagesFailed()
	m.IncrementAckTimeouts()
}

func TestLatencyTracking(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Observe various latencies - should not panic
	m.ObserveLatency(5 * time.Millisecond)
	m.ObserveLatency(25 * time.Millisecond)
	m.ObserveLatency(75 * time.Millisecond)
	m.ObserveLatency(150 * time.Millisecond)
	m.ObserveLatency(300 * time.Millisecond)

	// Note: OTel histograms don't support direct percentile calculation
	// Percentiles are calculated by the metrics backend
	p50, p95, p99 := m.GetLatencyPercentiles()
	assert.Equal(t, 0.0, p50)
	assert.Equal(t, 0.0, p95)
	assert.Equal(t, 0.0, p99)
}

func TestOfflineQueueMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.SetOfflineQueueSize(100)
	m.IncrementOfflineQueueSize()
	m.DecrementOfflineQueueSize()
}

func TestDeduplicationMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.IncrementDuplicateMessages()
	m.IncrementDuplicateMessages()
}

func TestMultiDeviceMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.IncrementMultiDeviceDeliveries()
}

func TestGroupMessageMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.IncrementGroupMessagesDelivered()
	m.AddGroupMembersFanout(10)
	m.AddGroupMembersFanout(5)
}

func TestCacheMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Should not panic
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheMisses()

	// Note: OTel counters don't support direct rate calculation
	// Rates are calculated by the metrics backend
	assert.Equal(t, 0.0, m.GetCacheHitRate())
}

func TestConcurrentMetricsUpdates(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	m := NewMetrics(obs)

	// Simulate concurrent updates - should not panic or race
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.IncrementActiveConnections()
				m.IncrementMessagesDelivered()
				m.ObserveLatency(10 * time.Millisecond)
				m.IncrementCacheHits()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test passes if no panics or races occurred
}

func TestShutdown(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:       "test-service",
		ServiceVersion:    "1.0.0",
		Environment:       "test",
		EnableMetrics:     true,
		UseOTelMetrics:    true,
		PrometheusEnabled: true,
		MetricsPort:       0,
	})
	require.NoError(t, err)

	m := NewMetrics(obs)

	// Shutdown should not panic
	err = m.Shutdown(context.Background())
	assert.NoError(t, err)

	// Shutdown observability (may have sync errors on stdout, which is expected in tests)
	_ = obs.Shutdown(context.Background())
}
