# Health Check System - Operational Runbook

## Overview

This runbook provides operational guidance for the standardized health check system deployed across all Go services. It covers troubleshooting, common issues, recovery procedures, and escalation paths.

**Last Updated**: 2026-02-02  
**Owner**: Platform Engineering Team  
**On-Call**: See PagerDuty rotation

---

## Quick Reference

### Health Check Endpoints

All services expose three health check endpoints:

- **`/healthz`** - Liveness probe (process health only)
- **`/readyz`** - Readiness probe (all dependencies)
- **`/health`** - Full health status (detailed JSON)

### Expected Response Times

- Liveness check: < 50ms
- Readiness check: < 200ms
- Full health status: < 200ms

### Monitoring Dashboards

- **Grafana Dashboard**: `Health Check Monitoring` (UID: `health-checks`)
- **Prometheus Alerts**: `deploy/docker/prometheus-health-alerts.yml`
- **AlertManager Config**: `deploy/docker/alertmanager-config.yml`

---

## Troubleshooting Decision Tree

```
Service Health Issue
│
├─ Is /healthz failing?
│  ├─ YES → Process is unhealthy → See "Liveness Failures"
│  └─ NO → Continue
│
├─ Is /readyz failing?
│  ├─ YES → Dependencies unhealthy → See "Readiness Failures"
│  └─ NO → Continue
│
├─ Are metrics showing degraded status?
│  ├─ YES → Intermittent issues → See "Degraded Components"
│  └─ NO → False alarm or resolved
│
└─ Check Grafana dashboard for component-level details
```

---

## Common Issues and Solutions

### 1. Liveness Check Failures

**Symptoms**:
- `/healthz` returns 503
- Alert: `LivenessCheckFailed`
- Kubernetes restarting pods

**Possible Causes**:
1. Process deadlock or hang
2. Memory exhaustion (> 90% of limit)
3. Goroutine leak (> 10,000 goroutines)
4. Heartbeat timeout (no heartbeat for 30s)

**Diagnosis**:
```bash
# Check pod status
kubectl get pods -n <namespace> | grep <service>

# Check pod logs
kubectl logs <pod-name> -n <namespace> --tail=100

# Check resource usage
kubectl top pod <pod-name> -n <namespace>

# Get detailed pod description
kubectl describe pod <pod-name> -n <namespace>
```

**Resolution**:
1. **If memory exhausted**: Increase memory limits in deployment
2. **If goroutine leak**: Check for unclosed connections, review recent code changes
3. **If deadlock**: Kubernetes will restart pod automatically
4. **If persistent**: Escalate to on-call engineer

**Prevention**:
- Monitor goroutine count trends
- Set appropriate memory limits
- Use context timeouts for all operations
- Regular code reviews for resource leaks

---

### 2. Readiness Check Failures

**Symptoms**:
- `/readyz` returns 503
- Alert: `ServiceNotReady`
- Kubernetes removing pod from service
- Traffic not reaching pod

**Possible Causes**:
1. Database connection failure
2. Redis connection failure
3. Downstream service unavailable
4. Network connectivity issues
5. Configuration errors

**Diagnosis**:
```bash
# Check full health status
curl http://<service>:8080/health | jq

# Check component-specific status
curl http://<service>:8080/health | jq '.components[] | select(.status != "healthy")'

# Check Prometheus metrics
# Query: component_status{service="<service>"}

# Check service logs for errors
kubectl logs <pod-name> -n <namespace> | grep -i "health\|error"
```

**Resolution by Component**:

#### Database Failures
```bash
# Check database connectivity
kubectl exec -it <pod-name> -n <namespace> -- sh
# Inside pod:
mysql -h <db-host> -u <user> -p<password> -e "SELECT 1"

# Check database status
kubectl get pods -n <db-namespace>

# Check auto-recovery attempts
# Prometheus query: rate(recovery_attempts_total{component="database"}[5m])
```

**Actions**:
1. Verify database is running
2. Check network policies/firewall rules
3. Verify credentials in secrets
4. Check connection pool exhaustion
5. Wait for auto-recovery (up to 60s)

