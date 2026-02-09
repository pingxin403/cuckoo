# Multi-Region Active-Active Alerting Guide

## Overview

This document describes the Prometheus alerting rules for the multi-region active-active architecture. The alerts are designed to provide proactive monitoring and early warning for cross-region synchronization issues, conflicts, and failover events.

## Alert Configuration File

**File**: `prometheus-multi-region-alerts.yml`

This file contains 6 alert groups covering all aspects of multi-region monitoring:

1. **Sync Latency Alerts** - Cross-region synchronization performance
2. **Conflict Alerts** - Data conflict detection and resolution
3. **Failover Alerts** - Failover event detection and performance
4. **Health Alerts** - Region availability and health checks
5. **Data Sync Alerts** - Data synchronization status and network partitions
6. **Reconciliation Alerts** - Data reconciliation and consistency
7. **SLO Alerts** - Service level objective compliance

## Requirements Traceability

| Requirement | Alert Group | Status |
|------------|-------------|--------|
| 5.1.1 - Monitor sync latency P50/P95/P99 | Sync Latency Alerts | ✅ |
| 5.1.2 - Monitor database replication delay | Sync Latency Alerts | ✅ |
| 5.1.3 - Alert when latency exceeds threshold | Sync Latency Alerts | ✅ |
| 5.2.1 - Count conflicts per minute | Conflict Alerts | ✅ |
| 5.2.2 - Classify conflicts by type | Conflict Alerts | ✅ |
| 5.2.3 - Alert when conflict rate > 0.1% | Conflict Alerts | ✅ |
| 5.3.1 - Record detection time, reason, impact | Failover Alerts | ✅ |
| 5.3.2 - Record failover duration | Failover Alerts | ✅ |
| 5.3.3 - Support event replay | Failover Alerts | ✅ |
| 4.1 - Automatic failure detection | Health Alerts | ✅ |
| 4.2 - Automatic failover | Failover Alerts | ✅ |
| 4.3 - Split-brain prevention | Data Sync Alerts | ✅ |
| 4.4 - Data reconciliation | Reconciliation Alerts | ✅ |

## Alert Groups

### 1. Sync Latency Alerts (Requirement 5.1)

Monitors cross-region synchronization latency to ensure data is replicated within acceptable timeframes.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `HighCrossRegionSyncLatency` | Warning | P99 > 500ms | 5m | General sync latency exceeds threshold |
| `CriticalCrossRegionSyncLatency` | Critical | P99 > 1000ms | 2m | Critical sync latency - possible network partition |
| `HighMessageSyncLatency` | Warning | P95 > 400ms | 5m | Message sync latency high |
| `HighDatabaseReplicationLatency` | Warning | P95 > 800ms | 5m | Database replication lag high |
| `SyncLatencyThresholdExceeded` | Warning | > 0.01/sec | 3m | Frequent latency threshold violations |

#### Example PromQL Queries

```promql
# P99 sync latency
histogram_quantile(0.99, rate(cross_region_sync_latency_ms_bucket[5m]))

# Message sync latency by region
histogram_quantile(0.95, rate(message_sync_latency_ms_bucket{source_region="region-a"}[5m]))

# Database replication lag
histogram_quantile(0.95, rate(database_replication_latency_ms_bucket[5m]))
```

#### Runbook Actions

**HighCrossRegionSyncLatency**:
1. Check network connectivity between regions
2. Review sync queue backlog
3. Verify Kafka replication lag
4. Check for resource constraints (CPU, memory, network)

**CriticalCrossRegionSyncLatency**:
1. Immediate investigation required
2. Check for network partition
3. Verify service health in both regions
4. Consider manual failover if persistent

### 2. Conflict Alerts (Requirement 5.2)

Monitors data conflict rate and resolution performance to ensure data consistency.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `HighConflictRate` | Critical | > 0.1% | 2m | Conflict rate exceeds acceptable threshold |
| `ElevatedConflictRate` | Warning | > 0.05% | 5m | Elevated conflict rate detected |
| `HighMessageConflictRate` | Warning | > 1/sec | 5m | High message conflict rate |
| `HighSessionConflictRate` | Warning | > 0.5/sec | 5m | High session conflict rate |
| `SlowConflictResolution` | Warning | P95 > 100ms | 5m | Conflict resolution taking too long |
| `ConflictResolutionFailures` | Critical | > 0.1/sec | 3m | Conflict resolution failures |

#### Example PromQL Queries

