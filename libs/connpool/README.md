# Connection Pool Library

A comprehensive connection pool management library optimized for cross-region multi-active architectures. Provides unified management of database (MySQL), Redis, and Kafka connections with built-in health checks and automatic optimization.

## Features

- **Unified Pool Management**: Single interface to manage all connection types
- **Cross-Region Optimization**: Optimized configurations for high-latency cross-region scenarios
- **Health Monitoring**: Automatic health checks with configurable thresholds
- **Connection Reuse**: Efficient connection pooling to minimize overhead
- **Metrics Collection**: Built-in metrics for monitoring and debugging
- **Auto-Optimization**: Runtime optimization based on usage patterns
- **Thread-Safe**: Safe for concurrent access from multiple goroutines
- **Cache Warming**: Intelligent preloading of hot data with cross-region synchronization
- **Flexible Invalidation**: Multiple cache invalidation strategies (TTL, LRU, Write-through, Hybrid)
- **Batch Processing**: Efficient batch operations for messages, offline writes, and reconciliation
- **Priority Ordering**: High-priority items processed first
- **Automatic Flushing**: Size-based and time-based batch flushing

## Supported Connection Types

### 1. Database (MySQL)
- Connection pooling with configurable limits
- Automatic connection lifecycle management
- Ping-based health checks
- Connection statistics and metrics

### 2. Redis
- Connection pooling with idle connection management
- Configurable timeouts for read/write operations
- Pool statistics (hits, misses, timeouts)
- Automatic stale connection cleanup
- **Cache warming with hot data preloading**
- **Cross-region cache synchronization**
- **Multiple invalidation strategies**

### 3. Kafka
- **Producer**: Optimized for high throughput with compression
- **Consumer**: Configurable fetch and session parameters
- Idempotent producer support
- Automatic retry with backoff

## Installation

```bash
go get github.com/pingxin403/cuckoo/libs/connpool
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/pingxin403/cuckoo/libs/connpool"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create pool manager with default configuration
    config := connpool.DefaultPoolConfig()
    pm := connpool.NewPoolManager(config)
    defer pm.Stop()
    
    // Start health checks
    if err := pm.Start(); err != nil {
        log.Fatal(err)
    }
    
    // Get database connection
    db, err := pm.GetDatabase("mysql-primary", 
        "user:password@tcp(localhost:3306)/dbname?parseTime=true")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get Redis connection
    redisClient, err := pm.GetRedis("redis-primary", &redis.Options{
        Addr: "localhost:6379",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Get Kafka producer
    producer, err := pm.GetKafkaProducer("kafka-producer", 
        []string{"localhost:9092"})
    if err != nil {
        log.Fatal(err)
    }
    
    // Use connections...
    ctx := context.Background()
    
    // Database query
    var count int
    db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
    
    // Redis operation
    redisClient.Set(ctx, "key", "value", 0)
    
    // Kafka produce
    // producer.SendMessage(...)
}
```

### Cache Warming

```go
// Create cache warmer for Redis pool
warmerConfig := connpool.DefaultCacheWarmerConfig()
warmer, err := pm.GetOrCreateCacheWarmer("redis-primary", warmerConfig)
if err != nil {
    log.Fatal(err)
}

// Manually warm cache with hot data
ctx := context.Background()
hotData := []connpool.HotDataItem{
    {
        Key:         "user:1001",
        Value:       `{"name":"Alice"}`,
        AccessCount: 250,
        TTL:         1 * time.Hour,
        Priority:    10,
    },
}

result := warmer.WarmCache(ctx, hotData)
log.Printf("Warmed %d items, hit rate improved by %.2f%%", 
    result.SuccessCount, result.HitRateImprovement*100)
```

**For detailed cache warming documentation, see [Cache Warming Guide](./CACHE_WARMING_GUIDE.md).**

### Cross-Region Configuration

