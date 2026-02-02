# Health Check Integration - Shortener Service

## Overview
This document describes the health check integration completed for the shortener-service as part of the health check standardization initiative.

## Changes Made

### 1. Dependencies Added
- Added `github.com/pingxin403/cuckoo/libs/health` to go.mod
- Added replace directive for local development
- Ran `go mod tidy` to update dependencies

### 2. Code Changes

#### main.go
- Imported the health library
- Initialized HealthChecker with appropriate configuration:
  - ServiceName: from environment or "shortener-service"
  - CheckInterval: 5 seconds
  - DefaultTimeout: 100 milliseconds
  - FailureThreshold: 3 consecutive failures
- Registered health checks:
  - **Database (MySQL)**: Critical check using `health.NewDatabaseCheck()`
  - **Redis**: Critical check (if Redis is configured) using `health.NewRedisCheck()`
- Started health checker before service startup
- Added health endpoints to HTTP router:
  - `/healthz` - Liveness probe
  - `/readyz` - Readiness probe
  - `/health` - Detailed health status (JSON)
- Applied readiness middleware to application routes (not health endpoints)
- Added health checker shutdown in graceful shutdown sequence

#### storage/mysql_store.go
- Added `DB() *sql.DB` method to expose underlying database connection for health checks

#### cache/l2_cache.go
- Added `Client() redis.UniversalClient` method to expose Redis client for health checks

#### service/redirect_handler.go
- Removed old health check endpoints (`/health`, `/ready`)
- Removed old health check methods (`HealthCheck()`, `ReadinessCheck()`)
- Cleaned up unused imports

### 3. Kubernetes Manifests Updated

#### deploy/k8s/services/shortener-service/shortener-service-deployment.yaml
Updated probe configuration to match health check library standards:

**Liveness Probe:**
- Path: `/healthz` (was `/health`)
- initialDelaySeconds: 10 (was 30)
- periodSeconds: 15 (was 10)
- timeoutSeconds: 5
- failureThreshold: 3

**Readiness Probe:**
- Path: `/readyz` (was `/ready`)
- initialDelaySeconds: 5 (was 10)
- periodSeconds: 5
- timeoutSeconds: 1 (was 3)
- failureThreshold: 1 (was 3)

**Startup Probe:**
- Path: `/readyz` (was `/health`)
- initialDelaySeconds: 0
- periodSeconds: 2 (was 5)
- timeoutSeconds: 1 (was 3)
- failureThreshold: 30

### 4. Tests Added

#### health_integration_test.go
Created comprehensive integration tests:
- `TestHealthEndpoints`: Verifies all three health endpoints return correct responses
- `TestHealthChecksWithDatabase`: Integration test for database health checks (skipped without DB)
- `TestReadinessMiddleware`: Verifies readiness middleware allows requests when ready
- `TestStorageDBMethod`: Verifies storage exposes DB() method

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
- Monitors all registered health checks:
  - Database connectivity (MySQL)
  - Redis connectivity (if configured)
- Uses anti-flapping logic (requires 3 consecutive failures before marking not ready)
- Returns:
  - `200 OK` with "READY" body if ready
  - `503 Service Unavailable` with "NOT READY" body if not ready

### Detailed Health Status (`/health`)
- Returns comprehensive JSON health status
- Includes:
  - Overall status (healthy/degraded/critical)
  - Health score (0.0 to 1.0)
  - Individual component health
  - Response times
  - Error messages
- Returns:
  - `200 OK` if status is healthy or degraded
  - `503 Service Unavailable` if status is critical

## Readiness Middleware

The readiness middleware is applied to all application routes (redirect handler) but NOT to health check endpoints. This ensures:
- Health endpoints are always accessible for Kubernetes probes
- Application routes reject traffic when dependencies are unhealthy
- Graceful degradation when services are not ready

## Metrics Exported

The health checker automatically exports Prometheus metrics:
- `health_status`: Overall health status (0=critical, 1=degraded, 2=healthy)
- `health_score`: Health score (0.0 to 1.0)
- `component_status`: Per-component health status
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

## Next Steps

1. Deploy to staging environment
2. Verify Kubernetes probes are working correctly
3. Monitor health metrics in Grafana
4. Test failure scenarios (database down, Redis down)
5. Verify auto-recovery behavior

## References

- Health Check Library: `libs/health/README.md`
- Design Document: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
