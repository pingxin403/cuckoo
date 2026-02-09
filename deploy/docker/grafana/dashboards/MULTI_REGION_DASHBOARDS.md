# Multi-Region Active-Active Grafana Dashboards

Comprehensive monitoring dashboards for the multi-region active-active architecture, providing real-time visibility into cross-region synchronization, conflict resolution, failover events, and system health.

## Overview

This collection includes 4 specialized dashboards designed to monitor different aspects of the multi-region system:

1. **Multi-Region Sync Latency** - Cross-region synchronization performance
2. **Multi-Region Conflict Rate** - Data conflict detection and resolution
3. **Multi-Region Failover Events** - Disaster recovery and failover tracking
4. **Multi-Region System Health Overview** - Comprehensive system health status

## Dashboard Details

### 1. Multi-Region Sync Latency

**File**: `multi-region-sync-latency.json`  
**UID**: `multi-region-sync-latency`  
**Requirements**: 5.1.1, 5.1.2, 5.1.3

#### Purpose
Monitor cross-region synchronization latency to ensure data replication meets performance requirements (P99 < 500ms).

#### Panels

1. **Cross-Region Sync Latency (P50/P95/P99)**
   - Displays general sync latency percentiles
   - Threshold: Yellow at 300ms, Red at 500ms
   - Query: `histogram_quantile(0.99, sum(rate(cross_region_sync_latency_ms_bucket[5m])) by (le, source_region, target_region))`

2. **Message Sync Latency (P50/P95/P99)**
   - Message-specific synchronization latency
   - Tracks real-time message replication performance
   - Query: `histogram_quantile(0.99, sum(rate(message_sync_latency_ms_bucket[5m])) by (le, source_region, target_region))`

3. **Database Replication Latency (P50/P95/P99)**
   - Database replication delay monitoring
   - Threshold: Yellow at 800ms, Red at 1000ms
   - Query: `histogram_quantile(0.99, sum(rate(database_replication_latency_ms_bucket[5m])) by (le, source_region, target_region))`

4. **Sync Operations Rate**
   - Total sync operations per second
   - Helps identify sync load patterns
   - Query: `rate(cross_region_sync_total[5m])`

5. **Latency Threshold Violations**
   - Gauge showing threshold exceedances (>500ms)
   - Alerts when sync latency consistently exceeds SLA
   - Query: `sum(increase(message_sync_latency_threshold_exceeded_total[5m]))`

#### Variables
- `$source_region` - Source region for sync operations
- `$target_region` - Target region for sync operations

#### Use Cases
- Monitor sync performance during normal operations
- Identify network degradation between regions
- Validate SLA compliance (P99 < 500ms)
- Troubleshoot slow replication issues

---

### 2. Multi-Region Conflict Rate

**File**: `multi-region-conflict-rate.json`  
**UID**: `multi-region-conflict-rate`  
**Requirements**: 5.2.1, 5.2.2, 5.2.3

#### Purpose
Track data conflicts and resolution effectiveness to ensure system consistency and identify problematic conflict patterns.

#### Panels

1. **Conflict Rate (per minute)**
   - Real-time conflict rate across all regions
   - Threshold: Yellow at 0.05/min, Red at 0.1/min (0.1%)
   - Query: `rate(cross_region_conflicts_total[5m]) * 60`

2. **Conflicts by Type (Last Hour)**
   - Pie chart showing conflict distribution
   - Helps identify most common conflict types
   - Query: `sum by (conflict_type) (increase(cross_region_conflicts_total[1h]))`

3. **Conflict Resolution Rate**
   - Bar chart of resolution outcomes
   - Shows local_wins, remote_wins, manual resolution
   - Query: `rate(conflict_resolutions_total[5m])`

4. **Conflict Resolution Duration**
   - Time taken to resolve conflicts (P50/P95/P99)
   - Threshold: Yellow at 10ms, Red at 50ms
   - Query: `histogram_quantile(0.99, sum(rate(conflict_resolution_duration_ms_bucket[5m])) by (le, conflict_type))`

