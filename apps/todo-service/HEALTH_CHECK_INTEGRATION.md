# Health Check Integration - TODO Service

## Overview
This document describes the health check integration completed for the todo-service as part of the health check standardization initiative.

## Changes Made

### 1. Dependencies Added
- Added `github.com/pingxin403/cuckoo/libs/health` to go.mod
- Added `github.com/IBM/sarama` (required by health library)
- Added `github.com/redis/go-redis/v9` (required by health library)
- Added replace directive for local development

### 2. Code Changes

#### main.go
- Imported the health library
- Initialized HealthChecker with appropriate configuration:
  - ServiceName: from environment or "todo-service"
  - CheckInterval: 5 seconds
  - DefaultTimeout: 100 milliseconds
  - FailureThreshold: 3 consecutive failures
- Started health checker before service startup
- Added HTTP server on port 8080 for health endpoints:
  - `/healthz` - Liveness probe
  - `/readyz` - Readiness probe
  - `/health` - Detailed health status (JSON)
- Added health checker shutdown in graceful shutdown sequence

**Note**: Unlike other services, todo-service uses in-memory storage and has no external dependencies (no database, no Redis, no Kafka). Therefore, only liveness checks are performed. The service will always be ready unless the process itself is unhealthy.

### 3. Kubernetes Manifests Updated

#### deploy/k8s/services/todo-service/todo-service-deployment.yaml
Updated probe configuration to match health check library standards:

**Ports:**
- Added HTTP port 8080 for health endpoints
- Kept gRPC port 9091 for service operations

**Environment Variables:**
- Added `HTTP_PORT=8080`
- Added `SERVICE_NAME=todo-service`
- Added `SERVICE_VERSION=1.0.0`
- Added `DEPLOYMENT_ENVIRONMENT=production`
- Added observability configuration

**Liveness Probe:**
- Path: `/healthz` (changed from gRPC probe)
- Port: 8080 (HTTP)
- initialDelaySeconds: 10 (was 15)
- periodSeconds: 15 (was 10)
- timeoutSeconds: 5
- failureThreshold: 3

**Readiness Probe:**
- Path: `/readyz` (changed from gRPC probe)
- Port: 8080 (HTTP)
- initialDelaySeconds: 5
- periodSeconds: 5
- timeoutSeconds: 1
- failureThreshold: 1

**Startup Probe:**
- Path: `/readyz`
- Port: 8080 (HTTP)
- initialDelaySeconds: 0
- periodSeconds: 2
- timeoutSeconds: 1
- failureThreshold: 30 (allows up to 60 seconds for startup)

### 4. Tests Added

#### health_integration_test.go
Created comprehensive integration tests:
- `TestHealthEndpoints`: Verifies all three health endpoints return correct responses
- `TestHealthDetailedResponse`: Verifies the /health endpoint returns proper JSON with required fields
- `TestHealthEndpointsUnderLoad`: Verifies health endpoints work under concurrent load (50 requests)

All tests can be run with the integration build tag.

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
- For todo-service: Since there are no external dependencies (in-memory storage only), this primarily checks the liveness state
- Uses anti-flapping logic (requires 3 consecutive failures before marking not ready)
- Returns:
  - `200 OK` with "READY" body if ready
  - `503 Service Unavailable` with "NOT READY" body if not ready

### Detailed Health Status (`/health`)
- Returns comprehensive JSON health status
- Includes:
  - Overall status (healthy/degraded/critical)
  - Health score (0.0 to 1.0)
  - Service name and timestamp
  - Individual component health (if any)
- Returns:
  - `200 OK` if status is healthy or degraded
  - `503 Service Unavailable` if status is critical

## Architecture Notes

### In-Memory Storage
Unlike other services in the monorepo, todo-service uses in-memory storage (`storage.NewMemoryStore()`) rather than a database. This means:
- No database health checks are registered
- No auto-recovery mechanisms needed for database connections
- Service is always ready unless the process itself is unhealthy
- Data is lost on service restart (acceptable for this demo service)

### No External Dependencies
The todo-service has no external dependencies:
- ❌ No database (uses in-memory storage)
- ❌ No Redis cache
- ❌ No Kafka messaging
- ❌ No downstream services

This makes it the simplest service in terms of health checking - only liveness checks are meaningful.

## Metrics Exported

The health checker automatically exports Prometheus metrics:
- `health_status`: Overall health status (0=critical, 1=degraded, 2=healthy)
- `health_score`: Health score (0.0 to 1.0)
- `component_status`: Per-component health status (none for todo-service)
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

# In another terminal:

# Test liveness
curl http://localhost:8080/healthz

# Test readiness
curl http://localhost:8080/readyz

# Test detailed health
curl http://localhost:8080/health | jq

# Test gRPC service (should still work)
grpcurl -plaintext localhost:9091 list
```

## Running Integration Tests

```bash
# Run all integration tests including health checks
go test -tags=integration -v ./...

# Run only health check tests
go test -tags=integration -v -run TestHealth ./...
```

## Deployment

### Local Development
```bash
# Using Docker Compose
docker-compose up todo-service

# Verify health
curl http://localhost:8080/health
```

### Kubernetes
```bash
# Deploy to Kubernetes
kubectl apply -f deploy/k8s/services/todo-service/

# Check pod status
kubectl get pods -l app=todo-service

# Check health endpoints
kubectl port-forward svc/todo-service 8080:8080
curl http://localhost:8080/health
```

## Comparison with Other Services

| Service | Database | Redis | Kafka | Health Checks |
|---------|----------|-------|-------|---------------|
| todo-service | ❌ In-memory | ❌ | ❌ | Liveness only |
| shortener-service | ✅ MySQL | ✅ | ✅ Optional | DB + Redis + Kafka |
| auth-service | ✅ | ✅ | ❌ | DB + Redis |
| user-service | ✅ | ✅ | ❌ | DB + Redis |
| im-service | ✅ | ✅ | ✅ | DB + Redis + Kafka + etcd |

## Next Steps

1. ✅ Code changes completed
2. ✅ Kubernetes manifests updated
3. ✅ Integration tests created
4. ⏳ Deploy to staging environment
5. ⏳ Verify Kubernetes probes are working correctly
6. ⏳ Monitor health metrics in Grafana
7. ⏳ Test failure scenarios (process deadlock, memory exhaustion)

## References

- Health Check Library: `libs/health/README.md`
- Design Document: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
- Shortener Service Integration: `apps/shortener-service/HEALTH_CHECK_INTEGRATION.md`

