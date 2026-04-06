# Flash Sale System Capacity Planning

## 1. System Overview

**Target**: Support 100,000+ QPS during flash sale peak
**Architecture**: Three-layer funnel model (Anti-fraud → Queue → Inventory)

## 2. Component Capacity Requirements

### 2.1 Redis (Inventory Layer)

| Metric | Target | Implementation |
|--------|--------|----------------|
| QPS per instance | ≥50,000 | Lua script atomic operations |
| Connection pool | 100-200 | Lettuce with Commons Pool2 |
| Memory | 10GB+ | Store stock, tokens, order status |
| Persistence | AOF + RDB | Prevent data loss on restart |

**Scaling Strategy**:
- Redis Cluster mode for 500K+ QPS
- Master-replica for high availability

### 2.2 Kafka (Async Processing)

| Metric | Target | Implementation |
|--------|--------|----------------|
| Throughput | ≥1M msg/s | 100 partitions, 3 replicas |
| Consumer | 10 consumer groups | Batch consumption (100/batch) |
| Latency | <10ms | Local SSD storage |
| DLQ | Enabled | 3 retries, then dead letter |

**Partition Strategy**:
- Hash by user_id to prevent hot spots
- 100 partitions for 100K+ QPS support

### 2.3 MySQL (Persistence)

| Metric | Target | Implementation |
|--------|--------|----------------|
| Write TPS | ≥2,000 | Batch inserts (100/batch) |
| Read QPS | ≥10,000 | Read replicas + Redis cache |
| Connection pool | 50-100 | HikariCP |

**Optimization**:
- Index on order_id, user_id, sku_id
- Partition orders by time

### 2.4 API Gateway (Higress/Envoy)

| Metric | Target | Implementation |
|--------|--------|----------------|
| Concurrent connections | ≥100,000 | Worker threads, connection pooling |
| QPS | ≥200,000 | WasmPlugin rate limiting |
| Latency | <50ms | Minimal hop count |

**Rate Limiting**:
- L1: 100 req/min per IP (WasmPlugin)
- L2: 10 req/min per user (Token bucket)

## 3. Traffic Model

### Peak Traffic Pattern

```
Time (minutes)    0    5    10   15   20   25   30
QPS               1K   10K  50K  100K 80K  40K  10K
Success Rate     100% 95%  80%  60%   70%  85%  95%
```

### Throughput Breakdown

| Layer | Input QPS | Output QPS | Reduction |
|-------|-----------|------------|-----------|
| L1 Gateway | 100,000 | 80,000 | 20% blocked |
| L2 Queue | 80,000 | 10,000 | 87.5% queued |
| L3 Inventory | 10,000 | 8,000 | 20% oversold |
| DB Writes | 8,000 | 2,000 | Batch 100x |

## 4. Cost Estimation

### Infrastructure (Monthly)

| Component | Instance | Qty | Cost/Month |
|-----------|----------|-----|------------|
| Redis | r6g.xlarge | 6 | $1,200 |
| Kafka | m6i.xlarge | 9 | $2,700 |
| MySQL | r6g.2xlarge | 6 | $1,800 |
| Gateway | c6i.xlarge | 4 | $800 |
| **Total** | | | **$6,500** |

## 5. Scaling Triggers

| Metric | Warning | Critical | Action |
|--------|---------|-----------|--------|
| Redis QPS | 40K | 45K | Scale horizontally |
| Kafka lag | 10K | 50K | Add consumers |
| DB connections | 80% | 95% | Connection pool tuning |
| Gateway QPS | 150K | 180K | Scale gateway pods |

## 6. Stress Test Plan

### Phase 1: Single Component

1. Redis: 50K QPS, 100 threads, 10M ops
2. Kafka: 100K msg/s, 100 partitions
3. MySQL: 2K TPS, batch inserts

### Phase 2: Integration

1. Gateway → Redis: 100K QPS
2. Redis → Kafka: 10K msg/s
3. Kafka → MySQL: 2K TPS

### Phase 3: Full System

- 100K concurrent users
- 30-minute sustained load
- Monitor: QPS, latency, error rate

## 7. Acceptance Criteria

| Metric | Target | P99 |
|--------|--------|-----|
| Inventory deduction | ≥50K QPS | <10ms |
| Gateway throughput | ≥200K QPS | <50ms |
| Kafka throughput | ≥1M msg/s | <5ms |
| DB writes | ≥2K TPS | <100ms |
| End-to-end latency | - | <200ms |
| Success rate | ≥95% | - |