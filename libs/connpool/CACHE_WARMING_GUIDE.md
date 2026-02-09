# Cache Warming Guide

## Overview

The cache warming system provides intelligent preloading of hot data into Redis cache, optimized for cross-region multi-active architectures. It includes automatic warming cycles, cross-region synchronization, and flexible invalidation strategies.

## Features

- **Hot Data Preloading**: Automatically identify and preload frequently accessed data
- **Cross-Region Synchronization**: Sync cache entries across multiple regions
- **Multiple Invalidation Strategies**: TTL, LRU, Write-through, and Hybrid
- **Batch Operations**: Efficient batch preloading with configurable batch sizes
- **Metrics Collection**: Track warming performance and cache hit rate improvements
- **Automatic Warming Cycles**: Periodic cache warming with configurable intervals

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/cuckoo-org/cuckoo/libs/connpool"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create pool manager
    config := connpool.DefaultPoolConfig()
    pm := connpool.NewPoolManager(config)
    defer pm.Stop()
    
    // Get Redis connection
    redisClient, err := pm.GetRedis("primary", &redis.Options{
        Addr: "localhost:6379",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Create cache warmer
    warmerConfig := connpool.DefaultCacheWarmerConfig()
    warmer, err := pm.GetOrCreateCacheWarmer("primary", warmerConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start pool manager (starts automatic warming)
    if err := pm.Start(); err != nil {
        log.Fatal(err)
    }
    
    // Manually warm cache with hot data
    ctx := context.Background()
    hotData := []connpool.HotDataItem{
        {
            Key:         "user:1001",
            Value:       `{"name":"Alice","email":"alice@example.com"}`,
            AccessCount: 250,
            TTL:         1 * time.Hour,
            Priority:    10,
        },
        {
            Key:         "product:5001",
            Value:       `{"name":"Laptop","price":999.99}`,
            AccessCount: 180,
            TTL:         30 * time.Minute,
            Priority:    8,
        },
    }
    
    result := warmer.WarmCache(ctx, hotData)
    log.Printf("Warmed %d items in %v", result.SuccessCount, result.Duration)
    log.Printf("Hit rate improvement: %.2f%%", result.HitRateImprovement*100)
}
```

## Configuration

### Cache Warmer Configuration

```go
type CacheWarmerConfig struct {
    // Enabled indicates if cache warming is enabled
    Enabled bool
    
    // WarmInterval is the interval between cache warming operations
    WarmInterval time.Duration
    
    // WarmTimeout is the timeout for warming operations
    WarmTimeout time.Duration
    
    // HotDataThreshold is the access count threshold for hot data
    HotDataThreshold int64
    
    // HotDataTTL is the TTL for hot data in cache
    HotDataTTL time.Duration
    
    // PreloadBatchSize is the batch size for preloading
    PreloadBatchSize int
    
    // CrossRegionSync enables cross-region cache synchronization
    CrossRegionSync bool
    
    // InvalidationStrategy defines the cache invalidation strategy
    InvalidationStrategy InvalidationStrategy
    
    // MaxCacheSize is the maximum cache size in bytes (0 = unlimited)
    MaxCacheSize int64
    
    // EvictionPolicy defines the eviction policy when cache is full
    EvictionPolicy EvictionPolicy
}
```

### Default Configuration

```go
config := connpool.DefaultCacheWarmerConfig()
// Returns:
// - Enabled: true
// - WarmInterval: 5 minutes
// - WarmTimeout: 30 seconds
// - HotDataThreshold: 100 accesses
// - HotDataTTL: 1 hour
// - PreloadBatchSize: 100 items
// - CrossRegionSync: true
// - InvalidationStrategy: Hybrid
// - MaxCacheSize: 1GB
// - EvictionPolicy: LRU
```

### Custom Configuration

```go
config := connpool.CacheWarmerConfig{
    Enabled:              true,
    WarmInterval:         10 * time.Minute,  // Warm every 10 minutes
    WarmTimeout:          1 * time.Minute,   // 1 minute timeout
    HotDataThreshold:     500,               // Items accessed 500+ times
    HotDataTTL:           2 * time.Hour,     // 2 hour TTL
    PreloadBatchSize:     50,                // Smaller batches
    CrossRegionSync:      true,
    InvalidationStrategy: connpool.InvalidationWrite,
    MaxCacheSize:         512 * 1024 * 1024, // 512MB
    EvictionPolicy:       connpool.EvictionLFU,
}
```

## Invalidation Strategies

### 1. TTL-Based Invalidation

Relies on Redis TTL for automatic expiration. No explicit invalidation needed.

```go
config := connpool.DefaultCacheWarmerConfig()
config.InvalidationStrategy = connpool.InvalidationTTL
```

**Use Case**: Simple caching with predictable expiration times.

### 2. Write-Through Invalidation

Immediately deletes cache entries on invalidation.

```go
config := connpool.DefaultCacheWarmerConfig()
config.InvalidationStrategy = connpool.InvalidationWrite
```

**Use Case**: Strong consistency requirements, immediate cache updates.

### 3. LRU Invalidation

Marks entries for invalidation with short TTL, allowing LRU eviction.

```go
config := connpool.DefaultCacheWarmerConfig()
config.InvalidationStrategy = connpool.InvalidationLRU
```

**Use Case**: Memory-constrained environments, gradual cache updates.

### 4. Hybrid Invalidation

Combines TTL and LRU strategies for balanced performance.

```go
config := connpool.DefaultCacheWarmerConfig()
config.InvalidationStrategy = connpool.InvalidationHybrid
```

**Use Case**: Most scenarios, balances consistency and performance.

## Cross-Region Synchronization

### Setup

```go
// Create pool manager
pm := connpool.NewPoolManager(connpool.DefaultPoolConfig())

// Get local Redis
localClient, _ := pm.GetRedis("local", &redis.Options{
    Addr: "local-redis:6379",
})

// Get remote Redis
remoteClient, _ := pm.GetRedis("remote", &redis.Options{
    Addr: "remote-redis:6379",
})

// Create cache warmer with cross-region sync enabled
config := connpool.DefaultCacheWarmerConfig()
config.CrossRegionSync = true
warmer, _ := pm.GetOrCreateCacheWarmer("local", config)
```

### Synchronize Cache Entries

```go
ctx := context.Background()

// Keys to synchronize
keys := []string{
    "user:1001",
    "user:1002",
    "product:5001",
}

// Sync from local to remote
err := warmer.SyncCrossRegion(ctx, remoteClient, keys)
if err != nil {
    log.Printf("Sync failed: %v", err)
}
```

### Automatic Synchronization

For automatic cross-region sync, implement a custom hot data identifier:

```go
// Custom hot data identifier
type CustomHotDataIdentifier struct {
    localRedis  *redis.Client
    remoteRedis *redis.Client
}

func (h *CustomHotDataIdentifier) IdentifyAndSync(ctx context.Context, warmer *connpool.CacheWarmer) error {
    // 1. Identify hot data from access logs
    hotData := h.identifyHotData(ctx)
    
    // 2. Warm local cache
    result := warmer.WarmCache(ctx, hotData)
    
    // 3. Sync to remote region
    keys := make([]string, len(hotData))
    for i, item := range hotData {
        keys[i] = item.Key
    }
    
    return warmer.SyncCrossRegion(ctx, h.remoteRedis, keys)
}
```

## Hot Data Identification

### Manual Identification

```go
// Identify hot data based on your application logic
hotData := []connpool.HotDataItem{
    {
        Key:         "user:1001",
        Value:       userData,
        AccessCount: 250,
        LastAccess:  time.Now(),
        TTL:         1 * time.Hour,
        Priority:    10, // Higher priority = more important
    },
}

result := warmer.WarmCache(ctx, hotData)
```

### Automatic Identification (Custom Implementation)

```go
// Implement custom hot data identification
func identifyHotDataFromLogs(ctx context.Context, db *sql.DB) ([]connpool.HotDataItem, error) {
    // Query access logs
    query := `
        SELECT key, COUNT(*) as access_count
        FROM access_logs
        WHERE timestamp > NOW() - INTERVAL 1 HOUR
        GROUP BY key
        HAVING access_count > 100
        ORDER BY access_count DESC
        LIMIT 1000
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var hotData []connpool.HotDataItem
    for rows.Next() {
        var key string
        var accessCount int64
        
        if err := rows.Scan(&key, &accessCount); err != nil {
            continue
        }
        
        // Fetch value from database
        value, err := fetchValueFromDB(ctx, db, key)
        if err != nil {
            continue
        }
        
        hotData = append(hotData, connpool.HotDataItem{
            Key:         key,
            Value:       value,
            AccessCount: accessCount,
            LastAccess:  time.Now(),
            TTL:         1 * time.Hour,
            Priority:    calculatePriority(accessCount),
        })
    }
    
    return hotData, nil
}
```

## Metrics and Monitoring

### Get Cache Warmer Metrics

```go
metrics := warmer.GetMetrics()

log.Printf("Total Warmed: %d", metrics.TotalWarmed)
log.Printf("Total Failed: %d", metrics.TotalFailed)
log.Printf("Total Invalidated: %d", metrics.TotalInvalidated)
log.Printf("Last Warm Time: %v", metrics.LastWarmTime)
log.Printf("Warm Duration: %v", metrics.WarmDuration)
log.Printf("Hit Rate Before: %.2f%%", metrics.HitRateBefore*100)
log.Printf("Hit Rate After: %.2f%%", metrics.HitRateAfter*100)
log.Printf("Hit Rate Improvement: %.2f%%", metrics.HitRateImprovement*100)
```

### Get All Cache Warmer Metrics

```go
allMetrics := pm.GetAllCacheWarmerMetrics()

for name, metrics := range allMetrics {
    log.Printf("Cache Warmer: %s", name)
    log.Printf("  Total Warmed: %d", metrics.TotalWarmed)
    log.Printf("  Hit Rate Improvement: %.2f%%", metrics.HitRateImprovement*100)
}
```

### Prometheus Metrics Integration

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    cacheWarmingDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "cache_warming_duration_seconds",
            Help: "Duration of cache warming operations",
        },
        []string{"cache_name"},
    )
    
    cacheHitRateImprovement = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cache_hit_rate_improvement",
            Help: "Cache hit rate improvement after warming",
        },
        []string{"cache_name"},
    )
)

