# Flash Sale Service - Metrics Documentation

This document describes the Prometheus metrics exposed by the flash-sale-service, following the project's observability patterns.

## Overview

The flash-sale-service exposes metrics via Spring Boot Actuator with Micrometer, providing comprehensive observability for:
- Request throughput (QPS)
- Response times (latency)
- Success/failure rates
- Inventory levels
- Queue lengths

## Metrics Endpoint

Metrics are exposed at:
```
http://localhost:9091/actuator/prometheus
```

The service also supports OTLP export for integration with OpenTelemetry collectors.

## Available Metrics

### Request Metrics

#### `flash_sale_requests_total`
- **Type**: Counter
- **Description**: Total number of seckill requests received
- **Labels**: None
- **Use**: Calculate QPS (queries per second)

#### `flash_sale_requests_success`
- **Type**: Counter
- **Description**: Total number of successful seckill requests
- **Labels**: None
- **Use**: Calculate success rate

#### `flash_sale_requests_failure`
- **Type**: Counter
- **Description**: Total number of failed seckill requests
- **Labels**: None
- **Use**: Calculate failure rate

#### `flash_sale_request_duration`
- **Type**: Timer (Histogram + Summary)
- **Description**: Duration of seckill request processing
- **Labels**: None
- **Percentiles**: p50, p95, p99
- **Buckets**: 50ms, 100ms, 200ms, 500ms, 1s
- **Use**: Monitor response time and latency

### Inventory Metrics

#### `flash_sale_inventory_deductions_total`
- **Type**: Counter
- **Description**: Total number of successful inventory deductions
- **Labels**: None
- **Use**: Track inventory operations

#### `flash_sale_inventory_rollbacks_total`
- **Type**: Counter
- **Description**: Total number of inventory rollbacks (timeout/cancellation)
- **Labels**: None
- **Use**: Monitor rollback frequency

#### `flash_sale_inventory_remaining`
- **Type**: Gauge
- **Description**: Remaining inventory for a specific SKU
- **Labels**: `sku_id`
- **Use**: Real-time inventory monitoring

#### `flash_sale_inventory_deduction_duration`
- **Type**: Timer (Histogram + Summary)
- **Description**: Duration of inventory deduction operations
- **Labels**: None
- **Use**: Monitor Redis Lua script performance

### Queue Metrics

#### `flash_sale_queue_tokens_acquired`
- **Type**: Counter
- **Description**: Total number of queue tokens successfully acquired
- **Labels**: None
- **Use**: Track successful queue entries

#### `flash_sale_queue_tokens_rejected`
- **Type**: Counter
- **Description**: Total number of queue tokens rejected (queuing)
- **Labels**: None
- **Use**: Monitor queue pressure

#### `flash_sale_queue_length`
- **Type**: Gauge
- **Description**: Current estimated queue length
- **Labels**: None
- **Use**: Real-time queue monitoring

#### `flash_sale_queue_acquisition_duration`
- **Type**: Timer (Histogram + Summary)
- **Description**: Duration of queue token acquisition
- **Labels**: None
- **Use**: Monitor queue service performance

## Standard Spring Boot Metrics

In addition to custom metrics, the service exposes standard Spring Boot Actuator metrics:

### HTTP Server Metrics
- `http_server_requests_total` - Total HTTP requests
- `http_server_requests_seconds` - HTTP request duration
- `http_server_requests_active` - Active HTTP requests

### JVM Metrics
- `jvm_memory_used_bytes` - JVM memory usage
- `jvm_gc_pause_seconds` - Garbage collection pause time
- `jvm_threads_live` - Number of live threads

### System Metrics
- `system_cpu_usage` - System CPU usage
- `process_cpu_usage` - Process CPU usage
- `system_load_average_1m` - System load average

## Prometheus Configuration

### Scrape Configuration

Add the following to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'flash-sale-service'
    metrics_path: '/actuator/prometheus'
    scrape_interval: 10s
    static_configs:
      - targets: ['localhost:9091']
        labels:
          application: 'flash-sale-service'
          environment: 'production'
