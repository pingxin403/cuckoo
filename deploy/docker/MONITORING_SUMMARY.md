# IM Gateway Service - Monitoring and Operations Summary

## Overview

This document provides a comprehensive overview of the monitoring and operations setup for the IM Gateway Service, completed as part of Phase 5 (Task 16).

## Completed Components

### 1. OpenTelemetry Metrics (Task 16.1) ✅

**Implementation**:
- Migrated from custom Prometheus to OpenTelemetry metrics
- Integrated with `libs/observability` library
- Dual export: Prometheus (pull) + OTLP (push)

**Metrics Categories**:
- Connection metrics (active, total, errors)
- Message delivery metrics (delivered, failed, latency)
- Offline queue metrics (size, backlog)
- Cache metrics (hits, misses, hit rate)
- Deduplication metrics (duplicates detected)
- Multi-device metrics (deliveries)
- Group message metrics (delivered, fanout)

**Documentation**: `apps/im-gateway-service/metrics/README.md`

### 2. Grafana Dashboards (Task 16.2) ✅

**Dashboards Created**:

1. **Connection Metrics Dashboard** (`im-gateway-connections.json`)
   - Active WebSocket connections
   - Connection rate
   - Connection errors
   - Connection error rate

2. **Message Delivery Dashboard** (`im-gateway-messages.json`)
   - Message delivery rate (delivered vs failed)
   - P50, P95, P99 latency
   - ACK timeout rate
   - Message duplication rate
   - Offline queue backlog
   - Group message metrics

3. **System Health Dashboard** (`im-gateway-health.json`)
   - Service status (UP/DOWN)
   - Message delivery success rate
   - Cache hit rate
   - P99 latency
   - OTel metrics instrumentation
   - OTel metrics errors

4. **SLO Tracking Dashboard** (`im-gateway-slo.json`)
   - Availability SLO (99.95%)
   - Latency SLO (P99 < 200ms)
   - Success Rate SLO (99.99%)
   - Error budget consumption

**Access**: http://localhost:3000

### 3. Alerting Rules (Task 16.3) ✅

**Alert Categories**:

**Critical Alerts**:
- Service down
- High message loss rate (circuit breaker)
- Fast SLO burn
- Error budget 80% consumed

**P1 Alerts**:
- High P99 latency (> 500ms for 5 minutes)

**P2 Alerts**:
- High ACK timeout rate (> 5%)

**Warning Alerts**:
- High connection error rate
- Too many active connections
- High offline queue backlog
- Low cache hit rate
- High message duplication rate
- Slow SLO burn
- Error budget 50% consumed

**SLO Alerts**:
- Availability SLO violation
- Latency SLO violation
- Success rate SLO violation

**Configuration Files**:
- `prometheus-alerts.yml`: Alert rules
- `alertmanager-config.yml`: Alert routing and notification

**Documentation**: `ALERTING_GUIDE.md`, `ALERTING_QUICKSTART.md`

### 4. Centralized Logging (Task 16.4) ✅

**Architecture**:
- Services → OTLP Exporter → OTel Collector → Loki → Grafana

**Log Format**: Structured JSON with standard fields
- timestamp, level, service, trace_id, span_id
- message, user_id, device_id, msg_id
- latency_ms, component, event

**Log Categories**:
- Message delivery events (success, failure)
- Connection events (established, closed)
- Error events (auth failures, database errors)
- Performance events (slow queries, high latency)

**Retention Policies**:
- Development: 7 days
- Staging: 30 days
- Production: 90 days

**Documentation**: `CENTRALIZED_LOGGING.md`

### 5. SLO Tracking (Task 16.5) ✅

**Service Level Objectives**:

1. **Availability SLO**: 99.95% (21.6 minutes downtime/month)
2. **Latency SLO**: P99 < 200ms (99% of requests)
3. **Success Rate SLO**: 99.99% (0.01% loss rate)

