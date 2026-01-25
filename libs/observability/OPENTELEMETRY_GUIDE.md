# OpenTelemetry Guide

This guide explains how to use OpenTelemetry for distributed tracing, metrics, and logging with the observability library.

## Overview

OpenTelemetry (OTel) is an open-source observability framework for cloud-native software. It provides:
- **Distributed tracing**: Track requests across multiple services
- **Metrics**: Collect and export metrics in a vendor-neutral format
- **Logging**: Structured logs with automatic trace correlation
- **Context propagation**: Pass trace context between services

## Prerequisites

To use OpenTelemetry, you need:
1. An OpenTelemetry collector running (e.g., Jaeger, or OTLP collector)
2. The collector endpoint (e.g., `localhost:4317` for gRPC)

## Quick Start

### 1. Enable OpenTelemetry

```go
import "github.com/pingxin403/cuckoo/libs/observability"

obs, err := observability.New(observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    
    // Unified OTLP endpoint
    OTLPEndpoint: "localhost:4317",
    
    // Enable all OpenTelemetry features
    UseOTelMetrics: true,  // OTel Metrics SDK
    UseOTelLogs:    true,  // OTel Logs SDK
    EnableTracing:  true,  // Distributed tracing
    
    // Optional: Keep Prometheus for dual export
    PrometheusEnabled: true,
    MetricsPort:       9090,
    
    // Tracing configuration
    TracingSampleRate: 0.1,  // Sample 10% of traces
    
    // Logging
    LogLevel: "info",
})
if err != nil {
    log.Fatalf("Failed to initialize observability: %v", err)
}
defer obs.Shutdown(context.Background())
```

## OpenTelemetry Metrics

### Overview

OpenTelemetry Metrics SDK provides vendor-neutral metric collection and export. The observability library supports:
- **Counters**: Always-increasing values (e.g., request count)
- **Gauges**: Values that can go up or down (e.g., active connections)
- **Histograms**: Distribution of values (e.g., request duration)

### Enabling OTel Metrics

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Enable OTel Metrics
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    
    // Optional: Dual export (OTLP + Prometheus)
    PrometheusEnabled: true,
    MetricsPort:       9090,
})
```

### Recording Metrics

The API is the same whether using Prometheus or OpenTelemetry:

```go
// Counter - always increasing
obs.Metrics().IncrementCounter("requests_total", map[string]string{
    "method": "GET",
    "status": "200",
})

obs.Metrics().AddCounter("bytes_sent_total", 1024, map[string]string{
    "protocol": "http",
})

// Gauge - can go up or down
obs.Metrics().SetGauge("active_connections", 42, nil)
obs.Metrics().IncrementGauge("queue_size", nil)
obs.Metrics().DecrementGauge("queue_size", nil)

// Histogram - distribution
obs.Metrics().RecordHistogram("request_duration_seconds", 0.123, map[string]string{
    "endpoint": "/api/users",
})

obs.Metrics().RecordDuration("request_duration_seconds",
    150*time.Millisecond,
    map[string]string{"endpoint": "/api/users"},
)
```

### Dual Export Mode

You can export metrics to both OTLP and Prometheus simultaneously:

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // OTLP export
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    
    // Prometheus export
    PrometheusEnabled: true,
    MetricsPort:       9090,
})
```

**Benefits**:
- Zero downtime migration from Prometheus to OpenTelemetry
- Existing Prometheus dashboards continue working
- Gradual transition to OTLP-based monitoring

**Trade-offs**:
- Slightly higher resource usage (metrics exported twice)
- Temporary complexity during migration

### Metric Configuration

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // OTLP configuration
    OTLPEndpoint:       "localhost:4317",
    OTLPMetricsEndpoint: "",  // Optional override
    OTLPProtocol:       "grpc",  // grpc or http
    OTLPTimeout:        10 * time.Second,
    OTLPBatchSize:      512,
    OTLPExportInterval: 10 * time.Second,
    
    // Enable OTel Metrics
    UseOTelMetrics: true,
})
```

## OpenTelemetry Logs

### Overview

OpenTelemetry Logs SDK provides structured logging with automatic trace correlation. When a log is written within an active span, the trace_id and span_id are automatically injected.

### Enabling OTel Logs

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

### Writing Logs

The API is the same whether using structured logging or OpenTelemetry:

```go
ctx := context.Background()