```

### Example Queries

#### Calculate QPS (Queries Per Second)
```promql
rate(flash_sale_requests_total[1m])
```

#### Calculate Success Rate
```promql
rate(flash_sale_requests_success[5m]) / rate(flash_sale_requests_total[5m]) * 100
```

#### Calculate Failure Rate
```promql
rate(flash_sale_requests_failure[5m]) / rate(flash_sale_requests_total[5m]) * 100
```

#### Monitor P99 Response Time
```promql
histogram_quantile(0.99, rate(flash_sale_request_duration_bucket[5m]))
```

#### Monitor Inventory Remaining
```promql
flash_sale_inventory_remaining{sku_id="SKU001"}
```

#### Monitor Queue Length
```promql
flash_sale_queue_length
```

#### Calculate Inventory Deduction Rate
```promql
rate(flash_sale_inventory_deductions_total[1m])
```

#### Calculate Rollback Rate
```promql
rate(flash_sale_inventory_rollbacks_total[1m])
```

## Grafana Dashboard

### Recommended Panels

1. **QPS Panel**
   - Query: `rate(flash_sale_requests_total[1m])`
   - Visualization: Graph
   - Unit: requests/sec

2. **Success Rate Panel**
   - Query: `rate(flash_sale_requests_success[5m]) / rate(flash_sale_requests_total[5m]) * 100`
   - Visualization: Gauge
   - Unit: percent
   - Thresholds: Green > 95%, Yellow > 90%, Red < 90%

3. **Response Time Panel**
   - Queries:
     - P50: `histogram_quantile(0.50, rate(flash_sale_request_duration_bucket[5m]))`
     - P95: `histogram_quantile(0.95, rate(flash_sale_request_duration_bucket[5m]))`
     - P99: `histogram_quantile(0.99, rate(flash_sale_request_duration_bucket[5m]))`
   - Visualization: Graph
   - Unit: seconds

4. **Inventory Remaining Panel**
   - Query: `flash_sale_inventory_remaining`
   - Visualization: Graph
   - Unit: items

5. **Queue Length Panel**
   - Query: `flash_sale_queue_length`
   - Visualization: Graph
   - Unit: requests

6. **Inventory Operations Panel**
   - Queries:
     - Deductions: `rate(flash_sale_inventory_deductions_total[1m])`
     - Rollbacks: `rate(flash_sale_inventory_rollbacks_total[1m])`
   - Visualization: Graph
   - Unit: operations/sec

## Alerting Rules

### Example Alert Rules

```yaml
groups:
  - name: flash_sale_alerts
    interval: 30s
    rules:
      # High failure rate
      - alert: HighFailureRate
        expr: |
          rate(flash_sale_requests_failure[5m]) / rate(flash_sale_requests_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High failure rate detected"
          description: "Failure rate is {{ $value | humanizePercentage }} over the last 5 minutes"

      # High response time
      - alert: HighResponseTime
        expr: |
          histogram_quantile(0.99, rate(flash_sale_request_duration_bucket[5m])) > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P99 response time"
          description: "P99 response time is {{ $value }}s"

      # Low inventory
      - alert: LowInventory
        expr: |
          flash_sale_inventory_remaining < 100
        for: 1m
        labels:
          severity: info
        annotations:
          summary: "Low inventory for SKU {{ $labels.sku_id }}"
          description: "Only {{ $value }} items remaining"

      # High queue length
      - alert: HighQueueLength
        expr: |
          flash_sale_queue_length > 1000
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High queue length"
          description: "Queue length is {{ $value }}"

      # High rollback rate
      - alert: HighRollbackRate
        expr: |
          rate(flash_sale_inventory_rollbacks_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High inventory rollback rate"
          description: "Rollback rate is {{ $value }} per second"
```

## Configuration

### Application Configuration

Metrics configuration is defined in `application.yml`:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health,info,metrics,prometheus
  metrics:
    tags:
      application: flash-sale-service
      environment: ${SPRING_PROFILES_ACTIVE:local}
    export:
      prometheus:
        enabled: true
        step: 10s
    distribution:
      percentiles-histogram:
        flash_sale.request.duration: true
        flash_sale.inventory.deduction.duration: true
        flash_sale.queue.acquisition.duration: true
      slo:
        flash_sale.request.duration: 50ms,100ms,200ms,500ms,1s
  server:
    port: 9091
```

### Environment Variables

- `TRACING_SAMPLE_RATE`: Tracing sample rate (default: 0.1)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint for metrics export (default: http://localhost:4317)
- `OTEL_METRICS_ENABLED`: Enable OTLP metrics export (default: false)

## Integration with Existing Observability Stack

The flash-sale-service integrates with the existing Prometheus + Grafana observability stack:

1. **Prometheus**: Scrapes metrics from `/actuator/prometheus` endpoint
2. **Grafana**: Visualizes metrics with custom dashboards
3. **OpenTelemetry** (optional): Exports metrics via OTLP protocol

## Performance Considerations

- Metrics collection has minimal overhead (< 1μs per operation)
- Histogram buckets are optimized for flash sale latency patterns
- Gauges are updated asynchronously to avoid blocking operations
- Counter increments are thread-safe and lock-free

## Troubleshooting

### Metrics Not Appearing

1. Check that actuator endpoint is accessible:
   ```bash
   curl http://localhost:9091/actuator/prometheus
   ```

2. Verify Prometheus scrape configuration

3. Check application logs for errors

### High Cardinality Issues

- Avoid adding high-cardinality labels (e.g., user_id, order_id)
- Use aggregation for SKU-level metrics
- Monitor Prometheus memory usage

## References

- [Micrometer Documentation](https://micrometer.io/docs)
- [Spring Boot Actuator](https://docs.spring.io/spring-boot/docs/current/reference/html/actuator.html)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Project Observability Library](../../libs/observability/README.md)
