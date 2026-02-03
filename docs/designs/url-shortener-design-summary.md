# URL 短链服务设计摘要

**服务文档**: `apps/shortener-service/README.md`

---

## 系统概述

URL 短链服务是一个高性能的短链生成和重定向服务，支持自定义短链、点击统计、过期管理等功能。

### 核心功能
- 短链生成（自动生成或自定义）
- URL 重定向
- 点击统计
- 过期管理
- 批量操作

---

## 架构设计

### 整体架构

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│      gRPC/HTTP Gateway          │
│  (Envoy/Higress)                │
└──────────┬──────────────────────┘
           │
           ▼
┌──────────────────────────────────┐
│   Shortener Service              │
│  ┌────────────────────────────┐  │
│  │  L1 Cache (Ristretto)      │  │
│  │  - 10MB in-memory          │  │
│  │  - LRU eviction            │  │
│  └────────────┬───────────────┘  │
│               │                  │
│  ┌────────────▼───────────────┐  │
│  │  L2 Cache (Redis)          │  │
│  │  - 7 days TTL (±1 day)     │  │
│  │  - Cluster support         │  │
│  └────────────┬───────────────┘  │
│               │                  │
│  ┌────────────▼───────────────┐  │
│  │  Storage (MySQL)           │  │
│  │  - Persistent storage      │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

### 缓存策略

#### 两级缓存 (L1 + L2)
1. **L1 Cache (Ristretto)**
   - 内存缓存，10MB
   - LRU 淘汰策略
   - 毫秒级响应

2. **L2 Cache (Redis)**
   - 分布式缓存
   - 7 天 TTL（±1 天抖动）
   - 支持集群模式

#### 缓存保护机制

1. **缓存穿透防护（空值缓存）**
   - 使用 `__EMPTY__` 标记不存在的数据
   - TTL 5 分钟
   - 减少 50-80% 无效数据库查询

2. **缓存击穿防护（SETNX + Singleflight）**
   - SETNX 锁协调并发请求
   - 指数退避重试机制
   - 99.2% 数据库负载减少

3. **缓存雪崩防护（TTL Jitter）**
   - TTL 抖动范围 6-8 天
   - 使用 crypto/rand 生成随机数
   - 防止大量缓存同时过期

4. **延时双删（Cache Consistency）**
   - 双删策略保证最终一致性
   - 延迟 500ms 异步执行
   - 防止读取到旧数据

---

## 核心组件

### 1. ID 生成器 (IDGenerator)
- **算法**: Base62 编码
- **长度**: 6-8 字符
- **冲突处理**: 重试机制（最多 3 次）
- **自定义支持**: 验证和保留自定义短码

### 2. 缓存管理器 (CacheManager)
- **L1 Cache**: Ristretto（内存）
- **L2 Cache**: Redis（分布式）
- **缓存加载器**: SETNX 防止缓存击穿
- **批量操作**: Pipeline 优化

### 3. 存储层 (Storage)
- **数据库**: MySQL
- **表结构**: url_mappings
- **索引**: short_code (唯一索引)
- **软删除**: deleted_at 字段

### 4. 缓存一致性 (CacheConsistency)
- **延时双删**: 500ms 延迟
- **异步执行**: Goroutine
- **错误处理**: 优雅降级

---

## 性能优化

### Redis 优化
1. **连接池优化**
   - PoolSize: 100
   - MinIdleConns: 10
   - MaxConnAge: 30 分钟

2. **Pipeline 批量操作**
   - 批量大小: 1000
   - 延迟降低: 80%

3. **Lua 脚本**
   - 原子操作
   - 减少网络往返

4. **集群支持**
   - Hash Tag 支持
   - 跨槽操作优化

### 性能指标
- **QPS**: 50K-100K（取决于硬件）
- **P99 延迟**: < 10ms（缓存命中）
- **缓存命中率**: > 95%（热点数据）
- **错误率**: < 1%（排除 Rate Limiter）

---

## 监控指标

### 缓存指标
- `redis_empty_cache_set_total` - 空值缓存设置
- `redis_empty_cache_hits_total` - 空值缓存命中
- `redis_setnx_lock_acquired_total` - SETNX 锁获取
- `redis_setnx_lock_contention_total` - SETNX 锁竞争
- `redis_ttl_seconds` - TTL 分布

### 业务指标
- `shortener_links_created_total` - 创建的短链数
- `shortener_links_deleted_total` - 删除的短链数
- `shortener_operation_duration_seconds` - 操作延迟
- `shortener_errors_total` - 错误计数

---

## 安全特性

### URL 验证
- **协议检查**: 只允许 http/https
- **长度限制**: 最大 2048 字符
- **恶意模式检测**: SQL 注入、XSS 等

### Rate Limiting
- **限流策略**: 每 IP 每分钟 100 请求（可配置）
- **算法**: Token Bucket
- **响应**: 429 Too Many Requests

### 审计日志
- **创建日志**: 记录创建者 IP
- **访问日志**: 记录访问来源
- **删除日志**: 记录删除操作

---

## 部署架构

### 单区域部署
```
Load Balancer
     │
     ├─── Shortener Service (3 replicas)
     │
     ├─── Redis Cluster (3 master + 3 replica)
     │
     └─── MySQL (Master-Slave)
```

### 多区域部署
- 支持跨区域部署
- 数据库主从复制
- Redis 集群跨区域同步

---

## 相关文档

- **服务文档**: `apps/shortener-service/README.md`
- **Redis 优化**: `apps/shortener-service/docs/REDIS_OPTIMIZATION_COMPLETE.md`
- **缓存保护**: `apps/shortener-service/docs/CACHE_PROTECTION_SUMMARY.md`
- **性能分析**: `apps/shortener-service/docs/PERFORMANCE_ANALYSIS.md`
- **负载测试**: `apps/shortener-service/load_test/README.md`

---

**最后更新**: 2026-02-03  
**维护者**: Backend Go Team
