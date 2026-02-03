# Redis Metrics Reference

This document defines all Redis-related metrics exposed by the URL Shortener Service.

## Metric Categories

### 1. Connection Pool Metrics

These metrics track Redis connection pool health and utilization.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_pool_hits_total` | Gauge | - | Total number of times a connection was successfully obtained from the pool |
| `redis_pool_misses_total` | Gauge | - | Total number of times a connection was not available in the pool |
| `redis_pool_timeouts_total` | Gauge | - | Total number of times waiting for a connection timed out |
| `redis_pool_connections` | Gauge | `state=total\|idle\|active` | Number of connections in different states |

**Alert Thresholds:**
- Pool utilization > 80%: Warning
- Pool timeouts > 10/min: Critical

---

### 2. Pipeline Metrics

These metrics track batch operation performance using Redis Pipeline.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_pipeline_duration_seconds` | Histogram | `operation=batch_set\|batch_get` | Duration of pipeline operations |
| `redis_pipeline_errors_total` | Counter | `operation=batch_set\|batch_get` | Total number of pipeline errors |
| `redis_pipeline_batch_size` | Histogram | - | Distribution of batch sizes |

**Alert Thresholds:**
- Pipeline P99 latency > 50ms: Warning
- Pipeline error rate > 1%: Critical

---

### 3. SETNX Lock Metrics

These metrics track cache loading with SETNX-based locking to prevent cache stampede.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_setnx_lock_acquired_total` | Counter | - | Total number of successful lock acquisitions |
| `redis_setnx_lock_contention_total` | Counter | - | Total number of lock contentions (failed acquisitions) |
| `redis_setnx_lock_wait_duration_seconds` | Histogram | - | Time spent waiting for locks |

**Alert Thresholds:**
- Lock contention rate > 20%: Warning
- Lock wait time P99 > 100ms: Warning

---

### 4. TTL Distribution Metrics

These metrics track cache entry TTL distribution to verify jitter implementation.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_ttl_seconds` | Histogram | `layer=L2` | Distribution of TTL values for cached entries |

**Expected Distribution:**
- L2 Cache: 6-8 days (7 days ± 1 day jitter)
- Should show good distribution, not clustered

---

### 5. Cache Consistency Metrics

These metrics track cache-DB consistency operations.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_cache_delete_total` | Counter | `type=immediate\|delayed` | Total number of cache deletions |
| `redis_cache_delete_errors_total` | Counter | `type=immediate\|delayed` | Total number of cache deletion errors |
| `redis_empty_placeholder_set_total` | Counter | - | Total number of empty placeholders set |

**Alert Thresholds:**
- Delete error rate > 5%: Warning

---

### 6. Singleflight Metrics

These metrics track request coalescing efficiency.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `shortener_singleflight_waits_total` | Counter | - | Total number of requests that waited for another |
| `redis_singleflight_wait_duration_seconds` | Histogram | - | Time spent waiting in singleflight |
| `redis_singleflight_errors_total` | Counter | - | Total number of singleflight errors |
| `redis_singleflight_timeouts_total` | Counter | - | Total number of singleflight timeouts |

**Alert Thresholds:**
- Singleflight timeout rate > 1%: Warning
- Wait time P99 > 5s: Critical

---

### 7. Cache Operation Metrics

These metrics track overall cache performance.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `shortener_cache_hits_total` | Counter | `layer=L1\|L2` | Total number of cache hits |
| `shortener_cache_misses_total` | Counter | `layer=L1\|L2` | Total number of cache misses |
| `shortener_cache_operations_total` | Counter | `operation=hit\|miss, layer=l1\|l2` | Total cache operations |
| `shortener_cache_warm_total` | Counter | `count` | Total number of cache warming operations |

**Alert Thresholds:**
- L2 cache hit rate < 90%: Warning
- L2 cache hit rate < 80%: Critical

---

### 8. Circuit Breaker Metrics (Future)

These metrics will track circuit breaker state for Redis operations.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_circuit_breaker_state` | Gauge | - | Current state (0=closed, 0.5=half-open, 1=open) |
| `redis_circuit_breaker_opened_total` | Counter | - | Total number of times circuit opened |
| `redis_circuit_breaker_closed_total` | Counter | - | Total number of times circuit closed |
| `redis_circuit_breaker_rejected_total` | Counter | - | Total number of rejected requests |

---

### 9. Lua Script Metrics (Future)

