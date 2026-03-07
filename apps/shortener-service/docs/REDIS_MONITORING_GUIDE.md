# Redis Monitoring Guide

---

## Overview

This guide provides comprehensive monitoring strategies for Redis optimizations in the shortener service. It covers key metrics to watch, alert thresholds, and Grafana dashboard usage.

---

## Key Metrics to Monitor

### 1. Performance Metrics

#### Request Latency

**Metric:** `http_request_duration_seconds`

**What to Monitor:**
- P50 (median): Should be <2ms
- P95: Should be <5ms
- P99: Should be <10ms

**Prometheus Query:**
```promql
# P50 latency
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[1m]))

# P95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[1m]))

# P99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))
```

**Alert Thresholds:**
- Warning: P99 > 8ms
- Critical: P99 > 15ms

#### Request Throughput

**Metric:** `http_requests_total`

**What to Monitor:**
- Current QPS
- Trend over time
- Sudden spikes or drops

**Prometheus Query:**
```promql
# Current QPS
rate(http_requests_total[1m])

# QPS by status code
sum by (status) (rate(http_requests_total[1m]))
```

**Alert Thresholds:**
- Warning: QPS drops >50% from baseline
- Critical: QPS drops >80% from baseline

#### Error Rate

**Metric:** `http_requests_failed_total`

**What to Monitor:**
- Overall error rate
- Error rate by type
- Trend over time

**Prometheus Query:**
```promql
# Error rate percentage
rate(http_requests_failed_total[1m]) / rate(http_requests_total[1m]) * 100

# Errors by type
sum by (error_type) (rate(http_requests_failed_total[1m]))
```

**Alert Thresholds:**
- Warning: Error rate > 0.5%
- Critical: Error rate > 1%

---

### 2. Cache Metrics

#### Cache Hit Rate

**Metric:** `redis_cache_hits_total`, `redis_cache_misses_total`

**What to Monitor:**
- Overall hit rate
- Hit rate by cache layer (L1/L2)
- Trend over time

**Prometheus Query:**
```promql
# Overall cache hit rate
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m])) * 100

# Hit rate by layer
sum by (layer) (rate(redis_cache_hits_total[1m])) / 
(sum by (layer) (rate(redis_cache_hits_total[1m])) + 
 sum by (layer) (rate(redis_cache_misses_total[1m]))) * 100
```

**Alert Thresholds:**
- Warning: Hit rate < 90%
- Critical: Hit rate < 80%

#### Cache Operations

**Metrics:** `redis_cache_get_total`, `redis_cache_set_total`, `redis_cache_delete_total`

**What to Monitor:**
- GET/SET/DELETE operation rates
- Operation latency
- Failed operations

**Prometheus Query:**
```promql
# Operation rates
rate(redis_cache_get_total[1m])
rate(redis_cache_set_total[1m])
rate(redis_cache_delete_total[1m])

# Failed operations
rate(redis_cache_errors_total[1m])
```

**Alert Thresholds:**
- Warning: Failed operations > 0.1%
- Critical: Failed operations > 1%

---

### 3. Connection Pool Metrics

#### Pool Utilization

**Metrics:** `redis_pool_active_connections`, `redis_pool_idle_connections`, `redis_pool_size`

**What to Monitor:**
- Active connection count
- Idle connection count
- Pool utilization percentage

**Prometheus Query:**
```promql
# Pool utilization percentage
redis_pool_active_connections / redis_pool_size * 100

# Active connections
redis_pool_active_connections

# Idle connections
redis_pool_idle_connections
```

**Alert Thresholds:**
- Warning: Utilization > 80%
- Critical: Utilization > 95%

#### Pool Efficiency

**Metrics:** `redis_pool_hits_total`, `redis_pool_misses_total`

**What to Monitor:**
- Pool hit rate
- Connection wait time
- Timeout count

**Prometheus Query:**
```promql
# Pool hit rate
rate(redis_pool_hits_total[1m]) / 
(rate(redis_pool_hits_total[1m]) + rate(redis_pool_misses_total[1m])) * 100

# Pool misses (new connections created)
rate(redis_pool_misses_total[1m])

# Timeouts
rate(redis_pool_timeouts_total[1m])
```

**Alert Thresholds:**
- Warning: Hit rate < 95%
- Critical: Hit rate < 90%
- Critical: Timeouts > 0

---

### 4. Circuit Breaker Metrics

#### Circuit Breaker State

**Metric:** `redis_circuit_breaker_state`

**What to Monitor:**
- Current state (0=closed, 1=open, 2=half-open)
- State transitions
- Time in open state

