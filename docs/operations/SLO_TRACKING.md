# SLO Tracking Guide

## Overview

This guide explains how to track Service Level Objectives (SLOs) for the IM Gateway Service, including SLI metrics, error budgets, and alerting.

## Service Level Objectives (SLOs)

### 1. Availability SLO

**Target**: 99.95% availability (21.6 minutes downtime per month)

**Definition**: Percentage of time the service is available and responding to health checks

**Measurement**:
```promql
# Availability over 30 days
avg_over_time(up{job="im-gateway-service"}[30d]) * 100
```

**Error Budget**: 0.05% (21.6 minutes per month)

### 2. Latency SLO

**Target**: 99% of message deliveries complete within 200ms

**Definition**: P99 message delivery latency

**Measurement**:
```promql
# P99 latency over 30 days
histogram_quantile(0.99, 
  rate(im_gateway_message_delivery_latency_seconds_bucket[30d])
) * 1000
```

**Error Budget**: 1% of requests can exceed 200ms

### 3. Success Rate SLO

**Target**: 99.99% of messages delivered successfully (0.01% loss rate)

**Definition**: Percentage of messages delivered without loss

**Measurement**:
```promql
# Success rate over 30 days
(
  sum(rate(im_gateway_messages_delivered_total[30d]))
  /
  (sum(rate(im_gateway_messages_delivered_total[30d])) + 
   sum(rate(im_gateway_messages_failed_total[30d])))
) * 100
```

**Error Budget**: 0.01% message loss allowed

## Service Level Indicators (SLIs)

### 1. Availability SLI

**Metric**: `up{job="im-gateway-service"}`

**Good Events**: Service responding to health checks (up=1)

**Total Events**: All health check attempts

**Formula**:
```promql
sum(up{job="im-gateway-service"}) / count(up{job="im-gateway-service"})
```

### 2. Latency SLI

**Metric**: `im_gateway_message_delivery_latency_seconds`

**Good Events**: Requests completed within 200ms

**Total Events**: All message delivery requests

**Formula**:
```promql
sum(rate(im_gateway_message_delivery_latency_seconds_bucket{le="0.2"}[5m]))
/
sum(rate(im_gateway_message_delivery_latency_seconds_count[5m]))
```

### 3. Success Rate SLI

**Metric**: `im_gateway_messages_delivered_total`, `im_gateway_messages_failed_total`

**Good Events**: Messages delivered successfully

**Total Events**: All message delivery attempts

**Formula**:
```promql
sum(rate(im_gateway_messages_delivered_total[5m]))
/
(sum(rate(im_gateway_messages_delivered_total[5m])) + 
 sum(rate(im_gateway_messages_failed_total[5m])))
```

## Error Budget

### Calculation

**Monthly Error Budget** = (1 - SLO) × Total Time

For 99.95% availability SLO:
- Error Budget = 0.05% × 30 days = 21.6 minutes/month
- Error Budget = 0.05% × 43,200 minutes = 21.6 minutes

### Error Budget Consumption

**Current Consumption**:
```promql
# Percentage of error budget consumed this month
(1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 * 100
```

**Remaining Budget**:
```promql
# Minutes remaining in error budget
(0.0005 - (1 - avg_over_time(up{job="im-gateway-service"}[30d]))) * 43200
```

### Error Budget Alerts

**50% Budget Consumed** (Warning):
```yaml
- alert: ErrorBudget50PercentConsumed
  expr: |
    (1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 > 0.5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "50% of monthly error budget consumed"
    description: "Error budget consumption: {{ $value | humanizePercentage }}"
```

**80% Budget Consumed** (Critical):
```yaml
- alert: ErrorBudget80PercentConsumed
  expr: |
    (1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 > 0.8
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "80% of monthly error budget consumed - Circuit breaker recommended"
    description: "Error budget consumption: {{ $value | humanizePercentage }}"
```

## SLO Dashboard

### Grafana Dashboard Panels

#### 1. Availability SLO Panel

**Query**:
```promql
avg_over_time(up{job="im-gateway-service"}[30d]) * 100
```

**Visualization**: Gauge
- Green: > 99.95%
- Yellow: 99.90% - 99.95%
- Red: < 99.90%

#### 2. Latency SLO Panel

**Query**:
```promql
histogram_quantile(0.99, 
  rate(im_gateway_message_delivery_latency_seconds_bucket[30d])
) * 1000
```

