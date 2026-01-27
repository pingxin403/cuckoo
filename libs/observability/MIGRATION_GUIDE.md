# Observability Library Migration Guide

This guide helps you migrate services from custom metrics/logging implementations to the unified observability library with full OpenTelemetry support.

## Overview

The unified observability library provides:
- **Metrics**: OpenTelemetry Metrics SDK with dual export (OTLP + Prometheus)
- **Logging**: OpenTelemetry Logs SDK with automatic trace correlation
- **Tracing**: OpenTelemetry distributed tracing with OTLP export
- **Profiling**: Built-in pprof support for performance analysis

## Benefits

- **80% less code**: No need to implement metrics/logging in each service
- **Consistency**: Same patterns across all services
- **Vendor-neutral**: OpenTelemetry standard for portability
- **Trace correlation**: Automatic trace_id and span_id in logs
- **Dual export**: Support both OTLP and Prometheus simultaneously
- **Testing**: Easy to mock for unit tests
- **Maintainability**: Single place to fix bugs/add features

## Migration Steps

### Step 1: Add Dependency

Update your service's `go.mod`:

```go
require (
    github.com/pingxin403/cuckoo/libs/observability v0.0.0
)

replace github.com/pingxin403/cuckoo/libs/observability => ../../libs/observability
```

Run:
```bash
go get github.com/pingxin403/cuckoo/libs/observability
go mod tidy
```

### Step 2: Initialize Observability

**Before** (custom metrics):
```go
import "github.com/yourservice/metrics"

func main() {
    m := metrics.NewMetrics()
    http.HandleFunc("/metrics", m.Handler())
    // ... rest of setup
}
```

**After** (unified observability with OpenTelemetry):
```go
import "github.com/pingxin403/cuckoo/libs/observability"

func main() {
    obs, err := observability.New(observability.Config{
        ServiceName:    "your-service",
        ServiceVersion: "1.0.0",
        Environment:    os.Getenv("DEPLOYMENT_ENVIRONMENT"),
        
        // OpenTelemetry configuration
        OTLPEndpoint:   "localhost:4317",  // Unified OTLP endpoint
        UseOTelMetrics: true,              // Use OTel Metrics SDK
        UseOTelLogs:    true,              // Use OTel Logs SDK
        EnableTracing:  true,              // Enable distributed tracing
        
        // Metrics
        EnableMetrics:     true,
        MetricsPort:       9090,
        PrometheusEnabled: true,  // Dual export: OTLP + Prometheus
        
        // Logging
        LogLevel:  "info",
        LogFormat: "json",
    })
    if err != nil {
        log.Fatalf("Failed to initialize observability: %v", err)
    }
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = obs.Shutdown(ctx)
    }()

    // Metrics automatically exposed on :9090/metrics
    // Your application runs on a different port
}
```

### Step 3: Replace Logging

**Before** (standard log):
```go
log.Printf("Service started on port %s", port)
log.Printf("Error: %v", err)
```

**After** (structured logging):
```go
obs.Logger().Info(ctx, "Service started", "port", port)
obs.Logger().Error(ctx, "Operation failed", "error", err)

// With additional context
logger := obs.Logger().With("request_id", requestID)
logger.Info(ctx, "Processing request", "user_id", userID)
```

### Step 4: Replace Metrics

**Before** (custom metrics):
```go
m.IncrementActiveConnections()
m.DecrementActiveConnections()
m.IncrementMessagesDelivered()
m.ObserveLatency(duration)
```

**After** (unified metrics):
```go
// Counters
obs.Metrics().IncrementCounter("active_connections", nil)
obs.Metrics().IncrementCounter("messages_delivered_total", 
    map[string]string{"type": "private"})

// Gauges
obs.Metrics().SetGauge("active_connections", float64(count), nil)
obs.Metrics().IncrementGauge("goroutines", nil)
obs.Metrics().DecrementGauge("goroutines", nil)

// Histograms (for latency)
obs.Metrics().RecordDuration("message_delivery_duration_seconds", 
    duration, map[string]string{"success": "true"})
```

### Step 5: Remove Old Metrics Package

After migration is complete and tested:

```bash
# Remove old metrics package
rm -rf metrics/

# Update imports in all files
# Remove: import "github.com/yourservice/metrics"
# Add: import "github.com/pingxin403/cuckoo/libs/observability"
```

## Metric Naming Conventions

Follow Prometheus naming best practices:

### Counters (always increasing)
- Use `_total` suffix
- Examples: `requests_total`, `messages_delivered_total`, `errors_total`

### Gauges (can go up or down)
- No suffix
- Examples: `active_connections`, `queue_size`, `memory_bytes`

