# Health Check System - FAQ

Frequently asked questions about the standardized health check system.

---

## General Questions

### What is the health check system?

The health check system is a standardized library (`libs/health`) that provides comprehensive health monitoring for all Go services. It includes liveness probes, readiness probes, auto-recovery, circuit breakers, and detailed metrics.

### Why do we need standardized health checks?

Before standardization, each service implemented health checks differently, leading to:
- Inconsistent behavior across services
- False positives causing unnecessary pod restarts
- Manual recovery required for transient failures
- Difficult troubleshooting

The standardized system solves these problems with consistent, reliable health monitoring.

### Which services use the health check system?

All 6 Go services in the monorepo:
- shortener-service
- todo-service
- auth-service
- user-service
- im-service
- im-gateway-service

---

## Architecture Questions

### What's the difference between liveness and readiness probes?

**Liveness Probe** (`/healthz`):
- Checks if the process is healthy (heartbeat, memory, goroutines)
- Used by Kubernetes to restart unhealthy pods
- Fast (< 50ms)
- Should NOT check dependencies

**Readiness Probe** (`/readyz`):
- Checks if the service can handle traffic (dependencies healthy)
- Used by Kubernetes to route traffic
- Comprehensive (< 200ms)
- Checks all dependencies (database, Redis, etc.)

**Key difference**: Liveness is about process health, readiness is about ability to serve traffic.

### Why separate liveness and readiness?

Separating them prevents unnecessary pod restarts. If a database is temporarily unavailable, the pod should be marked not ready (stop receiving traffic) but not restarted (liveness still healthy). Once the database recovers, the pod becomes ready again without restarting.

### What is anti-flapping logic?

Anti-flapping requires 3 consecutive failures before marking a service not ready. This prevents rapid ready/not-ready transitions caused by transient network issues, leading to more stable pod lifecycles.

### How does auto-recovery work?

When a dependency fails, the health check system automatically attempts to reconnect using exponential backoff:
1. Detect failure
2. Open circuit breaker
3. Attempt reconnection (1s, 2s, 4s, 8s, ...)
4. Close circuit breaker on success
5. Mark service ready again

No manual intervention required for transient failures.

### What is a circuit breaker?

A circuit breaker prevents cascading failures when dependencies are down. It has three states:
- **Closed** (normal): All checks pass
- **Open** (failing): After 3 consecutive failures, stops checking temporarily
- **Half-Open** (testing): Attempts recovery after timeout

This protects both the service and its dependencies from resource exhaustion.

---

## Integration Questions

### How do I add health checks to a new service?

Follow these steps:

1. Add dependency:
```go
import "github.com/your-org/monorepo/libs/health"
```

2. Create health checker:
```go
checker := health.NewHealthChecker(health.Config{
    ServiceName: "my-service",
})
```

3. Register checks:
```go
checker.RegisterCheck("database", health.NewDatabaseCheck(db))
checker.RegisterCheck("redis", health.NewRedisCheck(redisClient))
```

4. Start and expose endpoints:
```go
checker.Start()
defer checker.Stop()

http.HandleFunc("/healthz", checker.HealthzHandler())
http.HandleFunc("/readyz", checker.ReadyzHandler())
http.HandleFunc("/health", checker.HealthHandler())
```

See [Health Check Library README](../../../libs/health/README.md) for complete examples.

### What built-in health checks are available?

- `DatabaseCheck` - MySQL/PostgreSQL
- `RedisCheck` - Redis/Redis Cluster
- `KafkaCheck` - Kafka brokers
- `HTTPCheck` - HTTP endpoints
- `GRPCCheck` - gRPC services

### How do I create a custom health check?

Implement the `Check` interface:

```go
type MyCustomCheck struct {
    // your fields
}

func (c *MyCustomCheck) Check(ctx context.Context) error {
    // your health check logic
    if !healthy {
        return fmt.Errorf("not healthy: %v", reason)
    }
    return nil
}
```

Then register it:

```go
checker.RegisterCheck("my-check", &MyCustomCheck{})
```

