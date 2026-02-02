# Health Check Integration - IM Gateway Service

## Overview

This document describes the integration of the standardized health check library into the im-gateway-service. The im-gateway-service is a WebSocket gateway that manages real-time connections between clients and the IM backend services.

## Service Architecture

The im-gateway-service has the following key dependencies:
- **Redis**: For caching and session management (critical)
- **IM Service**: Downstream service for message routing (critical)
- **WebSocket Connections**: Active client connections (non-critical)

## Health Checks Registered

### 1. Redis Health Check
- **Type**: Built-in
- **Critical**: Yes
- **Timeout**: 100ms
- **Interval**: 5s
- **Purpose**: Verifies Redis connectivity for caching and session management

### 2. IM Service Health Check
- **Type**: Built-in HTTP check
- **Critical**: Yes
- **Timeout**: 100ms
- **Interval**: 5s
- **Purpose**: Verifies downstream im-service is available for message routing
- **Endpoint**: `http://{im-service-addr}/healthz`

### 3. WebSocket Connection Health Check
- **Type**: Custom
- **Critical**: No
- **Timeout**: 100ms
- **Interval**: 5s
- **Purpose**: Monitors WebSocket connection health and statistics
- **Implementation**: `health_checks.go`

## Custom Health Check Implementation

### WebSocketHealthCheck

The `WebSocketHealthCheck` monitors the health of WebSocket connections:

```go
type WebSocketHealthCheck struct {
    gateway  *service.GatewayService
    timeout  time.Duration
    interval time.Duration
    critical bool
}
```

**Check Logic**:
- Retrieves connection statistics from the gateway
- Verifies the gateway is operational
- Non-critical: Service can start without active connections
- Future enhancements could check for connection errors or pool exhaustion

### GatewayService.GetConnectionStats()

Added method to expose connection statistics:

```go
func (g *GatewayService) GetConnectionStats() ConnectionStats {
    var totalConnections int64
    var activeDevices int64
    
    g.connections.Range(func(key, value any) bool {
        totalConnections++
        activeDevices++
        return true
    })
    
    return ConnectionStats{
        TotalConnections: totalConnections,
        ActiveDevices:    activeDevices,
        ErrorCount:       0,
    }
}
```

## HTTP Endpoints

### Health Endpoints (No Middleware)
- `GET /healthz` → Liveness probe (200/503)
- `GET /readyz` → Readiness probe (200/503)
- `GET /health` → Detailed health status (JSON)

### WebSocket Endpoint (With Readiness Middleware)
- `GET /ws` → WebSocket upgrade endpoint
  - Protected by readiness middleware
  - Rejects connections with 503 when service is not ready
  - Ensures clients don't connect to unhealthy gateway instances

## Integration Details

### main.go Changes

1. **Import health library**:
```go
import "github.com/pingxin403/cuckoo/libs/health"
```

2. **Initialize HealthChecker**:
```go
healthChecker := health.NewHealthChecker(health.Config{
    ServiceName:      cfg.Observability.ServiceName,
    CheckInterval:    5 * time.Second,
    DefaultTimeout:   100 * time.Millisecond,
    FailureThreshold: 3,
}, obs)
```

3. **Register health checks**:
```go
// Redis check
healthChecker.RegisterCheck(health.NewRedisCheck("redis", redisClient))

// IM Service check
imServiceHealthURL := fmt.Sprintf("http://%s/healthz", cfg.ServiceDiscovery.IMServiceAddr)
healthChecker.RegisterCheck(health.NewHTTPCheck("im-service", imServiceHealthURL, true))

// WebSocket connection check
healthChecker.RegisterCheck(NewWebSocketHealthCheck(gateway))
```

4. **Start health checker**:
```go
if err := healthChecker.Start(); err != nil {
    log.Fatalf("Failed to start health checker: %v", err)
}
defer healthChecker.Stop()
```

5. **Setup HTTP endpoints with middleware**:
```go
mux := http.NewServeMux()

// Health endpoints (no middleware)
mux.HandleFunc("/healthz", health.HealthzHandler(healthChecker))
mux.HandleFunc("/readyz", health.ReadyzHandler(healthChecker))
mux.HandleFunc("/health", health.HealthHandler(healthChecker))

// WebSocket endpoint with readiness middleware
wsHandler := health.ReadinessMiddleware(healthChecker)(http.HandlerFunc(gateway.HandleWebSocket))
mux.Handle("/ws", wsHandler)
```

## Kubernetes Configuration

### Liveness Probe
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

**Purpose**: Detects if the gateway process is alive and not deadlocked.

### Readiness Probe
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

**Purpose**: Determines if the gateway is ready to accept WebSocket connections.

### Startup Probe
```yaml
startupProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 0
  periodSeconds: 2
  timeoutSeconds: 1
  failureThreshold: 30  # 60 seconds total
```

**Purpose**: Allows up to 60 seconds for the gateway to start before liveness checks begin.

## Behavior

### Normal Operation
1. Gateway starts and initializes dependencies
2. Health checker starts and begins checking Redis, IM Service, and WebSocket health
3. Once all critical checks pass, readiness probe returns 200
4. Kubernetes adds pod to service endpoints
5. WebSocket connections are accepted

### Dependency Failure Scenarios

#### Redis Failure
- **Detection**: Within 5 seconds (check interval)
- **Response**: Readiness probe fails after 3 consecutive failures (anti-flapping)
- **Impact**: 
  - Kubernetes removes pod from service endpoints
  - New WebSocket connections rejected with 503
  - Existing connections remain active