// Simple logging
obs.Logger().Info(ctx, "Service started", "port", 8080)
obs.Logger().Debug(ctx, "Debug message", "key", "value")
obs.Logger().Warn(ctx, "Warning message", "reason", "timeout")
obs.Logger().Error(ctx, "Error occurred", "error", err)

// With context
logger := obs.Logger().With("request_id", "abc123")
logger.Info(ctx, "Processing request", "user_id", "user456")
```

### Trace-Log Correlation

When using OpenTelemetry logs with tracing enabled, logs automatically include trace context:

```go
// Start a span
ctx, span := obs.Tracer().StartSpan(ctx, "process-request")
defer span.End()

span.SetAttribute("user_id", "user123")

// Log within span - trace_id and span_id automatically added
obs.Logger().Info(ctx, "Processing started", "user_id", "user123")
// Output includes: trace_id=4bf92f3577b34da6a3ce929d0e0e4736 span_id=00f067aa0ba902b7

// Do work...
result, err := doWork(ctx)
if err != nil {
    span.RecordError(err)
    obs.Logger().Error(ctx, "Work failed", "error", err)
    // Error log includes same trace_id and span_id
}
```

**Benefits**:
- Correlate logs with traces in your observability backend
- Find all logs for a specific trace
- Debug issues by following the trace through logs
- Unified view of traces and logs

### Log Format

With OpenTelemetry logs enabled, logs include trace context:

```json
{
  "timestamp": "2025-01-24T12:34:56.789Z",
  "level": "info",
  "service": "my-service",
  "message": "Processing request",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "user_id": "user123",
  "request_id": "abc123"
}
```

### Log Level Mapping

OpenTelemetry severity levels map to standard log levels:

| Log Level | OTel Severity | Numeric Value |
|-----------|---------------|---------------|
| debug | DEBUG | 5 |
| info | INFO | 9 |
| warn | WARN | 13 |
| error | ERROR | 17 |

## OpenTelemetry Tracing

### Overview

Distributed tracing tracks requests as they flow through multiple services, helping you understand system behavior and diagnose issues.

### Enabling Tracing

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Enable tracing
    OTLPEndpoint:      "localhost:4317",
    EnableTracing:     true,
    TracingSampleRate: 0.1,  // Sample 10% of traces
})
```

### Creating Spans

```go
ctx := context.Background()

// Start a span
ctx, span := obs.Tracer().StartSpan(ctx, "process-request")
defer span.End()

// Add attributes
span.SetAttribute("user_id", "user123")
span.SetAttribute("request_size", 1024)

// Do work...
result, err := doWork(ctx)
if err != nil {
    span.RecordError(err)
    span.SetStatus(tracing.StatusCodeError, "work failed")
    return err
}

span.SetStatus(tracing.StatusCodeOK, "success")
```

```go
ctx := context.Background()

// Start a span
ctx, span := obs.Tracer().StartSpan(ctx, "process-request")
defer span.End()

// Add attributes
span.SetAttribute("user_id", "user123")
span.SetAttribute("request_size", 1024)

// Do work...
result, err := doWork(ctx)
if err != nil {
    span.RecordError(err)
    span.SetStatus(tracing.StatusCodeError, "work failed")
    return err
}

span.SetStatus(tracing.StatusCodeOK, "success")
```

### Creating Child Spans

```go
func doWork(ctx context.Context) error {
    // Create child span
    ctx, span := obs.Tracer().StartSpan(ctx, "database-query")
    defer span.End()
    
    span.SetAttribute("query", "SELECT * FROM users")
    
    // Execute query...
    return nil
}
```

## Unified Configuration

### Single Endpoint for All Signals

