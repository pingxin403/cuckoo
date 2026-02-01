# URL Shortener Service - Performance Quick Reference

## 🎯 Performance Targets

| Metric | Target | Current Status |
|--------|--------|----------------|
| Redirect P99 Latency | ≤ 10ms | ⚠️ Not measured |
| Creation P99 Latency | ≤ 50ms | ⚠️ Not measured |
| Read Throughput | 500K+ QPS | ⚠️ Not tested |
| Write Throughput | 10K+ RPS | ⚠️ Not tested |
| Cache Hit Rate | ≥ 95% | ⚠️ Not measured |
| Availability | ≥ 99.99% | ⚠️ Not measured |

## ✅ What's Working Well

1. **Multi-tier caching** - L1 (Ristretto) → L2 (Redis) → MySQL
2. **Singleflight pattern** - Prevents cache stampede
3. **Async analytics** - Non-blocking click tracking
4. **Graceful degradation** - Continues on Redis failure
5. **Security headers** - Comprehensive protection
6. **Observability** - Metrics, logging, tracing

## ⚠️ Critical Issues (Fix Before Production)

### 1. Missing L2 Cache TTL Jitter 🔴
**Impact:** Thundering herd when many Redis keys expire simultaneously  
**Fix Time:** 1 hour  
**Priority:** HIGH

```go
// apps/shortener-service/cache/l2_cache.go
func (c *L2Cache) Set(ctx context.Context, shortCode, longURL string, createdAt time.Time) error {
    // Add random jitter: ±1 day
    jitter := time.Duration(rand.Intn(86400*2)-86400) * time.Second
    ttl := c.ttl + jitter
    return c.client.Set(ctx, key, longURL, ttl).Err()
}
```


### 2. No Rate Limiting 🔴
**Impact:** Vulnerable to abuse and DDoS attacks  
**Fix Time:** 4 hours  
**Priority:** HIGH

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters sync.Map // map[string]*rate.Limiter
}

func (rl *RateLimiter) Allow(ip string) bool {
    limiter, _ := rl.limiters.LoadOrStore(ip, 
        rate.NewLimiter(rate.Every(600*time.Millisecond), 100))
    return limiter.(*rate.Limiter).Allow()
}
```

### 3. No Load Testing 🔴
**Impact:** Unknown actual performance characteristics  
**Fix Time:** 8 hours  
**Priority:** HIGH

**Action Items:**
- Create k6 load test scripts
- Test sustained 100K QPS redirect load
- Test 5K QPS creation load
- Measure P99 latency under load
- Document performance baselines

### 4. No Bloom Filter 🟡
**Impact:** Vulnerable to cache penetration attacks  
**Fix Time:** 8 hours  
**Priority:** MEDIUM

```go
import "github.com/bits-and-blooms/bloom/v3"

var bf *bloom.BloomFilter

func init() {
    bf = bloom.NewWithEstimates(100_000_000, 0.001)
    loadAllShortCodesToBloom()
}

func MightExist(shortCode string) bool {
    return bf.Test([]byte(shortCode))
}
```


## 🔄 Recommended Improvements (For Scale)

### 1. Snowflake ID Generation 🟡
**Benefit:** Better scalability, no collision checks  
**Effort:** 16 hours  
**Priority:** MEDIUM (for >1M QPS)

**When to implement:** When collision rate exceeds 0.1% or scaling beyond single instance

### 2. Hot Key Detection 🟢
**Benefit:** Better performance for viral links  
**Effort:** 24 hours  
**Priority:** LOW (for MVP), HIGH (for scale)

**Implementation:**
- Monitor top 1000 hot keys via metrics
- Store in separate sync.Map (never evict)
- Automatically promote based on access frequency

### 3. Redis Cluster 🟡
**Benefit:** Higher throughput, better availability  
**Effort:** 24 hours  
**Priority:** MEDIUM (for production)

**When to implement:** When single Redis instance approaches 50K QPS

### 4. Database Sharding 🟢
**Benefit:** Higher write throughput  
**Effort:** 40 hours  
**Priority:** LOW (for MVP)

**When to implement:** When write throughput exceeds 10K QPS

---

## 📊 Performance Optimization Roadmap

### Phase 1: MVP Fixes (1-2 weeks) 🔴
**Goal:** Production-ready with basic performance

- [ ] Add L2 cache TTL jitter (1 hour)
- [ ] Implement rate limiting (4 hours)
- [ ] Conduct load testing (8 hours)
- [ ] Add Bloom Filter (8 hours)
- [ ] Document performance baselines (2 hours)

**Expected Performance:**
- Redirect P99: < 10ms
- Read throughput: 100K+ QPS
- Write throughput: 5K+ RPS


### Phase 2: Scale to 500K QPS (1-2 months) 🟡
**Goal:** Handle high traffic with excellent performance

- [ ] Deploy Redis Cluster (24 hours)
- [ ] Implement hot key detection (24 hours)
- [ ] Horizontal scaling (multiple instances)
- [ ] CDN integration for static redirects
- [ ] Database read replicas

**Expected Performance:**
- Redirect P99: < 5ms
- Read throughput: 500K+ QPS
- Write throughput: 10K+ RPS
- Cache hit rate: > 95%

### Phase 3: Scale to 1M+ QPS (2-3 months) 🟢
**Goal:** Global scale with multi-region deployment

- [ ] Migrate to Snowflake ID generation (16 hours)
- [ ] Implement database sharding (40 hours)
- [ ] Multi-region deployment
- [ ] Edge computing (Cloudflare Workers)
- [ ] Advanced monitoring and auto-scaling

**Expected Performance:**
- Redirect P99: < 3ms
- Read throughput: 1M+ QPS
- Write throughput: 50K+ RPS
- Global availability: 99.99%

---

## 🔍 Monitoring Checklist

### Key Metrics to Track

```bash
# Cache hit rates (should be > 95%)
shortener_cache_hits_total{layer="L1"}
shortener_cache_hits_total{layer="L2"}

