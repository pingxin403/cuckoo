# Capacity Monitoring Setup Guide

## Overview

This guide covers the capacity monitoring Grafana dashboards and Prometheus alerts configured for the multi-region active-active IM system. The capacity monitoring system tracks resource usage across regions and provides forecasting to prevent capacity issues.

## Components

### 1. Grafana Dashboards

#### Capacity Usage Trends Dashboard
**Location**: `deploy/docker/grafana/dashboards/capacity-usage-trends.json`  
**UID**: `capacity-usage-trends`  
**URL**: http://grafana:3000/d/capacity-usage-trends

**Features**:
- Real-time resource usage percentage by type and region
- MySQL storage usage trends (GB)
- Kafka topic storage usage trends (GB)
- Network bandwidth usage (MB/s)
- Collection success rate gauge
- Collection duration metrics (P95)

**Variables**:
- `resource_type`: Filter by resource type (mysql, kafka, redis, network)
- `region_id`: Filter by region (region-a, region-b)

**Refresh**: 30 seconds

#### Capacity Forecast Dashboard
**Location**: `deploy/docker/grafana/dashboards/capacity-forecast.json`  
**UID**: `capacity-forecast`  
**URL**: http://grafana:3000/d/capacity-forecast

**Features**:
- Current resource usage gauge with threshold indicators
- Days until capacity full gauge
- Resource growth rate trends (GB/day)
- 30-day capacity forecast projection
- Table of resources above 80% threshold

**Variables**:
- `resource_type`: Filter by resource type
- `region_id`: Filter by region
- `resource_name`: Filter by specific resource

**Refresh**: 1 minute

### 2. Prometheus Alerts

**Location**: `deploy/docker/prometheus-alerts.yml`  
**Group**: `capacity_alerts`

#### Alert Rules

| Alert Name | Severity | Threshold | Duration | Description |
|------------|----------|-----------|----------|-------------|
| `HighResourceUsage` | warning | ≥80% | 5m | Resource usage above 80% threshold |
| `CriticalResourceUsage` | critical | ≥90% | 2m | Resource usage above 90% threshold |
| `CapacityFullSoon` | warning | ≤7 days | 10m | Capacity will be full within 7 days |
| `CapacityFullImminently` | critical | ≤3 days | 5m | Capacity will be full within 3 days |
| `HighCapacityCollectionErrorRate` | warning | >10% | 5m | High error rate in capacity collection |
| `HighMySQLStorageGrowth` | warning | >10GB/day | 30m | MySQL storage growing too fast |
| `HighKafkaStorageGrowth` | warning | >5GB/day | 30m | Kafka topic storage growing too fast |
| `HighNetworkBandwidthUsage` | warning | ≥70% | 10m | Network bandwidth usage high |

#### Alert Annotations

All alerts include:
- **summary**: Brief description of the alert
- **description**: Detailed information with metric values
- **runbook_url**: Link to troubleshooting documentation
- **dashboard_url**: Link to relevant Grafana dashboard
- **action**: Recommended action to take

## Metrics

### Resource Usage Metrics

```promql
# Current resource usage percentage
capacity_resource_usage_percent{resource_type, region_id, resource_name}

# Resource usage in bytes
capacity_resource_usage_bytes{resource_type, region_id, resource_name}

# Total resource capacity in bytes
capacity_resource_total_bytes{resource_type, region_id, resource_name}
```

### Forecast Metrics

```promql
# Days until resource reaches 100% capacity
capacity_forecast_days_until_full{resource_type, region_id, resource_name}

# Current usage percentage (from forecast calculation)
capacity_forecast_current_usage_percent{resource_type, region_id, resource_name}

# Growth rate in bytes per day
capacity_forecast_growth_rate_bytes_per_day{resource_type, region_id, resource_name}
```

### Collection Metrics

```promql
# Successful collections counter
capacity_collection_success_total{resource_type, region_id}

# Failed collections counter
capacity_collection_errors_total{resource_type, region_id}

# Collection duration histogram
capacity_collection_duration_seconds{resource_type, region_id}
```

## Setup Instructions

### 1. Verify Prometheus Configuration

Ensure Prometheus is configured to load the alert rules:

```yaml
# prometheus.yml
rule_files:
  - /etc/prometheus/prometheus-alerts.yml
```

### 2. Validate Alert Rules

Run the validation script:

```bash
./deploy/docker/validate-capacity-alerts.sh
```

Expected output:
```
✓ All capacity monitoring alerts are properly configured
✓ Alert thresholds match requirements (80% warning, 90% critical)
✓ Forecast alerts configured (7 days warning, 3 days critical)
✓ Resource-specific alerts configured (MySQL, Kafka, Network)
```

### 3. Configure Alertmanager

Add routing rules for capacity alerts:

