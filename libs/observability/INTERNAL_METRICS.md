# Internal Observability Metrics

This document describes the internal metrics exposed by the observability library to monitor its own health and performance.

## Overview

The observability library exposes internal metrics to track:
- Export failures
- Instrument creation failures
- Total operations
- Cached instruments
- Log entries by level
- Trace correlation

These metrics help you monitor the health of your observability infrastructure and identify issues before they impact your application.

## Metrics Collector Internal Metrics

The `OTelMetricsCollector` exposes the following internal metrics:

### `otel.metrics.export_failures`
- **Type**: Gauge
- **Description**: Number of metric export failures
- **Use**: Monitor OTLP connection issues or exporter problems

### `otel.metrics.instrument_failures`
- **Type**: Gauge
- **Description**: Number of metric instrument creation failures
- **Use**: Detect issues with metric registration

### `otel.metrics.total_operations`
- **Type**: Gauge
- **Description**: Total number of metric operations (counter increments, gauge sets, histogram records)
- **Use**: Monitor metric recording throughput

### `otel.metrics.cached_counters`
- **Type**: Gauge
- **Description**: Number of cached counter instruments
- **Use**: Monitor memory usage and instrument cardinality

### `otel.metrics.cached_histograms`
- **Type**: Gauge
- **Description**: Number of cached histogram instruments
- **Use**: Monitor memory usage and instrument cardinality

### `otel.metrics.cached_gauges`
- **Type**: Gauge
- **Description**: Number of cached gauge instruments
- **Use**: Monitor memory usage and instrument cardinality

## Logger Internal Metrics

The `OTelLogger` tracks the following internal metrics (accessible via `GetInternalMetrics()` for testing):

### Export Failures
- **Key**: `export_failures`
- **Description**: Number of log export failures
- **Use**: Monitor OTLP connection issues

### Total Logs
- **Key**: `total_logs`
- **Description**: Total number of log entries emitted
- **Use**: Monitor logging throughput

### Logs by Level
- **Keys**: `debug_logs`, `info_logs`, `warn_logs`, `error_logs`
- **Description**: Number of log entries at each level
- **Use**: Monitor log level distribution

### Logs with Trace Context
- **Key**: `logs_with_trace`
- **Description**: Number of log entries that include trace context
- **Use**: Monitor trace-log correlation effectiveness

## Accessing Internal Metrics

### Metrics Collector

Internal metrics are automatically registered as observable gauges and exported via OTLP and/or Prometheus:

```go
// Metrics are automatically exposed via the metrics endpoint
// Example: http://localhost:9090/metrics

// For testing, you can access metrics programmatically:
collector, _ := metrics.NewOTelMetricsCollector(config)
internalMetrics := collector.GetInternalMetrics()
fmt.Printf("Total operations: %d\n", internalMetrics["total_operations"])
```

### Logger

Logger internal metrics are accessible via the `GetInternalMetrics()` method (primarily for testing):

```go
logger, _ := logging.NewOTelLogger(config)
internalMetrics := logger.GetInternalMetrics()
fmt.Printf("Total logs: %d\n", internalMetrics["total_logs"])
fmt.Printf("Logs with trace: %d\n", internalMetrics["logs_with_trace"])
```

## Monitoring Best Practices

### Alert on Export Failures

Set up alerts when `otel.metrics.export_failures` or `export_failures` increases:

```yaml
# Example Prometheus alert
- alert: ObservabilityExportFailures
  expr: otel_metrics_export_failures > 0
  for: 5m
  annotations:
    summary: "Observability export failures detected"
    description: "{{ $value }} export failures in the last 5 minutes"
```

### Monitor Instrument Cardinality

Track `cached_counters`, `cached_histograms`, and `cached_gauges` to detect cardinality explosions:

```yaml
# Example Prometheus alert
- alert: HighMetricCardinality
  expr: otel_metrics_cached_counters + otel_metrics_cached_histograms + otel_metrics_cached_gauges > 10000
  for: 10m
  annotations:
    summary: "High metric cardinality detected"
    description: "Total cached instruments: {{ $value }}"
```

### Monitor Trace-Log Correlation

Track the ratio of logs with trace context to ensure proper instrumentation:

```go
// In your monitoring dashboard
correlation_ratio = logs_with_trace / total_logs
// Alert if correlation_ratio < 0.5 (less than 50% of logs have trace context)
```

## Performance Impact

Internal metrics have minimal performance impact:
- Metrics use atomic operations for thread-safe updates
- Observable gauges are evaluated only during export (not on every operation)
- No additional allocations during normal operations

## OpenTelemetry SDK Built-in Features

The OpenTelemetry SDK provides additional built-in features:

### Automatic Retry Logic
- Exponential backoff for failed exports
- Configurable retry attempts
- Automatic buffering of failed batches

### Batch Processing
- Efficient batching of telemetry data
- Configurable batch size and export interval
- Automatic flushing on shutdown

### Resource Attributes
- Automatic inclusion of service name, version, environment
- Host and process information
- Custom resource attributes

## Troubleshooting

### High Export Failures

If you see high `export_failures`:
1. Check OTLP endpoint connectivity
2. Verify network configuration
3. Check collector logs for errors
4. Ensure collector has sufficient resources

### High Instrument Cardinality

If you see high cached instrument counts:
1. Review metric label usage
2. Avoid high-cardinality labels (e.g., user IDs, timestamps)
3. Use aggregation where possible
4. Consider metric sampling for high-volume metrics

### Low Trace-Log Correlation

If `logs_with_trace` is low:
1. Ensure context.Context is passed to logging calls
2. Verify tracing is enabled
3. Check that spans are properly created
4. Review instrumentation coverage

## References

- [OpenTelemetry Metrics Specification](https://opentelemetry.io/docs/specs/otel/metrics/)
- [OpenTelemetry Logs Specification](https://opentelemetry.io/docs/specs/otel/logs/)
- [Observability Library README](./README.md)
- [OpenTelemetry Guide](./OPENTELEMETRY_GUIDE.md)
