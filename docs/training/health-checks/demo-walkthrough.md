# Health Check System - Demo Walkthrough

This document provides a step-by-step demo script for presenting the health check system.

**Duration**: 15-20 minutes  
**Prerequisites**: Docker Compose environment running

---

## Setup

### Before the Demo

1. Start the infrastructure:
```bash
cd deploy/docker
docker-compose -f docker-compose.infra.yml up -d
docker-compose -f docker-compose.observability.yml up -d
```

2. Start a service (e.g., shortener-service):
```bash
cd apps/shortener-service
go run .
```

3. Open browser tabs:
- Grafana: http://localhost:3000 (Health Check Monitoring dashboard)
- Prometheus: http://localhost:9090
- Service health endpoint: http://localhost:8080/health

---

## Demo Script

### Part 1: Healthy Service (3 minutes)

**Narration**: "Let's start by looking at a healthy service."

#### Step 1: Check Health Endpoints

```bash
# Liveness check
curl http://localhost:8080/healthz
# Expected: 200 OK

# Readiness check
curl http://localhost:8080/readyz
# Expected: 200 OK

# Full health status
curl http://localhost:8080/health | jq
```

**Point out**:
- Status: "healthy"
- Health score: 100
- All components showing "healthy"
- Response times < 10ms

#### Step 2: Show Grafana Dashboard

**Navigate to**: Grafana → Health Check Monitoring dashboard

**Point out**:
- Overall health status gauge (green, "Healthy")
- Health score at 100%
- Component status table (all green)
- Response time graph (low latency)
- Failure rate graph (zero failures)

**Narration**: "This is what a healthy service looks like. All components are green, health score is 100%, and response times are low."

---

### Part 2: Database Failure (5 minutes)

**Narration**: "Now let's simulate a database failure and see how the system responds."

#### Step 1: Stop MySQL

```bash
docker stop mysql
```

**Narration**: "I've just stopped the MySQL container. Let's watch what happens."

#### Step 2: Observe Health Check Response (wait 10-15 seconds)

```bash
# Check readiness (should fail after 3 consecutive failures)
curl http://localhost:8080/readyz
# Expected: 503 Service Unavailable (after ~15 seconds)

# Check full health status
curl http://localhost:8080/health | jq
```

**Point out**:
- Status changed to "critical"
- Health score dropped (e.g., 50%)
- Database component showing "critical"
- Error message: "connection refused" or similar
- Other components still healthy

#### Step 3: Show Grafana Dashboard

**Refresh Grafana dashboard**

**Point out**:
- Overall health status gauge (red, "Critical")
- Health score dropped
- Component status table shows database as red
- Failure rate graph shows spike
- Alert may be firing (check Prometheus alerts)

#### Step 4: Check Prometheus Alerts

**Navigate to**: Prometheus → Alerts

**Point out**:
- `ServiceNotReady` alert firing (or pending)
- Alert shows service name and component
- Alert includes runbook link

**Narration**: "The system detected the failure within 15 seconds. The readiness probe failed, which means Kubernetes would remove this pod from the load balancer. An alert has been triggered to notify the on-call engineer."

---

### Part 3: Auto-Recovery (5 minutes)

**Narration**: "Now let's see the auto-recovery feature in action."

#### Step 1: Restart MySQL

```bash
docker start mysql
```

**Narration**: "I've restarted MySQL. The health check system will automatically attempt to reconnect."

#### Step 2: Watch Recovery (wait 30-60 seconds)

```bash
# Monitor health status
watch -n 2 'curl -s http://localhost:8080/health | jq ".components[] | select(.name==\"database\")"'
```

**Point out**:
- Initial state: "critical"
- Auto-recovery attempts (check logs)
- Exponential backoff (1s, 2s, 4s, 8s...)
- Connection restored
- Status changes to "healthy"

#### Step 3: Verify Full Recovery

```bash
# Check readiness
curl http://localhost:8080/readyz
# Expected: 200 OK

# Check full health
curl http://localhost:8080/health | jq
```

**Point out**:
- Status: "healthy"
- Health score: 100
- Database component: "healthy"
- No manual intervention required

#### Step 4: Show Grafana Dashboard

**Refresh Grafana dashboard**

**Point out**:
- Overall health status back to green
- Health score recovered to 100%
- Component status table shows database as green
- Response time graph shows brief spike during recovery
- Failure rate returned to zero

**Narration**: "The service automatically recovered without any manual intervention. This is the power of auto-recovery - it handles transient failures automatically, reducing the on-call burden."

---

### Part 4: Circuit Breaker (4 minutes)

**Narration**: "Let's look at the circuit breaker pattern in action."

#### Step 1: Check Circuit Breaker Metrics

```bash
# Query Prometheus for circuit breaker state
curl -s 'http://localhost:9090/api/v1/query?query=circuit_breaker_state{service="shortener-service"}' | jq
```

**Point out**:
- Circuit breaker state: 0 (Closed) when healthy
- Circuit breaker state: 1 (Open) when failing
- Circuit breaker state: 2 (Half-Open) during recovery

#### Step 2: Simulate Repeated Failures

