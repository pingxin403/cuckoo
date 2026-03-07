# Monitoring Dashboard Guide

## 📊 Overview

This guide provides comprehensive documentation for monitoring the multi-region active-active architecture. It includes dashboard configurations, key metrics, alerting rules, and troubleshooting tips.

## 🎯 Key Performance Indicators (KPIs)

### System Health KPIs

| Metric | Target | Critical Threshold | Description |
|--------|--------|-------------------|-------------|
| **System Availability** | 99.99% | < 99.9% | Overall system uptime |
| **Cross-Region Sync Latency (P99)** | < 500ms | > 1000ms | Message sync time between regions |
| **Failover RTO** | < 30s | > 60s | Recovery time objective |
| **Message RPO** | < 1s | > 5s | Recovery point objective |
| **Conflict Rate** | < 0.1% | > 1% | Percentage of conflicting writes |

## 📈 Grafana Dashboards

### Dashboard 1: Multi-Region Overview

**URL**: http://localhost:3000/d/multi-region-overview

**Panels**:

#### 1.1 System Health Status
```yaml
Panel Type: Stat
Query: up{job="im-service"}
Thresholds:
  - Green: 1 (healthy)
  - Red: 0 (down)
Display: Current value + sparkline
```

#### 1.2 Active Connections by Region
```yaml
Panel Type: Time Series
Queries:
  - active_websocket_connections{region="region-a"}
  - active_websocket_connections{region="region-b"}
Legend: {{region}}
Y-axis: Connections
```

#### 1.3 Cross-Region Sync Latency
```yaml
Panel Type: Time Series
Queries:
  - histogram_quantile(0.50, cross_region_sync_latency_ms)
  - histogram_quantile(0.95, cross_region_sync_latency_ms)
  - histogram_quantile(0.99, cross_region_sync_latency_ms)
Legend: P50, P95, P99
Y-axis: Milliseconds
Alert: P99 > 500ms
```

#### 1.4 Message Throughput
```yaml
Panel Type: Time Series
Queries:
  - rate(messages_sent_total{region="region-a"}[5m])
  - rate(messages_sent_total{region="region-b"}[5m])
Legend: {{region}}
Y-axis: Messages/sec
```

#### 1.5 Conflict Rate
```yaml
Panel Type: Time Series
Query: rate(cross_region_conflicts_total[5m])
Y-axis: Conflicts/sec
Alert: rate > 0.001 (0.1%)
Color: Red when alerting
```

#### 1.6 Failover Events
```yaml
Panel Type: Stat
Query: increase(failover_events_total[1h])
Display: Current value
Color: Red if > 0
```

### Dashboard 2: Performance Metrics

**URL**: http://localhost:3000/d/multi-region-performance

**Panels**:

#### 2.1 Message Processing Duration
```yaml
Panel Type: Heatmap
Query: message_processing_duration_ms_bucket
Y-axis: Duration (ms)
X-axis: Time
Color: Gradient (blue to red)
```

#### 2.2 Database Query Latency
```yaml
Panel Type: Time Series
Queries:
  - histogram_quantile(0.99, db_query_duration_ms{operation="insert"})
  - histogram_quantile(0.99, db_query_duration_ms{operation="select"})
  - histogram_quantile(0.99, db_query_duration_ms{operation="update"})
Legend: {{operation}} P99
```

#### 2.3 Redis Cache Hit Rate
```yaml
Panel Type: Gauge
Query: rate(redis_cache_hits[5m]) / (rate(redis_cache_hits[5m]) + rate(redis_cache_misses[5m]))
Min: 0
Max: 1
Thresholds:
  - Red: < 0.8
  - Yellow: 0.8 - 0.95
  - Green: > 0.95
```

#### 2.4 Kafka Consumer Lag
```yaml
Panel Type: Time Series
Query: kafka_consumer_lag{topic="cross_region_sync"}
Y-axis: Messages
Alert: lag > 1000
```

#### 2.5 Network Bandwidth Usage
```yaml
Panel Type: Time Series
Queries:
  - rate(network_bytes_sent{region="region-a"}[5m])
  - rate(network_bytes_received{region="region-a"}[5m])
Legend: Sent, Received
Y-axis: Bytes/sec
```

### Dashboard 3: Conflict Analysis

**URL**: http://localhost:3000/d/multi-region-conflicts

**Panels**:

#### 3.1 Conflicts by Type
```yaml
Panel Type: Pie Chart
Query: sum by (conflict_type) (cross_region_conflicts_total)
Legend: {{conflict_type}}
```

#### 3.2 Conflict Resolution Time
```yaml
Panel Type: Time Series
Query: histogram_quantile(0.99, conflict_resolution_duration_ms)
Y-axis: Milliseconds
```

#### 3.3 Tiebreaker Usage
```yaml
Panel Type: Stat
Query: increase(conflict_tiebreaker_used_total[1h])
Display: Current value
Info: "High usage may indicate clock sync issues"
```