```promql
# Conflict rate (percentage)
(rate(cross_region_conflicts_total[5m]) * 60) / (rate(cross_region_sync_total[5m]) * 60)

# Conflicts by type
rate(cross_region_conflicts_total[5m]) by (conflict_type)

# Conflict resolution duration
histogram_quantile(0.95, rate(conflict_resolution_duration_ms_bucket[5m]))
```

#### Runbook Actions

**HighConflictRate**:
1. Investigate conflict causes
2. Check for clock skew between regions
3. Review concurrent write patterns
4. Verify HLC clock synchronization
5. Consider adjusting write routing strategy

**ConflictResolutionFailures**:
1. Check conflict resolver logs
2. Verify database connectivity
3. Review LWW strategy implementation
4. Check for data corruption

### 3. Failover Alerts (Requirement 5.3)

Monitors failover events and performance to ensure RTO targets are met.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `FailoverEventDetected` | Critical | Any event | Immediate | Failover event occurred |
| `SlowFailoverDetected` | Critical | P95 > 30s | 1m | Failover duration exceeds RTO |
| `SlowFailoverDetection` | Warning | P95 > 15s | 2m | Slow failure detection |
| `FrequentFailoverEvents` | Critical | > 3 in 10m | 5m | Possible failover flapping |
| `FailoverDetectionThresholdExceeded` | Warning | > 0.01/sec | 3m | Frequent detection delays |

#### Example PromQL Queries

```promql
# Failover events
increase(failover_events_total[1h])

# Failover duration
histogram_quantile(0.95, rate(failover_duration_ms_bucket[5m]))

# Failover detection time
histogram_quantile(0.95, rate(failover_detection_time_ms_bucket[5m]))

# Failover events by reason
increase(failover_events_total[1h]) by (reason)
```

#### Runbook Actions

**FailoverEventDetected**:
1. Verify failover completed successfully
2. Check target region health
3. Monitor system metrics
4. Review failover logs
5. Notify on-call team

**SlowFailoverDetected**:
1. Review failover process
2. Optimize detection time
3. Check DNS propagation
4. Verify health check configuration
5. Consider pre-warming standby region

**FrequentFailoverEvents**:
1. Investigate root cause
2. Check for network instability
3. Review health check thresholds
4. Verify service stability
5. Consider increasing health check intervals

### 4. Health Alerts

Monitors region availability and health check performance.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `RegionUnavailable` | Critical | Availability = 0 | 1m | Region is unavailable |
| `HighHealthCheckLatency` | Warning | P95 > 1000ms | 5m | Health check latency high |
| `HealthCheckFailures` | Warning | > 10% failures | 3m | High health check failure rate |

#### Example PromQL Queries

```promql
# Region availability
region_availability

# Health check latency
histogram_quantile(0.95, rate(health_check_latency_ms_bucket[5m]))

# Health check failure rate
rate(health_checks_total{status="unhealthy"}[5m]) / rate(health_checks_total[5m])
```

#### Runbook Actions

**RegionUnavailable**:
1. Verify region health
2. Check for automatic failover initiation
3. Review service logs
4. Check infrastructure status
5. Escalate to infrastructure team

### 5. Data Sync Alerts

Monitors data synchronization status and network partitions.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `HighDataSyncFailureRate` | Critical | > 5% failures | 5m | High data sync failure rate |
| `NetworkPartitionDetected` | Critical | Any event | Immediate | Network partition detected |
| `LongNetworkPartition` | Critical | P95 > 60s | 2m | Long network partition duration |

#### Example PromQL Queries

```promql
# Data sync failure rate
rate(data_sync_status_total{status="failure"}[5m]) / rate(data_sync_status_total[5m])

# Network partition events
increase(network_partition_events_total[1h])

# Network partition duration
histogram_quantile(0.95, rate(network_partition_duration_ms_bucket[5m]))
```

#### Runbook Actions

**NetworkPartitionDetected**:
1. Verify split-brain prevention mechanisms
2. Monitor data consistency
3. Check network connectivity
4. Review etcd cluster status
5. Prepare for data reconciliation after recovery

### 6. Reconciliation Alerts (Requirement 4.4)

Monitors data reconciliation operations and consistency.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `ReconciliationDiscrepanciesFound` | Warning | > 100 | 5m | Discrepancies found during reconciliation |
| `HighReconciliationDiscrepancies` | Critical | > 1000 | 2m | High number of discrepancies |
| `ReconciliationFailures` | Warning | No events in 1h | 1h | Reconciliation task not running |
| `SlowReconciliation` | Warning | P95 > 5min | 10m | Reconciliation taking too long |
| `LowAutoRepairSuccessRate` | Warning | < 90% | 10m | Low auto-repair success rate |

