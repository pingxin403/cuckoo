# Health Check Library

A production-ready health check library for Go services with standardized liveness and readiness probes following Kubernetes best practices.

## Features

- ✅ **Liveness Probes**: Detect process deadlocks, memory leaks, and goroutine leaks
- ✅ **Readiness Probes**: Verify dependency health (database, Redis, Kafka, HTTP, gRPC)
- ✅ **Built-in Health Checks**: Common dependencies covered out-of-the-box
- ✅ **Circuit Breaker**: Prevent cascading failures with automatic circuit breaking
- ✅ **Auto-Recovery**: Automatic reconnection with exponential backoff
- ✅ **Anti-Flapping**: Prevent rapid state changes (configurable failure threshold)
- ✅ **HTTP Middleware**: Automatic traffic rejection when not ready
- ✅ **Observability**: Full metrics, logging, and tracing integration
- ✅ **High Performance**: Lock-free operations, < 1ms status checks
- ✅ **Thread-Safe**: Safe for concurrent access from multiple goroutines

## Installation

```bash
go get github.com/pingxin403/cuckoo/libs/health
```

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "net/http"
    "time"

    "github.com/pingxin403/cuckoo/libs/health"
    "github.com/pingxin403/cuckoo/libs/observability"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Initialize observability
    obs, _ := observability.New(observability.Config{
        ServiceName: "my-service",
    })
    defer obs.Shutdown(context.Background())

    // Create health checker
    hc := health.NewHealthChecker(health.Config{
        ServiceName:      "my-service",
        CheckInterval:    5 * time.Second,
        DefaultTimeout:   100 * time.Millisecond,
        FailureThreshold: 3,
    }, obs)

    // Register health checks
    db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/db")
    hc.RegisterCheck(health.NewDatabaseCheck("database", db))

    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))

    // Start health checking
    hc.Start()
    defer hc.Stop()

    // Setup HTTP endpoints
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", health.HealthzHandler(hc))
    mux.HandleFunc("/readyz", health.ReadyzHandler(hc))
    mux.HandleFunc("/health", health.HealthHandler(hc))

    // Wrap with readiness middleware
    handler := health.ReadinessMiddleware(hc)(mux)

    http.ListenAndServe(":8080", handler)
}
```

## Liveness vs Readiness

### Liveness Probe (`/healthz`)

Checks if the **process is alive** and not deadlocked. Does NOT check external dependencies.

- ✅ Heartbeat mechanism (detects goroutine deadlocks)
- ✅ Memory usage monitoring
- ✅ Goroutine count monitoring

**Kubernetes Configuration:**

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 15
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe (`/readyz`)

Checks if the service is **ready to handle traffic**. Verifies all critical dependencies.

- ✅ Database connectivity
- ✅ Redis connectivity
- ✅ Kafka connectivity
- ✅ Downstream service health
- ✅ Custom application checks

**Kubernetes Configuration:**

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 1
  failureThreshold: 1
```

## Built-in Health Checks

### Database Check

```go
db, _ := sql.Open("mysql", dsn)
hc.RegisterCheck(health.NewDatabaseCheck("database", db))
```

### Redis Check

```go
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))
```

### Kafka Check

```go
hc.RegisterCheck(health.NewKafkaCheck("kafka", []string{"localhost:9092"}))
```

### HTTP Service Check

```go
hc.RegisterCheck(health.NewHTTPCheck(
    "auth-service",
    "http://auth-service:8080/healthz",
    true, // critical
))
```

### gRPC Service Check

```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
hc.RegisterCheck(health.NewGRPCCheck("grpc-service", conn, true))
```

## Custom Health Checks

Implement the `Check` interface:

```go
type MyCheck struct{}

func (c *MyCheck) Name() string {
    return "my-custom-check"
}

func (c *MyCheck) Check(ctx context.Context) error {
    // Perform your health check logic
    if somethingWrong {
        return fmt.Errorf("something is wrong")
    }
    return nil
}

func (c *MyCheck) Timeout() time.Duration {
    return 100 * time.Millisecond
}

func (c *MyCheck) Interval() time.Duration {
    return 5 * time.Second
}

func (c *MyCheck) Critical() bool {
    return true // If true, failure marks service as not ready
}

// Register the check
hc.RegisterCheck(&MyCheck{})
```

## Circuit Breaker

Prevent cascading failures with automatic circuit breaking:

```go
check := health.NewHTTPCheckWithCircuitBreaker(
    "auth-service",
    "http://auth-service:8080/healthz",
    false, // non-critical
)
hc.RegisterCheck(check)
```

