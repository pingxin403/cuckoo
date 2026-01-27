# IM Chat System - Monitoring and Alerting Guide

**Last Updated**: 2026-01-25  
**Version**: 1.0  
**Maintained By**: Operations Team

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Quick Reference](#quick-reference)
3. [Monitoring Stack](#monitoring-stack)
4. [Key Metrics](#key-metrics)
5. [Alerting System](#alerting-system)
6. [Centralized Logging](#centralized-logging)
7. [SLO Tracking](#slo-tracking)
8. [Dashboards](#dashboards)
9. [Troubleshooting](#troubleshooting)
10. [Operational Procedures](#operational-procedures)
11. [Best Practices](#best-practices)
12. [Support](#support)

---

## Overview

This comprehensive guide covers the complete monitoring and alerting system for the IM Chat System. It integrates information from multiple sources to provide a single reference for operations teams.

### Purpose

- Provide a unified reference for monitoring and alerting
- Enable quick incident response
- Support proactive system health management
- Track Service Level Objectives (SLOs)

### Audience

- On-call engineers
- Site Reliability Engineers (SREs)
- DevOps teams
- System administrators

### Related Documentation

- [Alerting Guide](./ALERTING_GUIDE.md) - Detailed alert rules and response procedures
- [Centralized Logging](./CENTRALIZED_LOGGING.md) - Log aggregation and querying
- [SLO Tracking](./SLO_TRACKING.md) - Service Level Objectives and error budgets
- [Operational Runbooks](./OPERATIONAL_RUNBOOKS.md) - Step-by-step incident response procedures

---

## Quick Reference

### 🚨 Emergency Contacts

| Role | Contact | Response Time |
|------|---------|---------------|
| Primary On-Call | PagerDuty | < 5 minutes |
| Secondary On-Call | PagerDuty | < 15 minutes |
| Operations Team | #ops-team (Slack) | < 1 hour |
| SRE Team | #sre-team (Slack) | < 4 hours |

### 🔗 Quick Links

| Resource | URL | Purpose |
|----------|-----|---------|
| Grafana | http://localhost:3000 (dev)<br/>https://grafana.example.com (prod) | Dashboards and visualization |
| Prometheus | http://localhost:9090 (dev)<br/>https://prometheus.example.com (prod) | Metrics and alerts |
| Alertmanager | http://localhost:9093 (dev)<br/>https://alertmanager.example.com (prod) | Alert management |
| Jaeger | http://localhost:16686 (dev)<br/>https://jaeger.example.com (prod) | Distributed tracing |
| Loki | Grafana Explore | Log aggregation |

### 📊 Key Dashboards

1. **IM Gateway Connections** - Active connections, connection rate, errors
2. **IM Gateway Messages** - Message throughput, latency, delivery status
3. **IM Gateway Health** - CPU, memory, network, service health
4. **IM Gateway SLO** - Availability, latency, success rate, error budget

### ⚡ Critical Alerts

| Alert | Severity | Response Time | Runbook |
|-------|----------|---------------|---------|
| IMGatewayServiceDown | Critical | < 5 min | [Service Down](#service-down-critical) |
| HighMessageLossRate | Critical | < 5 min | [Message Loss](#message-loss-critical) |
| HighMessageDeliveryLatency | P1 | < 15 min | [High Latency](#high-latency-p1) |
| HighAckTimeoutRate | P2 | < 1 hour | [ACK Timeout](#ack-timeout-p2) |

### 🎯 Service Level Objectives (SLOs)

| SLO | Target | Current | Error Budget |
|-----|--------|---------|--------------|
| Availability | 99.95% | Check Grafana | 21.6 min/month |
| P99 Latency | < 200ms | Check Grafana | 1% requests |
| Success Rate | 99.99% | Check Grafana | 0.01% loss |

---

## Monitoring Stack

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     IM Chat Services                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Gateway  │  │ IM       │  │ Auth     │  │ User     │   │
│  │ Service  │  │ Service  │  │ Service  │  │ Service  │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
└───────┼─────────────┼─────────────┼─────────────┼──────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                      │
                      ▼
        ┌─────────────────────────┐
        │  OpenTelemetry          │
        │  Collector              │
        │  (Metrics, Logs, Traces)│
        └────┬──────────┬─────────┘
             │          │
    ┌────────▼──┐   ┌──▼─────────┐
    │ Prometheus│   │ Loki       │
    │ (Metrics) │   │ (Logs)     │
    └────┬──────┘   └──┬─────────┘
         │             │
         └──────┬──────┘
                ▼
        ┌───────────────┐
        │ Grafana       │
        │ (Visualization)│
        └───────────────┘
                │
                ▼
        ┌───────────────┐
        │ Alertmanager  │
        │ (Alerting)    │
        └───────────────┘
```

### Components

#### 1. OpenTelemetry Collector

**Purpose**: Unified telemetry collection and processing

**Configuration**: `deploy/docker/otel-collector-config.yaml`

**Features**:
- Receives metrics, logs, and traces from services
- Processes and transforms telemetry data
- Exports to Prometheus, Loki, and Jaeger
- Supports multiple protocols (OTLP, Prometheus, Jaeger)

**Health Check**:
```bash
curl http://localhost:13133/
```

#### 2. Prometheus

**Purpose**: Metrics collection and storage

**Configuration**: `deploy/docker/prometheus.yml`

**Features**:
- Scrapes metrics from services every 15 seconds
- Stores time-series data with 15-day retention
- Evaluates alert rules
- Provides PromQL query language

**Health Check**:
```bash
curl http://localhost:9090/-/healthy
```

**Key Metrics**:
- `im_gateway_active_connections` - Active WebSocket connections
- `im_gateway_message_delivery_latency_seconds` - Message delivery latency histogram
- `im_gateway_messages_delivered_total` - Total messages delivered
- `im_gateway_messages_failed_total` - Total messages failed
- `im_gateway_ack_timeout_total` - Total ACK timeouts

#### 3. Loki

**Purpose**: Log aggregation and storage

**Configuration**: `deploy/docker/loki-config.yaml`

**Features**:
- Stores structured logs in JSON format
- 90-day retention policy
- Efficient log querying with LogQL
- Integration with Grafana

**Health Check**:
```bash
curl http://localhost:3100/ready
```

#### 4. Grafana

**Purpose**: Visualization and dashboards

**Configuration**: `deploy/docker/grafana/`

**Features**:
- Pre-configured dashboards for all services
- Alerting and notification
- Log exploration with Loki
- Trace visualization with Jaeger

**Access**:
- URL: http://localhost:3000
- Default credentials: admin/admin (change on first login)

#### 5. Alertmanager

**Purpose**: Alert routing and notification

**Configuration**: `deploy/docker/alertmanager-config.yml`

**Features**:
- Routes alerts to appropriate channels
- Deduplicates and groups alerts
- Supports Slack, PagerDuty, Email
- Silencing and inhibition rules

**Health Check**:
```bash
curl http://localhost:9093/-/healthy
```

---

## Key Metrics

### Connection Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `im_gateway_active_connections` | Gauge | Current active WebSocket connections | > 100,000 (Warning) |
| `im_gateway_connection_errors_total` | Counter | Total connection errors | > 10% rate (Warning) |
| `im_gateway_connections_established_total` | Counter | Total connections established | - |
| `im_gateway_connections_closed_total` | Counter | Total connections closed | - |

**Key Queries**:
```promql
# Active connections
im_gateway_active_connections

# Connection error rate
rate(im_gateway_connection_errors_total[5m]) / rate(im_gateway_connections_established_total[5m]) * 100

# Connection churn rate
rate(im_gateway_connections_established_total[5m])
```

### Message Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `im_gateway_messages_delivered_total` | Counter | Total messages delivered | - |
| `im_gateway_messages_failed_total` | Counter | Total messages failed | > 0.01% rate (Critical) |
| `im_gateway_message_delivery_latency_seconds` | Histogram | Message delivery latency | P99 > 500ms (P1) |
| `im_gateway_ack_timeout_total` | Counter | Total ACK timeouts | > 5% rate (P2) |
| `im_gateway_message_duplication_total` | Counter | Total duplicate messages | > 1% rate (Warning) |

**Key Queries**:
```promql
# Message delivery rate
rate(im_gateway_messages_delivered_total[5m])

# Message loss rate
rate(im_gateway_messages_failed_total[5m]) / (rate(im_gateway_messages_delivered_total[5m]) + rate(im_gateway_messages_failed_total[5m])) * 100

# P99 latency
histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) * 1000

# ACK timeout rate
rate(im_gateway_ack_timeout_total[5m]) / rate(im_gateway_messages_delivered_total[5m]) * 100
```

### System Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `process_cpu_seconds_total` | Counter | CPU usage | > 80% (Warning) |
| `process_resident_memory_bytes` | Gauge | Memory usage | > 80% limit (Warning) |
| `go_goroutines` | Gauge | Number of goroutines | > 10,000 (Warning) |
| `up` | Gauge | Service availability | 0 (Critical) |

**Key Queries**:
```promql
# CPU usage percentage
rate(process_cpu_seconds_total[5m]) * 100

# Memory usage percentage
process_resident_memory_bytes / process_virtual_memory_max_bytes * 100

# Service uptime
up{job="im-gateway-service"}
```

### Cache Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `im_gateway_cache_hits_total` | Counter | Total cache hits | - |
| `im_gateway_cache_misses_total` | Counter | Total cache misses | - |
| `im_gateway_cache_size` | Gauge | Current cache size | - |

**Key Queries**:
```promql
# Cache hit rate
rate(im_gateway_cache_hits_total[5m]) / (rate(im_gateway_cache_hits_total[5m]) + rate(im_gateway_cache_misses_total[5m])) * 100
```

### Offline Queue Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| `im_service_offline_queue_size` | Gauge | Offline message queue size | > 10,000 (Warning) |
| `im_service_offline_worker_lag` | Gauge | Kafka consumer lag | > 1,000 (Warning) |

---

## Alerting System

### Alert Severity Levels

| Severity | Response Time | Notification | Escalation | Examples |
|----------|---------------|--------------|------------|----------|
| **Critical** | < 5 minutes | PagerDuty + Slack + Email | After 15 min | Service down, message loss |
| **P1** | < 15 minutes | PagerDuty + Slack | After 30 min | High latency, degraded performance |
| **P2** | < 1 hour | Slack | After 2 hours | ACK timeout, cache issues |
| **Warning** | < 4 hours | Slack | Business hours | Low cache hit rate, high connections |

### Critical Alerts

#### Service Down (Critical)

**Alert**: `IMGatewayServiceDown`

**Trigger**: Service not responding to health checks for 1 minute

**Impact**: Service unavailable, all connections lost

**Response**:
1. Check service status: `docker ps | grep im-gateway`
2. Check logs: `docker logs im-gateway-service --tail 100`
3. Check resource usage: `docker stats im-gateway-service`
4. Restart if crashed: `docker restart im-gateway-service`
5. Escalate if persistent

**Runbook**: [Service Down](./OPERATIONAL_RUNBOOKS.md#runbook-1-handle-gateway-node-failure)

#### Message Loss (Critical)

**Alert**: `HighMessageLossRate`

**Trigger**: Message loss rate > 0.01% for 2 minutes

**Impact**: Messages being lost, SLO violation

**Response**:
1. **IMMEDIATE**: Trigger circuit breaker
2. Check infrastructure health (MySQL, Redis, Kafka)
3. Check service logs for errors
4. Verify offline channel routing
5. Disable circuit breaker once healthy

**Runbook**: [Message Loss](./ALERTING_GUIDE.md#2-highmessagelossrate-critical---circuit-breaker)

### P1 Alerts

#### High Latency (P1)

**Alert**: `HighMessageDeliveryLatency`

**Trigger**: P99 latency > 500ms for 5 minutes

**Impact**: Users experience slow message delivery

**Response**:
1. Check Grafana dashboard
2. Review service logs
3. Check CPU/memory usage
4. Check database connection pool
5. Check Redis latency
6. Consider scaling out

**Runbook**: [High Latency](./ALERTING_GUIDE.md#1-highmessagedeliverylatency-p1)

### P2 Alerts

#### ACK Timeout (P2)

**Alert**: `HighAckTimeoutRate`

**Trigger**: ACK timeout rate > 5% for 5 minutes

**Impact**: Many messages not acknowledged by clients

**Response**:
1. Check Grafana dashboard
2. Review client-side logs
3. Check WebSocket stability
4. Review ACK timeout configuration
5. Consider increasing timeout

**Runbook**: [ACK Timeout](./ALERTING_GUIDE.md#3-highacktimeoutrate-p2)

### Warning Alerts

#### High Connections (Warning)

**Alert**: `TooManyActiveConnections`

**Trigger**: Active connections > 100,000 for 5 minutes

**Impact**: Gateway approaching capacity

**Response**:
1. Check connection growth trend
2. Verify legitimate traffic
3. Check for connection leaks
4. Scale out gateway nodes
5. Monitor distribution

**Runbook**: [Scale Out](./ALERTING_GUIDE.md#5-toomanyactiveconnections-warning)

### Alert Configuration

**File**: `deploy/docker/prometheus-alerts.yml`

**Example Alert Rule**:
```yaml
- alert: HighMessageDeliveryLatency
  expr: histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) > 0.5
  for: 5m
  labels:
    severity: P1
    service: im-gateway-service
  annotations:
    summary: "High P99 message delivery latency"
    description: "P99 latency is {{ $value }}s (threshold: 0.5s)"
    dashboard: "http://grafana:3000/d/im-gateway-messages"
```

### Notification Channels

#### Slack

**Configuration**: `deploy/docker/alertmanager-config.yml`

**Channels**:
- `#alerts-critical` - Critical and P1 alerts
- `#alerts-p2` - P2 alerts
- `#alerts-warnings` - Warning alerts

**Setup**:
1. Create Slack webhook
2. Update `slack_api_url` in config
3. Restart Alertmanager

#### PagerDuty

**Configuration**: `deploy/docker/alertmanager-config.yml`

**Setup**:
1. Create PagerDuty service integration
2. Add service key to config
3. Configure escalation policy
4. Restart Alertmanager

**Escalation**:
- Primary on-call: Immediate
- Secondary on-call: After 15 minutes
- Manager: After 30 minutes

#### Email

**Configuration**: `deploy/docker/alertmanager-config.yml`

**Setup**:
1. Configure SMTP settings
2. Add email addresses
3. Restart Alertmanager

### Alert Testing

**Test High Latency**:
```bash
curl -X POST http://im-gateway-service:8080/test/latency?delay=600ms
```

**Test Message Loss**:
```bash
curl -X POST http://im-gateway-service:8080/test/failure-rate?rate=0.02
```

**Test Service Down**:
```bash
docker stop im-gateway-service
# Wait 1 minute
docker start im-gateway-service
```

### Alert Tuning

**Adjust Thresholds**:
```yaml
# Edit prometheus-alerts.yml
- alert: HighMessageDeliveryLatency
  expr: histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[5m])) > 1.0  # Changed from 0.5
```

**Silence Alerts** (during maintenance):
```bash
amtool silence add alertname=~".+" service="im-gateway-service" --duration=2h --comment="Maintenance"
```

---

## Centralized Logging

### Log Architecture

```
Services → OTLP Exporter → OTel Collector → Loki → Grafana
```

### Log Format

All logs use structured JSON format:

```json
{
  "timestamp": "2025-01-25T10:30:45.123Z",
  "level": "INFO",
  "service": "im-gateway-service",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "message": "Message delivered successfully",
  "user_id": "user123",
  "device_id": "device456",
  "msg_id": "msg789",
  "latency_ms": 45
}
```

### Log Levels

- **DEBUG**: Detailed diagnostic (disabled in production)
- **INFO**: General informational messages
- **WARN**: Potentially harmful situations
- **ERROR**: Error events
- **FATAL**: Critical errors causing shutdown

### Log Retention

- **Development**: 7 days
- **Staging**: 30 days
- **Production**: 90 days

### Querying Logs

**Access**: Grafana → Explore → Select "Loki"

**Common Queries**:

```logql
# All logs from gateway service
{service_name="im-gateway-service"}

# Error logs only
{service_name="im-gateway-service"} |= "ERROR"

# Message delivery failures
{service_name="im-gateway-service"} | json | event="message_delivery_failed"

# Logs for specific user
{service_name="im-gateway-service"} | json | user_id="user123"

# High latency events
{service_name="im-gateway-service"} | json | event="high_latency" | latency_ms > 500

# Logs with trace ID
{service_name="im-gateway-service"} | json | trace_id="abc123..."
```

### Log Aggregation

**Count errors by type**:
```logql
sum by (error) (count_over_time({service_name="im-gateway-service"} |= "ERROR" [5m]))
```

**Average latency**:
```logql
avg_over_time({service_name="im-gateway-service"} | json | latency_ms != "" | unwrap latency_ms [5m])
```

**Top users by message count**:
```logql
topk(10, sum by (user_id) (count_over_time({service_name="im-gateway-service"} | json | event="message_delivered" [1h])))
```

### Log Categories

#### 1. Message Delivery Events

**Success**:
```json
{
  "level": "INFO",
  "event": "message_delivered",
  "user_id": "user123",
  "msg_id": "msg789",
  "latency_ms": 45
}
```

**Failure**:
```json
{
  "level": "ERROR",
  "event": "message_delivery_failed",
  "user_id": "user123",
  "msg_id": "msg789",
  "error": "connection timeout",
  "retry_count": 3
}
```

#### 2. Connection Events

**Established**:
```json
{
  "level": "INFO",
  "event": "connection_established",
  "user_id": "user123",
  "device_id": "device456",
  "remote_addr": "192.168.1.100"
}
```

**Closed**:
```json
{
  "level": "INFO",
  "event": "connection_closed",
  "user_id": "user123",
  "reason": "client_disconnect",
  "duration_seconds": 3600
}
```

#### 3. Error Events

**Authentication Failure**:
```json
{
  "level": "WARN",
  "event": "auth_failed",
  "remote_addr": "192.168.1.100",
  "error": "invalid_token"
}
```

**Database Error**:
```json
{
  "level": "ERROR",
  "event": "database_error",
  "operation": "insert_offline_message",
  "error": "connection pool exhausted"
}
```

### Integration with Tracing

Logs are correlated with traces using `trace_id`:

1. Find slow request in Jaeger
2. Copy trace ID
3. Query logs in Grafana:
   ```logql
   {service_name="im-gateway-service"} | json | trace_id="<trace_id>"
   ```

### Troubleshooting Logs

**Logs not appearing**:
1. Check OTel Collector: `docker logs otel-collector | grep "logs"`
2. Check Loki health: `curl http://localhost:3100/ready`
3. Verify log export: `docker logs im-gateway-service | grep "log export"`

**High log volume**:
1. Enable log sampling
2. Reduce DEBUG logs
3. Use log aggregation
4. Adjust retention period

---

## SLO Tracking

### Service Level Objectives

#### 1. Availability SLO

**Target**: 99.95% (21.6 minutes downtime per month)

**Measurement**:
```promql
avg_over_time(up{job="im-gateway-service"}[30d]) * 100
```

**Dashboard**: IM Gateway SLO → Availability Panel

**Error Budget**: 0.05% (21.6 minutes/month)

#### 2. Latency SLO

**Target**: P99 < 200ms

**Measurement**:
```promql
histogram_quantile(0.99, rate(im_gateway_message_delivery_latency_seconds_bucket[30d])) * 1000
```

**Dashboard**: IM Gateway SLO → Latency Panel

**Error Budget**: 1% of requests can exceed 200ms

#### 3. Success Rate SLO

**Target**: 99.99% (0.01% loss rate)

**Measurement**:
```promql
(sum(rate(im_gateway_messages_delivered_total[30d])) / 
 (sum(rate(im_gateway_messages_delivered_total[30d])) + 
  sum(rate(im_gateway_messages_failed_total[30d])))) * 100
```

**Dashboard**: IM Gateway SLO → Success Rate Panel

**Error Budget**: 0.01% message loss allowed

### Error Budget

**Current Consumption**:
```promql
(1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 * 100
```

**Remaining Budget**:
```promql
(0.0005 - (1 - avg_over_time(up{job="im-gateway-service"}[30d]))) * 43200
```

### Error Budget Alerts

**50% Consumed** (Warning):
```yaml
- alert: ErrorBudget50PercentConsumed
  expr: (1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 > 0.5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "50% of monthly error budget consumed"
```

**80% Consumed** (Critical):
```yaml
- alert: ErrorBudget80PercentConsumed
  expr: (1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 > 0.8
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "80% of error budget consumed - Circuit breaker recommended"
```

### SLO Burn Rate

**Fast Burn** (1 hour window):
```yaml
- alert: SLOFastBurn
  expr: |
    (1 - (sum(rate(im_gateway_messages_delivered_total[1h])) /
          (sum(rate(im_gateway_messages_delivered_total[1h])) + 
           sum(rate(im_gateway_messages_failed_total[1h]))))) > (0.0001 * 14.4)
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Fast SLO burn - 5% of monthly budget in 1 hour"
```

**Slow Burn** (6 hour window):
```yaml
- alert: SLOSlowBurn
  expr: |
    (1 - (sum(rate(im_gateway_messages_delivered_total[6h])) /
          (sum(rate(im_gateway_messages_delivered_total[6h])) + 
           sum(rate(im_gateway_messages_failed_total[6h]))))) > (0.0001 * 6)
  for: 15m
  labels:
    severity: warning
  annotations:
    summary: "Slow SLO burn - 2.5% of monthly budget in 6 hours"
```

### SLO Review Process

**Weekly Review**:
1. Check current SLO compliance
2. Review error budget consumption
3. Identify incidents that consumed budget
4. Plan improvements if needed

**Monthly Review**:
1. Calculate final SLO achievement
2. Document any SLO violations
3. Analyze root causes
4. Update SLO targets if needed
5. Reset error budget for new month

### Monthly SLO Report Template

```markdown
# IM Gateway Service - Monthly SLO Report
**Month**: January 2025

## SLO Achievement
- Availability: 99.97% ✅ (Target: 99.95%)
- Latency (P99): 185ms ✅ (Target: < 200ms)
- Success Rate: 99.995% ✅ (Target: 99.99%)

## Error Budget
- Consumed: 6.5 minutes (30% of budget)
- Remaining: 15.1 minutes (70% of budget)

## Incidents
1. Database outage (Jan 15): 3 minutes downtime
2. High latency spike (Jan 22): 2.5 minutes above threshold
3. Network issue (Jan 28): 1 minute downtime

## Actions
- Improve database failover time
- Add caching to reduce latency
- Implement better network monitoring

## Next Month Goals
- Maintain 99.95% availability
- Reduce P99 latency to < 150ms
- Improve error budget utilization
```

---

## Dashboards

### 1. IM Gateway Connections Dashboard

**File**: `deploy/docker/grafana/dashboards/im-gateway-connections.json`

**Panels**:
- Active Connections (Gauge)
- Connection Rate (Graph)
- Connection Errors (Graph)
- Connection Duration (Histogram)
- Top Users by Connections (Table)

**Key Metrics**:
- Current active connections
- Connection establishment rate
- Connection error rate
- Average connection duration

**Use Cases**:
- Monitor connection capacity
- Detect connection leaks
- Identify DDoS attacks
- Track user activity

### 2. IM Gateway Messages Dashboard

**File**: `deploy/docker/grafana/dashboards/im-gateway-messages.json`

**Panels**:
- Message Throughput (Graph)
- Message Delivery Latency (Graph with P50/P95/P99)
- Message Success Rate (Gauge)
- ACK Timeout Rate (Graph)
- Message Duplication Rate (Graph)
- Offline Queue Size (Graph)

**Key Metrics**:
- Messages per second
- P50/P95/P99 latency
- Success rate percentage
- ACK timeout percentage

**Use Cases**:
- Monitor message delivery performance
- Detect latency issues
- Track message loss
- Monitor offline queue backlog

### 3. IM Gateway Health Dashboard

**File**: `deploy/docker/grafana/dashboards/im-gateway-health.json`

**Panels**:
- CPU Usage (Graph)
- Memory Usage (Graph)
- Goroutines (Graph)
- Network I/O (Graph)
- Service Uptime (Stat)
- Error Rate (Graph)

**Key Metrics**:
- CPU percentage
- Memory usage
- Number of goroutines
- Network throughput

**Use Cases**:
- Monitor system resources
- Detect resource exhaustion
- Track service health
- Identify performance bottlenecks

### 4. IM Gateway SLO Dashboard

**File**: `deploy/docker/grafana/dashboards/im-gateway-slo.json`

**Panels**:
- Availability SLO (Gauge)
- Latency SLO (Gauge)
- Success Rate SLO (Gauge)
- Error Budget Consumption (Bar Gauge)
- Error Budget Remaining (Stat)
- SLO Burn Rate (Graph)

**Key Metrics**:
- 30-day availability percentage
- 30-day P99 latency
- 30-day success rate
- Error budget consumption

**Use Cases**:
- Track SLO compliance
- Monitor error budget
- Detect SLO violations
- Plan capacity and improvements

### Dashboard Access

**Development**:
```
http://localhost:3000/dashboards
```

**Production**:
```
https://grafana.example.com/dashboards
```

**Default Credentials**:
- Username: `admin`
- Password: `admin` (change on first login)

### Creating Custom Dashboards

1. Open Grafana
2. Click "+" → "Dashboard"
3. Add panels with queries
4. Save dashboard
5. Export JSON to `deploy/docker/grafana/dashboards/`

### Dashboard Best Practices

- Use consistent time ranges (Last 1 hour, Last 24 hours)
- Add annotations for deployments and incidents
- Use template variables for filtering
- Set appropriate refresh intervals (10s for real-time, 1m for historical)
- Add links to related dashboards and runbooks

---

## Troubleshooting

### Quick Diagnostic Steps

#### 1. Service Health Check

```bash
# Check if service is running
docker ps | grep im-gateway-service

# Check service logs
docker logs im-gateway-service --tail 100 --follow

# Check service health endpoint
curl http://localhost:8080/health

# Check service readiness
curl http://localhost:8080/ready

# Check service metrics
curl http://localhost:8080/metrics
```

#### 2. Infrastructure Health Check

```bash
# Check etcd cluster
curl http://localhost:2379/health

# Check MySQL
docker exec mysql mysql -u root -p -e "SHOW PROCESSLIST;"

# Check Redis
docker exec redis redis-cli ping

# Check Kafka
docker exec kafka kafka-topics.sh --list --bootstrap-server localhost:9092
```

#### 3. Monitoring Stack Health Check

```bash
# Check Prometheus
curl http://localhost:9090/-/healthy

# Check Alertmanager
curl http://localhost:9093/-/healthy

# Check Loki
curl http://localhost:3100/ready

# Check OTel Collector
curl http://localhost:13133/
```

### Common Issues

#### Issue 1: High Message Latency

**Symptoms**:
- P99 latency > 500ms
- Users reporting slow message delivery
- Alert: `HighMessageDeliveryLatency`

**Diagnosis**:
1. Check Grafana Messages dashboard
2. Review service logs for errors
3. Check CPU/memory usage
4. Check database connection pool
5. Check Redis latency

**Resolution**:
```bash
# Check CPU usage
docker stats im-gateway-service

# Check database connections
docker exec mysql mysql -u root -p -e "SHOW PROCESSLIST;"

# Check Redis latency
docker exec redis redis-cli --latency

# Scale out if needed
kubectl scale deployment im-gateway-service --replicas=5
```

**Runbook**: [High Latency](./ALERTING_GUIDE.md#1-highmessagedeliverylatency-p1)

#### Issue 2: Service Down

**Symptoms**:
- Service not responding
- All connections lost
- Alert: `IMGatewayServiceDown`

**Diagnosis**:
1. Check if container is running
2. Check service logs
3. Check resource usage
4. Check for OOM kills

**Resolution**:
```bash
# Check container status
docker ps -a | grep im-gateway-service

# Check logs for crash
docker logs im-gateway-service --tail 100

# Check for OOM
dmesg | grep -i "out of memory"

# Restart service
docker restart im-gateway-service

# If persistent, check for memory leaks
docker stats im-gateway-service
```

**Runbook**: [Service Down](./OPERATIONAL_RUNBOOKS.md#runbook-1-handle-gateway-node-failure)

#### Issue 3: High Connection Error Rate

**Symptoms**:
- Many clients unable to connect
- Connection error rate > 10%
- Alert: `HighConnectionErrorRate`

**Diagnosis**:
1. Check auth-service health
2. Review connection error logs
3. Check rate limiting
4. Verify JWT token generation

**Resolution**:
```bash
# Check auth-service
curl http://auth-service:9095/health

# Check connection errors in logs
docker logs im-gateway-service | grep "connection_error"

# Check rate limiting config
grep -r "rate_limit" deploy/docker/

# Test JWT validation
curl -H "Authorization: Bearer <token>" http://localhost:8080/ws
```

**Runbook**: [Connection Errors](./ALERTING_GUIDE.md#4-highconnectionerrorrate-warning)

#### Issue 4: Offline Queue Backlog

**Symptoms**:
- Offline queue size > 10,000
- Kafka consumer lag increasing
- Alert: `HighOfflineQueueBacklog`

**Diagnosis**:
1. Check offline worker logs
2. Check Kafka consumer lag
3. Check database write performance
4. Check memory usage

**Resolution**:
```bash
# Check offline worker logs
docker logs im-service | grep "offline_worker"

# Check Kafka consumer lag
docker exec kafka kafka-consumer-groups.sh --describe --group offline-worker --bootstrap-server localhost:9092

# Check database write performance
docker exec mysql mysql -u root -p -e "SHOW FULL PROCESSLIST;"

# Scale offline worker
kubectl scale deployment im-service --replicas=5
```

**Runbook**: [Offline Backlog](./ALERTING_GUIDE.md#6-highofflinequeuebacklog-warning)

#### Issue 5: Low Cache Hit Rate

**Symptoms**:
- Cache hit rate < 70%
- Increased load on etcd and User Service
- Alert: `LowCacheHitRate`

**Diagnosis**:
1. Check cache statistics
2. Review cache TTL configuration
3. Check cache memory limits
4. Analyze traffic patterns

**Resolution**:
```bash
# Check cache stats in Grafana
# Navigate to IM Gateway Health dashboard

# Check cache configuration
grep -r "cache_ttl" apps/im-gateway-service/

# Increase cache size or TTL
# Edit configuration and restart service

# Monitor cache hit rate improvement
```

**Runbook**: [Cache Performance](./ALERTING_GUIDE.md#7-lowcachehitrate-warning)

### Diagnostic Queries

**Find slow requests**:
```logql
{service_name="im-gateway-service"} | json | latency_ms > 1000
```

**Find errors by type**:
```logql
sum by (error) (count_over_time({service_name="im-gateway-service"} |= "ERROR" [1h]))
```

**Find top users by message count**:
```logql
topk(10, sum by (user_id) (count_over_time({service_name="im-gateway-service"} | json | event="message_delivered" [1h])))
```

**Find connection errors**:
```logql
{service_name="im-gateway-service"} | json | event="connection_error"
```

### Performance Analysis

**CPU Profiling**:
```bash
# Enable pprof endpoint
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze with go tool
go tool pprof cpu.prof
```

**Memory Profiling**:
```bash
# Get heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Analyze with go tool
go tool pprof heap.prof
```

**Goroutine Analysis**:
```bash
# Get goroutine dump
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof

# Analyze with go tool
go tool pprof goroutine.prof
```

---
## Operational Procedures

### Daily Operations

#### Morning Health Check (15 minutes)

1. **Review Overnight Alerts**:
   - Check Alertmanager for any fired alerts
   - Review resolved alerts and verify fixes
   - Check for any recurring patterns

2. **Service Health Dashboard Review**:
   - Open Grafana → IM Gateway Health dashboard
   - Verify all services are up (green status)
   - Check CPU/Memory usage is normal (< 70%)
   - Verify no resource exhaustion

3. **SLO Compliance Check**:
   - Open Grafana → IM Gateway SLO dashboard
   - Verify availability > 99.95%
   - Verify P99 latency < 200ms
   - Verify success rate > 99.99%
   - Check error budget consumption < 50%

4. **Log Review**:
   - Check Grafana Explore → Loki
   - Review ERROR logs from last 24 hours
   - Investigate any unusual patterns
   - Verify no security incidents

5. **Backup Verification**:
   - Check database backup completion
   - Verify backup size is reasonable
   - Test restore if needed (weekly)

**Checklist**:
```markdown
- [ ] Reviewed overnight alerts
- [ ] All services healthy
- [ ] SLOs within target
- [ ] No critical errors in logs
- [ ] Backups completed successfully
```

#### Continuous Monitoring (Throughout Day)

1. **Active Alert Monitoring**:
   - Keep Alertmanager open in browser tab
   - Respond to alerts within SLA
   - Escalate if needed

2. **Dashboard Monitoring**:
   - Check dashboards every 2 hours
   - Look for anomalies or trends
   - Proactively address issues

3. **Capacity Monitoring**:
   - Monitor connection growth
   - Check resource utilization trends
   - Plan scaling if needed

4. **User Feedback**:
   - Monitor support channels
   - Check for user-reported issues
   - Correlate with metrics

#### End of Day Review (15 minutes)

1. **Incident Summary**:
   - Document any incidents
   - Update runbooks if needed
   - Share learnings with team

2. **Metrics Review**:
   - Check daily metrics summary
   - Compare with previous days
   - Identify trends

3. **Handoff Preparation**:
   - Document any ongoing issues
   - Update on-call notes
   - Brief next shift

**Checklist**:
```markdown
- [ ] Documented incidents
- [ ] Updated runbooks
- [ ] Reviewed daily metrics
- [ ] Prepared handoff notes
```

### Weekly Operations

#### Monday: Week Planning (30 minutes)

1. **Weekend Review**:
   - Review weekend incidents
   - Check SLO compliance
   - Verify no degradation

2. **Capacity Planning**:
   - Review weekly capacity trends
   - Plan scaling if needed
   - Update capacity forecasts

3. **Maintenance Planning**:
   - Schedule maintenance windows
   - Plan upgrades or patches
   - Coordinate with teams

#### Wednesday: Mid-Week Check (30 minutes)

1. **Health Check**:
   - Review week-to-date metrics
   - Check SLO trajectory
   - Identify any concerns

2. **Alert Quality Review**:
   - Review alert frequency
   - Check false positive rate
   - Tune alerts if needed

3. **Documentation Update**:
   - Update runbooks
   - Document new procedures
   - Share knowledge

#### Friday: Week Wrap-Up (30 minutes)

1. **Weekly Summary**:
   - Summarize week's incidents
   - Calculate weekly SLO
   - Document lessons learned

2. **Weekend Preparation**:
   - Verify on-call schedule
   - Check for planned events
   - Prepare for traffic patterns

3. **Team Sync**:
   - Share weekly highlights
   - Discuss improvements
   - Plan next week

### Monthly Operations

#### First Week: Monthly Review (2 hours)

1. **SLO Review**:
   - Calculate monthly SLO achievement
   - Review error budget consumption
   - Document SLO violations
   - Generate monthly SLO report

2. **Capacity Planning**:
   - Review monthly capacity trends
   - Update capacity forecasts
   - Plan infrastructure changes
   - Update cost projections

3. **Security Review**:
   - Review security patches
   - Check vulnerability scans
   - Update security policies
   - Plan security improvements

4. **Incident Analysis**:
   - Review all incidents
   - Identify trends and patterns
   - Calculate MTTD and MTTR
   - Plan preventive measures

#### Second Week: Testing and Verification (2 hours)

1. **Disaster Recovery Drill**:
   - Test backup restoration
   - Verify DR procedures
   - Update DR runbooks
   - Document findings

2. **Performance Testing**:
   - Run load tests
   - Verify performance targets
   - Identify bottlenecks
   - Plan optimizations

3. **Alert Testing**:
   - Test critical alerts
   - Verify notification channels
   - Check escalation policies
   - Update alert configurations

#### Third Week: Improvement and Optimization (2 hours)

1. **Runbook Review**:
   - Review all runbooks
   - Update procedures
   - Add new runbooks
   - Remove obsolete content

2. **Dashboard Improvements**:
   - Review dashboard usage
   - Add new panels
   - Remove unused panels
   - Improve visualizations

3. **Alert Tuning**:
   - Review alert quality
   - Adjust thresholds
   - Reduce false positives
   - Add missing alerts

4. **Training**:
   - Conduct training sessions
   - Share knowledge
   - Practice runbooks
   - Improve skills

#### Fourth Week: Reporting and Planning (2 hours)

1. **Monthly Report**:
   - Prepare monthly operations report
   - Include SLO achievement
   - Document incidents
   - Highlight improvements

2. **Quarterly Planning** (if applicable):
   - Review quarterly goals
   - Plan major initiatives
   - Update roadmap
   - Allocate resources

3. **Process Improvements**:
   - Review operational processes
   - Identify inefficiencies
   - Implement improvements
   - Measure impact

4. **Team Retrospective**:
   - Conduct team retrospective
   - Discuss what went well
   - Identify areas for improvement
   - Plan action items

---

## Best Practices

### Monitoring Best Practices

#### 1. Use the Four Golden Signals

- **Latency**: How long it takes to service a request
- **Traffic**: How much demand is being placed on your system
- **Errors**: The rate of requests that fail
- **Saturation**: How "full" your service is

#### 2. Monitor Symptoms, Not Causes

- Alert on user-visible symptoms (high latency, errors)
- Use metrics to diagnose causes (CPU, memory, disk)
- Don't alert on every metric

#### 3. Use SLOs to Drive Alerting

- Alert when SLO is at risk (burn rate)
- Use error budget to prioritize work
- Don't alert on every small deviation

#### 4. Keep Dashboards Simple

- One dashboard per service
- Focus on key metrics
- Use consistent layouts
- Add links to runbooks

#### 5. Use Consistent Naming

- Use standard metric names
- Use consistent labels
- Follow naming conventions
- Document metric meanings

#### 6. Monitor the Monitoring System

- Alert if metrics stop flowing
- Monitor collector health
- Check storage capacity
- Verify dashboard access

### Alerting Best Practices

#### 1. Every Alert Should Be Actionable

- If you can't take action, don't alert
- Include runbook link in alert
- Provide context in alert message
- Make alerts specific

#### 2. Use Appropriate Severity Levels

- Critical: Immediate action required
- P1: Action required within 15 minutes
- P2: Action required within 1 hour
- Warning: Review during business hours

#### 3. Avoid Alert Fatigue

- Tune alert thresholds
- Reduce false positives
- Group related alerts
- Use alert inhibition

#### 4. Use Multi-Window Alerting

- Short window for fast burn (1 hour)
- Long window for slow burn (6 hours)
- Combine with error budget

#### 5. Test Your Alerts

- Regularly test alert firing
- Verify notification delivery
- Practice runbook procedures
- Update based on learnings

#### 6. Document Alert Response

- Create runbooks for each alert
- Include diagnosis steps
- Provide resolution steps
- Add verification steps

### Logging Best Practices

#### 1. Use Structured Logging

- Use JSON format
- Include standard fields (timestamp, level, service)
- Add context (trace_id, user_id, msg_id)
- Make logs machine-readable

#### 2. Log at Appropriate Levels

- DEBUG: Detailed diagnostic (disabled in prod)
- INFO: General informational
- WARN: Potentially harmful situations
- ERROR: Error events
- FATAL: Critical errors

#### 3. Don't Log Sensitive Data

- Never log passwords or tokens
- Mask PII (email, phone, address)
- Redact sensitive fields
- Follow privacy regulations

#### 4. Include Context

- Add trace_id for distributed tracing
- Include user_id for user tracking
- Add msg_id for message tracking
- Include operation context

#### 5. Use Log Sampling

- Sample high-volume logs
- Keep all ERROR logs
- Reduce DEBUG logs in production
- Balance volume and detail

#### 6. Correlate Logs with Traces

- Include trace_id in logs
- Use consistent IDs
- Link logs to traces
- Enable end-to-end visibility

### SLO Best Practices

#### 1. Start with Realistic SLOs

- Don't aim for 100% (unrealistic)
- 99.9% is often sufficient
- Consider user expectations
- Balance reliability and velocity

#### 2. Use Error Budgets

- Track error budget consumption
- Use budget to prioritize work
- Alert when budget at risk
- Reset budget monthly

#### 3. Measure What Users Experience

- Focus on user-facing metrics
- Measure end-to-end latency
- Track success rate
- Monitor availability

#### 4. Review SLOs Regularly

- Weekly SLO check
- Monthly SLO review
- Quarterly SLO adjustment
- Document violations

#### 5. Use SLO-Based Alerting

- Alert on burn rate
- Use multi-window alerting
- Combine with error budget
- Reduce alert fatigue

#### 6. Communicate SLOs

- Share SLOs with stakeholders
- Explain error budgets
- Report SLO achievement
- Discuss trade-offs

### Dashboard Best Practices

#### 1. Design for Your Audience

- Operations: Real-time health
- Engineers: Detailed metrics
- Management: High-level KPIs
- Users: Service status

#### 2. Use Consistent Layouts

- Key metrics at top
- Detailed metrics below
- Related metrics grouped
- Consistent time ranges

#### 3. Add Context

- Include SLO targets
- Show historical trends
- Add annotations for events
- Link to related dashboards

#### 4. Use Appropriate Visualizations

- Gauges for current values
- Graphs for trends
- Tables for lists
- Heatmaps for distributions

#### 5. Keep Dashboards Focused

- One purpose per dashboard
- Limit panels (< 20)
- Remove unused panels
- Regular cleanup

#### 6. Make Dashboards Discoverable

- Use clear names
- Add descriptions
- Organize in folders
- Create dashboard index

---

## Support

### Emergency Contacts

| Role | Contact | Response Time | Availability |
|------|---------|---------------|--------------|
| Primary On-Call | PagerDuty | < 5 minutes | 24/7 |
| Secondary On-Call | PagerDuty | < 15 minutes | 24/7 |
| Operations Team Lead | ops-lead@example.com | < 1 hour | Business hours |
| SRE Team Lead | sre-lead@example.com | < 2 hours | Business hours |
| Engineering Manager | eng-manager@example.com | < 4 hours | Business hours |
| VP Engineering | vp-eng@example.com | < 8 hours | Business hours |

### Communication Channels

#### Slack Channels

- **#alerts-critical**: Critical and P1 alerts (monitored 24/7)
- **#alerts-p2**: P2 alerts (monitored during business hours)
- **#alerts-warnings**: Warning alerts (reviewed daily)
- **#ops-team**: Operations team discussions
- **#sre-team**: SRE team discussions
- **#incidents**: Active incident coordination
- **#on-call**: On-call team coordination
- **#observability-support**: Monitoring and alerting support

#### Email Lists

- **ops@example.com**: Operations team
- **sre@example.com**: SRE team
- **incidents@example.com**: Incident notifications
- **observability-team@example.com**: Monitoring support

#### PagerDuty

- **Service**: IM Chat System
- **Escalation Policy**: 
  - Primary on-call → Secondary on-call (15 min) → Team Lead (30 min) → Manager (1 hour)
- **Integration**: Alertmanager, Grafana, Datadog

### Documentation Links

#### Internal Documentation

- **Wiki**: https://wiki.example.com/im-chat-system
- **Runbooks**: https://wiki.example.com/runbooks/im-gateway
- **Architecture**: https://wiki.example.com/architecture/im-chat
- **API Documentation**: https://api-docs.example.com/im-chat
- **Deployment Guide**: https://wiki.example.com/deployment/im-chat

#### External Documentation

- **Prometheus**: https://prometheus.io/docs/
- **Grafana**: https://grafana.com/docs/
- **Loki**: https://grafana.com/docs/loki/
- **OpenTelemetry**: https://opentelemetry.io/docs/
- **Alertmanager**: https://prometheus.io/docs/alerting/latest/alertmanager/

#### Service Documentation

- **IM Gateway Service**: [../../apps/im-gateway-service/README.md](../../apps/im-gateway-service/README.md)
- **IM Service**: [../../apps/im-service/README.md](../../apps/im-service/README.md)
- **Auth Service**: [../../apps/auth-service/README.md](../../apps/auth-service/README.md)
- **User Service**: [../../apps/user-service/README.md](../../apps/user-service/README.md)

### Training Resources

#### Onboarding

- **New Hire Onboarding**: 2-week program covering system architecture, monitoring, and operations
- **On-Call Training**: 1-week program covering incident response and runbooks
- **Tool Training**: Hands-on training for Grafana, Prometheus, Kubernetes

#### Ongoing Training

- **Monthly Tech Talks**: Deep dives into system components
- **Quarterly Runbook Walkthroughs**: Practice incident response
- **Annual Disaster Recovery Drills**: Test DR procedures
- **Conference Attendance**: SREcon, KubeCon, Observability conferences

#### Self-Service Resources

- **Video Tutorials**: https://training.example.com/im-chat
- **Interactive Labs**: https://labs.example.com/im-chat
- **Documentation**: This guide and related documents
- **Office Hours**: Weekly Q&A sessions with SRE team

### Monitoring Tools Access

#### Grafana

- **Development**: http://localhost:3000
- **Staging**: https://grafana-staging.example.com
- **Production**: https://grafana.example.com
- **Credentials**: SSO (Okta) or admin/admin (dev only)

#### Prometheus

- **Development**: http://localhost:9090
- **Staging**: https://prometheus-staging.example.com
- **Production**: https://prometheus.example.com
- **Access**: VPN required for production

#### Alertmanager

- **Development**: http://localhost:9093
- **Staging**: https://alertmanager-staging.example.com
- **Production**: https://alertmanager.example.com
- **Access**: VPN required for production

#### Jaeger

- **Development**: http://localhost:16686
- **Staging**: https://jaeger-staging.example.com
- **Production**: https://jaeger.example.com
- **Access**: VPN required for production

### Getting Help

#### For Monitoring Issues

1. **Check this guide first**: Most common issues are documented here
2. **Search Slack**: Check #observability-support for similar issues
3. **Ask in Slack**: Post in #observability-support with details
4. **Create ticket**: For non-urgent issues, create Jira ticket
5. **Escalate**: For urgent issues, page on-call via PagerDuty

#### For Incident Response

1. **Follow runbooks**: Use documented procedures
2. **Ask for help**: Don't hesitate to escalate
3. **Communicate**: Keep stakeholders informed
4. **Document**: Record actions taken
5. **Learn**: Conduct post-mortem for P0/P1

#### For Tool Access

1. **Request access**: Submit access request via IT portal
2. **VPN setup**: Follow VPN setup guide
3. **SSO issues**: Contact IT support
4. **Training**: Attend tool training sessions

### Feedback and Improvements

We continuously improve this guide based on feedback. Please contribute:

- **Slack**: Share feedback in #observability-support
- **Pull Requests**: Submit PRs to update documentation
- **Retrospectives**: Share learnings in team retrospectives
- **Surveys**: Participate in quarterly operations surveys

### Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2026-01-25 | Initial comprehensive guide | Operations Team |

---

## Appendix

### Glossary

- **SLO**: Service Level Objective - Target level of service reliability
- **SLI**: Service Level Indicator - Metric used to measure SLO
- **Error Budget**: Allowed amount of unreliability (1 - SLO)
- **Burn Rate**: Rate at which error budget is consumed
- **MTTD**: Mean Time To Detect - Average time to detect an incident
- **MTTR**: Mean Time To Resolve - Average time to resolve an incident
- **P99**: 99th percentile - 99% of requests are faster than this value
- **Cardinality**: Number of unique time series for a metric
- **Scrape Interval**: How often Prometheus collects metrics
- **Retention**: How long data is stored

### Metric Reference

#### Connection Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `im_gateway_active_connections` | Gauge | `node` | Current active WebSocket connections |
| `im_gateway_connections_established_total` | Counter | `node` | Total connections established |
| `im_gateway_connections_closed_total` | Counter | `node`, `reason` | Total connections closed |
| `im_gateway_connection_errors_total` | Counter | `node`, `error_type` | Total connection errors |

#### Message Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `im_gateway_messages_delivered_total` | Counter | `node`, `type` | Total messages delivered |
| `im_gateway_messages_failed_total` | Counter | `node`, `type`, `reason` | Total messages failed |
| `im_gateway_message_delivery_latency_seconds` | Histogram | `node`, `type` | Message delivery latency |
| `im_gateway_ack_timeout_total` | Counter | `node` | Total ACK timeouts |
| `im_gateway_message_duplication_total` | Counter | `node` | Total duplicate messages |

#### System Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `up` | Gauge | `job`, `instance` | Service availability (1=up, 0=down) |
| `process_cpu_seconds_total` | Counter | `job`, `instance` | CPU time used |
| `process_resident_memory_bytes` | Gauge | `job`, `instance` | Memory usage |
| `go_goroutines` | Gauge | `job`, `instance` | Number of goroutines |

### Alert Reference

See [ALERTING_GUIDE.md](./ALERTING_GUIDE.md) for complete alert documentation.

### Runbook Index

See [OPERATIONAL_RUNBOOKS.md](./OPERATIONAL_RUNBOOKS.md) for complete runbook documentation.

---

**Document Maintenance**:
- **Review Frequency**: Monthly
- **Owner**: Operations Team
- **Last Review**: 2026-01-25
- **Next Review**: 2026-02-25

**Feedback**: Please submit feedback or corrections via Slack (#observability-support) or pull request.

---

*This guide is a living document. Please keep it updated as the system evolves.*