5. **Current Conflict Rate (%)**
   - Gauge showing conflict rate as percentage of operations
   - Alert threshold: 0.1% (requirement 5.2.3)
   - Query: `rate(cross_region_conflicts_total[5m]) / (rate(cross_region_sync_total[5m]) + 1)`

6. **Conflict Trends (10-minute windows)**
   - Historical conflict trends by region
   - Helps identify patterns and anomalies
   - Query: `sum by (region) (increase(cross_region_conflicts_total[10m]))`

#### Variables
- `$region` - Region to monitor conflicts

#### Use Cases
- Monitor conflict rate against 0.1% threshold
- Identify conflict hotspots and patterns
- Validate LWW conflict resolution effectiveness
- Alert on abnormal conflict rates

---

### 3. Multi-Region Failover Events

**File**: `multi-region-failover-events.json`  
**UID**: `multi-region-failover-events`  
**Requirements**: 5.3.1, 5.3.2, 5.3.3, 4.1, 4.2

#### Purpose
Track all failover events with detailed metadata for disaster recovery analysis and RTO/RPO validation.

#### Panels

1. **Failover Events Timeline**
   - Bar chart showing failover events over time
   - Annotated with event details
   - Query: `increase(failover_events_total[5m])`

2. **Failover Events by Reason (Last 24h)**
   - Donut chart categorizing failover causes
   - Reasons: health_check_failed, manual, network_partition, etc.
   - Query: `sum by (reason) (increase(failover_events_total[24h]))`

3. **Total Failovers (24h)**
   - Stat panel showing total failover count
   - Threshold: Yellow at 1, Red at 5
   - Query: `sum(increase(failover_events_total[24h]))`

4. **Mean Time Between Failovers**
   - MTBF calculation
   - Green if > 24 hours, Yellow if > 1 hour
   - Query: `86400 / (sum(increase(failover_events_total[24h])) + 1)`

5. **Failover Duration (RTO)**
   - Failover completion time (P50/P95/P99)
   - SLA: RTO < 30 seconds (30000ms)
   - Query: `histogram_quantile(0.99, sum(rate(failover_duration_ms_bucket[5m])) by (le, from_region, to_region))`

6. **Failure Detection Time**
   - Time to detect failure before triggering failover
   - Threshold: Yellow at 10s, Red at 15s
   - Query: `histogram_quantile(0.99, sum(rate(failover_detection_time_ms_bucket[5m])) by (le, region))`

7. **Detection Time Threshold Violations**
   - Gauge showing times detection exceeded 15s
   - Validates requirement 4.1.1 (detect within 15s)
   - Query: `sum(increase(failover_detection_time_threshold_exceeded_total[1h]))`

8. **Recent Failover Events**
   - Table showing recent failover history
   - Supports event replay (requirement 5.3.3)
   - Query: `changes(failover_events_total[24h])`

#### Variables
- `$from_region` - Source region for failover
- `$to_region` - Target region for failover

#### Use Cases
- Validate RTO < 30 seconds requirement
- Analyze failover causes and patterns
- Track system reliability (MTBF)
- Post-incident analysis and reporting
- Verify automatic failure detection (< 15s)

---

### 4. Multi-Region System Health Overview

**File**: `multi-region-health-overview.json`  
**UID**: `multi-region-health-overview`  
**Requirements**: 4.1, 5.1, 5.2, 5.3, 4.4

#### Purpose
Comprehensive dashboard providing at-a-glance view of entire multi-region system health and key metrics.

#### Panels

1. **Region Availability Status**
   - Stat panel showing current availability of each region
   - Green (1) = Available, Red (0) = Unavailable
   - Query: `region_availability{region="$region"}`

2. **Health Check Success Rate**
   - Gauge showing percentage of successful health checks
   - Threshold: Red < 95%, Yellow < 99%, Green ≥ 99%
   - Query: `sum(rate(health_checks_total{status="healthy"}[5m])) / sum(rate(health_checks_total[5m]))`

3. **Sync Latency P99**
   - Cross-region sync latency overview
   - Quick view of replication performance
   - Query: `histogram_quantile(0.99, sum(rate(cross_region_sync_latency_ms_bucket[5m])) by (le, source_region, target_region))`

