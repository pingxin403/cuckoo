# Health Check Integration - IM Service

## Overview
This document describes the health check integration completed for the im-service as part of the health check standardization initiative. The im-service is the MOST COMPLEX service in the monorepo with multiple dependencies including MySQL, Redis, Kafka, etcd, and a custom offline worker component.

## Changes Made

### 1. Dependencies Added
- Added `github.com/pingxin403/cuckoo/libs/health` to go.mod
- Added replace directive for local development
- Ran `go get` and `go mod tidy` to update dependencies
- Fixed incorrect import in `sync/conflict_resolver.go` (changed from `github.com/cuckoo-org/cuckoo` to `github.com/pingxin403/cuckoo`)

### 2. Code Changes

#### main.go
- Imported the health library
- Initialized HealthChecker with appropriate configuration:
  - ServiceName: from config or "im-service"
  - CheckInterval: 5 seconds
  - DefaultTimeout: 100 milliseconds
  - FailureThreshold: 3 consecutive failures
- Registered health checks for ALL dependencies:
  - **Database (MySQL)**: Critical check using `health.NewDatabaseCheck()`
  - **Redis**: Critical check using `health.NewRedisCheck()`
  - **Kafka**: Non-critical check using `health.NewKafkaCheck()` (only if worker enabled)
  - **etcd**: Critical check using custom `NewEtcdHealthCheck()` for service discovery
  - **Offline Worker**: Non-critical check using custom `NewOfflineWorkerHealthCheck()` (only if worker enabled)
- Started health checker before service startup
- Replaced existing health endpoints with health library endpoints:
  - `/healthz` - Liveness probe (new)
  - `/readyz` - Readiness probe (new)
  - `/health` - Detailed health status (replaced)
  - `/ready` - Legacy endpoint (kept for backward compatibility)
- Applied readiness middleware to Read Receipt API routes (not health endpoints)
- Added health checker shutdown in graceful shutdown sequence

#### dedup/dedup_service.go
- Added `GetClient() redis.UniversalClient` method to expose Redis client for health checks

#### health_checks.go (NEW FILE)
Created custom health checks specific to im-service:

**EtcdHealthCheck:**
- Checks etcd connectivity via registry client
- Uses `GetServiceNodes()` to verify etcd is responding
- Timeout: 200ms
- Interval: 10 seconds
- Critical: true (etcd is critical for service discovery)

**OfflineWorkerHealthCheck:**
- Checks offline worker health using `GetStats()`
- Monitors error rate and processing status
- Fails if worker has errors but no messages processed
- Fails if error rate > 10%
- Timeout: 100ms
- Interval: 5 seconds
- Critical: false (worker is not critical for service operation)

#### startHTTPServer function
- Updated signature to accept `*health.HealthChecker` parameter
- Replaced old health check endpoints with health library handlers
- Applied readiness middleware to all Read Receipt API endpoints:
  - `/api/v1/messages/read`
  - `/api/v1/messages/unread/count`
  - `/api/v1/messages/unread`
  - `/api/v1/messages/receipts`
  - `/api/v1/conversations/read`

### 3. Kubernetes Manifests Updated

#### deploy/k8s/services/im-service/im-service-deployment.yaml
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

**Startup Probe (NEW):**
- Path: `/readyz`
- initialDelaySeconds: 0
- periodSeconds: 2
- timeoutSeconds: 1
- failureThreshold: 30 (60 seconds total startup time)

### 4. Tests Added

#### health_integration_test.go
Created comprehensive integration tests:
- `TestHealthEndpoints`: Verifies all three health endpoints return correct responses
- `TestHealthChecksWithDependencies`: Integration test for all health checks (requires infrastructure)
- `TestReadinessMiddleware`: Verifies readiness middleware allows/rejects requests correctly
- `TestCustomHealthChecks`: Tests custom etcd and offline worker health checks
- `TestStorageGetDBMethod`: Verifies storage exposes DB() method
- `TestDedupGetClientMethod`: Verifies dedup service exposes GetClient() method
- `TestWorkerHealthCheckWithStats`: Tests worker health check with different stats scenarios

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
  - Database connectivity (MySQL) - **CRITICAL**
  - Redis connectivity - **CRITICAL**
  - Kafka connectivity - **NON-CRITICAL** (only if worker enabled)
  - etcd connectivity - **CRITICAL** (service discovery)
  - Offline worker health - **NON-CRITICAL** (only if worker enabled)
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

The readiness middleware is applied to all Read Receipt API routes but NOT to health check endpoints. This ensures:
- Health endpoints are always accessible for Kubernetes probes
- API routes reject traffic when dependencies are unhealthy
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

## Dependency Health Checks

### 1. Database (MySQL) - CRITICAL
- Uses `health.NewDatabaseCheck("database", store.GetDB())`
- Performs ping and simple query check
- Timeout: 100ms
- Critical: Yes - service cannot operate without database

### 2. Redis - CRITICAL
- Uses `health.NewRedisCheck("redis", dedupService.GetClient())`
- Performs ping check
- Timeout: 100ms
- Critical: Yes - required for deduplication and sequence generation