Use a single OTLP endpoint for metrics, logs, and traces:

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Unified endpoint for all signals
    OTLPEndpoint: "localhost:4317",
    
    // Enable all signals
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
})
```

### Signal-Specific Endpoints

Override endpoints for specific signals:

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    
    // Default endpoint
    OTLPEndpoint: "localhost:4317",
    
    // Override for specific signals
    OTLPMetricsEndpoint: "metrics-collector:4317",
    OTLPLogsEndpoint:    "logs-collector:4317",
    TracingEndpoint:     "traces-collector:4317",
    
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
})
```

```go
func doWork(ctx context.Context) error {
    // Create child span
    ctx, span := obs.Tracer().StartSpan(ctx, "database-query")
    defer span.End()
    
    span.SetAttribute("query", "SELECT * FROM users")
    
    // Execute query...
    return nil
}
```

## Configuration

### Sample Rate

The sample rate determines what percentage of traces to collect:

```go
TracingSampleRate: 0.1,  // 10% of traces
TracingSampleRate: 1.0,  // 100% of traces (all)
TracingSampleRate: 0.01, // 1% of traces
```

**Recommendations**:
- **Development**: 1.0 (100%) - See all traces
- **Staging**: 0.5 (50%) - Balance between visibility and cost
- **Production**: 0.1 (10%) or lower - Reduce overhead

### Endpoint Configuration

The tracing endpoint is where traces are sent:

```go
// Local development (Jaeger)
TracingEndpoint: "localhost:4317"

// Kubernetes service
TracingEndpoint: "otel-collector.observability.svc.cluster.local:4317"

// Cloud service
TracingEndpoint: "api.honeycomb.io:443"
```

## Span Options

### Span Kind

Specify the type of span:

```go
import "github.com/pingxin403/cuckoo/libs/observability/tracing"

// Server span (handling incoming request)
ctx, span := obs.Tracer().StartSpan(ctx, "handle-request",
    tracing.WithSpanKind(tracing.SpanKindServer),
)

// Client span (making outgoing request)
ctx, span := obs.Tracer().StartSpan(ctx, "call-api",
    tracing.WithSpanKind(tracing.SpanKindClient),
)

// Internal span (internal operation)
ctx, span := obs.Tracer().StartSpan(ctx, "process-data",
    tracing.WithSpanKind(tracing.SpanKindInternal),
)
```

### Initial Attributes

Set attributes when creating a span:

```go
ctx, span := obs.Tracer().StartSpan(ctx, "process-order",
    tracing.WithAttributes(map[string]interface{}{
        "order_id":    "order123",
        "customer_id": "cust456",
        "amount":      99.99,
    }),
)
defer span.End()
```

## Span Attributes

### Adding Attributes

```go
// Single attribute
span.SetAttribute("user_id", "user123")
span.SetAttribute("request_size", 1024)
span.SetAttribute("is_premium", true)

// Multiple attributes
span.SetAttributes(map[string]interface{}{
    "method":      "POST",
    "path":        "/api/users",
    "status_code": 200,
})
```

### Attribute Types

Supported attribute types:
- `string`: Text values
- `int`, `int64`: Integer values
- `float64`: Floating-point values
- `bool`: Boolean values
- Other types are converted to strings

### Semantic Conventions

Follow OpenTelemetry semantic conventions for common attributes:

```go
// HTTP attributes
span.SetAttributes(map[string]interface{}{
    "http.method":      "GET",
    "http.url":         "/api/users",
    "http.status_code": 200,
    "http.user_agent":  "Mozilla/5.0...",
})

// Database attributes
span.SetAttributes(map[string]interface{}{
    "db.system":    "postgresql",
    "db.name":      "mydb",
    "db.statement": "SELECT * FROM users WHERE id = $1",
})

// RPC attributes
span.SetAttributes(map[string]interface{}{
    "rpc.system":  "grpc",
    "rpc.service": "UserService",
    "rpc.method":  "GetUser",
})
```

## Error Handling

### Recording Errors