Circuit breaker states:
- **Closed**: Normal operation, requests pass through
- **Open**: Too many failures, requests fail fast
- **Half-Open**: Testing recovery, limited requests allowed

## Auto-Recovery

Automatically reconnect to failed dependencies:

```go
db, _ := sql.Open("mysql", dsn)
dbCheck := health.NewDatabaseCheck("database", db)
dbRecoverer := health.NewDatabaseRecoverer(dsn, &db)

hc.RegisterCheck(dbCheck)
hc.RegisterRecoverer("database", dbRecoverer)
```

Recovery uses exponential backoff:
- Initial: 1s
- Max: 30s
- Factor: 2.0

## HTTP Middleware

Automatically reject requests when service is not ready:

```go
handler := health.ReadinessMiddleware(hc)(yourHandler)
http.ListenAndServe(":8080", handler)
```

When not ready, returns:
- Status: `503 Service Unavailable`
- Body: `Service not ready`

## Health Endpoints

### `/healthz` - Liveness

Returns `200 OK` if process is alive, `503` otherwise.

```bash
curl http://localhost:8080/healthz
# Response: OK
```

### `/readyz` - Readiness

Returns `200 OK` if ready to serve traffic, `503` otherwise.

```bash
curl http://localhost:8080/readyz
# Response: READY
```

### `/health` - Full Status

Returns detailed health status in JSON format.

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "service": "my-service",
  "timestamp": "2024-01-15T10:30:00Z",
  "score": 0.85,
  "summary": "All systems operational (3/3 healthy)",
  "components": {
    "database": {
      "name": "database",
      "status": "healthy",
      "last_check": "2024-01-15T10:29:58Z",
      "response_time_ms": 15,
      "error": ""
    },
    "redis": {
      "name": "redis",
      "status": "healthy",
      "last_check": "2024-01-15T10:29:59Z",
      "response_time_ms": 5,
      "error": ""
    },
    "kafka": {
      "name": "kafka",
      "status": "degraded",
      "last_check": "2024-01-15T10:29:57Z",
      "response_time_ms": 150,
      "error": "slow response"
    }
  }
}
```

## Configuration

### Basic Configuration

```go
config := health.Config{
    ServiceName:      "my-service",
    CheckInterval:    5 * time.Second,   // How often to run checks
    DefaultTimeout:   100 * time.Millisecond, // Default check timeout
    HealthyScore:     0.8,                // Score threshold for healthy
    DegradedScore:    0.5,                // Score threshold for degraded
    FailureThreshold: 3,                  // Consecutive failures before not ready
}
```

### Environment Variables

```bash
# Health check configuration
HEALTH_CHECK_INTERVAL=5s
HEALTH_CHECK_TIMEOUT=100ms
HEALTH_FAILURE_THRESHOLD=3

# Liveness probe
LIVENESS_HEARTBEAT_INTERVAL=1s
LIVENESS_HEARTBEAT_TIMEOUT=10s
LIVENESS_MEMORY_LIMIT=4294967296  # 4GB
LIVENESS_GOROUTINE_LIMIT=10000

# Circuit breaker
CIRCUIT_BREAKER_MAX_FAILURES=3
CIRCUIT_BREAKER_TIMEOUT=30s
CIRCUIT_BREAKER_HALF_OPEN_TIMEOUT=10s

# Auto-recovery
AUTO_RECOVERY_ENABLED=true
AUTO_RECOVERY_MAX_RETRIES=3
AUTO_RECOVERY_INITIAL_BACKOFF=1s
AUTO_RECOVERY_MAX_BACKOFF=30s
```

## Observability

### Metrics

The library exports Prometheus metrics:

- `health_status` - Overall health status (0=critical, 1=degraded, 2=healthy)
- `health_score` - Numerical health score (0.0 to 1.0)
- `component_status` - Status per component
- `component_response_time_seconds` - Response time histogram per component
- `health_check_failures_total` - Failure counter per component

### Logging

All health status changes are logged:

```
INFO  Component recovered component=database old_status=critical new_status=healthy
WARN  Component health changed component=redis old_status=healthy new_status=degraded
ERROR Component health critical component=kafka old_status=degraded new_status=critical
```

## Performance

The library is designed for minimal overhead:

| Metric | Target | Description |
|--------|--------|-------------|
| Health check execution | < 200ms | All checks combined (parallel) |
| Health status retrieval | < 1ms | Lock-free atomic operations |
| Middleware overhead | < 100μs | Single atomic load per request |
| Memory overhead | < 10MB | Per service instance |
| CPU overhead | < 1% | Background health checking |

## Anti-Flapping

Prevents rapid state changes that could cause pod restart storms:

- Requires **3 consecutive failures** before marking as not ready (configurable)
- Immediately marks as ready on **first success**
- Prevents unnecessary pod restarts in Kubernetes

## Thread Safety

All operations are thread-safe:
- ✅ Concurrent health check registration
- ✅ Concurrent status retrieval
- ✅ Lock-free readiness checks
- ✅ Safe for use in HTTP handlers

## Best Practices

### 1. Keep Checks Fast

Health checks should complete quickly (< 100ms each):

```go
// ✅ Good - fast ping check
func (c *DatabaseCheck) Check(ctx context.Context) error {
    return c.db.PingContext(ctx)
}

