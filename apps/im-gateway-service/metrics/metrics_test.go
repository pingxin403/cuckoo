package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	assert.NotNil(t, m)
	assert.Equal(t, int64(0), m.GetActiveConnections())
	assert.NotNil(t, m.latencyBuckets)
	assert.Equal(t, 9, len(m.latencyBuckets)) // 8 buckets + +Inf
}

func TestConnectionMetrics(t *testing.T) {
	m := NewMetrics()

	// Test increment
	m.IncrementActiveConnections()
	assert.Equal(t, int64(1), m.GetActiveConnections())
	assert.Equal(t, int64(1), m.totalConnections.Load())

	m.IncrementActiveConnections()
	assert.Equal(t, int64(2), m.GetActiveConnections())
	assert.Equal(t, int64(2), m.totalConnections.Load())

	// Test decrement
	m.DecrementActiveConnections()
	assert.Equal(t, int64(1), m.GetActiveConnections())
	assert.Equal(t, int64(2), m.totalConnections.Load()) // Total should not decrease

	// Test errors
	m.IncrementConnectionErrors()
	assert.Equal(t, int64(1), m.connectionErrors.Load())
}

func TestMessageDeliveryMetrics(t *testing.T) {
	m := NewMetrics()

	m.IncrementMessagesDelivered()
	assert.Equal(t, int64(1), m.messagesDelivered.Load())

	m.IncrementMessagesFailed()
	assert.Equal(t, int64(1), m.messagesFailed.Load())

	m.IncrementAckTimeouts()
	assert.Equal(t, int64(1), m.ackTimeouts.Load())
}

func TestLatencyTracking(t *testing.T) {
	m := NewMetrics()

	// Observe various latencies
	m.ObserveLatency(5 * time.Millisecond)
	m.ObserveLatency(25 * time.Millisecond)
	m.ObserveLatency(75 * time.Millisecond)
	m.ObserveLatency(150 * time.Millisecond)
	m.ObserveLatency(300 * time.Millisecond)

	// Check histogram buckets
	assert.Equal(t, int64(1), m.latencyBuckets["10"])
	assert.Equal(t, int64(2), m.latencyBuckets["50"])
	assert.Equal(t, int64(3), m.latencyBuckets["100"])
	assert.Equal(t, int64(4), m.latencyBuckets["200"])
	assert.Equal(t, int64(5), m.latencyBuckets["500"])
	assert.Equal(t, int64(5), m.latencyBuckets["+Inf"])

	// Check sum and count
	assert.Equal(t, int64(5), m.latencyCount)
	assert.Greater(t, m.latencySum, 0.0)
}

func TestLatencyPercentiles(t *testing.T) {
	m := NewMetrics()

	// Add 100 samples with known distribution
	for i := 0; i < 50; i++ {
		m.ObserveLatency(10 * time.Millisecond) // 50% at 10ms
	}
	for i := 0; i < 45; i++ {
		m.ObserveLatency(100 * time.Millisecond) // 45% at 100ms
	}
	for i := 0; i < 5; i++ {
		m.ObserveLatency(500 * time.Millisecond) // 5% at 500ms
	}

	p50, p95, p99 := m.GetLatencyPercentiles()

	// P50 should be around 10-50ms
	assert.LessOrEqual(t, p50, 100.0)

	// P95 should be around 100-200ms
	assert.LessOrEqual(t, p95, 500.0)

	// P99 should be around 500ms
	assert.LessOrEqual(t, p99, 1000.0)
}

func TestOfflineQueueMetrics(t *testing.T) {
	m := NewMetrics()

	m.SetOfflineQueueSize(100)
	assert.Equal(t, int64(100), m.offlineQueueSize.Load())

	m.IncrementOfflineQueueSize()
	assert.Equal(t, int64(101), m.offlineQueueSize.Load())

	m.DecrementOfflineQueueSize()
	assert.Equal(t, int64(100), m.offlineQueueSize.Load())
}

func TestDeduplicationMetrics(t *testing.T) {
	m := NewMetrics()

	m.IncrementDuplicateMessages()
	assert.Equal(t, int64(1), m.duplicateMessages.Load())

	m.IncrementDuplicateMessages()
	assert.Equal(t, int64(2), m.duplicateMessages.Load())
}

func TestMultiDeviceMetrics(t *testing.T) {
	m := NewMetrics()

	m.IncrementMultiDeviceDeliveries()
	assert.Equal(t, int64(1), m.multiDeviceDeliveries.Load())
}