```go
result, err := doSomething()
if err != nil {
    // Record error on span
    span.RecordError(err)
    span.SetStatus(tracing.StatusCodeError, err.Error())
    return err
}

span.SetStatus(tracing.StatusCodeOK, "success")
```

### Custom Error Information

```go
if err != nil {
    span.RecordError(err)
    span.SetAttributes(map[string]interface{}{
        "error.type":    "ValidationError",
        "error.message": err.Error(),
        "error.code":    "INVALID_INPUT",
    })
    span.SetStatus(tracing.StatusCodeError, "validation failed")
}
```

## Context Propagation

### HTTP Requests

```go
import (
    "net/http"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// Client: Inject trace context into HTTP headers
func makeRequest(ctx context.Context, url string) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    
    // Inject trace context
    otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
    
    resp, err := http.DefaultClient.Do(req)
    // ...
}

// Server: Extract trace context from HTTP headers
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := otel.GetTextMapPropagator().Extract(r.Context(), 
        propagation.HeaderCarrier(r.Header))
    
    // Use ctx for tracing
    ctx, span := obs.Tracer().StartSpan(ctx, "handle-request")
    defer span.End()
    
    // ...
}
```

### gRPC

Context propagation is automatic with gRPC interceptors (coming in Phase 2).

## Best Practices

### 1. Span Naming

Use descriptive, hierarchical names:

```go
// Good
"UserService.GetUser"
"database.query.users"
"http.GET./api/users"

// Bad
"span1"
"process"
"function"
```

### 2. Span Granularity

Create spans for:
- ✅ HTTP/gRPC requests
- ✅ Database queries
- ✅ External API calls
- ✅ Significant business logic

Avoid creating spans for:
- ❌ Every function call
- ❌ Simple calculations
- ❌ Logging operations

### 3. Attribute Cardinality

Keep attribute cardinality low:

```go
// Good (low cardinality)
span.SetAttribute("http.method", "GET")
span.SetAttribute("http.status_code", 200)

// Bad (high cardinality)
span.SetAttribute("user_id", "user123")      // Too many unique values
span.SetAttribute("timestamp", time.Now())   // Unique every time
```

### 4. Error Handling

Always record errors:

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(tracing.StatusCodeError, err.Error())
}
```

### 5. Defer span.End()

Always defer `span.End()` immediately after creating a span:

```go
ctx, span := obs.Tracer().StartSpan(ctx, "operation")
defer span.End() // Ensures span is ended even if panic occurs
```

## Example: HTTP Handler

```go
func handleUserRequest(w http.ResponseWriter, r *http.Request) {
    // Extract trace context from headers
    ctx := otel.GetTextMapPropagator().Extract(r.Context(),
        propagation.HeaderCarrier(r.Header))
    
    // Start span
    ctx, span := obs.Tracer().StartSpan(ctx, "handle-user-request",
        tracing.WithSpanKind(tracing.SpanKindServer),
        tracing.WithAttributes(map[string]interface{}{
            "http.method": r.Method,
            "http.url":    r.URL.Path,
        }),
    )
    defer span.End()
    
    // Get user ID from request
    userID := r.URL.Query().Get("user_id")
    span.SetAttribute("user_id", userID)
    
    // Fetch user from database
    user, err := fetchUser(ctx, userID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(tracing.StatusCodeError, "failed to fetch user")
        http.Error(w, "Internal Server Error", 500)
        return
    }
    
    // Return response
    span.SetAttribute("http.status_code", 200)
    span.SetStatus(tracing.StatusCodeOK, "success")
    json.NewEncoder(w).Encode(user)
}

