# IM Gateway Service Metrics

## Overview

The IM Gateway Service uses **OpenTelemetry metrics** for observability, integrated through the `libs/observability` library. Metrics are exported to both Prometheus (for scraping) and OTLP endpoints (for push-based collection).

## Migration from Custom Prometheus

This service was migrated from a custom Prometheus implementation to OpenTelemetry metrics to:
- Standardize observability across all services
- Enable push-based metrics export via OTLP
- Leverage OpenTelemetry's rich ecosystem
- Support multiple backends (Prometheus, Grafana Cloud, etc.)

## Available Metrics

### Connection Metrics
- `im_gateway_active_connections` (gauge) - Current number of active WebSocket connections
- `im_gateway_total_connections_total` (counter) - Total number of connections established
- `im_gateway_connection_errors_total` (counter) - Total number of connection errors

### Message Delivery Metrics
- `im_gateway_messages_delivered_total` (counter) - Total messages successfully delivered
- `im_gateway_messages_failed_total` (counter) - Total message delivery failures
- `im_gateway_ack_timeouts_total` (counter) - Total ACK timeouts
- `im_gateway_ack_pending_total` (counter) - Total ACK pending registrations
- `im_gateway_ack_success_total` (counter) - Total ACK successful transitions
- `im_gateway_ack_late_total` (counter) - Total late ACK transitions (after timeout)
- `im_gateway_message_delivery_latency_seconds` (histogram) - Message delivery latency distribution

### Cross-Gateway Forwarding Metrics
- `im_gateway_cross_gateway_forward_total{kind,result,reason}` (counter) - Cross-gateway forwarding totals by kind/result/reason
- `im_gateway_cross_gateway_forward_latency_seconds{kind}` (histogram) - Cross-gateway forwarding latency distribution

### Kafka Consumer Reliability Metrics
- `im_gateway_kafka_consumer_lag{topic}` (gauge) - Kafka consumer lag by topic
- `im_gateway_kafka_consumer_errors_total{topic}` (counter) - Kafka consumer read/process errors by topic

### Offline Queue Metrics
- `im_gateway_offline_queue_size` (gauge) - Current size of offline message queue

### Deduplication Metrics
- `im_gateway_duplicate_messages_total` (counter) - Total duplicate messages detected

### Multi-Device Metrics
- `im_gateway_multi_device_deliveries_total` (counter) - Total multi-device message deliveries

### Group Message Metrics
- `im_gateway_group_messages_delivered_total` (counter) - Total group messages delivered
- `im_gateway_group_members_fanout_total` (counter) - Total group member fanouts

### Cache Metrics
- `im_gateway_cache_hits_total` (counter) - Total cache hits
- `im_gateway_cache_misses_total` (counter) - Total cache misses

## Configuration

Metrics are configured via environment variables:

```bash
# Enable OpenTelemetry metrics
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317

# Deployment environment
DEPLOYMENT_ENVIRONMENT=production
```

## Accessing Metrics

### Prometheus Endpoint
Metrics are exposed at `http://localhost:9090/metrics` in Prometheus format.

### OTLP Export
Metrics are also pushed to the OTLP endpoint configured via `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Querying Metrics

### Prometheus Queries

**Active connections:**
```promql
im_gateway_active_connections
```

**Message delivery rate (per second):**
```promql
rate(im_gateway_messages_delivered_total[5m])
```

**P50, P95, P99 latency:**
```promql
histogram_quantile(0.50, rate(im_gateway_message_delivery_latency_seconds_bucket[5m]))
histogram_quantile(0.95, rate(im_gateway_message_delivery_latency_seconds_bucket[5m]))
histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m]))
```

**ACK timeout rate:**
```promql
rate(im_gateway_ack_timeouts_total[5m]) / rate(im_gateway_messages_delivered_total[5m])
```

**Cross-gateway forward failure rate:**
```promql
sum(rate(im_gateway_cross_gateway_forward_total{result="failure"}[5m]))
/
clamp_min(sum(rate(im_gateway_cross_gateway_forward_total[5m])), 1)
```

**Kafka consumer lag (max across critical topics):**
```promql
max(im_gateway_kafka_consumer_lag{topic=~"group_msg|read_receipt|membership_change"})
```

**Cache hit rate:**
```promql
rate(im_gateway_cache_hits_total[5m]) / (rate(im_gateway_cache_hits_total[5m]) + rate(im_gateway_cache_misses_total[5m]))
```

**Message duplication rate:**
```promql
rate(im_gateway_duplicate_messages_total[5m]) / rate(im_gateway_messages_delivered_total[5m])
```

## Architecture

```
┌─────────────────────┐
│  IM Gateway Service │
│                     │
│  ┌───────────────┐ │
│  │ Metrics       │ │
│  │ (OTel SDK)    │ │
│  └───────┬───────┘ │
└──────────┼─────────┘
           │
           ├─────────────────┐
           │                 │
           ▼                 ▼
    ┌─────────────┐   ┌─────────────┐
    │ Prometheus  │   │ OTLP        │
    │ Exporter    │   │ Exporter    │
    │ (Pull)      │   │ (Push)      │
    └──────┬──────┘   └──────┬──────┘
           │                 │
           ▼                 ▼
    ┌─────────────┐   ┌─────────────┐
    │ Prometheus  │   │ OTEL        │
    │ Server      │   │ Collector   │
    └─────────────┘   └─────────────┘
```

## Testing

Run metrics tests:
```bash
cd apps/im-gateway-service
go test -v ./metrics/...
```

All tests validate that metrics operations don't panic and work correctly with concurrent access.

## Notes

- **Percentile Calculation**: With OpenTelemetry, percentiles (P50, P95, P99) are calculated by the metrics backend (Prometheus/Grafana), not in the application code.
- **Rate Calculation**: Rates (cache hit rate, duplication rate, etc.) are calculated using PromQL queries, not in the application.
- **Direct Reads**: OTel metrics don't support direct value reads. Use the metrics backend for querying current values.