#### 3.4 Conflict Timeline
```yaml
Panel Type: Time Series
Query: increase(cross_region_conflicts_total[1m])
Y-axis: Conflicts
Annotations: Deployment events, failover events
```

### Dashboard 4: Infrastructure Health

**URL**: http://localhost:3000/d/multi-region-infrastructure

**Panels**:

#### 4.1 CPU Usage by Service
```yaml
Panel Type: Time Series
Query: rate(process_cpu_seconds_total[5m]) * 100
Legend: {{service}} - {{region}}
Y-axis: Percentage
Alert: > 80%
```

#### 4.2 Memory Usage by Service
```yaml
Panel Type: Time Series
Query: process_resident_memory_bytes / 1024 / 1024
Legend: {{service}} - {{region}}
Y-axis: MB
Alert: > 80% of limit
```

#### 4.3 Disk I/O
```yaml
Panel Type: Time Series
Queries:
  - rate(disk_reads_total[5m])
  - rate(disk_writes_total[5m])
Legend: Reads, Writes
Y-axis: Operations/sec
```

#### 4.4 etcd Cluster Health
```yaml
Panel Type: Stat
Query: etcd_server_has_leader
Display: Current value
Thresholds:
  - Green: 1 (has leader)
  - Red: 0 (no leader)
```

## 🔔 Alerting Rules

### Critical Alerts

#### 1. Region Down
```yaml
alert: RegionDown
expr: up{job="im-service"} == 0
for: 1m
labels:
  severity: critical
annotations:
  summary: "Region {{$labels.region}} is down"
  description: "IM Service in {{$labels.region}} has been down for 1 minute"
  runbook: "Check service logs and restart if necessary"
```

#### 2. High Sync Latency
```yaml
alert: HighCrossRegionSyncLatency
expr: histogram_quantile(0.99, cross_region_sync_latency_ms) > 1000
for: 5m
labels:
  severity: critical
annotations:
  summary: "Cross-region sync latency too high"
  description: "P99 sync latency is {{$value}}ms (threshold: 1000ms)"
  runbook: "Check network connectivity and Kafka lag"
```

#### 3. Failover Event
```yaml
alert: FailoverEventDetected
expr: increase(failover_events_total[1m]) > 0
labels:
  severity: critical
annotations:
  summary: "Failover event detected"
  description: "Failover from {{$labels.from_region}} to {{$labels.to_region}}"
  runbook: "Investigate cause and verify system stability"
```

### Warning Alerts

#### 4. High Conflict Rate
```yaml
alert: HighConflictRate
expr: rate(cross_region_conflicts_total[5m]) > 0.001
for: 2m
labels:
  severity: warning
annotations:
  summary: "Cross-region conflict rate > 0.1%"
  description: "Conflict rate is {{$value}} per second"
  runbook: "Analyze conflict logs and check for clock sync issues"
```

#### 5. Kafka Consumer Lag
```yaml
alert: HighKafkaConsumerLag
expr: kafka_consumer_lag{topic="cross_region_sync"} > 1000
for: 5m
labels:
  severity: warning
annotations:
  summary: "Kafka consumer lag too high"
  description: "Consumer lag is {{$value}} messages"
  runbook: "Scale up consumers or check processing bottlenecks"
```

#### 6. Low Cache Hit Rate
```yaml
alert: LowCacheHitRate
expr: rate(redis_cache_hits[5m]) / (rate(redis_cache_hits[5m]) + rate(redis_cache_misses[5m])) < 0.8
for: 10m
labels:
  severity: warning
annotations:
  summary: "Redis cache hit rate below 80%"
  description: "Cache hit rate is {{$value}}"
  runbook: "Review cache strategy and consider warming cache"
```

### Info Alerts

#### 7. Frequent Tiebreaker Usage
```yaml
alert: FrequentTiebreakerUsage
expr: rate(conflict_tiebreaker_used_total[5m]) > 0.0001
for: 5m
labels:
  severity: info
annotations:
  summary: "RegionID Tiebreaker frequently used"
  description: "May indicate clock synchronization issues"
  runbook: "Check NTP sync status on all servers"
```

## 📊 Prometheus Queries

### Essential Queries

#### System Health
```promql
# Overall system availability
avg(up{job="im-service"})

# Services by region
count by (region) (up{job="im-service"} == 1)

# Service uptime
time() - process_start_time_seconds
```

#### Performance
```promql
# Message throughput
sum(rate(messages_sent_total[5m])) by (region)

# Sync latency percentiles
histogram_quantile(0.50, cross_region_sync_latency_ms)
histogram_quantile(0.95, cross_region_sync_latency_ms)
histogram_quantile(0.99, cross_region_sync_latency_ms)

# Processing duration
histogram_quantile(0.99, message_processing_duration_ms)
```

#### Conflicts
```promql
# Total conflicts
sum(cross_region_conflicts_total)

# Conflict rate
rate(cross_region_conflicts_total[5m])

# Conflicts by type
sum by (conflict_type) (cross_region_conflicts_total)

# Tiebreaker usage
rate(conflict_tiebreaker_used_total[5m])
```