### Histograms (for latency/duration)
- Use `_seconds` suffix for duration
- Use `_bytes` suffix for size
- Examples: `request_duration_seconds`, `response_size_bytes`

### Labels
- Use lowercase with underscores
- Keep cardinality low (< 100 unique values per label)
- Examples: `method`, `status_code`, `error_type`

## Example: im-gateway-service Migration

### Before
```go
// main.go
import "github.com/pingxin403/cuckoo/apps/im-gateway-service/metrics"

func main() {
    metricsCollector := metrics.NewMetrics()
    log.Println("Metrics collector initialized")
    
    mux.HandleFunc("/metrics", metricsCollector.Handler())
    
    log.Printf("im-gateway-service listening on port %s", port)
}
```

### After
```go
// main.go
import "github.com/pingxin403/cuckoo/libs/observability"

func main() {
    obs, err := observability.New(observability.Config{
        ServiceName:    "im-gateway-service",
        ServiceVersion: "1.0.0",
        EnableMetrics:  true,
        MetricsPort:    9090,
        LogLevel:       "info",
        LogFormat:      "json",
    })
    if err != nil {
        log.Fatalf("Failed to initialize observability: %v", err)
    }
    defer obs.Shutdown(context.Background())

    obs.Logger().Info(ctx, "Service started", "port", port)
    
    // Metrics automatically exposed on :9090/metrics
}
```

## Testing

### Unit Tests

Use no-op implementations for testing:

```go
func TestMyFunction(t *testing.T) {
    obs, _ := observability.New(observability.Config{
        ServiceName:   "test",
        EnableMetrics: false, // Disable metrics in tests
        LogLevel:      "error", // Only log errors in tests
    })
    
    // Your test code
}
```

### Integration Tests

Enable metrics for integration tests:

```go
func TestIntegration(t *testing.T) {
    obs, _ := observability.New(observability.Config{
        ServiceName:   "test",
        EnableMetrics: true,
        MetricsPort:   9091, // Use different port
        LogLevel:      "debug",
    })
    defer obs.Shutdown(context.Background())
    
    // Your integration test code
}
```

## Migration Strategies

### Strategy 1: Full OpenTelemetry Migration (Recommended)

Migrate to full OpenTelemetry with OTLP export:

```go
obs, err := observability.New(observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    
    // OpenTelemetry
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
    
    // Disable Prometheus (OTLP only)
    PrometheusEnabled: false,
})
```

**Benefits**:
- Full OpenTelemetry integration
- Vendor-neutral observability
- Automatic trace-log correlation
- Future-proof architecture

**Requirements**:
- OpenTelemetry Collector running
- Update monitoring dashboards to use OTLP data

### Strategy 2: Dual Mode Migration (Gradual)

Keep Prometheus while adding OpenTelemetry:

```go
obs, err := observability.New(observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    
    // OpenTelemetry
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
    
    // Keep Prometheus (dual export)
    PrometheusEnabled: true,
    MetricsPort:       9090,
})
```

**Benefits**:
- Zero downtime migration
- Existing dashboards continue working
- Gradual transition to OpenTelemetry
- Easy rollback if needed

**Trade-offs**:
- Slightly higher resource usage (dual export)
- Temporary complexity

### Strategy 3: Prometheus-Only (Legacy)

Continue using Prometheus without OpenTelemetry:

```go
obs, err := observability.New(observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    
    // Disable OpenTelemetry
    UseOTelMetrics: false,
    UseOTelLogs:    false,
    EnableTracing:  false,
    
    // Prometheus only
    PrometheusEnabled: true,
    MetricsPort:       9090,
})
```

**Use when**:
- Not ready for OpenTelemetry
- Existing Prometheus infrastructure
- Simple monitoring needs

## Configuration

### Environment Variables

The library supports environment variables for configuration:

**Service Identification**:
- `SERVICE_NAME`: Service name (default: "unknown-service")
- `SERVICE_VERSION`: Service version (default: "unknown")
- `DEPLOYMENT_ENVIRONMENT`: Environment (default: "development")

**OpenTelemetry**:
- `OTLP_ENDPOINT`: Unified OTLP endpoint (default: "")
- `OTLP_METRICS_ENDPOINT`: Override for metrics (default: uses OTLP_ENDPOINT)
- `OTLP_LOGS_ENDPOINT`: Override for logs (default: uses OTLP_ENDPOINT)
- `OTLP_PROTOCOL`: Protocol (grpc or http, default: "grpc")
- `USE_OTEL_METRICS`: Use OTel Metrics SDK (default: "false")
- `USE_OTEL_LOGS`: Use OTel Logs SDK (default: "false")
- `PROMETHEUS_ENABLED`: Enable Prometheus exporter (default: "true")

