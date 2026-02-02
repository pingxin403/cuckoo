# Health Check Integration - Auth Service

## Overview
This document describes the health check integration completed for the auth-service as part of the health check standardization initiative.

## Changes Made

### 1. Dependencies Added
- Added `github.com/pingxin403/cuckoo/libs/health` to go.mod
- Added `github.com/stretchr/testify` for testing
- Added replace directive for local development
- Ran `go mod tidy` to update dependencies

### 2. Code Changes

#### main.go
- Imported the health library
- Initialized HealthChecker with appropriate configuration:
  - ServiceName: from config or "auth-service"
  - CheckInterval: 5 seconds
  - DefaultTimeout: 100 milliseconds
  - FailureThreshold: 3 consecutive failures
- Added HTTP server on port 8080 for health endpoints (alongside gRPC on port 9095)
- Started health checker before service startup
- Added health endpoints to HTTP server:
  - `/healthz` - Liveness probe
  - `/readyz` - Readiness probe
  - `/health` - Detailed health status (JSON)
- Added health checker shutdown in graceful shutdown sequence

**Note**: auth-service is currently stateless with no database or Redis dependencies. The service only validates JWT tokens. If dependencies are added in the future, health checks can be registered using:
```go
hc.RegisterCheck(health.NewDatabaseCheck("database", db))
hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))
```

### 3. Kubernetes Manifests Updated

#### deploy/k8s/services/auth-service/auth-service-deployment.yaml
Updated probe configuration to use HTTP health endpoints instead of gRPC probes:

**Ports Added:**
- HTTP port 8080 for health endpoints
- Metrics port 9090 for observability

**Environment Variables Added:**
- `HTTP_PORT=8080` - HTTP server port for health endpoints
- `METRICS_PORT=9090` - Metrics port

**Liveness Probe:**
- Changed from gRPC to HTTP
- Path: `/healthz`
- Port: 8080
- initialDelaySeconds: 10
- periodSeconds: 15
- timeoutSeconds: 5
- failureThreshold: 3

**Readiness Probe:**
- Changed from gRPC to HTTP
- Path: `/readyz`
- Port: 8080
- initialDelaySeconds: 5
- periodSeconds: 5
- timeoutSeconds: 1
- failureThreshold: 1

**Startup Probe (Added):**
- Path: `/readyz`
- Port: 8080
- initialDelaySeconds: 0
- periodSeconds: 2
- timeoutSeconds: 1
- failureThreshold: 30

### 4. Tests Added

#### health_integration_test.go
Created comprehensive integration tests:
- `TestHealthEndpoints`: Verifies all three health endpoints return correct responses
- `TestHealthCheckerIntegration`: Integration test for health checker with auth-service
- `TestHealthCheckerWithMockCheck`: Verifies health checker behavior with mock checks

All tests pass successfully.

## Health Check Behavior

### Liveness Check (`/healthz`)
- Checks if the service process is alive
- Monitors:
  - Heartbeat (goroutine deadlock detection)
  - Memory usage
  - Goroutine count
- Does NOT check external dependencies
- Returns:
  - `200 OK` with "OK" body if alive
  - `503 Service Unavailable` with "NOT ALIVE" body if not alive

### Readiness Check (`/readyz`)
- Checks if the service is ready to serve traffic
- Currently has no registered health checks (stateless service)
- If dependencies are added, will monitor:
  - Database connectivity
  - Redis connectivity
- Uses anti-flapping logic (requires 3 consecutive failures before marking not ready)
- Returns:
  - `200 OK` with "READY" body if ready
  - `503 Service Unavailable` with "NOT READY" body if not ready

### Detailed Health Status (`/health`)
- Returns comprehensive JSON health status
- Includes:
  - Overall status (healthy/degraded/critical)
  - Health score (0.0 to 1.0)
  - Individual component health (when dependencies are added)
  - Response times
  - Error messages
- Returns:
  - `200 OK` if status is healthy or degraded
  - `503 Service Unavailable` if status is critical

## Architecture

auth-service now runs two servers:
1. **gRPC Server** (port 9095): Handles authentication requests
2. **HTTP Server** (port 8080): Provides health check endpoints

This dual-server approach allows Kubernetes to use HTTP probes while maintaining gRPC for the main service functionality.

## Metrics Exported

The health checker automatically exports Prometheus metrics:
- `health_status`: Overall health status (0=critical, 1=degraded, 2=healthy)
- `health_score`: Health score (0.0 to 1.0)
- `component_status`: Per-component health status (when dependencies are added)
- `component_response_time_seconds`: Per-component response time histogram
- `health_check_failures_total`: Per-component failure counter

## Configuration

Health check behavior can be configured via the health.Config struct in main.go:
- `ServiceName`: Service identifier for metrics and logging
- `CheckInterval`: How often to run health checks (default: 5s)
- `DefaultTimeout`: Default timeout for health checks (default: 100ms)
- `FailureThreshold`: Number of consecutive failures before marking not ready (default: 3)
- `HealthyScore`: Minimum score for healthy status (default: 0.8)
- `DegradedScore`: Minimum score for degraded status (default: 0.5)

## Testing Locally

To test the health endpoints locally:

```bash
# Start the service
go run main.go

# Test liveness
curl http://localhost:8080/healthz

# Test readiness
curl http://localhost:8080/readyz

# Test detailed health
curl http://localhost:8080/health | jq
```

## Future Enhancements

If auth-service adds dependencies in the future (e.g., database for token storage, Redis for session management), health checks can be easily added:

```go
// Example: Add database health check
db, err := sql.Open("mysql", dsn)
hc.RegisterCheck(health.NewDatabaseCheck("database", db))

// Example: Add Redis health check
redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
hc.RegisterCheck(health.NewRedisCheck("redis", redisClient))
```

## Next Steps

1. Deploy to staging environment
2. Verify Kubernetes probes are working correctly
3. Monitor health metrics in Grafana
4. Test failure scenarios if dependencies are added
5. Verify auto-recovery behavior

## References

- Health Check Library: `libs/health/README.md`
- Design Document: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
- Shortener Service Integration: `apps/shortener-service/HEALTH_CHECK_INTEGRATION.md`
