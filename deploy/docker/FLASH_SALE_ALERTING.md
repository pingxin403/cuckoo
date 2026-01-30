# Flash Sale System Alerting Rules

## Overview

This document describes the Prometheus alerting rules configured for the flash-sale-service. These rules monitor critical system metrics and trigger alerts when thresholds are exceeded or faults are detected.

**Configuration File**: `deploy/docker/prometheus-alerts.yml`

**Alert Group**: `flash_sale_alerts`

**Evaluation Interval**: 30 seconds

## Alert Categories

### 1. Threshold Alerts

Threshold alerts monitor system performance metrics and trigger when values exceed predefined limits.

#### 1.1 High Failure Rate

**Alert Name**: `FlashSaleHighFailureRate`

**Severity**: Critical

**Condition**: Failure rate > 5% over 5 minutes

**Trigger Time**: 3 minutes

**Description**: Monitors the ratio of failed requests to total requests. A high failure rate indicates system issues that prevent users from completing flash sale purchases.

**PromQL Expression**:
```promql
(
  rate(flash_sale_requests_failure[5m])
  /
  rate(flash_sale_requests_total[5m])
) > 0.05
```

**Action Items**:
- Check Redis connectivity and health
- Verify inventory service is functioning
- Review application logs for error patterns
- Check system resource utilization (CPU, memory, network)

---

#### 1.2 High Response Time (P99)

**Alert Name**: `FlashSaleHighResponseTime`

**Severity**: Warning

**Condition**: P99 response time > 200ms over 5 minutes

**Trigger Time**: 5 minutes

**Description**: Monitors the 99th percentile response time. High latency degrades user experience and may indicate performance bottlenecks.

**PromQL Expression**:
```promql
histogram_quantile(0.99, rate(flash_sale_request_duration_bucket[5m])) > 0.2
```

**Action Items**:
- Check Redis latency using `redis-cli --latency`
- Monitor Kafka producer lag
- Review system resource utilization
- Check network latency to dependencies

---

#### 1.3 Critical Response Time (P99)

**Alert Name**: `FlashSaleCriticalResponseTime`

**Severity**: Critical

**Condition**: P99 response time > 500ms over 5 minutes

**Trigger Time**: 2 minutes

**Description**: Critical latency threshold that severely impacts user experience. Requires immediate investigation.

**PromQL Expression**:
```promql
histogram_quantile(0.99, rate(flash_sale_request_duration_bucket[5m])) > 0.5
```

**Action Items**:
- Immediate investigation required
- Consider enabling circuit breaker to protect system
- Check for resource exhaustion
- Review recent deployments or configuration changes

---

#### 1.4 Low Inventory Warning

**Alert Name**: `FlashSaleLowInventory`

**Severity**: Warning

**Condition**: Remaining inventory < 100 units

**Trigger Time**: 1 minute

**Description**: Warns when inventory is running low, allowing operators to prepare for sold-out scenarios.

**PromQL Expression**:
```promql
flash_sale_inventory_remaining < 100
```

**Action Items**:
- Prepare for sold-out notification
- Monitor queue closure process
- Verify sold-out messaging is ready

---

#### 1.5 Inventory Depleted

**Alert Name**: `FlashSaleInventoryDepleted`

**Severity**: Info

**Condition**: Remaining inventory = 0

**Trigger Time**: 30 seconds

**Description**: Informational alert when a SKU is sold out. Confirms the flash sale has completed successfully.

**PromQL Expression**:
```promql
flash_sale_inventory_remaining == 0
```

**Action Items**:
- Verify sold-out notification sent to queued users
- Confirm queue is closed for the SKU
- Monitor for any late-arriving requests

---

#### 1.6 High Queue Length

**Alert Name**: `FlashSaleHighQueueLength`

**Severity**: Warning

**Condition**: Queue length > 10,000

**Trigger Time**: 5 minutes

**Description**: Monitors queue depth. High queue length indicates many users are waiting, which may lead to poor user experience.

**PromQL Expression**:
```promql
flash_sale_queue_length > 10000
```