### Can I mark a check as non-critical?

Yes, use `RegisterNonCriticalCheck()`:

```go
checker.RegisterNonCriticalCheck("cache", health.NewRedisCheck(redisClient))
```

Non-critical checks don't affect readiness status but are still monitored.

### How do I configure health check timeouts?

Set timeouts in the config:

```go
checker := health.NewHealthChecker(health.Config{
    CheckInterval: 10 * time.Second,  // How often to run checks
    CheckTimeout:  5 * time.Second,   // Timeout for each check
})
```

Individual checks can also have their own timeouts.

---

## Operational Questions

### How do I check if a service is healthy?

Use the health endpoints:

```bash
# Liveness
curl http://service:8080/healthz

# Readiness
curl http://service:8080/readyz

# Detailed status
curl http://service:8080/health | jq
```

Or check the Grafana dashboard: "Health Check Monitoring"

### What do the HTTP status codes mean?

- `200 OK` - Healthy/Ready
- `503 Service Unavailable` - Unhealthy/Not Ready

### How long does it take to detect a failure?

- **Detection time**: < 15 seconds (with default 10s check interval and anti-flapping)
- **Recovery time**: < 60 seconds (with auto-recovery)

### What happens when a service is not ready?

1. Readiness probe returns 503
2. Kubernetes removes pod from service endpoints
3. No new traffic is routed to the pod
4. Existing connections are allowed to complete
5. Alert fires after 5 minutes (configurable)

### What happens when a service is not live?

1. Liveness probe returns 503
2. Kubernetes restarts the pod
3. Alert fires immediately (critical)

### How do I troubleshoot health check failures?

1. Check full health status:
```bash
curl http://service:8080/health | jq
```

2. Identify failing component
3. Check component-specific logs
4. Verify dependency is healthy
5. Check network connectivity
6. Consult [Operational Runbook](../../operations/health-check-runbook.md)

### Can I disable health checks temporarily?

Yes, but not recommended. Set environment variable:

```bash
HEALTH_ENABLED=false
```

Always re-enable after debugging. Disabling health checks removes monitoring and auto-recovery.

### How do I manually trigger recovery?

Restart the pod:

```bash
kubectl rollout restart deployment/service-name
```

However, auto-recovery should handle most cases automatically.

---

## Monitoring Questions

### Where can I see health metrics?

