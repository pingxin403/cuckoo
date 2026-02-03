# Redis Troubleshooting Guide

---

## Overview

This guide provides step-by-step troubleshooting procedures for common Redis issues in the shortener service. Use this guide to diagnose and resolve Redis-related problems quickly.

---

## Quick Diagnosis Checklist

Before diving into specific issues, run this quick checklist:

```bash
# 1. Check Redis is running
docker ps | grep redis
# or
redis-cli ping

# 2. Check service logs
docker logs shortener-service | tail -50

# 3. Check Redis metrics
curl http://localhost:9091/metrics | grep redis

# 4. Check connection pool status
curl http://localhost:9091/metrics | grep redis_pool

# 5. Check circuit breaker state
curl http://localhost:9091/metrics | grep circuit_breaker_state
```

---

## Common Issues

### Issue 1: High Latency (P99 > 10ms)

#### Symptoms
- Slow response times
- Increased P99 latency metrics
- User complaints about performance

#### Diagnosis

```bash
# Check current latency
curl http://localhost:9091/metrics | grep http_request_duration

# Check Redis latency
redis-cli --latency

# Check connection pool utilization
curl http://localhost:9091/metrics | grep redis_pool_active_connections

# Check for slow queries
redis-cli slowlog get 10
```

#### Common Causes

1. **Connection Pool Exhaustion**
   ```bash
   # Check pool metrics
   curl http://localhost:9091/metrics | grep redis_pool
   
   # Look for:
   # redis_pool_active_connections near redis_pool_size
   # redis_pool_misses_total increasing
   ```

2. **Network Issues**
   ```bash
   # Check network latency to Redis
   ping redis-host
   
   # Check Redis network stats
   redis-cli info stats | grep instantaneous
   ```

3. **Memory Pressure**
   ```bash
   # Check Redis memory usage
   redis-cli info memory
   
   # Look for:
   # used_memory_human
   # maxmemory_human
   # evicted_keys (should be low)
   ```

4. **GC Pauses (Service)**
   ```bash
   # Check service logs for GC
   docker logs shortener-service | grep "GC pause"
   
   # Check memory usage
   docker stats shortener-service
   ```

#### Solutions

**Solution 1: Increase Connection Pool Size**
```bash
# Edit configuration
export REDIS_POOL_SIZE=150
export REDIS_MIN_IDLE_CONNS=45

# Restart service
docker restart shortener-service
```

**Solution 2: Tune GC Settings**
```bash
# Add to service environment
export GOGC=200

# Restart service
docker restart shortener-service
```

**Solution 3: Add Redis Read Replicas**
```yaml
# docker-compose.yml
redis-replica:
  image: redis:7.0-alpine
  command: redis-server --replicaof redis-master 6379
```

**Solution 4: Increase Redis Memory**
```bash
# Edit redis.conf
maxmemory 4gb

# Restart Redis
docker restart redis
```

---

### Issue 2: Connection Pool Exhaustion

#### Symptoms
- "connection pool timeout" errors
- High `redis_pool_misses_total` metric
- `redis_pool_active_connections` equals `redis_pool_size`

#### Diagnosis

```bash
# Check pool metrics
curl http://localhost:9091/metrics | grep redis_pool

# Expected healthy values:
# redis_pool_active_connections: 60-80% of pool_size
# redis_pool_idle_connections: 20-40% of pool_size
# redis_pool_hits_total: >99% of total requests
# redis_pool_misses_total: <1% of total requests
```

#### Common Causes

1. **Pool Size Too Small**
   - Current pool size insufficient for load
   - Check: `redis_pool_size` vs `redis_pool_active_connections`

2. **Connection Leaks**
   - Connections not being returned to pool
   - Check: Increasing active connections over time

3. **Slow Queries Holding Connections**
   - Long-running queries blocking connections
   - Check: Redis slow log

#### Solutions