func TestGroupMessageMetrics(t *testing.T) {
	m := NewMetrics()

	m.IncrementGroupMessagesDelivered()
	assert.Equal(t, int64(1), m.groupMessagesDelivered.Load())

	m.AddGroupMembersFanout(10)
	assert.Equal(t, int64(10), m.groupMembersFanout.Load())

	m.AddGroupMembersFanout(5)
	assert.Equal(t, int64(15), m.groupMembersFanout.Load())
}

func TestCacheMetrics(t *testing.T) {
	m := NewMetrics()

	// Initially 0% hit rate
	assert.Equal(t, 0.0, m.GetCacheHitRate())

	// Add some hits and misses
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheHits()
	m.IncrementCacheMisses()

	// Hit rate should be 75%
	assert.Equal(t, 75.0, m.GetCacheHitRate())

	assert.Equal(t, int64(3), m.cacheHits.Load())
	assert.Equal(t, int64(1), m.cacheMisses.Load())
}

func TestPrometheusHandler(t *testing.T) {
	m := NewMetrics()

	// Set up some metrics
	m.IncrementActiveConnections()
	m.IncrementActiveConnections()
	m.IncrementMessagesDelivered()
	m.IncrementMessagesFailed()
	m.IncrementAckTimeouts()
	m.ObserveLatency(50 * time.Millisecond)
	m.SetOfflineQueueSize(10)
	m.IncrementDuplicateMessages()
	m.IncrementMultiDeviceDeliveries()
	m.IncrementGroupMessagesDelivered()
	m.AddGroupMembersFanout(5)
	m.IncrementCacheHits()
	m.IncrementCacheMisses()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler := m.Handler()
	handler(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; version=0.0.4", w.Header().Get("Content-Type"))

	body := w.Body.String()

	// Verify key metrics are present
	assert.Contains(t, body, "im_gateway_active_connections 2")
	assert.Contains(t, body, "im_gateway_total_connections_total 2")
	assert.Contains(t, body, "im_gateway_messages_delivered_total 1")
	assert.Contains(t, body, "im_gateway_messages_failed_total 1")
	assert.Contains(t, body, "im_gateway_ack_timeouts_total 1")
	assert.Contains(t, body, "im_gateway_offline_queue_size 10")
	assert.Contains(t, body, "im_gateway_duplicate_messages_total 1")
	assert.Contains(t, body, "im_gateway_multi_device_deliveries_total 1")
	assert.Contains(t, body, "im_gateway_group_messages_delivered_total 1")
	assert.Contains(t, body, "im_gateway_group_members_fanout_total 5")
	assert.Contains(t, body, "im_gateway_cache_hits_total 1")
	assert.Contains(t, body, "im_gateway_cache_misses_total 1")

	// Verify histogram is present
	assert.Contains(t, body, "im_gateway_message_delivery_latency_seconds_bucket")
	assert.Contains(t, body, "im_gateway_message_delivery_latency_seconds_sum")
	assert.Contains(t, body, "im_gateway_message_delivery_latency_seconds_count")

	// Verify calculated metrics
	assert.Contains(t, body, "im_gateway_message_duplication_rate_percent")
	assert.Contains(t, body, "im_gateway_cache_hit_rate_percent 50.00")
	assert.Contains(t, body, "im_gateway_ack_timeout_rate_percent 50.00")
}

func TestPrometheusHandlerFormat(t *testing.T) {
	m := NewMetrics()
	m.IncrementActiveConnections()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := m.Handler()
	handler(w, req)

	body := w.Body.String()
	lines := strings.Split(body, "\n")

	// Verify Prometheus format
	helpFound := false
	typeFound := false
	metricFound := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# HELP im_gateway_active_connections") {
			helpFound = true
		}
		if strings.HasPrefix(line, "# TYPE im_gateway_active_connections gauge") {
			typeFound = true
		}
		if strings.HasPrefix(line, "im_gateway_active_connections 1") {
			metricFound = true
		}
	}

	assert.True(t, helpFound, "HELP line should be present")
	assert.True(t, typeFound, "TYPE line should be present")
	assert.True(t, metricFound, "Metric line should be present")
}

func TestConcurrentMetricsUpdates(t *testing.T) {
	m := NewMetrics()

	// Simulate concurrent updates
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

	// Verify counts
	assert.Equal(t, int64(1000), m.GetActiveConnections())
	assert.Equal(t, int64(1000), m.messagesDelivered.Load())
	assert.Equal(t, int64(1000), m.latencyCount)
	assert.Equal(t, int64(1000), m.cacheHits.Load())
}