```bash
# Stop MySQL again
docker stop mysql

# Wait for circuit breaker to open (after 3 failures)
sleep 20

# Check circuit breaker state
curl -s 'http://localhost:9090/api/v1/query?query=circuit_breaker_state{service="shortener-service",component="database"}' | jq
```

**Point out**:
- Circuit breaker opened after 3 consecutive failures
- Prevents resource exhaustion
- Fails fast instead of hanging

#### Step 3: Show Recovery Testing

```bash
# Restart MySQL
docker start mysql

# Watch circuit breaker transition to half-open, then closed
watch -n 2 'curl -s "http://localhost:9090/api/v1/query?query=circuit_breaker_state{service=\"shortener-service\",component=\"database\"}" | jq ".data.result[0].value[1]"'
```

**Point out**:
- Circuit breaker transitions to half-open (2)
- Tests recovery with single request
- Closes (0) on success
- Automatic recovery testing

**Narration**: "The circuit breaker prevents cascading failures and automatically tests for recovery. This protects both the service and its dependencies."

---

### Part 5: Monitoring and Metrics (3 minutes)

**Narration**: "Let's explore the monitoring capabilities."

#### Step 1: Prometheus Metrics

**Navigate to**: Prometheus → Graph

**Run queries**:

```promql
# Overall health status
health_status{service="shortener-service"}

# Component health
component_status{service="shortener-service"}

# Health check latency (P99)
histogram_quantile(0.99, rate(component_response_time_seconds_bucket{service="shortener-service"}[5m]))

# Failure rate
rate(health_check_failures_total{service="shortener-service"}[5m])
```

**Point out**:
- Real-time metrics
- Historical data
- Multiple dimensions (service, component)
- Percentile calculations

#### Step 2: Grafana Dashboard Features

**Navigate to**: Grafana → Health Check Monitoring

**Demonstrate**:
- Service filter dropdown (select specific service)
- Time range selector (last 1h, 6h, 24h)
- Auto-refresh (10 seconds)
- Panel drill-down (click on component)
- Export dashboard JSON

**Point out**:
- Customizable views
- Real-time updates
- Historical trends
- Easy troubleshooting

---

## Cleanup

After the demo:

```bash
# Stop services
docker-compose -f docker-compose.infra.yml down
docker-compose -f docker-compose.observability.yml down

# Or keep running for Q&A
```

---

## Q&A Preparation

### Common Questions

**Q: What happens if health checks are too slow?**
A: Alert `SlowHealthCheck` fires. Investigate component performance, adjust timeouts, or optimize checks.

**Q: Can we disable health checks temporarily?**
A: Yes, but not recommended. Set `HEALTH_ENABLED=false` environment variable. Always re-enable after debugging.

**Q: How do we add custom health checks?**
A: Implement the `Check` interface. See `libs/health/README.md` for examples.

**Q: What if auto-recovery fails?**
A: Alert `AutoRecoveryFailing` fires. Manual intervention required. Check runbook for specific component.

**Q: How do we test health checks in development?**
A: Use Docker Compose to simulate failures. Run integration tests with `./scripts/run-integration-tests.sh`.

---

## Tips for Presenters

### Do's ✅
- Practice the demo beforehand
- Have backup terminal windows ready
- Explain what you're doing before each step
- Point out key observations
- Relate to real-world scenarios
- Encourage questions throughout

### Don'ts ❌
- Don't rush through steps
- Don't skip error messages
- Don't assume everyone knows Docker/Kubernetes
- Don't ignore questions
- Don't go too deep into implementation details (save for Q&A)

### Troubleshooting During Demo

**If service won't start:**
- Check if ports are already in use
- Verify environment variables are set
- Check Docker containers are running

**If health checks don't fail:**
- Verify you stopped the correct container
- Wait longer (anti-flapping requires 3 failures)
- Check service logs for errors

**If Grafana doesn't show data:**
- Verify Prometheus is scraping metrics
- Check time range in Grafana
- Refresh the dashboard

---

## Additional Demo Scenarios

### Scenario: Redis Failure (Optional)

```bash
# Stop Redis
docker stop redis

# Observe graceful degradation
curl http://localhost:8080/health | jq

# Service continues operating without cache
# Readiness may remain healthy (Redis is non-critical)

# Restart Redis
docker start redis
```

### Scenario: High Load (Optional)

```bash
# Generate load
hey -z 30s -c 100 http://localhost:8080/abc1234

# Monitor health check performance
curl http://localhost:8080/health | jq '.components[] | .response_time_ms'

# Health checks should remain fast (< 200ms)
```

### Scenario: Readiness Middleware (Optional)

```bash
# Stop MySQL
docker stop mysql

# Wait for readiness to fail
sleep 20

# Try to make requests
curl http://localhost:8080/abc1234
# Expected: 503 Service Unavailable

# Restart MySQL
docker start mysql

# Wait for recovery
sleep 30

# Requests work again
curl http://localhost:8080/abc1234
# Expected: 302 redirect
```

---

## Resources for Demo

- [Health Check Library](../../../libs/health/README.md)
- [Operational Runbook](../../operations/health-check-runbook.md)
- [Grafana Dashboard JSON](../../../deploy/docker/grafana/dashboards/health-checks.json)
- [Prometheus Alerts](../../../deploy/docker/prometheus-health-alerts.yml)

