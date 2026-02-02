# Health Check Integration - User Service

## Overview
This document describes the health check integration completed for the user-service as part of the health check standardization initiative.

## Changes Made

### 1. Dependencies Added
- Added `github.com/pingxin403/cuckoo/libs/health` to go.mod
- Added replace directive for local development
- Ran `go get` to update dependencies

### 2. Code Changes

#### main.go
- Imported the health library
- Initialized HealthChecker with appropriate configuration:
  - ServiceName: from config or "user-service"
  - CheckInterval: 5 seconds
  - DefaultTimeout: 100 milliseconds
  - FailureThreshold: 3 consecutive failures
- Added HTTP server on port 8080 for health endpoints (alongside gRPC on port 9096)
- Registered health checks:
  - **Database (MySQL)**: Critical check using `health.NewDatabaseCheck()`
- Started health checker before service startup
- Added health endpoints to HTTP server:
  - `/healthz` - Liveness probe
  - `/readyz` - Readiness probe
  - `/health` - Detailed health status (JSON)
- Added health checker shutdown in graceful shutdown sequence

#### storage/mysql_store.go
- Added `DB() *sql.DB` method to expose underlying database connection for health checks

### 3. Kubernetes Manifests Updated

#### deploy/k8s/services/user-service/user-service-deployment.yaml
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
- `TestHealthChecksWithDatabase`: Integration test for database health checks (skipped without DB)
- `TestStorageDBMethod`: Verifies storage exposes DB() method
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
- Monitors all registered health checks:
  - Database connectivity (MySQL)
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

## Architecture

user-service now runs two servers:
1. **gRPC Server** (port 9096): Handles user service requests
2. **HTTP Server** (port 8080): Provides health check endpoints

This dual-server approach allows Kubernetes to use HTTP probes while maintaining gRPC for the main service functionality.

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

## Dependencies

user-service has the following dependencies that are monitored:
- **MySQL Database**: Critical dependency for user data storage
  - Health check: Database ping and simple query
  - Auto-recovery: Not implemented (database connection pool handles reconnection)

**Note**: user-service does not use Redis, so no Redis health check is registered.

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

## Testing with Database

To run integration tests with database:

```bash
# Run all tests including database tests
go test -v ./...

# Run only health tests (skips database tests in short mode)
go test -v -short -run TestHealth
```

## Next Steps

1. Deploy to staging environment
2. Verify Kubernetes probes are working correctly
3. Monitor health metrics in Grafana
4. Test failure scenarios (database down)
5. Verify auto-recovery behavior

## References

- Health Check Library: `libs/health/README.md`
- Design Document: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
- Auth Service Integration: `apps/auth-service/HEALTH_CHECK_INTEGRATION.md`
- Shortener Service Integration: `apps/shortener-service/HEALTH_CHECK_INTEGRATION.md`

