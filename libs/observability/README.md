# Observability Library

A unified observability framework for all services in the monorepo, providing metrics, tracing, and logging capabilities with full OpenTelemetry integration.

## Overview

This library provides a standardized way to instrument Go services with:
- **Metrics**: OpenTelemetry Metrics SDK with dual export (OTLP + Prometheus)
- **Tracing**: OpenTelemetry distributed tracing with OTLP export
- **Logging**: OpenTelemetry Logs SDK with automatic trace correlation
- **Profiling**: Built-in pprof support for performance analysis

## Features

- 🎯 **Zero-boilerplate**: Automatic instrumentation for common patterns
- 🔌 **Pluggable**: Easy to integrate with existing services
- 📊 **Standardized**: Consistent metrics, traces, and logs across all services
- 🚀 **Performance**: Minimal overhead with efficient implementations
- 🧪 **Testable**: Mock implementations for unit testing
- 🔍 **Profiling**: Built-in pprof support for performance analysis
- 🌐 **OpenTelemetry**: Full OTel SDK integration for vendor-neutral observability
- 🔄 **Dual Export**: Support both OTLP and Prometheus simultaneously
- 🔗 **Trace Correlation**: Automatic trace_id and span_id injection into logs

## Quick Start

### 1. Initialize Observability

```go
import "github.com/pingxin403/cuckoo/libs/observability"

func main() {
    // Initialize with OpenTelemetry
    obs, err := observability.New(observability.Config{
        ServiceName:    "my-service",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        
        // OpenTelemetry configuration
        OTLPEndpoint:   "localhost:4317",  // Unified OTLP endpoint
        UseOTelMetrics: true,              // Use OTel Metrics SDK
        UseOTelLogs:    true,              // Use OTel Logs SDK
        EnableTracing:  true,              // Enable distributed tracing
        
        // Metrics
        EnableMetrics:     true,
        MetricsPort:       9090,
        PrometheusEnabled: true,  // Dual export: OTLP + Prometheus
        
        // Profiling
        EnablePprof: true,  // Enable pprof profiling endpoints
        
        // Logging
        LogLevel: "info",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer obs.Shutdown(context.Background())

    // Your service code here
}
```

### 2. Add Metrics

```go
// Increment a counter
obs.Metrics().IncrementCounter("requests_total", map[string]string{
    "method": "GET",
    "path":   "/api/users",
})

// Record a histogram
obs.Metrics().RecordHistogram("request_duration_seconds", 0.123, map[string]string{
    "method": "GET",
})

// Set a gauge
obs.Metrics().SetGauge("active_connections", 42)
```

### 3. Add Tracing

```go
// Start a span
ctx, span := obs.Tracer().StartSpan(ctx, "process_request")
defer span.End()

// Add attributes
span.SetAttribute("user_id", "12345")
span.SetAttribute("request_size", 1024)

// Record errors
if err != nil {
    span.RecordError(err)
}
```

### 4. Structured Logging

```go
// Log with context
obs.Logger().Info(ctx, "Processing request",
    "user_id", "12345",
    "method", "GET",
)

// Log errors
obs.Logger().Error(ctx, "Failed to process request",
    "error", err,
    "user_id", "12345",
)
```

## Architecture

```
libs/observability/
├── README.md
├── observability.go          # Main entry point
├── config.go                 # Configuration
├── metrics/
│   ├── metrics.go           # Metrics interface and implementation
│   ├── prometheus.go        # Prometheus implementation
│   └── mock.go              # Mock for testing
├── tracing/
│   ├── tracing.go           # Tracing interface
│   ├── otel.go              # OpenTelemetry implementation
│   └── mock.go              # Mock for testing
├── logging/
│   ├── logging.go           # Logging interface
│   ├── structured.go        # Structured logger implementation
│   └── mock.go              # Mock for testing
└── middleware/
    ├── http.go              # HTTP middleware
    ├── grpc.go              # gRPC interceptors
    └── websocket.go         # WebSocket middleware
```

## Standard Metrics

All services automatically get these standard metrics:

### HTTP Services
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - HTTP request latency
- `http_requests_in_flight` - Current in-flight requests
- `http_response_size_bytes` - HTTP response size

### gRPC Services
- `grpc_requests_total` - Total gRPC requests
- `grpc_request_duration_seconds` - gRPC request latency
- `grpc_requests_in_flight` - Current in-flight requests

### Common Metrics
- `process_cpu_seconds_total` - CPU usage
- `process_memory_bytes` - Memory usage
- `process_open_fds` - Open file descriptors
- `go_goroutines` - Number of goroutines

## Middleware Integration

### HTTP Server