**Solution 1: Increase Pool Size**
```bash
# Calculate required pool size
# Formula: (Expected QPS / 1000) * 2
# Example: 100K QPS → 200 connections

export REDIS_POOL_SIZE=200
export REDIS_MIN_IDLE_CONNS=60

docker restart shortener-service
```

**Solution 2: Check for Connection Leaks**
```bash
# Monitor active connections over time
watch -n 5 'curl -s http://localhost:9091/metrics | grep redis_pool_active_connections'

# If continuously increasing, check code for:
# - Missing defer statements
# - Unclosed connections
# - Goroutine leaks
```

**Solution 3: Optimize Slow Queries**
```bash
# Check slow log
redis-cli slowlog get 10

# Optimize queries:
# - Use pipeline for batch operations
# - Add indexes if needed
# - Reduce data size
```

---

### Issue 3: Circuit Breaker Open

#### Symptoms
- "circuit breaker open" errors in logs
- `redis_circuit_breaker_state` = 1 (open)
- Requests falling back to database
- Increased database load

#### Diagnosis

```bash
# Check circuit breaker state
curl http://localhost:9091/metrics | grep circuit_breaker

# Check Redis health
redis-cli ping

# Check Redis errors
docker logs redis | grep ERROR

# Check network connectivity
telnet redis-host 6379
```

#### Common Causes

1. **Redis Down or Unreachable**
   - Redis service stopped
   - Network issues
   - Firewall blocking connection

2. **Redis Overloaded**
   - Too many connections
   - Memory exhausted
   - CPU saturation

3. **Timeout Issues**
   - Read/write timeouts too aggressive
   - Network latency increased

#### Solutions

**Solution 1: Restart Redis**
```bash
# Check Redis status
docker ps | grep redis

# Restart if needed
docker restart redis

# Wait for circuit breaker to recover (30 seconds)
watch -n 1 'curl -s http://localhost:9091/metrics | grep circuit_breaker_state'
```

**Solution 2: Check Redis Resources**
```bash
# Check Redis memory
redis-cli info memory

# Check Redis CPU
docker stats redis

# If overloaded:
# - Increase Redis memory
# - Add read replicas
# - Scale horizontally
```

**Solution 3: Adjust Timeouts**
```bash
# Increase timeouts if network latency is high
export REDIS_READ_TIMEOUT=5s
export REDIS_WRITE_TIMEOUT=5s

docker restart shortener-service
```

**Solution 4: Manual Circuit Breaker Reset**
```bash
# If circuit breaker stuck open, restart service
docker restart shortener-service

# Circuit breaker will reset to closed state
```

---

### Issue 4: Cache Stampede

#### Symptoms
- Sudden spike in database queries
- Multiple concurrent requests for same key
- High `singleflight_execute_total` metric
- Database overload

#### Diagnosis

```bash
# Check singleflight metrics
curl http://localhost:9091/metrics | grep singleflight

# Check SETNX lock metrics
curl http://localhost:9091/metrics | grep setnx

# Check database query rate
curl http://localhost:9091/metrics | grep db_queries_total

# Check service logs for cache misses
docker logs shortener-service | grep "cache miss"
```

#### Common Causes

1. **SETNX Not Working**
   - Lock acquisition failing
   - Lock timeout too short
   - Redis connection issues

2. **Singleflight Not Working**
   - Multiple goroutines not coalescing
   - Context timeout too short
   - Implementation bug

3. **Mass Cache Expiration**
   - Many keys expiring simultaneously
   - TTL jitter not working
   - Cache cleared manually

#### Solutions

**Solution 1: Verify SETNX Configuration**
```bash
# Check SETNX metrics
curl http://localhost:9091/metrics | grep setnx_lock

# Expected:
# redis_setnx_lock_acquired_total: Low (1-10 per cache miss)
# redis_setnx_lock_wait_total: High (90%+ of requests)

# If not working, check Redis version
redis-cli info server | grep redis_version
# Should be 2.6.12 or higher for SETNX
```