// ❌ Bad - slow query
func (c *DatabaseCheck) Check(ctx context.Context) error {
    _, err := c.db.QueryContext(ctx, "SELECT COUNT(*) FROM large_table")
    return err
}
```

### 2. Mark Critical Dependencies

Only mark dependencies as critical if they're required for operation:

```go
// Critical - service can't function without database
hc.RegisterCheck(health.NewDatabaseCheck("database", db)) // critical=true

// Non-critical - service can function with degraded features
hc.RegisterCheck(health.NewKafkaCheck("kafka", brokers)) // critical=false
```

### 3. Use Circuit Breakers for Downstream Services

Prevent cascading failures:

```go
check := health.NewHTTPCheckWithCircuitBreaker(
    "auth-service",
    "http://auth-service:8080/healthz",
    false, // non-critical
)
```

### 4. Configure Appropriate Timeouts

Set timeouts based on expected response times:

```go
// Fast local services
dbCheck := health.NewDatabaseCheck("database", db)
dbCheck.SetTimeout(50 * time.Millisecond)

// Slower remote services
httpCheck := health.NewHTTPCheck("external-api", url, false)
httpCheck.SetTimeout(200 * time.Millisecond)
```

### 5. Use Auto-Recovery for Transient Failures

Enable auto-recovery for dependencies that can reconnect:

```go
db, _ := sql.Open("mysql", dsn)
dbRecoverer := health.NewDatabaseRecoverer(dsn, &db)
hc.RegisterRecoverer("database", dbRecoverer)
```

## Troubleshooting

### Service Not Ready

Check the `/health` endpoint for detailed status:

```bash
curl http://localhost:8080/health | jq
```

Look for components with `status: "critical"` or `status: "degraded"`.

### False Positives

If health checks are too sensitive, adjust the failure threshold:

```go
config := health.Config{
    FailureThreshold: 5, // Require 5 consecutive failures
}
```

### Slow Health Checks

Check component response times in the `/health` endpoint:

```json
{
  "components": {
    "database": {
      "response_time_ms": 250  // Too slow!
    }
  }
}
```

Optimize slow checks or increase timeouts.

### Memory Leaks

Monitor the `health_check_failures_total` metric. Increasing failures may indicate:
- Network issues
- Dependency failures
- Configuration problems

## Migration Guide

### From Existing Health Checks

1. **Add dependency:**
   ```bash
   go get github.com/pingxin403/cuckoo/libs/health
   ```

2. **Replace old health check code:**
   ```go
   // Old
   http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
       w.WriteHeader(http.StatusOK)
       w.Write([]byte("OK"))
   })

   // New
   hc := health.NewHealthChecker(config, obs)
   hc.RegisterCheck(health.NewDatabaseCheck("database", db))
   hc.Start()
   http.HandleFunc("/healthz", health.HealthzHandler(hc))
   http.HandleFunc("/readyz", health.ReadyzHandler(hc))
   ```

3. **Update Kubernetes manifests:**
   ```yaml
   # Old
   livenessProbe:
     httpGet:
       path: /health
       port: 8080

   # New
   livenessProbe:
     httpGet:
       path: /healthz
       port: 8080
   readinessProbe:
     httpGet:
       path: /readyz
       port: 8080
   ```

## Examples

See the [examples](./examples) directory for complete examples:

- [Basic usage](./examples/basic)
- [With auto-recovery](./examples/auto-recovery)
- [Custom health checks](./examples/custom-checks)
- [Circuit breaker](./examples/circuit-breaker)
- [Kubernetes integration](./examples/kubernetes)

## Contributing

Contributions are welcome! Please read the [contributing guide](../../CONTRIBUTING.md) first.

## License

See [LICENSE](../../LICENSE) for details.

## References

- [Kubernetes Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