**Prometheus Query:**
```promql
# Current state
redis_circuit_breaker_state

# State changes
changes(redis_circuit_breaker_state[5m])

# Time in open state
time() - redis_circuit_breaker_last_state_change{state="open"}
```

**Alert Thresholds:**
- Warning: State = open (1)
- Critical: State = open for >5 minutes

#### Circuit Breaker Operations

**Metrics:** `redis_circuit_breaker_requests_total`, `redis_circuit_breaker_failures_total`, `redis_circuit_breaker_rejected_total`

**What to Monitor:**
- Total requests
- Failure count
- Rejected requests (when open)

**Prometheus Query:**
```promql
# Failure rate
rate(redis_circuit_breaker_failures_total[1m]) / 
rate(redis_circuit_breaker_requests_total[1m]) * 100

# Rejected requests
rate(redis_circuit_breaker_rejected_total[1m])
```

**Alert Thresholds:**
- Warning: Failure rate > 5%
- Critical: Rejected requests > 0

---

### 5. Singleflight Metrics

#### Singleflight Efficiency

**Metrics:** `singleflight_execute_total`, `singleflight_wait_total`

**What to Monitor:**
- Coalescing rate
- Wait time distribution
- Timeout count

**Prometheus Query:**
```promql
# Coalescing rate (percentage of requests that waited)
rate(singleflight_wait_total[1m]) / 
(rate(singleflight_execute_total[1m]) + rate(singleflight_wait_total[1m])) * 100

# Executions (actual DB queries)
rate(singleflight_execute_total[1m])

# Waits (coalesced requests)
rate(singleflight_wait_total[1m])

# Timeouts
rate(singleflight_timeout_total[1m])
```

**Alert Thresholds:**
- Warning: Coalescing rate < 80% (during cache misses)
- Critical: Timeout count > 0

---

### 6. SETNX Lock Metrics

#### Lock Operations

**Metrics:** `redis_setnx_lock_acquired_total`, `redis_setnx_lock_wait_total`, `redis_setnx_lock_timeout_total`

**What to Monitor:**
- Lock acquisition rate
- Lock wait rate
- Lock contention
- Timeout count

**Prometheus Query:**
```promql
# Lock contention (percentage of requests that waited)
rate(redis_setnx_lock_wait_total[1m]) / 
(rate(redis_setnx_lock_acquired_total[1m]) + rate(redis_setnx_lock_wait_total[1m])) * 100

# Lock acquisitions
rate(redis_setnx_lock_acquired_total[1m])

# Lock waits
rate(redis_setnx_lock_wait_total[1m])

# Timeouts
rate(redis_setnx_lock_timeout_total[1m])
```

**Alert Thresholds:**
- Warning: Timeout count > 0
- Critical: Contention > 99% (indicates stampede)

---

### 7. Pipeline Metrics

#### Pipeline Efficiency

**Metrics:** `redis_pipeline_batch_size`, `redis_pipeline_duration_seconds`

**What to Monitor:**
- Average batch size
- Pipeline execution time
- Success/failure rate

**Prometheus Query:**
```promql
# Average batch size
avg(redis_pipeline_batch_size)

# P99 pipeline duration
histogram_quantile(0.99, rate(redis_pipeline_duration_seconds_bucket[1m]))

# Pipeline operations
rate(redis_pipeline_operations_total[1m])

# Pipeline errors
rate(redis_pipeline_errors_total[1m])
```

**Alert Thresholds:**
- Warning: P99 duration > 10ms
- Critical: Error rate > 1%

---

### 8. Redis Server Metrics

#### Memory Usage

**Metric:** `redis_memory_used_bytes`, `redis_memory_max_bytes`

**What to Monitor:**
- Current memory usage
- Memory utilization percentage
- Evicted keys count

**Prometheus Query:**
```promql
# Memory utilization
redis_memory_used_bytes / redis_memory_max_bytes * 100

# Evicted keys
rate(redis_evicted_keys_total[1m])
```

**Alert Thresholds:**
- Warning: Memory utilization > 80%
- Critical: Memory utilization > 95%
- Warning: Evicted keys > 0

#### Redis Operations

**Metrics:** `redis_commands_processed_total`, `redis_keyspace_hits_total`, `redis_keyspace_misses_total`

**What to Monitor:**
- Commands per second
- Keyspace hit rate
- Slow commands

**Prometheus Query:**
```promql
# Commands per second
rate(redis_commands_processed_total[1m])

# Keyspace hit rate
rate(redis_keyspace_hits_total[1m]) / 
(rate(redis_keyspace_hits_total[1m]) + rate(redis_keyspace_misses_total[1m])) * 100
```