**Solution 2: Verify Singleflight Configuration**
```bash
# Check singleflight metrics
curl http://localhost:9091/metrics | grep singleflight

# Expected:
# singleflight_execute_total: Low
# singleflight_wait_total: High
# Coalescing rate: >90%

# If not working, check service logs
docker logs shortener-service | grep singleflight
```

**Solution 3: Verify TTL Jitter**
```bash
# Check TTL distribution
curl http://localhost:9091/metrics | grep redis_ttl_seconds

# Should show distribution around 7 days ± 1 day

# Check cache expiration pattern
redis-cli --scan --pattern "shortener:*" | while read key; do
  redis-cli ttl "$key"
done | sort -n
```

**Solution 4: Warm Up Cache**
```bash
# Pre-populate cache for hot keys
# Run cache warming script
./scripts/warm-cache.sh

# Or manually warm specific keys
curl http://localhost:8080/api/v1/shortener/{popular-code}
```

---

### Issue 5: Low Cache Hit Rate (<90%)

#### Symptoms
- `redis_cache_hit_rate` < 90%
- High database load
- Slow response times
- Increased costs

#### Diagnosis

```bash
# Check cache hit rate
curl http://localhost:9091/metrics | grep cache_hit

# Calculate hit rate
# hit_rate = hits / (hits + misses)

# Check cache size
redis-cli dbsize

# Check eviction stats
redis-cli info stats | grep evicted_keys

# Check memory usage
redis-cli info memory | grep used_memory_human
```

#### Common Causes

1. **Insufficient Redis Memory**
   - Cache evicting keys too frequently
   - `evicted_keys` increasing rapidly

2. **TTL Too Short**
   - Keys expiring too quickly
   - Not enough time to benefit from cache

3. **Cache Not Warming Up**
   - Cold start after deployment
   - No pre-population of hot keys

4. **Traffic Pattern Changed**
   - New keys being requested
   - Long tail distribution

#### Solutions

**Solution 1: Increase Redis Memory**
```bash
# Check current memory
redis-cli info memory | grep maxmemory

# Increase memory limit
# Edit redis.conf
maxmemory 4gb

# Restart Redis
docker restart redis
```

**Solution 2: Adjust TTL**
```bash
# Current TTL: 7 days ± 1 day
# If hit rate low, consider increasing

# Edit cache configuration
export CACHE_TTL_DAYS=14
export CACHE_TTL_JITTER_DAYS=2

docker restart shortener-service
```

**Solution 3: Implement Cache Warming**
```bash
# Create cache warming script
cat > warm-cache.sh << 'EOF'
#!/bin/bash
# Warm up cache with top 1000 popular short codes
curl http://localhost:8080/api/v1/shortener/popular | \
  jq -r '.[]' | \
  xargs -P 10 -I {} curl http://localhost:8080/api/v1/shortener/{}
EOF

chmod +x warm-cache.sh
./warm-cache.sh
```

**Solution 4: Analyze Traffic Pattern**
```bash
# Check key access pattern
redis-cli --hotkeys

# Check key distribution
redis-cli --scan --pattern "shortener:*" | wc -l

# Consider:
# - Increasing cache size
# - Using LRU eviction (already configured)
# - Implementing tiered caching
```

---

### Issue 6: Redis Cluster Issues

#### Symptoms
- "MOVED" or "ASK" errors in logs
- Cross-slot operation errors
- Uneven key distribution
- Node failures

#### Diagnosis

```bash
# Check cluster status
redis-cli cluster info

# Check cluster nodes
redis-cli cluster nodes

# Check slot distribution
redis-cli cluster slots

# Check cluster metrics
curl http://localhost:9091/metrics | grep cluster
```

#### Common Causes

1. **Missing Hash Tags**
   - Keys not using hash tags
   - Related keys on different slots
   - Cross-slot operations failing

2. **Node Failures**
   - Master node down
   - Replica not promoted
   - Network partition