4. **Conflict Rate**
   - Current conflict rate per minute
   - Monitors data consistency health
   - Query: `rate(cross_region_conflicts_total[5m]) * 60`

5. **Health Check Latency P95**
   - Latency to check peer region health
   - Threshold: Yellow at 100ms, Red at 200ms
   - Query: `histogram_quantile(0.95, sum(rate(health_check_latency_ms_bucket[5m])) by (le, source_region, target_region, status))`

6. **Data Sync Status**
   - Bar chart showing sync operation outcomes
   - Status: success, failure, pending
   - Query: `rate(data_sync_status_total[5m])`

7. **Network Partition Events**
   - Bar chart of network partition occurrences
   - Critical for split-brain prevention monitoring
   - Query: `increase(network_partition_events_total[5m])`

8. **Reconciliation Status**
   - Data reconciliation discrepancies and fixes
   - Validates requirement 4.4 (data reconciliation)
   - Query: `reconciliation_discrepancies{source_region="$region"}` and `reconciliation_fixed_count{source_region="$region"}`

9. **System Health Summary**
   - Multi-stat panel with key metrics:
     - Sync Latency P99 (ms)
     - Conflict Rate (%)
     - Failovers (24h)
     - Health Check Success (%)
   - Color-coded thresholds for quick assessment

#### Variables
- `$region` - Region to monitor

#### Use Cases
- Daily operations monitoring
- Quick health assessment
- Incident detection and triage
- Executive dashboard for system status
- SLA compliance verification

---

## Installation

### Automatic Provisioning

The dashboards are automatically provisioned when using Docker Compose:

```yaml
# deploy/docker/docker-compose.observability.yml
grafana:
  volumes:
    - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    - ./grafana/provisioning:/etc/grafana/provisioning
```

### Manual Import