// Record metrics after warming
result := warmer.WarmCache(ctx, hotData)
cacheWarmingDuration.WithLabelValues("primary").Observe(result.Duration.Seconds())
cacheHitRateImprovement.WithLabelValues("primary").Set(result.HitRateImprovement)
```

## Best Practices

### 1. Batch Size Tuning

```go
// For high-latency networks (cross-region)
config.PreloadBatchSize = 50  // Smaller batches

// For low-latency networks (same region)
config.PreloadBatchSize = 200 // Larger batches
```

### 2. Warm Interval Tuning

```go
// For frequently changing data
config.WarmInterval = 1 * time.Minute

// For stable data
config.WarmInterval = 15 * time.Minute
```

### 3. TTL Configuration

```go
// Short-lived data (session data)
item.TTL = 15 * time.Minute

// Long-lived data (user profiles)
item.TTL = 24 * time.Hour

// Static data (configuration)
item.TTL = 7 * 24 * time.Hour
```

### 4. Priority-Based Warming

```go
// Critical data (authentication)
item.Priority = 10

// Important data (user profiles)
item.Priority = 7

// Normal data (product listings)
item.Priority = 5

// Low priority data (analytics)
item.Priority = 2
```

### 5. Error Handling

```go
result := warmer.WarmCache(ctx, hotData)