#### Example PromQL Queries

```promql
# Reconciliation discrepancies
reconciliation_discrepancies

# Reconciliation events
increase(reconciliation_events_total[1h])

# Reconciliation duration
histogram_quantile(0.95, rate(reconciliation_duration_ms_bucket[1h]))

# Auto-repair success rate
reconciliation_fixed_count / reconciliation_discrepancies
```

#### Runbook Actions

**ReconciliationDiscrepanciesFound**:
1. Review reconciliation report
2. Verify auto-repair completion
3. Check for data sync issues
4. Monitor conflict rate

**HighReconciliationDiscrepancies**:
1. Immediate investigation required
2. Check for data sync failure
3. Review sync logs
4. Verify database replication status
5. Consider manual data verification

### 7. SLO Alerts

Monitors service level objective compliance.

#### Alerts

| Alert Name | Severity | Threshold | Duration | Description |
|-----------|----------|-----------|----------|-------------|
| `CrossRegionRTOViolation` | Critical | 30-day P95 > 30s | 5m | RTO SLO violated |
| `CrossRegionRPOViolation` | Critical | 30-day P99 > 1s | 5m | RPO SLO violated |
| `MultiRegionAvailabilitySLOViolation` | Critical | 30-day < 99.99% | 5m | Availability SLO violated |

#### Example PromQL Queries

```promql
# RTO compliance
histogram_quantile(0.95, rate(failover_duration_ms_bucket[30d]))

# RPO compliance
histogram_quantile(0.99, rate(message_sync_latency_ms_bucket[30d]))

# Availability
avg_over_time(region_availability[30d])
```

## Integration with Prometheus

### 1. Add Alert Rules to Prometheus Configuration

Edit `prometheus.yml` to include the multi-region alert rules:

```yaml
rule_files:
  - "prometheus-alerts.yml"
  - "prometheus-health-alerts.yml"
  - "prometheus-multi-region-alerts.yml"  # Add this line
```

### 2. Reload Prometheus Configuration

```bash
# Validate configuration
./validate-prometheus-alerts.sh

# Reload Prometheus
curl -X POST http://localhost:9090/-/reload
```

### 3. Verify Alert Rules

```bash
# Check alert rules are loaded
curl http://localhost:9090/api/v1/rules | jq '.data.groups[] | select(.name | contains("multi_region"))'

# Check active alerts
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | select(.labels.service == "multi-region")'
```

## Integration with Alertmanager

### Configure Alert Routing

Edit `alertmanager-config.yml` to add multi-region alert routing:

```yaml
route:
  receiver: 'default-receiver'
  group_by: ['alertname', 'service', 'region']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  
  routes:
    # Multi-region critical alerts
    - match:
        service: multi-region
        severity: critical
      receiver: 'multi-region-critical'
      group_wait: 0s
      group_interval: 5m
      repeat_interval: 4h
      
    # Multi-region warning alerts
    - match:
        service: multi-region
        severity: warning
      receiver: 'multi-region-warning'
      group_wait: 30s
      group_interval: 10m
      repeat_interval: 12h

receivers:
  - name: 'multi-region-critical'
    webhook_configs:
      - url: 'http://alerting-service:8080/webhook/critical'
    pagerduty_configs:
      - service_key: '<pagerduty-service-key>'
    slack_configs:
      - api_url: '<slack-webhook-url>'
        channel: '#multi-region-critical'
        
  - name: 'multi-region-warning'
    slack_configs:
      - api_url: '<slack-webhook-url>'
        channel: '#multi-region-warnings'
```

## Alert Severity Levels

| Severity | Response Time | Escalation | Examples |
|----------|--------------|------------|----------|
| **Critical** | Immediate | Page on-call | Failover events, high conflict rate, region unavailable |
| **Warning** | 15 minutes | Slack notification | Elevated latency, slow reconciliation |
| **Info** | Best effort | Log only | Reconciliation completed, inventory depleted |

## Alert Notification Templates

### Slack Notification Template

```yaml
slack_configs:
  - api_url: '<slack-webhook-url>'
    channel: '#multi-region-alerts'
    title: '{{ .GroupLabels.alertname }}'
    text: |
      *Summary:* {{ .CommonAnnotations.summary }}
      *Description:* {{ .CommonAnnotations.description }}
      *Severity:* {{ .CommonLabels.severity }}
      *Service:* {{ .CommonLabels.service }}
      *Component:* {{ .CommonLabels.component }}
      *Action:* {{ .CommonAnnotations.action }}
      *Runbook:* {{ .CommonAnnotations.runbook_url }}
      *Dashboard:* {{ .CommonAnnotations.dashboard_url }}
```