```go
// Optimized configuration for cross-region deployment
config := connpool.PoolConfig{
    Database: connpool.DatabasePoolConfig{
        MaxOpenConns:    50,  // Higher limit for cross-region load
        MaxIdleConns:    10,  // Keep more idle connections
        ConnMaxLifetime: 5 * time.Minute,
        ConnMaxIdleTime: 2 * time.Minute,
        PingTimeout:     3 * time.Second,
    },
    Redis: connpool.RedisPoolConfig{
        PoolSize:        100, // Increased for high throughput
        MinIdleConns:    20,  // Keep more idle connections
        ConnMaxLifetime: 5 * time.Minute,
        ConnMaxIdleTime: 2 * time.Minute,
        PoolTimeout:     5 * time.Second,
        ReadTimeout:     3 * time.Second,
        WriteTimeout:    3 * time.Second,
    },
    Kafka: connpool.KafkaPoolConfig{
        Producer: connpool.KafkaProducerConfig{
            MaxOpenRequests: 5,
            RequiredAcks:    sarama.WaitForLocal, // Balance performance/reliability
            Timeout:         10 * time.Second,
            Compression:     sarama.CompressionSnappy,
            Idempotent:      true,
            RetryMax:        3,
            RetryBackoff:    100 * time.Millisecond,
        },
        Consumer: connpool.KafkaConsumerConfig{
            SessionTimeout:    10 * time.Second,
            HeartbeatInterval: 3 * time.Second,
            RebalanceTimeout:  60 * time.Second,
            MaxProcessingTime: 30 * time.Second,
            FetchMin:          1,
            FetchDefault:      1024 * 1024, // 1MB
            MaxWaitTime:       500 * time.Millisecond,
        },
    },
    HealthCheck: connpool.HealthCheckConfig{
        Enabled:          true,
        Interval:         30 * time.Second,
        Timeout:          5 * time.Second,
        FailureThreshold: 3,
        SuccessThreshold: 2,
    },
}

pm := connpool.NewPoolManager(config)
```

## Health Monitoring

### Check Health Status

```go
// Get health status of all pools
status := pm.GetHealthStatus()
for name, s := range status {
    log.Printf("Pool %s: healthy=%v, latency=%v", name, s.Healthy, s.Latency)
}

// Get health summary
if pm.healthChecker != nil {
    summary := pm.healthChecker.GetHealthSummary()
    log.Printf("Health: %s", summary.String())
    
    if !summary.OverallHealthy {
        log.Printf("Unhealthy pools: %v", summary.UnhealthyNames)
    }
}
```

### Custom Health Checks

```go
config := connpool.HealthCheckConfig{
    Enabled:          true,
    Interval:         10 * time.Second,  // Check every 10 seconds
    Timeout:          3 * time.Second,   // 3 second timeout
    FailureThreshold: 5,                 // Mark unhealthy after 5 failures
    SuccessThreshold: 3,                 // Mark healthy after 3 successes
}
```

## Metrics

### Database Metrics

```go
// Get database pool
db, _ := pm.GetDatabase("mysql-primary", dsn)

// Access the pool directly for metrics
pool := pm.dbs["mysql-primary"]
metrics := pool.GetMetrics()

log.Printf("Database Metrics:")
log.Printf("  Total Connections: %d", metrics.TotalConnections)
log.Printf("  Active Connections: %d", metrics.ActiveConnections)
log.Printf("  Idle Connections: %d", metrics.IdleConnections)
log.Printf("  Wait Count: %d", metrics.WaitCount)
log.Printf("  Avg Wait Time: %v", metrics.AvgWaitTime)
```

### Redis Metrics

```go
pool := pm.redisClients["redis-primary"]
metrics := pool.GetMetrics()

log.Printf("Redis Metrics:")
log.Printf("  Total Connections: %d", metrics.TotalConns)
log.Printf("  Idle Connections: %d", metrics.IdleConns)
log.Printf("  Hit Rate: %.2f%%", metrics.HitRate*100)
log.Printf("  Timeouts: %d", metrics.TotalTimeouts)
```

