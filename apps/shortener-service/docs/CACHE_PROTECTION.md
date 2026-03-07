# Cache Protection Mechanisms

This document describes the cache protection mechanisms implemented in the shortener-service to handle common caching challenges.

## Overview

The service implements multiple layers of protection against cache-related issues:

1. **Cache Penetration Protection** - Prevents repeated queries for non-existent data
2. **Cache Breakdown Protection** - Prevents database overload when hot keys expire
3. **Cache Avalanche Protection** - Prevents cascading failures when many keys expire simultaneously
4. **Delayed Double Delete** - Ensures cache consistency during updates

## 1. Cache Penetration (缓存穿透)

### Problem
When clients repeatedly query for non-existent keys, every request bypasses the cache and hits the database, potentially causing database overload.

### Current Protection
- **Singleflight**: Concurrent requests for the same non-existent key are coalesced into a single database query
- **Reduction**: ~89% reduction in database queries for concurrent requests

### Test Coverage
- `TestCachePenetration`: Sequential requests for non-existent keys
- `TestCachePenetrationConcurrent`: Concurrent requests for non-existent keys

### Future Enhancements
- **Bloom Filter**: Check if a key might exist before querying cache/database
- **Null Value Caching**: Cache a special marker for non-existent keys with short TTL

## 2. Cache Breakdown (缓存击穿)

### Problem
When a hot key expires, many concurrent requests hit the database simultaneously, causing a spike in database load.

### Current Protection
- **Singleflight**: Only one request loads data from database when cache misses
- **Reduction**: ~98% reduction in database queries during hotspot expiration

### Implementation
```go
// CacheManager uses singleflight to coalesce concurrent requests
func (cm *CacheManager) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
    // Singleflight ensures only one goroutine loads from DB
    result, err, _ := cm.sf.Do(shortCode, func() (interface{}, error) {
        return cm.loadFromStorage(ctx, shortCode)
    })
    // ...
}
```

### Test Coverage
- `TestCacheBreakdown`: Simulates hot key expiration with 100 concurrent requests

## 3. Cache Avalanche (缓存雪崩)

### Problem
When many keys expire at the same time, a large number of requests hit the database simultaneously, potentially overwhelming it.

### Current Protection
- **Singleflight**: Coalesces concurrent requests for each key
- **Reduction**: ~59% reduction in database queries during mass expiration

### Implementation
- Each key is protected independently by singleflight
- Multiple keys expiring simultaneously are handled in parallel
- Database load is distributed across multiple queries instead of one massive spike

### Test Coverage
- `TestCacheAvalanche`: Simulates 50 keys expiring simultaneously with 10 requests per key

### Future Enhancements
- **Random TTL**: Add random jitter to cache TTL to prevent synchronized expiration
- **Gradual Expiration**: Implement staggered expiration for related keys

## 4. Delayed Double Delete (延时双删)

### Problem
When updating data, there's a race condition where:
1. Update database
2. Delete cache
3. Another request reads old data from database (before replication)
4. Cache the stale data

### Current Protection
- **Double Delete Pattern**:
  1. Delete cache immediately after database update
  2. Wait a short delay (50-100ms)
  3. Delete cache again to remove any stale data cached during the update

### Implementation
```go
// Update flow
storage.Update(key, newValue)  // 1. Update database
cache.Delete(key)               // 2. First delete
time.Sleep(100 * time.Millisecond)  // 3. Wait for replication
cache.Delete(key)               // 4. Second delete
```

### Test Coverage
- `TestDelayedDoubleDelete`: Single update with delayed double delete
- `TestDelayedDoubleDeleteConcurrent`: Multiple concurrent updates

### Configuration
- **Delay Duration**: 50-100ms (configurable based on database replication lag)
- **Trade-off**: Longer delay = better consistency, but higher latency

## 5. Cache Consistency Under Load

### Test Coverage
- `TestCacheConsistencyUnderLoad`: Mixed workload with reads, updates, and deletes
- Verifies cache remains consistent under high concurrent load

### Metrics
- Read operations: ~50 concurrent readers
- Update operations: ~5 concurrent updaters
- Delete operations: ~5 concurrent deleters
- Duration: 2 seconds of sustained load

## Performance Metrics

### Singleflight Effectiveness

| Scenario | Concurrent Requests | DB Queries | Reduction |
|----------|-------------------|------------|-----------|
| Cache Breakdown | 100 | 1-2 | 98% |
| Cache Penetration | 100 | 11 | 89% |
| Cache Avalanche (50 keys) | 500 | 206 | 59% |

### Key Insights
1. **Existing Keys**: Singleflight is highly effective (98% reduction)
2. **Non-existent Keys**: Moderate effectiveness (89% reduction) - errors are not cached
3. **Multiple Keys**: Per-key protection (59% reduction) - each key protected independently

## Running Tests

### All Cache Protection Tests
```bash
./apps/shortener-service/scripts/testing/test-cache-protection.sh
```

### Specific Test Categories
```bash
# Cache penetration tests
go test -v -run TestCache.*Penetration ./cache/

# Cache breakdown tests
go test -v -run TestCache.*Breakdown ./cache/

# Cache avalanche tests
go test -v -run TestCache.*Avalanche ./cache/

# Delayed double delete tests
go test -v -run TestDelayed ./cache/
```

## Best Practices

### 1. Cache Key Design
- Use consistent key naming conventions
- Include version information in keys for easy invalidation
- Consider key expiration patterns to avoid synchronized expiration

### 2. TTL Strategy
- Set appropriate TTL based on data update frequency
- Add random jitter to prevent cache avalanche
- Use shorter TTL for frequently updated data

### 3. Update Strategy
- Always use delayed double delete for data updates
- Consider using cache-aside pattern for reads
- Monitor cache hit rates and adjust TTL accordingly

### 4. Monitoring
- Track cache hit/miss rates
- Monitor database query patterns
- Alert on unusual cache penetration patterns
- Measure singleflight effectiveness

## Future Improvements

### 1. Bloom Filter Integration
- Implement bloom filter to check key existence before cache/DB query
- Reduces cache penetration for non-existent keys
- Trade-off: Small false positive rate vs. significant performance gain

### 2. Null Value Caching
- Cache special marker for non-existent keys
- Set short TTL (e.g., 1 minute) for null values
- Prevents repeated DB queries for non-existent keys

### 3. Adaptive TTL
- Dynamically adjust TTL based on access patterns
- Hot keys get longer TTL
- Cold keys get shorter TTL or are evicted

### 4. Circuit Breaker Integration
- Implement circuit breaker for database queries
- Fail fast when database is overloaded
- Return cached data even if stale

## References

- [Singleflight Pattern](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [Cache Stampede Prevention](https://en.wikipedia.org/wiki/Cache_stampede)
- [Delayed Double Delete Pattern](https://redis.io/docs/manual/patterns/distributed-locks/)