### PagerDuty Integration

```yaml
pagerduty_configs:
  - service_key: '<pagerduty-service-key>'
    description: '{{ .CommonAnnotations.summary }}'
    details:
      severity: '{{ .CommonLabels.severity }}'
      service: '{{ .CommonLabels.service }}'
      component: '{{ .CommonLabels.component }}'
      description: '{{ .CommonAnnotations.description }}'
      action: '{{ .CommonAnnotations.action }}'
      runbook_url: '{{ .CommonAnnotations.runbook_url }}'
      dashboard_url: '{{ .CommonAnnotations.dashboard_url }}'
```

## Testing Alerts

### 1. Test Alert Rules Syntax

```bash
# Validate alert rules
promtool check rules deploy/docker/prometheus-multi-region-alerts.yml
```

### 2. Simulate Alert Conditions

```bash
# Simulate high sync latency
curl -X POST http://localhost:9090/api/v1/admin/tsdb/delete_series \
  -d 'match[]=cross_region_sync_latency_ms'

# Inject test metrics
cat <<EOF | curl --data-binary @- http://localhost:9090/api/v1/write
cross_region_sync_latency_ms{source_region="region-a",target_region="region-b"} 600
EOF
```

### 3. Test Alertmanager Routing

```bash
# Send test alert
amtool alert add \
  --alertmanager.url=http://localhost:9093 \
  --annotation=summary="Test alert" \
  --annotation=description="This is a test" \
  alertname=TestAlert \
  service=multi-region \
  severity=warning
```

## Monitoring Alert Health

### Alert Manager Metrics

```promql
# Alerts firing
ALERTS{alertstate="firing", service="multi-region"}

# Alert notifications sent
alertmanager_notifications_total

# Alert notification failures
alertmanager_notifications_failed_total
```

### Alert Fatigue Prevention

1. **Proper Thresholds**: Set thresholds based on actual system behavior
2. **Appropriate Durations**: Use `for` clause to avoid flapping alerts
3. **Alert Grouping**: Group related alerts to reduce noise
4. **Runbook Links**: Provide clear action items for each alert
5. **Regular Review**: Review and tune alerts based on feedback

## Best Practices

### 1. Alert Design

- **Actionable**: Every alert should have a clear action
- **Specific**: Include relevant context (region, component, etc.)
- **Timely**: Alert before user impact when possible
- **Documented**: Link to runbooks and dashboards

### 2. Alert Tuning

- **Baseline**: Establish normal behavior baselines
- **Iterate**: Adjust thresholds based on false positives/negatives
- **Seasonal**: Consider time-of-day and day-of-week patterns
- **Review**: Regular alert review meetings

### 3. Alert Response

- **Acknowledge**: Acknowledge alerts promptly
- **Investigate**: Follow runbook procedures
- **Document**: Document findings and actions taken
- **Improve**: Update runbooks and alerts based on learnings

## Troubleshooting

### Alerts Not Firing

1. Check Prometheus is scraping metrics:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

2. Verify alert rules are loaded:
   ```bash
   curl http://localhost:9090/api/v1/rules
   ```

3. Check alert evaluation:
   ```bash
   curl http://localhost:9090/api/v1/alerts
   ```

### Alerts Not Notifying

1. Check Alertmanager is receiving alerts:
   ```bash
   curl http://localhost:9093/api/v1/alerts
   ```

2. Verify routing configuration:
   ```bash
   amtool config routes show
   ```

3. Check notification logs:
   ```bash
   docker logs alertmanager
   ```

### False Positives

1. Review alert threshold and duration
2. Check for metric collection issues
3. Verify system behavior is normal
4. Adjust alert parameters if needed

## References

- [Prometheus Alerting Documentation](https://prometheus.io/docs/alerting/latest/overview/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Multi-Region Metrics README](../../libs/metrics/README.md)
- [Multi-Region Design Document](../../.kiro/specs/multi-region-active-active/design.md)
- [Multi-Region Requirements](../../.kiro/specs/multi-region-active-active/requirements.md)

## Changelog

### 2024-01-XX - Initial Release
- Created comprehensive multi-region alerting rules
- Implemented 6 alert groups covering all requirements
- Added documentation and runbooks
- Integrated with existing Prometheus/Alertmanager setup

