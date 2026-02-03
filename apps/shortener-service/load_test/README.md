# Load Testing with k6

This directory contains k6 load test scenarios for validating Redis optimizations in the shortener service.

## Prerequisites

1. **Install k6:**
   ```bash
   # macOS
   brew install k6
   
   # Linux
   sudo gpg -k
   sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
   echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
   sudo apt-get update
   sudo apt-get install k6
   
   # Windows
   choco install k6
   ```

2. **Start the service:**
   ```bash
   # From project root
   cd apps/shortener-service
   
   # Start dependencies (Redis, MySQL)
   docker-compose -f integration_test/docker-compose.yml up -d
   
   # Start the service
   go run main.go
   ```

3. **Verify service is running:**
   ```bash
   curl http://localhost:8080/health
   ```

## Test Scenarios

### 1. Sustained Load Test

**File:** `sustained-load.js`

**Description:** Tests sustained high load (100K QPS) for 10 minutes.

**Targets:**
- Throughput: 100K QPS sustained
- P99 Latency: < 5ms
- Cache Hit Rate: > 95%
- Error Rate: < 0.1%

**Run:**
```bash
k6 run sustained-load.js
```

**With custom base URL:**
```bash
k6 run -e BASE_URL=http://localhost:8080 sustained-load.js
```

**Expected Results:**
- Consistent throughput around 100K QPS
- P99 latency under 5ms
- High cache hit rate (>95%)
- Minimal errors (<0.1%)

---

### 2. Spike Test

**File:** `spike-test.js`

**Description:** Tests sudden traffic spike (0 → 500K QPS in 1 minute).

**Targets:**
- Peak Throughput: 500K QPS
- P99 Latency: < 10ms (during spike)
- Error Rate: < 1%
- System Recovery: < 30s after spike

**Run:**
```bash
k6 run spike-test.js
```

**Expected Results:**
- System handles spike without crashing
- Latency increases but stays under 10ms (P99)
- Error rate stays under 1%
- System recovers quickly after spike

---

### 3. Cache Stampede Test

**File:** `cache-stampede.js`

**Description:** Tests cache stampede scenario (1000 concurrent requests for same key).

**Targets:**
- DB Queries: < 10 (for 1000 concurrent requests)
- DB Load Reduction: > 90%
- P99 Latency: < 100ms
- Error Rate: < 0.1%

**Run:**
```bash
k6 run cache-stampede.js
```

**Expected Results:**
- Only 1-2 DB queries (SETNX + Singleflight working)
- 99% DB load reduction
- All requests succeed
- Latency under 100ms (P99)

---

## Monitoring During Tests

### 1. Prometheus Metrics

Access Prometheus at `http://localhost:9090` and query:

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
```

### 2. Grafana Dashboard

Access Grafana at `http://localhost:3000` and view the Redis dashboard.

Key panels to watch:
- Request throughput
- Latency percentiles (P50, P95, P99)
- Cache hit rate
- Connection pool stats
- Circuit breaker state

### 3. k6 Output

k6 provides real-time metrics during test execution:

```
     ✓ status is 200
     ✓ response time < 5ms

     checks.........................: 100.00% ✓ 6000000 ✗ 0
     data_received..................: 1.2 GB  2.0 MB/s
     data_sent......................: 600 MB  1.0 MB/s
     http_req_blocked...............: avg=1.2µs   min=0s     med=1µs    max=10ms   p(90)=2µs    p(95)=3µs
     http_req_connecting............: avg=0s      min=0s     med=0s     max=5ms    p(90)=0s     p(95)=0s
     http_req_duration..............: avg=2.5ms   min=0.5ms  med=2ms    max=50ms   p(90)=4ms    p(95)=4.5ms
     http_req_failed................: 0.00%   ✓ 0       ✗ 6000000
     http_req_receiving.............: avg=50µs    min=10µs   med=40µs   max=5ms    p(90)=80µs   p(95)=100µs
     http_req_sending...............: avg=20µs    min=5µs    med=15µs   max=2ms    p(90)=30µs   p(95)=40µs
     http_req_tls_handshaking.......: avg=0s      min=0s     med=0s     max=0s     p(90)=0s     p(95)=0s
     http_req_waiting...............: avg=2.43ms  min=0.48ms med=1.95ms max=49ms   p(90)=3.9ms  p(95)=4.4ms
     http_reqs......................: 6000000 100000/s
     iteration_duration.............: avg=2.6ms   min=0.6ms  med=2.1ms  max=51ms   p(90)=4.1ms  p(95)=4.6ms
     iterations.....................: 6000000 100000/s
     vus............................: 1000    min=1000  max=1000
     vus_max........................: 1000    min=1000  max=1000
```

---

## Advanced Usage

### Run with custom VUs

```bash
k6 run --vus 2000 --duration 5m sustained-load.js
```

### Run with custom thresholds

```bash
k6 run --threshold 'http_req_duration{type:get}=p(99)<3' sustained-load.js
```

### Output results to file

```bash
k6 run --out json=results.json sustained-load.js
```

### Run with Prometheus remote write

```bash
k6 run --out experimental-prometheus-rw sustained-load.js
```

---

## Troubleshooting

### Issue: Connection refused

**Solution:** Ensure the service is running on the correct port.

```bash
# Check if service is running
curl http://localhost:8080/health

# Check logs
docker-compose logs shortener-service
```

### Issue: High error rate

**Possible causes:**
1. Service not scaled properly (increase resources)
2. Database connection pool exhausted (increase pool size)
3. Redis connection pool exhausted (increase pool size)
4. Circuit breaker open (check Redis health)

**Solution:** Check service logs and metrics.

### Issue: Low throughput

**Possible causes:**
1. Not enough VUs allocated (increase `preAllocatedVUs`)
2. Network bottleneck (run k6 on same machine)
3. Service resource limits (increase CPU/memory)

**Solution:** Increase VUs and check resource utilization.

---

## Performance Targets

### Sustained Load (100K QPS)

| Metric | Target | Acceptable |
|--------|--------|------------|
| Throughput | 100K QPS | 90K+ QPS |
| P99 Latency | < 5ms | < 10ms |
| Cache Hit Rate | > 95% | > 90% |
| Error Rate | < 0.1% | < 1% |

### Spike Test (500K QPS)

| Metric | Target | Acceptable |
|--------|--------|------------|
| Peak Throughput | 500K QPS | 400K+ QPS |
| P99 Latency | < 10ms | < 20ms |
| Error Rate | < 1% | < 5% |
| Recovery Time | < 30s | < 60s |

### Cache Stampede

| Metric | Target | Acceptable |
|--------|--------|------------|
| DB Queries | < 10 | < 50 |
| DB Load Reduction | > 90% | > 80% |
| P99 Latency | < 100ms | < 200ms |
| Error Rate | < 0.1% | < 1% |

---

## Next Steps

After running load tests:

1. **Analyze Results:**
   - Review k6 output
   - Check Prometheus metrics
   - Review Grafana dashboards

2. **Document Findings:**
   - Create load test report
   - Compare with baseline
   - Identify bottlenecks

3. **Optimize:**
   - Tune connection pools
   - Adjust cache TTLs
   - Scale resources

4. **Repeat:**
   - Run tests again
   - Validate improvements
   - Update documentation

---

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Metrics](https://k6.io/docs/using-k6/metrics/)
- [k6 Thresholds](https://k6.io/docs/using-k6/thresholds/)

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-03  