# Request latency (P99 should be < 10ms)
shortener_request_duration_seconds{quantile="0.99"}

# Error rates (should be < 0.1%)
shortener_errors_total

# Singleflight efficiency
shortener_singleflight_waits_total

# Redirect throughput
rate(shortener_redirects_total[1m])
```


### Alerts to Configure

```yaml
# High latency alert
- alert: HighRedirectLatency
  expr: histogram_quantile(0.99, rate(shortener_request_duration_seconds_bucket[5m])) > 0.01
  for: 5m
  annotations:
    summary: "P99 latency exceeds 10ms"

# Low cache hit rate
- alert: LowCacheHitRatio
  expr: sum(rate(shortener_cache_hits_total[5m])) / 
        (sum(rate(shortener_cache_hits_total[5m])) + 
         sum(rate(shortener_cache_misses_total[5m]))) < 0.95
  for: 10m
  annotations:
    summary: "Cache hit ratio below 95%"

# High error rate
- alert: HighErrorRate
  expr: rate(shortener_errors_total[5m]) > 10
  for: 5m
  annotations:
    summary: "Error rate exceeds 10/sec"
```

---

## 🚀 Quick Start for Performance Testing

### 1. Start Test Environment

```bash
# Start all dependencies
docker-compose -f apps/shortener-service/docker-compose.test.yml up -d

# Wait for services to be healthy
docker-compose -f apps/shortener-service/docker-compose.test.yml ps
```

### 2. Run Load Tests

```bash
# Install k6
brew install k6  # macOS
# or download from https://k6.io/

# Run redirect load test (100K QPS)
k6 run --vus 1000 --duration 5m apps/shortener-service/load-tests/redirect-test.js

# Run creation load test (5K QPS)
k6 run --vus 100 --duration 5m apps/shortener-service/load-tests/create-test.js
```

### 3. Monitor Metrics

```bash
# View Prometheus metrics
curl http://localhost:9090/metrics | grep shortener

# Check cache hit rate
curl -s http://localhost:9090/metrics | grep cache_hits_total

# Check latency
curl -s http://localhost:9090/metrics | grep request_duration_seconds
```


---

## 📚 Additional Resources

### Documentation
- [Full Performance Analysis](./docs/PERFORMANCE_ANALYSIS.md) - Detailed comparison with article
- [Service README](./README.md) - Setup and usage guide
- [API Documentation](./docs/API.md) - Complete API reference
- [Design Document](../../.kiro/specs/url-shortener-service/design.md) - Architecture details

### Article Reference
- **Title:** 构建百万 QPS 短链系统的 Go 实践
- **Key Topics:** Snowflake, sync.Map, Singleflight, Bloom Filter, Token Bucket

### External Resources
- [Ristretto Cache](https://github.com/dgraph-io/ristretto)
- [Singleflight Pattern](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [Bloom Filter](https://github.com/bits-and-blooms/bloom)
- [Rate Limiting](https://pkg.go.dev/golang.org/x/time/rate)
- [k6 Load Testing](https://k6.io/docs/)

---

## 🎓 Key Learnings from Article

### 1. The Essence of High Concurrency
> "百万 QPS 不是靠单机堆性能，而是靠分层削峰、智能缓存、优雅降级"  
> (Million QPS is not achieved by single-machine performance, but by layered peak shaving, intelligent caching, and graceful degradation)

### 2. Design Philosophy
- **Read path:** Memory → Jump, avoid any unnecessary IO
- **Write path:** Async persistence + idempotency
- **Fault tolerance:** Cache failure ≠ Service unavailable
- **Cost control:** Hot keys auto-converge, cold data auto-evict

### 3. Best Performance Optimization
> "最好的性能优化，是让请求根本不需要处理"  
> (The best performance optimization is to make requests not need processing at all)

**Translation:** Use caching aggressively to avoid hitting the database.

---

**Document Version:** 1.0  
**Last Updated:** 2025-02-01  
**Status:** Ready for Review

