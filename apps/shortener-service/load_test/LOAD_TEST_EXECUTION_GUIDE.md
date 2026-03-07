# Load Test Execution Guide

**Date:** 2026-02-03  
**Task:** 14.2 - Run load tests with k6

---

## Overview

This guide provides step-by-step instructions for executing load tests with k6 to validate Redis optimizations.

---

## Prerequisites

### 1. Install k6

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Verify installation
k6 version
```

### 2. System Requirements

- **CPU:** 8+ cores recommended for high QPS tests
- **Memory:** 16GB+ RAM recommended
- **Network:** Low latency connection to test environment
- **Docker:** For running test environment

---

## Setup Test Environment

### Option 1: Automated Setup (Recommended)

```bash
cd apps/shortener-service/load_test

# Run setup script
./setup-loadtest.sh
```

This script will:
1. Check Docker is running
2. Build service image
3. Start all services (MySQL, Redis, Prometheus, Grafana, Shortener)
4. Wait for all services to be healthy
5. Display service URLs

### Option 2: Manual Setup

```bash
cd apps/shortener-service/load_test

# Stop any existing containers
docker-compose -f docker-compose.loadtest.yml down -v

# Build service image
cd ..
docker build -t shortener-service:loadtest .
cd load_test

# Start services
docker-compose -f docker-compose.loadtest.yml up -d

# Wait for services to be ready
docker-compose -f docker-compose.loadtest.yml ps

# Check service health
curl http://localhost:8080/health
```

### Verify Environment

```bash
# Check all services are running
docker-compose -f docker-compose.loadtest.yml ps

# Expected output:
# NAME                              STATUS
# shortener-mysql-loadtest          Up (healthy)
# shortener-redis-loadtest          Up (healthy)
# shortener-service-loadtest        Up (healthy)
# shortener-prometheus-loadtest     Up
# shortener-grafana-loadtest        Up

# Test service endpoint
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Test Prometheus
curl http://localhost:9090/-/healthy
# Expected: Prometheus is Healthy.

# Test Grafana
curl http://localhost:3000/api/health
# Expected: {"commit":"...","database":"ok","version":"..."}
```

---

## Running Load Tests

### Test 1: Cache Stampede Test (Quick - ~30 seconds)

**Purpose:** Validate SETNX + Singleflight prevents cache stampede

```bash
cd apps/shortener-service/load_test

# Run test
k6 run cache-stampede.js

# With custom base URL
k6 run -e BASE_URL=http://localhost:8080 cache-stampede.js

# Save results to file
k6 run --out json=results/cache-stampede.json cache-stampede.js
```

**Expected Results:**
- ✅ All 1000 requests succeed
- ✅ Only 1-2 DB queries (99% reduction)
- ✅ P99 latency < 100ms
- ✅ Error rate < 0.1%

**Monitor During Test:**
```bash
# Watch Redis metrics
watch -n 1 'curl -s http://localhost:9091/metrics | grep singleflight'

# Watch DB queries
docker-compose -f docker-compose.loadtest.yml logs -f shortener-service | grep "DB query"
```

---

### Test 2: Spike Test (Medium - ~4 minutes)

**Purpose:** Validate system handles sudden traffic spikes

```bash
cd apps/shortener-service/load_test

# Run test
k6 run spike-test.js

# With custom settings
k6 run -e BASE_URL=http://localhost:8080 spike-test.js

# Save results
k6 run --out json=results/spike-test.json spike-test.js
```

**Expected Results:**
- ✅ Peak throughput: 400K+ QPS
- ✅ P99 latency < 10ms during spike
- ✅ Error rate < 1%
- ✅ System recovers in < 30s after spike

**Monitor During Test:**
```bash
# Watch request rate
watch -n 1 'curl -s http://localhost:9091/metrics | grep http_requests_total'

# Watch latency
watch -n 1 'curl -s http://localhost:9091/metrics | grep http_request_duration'

# Watch connection pool
watch -n 1 'curl -s http://localhost:9091/metrics | grep redis_pool'
```

---

### Test 3: Sustained Load Test (Long - ~10 minutes)

**Purpose:** Validate system handles sustained high load

```bash
cd apps/shortener-service/load_test

