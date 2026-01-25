package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all Prometheus metrics for the gateway service
type Metrics struct {
	// Connection metrics
	activeConnections atomic.Int64
	totalConnections  atomic.Int64
	connectionErrors  atomic.Int64

	// Message delivery metrics
	messagesDelivered atomic.Int64
	messagesFailed    atomic.Int64
	ackTimeouts       atomic.Int64

	// Latency tracking (using histogram buckets)
	latencyMu      sync.RWMutex
	latencyBuckets map[string]int64 // bucket label -> count
	latencySum     float64          // sum of all latencies in seconds
	latencyCount   int64            // total number of measurements

	// Offline queue metrics
	offlineQueueSize atomic.Int64

	// Deduplication metrics
	duplicateMessages atomic.Int64

	// Multi-device metrics
	multiDeviceDeliveries atomic.Int64

	// Group message metrics
	groupMessagesDelivered atomic.Int64
	groupMembersFanout     atomic.Int64

	// Cache metrics
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		latencyBuckets: make(map[string]int64),
	}
	// Initialize histogram buckets (in milliseconds)
	// Buckets: 10ms, 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s, +Inf
	m.latencyBuckets["10"] = 0
	m.latencyBuckets["50"] = 0
	m.latencyBuckets["100"] = 0
	m.latencyBuckets["200"] = 0
	m.latencyBuckets["500"] = 0
	m.latencyBuckets["1000"] = 0
	m.latencyBuckets["2000"] = 0
	m.latencyBuckets["5000"] = 0
	m.latencyBuckets["+Inf"] = 0
	return m
}

// Connection metrics

func (m *Metrics) IncrementActiveConnections() {
	m.activeConnections.Add(1)
	m.totalConnections.Add(1)
}

func (m *Metrics) DecrementActiveConnections() {
	m.activeConnections.Add(-1)
}

func (m *Metrics) IncrementConnectionErrors() {
	m.connectionErrors.Add(1)
}

func (m *Metrics) GetActiveConnections() int64 {
	return m.activeConnections.Load()
}

// Message delivery metrics

func (m *Metrics) IncrementMessagesDelivered() {
	m.messagesDelivered.Add(1)
}

func (m *Metrics) IncrementMessagesFailed() {
	m.messagesFailed.Add(1)
}

func (m *Metrics) IncrementAckTimeouts() {
	m.ackTimeouts.Add(1)
}

// Latency tracking

func (m *Metrics) ObserveLatency(duration time.Duration) {
	latencyMs := float64(duration.Milliseconds())
	latencySec := duration.Seconds()

	m.latencyMu.Lock()
	defer m.latencyMu.Unlock()

	// Update histogram buckets
	if latencyMs <= 10 {
		m.latencyBuckets["10"]++
	}
	if latencyMs <= 50 {
		m.latencyBuckets["50"]++
	}
	if latencyMs <= 100 {
		m.latencyBuckets["100"]++
	}
	if latencyMs <= 200 {
		m.latencyBuckets["200"]++
	}
	if latencyMs <= 500 {
		m.latencyBuckets["500"]++
	}
	if latencyMs <= 1000 {
		m.latencyBuckets["1000"]++
	}
	if latencyMs <= 2000 {
		m.latencyBuckets["2000"]++
	}
	if latencyMs <= 5000 {
		m.latencyBuckets["5000"]++
	}
	m.latencyBuckets["+Inf"]++

	// Update sum and count for average calculation
	m.latencySum += latencySec
	m.latencyCount++
}

// GetLatencyPercentiles calculates P50, P95, P99 from histogram buckets
func (m *Metrics) GetLatencyPercentiles() (p50, p95, p99 float64) {
	m.latencyMu.RLock()
	defer m.latencyMu.RUnlock()

	if m.latencyCount == 0 {
		return 0, 0, 0
	}

	// Simple approximation using histogram buckets
	// For production, use a proper histogram library like prometheus/client_golang
	total := m.latencyCount
	p50Target := int64(float64(total) * 0.50)
	p95Target := int64(float64(total) * 0.95)
	p99Target := int64(float64(total) * 0.99)

	buckets := []struct {
		le    float64
		count int64
	}{
		{10, m.latencyBuckets["10"]},
		{50, m.latencyBuckets["50"]},
		{100, m.latencyBuckets["100"]},
		{200, m.latencyBuckets["200"]},
		{500, m.latencyBuckets["500"]},
		{1000, m.latencyBuckets["1000"]},
		{2000, m.latencyBuckets["2000"]},
		{5000, m.latencyBuckets["5000"]},
	}

	for _, b := range buckets {
		if b.count >= p50Target && p50 == 0 {
			p50 = b.le
		}
		if b.count >= p95Target && p95 == 0 {
			p95 = b.le
		}
		if b.count >= p99Target && p99 == 0 {
			p99 = b.le
		}
	}

	// If not found in buckets, use +Inf
	if p50 == 0 {
		p50 = 5000
	}
	if p95 == 0 {
		p95 = 5000
	}
	if p99 == 0 {
		p99 = 5000
	}

	return p50, p95, p99
}

