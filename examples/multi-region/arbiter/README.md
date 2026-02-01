# Arbiter - Distributed Coordination for Multi-Region Active-Active

The arbiter package provides distributed coordination and split-brain prevention for the multi-region active-active IM system. It uses Apache Zookeeper as the consensus mechanism to ensure only one region acts as the primary at any given time.

## Features

- **Distributed Leader Election**: Uses Zookeeper distributed locks to elect a primary region
- **Health-Based Arbitration**: Elections consider the health status of all regions
- **Split-Brain Prevention**: Ensures only one region can be primary, even during network partitions
- **Deterministic Elections**: Consistent election results using predefined region preferences
- **Health Monitoring**: Tracks and reports health status of critical services
- **Leader Change Notifications**: Watch for leadership changes in real-time

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Region A      │    │   Zookeeper     │    │   Region B      │
│                 │    │   (Third AZ)    │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Arbiter   │◄┼────┼►│ Distributed │◄┼────┼►│   Arbiter   │ │
│ │   Client    │ │    │ │    Locks    │ │    │ │   Client    │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Health    │ │    │ │   Health    │ │    │ │   Health    │ │
│ │  Checker    │ │    │ │  Reports    │ │    │ │  Checker    │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Usage

### Basic Setup

```go
import "github.com/example/im-system/arbiter"

// Create arbiter client
config := arbiter.Config{
    ZookeeperHosts: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
    RegionID:       "region-a",
    SessionTimeout: 10 * time.Second,
    ElectionTTL:    30 * time.Second,
}

client, err := arbiter.NewArbiterClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Health Reporting

```go
// Report health status of critical services
healthStatus := map[string]bool{
    "im-service": true,
    "redis":      true,
    "database":   true,
}

err := client.ReportHealth(healthStatus)
if err != nil {
    log.Printf("Failed to report health: %v", err)
}
```

### Leader Election

```go
ctx := context.Background()

// Perform leader election
result, err := client.ElectPrimary(ctx, healthStatus)
if err != nil {
    log.Printf("Election failed: %v", err)
    return
}

if result.IsPrimary {
    log.Printf("This region is PRIMARY - handling write traffic")
    // Enable write operations
} else {
    log.Printf("This region is SECONDARY (leader: %s)", result.Leader)
    // Switch to read-only mode
}
```

### Watching Leader Changes

```go
// Watch for leader changes
go func() {
    err := client.WatchLeaderChanges(ctx, func(leader string) {
        log.Printf("Leader changed to: %s", leader)
        // Handle leadership change
    })
    if err != nil {
        log.Printf("Error watching leader changes: %v", err)
    }
}()
```

## Election Rules

The arbiter implements the following election rules to ensure deterministic and stable leadership:

1. **Current Leader Preference**: If the current primary is still healthy, it remains the leader
2. **No Healthy Regions**: If no regions are healthy, no leader is elected (read-only mode)
3. **Deterministic Selection**: When multiple regions are healthy, prefer `region-a` over `region-b`
4. **Fallback**: If neither preferred region is available, elect the first healthy region

## Health Criteria

A region is considered healthy if all critical services are operational:

- **im-service**: Core IM application service
- **redis**: Session state and caching
- **database**: Persistent data storage

Additional services can be monitored but don't affect the health determination.

## Split-Brain Prevention

The arbiter prevents split-brain scenarios through several mechanisms:

### Distributed Consensus
- Uses Zookeeper's distributed locks for leader election
- Only one region can hold the primary lock at a time
- Network partitions automatically trigger failover

### Health-Based Decisions
- Elections consider real-time health status
- Unhealthy regions cannot become primary
- Health reports have TTL to detect stale data

### Deterministic Rules
- Consistent election outcomes across all regions
- Prevents conflicts when multiple regions are healthy
- Clear precedence order for tie-breaking

## Configuration

### Zookeeper Setup

The arbiter requires a Zookeeper cluster deployed in a third availability zone:

```yaml
# docker-compose.yml
zookeeper:
  image: confluentinc/cp-zookeeper:7.4.0
  environment:
    ZOOKEEPER_CLIENT_PORT: 2181
    ZOOKEEPER_TICK_TIME: 2000
  volumes:
    - zookeeper-data:/var/lib/zookeeper/data
```

### Client Configuration

```go
config := arbiter.Config{
    ZookeeperHosts: []string{"zk1:2181", "zk2:2181", "zk3:2181"},
    RegionID:       "region-a",           // Unique region identifier
    SessionTimeout: 10 * time.Second,     // ZK session timeout
    ElectionTTL:    30 * time.Second,     // Leadership TTL
    Logger:         customLogger,         // Optional custom logger
}
```

## Monitoring

### Health Status

```go
// Get current health status
health := client.GetHealthStatus()
isHealthy := client.IsHealthy()