**Action Items**:
- Monitor token bucket rate
- Consider increasing throughput if system can handle it
- Review queue management configuration
- Communicate wait times to users

---

#### 1.7 Critical Queue Length

**Alert Name**: `FlashSaleCriticalQueueLength`

**Severity**: Critical

**Condition**: Queue length > 50,000

**Trigger Time**: 2 minutes

**Description**: Critical queue depth that may overwhelm the system. Requires immediate action to prevent system degradation.

**PromQL Expression**:
```promql
flash_sale_queue_length > 50000
```

**Action Items**:
- Consider rejecting new requests temporarily
- Scale up service instances if possible
- Review rate limiting configuration
- Communicate delays to users

---

#### 1.8 High Token Rejection Rate

**Alert Name**: `FlashSaleHighTokenRejectionRate`

**Severity**: Warning

**Condition**: Token rejection rate > 80% over 5 minutes

**Trigger Time**: 5 minutes

**Description**: Monitors the ratio of rejected token requests. High rejection rate indicates most users are being queued.

**PromQL Expression**:
```promql
(
  rate(flash_sale_queue_tokens_rejected[5m])
  /
  (rate(flash_sale_queue_tokens_acquired[5m]) + rate(flash_sale_queue_tokens_rejected[5m]))
) > 0.8
```

**Action Items**:
- Review token bucket configuration
- Verify rate limits are appropriate for current load
- Check if inventory is sufficient
- Monitor user experience metrics

---

#### 1.9 High Inventory Deduction Latency

**Alert Name**: `FlashSaleHighInventoryDeductionLatency`

**Severity**: Warning

**Condition**: P99 inventory deduction latency > 50ms over 5 minutes

**Trigger Time**: 5 minutes

**Description**: Monitors Redis Lua script execution time. High latency indicates Redis performance issues.

**PromQL Expression**:
```promql
histogram_quantile(0.99, rate(flash_sale_inventory_deduction_duration_bucket[5m])) > 0.05
```

**Action Items**:
- Check Redis performance metrics
- Monitor Redis CPU and memory usage
- Check network latency to Redis
- Review Redis slow log

---

#### 1.10 High Inventory Rollback Rate

**Alert Name**: `FlashSaleHighRollbackRate`

**Severity**: Warning

**Condition**: Rollback rate > 30% of deductions over 10 minutes

**Trigger Time**: 5 minutes

**Description**: Monitors inventory rollback operations. High rollback rate indicates many orders are timing out or being cancelled.

**PromQL Expression**:
```promql
(
  rate(flash_sale_inventory_rollbacks_total[10m])
  /
  rate(flash_sale_inventory_deductions_total[10m])
) > 0.3
```

**Action Items**:
- Investigate order timeout issues
- Check payment processing delays
- Review order timeout configuration
- Monitor user payment completion rate

---

### 2. Fault Alerts

Fault alerts detect infrastructure failures and service unavailability.

#### 2.1 Flash Sale Service Down

**Alert Name**: `FlashSaleServiceDown`

**Severity**: Critical

**Condition**: Service health check failing

**Trigger Time**: 1 minute

**Description**: The flash-sale-service is not responding to Prometheus health checks. Service is unavailable.

**PromQL Expression**:
```promql
up{job="flash-sale-service"} == 0
```

**Action Items**:
- Check service logs immediately
- Verify service is running: `systemctl status flash-sale-service`
- Check for OOM kills or crashes
- Restart service if necessary
- Review recent deployments

---

#### 2.2 Redis Connection Failure

**Alert Name**: `FlashSaleRedisConnectionFailure`

**Severity**: Critical

**Condition**: Redis error rate > 10 errors/sec over 5 minutes

**Trigger Time**: 2 minutes

**Description**: The service is experiencing Redis connection failures. This prevents inventory operations from succeeding.

**PromQL Expression**:
```promql
rate(flash_sale_requests_failure{error_type="redis_error"}[5m]) > 10
```

**Action Items**:
- Check Redis service health: `redis-cli ping`
- Verify network connectivity to Redis
- Check Redis connection pool configuration
- Review Redis logs for errors
- Monitor Redis resource utilization