**Alert Thresholds:**
- Warning: Keyspace hit rate < 90%
- Critical: Commands per second > 100K (capacity limit)

---

## Alert Configuration

### Prometheus Alert Rules

Create file: `prometheus-redis-alerts.yml`

```yaml
groups:
  - name: redis_optimization_alerts
    interval: 30s
    rules:
      # Performance Alerts
      - alert: HighP99Latency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m])) > 0.008
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "High P99 latency detected"
          description: "P99 latency is {{ $value }}s (threshold: 8ms)"

      - alert: CriticalP99Latency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m])) > 0.015
        for: 2m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Critical P99 latency detected"
          description: "P99 latency is {{ $value }}s (threshold: 15ms)"

      - alert: HighErrorRate
        expr: rate(http_requests_failed_total[1m]) / rate(http_requests_total[1m]) * 100 > 0.5
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }}% (threshold: 0.5%)"

      # Cache Alerts
      - alert: LowCacheHitRate
        expr: rate(redis_cache_hits_total[1m]) / (rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m])) * 100 < 90
        for: 10m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value }}% (threshold: 90%)"

      - alert: CriticalCacheHitRate
        expr: rate(redis_cache_hits_total[1m]) / (rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m])) * 100 < 80
        for: 5m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Critical cache hit rate"
          description: "Cache hit rate is {{ $value }}% (threshold: 80%)"

      # Connection Pool Alerts
      - alert: HighPoolUtilization
        expr: redis_pool_active_connections / redis_pool_size * 100 > 80
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "High connection pool utilization"
          description: "Pool utilization is {{ $value }}% (threshold: 80%)"

      - alert: CriticalPoolUtilization
        expr: redis_pool_active_connections / redis_pool_size * 100 > 95
        for: 2m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Critical connection pool utilization"
          description: "Pool utilization is {{ $value }}% (threshold: 95%)"

      - alert: PoolTimeouts
        expr: rate(redis_pool_timeouts_total[1m]) > 0
        for: 1m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Connection pool timeouts detected"
          description: "Pool timeout rate: {{ $value }}/s"

      # Circuit Breaker Alerts
      - alert: CircuitBreakerOpen
        expr: redis_circuit_breaker_state == 1
        for: 1m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "Circuit breaker is open"
          description: "Redis circuit breaker has opened, requests falling back to database"

      - alert: CircuitBreakerStuckOpen
        expr: redis_circuit_breaker_state == 1
        for: 5m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Circuit breaker stuck open"
          description: "Circuit breaker has been open for >5 minutes"

      # Singleflight Alerts
      - alert: SingleflightTimeouts
        expr: rate(singleflight_timeout_total[1m]) > 0
        for: 1m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "Singleflight timeouts detected"
          description: "Timeout rate: {{ $value }}/s"

      # SETNX Lock Alerts
      - alert: SETNXLockTimeouts
        expr: rate(redis_setnx_lock_timeout_total[1m]) > 0
        for: 1m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "SETNX lock timeouts detected"
          description: "Lock timeout rate: {{ $value }}/s"

      - alert: CacheStampede
        expr: rate(redis_setnx_lock_wait_total[1m]) / (rate(redis_setnx_lock_acquired_total[1m]) + rate(redis_setnx_lock_wait_total[1m])) * 100 > 99
        for: 2m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Possible cache stampede detected"
          description: "Lock contention is {{ $value }}% (threshold: 99%)"

      # Redis Server Alerts
      - alert: HighRedisMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes * 100 > 80
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "High Redis memory usage"
          description: "Memory usage is {{ $value }}% (threshold: 80%)"

      - alert: CriticalRedisMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes * 100 > 95
        for: 2m
        labels:
          severity: critical
          component: redis
        annotations:
          summary: "Critical Redis memory usage"
          description: "Memory usage is {{ $value }}% (threshold: 95%)"

      - alert: RedisEvictingKeys
        expr: rate(redis_evicted_keys_total[1m]) > 0
        for: 5m
        labels:
          severity: warning
          component: redis
        annotations:
          summary: "Redis is evicting keys"
          description: "Eviction rate: {{ $value }}/s"
```

### AlertManager Configuration

Create file: `alertmanager-redis-config.yml`

```yaml
route:
  group_by: ['alertname', 'component']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'redis-team'
  routes:
    - match:
        severity: critical
        component: redis
      receiver: 'redis-oncall'
      continue: true
    - match:
        severity: warning
        component: redis
      receiver: 'redis-team'

receivers:
  - name: 'redis-team'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#redis-alerts'
        title: 'Redis Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'redis-oncall'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#redis-critical'
        title: 'CRITICAL Redis Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

---

## Grafana Dashboard Usage

### Accessing the Dashboard

1. Open Grafana: `http://localhost:3000`
2. Login: admin/admin
3. Navigate to: Dashboards → Redis Optimization