// Offline queue metrics

func (m *Metrics) SetOfflineQueueSize(size int64) {
	m.offlineQueueSize.Store(size)
}

func (m *Metrics) IncrementOfflineQueueSize() {
	m.offlineQueueSize.Add(1)
}

func (m *Metrics) DecrementOfflineQueueSize() {
	m.offlineQueueSize.Add(-1)
}

// Deduplication metrics

func (m *Metrics) IncrementDuplicateMessages() {
	m.duplicateMessages.Add(1)
}

// Multi-device metrics

func (m *Metrics) IncrementMultiDeviceDeliveries() {
	m.multiDeviceDeliveries.Add(1)
}

// Group message metrics

func (m *Metrics) IncrementGroupMessagesDelivered() {
	m.groupMessagesDelivered.Add(1)
}

func (m *Metrics) AddGroupMembersFanout(count int64) {
	m.groupMembersFanout.Add(count)
}

// Cache metrics

func (m *Metrics) IncrementCacheHits() {
	m.cacheHits.Add(1)
}

func (m *Metrics) IncrementCacheMisses() {
	m.cacheMisses.Add(1)
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (m *Metrics) GetCacheHitRate() float64 {
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// Handler returns an HTTP handler that exposes metrics in Prometheus format
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Connection metrics
		fmt.Fprintf(w, "# HELP im_gateway_active_connections Current number of active WebSocket connections\n")
		fmt.Fprintf(w, "# TYPE im_gateway_active_connections gauge\n")
		fmt.Fprintf(w, "im_gateway_active_connections %d\n", m.activeConnections.Load())

		fmt.Fprintf(w, "# HELP im_gateway_total_connections_total Total number of connections established\n")
		fmt.Fprintf(w, "# TYPE im_gateway_total_connections_total counter\n")
		fmt.Fprintf(w, "im_gateway_total_connections_total %d\n", m.totalConnections.Load())

		fmt.Fprintf(w, "# HELP im_gateway_connection_errors_total Total number of connection errors\n")
		fmt.Fprintf(w, "# TYPE im_gateway_connection_errors_total counter\n")
		fmt.Fprintf(w, "im_gateway_connection_errors_total %d\n", m.connectionErrors.Load())

		// Message delivery metrics
		fmt.Fprintf(w, "# HELP im_gateway_messages_delivered_total Total number of messages successfully delivered\n")
		fmt.Fprintf(w, "# TYPE im_gateway_messages_delivered_total counter\n")
		fmt.Fprintf(w, "im_gateway_messages_delivered_total %d\n", m.messagesDelivered.Load())

		fmt.Fprintf(w, "# HELP im_gateway_messages_failed_total Total number of message delivery failures\n")
		fmt.Fprintf(w, "# TYPE im_gateway_messages_failed_total counter\n")
		fmt.Fprintf(w, "im_gateway_messages_failed_total %d\n", m.messagesFailed.Load())

		fmt.Fprintf(w, "# HELP im_gateway_ack_timeouts_total Total number of ACK timeouts\n")
		fmt.Fprintf(w, "# TYPE im_gateway_ack_timeouts_total counter\n")
		fmt.Fprintf(w, "im_gateway_ack_timeouts_total %d\n", m.ackTimeouts.Load())

		// Latency histogram
		m.latencyMu.RLock()
		fmt.Fprintf(w, "# HELP im_gateway_message_delivery_latency_seconds Message delivery latency histogram\n")
		fmt.Fprintf(w, "# TYPE im_gateway_message_delivery_latency_seconds histogram\n")
		for _, le := range []string{"10", "50", "100", "200", "500", "1000", "2000", "5000", "+Inf"} {
			leValue := le
			if le != "+Inf" {
				leValue = fmt.Sprintf("0.%s", le) // Convert ms to seconds
			}
			fmt.Fprintf(w, "im_gateway_message_delivery_latency_seconds_bucket{le=\"%s\"} %d\n",
				leValue, m.latencyBuckets[le])
		}
		fmt.Fprintf(w, "im_gateway_message_delivery_latency_seconds_sum %.6f\n", m.latencySum)
		fmt.Fprintf(w, "im_gateway_message_delivery_latency_seconds_count %d\n", m.latencyCount)
		m.latencyMu.RUnlock()

		// Offline queue metrics
		fmt.Fprintf(w, "# HELP im_gateway_offline_queue_size Current size of offline message queue\n")
		fmt.Fprintf(w, "# TYPE im_gateway_offline_queue_size gauge\n")
		fmt.Fprintf(w, "im_gateway_offline_queue_size %d\n", m.offlineQueueSize.Load())

		// Deduplication metrics
		fmt.Fprintf(w, "# HELP im_gateway_duplicate_messages_total Total number of duplicate messages detected\n")
		fmt.Fprintf(w, "# TYPE im_gateway_duplicate_messages_total counter\n")
		fmt.Fprintf(w, "im_gateway_duplicate_messages_total %d\n", m.duplicateMessages.Load())

		// Calculate duplication rate
		totalMessages := m.messagesDelivered.Load() + m.duplicateMessages.Load()
		dupRate := 0.0
		if totalMessages > 0 {
			dupRate = float64(m.duplicateMessages.Load()) / float64(totalMessages) * 100
		}
		fmt.Fprintf(w, "# HELP im_gateway_message_duplication_rate_percent Percentage of duplicate messages\n")
		fmt.Fprintf(w, "# TYPE im_gateway_message_duplication_rate_percent gauge\n")
		fmt.Fprintf(w, "im_gateway_message_duplication_rate_percent %.2f\n", dupRate)

		// Multi-device metrics
		fmt.Fprintf(w, "# HELP im_gateway_multi_device_deliveries_total Total number of multi-device message deliveries\n")
		fmt.Fprintf(w, "# TYPE im_gateway_multi_device_deliveries_total counter\n")
		fmt.Fprintf(w, "im_gateway_multi_device_deliveries_total %d\n", m.multiDeviceDeliveries.Load())

		// Group message metrics
		fmt.Fprintf(w, "# HELP im_gateway_group_messages_delivered_total Total number of group messages delivered\n")
		fmt.Fprintf(w, "# TYPE im_gateway_group_messages_delivered_total counter\n")
		fmt.Fprintf(w, "im_gateway_group_messages_delivered_total %d\n", m.groupMessagesDelivered.Load())

		fmt.Fprintf(w, "# HELP im_gateway_group_members_fanout_total Total number of group member fanouts\n")
		fmt.Fprintf(w, "# TYPE im_gateway_group_members_fanout_total counter\n")
		fmt.Fprintf(w, "im_gateway_group_members_fanout_total %d\n", m.groupMembersFanout.Load())

		// Cache metrics
		fmt.Fprintf(w, "# HELP im_gateway_cache_hits_total Total number of cache hits\n")
		fmt.Fprintf(w, "# TYPE im_gateway_cache_hits_total counter\n")
		fmt.Fprintf(w, "im_gateway_cache_hits_total %d\n", m.cacheHits.Load())

		fmt.Fprintf(w, "# HELP im_gateway_cache_misses_total Total number of cache misses\n")
		fmt.Fprintf(w, "# TYPE im_gateway_cache_misses_total counter\n")
		fmt.Fprintf(w, "im_gateway_cache_misses_total %d\n", m.cacheMisses.Load())

		fmt.Fprintf(w, "# HELP im_gateway_cache_hit_rate_percent Cache hit rate percentage\n")
		fmt.Fprintf(w, "# TYPE im_gateway_cache_hit_rate_percent gauge\n")
		fmt.Fprintf(w, "im_gateway_cache_hit_rate_percent %.2f\n", m.GetCacheHitRate())

		// ACK timeout rate
		totalDeliveries := m.messagesDelivered.Load() + m.messagesFailed.Load()
		ackTimeoutRate := 0.0
		if totalDeliveries > 0 {
			ackTimeoutRate = float64(m.ackTimeouts.Load()) / float64(totalDeliveries) * 100
		}
		fmt.Fprintf(w, "# HELP im_gateway_ack_timeout_rate_percent Percentage of ACK timeouts\n")
		fmt.Fprintf(w, "# TYPE im_gateway_ack_timeout_rate_percent gauge\n")
		fmt.Fprintf(w, "im_gateway_ack_timeout_rate_percent %.2f\n", ackTimeoutRate)
	}
}