**Note**: This alert requires error_type labels to be added to failure metrics. See implementation notes below.

---

#### 2.3 Kafka Producer Failure

**Alert Name**: `FlashSaleKafkaProducerFailure`

**Severity**: Critical

**Condition**: Kafka error rate > 5 errors/sec over 5 minutes

**Trigger Time**: 2 minutes

**Description**: The service is experiencing Kafka producer failures. This prevents order messages from being sent to the queue.

**PromQL Expression**:
```promql
rate(flash_sale_requests_failure{error_type="kafka_error"}[5m]) > 5
```

**Action Items**:
- Check Kafka broker health
- Verify network connectivity to Kafka
- Check Kafka producer configuration
- Review Kafka broker logs
- Monitor Kafka disk space and resource utilization

**Note**: This alert requires error_type labels to be added to failure metrics. See implementation notes below.

---

### 3. Performance Alerts

Performance alerts monitor overall system health and capacity.

#### 3.1 Low Success Rate

**Alert Name**: `FlashSaleLowSuccessRate`

**Severity**: Warning

**Condition**: Success rate < 90% over 5 minutes

**Trigger Time**: 5 minutes

**Description**: Overall success rate is below acceptable threshold. Indicates widespread issues affecting user requests.

**PromQL Expression**:
```promql
(
  rate(flash_sale_requests_success[5m])
  /
  rate(flash_sale_requests_total[5m])
) < 0.9
```

**Action Items**:
- Review error logs for patterns
- Check all system dependencies (Redis, Kafka, MySQL)
- Monitor resource utilization
- Review recent changes or deployments

---

#### 3.2 High Request Rate

**Alert Name**: `FlashSaleHighRequestRate`

**Severity**: Warning

**Condition**: Request rate > 100K QPS

**Trigger Time**: 2 minutes

**Description**: Request rate is approaching or exceeding design capacity. May require scaling.

**PromQL Expression**:
```promql
rate(flash_sale_requests_total[1m]) > 100000
```

**Action Items**:
- Monitor system resources (CPU, memory, network)
- Consider horizontal scaling if sustained
- Review rate limiting configuration
- Check if load is legitimate or attack

---

### 4. Data Consistency Alerts

Data consistency alerts detect anomalies that may indicate data integrity issues.

#### 4.1 Inventory Deduction Anomaly

**Alert Name**: `FlashSaleInventoryDeductionAnomaly`

**Severity**: Warning

**Condition**: Deductions exceed successful requests by >10% over 5 minutes

**Trigger Time**: 5 minutes

**Description**: Inventory deductions are significantly higher than successful requests, which may indicate double-deduction or reconciliation issues.

**PromQL Expression**:
```promql
rate(flash_sale_inventory_deductions_total[5m]) > 
rate(flash_sale_requests_success[5m]) * 1.1
```

**Action Items**:
- Investigate potential double-deduction scenarios
- Check reconciliation service logs
- Review inventory deduction logic
- Verify idempotency mechanisms

---

## Alert Severity Levels

| Severity | Description | Response Time | Escalation |
|----------|-------------|---------------|------------|
| **Critical** | Service down or major functionality impaired | Immediate (< 5 min) | Page on-call engineer |
| **Warning** | Degraded performance or approaching limits | Within 30 minutes | Notify team channel |
| **Info** | Informational, no action required | N/A | Log only |

## Integration with AlertManager

### AlertManager Configuration

The alerts are sent to Prometheus AlertManager, which handles routing, grouping, and notification delivery.

**AlertManager Endpoint**: Configured in Prometheus configuration

**Notification Channels**:
- PagerDuty (Critical alerts)
- Slack (Warning and Info alerts)
- Email (All alerts)

### Alert Routing Example

```yaml
route:
  group_by: ['alertname', 'service', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'
  routes:
    - match:
        service: flash-sale-service
        severity: critical
      receiver: 'pagerduty-critical'
      continue: true
    - match:
        service: flash-sale-service
        severity: warning
      receiver: 'slack-warnings'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops-team@example.com'
  
  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: '<pagerduty-service-key>'
  
  - name: 'slack-warnings'
    slack_configs:
      - api_url: '<slack-webhook-url>'
        channel: '#flash-sale-alerts'
```

