package metrics

import (
	"context"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// Metrics holds all OpenTelemetry metrics for the gateway service
type Metrics struct {
	obs observability.Observability
}

// NewMetrics creates a new Metrics instance using observability library
func NewMetrics(obs observability.Observability) *Metrics {
	return &Metrics{
		obs: obs,
	}
}

// Connection metrics

func (m *Metrics) IncrementActiveConnections() {
	m.obs.Metrics().IncrementGauge("im_gateway_active_connections", nil)
	m.obs.Metrics().IncrementCounter("im_gateway_total_connections_total", nil)
}

func (m *Metrics) DecrementActiveConnections() {
	m.obs.Metrics().DecrementGauge("im_gateway_active_connections", nil)
}

func (m *Metrics) IncrementConnectionErrors() {
	m.obs.Metrics().IncrementCounter("im_gateway_connection_errors_total", nil)
}

func (m *Metrics) GetActiveConnections() int64 {
	// Note: OTel gauges don't support direct reads, this is for backward compatibility
	// In production, query metrics from Prometheus/OTEL collector
	return 0
}

// Message delivery metrics

func (m *Metrics) IncrementMessagesDelivered() {
	m.obs.Metrics().IncrementCounter("im_gateway_messages_delivered_total", nil)
}

func (m *Metrics) IncrementMessagesFailed() {
	m.obs.Metrics().IncrementCounter("im_gateway_messages_failed_total", nil)
}

func (m *Metrics) IncrementAckTimeouts() {
	m.obs.Metrics().IncrementCounter("im_gateway_ack_timeouts_total", nil)
}

func (m *Metrics) IncrementAckPending() {
	m.obs.Metrics().IncrementCounter("im_gateway_ack_pending_total", nil)
}

func (m *Metrics) IncrementAckSuccess() {
	m.obs.Metrics().IncrementCounter("im_gateway_ack_success_total", nil)
}

func (m *Metrics) IncrementAckLate() {
	m.obs.Metrics().IncrementCounter("im_gateway_ack_late_total", nil)
}

func (m *Metrics) IncForwardSuccess(kind string) {
	m.obs.Metrics().IncrementCounter("im_gateway_cross_gateway_forward_total", map[string]string{
		"kind":   kind,
		"result": "success",
	})
}

func (m *Metrics) IncForwardFailure(kind, reason string) {
	m.obs.Metrics().IncrementCounter("im_gateway_cross_gateway_forward_total", map[string]string{
		"kind":   kind,
		"result": "failure",
		"reason": reason,
	})
}

func (m *Metrics) ObserveForwardLatency(kind string, duration time.Duration) {
	m.obs.Metrics().RecordDuration("im_gateway_cross_gateway_forward_latency_seconds", duration, map[string]string{
		"kind": kind,
	})
}

func (m *Metrics) SetKafkaConsumerLag(topic string, lag int64) {
	m.obs.Metrics().SetGauge("im_gateway_kafka_consumer_lag", float64(lag), map[string]string{
		"topic": topic,
	})
}

func (m *Metrics) IncrementKafkaConsumerErrors(topic string) {
	m.obs.Metrics().IncrementCounter("im_gateway_kafka_consumer_errors_total", map[string]string{
		"topic": topic,
	})
}

// Latency tracking

func (m *Metrics) ObserveLatency(duration time.Duration) {
	m.obs.Metrics().RecordDuration("im_gateway_message_delivery_latency_seconds", duration, nil)
}

// GetLatencyPercentiles calculates P50, P95, P99 from histogram buckets
// Note: With OTel, percentiles are calculated by the backend (Prometheus/Grafana)
func (m *Metrics) GetLatencyPercentiles() (p50, p95, p99 float64) {
	// OTel histograms don't support direct percentile calculation
	// Percentiles are calculated by the metrics backend (Prometheus/Grafana)
	return 0, 0, 0
}

// Offline queue metrics

func (m *Metrics) SetOfflineQueueSize(size int64) {
	m.obs.Metrics().SetGauge("im_gateway_offline_queue_size", float64(size), nil)
}

func (m *Metrics) IncrementOfflineQueueSize() {
	m.obs.Metrics().IncrementGauge("im_gateway_offline_queue_size", nil)
}

func (m *Metrics) DecrementOfflineQueueSize() {
	m.obs.Metrics().DecrementGauge("im_gateway_offline_queue_size", nil)
}

// Deduplication metrics

func (m *Metrics) IncrementDuplicateMessages() {
	m.obs.Metrics().IncrementCounter("im_gateway_duplicate_messages_total", nil)
}

// Multi-device metrics

func (m *Metrics) IncrementMultiDeviceDeliveries() {
	m.obs.Metrics().IncrementCounter("im_gateway_multi_device_deliveries_total", nil)
}

// Group message metrics

func (m *Metrics) IncrementGroupMessagesDelivered() {
	m.obs.Metrics().IncrementCounter("im_gateway_group_messages_delivered_total", nil)
}

func (m *Metrics) AddGroupMembersFanout(count int64) {
	m.obs.Metrics().AddCounter("im_gateway_group_members_fanout_total", float64(count), nil)
}

// Cache metrics

func (m *Metrics) IncrementCacheHits() {
	m.obs.Metrics().IncrementCounter("im_gateway_cache_hits_total", nil)
}

func (m *Metrics) IncrementCacheMisses() {
	m.obs.Metrics().IncrementCounter("im_gateway_cache_misses_total", nil)
}

// GetCacheHitRate returns the cache hit rate as a percentage
// Note: With OTel, rates are calculated by the backend (Prometheus/Grafana)
func (m *Metrics) GetCacheHitRate() float64 {
	// OTel counters don't support direct rate calculation
	// Rates are calculated by the metrics backend using PromQL
	return 0
}

// Shutdown gracefully shuts down the metrics collector
func (m *Metrics) Shutdown(ctx context.Context) error {
	// Observability shutdown is handled by the main application
	return nil
}