func fetchUser(ctx context.Context, userID string) (*User, error) {
    // Create child span for database query
    ctx, span := obs.Tracer().StartSpan(ctx, "database.query.users",
        tracing.WithSpanKind(tracing.SpanKindClient),
    )
    defer span.End()
    
    span.SetAttributes(map[string]interface{}{
        "db.system":    "postgresql",
        "db.statement": "SELECT * FROM users WHERE id = $1",
        "db.name":      "mydb",
    })
    
    // Execute query...
    user, err := db.QueryUser(ctx, userID)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(tracing.StatusCodeError, "query failed")
        return nil, err
    }
    
    span.SetStatus(tracing.StatusCodeOK, "success")
    return user, nil
}
```

## Viewing Traces

### Jaeger

1. Start Jaeger:
   ```bash
   docker run -d --name jaeger \
     -p 4317:4317 \
     -p 16686:16686 \
     jaegertracing/all-in-one:latest
   ```

2. View traces: http://localhost:16686

### Zipkin

1. Start Zipkin:
   ```bash
   docker run -d --name zipkin \
     -p 9411:9411 \
     openzipkin/zipkin
   ```

2. View traces: http://localhost:9411

## Troubleshooting

### Traces not appearing

1. **Check collector is running**:
   ```bash
   curl http://localhost:4317
   ```

2. **Verify endpoint configuration**:
   ```go
   TracingEndpoint: "localhost:4317" // Correct
   TracingEndpoint: "http://localhost:4317" // Wrong (no http://)
   ```

3. **Check sample rate**:
   ```go
   TracingSampleRate: 1.0 // 100% for testing
   ```

4. **Check logs for errors**:
   ```go
   obs.Logger().Error(ctx, "Tracing error", "error", err)
   ```

### High overhead

1. **Reduce sample rate**:
   ```go
   TracingSampleRate: 0.1 // 10% instead of 100%
   ```

2. **Reduce span granularity**: Create fewer spans

3. **Use batch exporter**: Already configured by default

## Next Steps

- **Phase 2**: HTTP/gRPC middleware for automatic tracing
- **Phase 3**: Service mesh integration (Istio, Linkerd)
- **Phase 4**: Advanced features (baggage, events, links)

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Jaeger](https://www.jaegertracing.io/)
- [Zipkin](https://zipkin.io/)


## Troubleshooting

### OTLP Export Issues

#### Metrics not appearing

1. **Check collector is running**:
   ```bash
   curl http://localhost:4317
   ```

2. **Verify configuration**:
   ```go
   OTLPEndpoint: "localhost:4317"  // Correct
   OTLPEndpoint: "http://localhost:4317"  // Wrong (no http://)
   ```

3. **Check UseOTelMetrics is enabled**:
   ```go
   UseOTelMetrics: true
   ```

4. **Check collector logs** for export errors

5. **Verify collector configuration** accepts metrics on OTLP receiver

#### Logs not appearing

1. **Check UseOTelLogs is enabled**:
   ```go
   UseOTelLogs: true
   ```

2. **Verify OTLP endpoint** is correct

3. **Check collector configuration** accepts logs on OTLP receiver

4. **Check collector logs** for export errors

#### Traces not appearing

1. **Check EnableTracing is enabled**:
   ```go
   EnableTracing: true
   ```

2. **Verify sample rate** is not too low:
   ```go
   TracingSampleRate: 1.0  // 100% for testing
   ```

3. **Check collector configuration** accepts traces on OTLP receiver

4. **Verify spans are being created**:
   ```go
   ctx, span := obs.Tracer().StartSpan(ctx, "test")
   defer span.End()
   ```

### Trace-Log Correlation Issues

#### Logs missing trace_id

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

4. **Check UseOTelLogs is enabled**:
   ```go
   UseOTelLogs: true
   ```

### Performance Issues

#### High CPU usage

1. **Reduce sample rate** for tracing:
   ```go
   TracingSampleRate: 0.1  // 10% instead of 100%
   ```

2. **Increase export interval**:
   ```go
   OTLPExportInterval: 30 * time.Second  // Export less frequently
   ```

3. **Increase batch size**:
   ```go
   OTLPBatchSize: 1024  // Larger batches
   ```

4. **Disable dual export** if not needed:
   ```go
   PrometheusEnabled: false  // OTLP only
   ```

#### High memory usage

1. **Reduce batch size**:
   ```go
   OTLPBatchSize: 256  // Smaller batches
   ```

2. **Reduce export interval**:
   ```go
   OTLPExportInterval: 5 * time.Second  // Export more frequently
   ```

3. **Check for span leaks** (spans not ended):
   ```go
   ctx, span := obs.Tracer().StartSpan(ctx, "operation")
   defer span.End()  // Always defer span.End()
   ```

### Connection Issues

#### Connection refused

1. **Check collector is running**:
   ```bash
   docker ps | grep otel-collector
   ```

2. **Verify endpoint is reachable**:
   ```bash
   telnet localhost 4317
   ```

3. **Check firewall rules** allow connections to collector

4. **Verify collector is listening** on correct port:
   ```bash
   netstat -an | grep 4317
   ```

#### TLS/SSL errors

1. **Use insecure connection** for local development:
   ```go
   OTLPProtocol: "grpc"  // Uses insecure connection by default
   ```

2. **For production**, configure TLS properly in collector

#### Timeout errors

1. **Increase timeout**:
   ```go
   OTLPTimeout: 30 * time.Second
   ```

2. **Check network latency** to collector

3. **Verify collector is not overloaded**

### Collector Configuration

#### Example OTLP Collector Config

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 512

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [loki]
```