// Get current leader
leader, err := client.GetCurrentLeader(ctx)
```

### Election History

```go
// Get recent election events
history, err := client.GetElectionHistory(ctx, 10)
for _, event := range history {
    fmt.Printf("Leader: %s, Time: %v, Reason: %s\n", 
        event["leader"], event["timestamp"], event["reason"])
}
```

### Metrics Integration

The arbiter can be integrated with Prometheus for monitoring:

```go
// Custom metrics (implement as needed)
var (
    electionCount = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "arbiter_elections_total",
        Help: "Total number of elections performed",
    })
    
    healthCheckDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "arbiter_health_check_duration_seconds",
        Help: "Duration of health checks",
    })
)
```

## Error Handling

### Connection Failures

```go
client, err := arbiter.NewArbiterClient(config)
if err != nil {
    // Handle connection failure
    // - Check Zookeeper connectivity
    // - Verify network configuration
    // - Fall back to read-only mode
}
```

### Election Failures

```go
result, err := client.ElectPrimary(ctx, healthStatus)
if err != nil {
    // Handle election failure
    // - Log the error
    // - Retry with backoff
    // - Switch to read-only mode
}
```

### Network Partitions

When a region loses connectivity to Zookeeper:

1. The region cannot participate in elections
2. It should switch to read-only mode
3. Health checks continue locally
4. When connectivity is restored, normal operation resumes

## Testing

### Unit Tests

```bash
go test ./arbiter
```

### Integration Tests

Requires Zookeeper to be running:

```bash
export ZK_TEST_HOSTS="localhost:2181"
go test ./arbiter -v
```

### Docker Compose Testing

```bash
cd deploy/mvp
docker-compose up -d zookeeper
export ZK_TEST_HOSTS="localhost:2181"
go test ./arbiter -v
```

## Performance Considerations

### Election Frequency
- Elections should be triggered by health changes, not on a timer
- Typical election frequency: 1-2 per minute under normal conditions
- During failures: may increase to several per minute

### Zookeeper Load
- Each region reports health every 10-30 seconds
- Elections use distributed locks (low overhead)
- Watch operations are efficient for leader change notifications

### Network Latency
- Elections complete within 1-5 seconds under normal conditions
- Cross-region latency affects election time
- Zookeeper should be deployed in a central location

## Troubleshooting

### Common Issues

1. **Connection Timeouts**
   - Check Zookeeper connectivity
   - Verify network configuration
   - Increase session timeout if needed

2. **Split Elections**
   - Verify clock synchronization (NTP)
   - Check Zookeeper cluster health
   - Review election logs for conflicts

3. **Stale Leaders**
   - Check TTL configuration
   - Verify health reporting frequency
   - Monitor Zookeeper session health

### Debug Logging

Enable debug logging for troubleshooting:

```go
logger := log.New(os.Stdout, "[ARBITER-DEBUG] ", log.LstdFlags|log.Lshortfile)
config.Logger = logger
```

### Health Check Debugging

```go
// Check individual service health
health := client.GetHealthStatus()
for service, healthy := range health {
    log.Printf("Service %s: %v", service, healthy)
}

// Overall health status
log.Printf("Region healthy: %v", client.IsHealthy())
```

## Security Considerations

### Zookeeper Security
- Use authentication (SASL/Kerberos) in production
- Enable TLS for encrypted communication
- Restrict network access to Zookeeper ports

### Access Control
- Implement proper ACLs in Zookeeper
- Limit which services can perform elections
- Monitor election events for anomalies

## Production Deployment

### High Availability
- Deploy Zookeeper cluster with 3+ nodes
- Use separate availability zones for Zookeeper nodes
- Monitor Zookeeper cluster health

### Capacity Planning
- Zookeeper: 2-4 CPU cores, 4-8GB RAM per node
- Network: Low latency connection between regions and Zookeeper
- Storage: Fast SSD for Zookeeper transaction logs

### Monitoring
- Monitor election frequency and duration
- Alert on election failures or split-brain conditions
- Track health check success rates
- Monitor Zookeeper cluster metrics

## Related Components

- **Health Checker**: Provides health status input for elections
- **Message Syncer**: Responds to leadership changes for data sync
- **Geo Router**: Uses leadership information for traffic routing
- **Monitoring**: Collects arbiter metrics and events