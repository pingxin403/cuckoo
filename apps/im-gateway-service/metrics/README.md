# Gateway Service Metrics

This package provides Prometheus-compatible metrics for the IM Gateway Service.

## Available Metrics

### Connection Metrics

- `im_gateway_active_connections` (gauge): Current number of active WebSocket connections
- `im_gateway_total_connections_total` (counter): Total number of connections established since startup
- `im_gateway_connection_errors_total` (counter): Total number of connection errors

### Message Delivery Metrics

- `im_gateway_messages_delivered_total` (counter): Total number of messages successfully delivered
- `im_gateway_messages_failed_total` (counter): Total number of message delivery failures
- `im_gateway_ack_timeouts_total` (counter): Total number of ACK timeouts
- `im_gateway_ack_timeout_rate_percent` (gauge): Percentage of ACK timeouts (calculated)

### Latency Metrics

- `im_gateway_message_delivery_latency_seconds` (histogram): Message delivery latency distribution
  - Buckets: 10ms, 50ms, 100ms, 200ms, 500ms, 1s, 2s, 5s, +Inf
  - Includes `_sum` and `_count` for average calculation
  - Use for P50, P95, P99 percentile calculations

### Offline Queue Metrics

- `im_gateway_offline_queue_size` (gauge): Current size of offline message queue

### Deduplication Metrics

- `im_gateway_duplicate_messages_total` (counter): Total number of duplicate messages detected
- `im_gateway_message_duplication_rate_percent` (gauge): Percentage of duplicate messages (calculated)

### Multi-Device Metrics

- `im_gateway_multi_device_deliveries_total` (counter): Total number of multi-device message deliveries

### Group Message Metrics

- `im_gateway_group_messages_delivered_total` (counter): Total number of group messages delivered
- `im_gateway_group_members_fanout_total` (counter): Total number of group member fanouts

### Cache Metrics

- `im_gateway_cache_hits_total` (counter): Total number of cache hits
- `im_gateway_cache_misses_total` (counter): Total number of cache misses
- `im_gateway_cache_hit_rate_percent` (gauge): Cache hit rate percentage (calculated)

## Usage

### Initialization

```go
import "github.com/pingxin403/cuckoo/apps/im-gateway-service/metrics"

// Create metrics collector
m := metrics.NewMetrics()
```

### Recording Metrics

```go
// Connection metrics
m.IncrementActiveConnections()
m.DecrementActiveConnections()
m.IncrementConnectionErrors()

// Message delivery metrics
m.IncrementMessagesDelivered()
m.IncrementMessagesFailed()
m.IncrementAckTimeouts()

// Latency tracking
startTime := time.Now()
// ... deliver message ...
m.ObserveLatency(time.Since(startTime))

// Offline queue
m.SetOfflineQueueSize(100)
m.IncrementOfflineQueueSize()
m.DecrementOfflineQueueSize()

// Deduplication
m.IncrementDuplicateMessages()

// Multi-device
m.IncrementMultiDeviceDeliveries()

// Group messages
m.IncrementGroupMessagesDelivered()
m.AddGroupMembersFanout(10) // 10 members received the message

// Cache
m.IncrementCacheHits()
m.IncrementCacheMisses()
```

### Exposing Metrics

```go
// Add metrics endpoint to HTTP server
http.HandleFunc("/metrics", m.Handler())
```

### Accessing Metrics

```bash
# Get metrics in Prometheus format
curl http://localhost:9093/metrics
```

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'im-gateway'
    static_configs:
      - targets: ['localhost:9093']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Grafana Queries

### Active Connections

```promql
im_gateway_active_connections
```

### Message Delivery Rate (per second)

```promql
rate(im_gateway_messages_delivered_total[5m])
```

### P95 Latency

```promql
histogram_quantile(0.95, rate(im_gateway_message_delivery_latency_seconds_bucket[5m]))
```

### P99 Latency

```promql
histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m]))
```

### ACK Timeout Rate

```promql
im_gateway_ack_timeout_rate_percent
```

### Cache Hit Rate

```promql
im_gateway_cache_hit_rate_percent
```

### Message Duplication Rate

```promql
im_gateway_message_duplication_rate_percent
```

## Alerting Rules

### High ACK Timeout Rate

```yaml
- alert: HighAckTimeoutRate
  expr: im_gateway_ack_timeout_rate_percent > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High ACK timeout rate detected"
    description: "ACK timeout rate is {{ $value }}% (threshold: 5%)"
```

### High P99 Latency

```yaml
- alert: HighP99Latency
  expr: histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) > 0.5
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High P99 latency detected"
    description: "P99 latency is {{ $value }}s (threshold: 500ms)"
```

### High Message Duplication Rate

```yaml
- alert: HighMessageDuplicationRate
  expr: im_gateway_message_duplication_rate_percent > 0.01
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High message duplication rate detected"
    description: "Message duplication rate is {{ $value }}% (threshold: 0.01%)"
```

## Performance Considerations

- All metrics use atomic operations for thread-safe updates
- Histogram buckets are pre-allocated to minimize memory allocations
- The `/metrics` endpoint is read-only and does not modify state
- Metrics are stored in memory and reset on service restart

## Testing

Run the test suite:

```bash
make test APP=im-gateway
```

Or test the metrics package specifically:

```bash
cd apps/im-gateway-service
go test -v ./metrics/...
```

## Integration with Gateway Service

The metrics collector is integrated into the gateway service and automatically tracks:

- WebSocket connection lifecycle
- Message delivery success/failure
- Latency for each message delivery
- Cache hit/miss rates
- Group message fanout
- Multi-device deliveries

No manual instrumentation is required for these metrics.