**Metrics**:
- `METRICS_PORT`: Metrics server port (default: 9090)
- `METRICS_PATH`: Metrics endpoint path (default: "/metrics")

**Tracing**:
- `TRACING_ENABLED`: Enable tracing (default: "false")
- `TRACING_ENDPOINT`: Tracing endpoint (default: "")
- `TRACING_SAMPLE_RATE`: Sample rate 0.0-1.0 (default: "0.1")

**Logging**:
- `LOG_LEVEL`: Log level (default: "info")
- `LOG_FORMAT`: Log format (default: "json")
- `LOG_OUTPUT`: Log output (default: "stdout")

### Example with Environment Variables

```bash
# Full OpenTelemetry configuration
export SERVICE_NAME=my-service
export SERVICE_VERSION=1.0.0
export DEPLOYMENT_ENVIRONMENT=production
export OTLP_ENDPOINT=localhost:4317
export USE_OTEL_METRICS=true
export USE_OTEL_LOGS=true
export TRACING_ENABLED=true
export PROMETHEUS_ENABLED=false
```

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    // Other fields will be read from environment variables
}.WithDefaults())
```

## Troubleshooting

### Metrics not appearing

1. Check metrics server is running: `curl http://localhost:9090/metrics`
2. Verify `EnableMetrics: true` in config
3. Check logs for errors during initialization

### Logs not appearing

1. Check log level is appropriate (debug < info < warn < error)
2. Verify log output destination (stdout/stderr)
3. Check log format (json/text)

### Build errors

1. Run `go mod tidy` to update dependencies
2. Verify replace directive in go.mod points to correct path
3. Check import paths are correct

## Next Steps

After migrating to the observability library:

1. **Add middleware** (Phase 2): Automatic instrumentation for HTTP/gRPC
2. **Enable tracing** (Phase 4): OpenTelemetry distributed tracing
3. **Create dashboards**: Grafana dashboards for your service
4. **Set up alerts**: Prometheus alerting rules

## Support

For questions or issues:
- Check the [README](README.md) for detailed documentation
- Review [example_test.go](example_test.go) for usage examples
- See [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for roadmap

## Checklist

- [ ] Add observability dependency to go.mod
- [ ] Initialize observability in main.go
- [ ] Replace log.Printf with obs.Logger()
- [ ] Replace custom metrics with obs.Metrics()
- [ ] Update metric names to follow conventions
- [ ] Remove old metrics package
- [ ] Update tests to use observability
- [ ] Verify metrics endpoint works
- [ ] Verify logs are structured
- [ ] Update documentation


## Migrating from Prometheus to OpenTelemetry Metrics

### Understanding the Difference

**Prometheus Metrics** (old):
- Pull-based model (Prometheus scrapes `/metrics`)
- Prometheus exposition format
- Limited to Prometheus ecosystem

**OpenTelemetry Metrics** (new):
- Push-based model (service pushes to OTLP collector)
- Vendor-neutral format
- Works with any OTLP-compatible backend
- Supports dual export (OTLP + Prometheus)

### Migration Steps

#### Step 1: Enable OpenTelemetry Metrics

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Enable OTel Metrics
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    
    // Keep Prometheus during transition
    PrometheusEnabled: true,
    MetricsPort:       9090,
})
```

#### Step 2: Verify Dual Export

Both OTLP and Prometheus should work:

```bash
# Check Prometheus endpoint (should still work)
curl http://localhost:9090/metrics

# Check OTLP export (check collector logs)
# Metrics should appear in your OTLP backend
```

#### Step 3: Update Dashboards

Create new dashboards using OTLP data source while keeping old Prometheus dashboards.

#### Step 4: Disable Prometheus (Optional)

Once confident with OpenTelemetry:

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // OTel only
    OTLPEndpoint:      "localhost:4317",
    UseOTelMetrics:    true,
    PrometheusEnabled: false,  // Disable Prometheus
})
```

### Metric Type Mapping

OpenTelemetry metrics map to Prometheus types:

| Prometheus | OpenTelemetry | Notes |
|------------|---------------|-------|
| Counter | Counter | Always increasing |
| Gauge | Gauge | Can go up or down |
| Histogram | Histogram | Distribution with buckets |
| Summary | (not supported) | Use Histogram instead |

### Code Changes

No code changes needed! The API remains the same:

```go
// Works with both Prometheus and OpenTelemetry
obs.Metrics().IncrementCounter("requests_total", labels)
obs.Metrics().SetGauge("active_connections", value, labels)
obs.Metrics().RecordHistogram("duration_seconds", duration, labels)
```

## Migrating from Structured Logging to OpenTelemetry Logs

### Understanding the Difference