**Error Budget**:
- Monthly budget: 21.6 minutes (0.05%)
- Alerts at 50% and 80% consumption
- Fast burn alert (14.4x rate, 1h window)
- Slow burn alert (6x rate, 6h window)

**SLO Dashboard**: Grafana dashboard with gauges for each SLO

**Documentation**: `SLO_TRACKING.md`

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    IM Gateway Service                        │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │ Metrics  │  │  Logs    │  │  Traces  │                 │
│  │ (OTel)   │  │ (OTel)   │  │ (OTel)   │                 │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                 │
└───────┼─────────────┼─────────────┼────────────────────────┘
        │             │             │
        │             │             │
        ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│              OpenTelemetry Collector                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │ Metrics  │  │  Logs    │  │  Traces  │                 │
│  │ Pipeline │  │ Pipeline │  │ Pipeline │                 │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                 │
└───────┼─────────────┼─────────────┼────────────────────────┘
        │             │             │
        ▼             ▼             ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│Prometheus│  │   Loki   │  │  Jaeger  │
│(Metrics) │  │  (Logs)  │  │ (Traces) │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │
     └─────────────┴─────────────┘
                   │
                   ▼
           ┌──────────────┐
           │   Grafana    │
           │ (Dashboards) │
           └──────────────┘
                   │
                   ▼
           ┌──────────────┐
           │ Alertmanager │
           │  (Alerts)    │
           └──────────────┘
                   │
                   ▼
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
   ┌────────┐          ┌──────────┐
   │ Slack  │          │PagerDuty │
   └────────┘          └──────────┘
```

## Access Points

### Web UIs

- **Grafana**: http://localhost:3000
  - Username: admin
  - Password: admin
  - Dashboards: Connections, Messages, Health, SLO

- **Prometheus**: http://localhost:9090
  - Metrics browser
  - Alert rules
  - Targets status

- **Alertmanager**: http://localhost:9093
  - Active alerts
  - Silences
  - Alert routing

- **Jaeger**: http://localhost:16686
  - Distributed tracing
  - Service dependencies
  - Trace search

- **Loki**: http://localhost:3100
  - Log ingestion endpoint
  - Query API

### Metrics Endpoints

- **IM Gateway Service**: http://localhost:9090/metrics
- **OTel Collector**: http://localhost:8888/metrics
- **Prometheus Exporter**: http://localhost:8889/metrics

## Quick Start

### 1. Start Observability Stack

```bash
cd deploy/docker

# Start all observability services
docker compose -f docker-compose.observability.yml up -d

# Verify all services are healthy
docker compose -f docker-compose.observability.yml ps
```

### 2. Verify Metrics Collection

```bash
# Check Prometheus is scraping metrics
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# Query a metric
curl 'http://localhost:9090/api/v1/query?query=im_gateway_active_connections' | jq '.data.result'
```

### 3. View Dashboards

```bash
# Open Grafana
open http://localhost:3000

# Navigate to Dashboards → Browse
# Select "IM Gateway - Connection Metrics"
```

### 4. Test Alerting

```bash
# View active alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | {name: .labels.alertname, state: .state}'

# Send test alert
curl -X POST http://localhost:9093/api/v1/alerts -H "Content-Type: application/json" -d '[
  {
    "labels": {"alertname": "TestAlert", "severity": "warning"},
    "annotations": {"summary": "Test alert"}
  }
]'
```

### 5. Query Logs

```bash
# Open Grafana Explore
open http://localhost:3000/explore