- **Recovery**: Automatic when Redis becomes available

#### IM Service Failure
- **Detection**: Within 5 seconds
- **Response**: Readiness probe fails after 3 consecutive failures
- **Impact**: Same as Redis failure
- **Recovery**: Automatic when IM Service becomes available

#### WebSocket Health Issues
- **Detection**: Within 5 seconds
- **Response**: Logged but does not affect readiness (non-critical)
- **Impact**: None on traffic routing
- **Purpose**: Monitoring and alerting only

### Graceful Shutdown
1. SIGTERM received
2. Health checker marks service as not ready
3. Readiness probe fails immediately
4. Kubernetes removes pod from service endpoints
5. New connections rejected
6. Existing connections allowed to complete (30s timeout)
7. Gateway shuts down

## Testing

### Unit Tests
- `TestHealthEndpoints`: Verifies all health endpoints work correctly
- `TestReadinessMiddleware`: Tests middleware behavior when ready/not ready
- `TestWebSocketHealthCheck`: Tests custom WebSocket health check
- `TestGatewayGetConnectionStats`: Verifies connection stats method

### Integration Tests
- `TestHealthChecksWithDependencies`: Tests with real Redis and mock IM Service
- Requires Redis running on localhost:6379
- Run with: `go test -v ./...`
- Skip with: `go test -short ./...`

### Manual Testing

1. **Start the service**:
```bash
cd apps/im-gateway-service
go run .
```

2. **Check health endpoints**:
```bash
# Liveness
curl http://localhost:8080/healthz

# Readiness
curl http://localhost:8080/readyz

# Detailed health
curl http://localhost:8080/health | jq
```

3. **Test with Redis down**:
```bash
# Stop Redis
docker stop redis

# Check readiness (should fail after 3 checks)
curl http://localhost:8080/readyz

# Try WebSocket connection (should be rejected)
wscat -c ws://localhost:8080/ws?token=test
```

4. **Test recovery**:
```bash
# Start Redis
docker start redis

# Check readiness (should recover immediately)
curl http://localhost:8080/readyz
```

## Metrics

The health checker automatically exports the following Prometheus metrics:

- `health_status{service="im-gateway-service"}` - Overall health status (0=critical, 1=degraded, 2=healthy)
- `health_score{service="im-gateway-service"}` - Health score (0.0 to 1.0)
- `component_status{service="im-gateway-service",component="redis"}` - Per-component status
- `component_status{service="im-gateway-service",component="im-service"}` - IM Service status
- `component_status{service="im-gateway-service",component="websocket-connections"}` - WebSocket health
- `component_response_time_seconds{service="im-gateway-service",component="*"}` - Response time histogram
- `health_check_failures_total{service="im-gateway-service",component="*"}` - Failure counter

## Files Modified

### Core Integration
- `apps/im-gateway-service/go.mod` - Added health library dependency
- `apps/im-gateway-service/main.go` - Integrated health checker
- `apps/im-gateway-service/service/gateway_service.go` - Added GetConnectionStats()

### New Files
- `apps/im-gateway-service/health_checks.go` - Custom WebSocket health check
- `apps/im-gateway-service/health_integration_test.go` - Integration tests
- `apps/im-gateway-service/HEALTH_CHECK_INTEGRATION.md` - This document

### Kubernetes
- `deploy/k8s/services/im-gateway-service/im-gateway-service-deployment.yaml` - Updated probes

## Comparison with Other Services

| Service | Dependencies | Custom Checks | Complexity |
|---------|-------------|---------------|------------|
| shortener-service | 2 | 0 | Low |
| todo-service | 1 | 0 | Low |
| auth-service | 2 | 0 | Low |
| user-service | 2 | 0 | Low |
| im-service | 5 | 2 | Very High |
| **im-gateway-service** | **3** | **1** | **Medium** |

### Why im-gateway-service is Medium Complexity

1. **WebSocket-Specific**: Requires custom health check for connection monitoring
2. **Downstream Dependency**: HTTP health check to im-service
3. **Readiness Middleware**: Critical for WebSocket endpoint to reject unhealthy connections
4. **Connection Management**: GetConnectionStats() method for monitoring

## Best Practices Applied

1. ✅ **Separate Liveness and Readiness**: Liveness checks process health, readiness checks dependencies
2. ✅ **Anti-Flapping**: 3 consecutive failures required before marking not ready
3. ✅ **Fast Checks**: All checks complete within 100-200ms
4. ✅ **Graceful Degradation**: Non-critical checks don't affect readiness
5. ✅ **Observability**: Full metrics and logging integration
6. ✅ **Kubernetes-Native**: Proper probe configuration with startup probe
7. ✅ **Middleware Protection**: WebSocket endpoint protected by readiness check

## Future Enhancements

1. **Connection Pool Monitoring**: Check for connection pool exhaustion
2. **Error Rate Tracking**: Monitor WebSocket connection error rates
3. **Latency Monitoring**: Track WebSocket message latency
4. **Circuit Breaker**: Add circuit breaker for downstream im-service calls
5. **Auto-Recovery**: Implement reconnection logic for failed dependencies

## References

- Health Library: `libs/health/README.md`
- Design Spec: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
- IM Service Integration: `apps/im-service/HEALTH_CHECK_INTEGRATION.md`