### Kafka Metrics

```go
pool := pm.kafkaProducers["kafka-producer"]
metrics := pool.GetMetrics()

log.Printf("Kafka Producer Metrics:")
log.Printf("  Messages Sent: %d", metrics.MessagesSent)
log.Printf("  Messages Failed: %d", metrics.MessagesFailed)
log.Printf("  Success Rate: %.2f%%", metrics.SuccessRate*100)
log.Printf("  Avg Latency: %v", metrics.AvgLatency)
```

### Cache Warmer Metrics

```go
// Get metrics for a specific cache warmer
warmer, _ := pm.GetCacheWarmer("redis-primary")
metrics := warmer.GetMetrics()

log.Printf("Cache Warmer Metrics:")
log.Printf("  Total Warmed: %d", metrics.TotalWarmed)
log.Printf("  Total Failed: %d", metrics.TotalFailed)
log.Printf("  Total Invalidated: %d", metrics.TotalInvalidated)
log.Printf("  Last Warm Time: %v", metrics.LastWarmTime)
log.Printf("  Warm Duration: %v", metrics.WarmDuration)
log.Printf("  Hit Rate Improvement: %.2f%%", metrics.HitRateImprovement*100)

// Get metrics for all cache warmers
allMetrics := pm.GetAllCacheWarmerMetrics()
for name, m := range allMetrics {
    log.Printf("Cache %s: warmed=%d, improvement=%.2f%%", 
        name, m.TotalWarmed, m.HitRateImprovement*100)
}
```

## Best Practices

### 1. Connection Limits

For cross-region deployments:
- **Database**: Set `MaxOpenConns` to 50-100 to handle higher latency
- **Redis**: Set `PoolSize` to 100-200 for high throughput scenarios
- **Kafka**: Use `MaxOpenRequests=5` to balance throughput and memory

### 2. Timeouts

Configure appropriate timeouts for cross-region:
- **Database**: `PingTimeout=3s`, `ConnMaxLifetime=5m`
- **Redis**: `PoolTimeout=5s`, `ReadTimeout=3s`, `WriteTimeout=3s`
- **Kafka**: `Timeout=10s`, `SessionTimeout=10s`

### 3. Health Checks

- Enable health checks in production: `HealthCheck.Enabled=true`
- Set appropriate intervals: `Interval=30s` for production
- Configure thresholds: `FailureThreshold=3`, `SuccessThreshold=2`

### 4. Connection Reuse

- Keep `MaxIdleConns` at 20-30% of `MaxOpenConns`
- Set `ConnMaxIdleTime` to 2-5 minutes
- Monitor `WaitCount` and `WaitDuration` metrics

### 5. Kafka Optimization

- Enable idempotent producer: `Idempotent=true`
- Use compression: `Compression=CompressionSnappy`
- Set appropriate acks: `RequiredAcks=WaitForLocal` for balance

## Testing

### Unit Tests

```bash
go test ./libs/connpool -v
```

### Integration Tests

Requires MySQL, Redis, and Kafka running locally:

```bash
# Start dependencies with Docker Compose
docker-compose -f deploy/docker/docker-compose.infrastructure.yml up -d

# Run integration tests
go test ./libs/connpool -v -tags=integration
```

## Performance Considerations

### Database

- **Connection Pooling**: Reuses connections to avoid TCP handshake overhead
- **Prepared Statements**: Use prepared statements for repeated queries
- **Batch Operations**: Use transactions for multiple operations

### Redis

- **Pipelining**: Use pipelining for multiple commands
- **Connection Pooling**: Maintains idle connections for fast access
- **Compression**: Consider using compression for large values

### Kafka

- **Batching**: Producer batches messages for efficiency
- **Compression**: Snappy compression reduces network bandwidth
- **Idempotency**: Prevents duplicate messages in case of retries