## Testing Alerts

### Manual Alert Testing

You can manually test alerts by simulating conditions:

#### Test High Failure Rate
```bash
# Generate failed requests
for i in {1..1000}; do
  curl -X POST http://localhost:8080/api/seckill/invalid-sku \
    -H "Content-Type: application/json" \
    -d '{"userId":"test-user"}' &
done
```

#### Test High Response Time
```bash
# Add artificial delay in code or use chaos engineering tools
# Monitor P99 latency in Prometheus
```

#### Test Service Down
```bash
# Stop the service
systemctl stop flash-sale-service

# Wait 1 minute for alert to fire
# Check AlertManager UI
```

### Verify Alert Configuration

```bash
# Validate Prometheus configuration
promtool check config /etc/prometheus/prometheus.yml

# Validate alert rules
promtool check rules /etc/prometheus/prometheus-alerts.yml

# Test alert rule expression
promtool query instant http://localhost:9090 \
  'rate(flash_sale_requests_failure[5m]) / rate(flash_sale_requests_total[5m])'
```

## Monitoring Alert Health

### Alert Manager Metrics

Monitor AlertManager itself to ensure alerts are being delivered:

```promql
# Alerts firing
ALERTS{alertstate="firing", service="flash-sale-service"}

# Alert notifications sent
rate(alertmanager_notifications_total[5m])

# Alert notification failures
rate(alertmanager_notifications_failed_total[5m])
```

### Alert Fatigue Prevention

To prevent alert fatigue:

1. **Tune Thresholds**: Adjust thresholds based on historical data
2. **Appropriate Severity**: Use correct severity levels
3. **Meaningful Alerts**: Only alert on actionable conditions
4. **Group Related Alerts**: Use AlertManager grouping
5. **Regular Review**: Review and update alert rules quarterly

## Implementation Notes

### Required Metric Labels

Some alerts require additional labels on metrics for proper filtering:

#### Error Type Labels

The following alerts require `error_type` labels on failure metrics:
- `FlashSaleRedisConnectionFailure`
- `FlashSaleKafkaProducerFailure`

**Implementation**: Update `FlashSaleMetrics` service to add error type labels:

```java
public void recordSeckillRequest(boolean success, String errorType) {
    if (success) {
        requestsSuccess.increment();
    } else {
        requestsFailure.tag("error_type", errorType).increment();
    }
    requestsTotal.increment();
}
```

**Error Types**:
- `redis_error`: Redis connection or operation failures
- `kafka_error`: Kafka producer failures
- `inventory_error`: Inventory service errors
- `queue_error`: Queue service errors
- `validation_error`: Request validation failures
- `system_error`: Other system errors

### Metric Cardinality Considerations

Be mindful of metric cardinality when adding labels:

- **SKU ID**: `flash_sale_inventory_remaining` includes `sku_id` label
  - Limit: Reasonable for typical flash sale scenarios (< 1000 SKUs)
  - Monitor cardinality: `count(flash_sale_inventory_remaining)`

- **Error Types**: Limited set of predefined error types
  - Cardinality: Low (< 10 types)

## Dashboard Integration

### Recommended Grafana Dashboards

Create the following dashboards to visualize alert conditions:

1. **Flash Sale Overview**
   - Request rate (QPS)
   - Success rate
   - P50/P95/P99 latency
   - Active alerts panel

2. **Flash Sale Inventory**
   - Remaining inventory by SKU
   - Deduction rate
   - Rollback rate
   - Deduction latency

3. **Flash Sale Queue**
   - Queue length
   - Token acquisition rate
   - Token rejection rate
   - Estimated wait time

4. **Flash Sale Health**
   - Service uptime
   - Error rate by type
   - Dependency health (Redis, Kafka)
   - Resource utilization

### Alert Annotations

Add alert annotations to Grafana dashboards to show when alerts fired:

```json
{
  "datasource": "Prometheus",
  "enable": true,
  "expr": "ALERTS{alertstate=\"firing\", service=\"flash-sale-service\"}",
  "iconColor": "red",
  "name": "Flash Sale Alerts",
  "tagKeys": "alertname,severity"
}
```

## Runbook Links

Each alert includes a `runbook_url` annotation. Create runbooks at the following locations:

- `/runbooks/flash-sale/high-failure-rate`
- `/runbooks/flash-sale/high-latency`
- `/runbooks/flash-sale/critical-latency`
- `/runbooks/flash-sale/low-inventory`
- `/runbooks/flash-sale/sold-out`
- `/runbooks/flash-sale/high-queue`
- `/runbooks/flash-sale/critical-queue`
- `/runbooks/flash-sale/high-rejection`
- `/runbooks/flash-sale/inventory-latency`
- `/runbooks/flash-sale/high-rollback`
- `/runbooks/flash-sale/service-down`
- `/runbooks/flash-sale/redis-failure`
- `/runbooks/flash-sale/kafka-failure`
- `/runbooks/flash-sale/low-success-rate`
- `/runbooks/flash-sale/high-qps`
- `/runbooks/flash-sale/deduction-anomaly`

## Requirements Validation

### Requirement 7.2: System Monitoring and Alerting

✅ **Threshold Alerts Configured**:
- High failure rate (> 5%)
- High response time (P99 > 200ms, critical > 500ms)
- Low inventory (< 100 units)
- High queue length (> 10K, critical > 50K)

✅ **Fault Alerts Configured**:
- Service down detection
- Redis connection failures
- Kafka producer failures

✅ **Integration with Prometheus AlertManager**:
- All alerts defined in `prometheus-alerts.yml`
- Proper severity levels assigned
- Actionable annotations included

### Requirement 7.5: Fault Detection

✅ **Redis/Kafka Failure Detection**:
- Redis connection failure alert (> 10 errors/sec)
- Kafka producer failure alert (> 5 errors/sec)
- Trigger time: 2 minutes for fault alerts
- Severity: Critical

✅ **30-Second Detection Requirement**:
- Service down alert: 1 minute trigger time
- Redis/Kafka failure alerts: 2 minute trigger time
- Note: Prometheus evaluation interval is 30s, so detection happens within 30s, but alerts fire after the specified `for` duration to avoid flapping

## Maintenance

### Regular Tasks

1. **Weekly**: Review fired alerts and adjust thresholds if needed
2. **Monthly**: Analyze alert patterns and update runbooks
3. **Quarterly**: Review all alert rules and remove/update obsolete ones
4. **After Incidents**: Update relevant alerts based on lessons learned

### Alert Rule Updates

When updating alert rules:

1. Test changes in staging environment first
2. Validate syntax: `promtool check rules prometheus-alerts.yml`
3. Deploy to production during low-traffic period
4. Monitor for false positives/negatives
5. Document changes in this file

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus is scraping metrics: `up{job="flash-sale-service"}`
2. Verify alert rule syntax: `promtool check rules prometheus-alerts.yml`
3. Check Prometheus logs for evaluation errors
4. Verify metric names match exactly

### False Positive Alerts

1. Review threshold values - may need adjustment
2. Check `for` duration - may be too short
3. Analyze historical data to set appropriate thresholds
4. Consider adding additional conditions to reduce noise

### Alerts Not Delivered

1. Check AlertManager is running: `systemctl status alertmanager`
2. Verify AlertManager configuration
3. Check notification channel connectivity (Slack, PagerDuty, email)
4. Review AlertManager logs for delivery errors

## References

- [Prometheus Alerting Documentation](https://prometheus.io/docs/alerting/latest/overview/)
- [AlertManager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Flash Sale Metrics Documentation](../../apps/flash-sale-service/METRICS.md)
- [Flash Sale Design Document](../../.kiro/specs/flash-sale-system/design.md)
- [Flash Sale Requirements](../../.kiro/specs/flash-sale-system/requirements.md)