if len(result.Errors) > 0 {
    log.Printf("Warming completed with %d errors", len(result.Errors))
    for _, err := range result.Errors {
        log.Printf("  Error: %v", err)
    }
}

// Check success rate
successRate := float64(result.SuccessCount) / float64(result.TotalItems)
if successRate < 0.95 {
    log.Printf("Warning: Low success rate: %.2f%%", successRate*100)
}
```

## Performance Optimization

### 1. Connection Pooling

Ensure Redis connection pool is properly configured:

```go
config := connpool.DefaultPoolConfig()
config.Redis.PoolSize = 100        // Increase for high throughput
config.Redis.MinIdleConns = 20     // Keep idle connections
config.Redis.PoolTimeout = 5 * time.Second
```

### 2. Pipeline Operations

The cache warmer automatically uses Redis pipelining for batch operations, reducing round-trip time.

### 3. Compression

For large values, consider compression:

```go
import "compress/gzip"

func compressValue(value string) ([]byte, error) {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    if _, err := gz.Write([]byte(value)); err != nil {
        return nil, err
    }
    if err := gz.Close(); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
```

### 4. Monitoring and Alerting

Set up alerts for:
- Low cache hit rate improvement (< 5%)
- High warming failure rate (> 5%)
- Long warming duration (> 1 minute)
- High invalidation rate (> 10% of warmed items)

## Troubleshooting

### Issue: Low Hit Rate Improvement

**Possible Causes:**
- Hot data identification is inaccurate
- Warm interval is too long
- TTL is too short

**Solutions:**
```go
// Increase warm frequency
config.WarmInterval = 2 * time.Minute

// Increase TTL
config.HotDataTTL = 2 * time.Hour

// Lower hot data threshold
config.HotDataThreshold = 50
```

### Issue: High Memory Usage

**Possible Causes:**
- Too many items being warmed
- Large value sizes
- TTL is too long

**Solutions:**
```go
// Set max cache size
config.MaxCacheSize = 512 * 1024 * 1024 // 512MB

// Use LFU eviction
config.EvictionPolicy = connpool.EvictionLFU

// Reduce TTL
config.HotDataTTL = 30 * time.Minute
```

### Issue: Slow Warming Operations

**Possible Causes:**
- Large batch size
- Network latency
- Redis overload

**Solutions:**
```go
// Reduce batch size
config.PreloadBatchSize = 25

// Increase timeout
config.WarmTimeout = 2 * time.Minute

// Check Redis performance
stats := redisClient.PoolStats()
log.Printf("Redis pool stats: %+v", stats)
```

## Examples

### Example 1: E-commerce Product Cache

```go
func warmProductCache(ctx context.Context, warmer *connpool.CacheWarmer, db *sql.DB) error {
    // Get top selling products
    query := `
        SELECT product_id, product_data
        FROM products
        WHERE sales_count > 100
        ORDER BY sales_count DESC
        LIMIT 500
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    var hotData []connpool.HotDataItem
    for rows.Next() {
        var productID string
        var productData string
        
        if err := rows.Scan(&productID, &productData); err != nil {
            continue
        }
        
        hotData = append(hotData, connpool.HotDataItem{
            Key:         fmt.Sprintf("product:%s", productID),
            Value:       productData,
            AccessCount: 100,
            TTL:         1 * time.Hour,
            Priority:    8,
        })
    }
    
    result := warmer.WarmCache(ctx, hotData)
    log.Printf("Warmed %d products in %v", result.SuccessCount, result.Duration)
    
    return nil
}
```

### Example 2: User Session Cache

```go
func warmUserSessions(ctx context.Context, warmer *connpool.CacheWarmer, activeUsers []string) error {
    var hotData []connpool.HotDataItem
    
    for _, userID := range activeUsers {
        // Fetch user session from database
        session, err := fetchUserSession(ctx, userID)
        if err != nil {
            continue
        }
        
        hotData = append(hotData, connpool.HotDataItem{
            Key:         fmt.Sprintf("session:%s", userID),
            Value:       session,
            AccessCount: 200,
            TTL:         15 * time.Minute,
            Priority:    10, // High priority for sessions
        })
    }
    
    result := warmer.WarmCache(ctx, hotData)
    return nil
}
```

### Example 3: Cross-Region Sync

```go
func syncCacheAcrossRegions(ctx context.Context, pm *connpool.PoolManager) error {
    // Get warmers for both regions
    localWarmer, _ := pm.GetCacheWarmer("region-a")
    remoteClient, _ := pm.GetRedis("region-b", &redis.Options{
        Addr: "region-b-redis:6379",
    })
    
    // Identify hot data in local region
    hotData := identifyLocalHotData(ctx)
    
    // Warm local cache
    result := localWarmer.WarmCache(ctx, hotData)
    
    // Sync to remote region
    keys := make([]string, len(hotData))
    for i, item := range hotData {
        keys[i] = item.Key
    }
    
    return localWarmer.SyncCrossRegion(ctx, remoteClient, keys)
}
```

## Related Documentation

- [Connection Pool README](./README.md)
- [Multi-Region Architecture Design](../../.kiro/specs/multi-region-active-active/design.md)
- [Performance Optimization Guide](../../docs/operations/performance-optimization.md)

## License

MIT License - see LICENSE file for details