#### Redis Failures
```bash
# Check Redis connectivity
kubectl exec -it <pod-name> -n <namespace> -- sh
# Inside pod:
redis-cli -h <redis-host> -p 6379 PING

# Check Redis status
kubectl get pods -n <redis-namespace>
```

**Actions**:
1. Verify Redis is running
2. Check network connectivity
3. Verify Redis password/auth
4. Check for Redis memory issues
5. Wait for auto-recovery (up to 60s)

#### Downstream Service Failures
```bash
# Check downstream service health
curl http://<downstream-service>:8080/healthz

# Check service discovery
kubectl get svc -n <namespace>
kubectl get endpoints <service-name> -n <namespace>
```

**Actions**:
1. Verify downstream service is healthy
2. Check service endpoints have ready pods
3. Verify network policies allow traffic
4. Check for circuit breaker open state

---

### 3. Circuit Breaker Open

**Symptoms**:
- Alert: `CircuitBreakerOpen`
- Component showing repeated failures
- Requests being rejected

**Diagnosis**:
```bash
# Check circuit breaker state
# Prometheus query: circuit_breaker_state{service="<service>",component="<component>"}
# 0 = Closed (normal)
# 1 = Open (failing)
# 2 = Half-Open (testing)

# Check failure count
# Prometheus query: rate(health_check_failures_total{component="<component>"}[5m])
```

**Resolution**:
1. **Identify root cause**: Check why component is failing
2. **Fix underlying issue**: Database down, network issue, etc.
3. **Wait for recovery**: Circuit breaker will test recovery automatically
4. **Monitor half-open state**: Circuit breaker will attempt recovery
5. **Verify closure**: Circuit breaker closes after successful checks

**Manual Intervention** (if auto-recovery fails):
```bash
# Restart the pod to reset circuit breaker
kubectl rollout restart deployment/<service> -n <namespace>
```

---

### 4. Auto-Recovery Failing

**Symptoms**:
- Alert: `AutoRecoveryFailing`
- Component stuck in failed state
- Recovery attempts not succeeding

**Diagnosis**:
```bash
# Check recovery attempts
# Prometheus query: rate(recovery_attempts_total{component="<component>"}[10m])

# Check recovery success rate
# Prometheus query: rate(recovery_success_total{component="<component>"}[10m])

# Check service logs for recovery errors
kubectl logs <pod-name> -n <namespace> | grep -i "recovery"
```

**Resolution**:
1. **Check dependency health**: Ensure database/Redis is actually healthy
2. **Check credentials**: Verify secrets are correct
3. **Check network**: Ensure connectivity to dependency
4. **Manual recovery**: Restart pod if auto-recovery stuck
5. **Escalate**: If issue persists after manual restart

---

### 5. Slow Health Checks

**Symptoms**:
- Alert: `SlowHealthCheck`
- Health check latency > 200ms
- Delayed failure detection

**Diagnosis**:
```bash
# Check response time metrics
# Prometheus query: histogram_quantile(0.99, component_response_time_seconds_bucket{service="<service>"})

# Check for slow components
curl http://<service>:8080/health | jq '.components[] | select(.response_time_ms > 100)'
```

**Resolution**:
1. **Database slow**: Check database performance, add indexes
2. **Redis slow**: Check Redis memory, consider scaling
3. **Network latency**: Check network between service and dependency
4. **Timeout too high**: Reduce health check timeout in config
5. **Resource contention**: Check CPU/memory usage

---

### 6. Readiness Flapping

**Symptoms**:
- Alert: `ReadinessFlapping`
- Pod repeatedly marked ready/not-ready
- Intermittent traffic issues

**Diagnosis**:
```bash
# Check status change frequency
# Prometheus query: changes(health_status{service="<service>"}[10m])

# Check for intermittent failures
kubectl logs <pod-name> -n <namespace> | grep "readiness"
```

**Possible Causes**:
1. Network instability
2. Dependency intermittent failures
3. Resource constraints (CPU throttling)
4. Health check timeout too aggressive

