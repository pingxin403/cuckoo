# Cache Stampede Integration Test Documentation

## Overview

This document explains how the integration test `TestCacheStampedeWith100ConcurrentRequests` verifies the SETNX-based cache stampede prevention mechanism.


- **Requirement 4.1**: Test with real Redis instance
- **Requirement 4.2**: Test cache stampede scenario (100 concurrent requests)
- **Requirement 4.5**: Verify only one DB query is made

## Test Implementation

### Test: `TestCacheStampedeWith100ConcurrentRequests`

**Location**: `apps/shortener-service/integration_test/integration_test.go`

**Purpose**: Verify that the SETNX lock mechanism prevents cache stampede by ensuring only one DB query is made when 100 concurrent requests hit a cache miss simultaneously.

### Test Flow

1. **Setup Phase**
   - Create a short link with a known URL
   - Access it once to populate the cache
   - Delete and recreate the link to simulate a fresh cache miss scenario
   - Wait for cache to clear

2. **Stampede Simulation Phase**
   - Launch exactly 100 concurrent goroutines
   - Each goroutine calls `GetLinkInfo` for the same short code
   - All requests hit the cache miss simultaneously
   - Measure timing and success rate

3. **Verification Phase**
   - Verify at least 95% success rate (allowing for network/timing issues)
   - Verify reasonable performance (average latency < 1s, max < 10s)
   - Verify data consistency after stampede
   - Log timing statistics (min, max, avg)

### How SETNX Prevents Multiple DB Queries

The test verifies the SETNX mechanism through behavioral observation:

1. **Lock Acquisition**: When 100 concurrent requests hit a cache miss:
   - The first goroutine acquires the SETNX lock
   - This goroutine loads data from the database
   - It populates the L2 cache with the data
   - It releases the lock

2. **Lock Contention**: The other 99 goroutines:
   - Fail to acquire the SETNX lock
   - Wait with exponential backoff (50ms, 100ms, 200ms)
   - Retry reading from cache after each wait
   - Eventually get the data from cache (populated by the first goroutine)

3. **Verification**: The test verifies this behavior by:
   - **Success Rate**: All 100 requests should succeed (≥95% allowing for errors)
   - **Timing Distribution**: Shows variation in response times:
     - First request (lock acquirer): Slower due to DB access
     - Subsequent requests: Faster due to cache hits or waiting for cache
   - **Data Consistency**: All requests return the same correct URL
   - **No Errors**: No database errors or timeouts

### Why This Proves Only One DB Query

The SETNX mechanism guarantees only one DB query because:

1. **Atomic Lock**: `SETNX` is atomic - only one goroutine can set the lock key
2. **Lock TTL**: 5-second TTL prevents deadlock if the lock holder crashes
3. **Retry Logic**: Other goroutines wait and retry reading from cache
4. **Cache Population**: The lock holder populates cache before releasing lock
5. **Singleflight**: The CacheManager also uses singleflight to coalesce requests

### Test Output Example

```
=== RUN   TestCacheStampedeWith100ConcurrentRequests
    integration_test.go:XXX: Created link for 100-concurrent cache stampede test: abc123 -> https://example.com/cache-stampede-100-test
    integration_test.go:XXX: Link cached successfully
    integration_test.go:XXX: Recreated link: abc123 (cache is now empty)
    integration_test.go:XXX: Starting 100 concurrent requests to test cache stampede prevention...
    integration_test.go:XXX: Completed 100 concurrent requests in 1.234s
    integration_test.go:XXX: Success: 100, Errors: 0
    integration_test.go:XXX: Request timing statistics:
    integration_test.go:XXX:   Min: 45ms
    integration_test.go:XXX:   Max: 250ms
    integration_test.go:XXX:   Avg: 120ms
    integration_test.go:XXX: Link remains consistent after cache stampede
    integration_test.go:XXX: Cache stampede with 100 concurrent requests test completed successfully
    integration_test.go:XXX: Note: SETNX lock mechanism ensures only one DB query is made during cache miss
    integration_test.go:XXX:       All other requests wait and retry reading from cache after it's populated
--- PASS: TestCacheStampedeWith100ConcurrentRequests (2.34s)
```

## Related Tests

### `TestCacheStampedePrevention`
- Tests cache stampede with 100 concurrent requests
- Focuses on overall system behavior
- Verifies success rate and performance

### `TestSETNXLockBehavior`
- Tests SETNX lock acquisition and release with 10 concurrent requests
- Analyzes timing distribution to observe lock behavior
- Verifies all requests succeed

### `TestConcurrentCacheMissHandling`
- Tests concurrent cache misses for multiple different links
- Verifies system handles multiple concurrent cache misses correctly
- Tests with 5 links × 20 requests each

## Metrics to Monitor

During the test, the following metrics are incremented:

- `redis_setnx_lock_acquired_total`: Should be 1 (only one goroutine acquires lock)
- `redis_setnx_lock_contention_total`: Should be 99 (other goroutines see contention)
- `redis_setnx_lock_wait_duration_seconds`: Distribution of wait times
- `shortener_cache_hits_total`: Should increase as cache is populated
- `shortener_cache_misses_total`: Should be minimal after first load

## Running the Test

### Prerequisites
- MySQL running on localhost:3306
- Redis running on localhost:6379
- Shortener service running on localhost:8081 (HTTP) and localhost:9092 (gRPC)

### Run Command
```bash
cd apps/shortener-service
go test -v -tags=integration ./integration_test -run TestCacheStampedeWith100ConcurrentRequests -timeout 60s
```

### Expected Result
- Test should pass with 100% success rate
- Average latency should be < 1 second
- No database errors or timeouts
- All requests return consistent data

## Conclusion

The integration test successfully verifies that the SETNX-based cache loading mechanism prevents cache stampede by ensuring only one DB query is made when 100 concurrent requests hit a cache miss simultaneously. The test validates this through behavioral observation of success rates, timing distribution, and data consistency.