3. **Uneven Distribution**
   - Some nodes overloaded
   - Slot migration needed
   - Hot keys on same node

#### Solutions

**Solution 1: Verify Hash Tags**
```bash
# Check key format
redis-cli --scan --pattern "shortener:*" | head -10

# Should be: {shortener}:key
# Not: shortener:key

# If wrong format, update code to use hash tags
# See: cache/cluster_client.go
```

**Solution 2: Handle Node Failures**
```bash
# Check failed nodes
redis-cli cluster nodes | grep fail

# Manual failover if needed
redis-cli -c cluster failover

# Or restart failed node
docker restart redis-node-X
```

**Solution 3: Rebalance Slots**
```bash
# Check slot distribution
redis-cli cluster slots

# Rebalance if uneven
redis-cli --cluster rebalance localhost:7000

# Or use redis-trib
redis-trib.rb rebalance localhost:7000
```

**Solution 4: Handle Hot Keys**
```bash
# Identify hot keys
redis-cli --hotkeys

# Solutions:
# - Add read replicas
# - Use local cache for hot keys
# - Implement request coalescing
```

---

## Debugging Tools

### 1. Redis CLI Commands

```bash
# Basic health check
redis-cli ping

# Get server info
redis-cli info

# Monitor commands in real-time
redis-cli monitor

# Check slow queries
redis-cli slowlog get 10

# Check memory usage
redis-cli info memory

# Check connected clients
redis-cli client list

# Check key statistics
redis-cli --bigkeys
redis-cli --hotkeys

# Check latency
redis-cli --latency
redis-cli --latency-history

# Cluster commands
redis-cli cluster info
redis-cli cluster nodes
redis-cli cluster slots
```

### 2. Metrics Queries

```bash
# Connection pool
curl http://localhost:9091/metrics | grep redis_pool

# Cache hit rate
curl http://localhost:9091/metrics | grep cache_hit

# Circuit breaker
curl http://localhost:9091/metrics | grep circuit_breaker

# Singleflight
curl http://localhost:9091/metrics | grep singleflight

# SETNX locks
curl http://localhost:9091/metrics | grep setnx

# Pipeline
curl http://localhost:9091/metrics | grep pipeline

# Lua scripts
curl http://localhost:9091/metrics | grep lua_script
```

### 3. Prometheus Queries

```promql
# Request rate
rate(http_requests_total[1m])

# P99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))

# Cache hit rate
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m]))

# Connection pool utilization
redis_pool_active_connections / redis_pool_size

# Circuit breaker state
redis_circuit_breaker_state

# Singleflight efficiency
rate(singleflight_wait_total[1m]) / 
(rate(singleflight_execute_total[1m]) + rate(singleflight_wait_total[1m]))
```

### 4. Service Logs

```bash
# Follow logs
docker logs -f shortener-service

# Filter for errors
docker logs shortener-service | grep ERROR

# Filter for Redis
docker logs shortener-service | grep redis

# Filter for cache
docker logs shortener-service | grep cache

# Filter for circuit breaker
docker logs shortener-service | grep "circuit breaker"

# Filter for singleflight
docker logs shortener-service | grep singleflight
```

---

## Performance Tuning

### Connection Pool Tuning

```bash
# Calculate optimal pool size
# Formula: (Expected QPS / 1000) * 2

# For 100K QPS:
export REDIS_POOL_SIZE=200
export REDIS_MIN_IDLE_CONNS=60

# For 500K QPS:
export REDIS_POOL_SIZE=1000
export REDIS_MIN_IDLE_CONNS=300
```

### Timeout Tuning

```bash
# Default timeouts (aggressive)
export REDIS_DIAL_TIMEOUT=5s
export REDIS_READ_TIMEOUT=3s
export REDIS_WRITE_TIMEOUT=3s

# For high latency networks
export REDIS_DIAL_TIMEOUT=10s
export REDIS_READ_TIMEOUT=5s
export REDIS_WRITE_TIMEOUT=5s
```