# Run test (will take 10 minutes)
k6 run sustained-load.js

# With custom settings
k6 run -e BASE_URL=http://localhost:8080 sustained-load.js

# Save results
k6 run --out json=results/sustained-load.json sustained-load.js
```

**Expected Results:**
- ✅ Sustained throughput: 90K+ QPS
- ✅ P99 latency < 5ms
- ✅ Cache hit rate > 95%
- ✅ Error rate < 0.1%

**Monitor During Test:**
```bash
# Watch all metrics in Grafana
open http://localhost:3000

# Or use Prometheus queries
open http://localhost:9090

# Key queries:
# - rate(http_requests_total[1m])
# - histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))
# - rate(redis_cache_hits_total[1m]) / (rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m]))
```

---

### Run All Tests (Automated)

```bash
cd apps/shortener-service/load_test

# Run all tests sequentially
./run-all-tests.sh

# This will:
# 1. Run cache stampede test (~30s)
# 2. Run spike test (~4m)
# 3. Ask to run sustained load test (~10m)
# 4. Generate summary report
```

---

## Monitoring During Tests

### 1. Real-time k6 Output

k6 provides real-time metrics during test execution:

```
     ✓ status is 200
     ✓ response time < 5ms

     checks.........................: 100.00% ✓ 6000000 ✗ 0
     data_received..................: 1.2 GB  2.0 MB/s
     data_sent......................: 600 MB  1.0 MB/s
     http_req_duration..............: avg=2.5ms   p(99)=4.5ms
     http_req_failed................: 0.00%   ✓ 0       ✗ 6000000
     http_reqs......................: 6000000 100000/s
     iterations.....................: 6000000 100000/s
     vus............................: 1000    min=1000  max=1000
```

### 2. Prometheus Queries

Access Prometheus at `http://localhost:9090` and run queries:

```promql
# Request rate
rate(http_requests_total[1m])

# P99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))

# Cache hit rate
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m]))

# DB query rate
rate(db_queries_total[1m])

# Connection pool utilization
redis_pool_active_connections / redis_pool_size

# Circuit breaker state
redis_circuit_breaker_state

# Singleflight efficiency
rate(singleflight_wait_total[1m]) / 
(rate(singleflight_execute_total[1m]) + rate(singleflight_wait_total[1m]))
```

### 3. Grafana Dashboard

Access Grafana at `http://localhost:3000` (admin/admin)

**Key Panels to Watch:**
1. **Request Throughput** - Should match target QPS
2. **Latency Percentiles** - P50, P95, P99
3. **Cache Hit Rate** - Should be >95%
4. **Connection Pool Stats** - Active/idle connections
5. **Circuit Breaker State** - Should stay closed
6. **Singleflight Coalescing** - Should show high efficiency

### 4. Service Logs

```bash
# Follow service logs
docker-compose -f docker-compose.loadtest.yml logs -f shortener-service

# Filter for errors
docker-compose -f docker-compose.loadtest.yml logs shortener-service | grep ERROR

# Filter for cache metrics
docker-compose -f docker-compose.loadtest.yml logs shortener-service | grep cache
```

---

## Analyzing Results

### 1. k6 Summary Report

After each test, k6 displays a summary:

```
     ✓ status is 200
     ✓ response time < 5ms

     checks.........................: 100.00% ✓ 6000000 ✗ 0
     data_received..................: 1.2 GB  2.0 MB/s
     data_sent......................: 600 MB  1.0 MB/s
     http_req_blocked...............: avg=1.2µs   p(99)=10ms
     http_req_connecting............: avg=0s      p(99)=5ms
     http_req_duration..............: avg=2.5ms   p(99)=4.5ms
       { expected_response:true }...: avg=2.5ms   p(99)=4.5ms
     http_req_failed................: 0.00%   ✓ 0       ✗ 6000000
     http_req_receiving.............: avg=50µs    p(99)=100µs
     http_req_sending...............: avg=20µs    p(99)=40µs
     http_req_tls_handshaking.......: avg=0s      p(99)=0s
     http_req_waiting...............: avg=2.43ms  p(99)=4.4ms
     http_reqs......................: 6000000 100000/s
     iteration_duration.............: avg=2.6ms   p(99)=4.6ms
     iterations.....................: 6000000 100000/s
     vus............................: 1000    min=1000  max=1000
     vus_max........................: 1000    min=1000  max=1000
```