```yaml
# alertmanager.yml
route:
  routes:
    - match:
        service: capacity-monitor
      receiver: capacity-team
      group_by: ['alertname', 'region_id', 'resource_type']
      group_wait: 30s
      group_interval: 5m
      repeat_interval: 4h

receivers:
  - name: capacity-team
    slack_configs:
      - channel: '#capacity-alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

### 4. Import Grafana Dashboards

The dashboards are automatically provisioned if using the Docker Compose setup. To manually import:

1. Open Grafana (http://localhost:3000)
2. Navigate to Dashboards → Import
3. Upload the JSON files:
   - `capacity-usage-trends.json`
   - `capacity-forecast.json`

### 5. Configure Capacity Monitor

Ensure the capacity monitor is running and collecting metrics:

```go
// Example configuration
monitor := capacity.NewDefaultCapacityMonitor(
    capacity.ThresholdConfig{
        DefaultPercent: 80.0,
        Overrides: map[capacity.ResourceType]float64{
            capacity.ResourceNetwork: 70.0, // Lower threshold for network
        },
    },
    historyStore,
)

// Register collectors
monitor.RegisterCollector(capacity.ResourceMySQL, mysqlCollector)
monitor.RegisterCollector(capacity.ResourceKafka, kafkaCollector)
monitor.RegisterCollector(capacity.ResourceNetwork, networkCollector)

// Start periodic collection
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        usages, err := monitor.CollectUsage(ctx, regionID)
        if err != nil {
            log.Error("Failed to collect capacity usage", "error", err)
            continue
        }
        
        // Check thresholds
        exceeded := monitor.CheckThresholds(ctx, usages)
        if len(exceeded) > 0 {
            log.Warn("Resources exceeding threshold", "count", len(exceeded))
        }
    }
}()
```

## Testing

### 1. Test Alert Rules

Use Prometheus's `promtool` to test alert rules:

```bash
promtool test rules deploy/docker/test-capacity-alerts.yml
```

### 2. Simulate High Usage

Temporarily adjust thresholds to trigger alerts:

```bash
# In Prometheus, evaluate:
capacity_resource_usage_percent{resource_type="mysql"} >= 80
```

### 3. Verify Dashboard Data

Check that metrics are being collected:

```bash
# Query Prometheus
curl 'http://localhost:9090/api/v1/query?query=capacity_resource_usage_percent'
```

## Troubleshooting

### No Data in Dashboards

**Symptoms**: Dashboards show "No data"

**Solutions**:
1. Verify capacity monitor is running
2. Check Prometheus is scraping metrics:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```
3. Verify metrics are being exposed:
   ```bash
   curl http://localhost:8080/metrics | grep capacity_
   ```

### Alerts Not Firing

**Symptoms**: Expected alerts not appearing in Alertmanager

**Solutions**:
1. Check alert rule syntax:
   ```bash
   promtool check rules prometheus-alerts.yml
   ```
2. Verify alert evaluation in Prometheus UI (http://localhost:9090/alerts)
3. Check Alertmanager configuration and routing rules

### Collection Errors

**Symptoms**: High `capacity_collection_errors_total` metric

**Solutions**:
1. Check database connectivity
2. Verify collector permissions
3. Review collector logs for specific errors
4. Ensure resource endpoints are accessible

### Forecast Inaccurate

**Symptoms**: Forecast predictions don't match actual usage

**Solutions**:
1. Ensure at least 7 days of historical data
2. Check for data gaps in history store
3. Verify linear regression calculation
4. Consider seasonal patterns (may need more sophisticated forecasting)

## Maintenance

### Regular Tasks

1. **Weekly**: Review capacity forecast dashboard
2. **Monthly**: Validate alert thresholds are appropriate
3. **Quarterly**: Review and update retention policies
4. **Annually**: Audit capacity planning accuracy

### Updating Thresholds

To adjust alert thresholds, edit `prometheus-alerts.yml`:

```yaml
# Example: Change warning threshold to 75%
- alert: HighResourceUsage
  expr: capacity_resource_usage_percent >= 75  # Changed from 80
```

Then reload Prometheus:
```bash
curl -X POST http://localhost:9090/-/reload
```

## Requirements Validation

This implementation satisfies the following requirements:

- ✅ **7.1.1**: Capacity_Monitor collects MySQL storage usage, table rows, and disk growth rate
- ✅ **7.1.2**: Capacity_Monitor collects Kafka topic message backlog and partition disk usage
- ✅ **7.1.3**: Capacity_Monitor collects cross-region network bandwidth and transfer bytes
- ✅ **7.1.4**: Capacity_Monitor triggers alerts when resource usage exceeds 80% threshold
- ✅ **7.1.5**: Capacity_Monitor calculates linear regression forecast and outputs days until capacity limit

## References

- [Capacity Monitor Implementation](../../libs/capacity/)
- [Prometheus Alert Rules](./prometheus-alerts.yml)
- [Grafana Dashboards](./grafana/dashboards/)
- [Multi-Region Design Document](../../.kiro/specs/multi-region-active-active/design.md)
- [Requirements Document](../../.kiro/specs/multi-region-active-active/requirements.md)