**Visualization**: Gauge
- Green: < 200ms
- Yellow: 200ms - 300ms
- Red: > 300ms

#### 3. Success Rate SLO Panel

**Query**:
```promql
(
  sum(rate(im_gateway_messages_delivered_total[30d]))
  /
  (sum(rate(im_gateway_messages_delivered_total[30d])) + 
   sum(rate(im_gateway_messages_failed_total[30d])))
) * 100
```

**Visualization**: Gauge
- Green: > 99.99%
- Yellow: 99.95% - 99.99%
- Red: < 99.95%

#### 4. Error Budget Consumption Panel

**Query**:
```promql
(1 - avg_over_time(up{job="im-gateway-service"}[30d])) / 0.0005 * 100
```

**Visualization**: Bar gauge
- Green: < 50%
- Yellow: 50% - 80%
- Red: > 80%

#### 5. Error Budget Remaining Panel

**Query**:
```promql
(0.0005 - (1 - avg_over_time(up{job="im-gateway-service"}[30d]))) * 43200
```

**Visualization**: Stat
- Display: Minutes remaining
- Threshold: < 5 minutes = Red

## SLO Burn Rate

### Fast Burn (1 hour window)

**Alert when burning budget 14.4x faster than allowed**:
```yaml
- alert: SLOFastBurn
  expr: |
    (
      1 - (
        sum(rate(im_gateway_messages_delivered_total[1h]))
        /
        (sum(rate(im_gateway_messages_delivered_total[1h])) + 
         sum(rate(im_gateway_messages_failed_total[1h])))
      )
    ) > (0.0001 * 14.4)
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Fast SLO burn detected - 5% of monthly budget in 1 hour"
```

### Slow Burn (6 hour window)

**Alert when burning budget 6x faster than allowed**:
```yaml
- alert: SLOSlowBurn
  expr: |
    (
      1 - (
        sum(rate(im_gateway_messages_delivered_total[6h]))
        /
        (sum(rate(im_gateway_messages_delivered_total[6h])) + 
         sum(rate(im_gateway_messages_failed_total[6h])))
      )
    ) > (0.0001 * 6)
  for: 15m
  labels:
    severity: warning
  annotations:
    summary: "Slow SLO burn detected - 2.5% of monthly budget in 6 hours"
```

## SLO Review Process

### Weekly Review

1. Check current SLO compliance
2. Review error budget consumption
3. Identify incidents that consumed budget
4. Plan improvements if needed

### Monthly Review

1. Calculate final SLO achievement
2. Document any SLO violations
3. Analyze root causes
4. Update SLO targets if needed
5. Reset error budget for new month

## SLO-Based Alerting Strategy

### Tier 1: Fast Burn (Critical)
- **Window**: 1 hour
- **Threshold**: 14.4x burn rate
- **Action**: Immediate investigation
- **Impact**: 5% of monthly budget consumed in 1 hour

### Tier 2: Slow Burn (Warning)
- **Window**: 6 hours
- **Threshold**: 6x burn rate
- **Action**: Investigation within 1 hour
- **Impact**: 2.5% of monthly budget consumed in 6 hours

### Tier 3: Budget Threshold (Warning)
- **Threshold**: 50% budget consumed
- **Action**: Review and plan improvements
- **Impact**: Half of monthly budget consumed

### Tier 4: Budget Critical (Critical)
- **Threshold**: 80% budget consumed
- **Action**: Consider circuit breaker
- **Impact**: Most of monthly budget consumed

## Best Practices

### DO:
- Track SLOs over 30-day rolling windows
- Use error budgets to prioritize work
- Alert on burn rate, not just SLO violations
- Review SLOs regularly with stakeholders
- Document SLO violations and learnings

### DON'T:
- Set SLOs too high (99.999% is often unrealistic)
- Ignore error budget consumption
- Alert on every small SLO deviation
- Change SLOs frequently without reason
- Measure SLOs without taking action

## Reporting

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

## Tools and Resources

- **Prometheus**: Metrics collection and querying
- **Grafana**: SLO dashboard and visualization
- **Alertmanager**: SLO-based alerting
- **SLO Calculator**: https://sre.google/workbook/slo-document/

## Support

For questions about SLOs:
- Slack: #sre-team
- Documentation: https://sre.google/sre-book/service-level-objectives/
- SLO Workshop: Monthly on first Tuesday