## Best Practices

### 1. Use Unified Endpoint

Use a single OTLP endpoint for all signals:

```go
OTLPEndpoint: "localhost:4317"  // Good
```

Instead of:
```go
OTLPMetricsEndpoint: "localhost:4317"  // Avoid
OTLPLogsEndpoint:    "localhost:4317"
TracingEndpoint:     "localhost:4317"
```

### 2. Enable Trace-Log Correlation

Always pass context to logger for automatic trace correlation:

```go
// Good
ctx, span := obs.Tracer().StartSpan(ctx, "operation")
defer span.End()
obs.Logger().Info(ctx, "message")

// Bad
obs.Logger().Info(context.Background(), "message")
```

### 3. Use Dual Export During Migration

Keep Prometheus while migrating to OpenTelemetry:

```go
UseOTelMetrics:    true,   // New OTLP export
PrometheusEnabled: true,   // Keep existing Prometheus
```

### 4. Tune Export Settings

Adjust batch size and interval based on your workload:

```go
// High throughput
OTLPBatchSize:      1024,
OTLPExportInterval: 5 * time.Second,

// Low throughput
OTLPBatchSize:      256,
OTLPExportInterval: 30 * time.Second,
```

### 5. Use Appropriate Sample Rates

Adjust tracing sample rate based on environment:

```go
// Development
TracingSampleRate: 1.0  // 100%

// Staging
TracingSampleRate: 0.5  // 50%

// Production
TracingSampleRate: 0.1  // 10%
```

### 6. Always Defer span.End()

Ensure spans are always ended:

```go
ctx, span := obs.Tracer().StartSpan(ctx, "operation")
defer span.End()  // Always defer immediately
```

### 7. Add Meaningful Attributes

Add attributes that help with debugging:

```go
span.SetAttributes(map[string]interface{}{
    "user_id":      userID,
    "request_size": len(data),
    "cache_hit":    cacheHit,
})
```

### 8. Record Errors Properly

Always record errors in spans and logs:

```go
if err != nil {
    span.RecordError(err)
    span.SetStatus(tracing.StatusCodeError, err.Error())
    obs.Logger().Error(ctx, "Operation failed", "error", err)
}
```

## Deployment

### Docker Compose

See `deploy/docker/docker-compose.observability.yml` for complete setup:

```bash
# Start observability stack
make observability-up

# Check status
make observability-status

# View logs
make observability-logs
```

### Kubernetes

See `deploy/k8s/observability/` for Kubernetes manifests:

```bash
# Deploy observability stack
kubectl apply -f deploy/k8s/observability/

# Check status
kubectl get pods -n observability

# View collector logs
kubectl logs -n observability -l app=otel-collector
```

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [Deployment Guide](../deploy/docker/OBSERVABILITY.md)