#### Infrastructure
```promql
# CPU usage
rate(process_cpu_seconds_total[5m]) * 100

# Memory usage
process_resident_memory_bytes / 1024 / 1024

# Goroutines
go_goroutines

# GC duration
rate(go_gc_duration_seconds_sum[5m])
```

## 🖼️ Dashboard Screenshots

### Screenshot 1: Multi-Region Overview
![Multi-Region Overview](./screenshots/grafana-overview.png)

**Key Elements**:
- System health status (green/red indicators)
- Active connections by region (time series)
- Cross-region sync latency (P50/P95/P99)
- Message throughput (messages/sec)
- Conflict rate (conflicts/sec)
- Recent failover events

### Screenshot 2: Cross-Region Sync Latency
![Cross-Region Latency](./screenshots/cross-region-latency.png)

**Key Elements**:
- P50, P95, P99 latency over time
- Target threshold line (500ms)
- Alert annotations when threshold exceeded
- Latency breakdown by operation

### Screenshot 3: Conflict Analysis
![Conflict Rate](./screenshots/conflict-rate.png)

**Key Elements**:
- Conflict rate over time
- Conflicts by type (pie chart)
- Tiebreaker usage counter
- Conflict resolution duration

### Screenshot 4: Failover Events
![Failover Events](./screenshots/failover-events.png)

**Key Elements**:
- Failover event timeline
- Failover duration (RTO)
- Source and target regions
- Impact on active connections

## 🔍 Log Analysis

### Structured Logging Format

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "service": "im-service",
  "region": "region-a",
  "message": "Message synced",
  "global_id": "region-a-1704067200000-0-1",
  "sync_latency_ms": 250,
  "target_region": "region-b"
}
```

### Useful Log Queries

#### Conflict Logs
```bash
# View all conflicts in the last hour
grep "conflict detected" /var/log/im-service.log | jq 'select(.timestamp > "2024-01-01T11:00:00Z")'

# Conflicts by type
grep "conflict detected" /var/log/im-service.log | jq -r '.conflict_type' | sort | uniq -c
```

#### Failover Logs
```bash
# View failover events
grep "failover" /var/log/im-gateway.log | jq

# Failover duration
grep "failover completed" /var/log/im-gateway.log | jq '.duration_ms'
```

#### High Latency Logs
```bash
# Messages with sync latency > 500ms
grep "Message synced" /var/log/im-service.log | jq 'select(.sync_latency_ms > 500)'
```

## 🛠️ Troubleshooting with Dashboards

### Issue 1: High Sync Latency

**Dashboard**: Multi-Region Performance

**Steps**:
1. Check "Cross-Region Sync Latency" panel
2. If P99 > 500ms, investigate:
   - Network latency between regions
   - Kafka consumer lag
   - Database replication lag
3. Check "Kafka Consumer Lag" panel
4. Review "Network Bandwidth Usage" panel

### Issue 2: High Conflict Rate

**Dashboard**: Conflict Analysis

**Steps**:
1. Check "Conflicts by Type" pie chart
2. Review "Conflict Timeline" for patterns
3. Check "Tiebreaker Usage" - high usage indicates clock issues
4. Investigate application logic causing concurrent writes

### Issue 3: Service Degradation

**Dashboard**: Infrastructure Health

**Steps**:
1. Check "CPU Usage by Service" - look for spikes
2. Review "Memory Usage by Service" - check for leaks
3. Check "Goroutines" - look for goroutine leaks
4. Review "GC Duration" - excessive GC indicates memory pressure

## 📱 Mobile Dashboard Access

### Grafana Mobile App

1. Install Grafana Mobile App (iOS/Android)
2. Add server: http://your-grafana-url:3000
3. Login with credentials
4. Access dashboards on-the-go
5. Receive push notifications for alerts

## 🔐 Dashboard Access Control

### Role-Based Access

```yaml
# Viewer Role
- Can view all dashboards
- Cannot edit dashboards
- Cannot acknowledge alerts

# Editor Role
- Can view and edit dashboards
- Can create new dashboards
- Can acknowledge alerts

# Admin Role
- Full access to all features
- Can manage users and permissions
- Can configure data sources
```

## 📚 Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Tutorial](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Alerting Best Practices](https://prometheus.io/docs/practices/alerting/)

## 🎓 Training Materials

### Dashboard Tour Video
- Duration: 10 minutes
- Topics: Navigation, key metrics, alert interpretation
- URL: [Internal training portal]

### Runbook Workshop
- Duration: 2 hours
- Topics: Alert response, troubleshooting, escalation
- Schedule: Monthly

---

**Last Updated**: 2024  
**Maintained By**: Platform Engineering Team  
**Next Review**: Quarterly

**Related Documents**:
- [Architecture Overview](./architecture-overview.md)
- [Demo Scenarios](./demo-scenarios.md)
- [Troubleshooting Guide](./troubleshooting-guide.md)