**Resolution**:
1. **Increase failure threshold**: Requires 3 consecutive failures (already configured)
2. **Increase timeout**: Adjust health check timeout in config
3. **Fix underlying instability**: Address network/resource issues
4. **Check anti-flapping logic**: Verify it's working correctly

---

## Metric Interpretation Guide

### Health Status Metrics

**`health_status`** - Overall service health
- `0` = Critical (not ready)
- `1` = Degraded (some components unhealthy)
- `2` = Healthy (all components healthy)

**`health_score`** - Health score percentage (0-100)
- `100` = All components healthy
- `50-99` = Some components degraded
- `0-49` = Multiple components critical

**`component_status`** - Individual component health
- `0` = Critical
- `1` = Degraded  
- `2` = Healthy

### Performance Metrics

**`component_response_time_seconds`** - Health check latency
- Target: P99 < 200ms
- Warning: P99 > 200ms
- Critical: P99 > 500ms

**`health_check_failures_total`** - Failure counter
- Normal: < 0.01 failures/sec
- Warning: > 0.1 failures/sec
- Critical: > 1 failure/sec

### Circuit Breaker Metrics

**`circuit_breaker_state`**
- `0` = Closed (normal operation)
- `1` = Open (failing, requests rejected)
- `2` = Half-Open (testing recovery)

---

## Manual Recovery Procedures

### Restart Service Pod

```bash
# Graceful restart (rolling update)
kubectl rollout restart deployment/<service> -n <namespace>

# Force delete pod (emergency)
kubectl delete pod <pod-name> -n <namespace> --force --grace-period=0
```

### Reset Circuit Breaker

Circuit breakers reset automatically, but if needed:

```bash
# Restart the pod (circuit breaker state is in-memory)
kubectl rollout restart deployment/<service> -n <namespace>
```

### Manually Mark Service Not Ready

If you need to drain traffic from a service:

```bash
# Scale down to 0 replicas
kubectl scale deployment/<service> -n <namespace> --replicas=0

# Or use pod disruption budget
kubectl drain <node-name> --ignore-daemonsets
```

### Force Service Ready (Emergency Only)

**WARNING**: Only use in emergencies when health checks are incorrectly failing

```bash
# Disable readiness probe temporarily
kubectl patch deployment/<service> -n <namespace> -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","readinessProbe":null}]}}}}'

# Remember to re-enable after fixing the issue!
```

---

## Escalation Paths

### Severity Levels

**P1 - Critical** (Immediate Response)
- Multiple services down
- Production traffic impacted
- Liveness failures causing pod restarts
- **Action**: Page on-call engineer immediately

**P2 - High** (Response within 30 minutes)
- Single service readiness failing
- Circuit breakers open
- Auto-recovery failing
- **Action**: Notify on-call engineer via Slack

**P3 - Medium** (Response within 2 hours)
- Component degraded
- Slow health checks
- Readiness flapping
- **Action**: Create ticket, notify team

**P4 - Low** (Response within 24 hours)
- Metrics anomalies
- Performance degradation
- **Action**: Create ticket

### Contact Information

- **On-Call Engineer**: See PagerDuty rotation
- **Platform Team Slack**: `#platform-engineering`
- **Alerts Channel**: `#alerts-health`
- **Incident Channel**: `#incidents`

### When to Escalate

Escalate to on-call engineer if:
1. Manual recovery procedures don't resolve issue
2. Multiple services affected
3. Production traffic impacted
4. Root cause unclear after 15 minutes
5. Issue recurring frequently

---

## Configuration Reference

### Health Check Configuration

Each service configures health checks via environment variables:

```yaml
# Liveness configuration
HEALTH_LIVENESS_ENABLED: "true"
HEALTH_LIVENESS_HEARTBEAT_TIMEOUT: "30s"
HEALTH_LIVENESS_MEMORY_THRESHOLD: "90"
HEALTH_LIVENESS_GOROUTINE_THRESHOLD: "10000"

# Readiness configuration
HEALTH_READINESS_ENABLED: "true"
HEALTH_READINESS_FAILURE_THRESHOLD: "3"
HEALTH_READINESS_CHECK_INTERVAL: "10s"

# Circuit breaker configuration
HEALTH_CIRCUIT_BREAKER_ENABLED: "true"
HEALTH_CIRCUIT_BREAKER_THRESHOLD: "3"
HEALTH_CIRCUIT_BREAKER_TIMEOUT: "60s"

# Auto-recovery configuration
HEALTH_RECOVERY_ENABLED: "true"
HEALTH_RECOVERY_MAX_RETRIES: "5"
HEALTH_RECOVERY_INITIAL_BACKOFF: "1s"
HEALTH_RECOVERY_MAX_BACKOFF: "60s"
```

### Kubernetes Probe Configuration

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
  successThreshold: 1
```

---

## Monitoring and Alerting

### Dashboard Locations

- **Grafana**: http://grafana.example.com/d/health-checks
- **Prometheus**: http://prometheus.example.com
- **AlertManager**: http://alertmanager.example.com

### Alert Routing

- **Critical Health Alerts** → PagerDuty + Slack `#alerts-health-critical`
- **Warning Health Alerts** → Slack `#alerts-health`
- **Circuit Breaker Alerts** → Slack `#alerts-circuit-breaker`
- **Performance Alerts** → Slack `#alerts-warnings`

### Key Queries

```promql
# Service health status
health_status{service="<service>"}

# Component health
component_status{service="<service>",component="<component>"}

# Health check latency (P99)
histogram_quantile(0.99, rate(component_response_time_seconds_bucket[5m]))

# Failure rate
rate(health_check_failures_total[5m])

# Circuit breaker state
circuit_breaker_state{service="<service>",component="<component>"}
```

---

## Maintenance Procedures

### Planned Maintenance

When performing planned maintenance on dependencies:

1. **Notify team** in `#platform-engineering` 24 hours in advance
2. **Create maintenance window** in PagerDuty
3. **Silence alerts** in AlertManager for affected services
4. **Monitor health dashboard** during maintenance
5. **Verify recovery** after maintenance complete
6. **Remove alert silences** after verification

### Updating Health Check Configuration

```bash
# Update configuration in deployment
kubectl edit deployment/<service> -n <namespace>

# Or update via ConfigMap
kubectl edit configmap/<service>-config -n <namespace>

# Restart pods to pick up new config
kubectl rollout restart deployment/<service> -n <namespace>

# Verify new configuration
kubectl logs <pod-name> -n <namespace> | grep "health config"
```

### Disabling Health Checks (Emergency)

If health checks are causing issues:

```bash
# Disable via environment variable
kubectl set env deployment/<service> -n <namespace> HEALTH_ENABLED=false

# Or remove probes from deployment
kubectl patch deployment/<service> -n <namespace> --type=json \
  -p='[{"op": "remove", "path": "/spec/template/spec/containers/0/readinessProbe"}]'
```

**Remember to re-enable after fixing the issue!**

---

## Troubleshooting Checklist

When investigating health check issues, work through this checklist:

- [ ] Check service logs for errors
- [ ] Verify `/health` endpoint shows component details
- [ ] Check Grafana dashboard for metrics
- [ ] Verify dependencies are healthy
- [ ] Check network connectivity
- [ ] Review recent deployments/changes
- [ ] Check resource usage (CPU/memory)
- [ ] Verify configuration is correct
- [ ] Check for circuit breaker open state
- [ ] Review auto-recovery attempts
- [ ] Check Kubernetes events
- [ ] Verify secrets/credentials are valid

---

## Additional Resources

- **Library Documentation**: `libs/health/README.md`
- **Service Integration Docs**: `apps/<service>/HEALTH_CHECK_INTEGRATION.md`
- **Architecture Docs**: `docs/architecture/OBSERVABILITY_SYSTEM.md`
- **Monitoring Guide**: `docs/operations/MONITORING_ALERTING_GUIDE.md`

---

## Changelog

| Date | Change | Author |
|------|--------|--------|
| 2026-02-02 | Initial runbook creation | Platform Team |

