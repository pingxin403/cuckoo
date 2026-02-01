# Multi-Region Monitoring Dashboard

This package provides a web-based monitoring dashboard for the multi-region active-active architecture.

## Features

- **Real-time Metrics**: Auto-refreshing dashboard with 5-second intervals
- **HLC Clock Monitoring**: Track Hybrid Logical Clock status across regions
- **Conflict Resolution Metrics**: Monitor conflict rates and resolution performance
- **Sync Performance**: Track cross-region synchronization latency and throughput
- **System Health**: Overall health status based on key performance indicators

## Components

### WebDashboard

The main dashboard component that provides:

- HTTP server with web interface
- REST API for metrics
- Health check endpoint
- Real-time metric collection

### Metrics Collected

#### HLC Metrics
- Physical time (milliseconds)
- Logical time counter
- Region and node identification
- Sequence numbers

#### Conflict Resolution Metrics
- Total conflicts detected
- Local vs remote wins
- Average resolution time
- Conflict rate (conflicts/second)

#### Synchronization Metrics
- Async sync count
- Sync sync count
- Average latency
- Sync rate (messages/second)
- Error count

#### System Health
- Overall status (healthy/degraded/critical)
- Conflict rate monitoring
- Sync latency P99 approximation
- Error rate calculation

## Usage

### Basic Setup

```go
package main

import (
    "github.com/cuckoo-org/cuckoo/monitoring"
    "github.com/cuckoo-org/cuckoo/libs/hlc"
    "github.com/cuckoo-org/cuckoo/sync"
)

func main() {
    // Initialize components
    hlcClock := hlc.NewHLC("region-a", "node-1")
    conflictResolver := sync.NewConflictResolver(config, logger)
    messageSyncer := sync.NewMessageSyncer(...)
    
    // Create dashboard
    dashboard := monitoring.NewWebDashboard(8090, hlcClock, conflictResolver, messageSyncer)
    
    // Start dashboard
    go dashboard.Start()
    
    // Dashboard available at http://localhost:8090
}
```

### Running the Example

```bash
# Run the example dashboard
go run monitoring/example_dashboard.go

# Visit http://localhost:8090 to view the dashboard
```

## API Endpoints

### GET /
Main dashboard HTML interface with real-time metrics display.

### GET /api/metrics
Returns JSON metrics data:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "region_id": "region-a",
  "hlc_metrics": {
    "physical_time_ms": 1705312200000,
    "logical_time": 42,
    "region_id": "region-a",
    "node_id": "node-1",
    "sequence": 1234
  },
  "conflict_metrics": {
    "total_conflicts": 5,
    "local_wins": 3,
    "remote_wins": 2,
    "avg_resolution_time_us": 150.5,
    "conflict_rate": 0.0001
  },
  "sync_metrics": {
    "async_sync_count": 1000,
    "sync_sync_count": 50,
    "avg_latency_ms": 45.2,
    "sync_rate": 10.5,
    "error_count": 2
  },
  "system_health": {
    "status": "healthy",
    "conflict_rate": 0.0001,
    "sync_latency_p99_ms": 67.8,
    "error_rate": 0.002
  }
}
```

### GET /health
Simple health check endpoint:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "region": "region-a"
}
```

## Health Status Thresholds

The dashboard calculates system health based on the following thresholds:

### Healthy
- Conflict rate < 0.05%
- Sync latency < 500ms
- Error rate < 0.5%

### Degraded
- Conflict rate 0.05% - 0.1%
- Sync latency 500ms - 1000ms
- Error rate 0.5% - 1%

### Critical
- Conflict rate > 0.1%
- Sync latency > 1000ms
- Error rate > 1%

## Integration with Prometheus/Grafana

While this dashboard provides a simple web interface, for production monitoring you should also integrate with Prometheus and Grafana:

1. **Prometheus Metrics**: Expose metrics in Prometheus format
2. **Grafana Dashboards**: Use the provided Grafana dashboard configuration
3. **Alerting**: Configure alerts based on the same thresholds

See the `deploy/mvp/` directory for Prometheus and Grafana configurations.

## Development

### Adding New Metrics

1. Add the metric to the appropriate struct (HLCMetrics, ConflictMetrics, etc.)
2. Update the collection method in `collectMetrics()`
3. Update the HTML template to display the new metric
4. Update the JavaScript rendering function

### Customizing the UI

The dashboard uses a simple HTML template with inline CSS and JavaScript. You can customize:

- Colors and styling in the CSS section
- Metric display format in the JavaScript
- Refresh interval (currently 5 seconds)
- Layout and grid structure

## Security Considerations

- The dashboard runs on HTTP by default (suitable for internal networks)
- For production, consider adding HTTPS support
- No authentication is implemented (add if needed for production)
- CORS is enabled for API endpoints (restrict as needed)

## Performance

- Metrics collection is lightweight and non-blocking
- Dashboard auto-refresh can be disabled if needed
- Consider caching metrics for high-traffic scenarios
- Memory usage is minimal for the web interface