### Circuit Breaker Tuning

```bash
# Default settings
export CIRCUIT_BREAKER_THRESHOLD=5
export CIRCUIT_BREAKER_TIMEOUT=30s

# More aggressive (fail fast)
export CIRCUIT_BREAKER_THRESHOLD=3
export CIRCUIT_BREAKER_TIMEOUT=10s

# More tolerant (allow more failures)
export CIRCUIT_BREAKER_THRESHOLD=10
export CIRCUIT_BREAKER_TIMEOUT=60s
```

---

## Emergency Procedures

### Procedure 1: Redis Down

```bash
# 1. Verify Redis is down
redis-cli ping
# Error: Could not connect

# 2. Check service status
docker ps | grep redis

# 3. Check logs
docker logs redis

# 4. Restart Redis
docker restart redis

# 5. Verify recovery
redis-cli ping
# PONG

# 6. Check service recovery
curl http://localhost:8080/health
# {"status":"ok"}

# 7. Monitor metrics
watch -n 1 'curl -s http://localhost:9091/metrics | grep circuit_breaker_state'
# Should return to 0 (closed) within 30 seconds
```

### Procedure 2: Cache Stampede

```bash
# 1. Identify stampede
curl http://localhost:9091/metrics | grep singleflight_execute_total
# Rapidly increasing

# 2. Check database load
# Monitor database connections and query rate

# 3. Warm up cache
./scripts/warm-cache.sh

# 4. If severe, temporarily increase cache TTL
export CACHE_TTL_DAYS=30
docker restart shortener-service

# 5. Monitor recovery
watch -n 1 'curl -s http://localhost:9091/metrics | grep cache_hit'
```

### Procedure 3: Connection Pool Exhausted

```bash
# 1. Verify exhaustion
curl http://localhost:9091/metrics | grep redis_pool_active_connections
# Equals redis_pool_size

# 2. Immediate fix: Increase pool size
export REDIS_POOL_SIZE=300
export REDIS_MIN_IDLE_CONNS=90
docker restart shortener-service

# 3. Monitor recovery
watch -n 1 'curl -s http://localhost:9091/metrics | grep redis_pool'

# 4. Investigate root cause
# - Check for connection leaks
# - Check for slow queries
# - Check for traffic spike
```

---

## Preventive Maintenance

### Daily Checks

```bash
# 1. Check Redis health
redis-cli ping

# 2. Check memory usage
redis-cli info memory | grep used_memory_human

# 3. Check cache hit rate
curl http://localhost:9091/metrics | grep cache_hit

# 4. Check error rate
curl http://localhost:9091/metrics | grep http_req_failed

# 5. Check circuit breaker state
curl http://localhost:9091/metrics | grep circuit_breaker_state
```

### Weekly Checks

```bash
# 1. Review slow queries
redis-cli slowlog get 100

# 2. Check key distribution
redis-cli --bigkeys

# 3. Review metrics trends in Grafana
open http://localhost:3000

# 4. Check for memory leaks
docker stats shortener-service

# 5. Review error logs
docker logs shortener-service | grep ERROR | tail -100
```

### Monthly Checks

```bash
# 1. Review performance baselines
# Compare current metrics with baseline

# 2. Check for Redis updates
docker pull redis:7.0-alpine

# 3. Review and update documentation

# 4. Conduct load testing
cd apps/shortener-service/load_test
./run-all-tests.sh

# 5. Review and update alerts
# Check Prometheus alert rules
```

---

## References

- [Redis Documentation](https://redis.io/documentation)
- [Redis Troubleshooting](https://redis.io/topics/problems)
- [Monitoring Guide](./REDIS_MONITORING_GUIDE.md)
- [Configuration Guide](./REDIS_CONFIGURATION_GUIDE.md)
- [Disaster Recovery Guide](./REDIS_DISASTER_RECOVERY_GUIDE.md)

---