- **Grafana Dashboard**: "Health Check Monitoring" (http://grafana.example.com/d/health-checks)
- **Prometheus**: http://prometheus.example.com
- **Service endpoint**: `http://service:8080/health`

### What metrics are available?

**Health Status**:
- `health_status` - Overall service health (0=critical, 1=degraded, 2=healthy)
- `health_score` - Health score percentage (0-100)
- `component_status` - Individual component health

**Performance**:
- `component_response_time_seconds` - Health check latency
- `health_check_failures_total` - Failure counter

**Circuit Breaker**:
- `circuit_breaker_state` - Circuit breaker state (0=closed, 1=open, 2=half-open)
- `recovery_attempts_total` - Recovery attempts
- `recovery_success_total` - Successful recoveries

### What alerts are configured?

**Critical** (PagerDuty):
- `LivenessCheckFailed` - Immediate
- `ServiceNotReady` - After 5 minutes

**Warning** (Slack):
- `HighHealthCheckFailureRate` - > 10% failures
- `SlowHealthCheck` - P99 > 200ms
- `ComponentCritical` - Component down > 2 minutes
- `CircuitBreakerOpen` - Circuit breaker open > 5 minutes

See [Prometheus Alerts](../../../deploy/docker/prometheus-health-alerts.yml) for complete list.

### How do I create custom alerts?

Add alert rules to `prometheus-health-alerts.yml`:

```yaml
- alert: MyCustomAlert
  expr: health_score{service="my-service"} < 80
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Custom alert for {{ $labels.service }}"
    description: "Health score below 80%"
```

---

## Performance Questions

### What is the performance impact?

Minimal:
- Health check latency: ~50ms (target < 200ms)
- Middleware overhead: ~10μs (target < 100μs)
- Memory overhead: ~5MB per service (target < 10MB)
- CPU overhead: < 0.5% (target < 1%)

### Why are health checks slow?

Possible causes:
1. Slow dependency (database, Redis)
2. Network latency
3. Timeout too high
4. Too many checks running sequentially

**Solution**: Check component response times in `/health` endpoint, optimize slow checks, or adjust timeouts.

### Can health checks cause performance issues?

No, if configured correctly:
- Checks run in parallel
- Timeouts prevent hanging
- Lock-free readiness check
- Minimal resource usage

If you see performance issues, check for:
- Timeout too high (> 5s)
- Too many checks (> 10)
- Slow dependencies

---

## Configuration Questions

### What environment variables are supported?

```bash
# Liveness configuration
HEALTH_LIVENESS_ENABLED=true
HEALTH_LIVENESS_HEARTBEAT_TIMEOUT=30s
HEALTH_LIVENESS_MEMORY_THRESHOLD=90
HEALTH_LIVENESS_GOROUTINE_THRESHOLD=10000

# Readiness configuration
HEALTH_READINESS_ENABLED=true
HEALTH_READINESS_FAILURE_THRESHOLD=3
HEALTH_READINESS_CHECK_INTERVAL=10s

# Circuit breaker configuration
HEALTH_CIRCUIT_BREAKER_ENABLED=true
HEALTH_CIRCUIT_BREAKER_THRESHOLD=3
HEALTH_CIRCUIT_BREAKER_TIMEOUT=60s

# Auto-recovery configuration
HEALTH_RECOVERY_ENABLED=true
HEALTH_RECOVERY_MAX_RETRIES=5
HEALTH_RECOVERY_INITIAL_BACKOFF=1s
HEALTH_RECOVERY_MAX_BACKOFF=60s
```

### How do I configure Kubernetes probes?

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

### Should I enable auto-recovery?

Yes, for most dependencies:
- Database: ✅ Yes (transient connection issues)
- Redis: ✅ Yes (cache can be rebuilt)
- Kafka: ⚠️ Maybe (depends on use case)
- HTTP services: ⚠️ Maybe (depends on service)

Disable for dependencies that require manual intervention.

### Should I enable circuit breakers?

Yes, always. Circuit breakers:
- Prevent cascading failures
- Protect dependencies
- Enable automatic recovery
- Minimal overhead

---

## Testing Questions

### How do I test health checks locally?

1. Start dependencies with Docker Compose:
```bash
docker-compose -f docker-compose.infra.yml up -d
```

2. Run service:
```bash
go run .
```

3. Test endpoints:
```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/health | jq
```

4. Simulate failures:
```bash
docker stop mysql
# Wait and check health
curl http://localhost:8080/health | jq
```

### How do I test auto-recovery?

1. Stop dependency:
```bash
docker stop mysql
```

2. Wait for failure detection (~15s)

3. Restart dependency:
```bash
docker start mysql
```

4. Watch recovery:
```bash
watch -n 2 'curl -s http://localhost:8080/health | jq ".components[] | select(.name==\"database\")"'
```

### How do I test circuit breakers?

1. Stop dependency (circuit breaker opens after 3 failures)
2. Check circuit breaker state in Prometheus:
```promql
circuit_breaker_state{service="my-service",component="database"}
```
3. Restart dependency (circuit breaker transitions to half-open, then closed)

### Are there integration tests?

Yes, each service has integration tests:
- `apps/*/health_integration_test.go`

Run with:
```bash
go test -v ./... -tags=integration
```

---

## Troubleshooting Questions

### Why is my service not ready?

1. Check full health status:
```bash
curl http://service:8080/health | jq
```

2. Identify failing component
3. Check dependency health
4. Verify network connectivity
5. Check service logs
6. Consult [Operational Runbook](../../operations/health-check-runbook.md)

### Why is auto-recovery not working?

Possible causes:
1. Dependency is actually down (not transient)
2. Network connectivity issues
3. Credentials invalid
4. Max retries exceeded
5. Auto-recovery disabled

Check logs for recovery attempts and errors.

### Why is the circuit breaker stuck open?

Possible causes:
1. Dependency is still down
2. Timeout too short
3. Threshold too low
4. Recovery not working

Check dependency health and circuit breaker configuration.

### Why are health checks flapping?

Possible causes:
1. Intermittent network issues
2. Dependency instability
3. Timeout too aggressive
4. Resource constraints (CPU throttling)

**Solution**: Anti-flapping should prevent this (requires 3 consecutive failures). If still flapping, increase timeout or fix underlying instability.

### How do I get help?

1. Check [Operational Runbook](../../operations/health-check-runbook.md)
2. Check [Health Check Library README](../../../libs/health/README.md)
3. Ask in Slack: #platform-engineering
4. Contact on-call engineer (for production issues)
5. Attend office hours (TBD)

---

## Best Practices

### What should I check in liveness probes?

✅ **DO check**:
- Process heartbeat
- Memory usage
- Goroutine count
- Deadlock detection

❌ **DON'T check**:
- Database connectivity
- Redis connectivity
- Downstream services
- External APIs

**Why**: Liveness failures cause pod restarts. Only check things that require a restart to fix.

### What should I check in readiness probes?

✅ **DO check**:
- Database connectivity
- Redis connectivity
- Required downstream services
- Critical dependencies

❌ **DON'T check**:
- Optional dependencies (use non-critical checks)
- External APIs with high latency
- Services that don't affect traffic handling

**Why**: Readiness failures remove pod from load balancer. Only check things that prevent serving traffic.

### How often should health checks run?

**Recommended**:
- Check interval: 10 seconds
- Timeout: 5 seconds
- Failure threshold: 3

**Adjust based on**:
- Service criticality (more critical = more frequent)
- Dependency stability (unstable = less frequent to avoid flapping)
- Performance requirements (high QPS = less frequent)

### When should I use custom health checks?

Use custom checks for:
- Non-standard dependencies (etcd, custom protocols)
- Business logic validation
- Resource availability (disk space, file handles)
- Service-specific requirements

See [Custom Health Checks](../../../libs/health/README.md#custom-health-checks) for examples.

---

## Migration Questions

### How do I migrate from old health checks?

1. Add health library dependency
2. Create health checker
3. Register checks for existing dependencies
4. Add new endpoints (`/healthz`, `/readyz`, `/health`)
5. Update Kubernetes manifests
6. Test in staging
7. Deploy to production
8. Remove old health check code

See service integration docs for examples:
- `apps/*/HEALTH_CHECK_INTEGRATION.md`

### Can I run old and new health checks in parallel?

Yes, during migration:
1. Keep old endpoints (`/health`, `/ready`)
2. Add new endpoints (`/healthz`, `/readyz`)
3. Update Kubernetes to use new endpoints
4. Verify in staging
5. Deploy to production
6. Remove old endpoints

### What if my service has custom health logic?

Implement a custom check:

```go
type MyCustomCheck struct {
    // your fields
}

func (c *MyCustomCheck) Check(ctx context.Context) error {
    // your custom logic
    return nil
}

checker.RegisterCheck("my-check", &MyCustomCheck{})
```

---

## Additional Resources

- [Health Check Library README](../../../libs/health/README.md)
- [Operational Runbook](../../operations/health-check-runbook.md)
- [Design Document](../../../.kiro/specs/health-check-standardization/design.md)
- [Service Integration Examples](../../../apps/*/HEALTH_CHECK_INTEGRATION.md)
- [Grafana Dashboard](../../../deploy/docker/grafana/dashboards/health-checks.json)
- [Prometheus Alerts](../../../deploy/docker/prometheus-health-alerts.yml)

---

## Still Have Questions?

- **Slack**: #platform-engineering
- **Email**: platform-team@example.com
- **Office Hours**: TBD
- **On-Call**: See PagerDuty rotation (for production issues)