```go
import "github.com/pingxin403/cuckoo/libs/observability/middleware"

mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)

// Wrap with observability middleware
handler := middleware.HTTPObservability(obs, mux)

http.ListenAndServe(":8080", handler)
```

### gRPC Server

```go
import "github.com/pingxin403/cuckoo/libs/observability/middleware"

server := grpc.NewServer(
    grpc.UnaryInterceptor(middleware.GRPCUnaryServerInterceptor(obs)),
    grpc.StreamInterceptor(middleware.GRPCStreamServerInterceptor(obs)),
)
```

### WebSocket

```go
import "github.com/pingxin403/cuckoo/libs/observability/middleware"

upgrader := websocket.Upgrader{}
handler := middleware.WebSocketObservability(obs, func(conn *websocket.Conn) {
    // Handle WebSocket connection
})
```

## Custom Metrics

Define service-specific metrics:

```go
// Define custom metrics
type MyServiceMetrics struct {
    obs observability.Observability
}

func (m *MyServiceMetrics) RecordMessageDelivery(success bool, latency time.Duration) {
    labels := map[string]string{
        "success": fmt.Sprintf("%v", success),
    }
    
    m.obs.Metrics().IncrementCounter("messages_delivered_total", labels)
    m.obs.Metrics().RecordHistogram("message_delivery_duration_seconds", 
        latency.Seconds(), labels)
}
```

## Performance Profiling with pprof

The observability library includes built-in support for Go's pprof profiling tool, allowing you to analyze CPU usage, memory allocation, goroutines, and blocking operations.

### Enabling pprof

```go
obs, err := observability.New(observability.Config{
    ServiceName: "my-service",
    EnableMetrics: true,
    MetricsPort: 9090,
    
    // Enable pprof endpoints
    EnablePprof: true,
    
    // Optional: Configure profiling rates
    PprofBlockProfileRate: 1000,      // Sample 1 in every 1000ns of blocking
    PprofMutexProfileFraction: 10,    // Sample 1 in every 10 mutex events
})
```

### Available pprof Endpoints

When pprof is enabled, the following endpoints are available on the metrics server:

- `/debug/pprof/` - Index page with all available profiles
- `/debug/pprof/profile` - CPU profile (30 seconds by default)
- `/debug/pprof/heap` - Memory allocation profile
- `/debug/pprof/goroutine` - Stack traces of all current goroutines
- `/debug/pprof/block` - Stack traces that led to blocking on synchronization primitives
- `/debug/pprof/mutex` - Stack traces of holders of contended mutexes
- `/debug/pprof/allocs` - All past memory allocations
- `/debug/pprof/threadcreate` - Stack traces that led to creation of new OS threads
- `/debug/pprof/trace` - Execution trace (use `?seconds=5` to specify duration)

### Using pprof

**Analyze CPU profile:**
```bash
# Collect 30-second CPU profile
go tool pprof http://localhost:9090/debug/pprof/profile

# Collect 60-second CPU profile
go tool pprof http://localhost:9090/debug/pprof/profile?seconds=60
```

**Analyze memory profile:**
```bash
go tool pprof http://localhost:9090/debug/pprof/heap
```

**View goroutines:**
```bash
go tool pprof http://localhost:9090/debug/pprof/goroutine
```

**Analyze blocking operations:**
```bash
go tool pprof http://localhost:9090/debug/pprof/block
```

**Analyze mutex contention:**
```bash
go tool pprof http://localhost:9090/debug/pprof/mutex
```

### pprof Configuration

| Config Field | Type | Default | Description |
|-------------|------|---------|-------------|
| `EnablePprof` | bool | false | Enable/disable pprof endpoints (opt-in for security) |
| `PprofBlockProfileRate` | int | 0 | Block profile sampling rate in nanoseconds (0 = disabled) |
| `PprofMutexProfileFraction` | int | 0 | Mutex profile sampling fraction (0 = disabled, 1 = all events) |

**Environment Variables:**
```bash
PPROF_BLOCK_PROFILE_RATE=1000
PPROF_MUTEX_PROFILE_FRACTION=10
```

### Security Considerations

- pprof is **disabled by default** for security reasons
- Only enable pprof in development or controlled production environments
- Consider using authentication/authorization if exposing pprof in production
- pprof endpoints can reveal sensitive information about your application's internals

## Configuration

### Environment Variables