### 3. Kafka - NON-CRITICAL
- Uses `health.NewKafkaCheck("kafka", cfg.Kafka.Brokers)`
- Lists topics to verify connectivity
- Timeout: 200ms
- Critical: No - only affects offline message processing
- Only registered if `OFFLINE_WORKER_ENABLED=true`

### 4. etcd - CRITICAL
- Uses custom `NewEtcdHealthCheck("etcd", registryClient)`
- Performs GetServiceNodes operation
- Timeout: 200ms
- Critical: Yes - required for service discovery and registration

### 5. Offline Worker - NON-CRITICAL
- Uses custom `NewOfflineWorkerHealthCheck("offline-worker", offlineWorker)`
- Monitors worker stats and error rates
- Timeout: 100ms
- Critical: No - service can operate without worker
- Only registered if worker is enabled and initialized

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

# Test legacy endpoint (backward compatibility)
curl http://localhost:8080/ready
```

## Testing with Dependencies

To run integration tests with actual dependencies:

```bash
# Run all tests (requires MySQL, Redis, Kafka, etcd)
go test -v ./...

# Run only unit tests (no infrastructure required)
go test -v -short ./...

# Run specific health check tests
go test -v -run TestHealthEndpoints
go test -v -run TestCustomHealthChecks
```

## Known Issues

### Compilation Errors
There are existing compilation errors in the codebase that need to be fixed before tests can run:

1. **storage/offline_store.go**: Type mismatch with `hlc.GlobalID` (string vs struct)
2. **sequence/sequence_generator.go**: HLC API changes (PhysicalTime/LogicalTime fields, UpdateFromRemote signature)

These errors are unrelated to the health check integration and exist in the multi-region feature code.

## Failure Scenarios

### Database Failure
- Health check detects failure within 5 seconds (check interval)
- After 3 consecutive failures (15 seconds), service marked as not ready
- Kubernetes removes pod from service load balancer
- New requests to API endpoints return 503 Service Unavailable
- In-flight requests allowed to complete

### Redis Failure
- Similar behavior to database failure
- Service marked as not ready after 3 consecutive failures
- Affects deduplication and sequence generation

### Kafka Failure
- Health check detects failure
- Service remains READY (Kafka is non-critical)
- Offline message processing affected but service continues operating
- Status may show as "degraded" instead of "critical"

### etcd Failure
- Health check detects failure within 10 seconds
- After 3 consecutive failures (30 seconds), service marked as not ready
- Service discovery and registration affected
- Service cannot register itself or discover other services

### Offline Worker Failure
- Health check detects high error rate or stuck worker
- Service remains READY (worker is non-critical)
- Only affects offline message processing
- Status may show as "degraded"

## Next Steps

1. **Fix Compilation Errors**: Resolve HLC-related type mismatches in storage and sequence packages
2. **Run Tests**: Execute integration tests once compilation errors are fixed
3. **Deploy to Staging**: Test with actual infrastructure
4. **Verify Kubernetes Probes**: Ensure probes work correctly in K8s environment
5. **Monitor Metrics**: Verify health metrics appear in Grafana
6. **Test Failure Scenarios**: Simulate dependency failures and verify behavior
7. **Verify Auto-Recovery**: Test that service recovers when dependencies restore

## Comparison with Other Services

### Complexity Comparison

| Service | Dependencies | Custom Checks | Complexity |
|---------|-------------|---------------|------------|
| shortener-service | MySQL, Redis | 0 | Low |
| todo-service | MySQL | 0 | Low |
| auth-service | None (stateless) | 0 | Very Low |
| user-service | MySQL, Redis | 0 | Low |
| **im-service** | **MySQL, Redis, Kafka, etcd, Worker** | **2** | **Very High** |

### Unique Aspects of im-service Integration

1. **Most Dependencies**: 5 health checks (vs 1-2 for other services)
2. **Custom Health Checks**: 2 custom checks (etcd, offline worker)
3. **Mixed Criticality**: Both critical and non-critical dependencies
4. **Conditional Checks**: Kafka and worker checks only registered if enabled
5. **Complex Worker Health**: Custom logic to monitor worker stats and error rates
6. **Service Discovery**: etcd health check for service registration

## References

- Health Check Library: `libs/health/README.md`
- Design Document: `.kiro/specs/health-check-standardization/design.md`
- Requirements: `.kiro/specs/health-check-standardization/requirements.md`
- Shortener Service Integration: `apps/shortener-service/HEALTH_CHECK_INTEGRATION.md`
- Auth Service Integration: `apps/auth-service/HEALTH_CHECK_INTEGRATION.md`
- User Service Integration: `apps/user-service/HEALTH_CHECK_INTEGRATION.md`
- Todo Service Integration: `apps/todo-service/HEALTH_CHECK_INTEGRATION.md`

## Conclusion

The im-service health check integration is complete and follows the standardized pattern established by other services. The integration properly handles the service's complexity with multiple dependencies, custom health checks, and mixed criticality levels. Once the existing compilation errors are resolved, the service will be ready for testing and deployment.