1. Open Grafana UI (default: http://localhost:3000)
2. Navigate to **Dashboards** → **Import**
3. Upload the JSON file or paste the JSON content
4. Select **Prometheus** as the data source
5. Click **Import**

## Configuration

### Data Source

All dashboards require a Prometheus data source named **"Prometheus"**. Configure in Grafana:

1. **Configuration** → **Data Sources** → **Add data source**
2. Select **Prometheus**
3. Set URL: `http://prometheus:9090` (Docker) or `http://localhost:9090` (local)
4. Click **Save & Test**

### Variables

Each dashboard includes template variables for filtering:

- **Region variables**: Filter by source/target region
- **Auto-refresh**: Dashboards refresh every 10 seconds
- **Time range**: Default 1 hour (configurable)

### Alerts

Configure Prometheus alerting rules in `deploy/docker/prometheus-alerts.yml`:

```yaml
groups:
  - name: multi_region_alerts
    rules:
      - alert: HighCrossRegionSyncLatency
        expr: histogram_quantile(0.99, cross_region_sync_latency_ms) > 500
        for: 5m
        
      - alert: HighConflictRate
        expr: rate(cross_region_conflicts_total[5m]) > 0.001
        for: 2m
        
      - alert: FailoverEventDetected
        expr: increase(failover_events_total[1m]) > 0
```

## Metrics Reference

### Histograms
- `cross_region_sync_latency_ms` - General sync latency
- `message_sync_latency_ms` - Message sync latency
- `database_replication_latency_ms` - DB replication latency
- `conflict_resolution_duration_ms` - Conflict resolution time
- `failover_duration_ms` - Failover completion time
- `failover_detection_time_ms` - Failure detection time
- `health_check_latency_ms` - Health check latency
- `network_partition_duration_ms` - Partition duration
- `reconciliation_duration_ms` - Reconciliation time

### Counters
- `cross_region_sync_total` - Total sync operations
- `message_sync_latency_threshold_exceeded_total` - Latency violations
- `cross_region_conflicts_total` - Total conflicts
- `conflict_resolutions_total` - Total resolutions
- `failover_events_total` - Total failovers
- `failover_detection_time_threshold_exceeded_total` - Detection violations
- `health_checks_total` - Total health checks
- `data_sync_status_total` - Sync status counts
- `network_partition_events_total` - Partition events
- `reconciliation_events_total` - Reconciliation runs

### Gauges
- `region_availability` - Region availability (0 or 1)
- `reconciliation_discrepancies` - Current discrepancies
- `reconciliation_fixed_count` - Fixed discrepancies

## Troubleshooting

### No Data Displayed

1. **Check Prometheus connection**:
   ```bash
   curl http://localhost:9090/api/v1/query?query=up
   ```

2. **Verify metrics are being collected**:
   ```bash
   curl http://localhost:9090/api/v1/label/__name__/values | grep cross_region
   ```

3. **Check service is exposing metrics**:
   ```bash
   curl http://localhost:9090/metrics | grep cross_region
   ```

### Incorrect Values

1. **Verify time range** - Ensure sufficient data in selected time window
2. **Check aggregation** - Some queries use `rate()` which requires 2+ data points
3. **Validate labels** - Ensure `source_region`, `target_region` labels exist

### Dashboard Not Loading

1. **Check Grafana logs**:
   ```bash
   docker logs grafana
   ```

2. **Verify JSON syntax**:
   ```bash
   jq . multi-region-sync-latency.json
   ```

3. **Check provisioning**:
   ```bash
   docker exec grafana ls /etc/grafana/provisioning/dashboards
   ```

## Best Practices

### Monitoring Strategy

1. **Start with System Health Overview** - Get overall system status
2. **Drill down to specific dashboards** - Investigate anomalies
3. **Use time range controls** - Compare current vs historical performance
4. **Set up alerts** - Proactive notification of issues

### Performance Optimization

1. **Adjust refresh rate** - Reduce from 10s to 30s for less critical dashboards
2. **Limit time range** - Use shorter windows (1h vs 24h) for faster queries
3. **Use recording rules** - Pre-compute expensive queries in Prometheus

### Customization

1. **Clone dashboards** - Create custom versions without modifying originals
2. **Add annotations** - Mark deployments, incidents, maintenance windows
3. **Create playlists** - Rotate through dashboards on wall displays
4. **Export/Import** - Share customized dashboards across environments

## Requirements Traceability

| Requirement | Dashboard | Panel |
|------------|-----------|-------|
| 5.1.1 - Monitor sync latency P50/P95/P99 | Sync Latency | Cross-Region Sync Latency |
| 5.1.2 - Monitor DB replication delay | Sync Latency | Database Replication Latency |
| 5.1.3 - Alert on latency threshold | Sync Latency | Latency Threshold Violations |
| 5.2.1 - Count conflicts per minute | Conflict Rate | Conflict Rate (per minute) |
| 5.2.2 - Classify conflicts by type | Conflict Rate | Conflicts by Type |
| 5.2.3 - Alert when rate > 0.1% | Conflict Rate | Current Conflict Rate |
| 5.3.1 - Record detection time, reason | Failover Events | Failure Detection Time |
| 5.3.2 - Record failover duration | Failover Events | Failover Duration (RTO) |
| 5.3.3 - Support event replay | Failover Events | Recent Failover Events |
| 4.1 - Automatic failure detection | Health Overview | Health Check Success Rate |
| 4.2 - Automatic failover RTO < 30s | Failover Events | Failover Duration (RTO) |
| 4.4 - Data reconciliation | Health Overview | Reconciliation Status |

## Related Documentation

- [Multi-Region Metrics Package](../../../../libs/metrics/README.md) - Metrics implementation details
- [Multi-Region Architecture Design](../../../../.kiro/specs/multi-region-active-active/design.md) - System architecture
- [Prometheus Alerting Rules](../prometheus-alerts.yml) - Alert configurations
- [Observability Guide](../OBSERVABILITY.md) - General observability setup

## Support

For issues or questions:
1. Check [Troubleshooting](#troubleshooting) section
2. Review Grafana logs: `docker logs grafana`
3. Verify Prometheus metrics: `http://localhost:9090/targets`
4. Consult metrics package documentation

## Version History

- **v1.0** (2024-01) - Initial release with 4 dashboards
  - Multi-Region Sync Latency
  - Multi-Region Conflict Rate
  - Multi-Region Failover Events
  - Multi-Region System Health Overview