# Select Loki data source
# Query: {service_name="im-gateway-service"} |= "ERROR"
```

## Monitoring Best Practices

### Metrics

1. **Use OpenTelemetry SDK** for all metrics
2. **Export to multiple backends** (Prometheus + OTLP)
3. **Use histograms** for latency measurements
4. **Add labels** for filtering (user_id, device_id, etc.)
5. **Monitor instrumentation health** (OTel metrics)

### Dashboards

1. **Organize by concern** (connections, messages, health, SLO)
2. **Use appropriate visualizations** (gauges, time series, stats)
3. **Set meaningful thresholds** (green/yellow/red)
4. **Include links** to runbooks and related dashboards
5. **Refresh automatically** (10s - 1m intervals)

### Alerting

1. **Alert on symptoms, not causes** (high latency, not high CPU)
2. **Use multiple severity levels** (critical, P1, P2, warning)
3. **Set appropriate thresholds** (based on SLOs)
4. **Include runbook links** in alert annotations
5. **Use inhibition rules** to reduce alert noise

### Logging

1. **Use structured logging** (JSON format)
2. **Include trace context** (trace_id, span_id)
3. **Log at appropriate levels** (ERROR for failures, INFO for events)
4. **Add context** (user_id, msg_id, etc.)
5. **Don't log sensitive data** (passwords, tokens, PII)

### SLO Tracking

1. **Set realistic SLOs** (99.95% is achievable, 99.999% is not)
2. **Track error budgets** (monthly rolling windows)
3. **Alert on burn rate** (fast and slow burns)
4. **Review SLOs regularly** (weekly/monthly)
5. **Use error budgets** to prioritize work

## Troubleshooting

### Metrics Not Appearing

1. Check service is exporting metrics:
   ```bash
   curl http://localhost:9090/metrics | grep im_gateway
   ```

2. Check OTel Collector is receiving metrics:
   ```bash
   docker logs otel-collector | grep "metrics"
   ```

3. Check Prometheus is scraping:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

### Alerts Not Firing

1. Check alert rules are loaded:
   ```bash
   curl http://localhost:9090/api/v1/rules
   ```

2. Check alert evaluation:
   ```bash
   curl http://localhost:9090/api/v1/alerts
   ```

3. Check Alertmanager is receiving alerts:
   ```bash
   docker logs alertmanager
   ```

### Logs Not Appearing

1. Check OTel Collector is receiving logs:
   ```bash
   docker logs otel-collector | grep "logs"
   ```

2. Check Loki is healthy:
   ```bash
   curl http://localhost:3100/ready
   ```

3. Query Loki directly:
   ```bash
   curl 'http://localhost:3100/loki/api/v1/query?query={service_name="im-gateway-service"}'
   ```

## Next Steps

### Production Deployment

1. **Configure notification channels**:
   - Set up Slack webhooks
   - Configure PagerDuty integration
   - Set up email notifications

2. **Tune alert thresholds**:
   - Adjust based on production traffic
   - Reduce false positives
   - Ensure critical alerts are actionable

3. **Set up log retention**:
   - Configure Loki retention (90 days)
   - Set up log archival to S3
   - Implement log sampling for high volume

4. **Create runbooks**:
   - Document response procedures
   - Add troubleshooting steps
   - Include escalation paths

5. **Train team**:
   - Dashboard navigation
   - Alert response procedures
   - Log querying techniques
   - SLO tracking and reporting

### Continuous Improvement

1. **Monitor alert quality**:
   - Track false positive rate
   - Measure time to resolution
   - Gather feedback from on-call

2. **Optimize dashboards**:
   - Add new panels based on needs
   - Remove unused panels
   - Improve visualization clarity

3. **Review SLOs**:
   - Adjust targets based on reality
   - Add new SLOs as needed
   - Document SLO violations

4. **Enhance logging**:
   - Add more structured fields
   - Improve log correlation
   - Implement log sampling

## Documentation

- **Metrics**: `apps/im-gateway-service/metrics/README.md`
- **Alerting**: `deploy/docker/ALERTING_GUIDE.md`
- **Alerting Quick Start**: `deploy/docker/ALERTING_QUICKSTART.md`
- **Centralized Logging**: `deploy/docker/CENTRALIZED_LOGGING.md`
- **SLO Tracking**: `deploy/docker/SLO_TRACKING.md`
- **Observability Stack**: `deploy/docker/OBSERVABILITY.md`

## Support

For questions or issues:
- **Slack**: #observability-support
- **Email**: observability-team@example.com
- **On-call**: PagerDuty escalation policy
- **Documentation**: https://wiki.example.com/observability