## Troubleshooting

### High Wait Times

If you see high `WaitCount` or `WaitDuration`:
1. Increase `MaxOpenConns`
2. Check for long-running queries/operations
3. Monitor connection lifecycle

### Connection Timeouts

If you see frequent timeouts:
1. Increase timeout values
2. Check network latency
3. Verify service availability

### Memory Usage

If memory usage is high:
1. Reduce `MaxOpenConns` and `PoolSize`
2. Decrease `ConnMaxLifetime`
3. Monitor connection statistics

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    PoolManager                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │  Database    │  │    Redis     │  │    Kafka     │ │
│  │    Pools     │  │    Pools     │  │    Pools     │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                         │
│  ┌─────────────────────────────────────────────────┐  │
│  │           Health Checker                        │  │
│  │  - Periodic health checks                       │  │
│  │  - Failure/success tracking                     │  │
│  │  - Auto-optimization                            │  │
│  └─────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please see CONTRIBUTING.md for guidelines.

## Related Documentation

- [Multi-Region Architecture Design](../../.kiro/specs/multi-region-active-active/design.md)
- [Performance Optimization Guide](../../docs/operations/performance-optimization.md)
- [Health Monitoring Guide](../../docs/operations/health-monitoring.md)


## Batch Processing

The library includes a comprehensive batch processor for optimizing cross-region operations:

### Features

- **Message Batch Synchronization**: Efficient cross-region message sync
- **Offline Message Batch Writes**: Optimized database batch inserts
- **Reconciliation Batch Processing**: Efficient data consistency checks
- **Automatic Batching**: Size-based and time-based flush triggers
- **Priority Ordering**: High-priority items processed first
- **Retry Logic**: Automatic retry with configurable limits
- **Concurrent Processing**: Multiple batch types processed in parallel

### Quick Start

```go
// Create batch processor
config := connpool.DefaultBatchProcessorConfig()
bp := connpool.NewBatchProcessor(config)

// Start the processor
bp.Start()
defer bp.Stop()

// Add messages to batch
msg := connpool.BatchMessage{
    ID:             "msg-001",
    RegionID:       "region-a",
    GlobalID:       "global-001",
    ConversationID: "conv-123",
    SenderID:       "user-456",
    Content:        "Hello, World!",
    Timestamp:      time.Now().UnixMilli(),
    Priority:       5,
    CreatedAt:      time.Now(),
}
bp.AddMessage(msg)

// Messages are automatically flushed based on:
// 1. Batch size (default: 100 messages)
// 2. Flush interval (default: 100ms)
```

### Integration with Pool Manager

```go
// Create pool manager
pm := connpool.NewPoolManager(connpool.DefaultPoolConfig())
defer pm.Stop()

// Create batch processor
batchConfig := connpool.DefaultBatchProcessorConfig()
bp, err := pm.GetOrCreateBatchProcessor("main", batchConfig)
if err != nil {
    panic(err)
}

// Start pool manager (starts all processors)
pm.Start()

// Use batch processor
msg := connpool.BatchMessage{
    ID:       "msg-001",
    Priority: 5,
}
bp.AddMessage(msg)

// Get metrics
metrics := bp.GetMetrics()
fmt.Printf("Total batched: %d\n", metrics.TotalMessagesBatched)
fmt.Printf("Avg batch size: %.2f\n", metrics.AvgBatchSize)
```

### Performance

**High Throughput Configuration:**
- Messages/sec: ~10,000
- Avg batch size: 95
- Avg flush duration: 15ms
- Memory usage: ~50MB

**Low Latency Configuration:**
- P50 latency: 25ms
- P95 latency: 75ms
- P99 latency: 150ms
- Throughput: ~5,000 messages/sec

For detailed documentation, see [Batch Optimization Guide](BATCH_OPTIMIZATION_GUIDE.md).