### Dashboard Panels

#### 1. Overview Panel

**Metrics Displayed:**
- Current QPS
- P99 Latency
- Error Rate
- Cache Hit Rate

**What to Look For:**
- QPS should match expected load
- P99 latency should be <10ms
- Error rate should be <0.1%
- Cache hit rate should be >95%

#### 2. Performance Panel

**Metrics Displayed:**
- Request latency (P50, P95, P99)
- Request throughput
- Error rate by type

**What to Look For:**
- Latency spikes
- Throughput drops
- Error rate increases

#### 3. Cache Panel

**Metrics Displayed:**
- Cache hit rate
- Cache operations (GET/SET/DELETE)
- Cache size
- Eviction rate

**What to Look For:**
- Hit rate trends
- Unusual operation patterns
- Evictions (should be zero)

#### 4. Connection Pool Panel

**Metrics Displayed:**
- Active connections
- Idle connections
- Pool utilization
- Pool hit rate

**What to Look For:**
- Utilization approaching 100%
- Pool misses
- Timeouts

#### 5. Circuit Breaker Panel

**Metrics Displayed:**
- Circuit breaker state
- Failure rate
- Rejected requests

**What to Look For:**
- State changes
- Open circuit
- High failure rate

#### 6. Singleflight Panel

**Metrics Displayed:**
- Coalescing rate
- Execute count
- Wait count
- Timeout count

**What to Look For:**
- Low coalescing rate
- Timeouts
- High execute count (indicates cache misses)

#### 7. Redis Server Panel

**Metrics Displayed:**
- Memory usage
- Commands per second
- Keyspace hit rate
- Connected clients

**What to Look For:**
- Memory approaching limit
- High command rate
- Low keyspace hit rate

---

## Monitoring Best Practices

### 1. Set Up Baseline Metrics

```bash
# Collect baseline metrics during normal operation
# Run for at least 24 hours to capture daily patterns

# Export metrics
curl http://localhost:9091/metrics > baseline-metrics.txt

# Analyze baseline
cat baseline-metrics.txt | grep -E "(http_request_duration|cache_hit|pool_utilization)"
```

### 2. Create Custom Dashboards

```bash
# Export existing dashboard
curl http://localhost:3000/api/dashboards/uid/redis-optimization > dashboard.json

# Customize and import
# Edit dashboard.json
curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @dashboard.json
```

### 3. Set Up Log Aggregation

```yaml
# Loki configuration for log aggregation
# docker-compose.yml
loki:
  image: grafana/loki:latest
  ports:
    - "3100:3100"
  volumes:
    - ./loki-config.yaml:/etc/loki/local-config.yaml

promtail:
  image: grafana/promtail:latest
  volumes:
    - /var/log:/var/log
    - ./promtail-config.yaml:/etc/promtail/config.yml
```

### 4. Regular Review Schedule

**Daily:**
- Check dashboard for anomalies
- Review critical alerts
- Verify cache hit rate

**Weekly:**
- Review metric trends
- Analyze slow queries
- Check for memory leaks

**Monthly:**
- Compare with baseline
- Review alert thresholds
- Update documentation

---

## Troubleshooting with Metrics

### Scenario 1: High Latency

```promql
# Check P99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))

# If high, check:
# 1. Connection pool utilization
redis_pool_active_connections / redis_pool_size * 100

# 2. Circuit breaker state
redis_circuit_breaker_state

# 3. Cache hit rate
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m])) * 100
```

### Scenario 2: Low Cache Hit Rate

```promql
# Check cache hit rate
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m])) * 100

# If low, check:
# 1. Redis memory usage
redis_memory_used_bytes / redis_memory_max_bytes * 100

# 2. Eviction rate
rate(redis_evicted_keys_total[1m])

# 3. Cache operations
rate(redis_cache_set_total[1m])
```

### Scenario 3: Circuit Breaker Open

```promql
# Check circuit breaker state
redis_circuit_breaker_state

# If open, check:
# 1. Redis health
up{job="redis"}

# 2. Failure rate
rate(redis_circuit_breaker_failures_total[1m])

# 3. Rejected requests
rate(redis_circuit_breaker_rejected_total[1m])
```

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Redis Metrics](https://redis.io/topics/metrics)
- [Troubleshooting Guide](./REDIS_TROUBLESHOOTING_GUIDE.md)
- [Configuration Guide](./REDIS_CONFIGURATION_GUIDE.md)

---