These metrics will track Lua script execution performance.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_lua_script_duration_seconds` | Histogram | `script=cache_load\|increment_expire` | Lua script execution time |
| `redis_lua_script_errors_total` | Counter | `script=cache_load\|increment_expire` | Lua script errors |

---

### 10. Cluster Metrics (Future)

These metrics will track Redis Cluster operations.

| Metric Name | Type | Labels | Description |
|------------|------|--------|-------------|
| `redis_cluster_redirects_total` | Counter | `type=MOVED\|ASK` | Total number of cluster redirects |
| `redis_cluster_errors_total` | Counter | `operation` | Total number of cluster errors |

---

## Metric Collection

All metrics are collected using the Prometheus format and exposed at `/metrics` endpoint.

### Collection Frequency

- **Connection Pool**: Every 10 seconds (background goroutine)
- **Pipeline Operations**: Per operation
- **SETNX Locks**: Per lock attempt
- **TTL**: Per cache set operation
- **Cache Operations**: Per operation

### Metric Types

- **Counter**: Monotonically increasing value (e.g., total requests)
- **Gauge**: Value that can go up or down (e.g., active connections)
- **Histogram**: Distribution of values (e.g., latency, batch size)

---

## Example Queries

### Connection Pool Health

```promql
# Pool utilization percentage
(redis_pool_connections{state="active"} / redis_pool_connections{state="total"}) * 100

# Pool timeout rate
rate(redis_pool_timeouts_total[5m])
```

### Pipeline Performance

```promql
# Pipeline P99 latency
histogram_quantile(0.99, rate(redis_pipeline_duration_seconds_bucket[5m]))

# Pipeline error rate
rate(redis_pipeline_errors_total[5m]) / rate(redis_pipeline_duration_seconds_count[5m])
```

### Cache Hit Rate

```promql
# L2 cache hit rate
rate(shortener_cache_hits_total{layer="L2"}[5m]) / 
(rate(shortener_cache_hits_total{layer="L2"}[5m]) + rate(shortener_cache_misses_total{layer="L2"}[5m]))
```

### SETNX Lock Contention

```promql
# Lock contention rate
rate(redis_setnx_lock_contention_total[5m]) / 
(rate(redis_setnx_lock_acquired_total[5m]) + rate(redis_setnx_lock_contention_total[5m]))

# Lock wait time P99
histogram_quantile(0.99, rate(redis_setnx_lock_wait_duration_seconds_bucket[5m]))
```

### Singleflight Efficiency

```promql
# Coalescing ratio (higher is better)
rate(shortener_singleflight_waits_total[5m]) / 
rate(shortener_cache_misses_total{layer="L2"}[5m])

# Singleflight timeout rate
rate(redis_singleflight_timeouts_total[5m])
```

---

## Grafana Dashboard Panels

### Recommended Dashboard Layout

1. **Overview Row**
   - Cache hit rate (L1 and L2)
   - Request rate
   - Error rate

2. **Connection Pool Row**
   - Pool utilization
   - Active/Idle connections
   - Pool timeouts

3. **Pipeline Row**
   - Pipeline latency (P50, P95, P99)
   - Batch size distribution
   - Pipeline error rate

4. **Cache Stampede Prevention Row**
   - SETNX lock acquisitions
   - Lock contention rate
   - Singleflight waits

5. **TTL Distribution Row**
   - TTL histogram
   - TTL min/max/avg

---

## Alert Rules

### Critical Alerts

```yaml
groups:
  - name: redis_critical
    rules:
      - alert: RedisPoolExhausted
        expr: redis_pool_timeouts_total > 10
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Redis connection pool exhausted"
          
      - alert: RedisCacheHitRateLow
        expr: |
          rate(shortener_cache_hits_total{layer="L2"}[5m]) / 
          (rate(shortener_cache_hits_total{layer="L2"}[5m]) + 
           rate(shortener_cache_misses_total{layer="L2"}[5m])) < 0.8
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "L2 cache hit rate below 80%"
```

### Warning Alerts

```yaml
  - name: redis_warning
    rules:
      - alert: RedisPoolUtilizationHigh
        expr: |
          (redis_pool_connections{state="active"} / 
           redis_pool_connections{state="total"}) > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis pool utilization above 80%"
          
      - alert: RedisPipelineLatencyHigh
        expr: |
          histogram_quantile(0.99, 
            rate(redis_pipeline_duration_seconds_bucket[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Pipeline P99 latency above 50ms"
```

---

## Monitoring Best Practices

1. **Set up alerts** for critical metrics (pool exhaustion, low hit rate)
2. **Monitor trends** over time to detect gradual degradation
3. **Correlate metrics** (e.g., high latency + high pool utilization)
4. **Use dashboards** for quick visual health checks
5. **Review metrics** during deployments and load tests