**Structured Logging** (old):
- Logs written to stdout/stderr
- No automatic trace correlation
- Manual log aggregation needed

**OpenTelemetry Logs** (new):
- Logs sent to OTLP collector
- Automatic trace_id and span_id injection
- Integrated with traces and metrics
- Vendor-neutral format

### Migration Steps

#### Step 1: Enable OpenTelemetry Logs

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Enable OTel Logs
    OTLPEndpoint: "localhost:4317",
    UseOTelLogs:  true,
    
    LogLevel:  "info",
    LogFormat: "json",
})
```

#### Step 2: Verify Trace Correlation

Logs should now include trace_id and span_id:

```json
{
  "timestamp": "2025-01-24T12:34:56.789Z",
  "level": "info",
  "service": "my-service",
  "message": "Processing request",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "user_id": "user123"
}
```

#### Step 3: Update Log Aggregation

Configure your OTLP collector to forward logs to your log aggregation system (Loki, Elasticsearch, etc.).

### Code Changes

No code changes needed! The API remains the same:

```go
// Works with both structured logging and OpenTelemetry
obs.Logger().Info(ctx, "Processing request", "user_id", "user123")
obs.Logger().Error(ctx, "Operation failed", "error", err)
```

### Trace-Log Correlation

When using OpenTelemetry logs with tracing enabled, logs automatically include trace context:

```go
// Start a span
ctx, span := obs.Tracer().StartSpan(ctx, "process-request")
defer span.End()

// Log within span - trace_id and span_id automatically added
obs.Logger().Info(ctx, "Processing started", "user_id", "user123")
```

## Troubleshooting

### Metrics not appearing in OTLP backend

1. **Check OTLP collector is running**:
   ```bash
   curl http://localhost:4317
   ```

2. **Verify configuration**:
   ```go
   OTLPEndpoint: "localhost:4317"  // Correct
   OTLPEndpoint: "http://localhost:4317"  // Wrong (no http://)
   ```

3. **Check collector logs** for export errors

4. **Verify UseOTelMetrics is true**:
   ```go
   UseOTelMetrics: true
   ```

### Logs not appearing in OTLP backend

1. **Verify UseOTelLogs is true**:
   ```go
   UseOTelLogs: true
   ```

2. **Check OTLP endpoint** is correct

3. **Verify collector configuration** accepts logs

### Trace correlation not working

1. **Ensure tracing is enabled**:
   ```go
   EnableTracing: true
   ```

2. **Pass context to logger**:
   ```go
   obs.Logger().Info(ctx, "message")  // Correct
   obs.Logger().Info(context.Background(), "message")  // Wrong
   ```

3. **Verify span is active**:
   ```go
   ctx, span := obs.Tracer().StartSpan(ctx, "operation")
   defer span.End()
   obs.Logger().Info(ctx, "message")  // Will include trace_id
   ```

### High resource usage with dual export

1. **Disable Prometheus** if not needed:
   ```go
   PrometheusEnabled: false
   ```

2. **Adjust batch settings**:
   ```go
   OTLPBatchSize:      512,
   OTLPExportInterval: 10 * time.Second,
   ```

3. **Reduce sample rate** for tracing:
   ```go
   TracingSampleRate: 0.1  // 10% instead of 100%
   ```

## Next Steps

After migrating to the observability library:

1. **Deploy OpenTelemetry Collector**: Use the provided Docker Compose or Kubernetes configurations in `deploy/`
2. **Create dashboards**: Build Grafana dashboards using OTLP data sources
3. **Set up alerts**: Configure alerting rules in your observability backend
4. **Enable tracing**: Add distributed tracing to track requests across services
5. **Optimize**: Tune batch sizes, sample rates, and export intervals for your workload

## Support

For questions or issues:
- Check the [README](README.md) for detailed documentation
- Review [OPENTELEMETRY_GUIDE.md](OPENTELEMETRY_GUIDE.md) for tracing guide
- See [example_test.go](example_test.go) for usage examples
- Check deployment guides in `deploy/docker/OBSERVABILITY.md` and `deploy/k8s/observability/README.md`

## Checklist

- [ ] Add observability dependency to go.mod
- [ ] Initialize observability in main.go with OpenTelemetry configuration
- [ ] Replace log.Printf with obs.Logger()
- [ ] Replace custom metrics with obs.Metrics()
- [ ] Update metric names to follow conventions
- [ ] Remove old metrics package
- [ ] Update tests to use observability
- [ ] Deploy OpenTelemetry Collector
- [ ] Verify OTLP export works
- [ ] Verify Prometheus endpoint works (if dual mode)
- [ ] Verify trace-log correlation
- [ ] Update dashboards to use OTLP data
- [ ] Update documentation