```bash
# Service identification
SERVICE_NAME=my-service
SERVICE_VERSION=1.0.0
DEPLOYMENT_ENVIRONMENT=production

# OpenTelemetry
OTLP_ENDPOINT=localhost:4317              # Unified OTLP endpoint
OTLP_METRICS_ENDPOINT=localhost:4317      # Override for metrics
OTLP_LOGS_ENDPOINT=localhost:4317         # Override for logs
OTLP_PROTOCOL=grpc                        # grpc or http
USE_OTEL_METRICS=true                     # Use OTel Metrics SDK
USE_OTEL_LOGS=true                        # Use OTel Logs SDK
PROMETHEUS_ENABLED=true                   # Enable Prometheus exporter

# Metrics
METRICS_ENABLED=true
METRICS_PORT=9090
METRICS_PATH=/metrics

# Tracing
TRACING_ENABLED=true
TRACING_ENDPOINT=localhost:4317
TRACING_SAMPLE_RATE=0.1

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# pprof Profiling
ENABLE_PPROF=true
PPROF_BLOCK_PROFILE_RATE=1000
PPROF_MUTEX_PROFILE_FRACTION=10
```

### Code Configuration

```go
config := observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    
    // OpenTelemetry
    OTLPEndpoint:        "localhost:4317",  // Unified endpoint
    OTLPMetricsEndpoint: "",                // Optional override
    OTLPLogsEndpoint:    "",                // Optional override
    OTLPProtocol:        "grpc",            // grpc or http
    UseOTelMetrics:      true,              // Use OTel Metrics SDK
    UseOTelLogs:         true,              // Use OTel Logs SDK
    PrometheusEnabled:   true,              // Dual export
    
    // Metrics
    EnableMetrics:  true,
    MetricsPort:    9090,
    MetricsPath:    "/metrics",
    
    // Tracing
    EnableTracing:     true,
    TracingEndpoint:   "localhost:4317",
    TracingSampleRate: 0.1,
    
    // Logging
    LogLevel:  "info",
    LogFormat: "json",
    
    // pprof Profiling
    EnablePprof:               true,
    PprofBlockProfileRate:     1000,
    PprofMutexProfileFraction: 10,
    
    // Custom resource attributes
    ResourceAttributes: map[string]string{
        "team": "platform",
        "region": "us-west-2",
    },
}
```

## Testing

Use mock implementations for unit testing:

```go
import "github.com/pingxin403/cuckoo/libs/observability/metrics"

func TestMyService(t *testing.T) {
    // Create mock metrics
    mockMetrics := metrics.NewMock()
    
    // Your test code
    service := NewMyService(mockMetrics)
    service.DoSomething()
    
    // Verify metrics were recorded
    assert.Equal(t, 1, mockMetrics.GetCounter("requests_total"))
}
```

## Migration Guide

### From Custom Metrics Package

**Before:**
```go
m := metrics.NewMetrics()
m.IncrementActiveConnections()
http.HandleFunc("/metrics", m.Handler())
```

**After:**
```go
obs, _ := observability.New(observability.Config{
    ServiceName: "my-service",
})
obs.Metrics().IncrementCounter("active_connections", nil)
// Metrics endpoint automatically exposed
```

### From Manual Instrumentation

**Before:**
```go
start := time.Now()
// ... do work ...
duration := time.Since(start)
log.Printf("Request took %v", duration)
```

**After:**
```go
ctx, span := obs.Tracer().StartSpan(ctx, "do_work")
defer span.End()
// ... do work ...
// Duration automatically recorded
```

## Best Practices

1. **Initialize once**: Create observability instance at service startup
2. **Use context**: Always pass context for trace propagation
3. **Label cardinality**: Keep metric labels low-cardinality (< 100 unique values)
4. **Span naming**: Use consistent, descriptive span names
5. **Error handling**: Always record errors in spans
6. **Structured logging**: Use key-value pairs instead of string formatting

## Performance

- Metrics: < 1μs per operation
- Tracing: < 10μs per span (when sampled)
- Logging: < 5μs per log line
- Memory: ~10MB baseline + ~100 bytes per active span

## Examples

See `examples/` directory for complete examples:
- `examples/http-service/` - HTTP service with observability
- `examples/grpc-service/` - gRPC service with observability
- `examples/websocket-service/` - WebSocket service with observability
- `examples/worker-service/` - Background worker with observability

## Roadmap

- [ ] OpenTelemetry tracing integration
- [ ] Jaeger exporter
- [ ] Prometheus remote write
- [ ] Log aggregation (ELK/Loki)
- [ ] Custom metric types (Summary, etc.)
- [ ] Distributed context propagation
- [ ] Service mesh integration

## Contributing

When adding new features to the observability library:
1. Add interface definition
2. Implement for production use
3. Implement mock for testing
4. Add unit tests
5. Update documentation
6. Add example usage

## License

Internal use only - Part of Cuckoo monorepo