**Key Metrics:**
- `http_reqs`: Total requests and rate (should match target QPS)
- `http_req_duration`: Latency statistics (check P99)
- `http_req_failed`: Error rate (should be <0.1%)
- `checks`: Validation pass rate (should be 100%)

### 2. JSON Results

If you saved results to JSON:

```bash
# View results
cat results/cache-stampede.json | jq '.metrics'

# Extract P99 latency
cat results/cache-stampede.json | jq '.metrics.http_req_duration.values.p(99)'

# Extract error rate
cat results/cache-stampede.json | jq '.metrics.http_req_failed.values.rate'
```

### 3. Compare with Targets

| Test | Metric | Target | Acceptable | Result |
|------|--------|--------|------------|--------|
| **Sustained Load** | Throughput | 100K QPS | 90K+ QPS | ? |
| | P99 Latency | < 5ms | < 10ms | ? |
| | Cache Hit Rate | > 95% | > 90% | ? |
| | Error Rate | < 0.1% | < 1% | ? |
| **Spike Test** | Peak Throughput | 500K QPS | 400K+ QPS | ? |
| | P99 Latency | < 10ms | < 20ms | ? |
| | Error Rate | < 1% | < 5% | ? |
| | Recovery Time | < 30s | < 60s | ? |
| **Cache Stampede** | DB Queries | < 10 | < 50 | ? |
| | DB Load Reduction | > 90% | > 80% | ? |
| | P99 Latency | < 100ms | < 200ms | ? |
| | Error Rate | < 0.1% | < 1% | ? |

---

## Troubleshooting

### Issue: k6 reports high error rate

**Possible Causes:**
1. Service not scaled properly
2. Database connection pool exhausted
3. Redis connection pool exhausted
4. Circuit breaker open

**Solutions:**
```bash
# Check service logs
docker-compose -f docker-compose.loadtest.yml logs shortener-service

# Check connection pool metrics
curl http://localhost:9091/metrics | grep pool

# Check circuit breaker state
curl http://localhost:9091/metrics | grep circuit_breaker_state

# Increase connection pool size
# Edit docker-compose.loadtest.yml:
# REDIS_POOL_SIZE: 200
# REDIS_MIN_IDLE_CONNS: 60

# Restart service
docker-compose -f docker-compose.loadtest.yml restart shortener-service
```

### Issue: Low throughput

**Possible Causes:**
1. Not enough VUs allocated
2. Network bottleneck
3. Service resource limits

**Solutions:**
```bash
# Increase VUs in test script
# Edit sustained-load.js:
# preAllocatedVUs: 2000
# maxVUs: 4000

# Check resource usage
docker stats

# Increase service resources
# Edit docker-compose.loadtest.yml:
# deploy:
#   resources:
#     limits:
#       cpus: '4'
#       memory: 4G
```

### Issue: High latency

**Possible Causes:**
1. Cache misses
2. Slow database queries
3. Connection pool contention

**Solutions:**
```bash
# Check cache hit rate
curl http://localhost:9091/metrics | grep cache_hit

# Check DB query time
docker-compose -f docker-compose.loadtest.yml logs shortener-service | grep "query time"

# Warm up cache before test
# Run a few requests first to populate cache
```

---

## Cleanup

### Stop Test Environment

```bash
cd apps/shortener-service/load_test

# Stop all services
docker-compose -f docker-compose.loadtest.yml down

# Stop and remove volumes (clean slate)
docker-compose -f docker-compose.loadtest.yml down -v
```

### Save Results

```bash
# Create results archive
cd apps/shortener-service/load_test
tar -czf load-test-results-$(date +%Y%m%d).tar.gz results/

# Move to docs
mv load-test-results-*.tar.gz ../docs/
```

---

## Next Steps

After running load tests:

1. **Analyze Results** - Review all metrics and compare with targets

---

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Best Practices](https://k6.io/docs/testing-guides/test-types/)
- [Benchmark Results](../docs/BENCHMARK_RESULTS.md)

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-03  
