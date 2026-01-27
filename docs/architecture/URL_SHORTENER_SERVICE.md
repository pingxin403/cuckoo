# URL Shortener Service 架构设计

## 概述

URL Shortener Service 是一个高性能、分布式的 Go 微服务，将长 URL 转换为短链接，并提供亚 10ms 的重定向服务。服务围绕多层缓存架构设计，实现 500,000+ QPS 读取吞吐量，同时通过 MySQL 持久化保证数据持久性。

## 系统架构

### 高层架构图

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────┐
│         API Gateway (Higress)           │
│  - gRPC-Web conversion                  │
│  - Rate limiting (100 req/min per IP)   │
│  - TLS termination                      │
└──────┬──────────────────────────────────┘
       │
       ├─────────────────┬─────────────────┐
       │                 │                 │
       ▼                 ▼                 ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│  Shortener  │   │  Shortener  │   │  Shortener  │
│  Instance 1 │   │  Instance 2 │   │  Instance N │
└──────┬──────┘   └──────┬──────┘   └──────┬──────┘
       │                 │                 │
       └─────────────────┴─────────────────┘
                         │
       ┌─────────────────┼─────────────────┐
       │                 │                 │
       ▼                 ▼                 ▼
┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│ L1: Ristretto│   │ L2: Redis   │   │ L3: MySQL   │
│  (In-Memory) │   │   Cluster   │   │  (Primary)  │
│  TTL: 1h     │   │  TTL: 7d    │   │  Permanent  │
└─────────────┘   └─────────────┘   └──────┬──────┘
                                            │
                                            ▼
                                    ┌─────────────┐
                                    │   MySQL     │
                                    │  (Replica)  │
                                    └─────────────┘
```

## 请求流程

### 创建短链接 (写路径)

```
1. Client → API Gateway → Shortener Service
2. 验证输入 (URL 格式、长度、协议)
3. 生成 7 字符短码 (ID Generator)
4. 写入 MySQL (ACID 事务)
5. 写入 Redis (缓存预热)
6. 返回短 URL 给客户端
7. 异步: 记录创建事件到 Kafka
```

### 重定向 (读路径 - 热数据)

```
1. Client → API Gateway → Shortener Service
2. 检查 L1 缓存 (Ristretto) → HIT (0.2ms)
3. 返回 302 重定向
4. 异步: 增加点击计数，记录到 Kafka
```

### 重定向 (读路径 - 冷数据)

```
1. Client → API Gateway → Shortener Service
2. 检查 L1 缓存 → MISS
3. 检查 L2 缓存 (Redis) → HIT (2ms)
4. 回填 L1 缓存
5. 返回 302 重定向
6. 异步: 增加点击计数，记录到 Kafka
```

### 重定向 (读路径 - 缓存未命中)

```
1. Client → API Gateway → Shortener Service
2. 检查 L1 缓存 → MISS
3. 检查 L2 缓存 (Redis) → MISS
4. Singleflight: 合并并发请求
5. 查询 MySQL → HIT (8ms)
6. 回填 L2 和 L1 缓存
7. 返回 302 重定向
8. 异步: 增加点击计数，记录到 Kafka
```

## 核心组件

### 1. ID Generator

**职责**: 生成唯一、不可预测的 7 字符短码

**算法**: 加密随机生成 + 碰撞检测

```go
type IDGenerator interface {
    Generate(ctx context.Context) (string, error)
    ValidateCustomCode(ctx context.Context, code string) error
}
```

**设计决策**:
- **为什么用 crypto/rand?** 确保不可预测性，防止枚举攻击
- **为什么 7 字符?** 提供 3.5 万亿组合 (62^7)，足够数十亿链接
- **为什么碰撞重试?** 碰撞概率极低 (<0.001%)，简单重试可接受
- **动态扩展**: 监控碰撞率；如果 >0.1%，自动扩展到 8 字符

### 2. 多层缓存架构

| 层级 | 技术 | TTL | 延迟 | 用途 |
|------|------|-----|------|------|
| L1 | Ristretto (内存) | 1h | 0.2ms | 热点数据本地缓存 |
| L2 | Redis Cluster | 7d | 2ms | 分布式共享缓存 |
| L3 | MySQL | 永久 | 8ms | 持久化存储 |

**缓存防护机制**:
- **Singleflight**: 合并并发请求，防止缓存击穿
- **TTL Jitter**: 随机抖动防止缓存雪崩
- **优雅降级**: Redis 故障时直接查询 MySQL

### 3. 存储层

**MySQL Schema**:
```sql
CREATE TABLE url_mappings (
    short_code VARCHAR(10) PRIMARY KEY,
    long_url TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    expires_at BIGINT NULL,
    creator_ip VARCHAR(45) NOT NULL,
    click_count BIGINT DEFAULT 0,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    INDEX idx_expires (expires_at),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 4. Analytics Writer

**职责**: 异步记录点击事件到 Kafka

```go
type ClickEvent struct {
    ShortCode string
    Timestamp time.Time
    SourceIP  string
    UserAgent string
}
```

**特点**:
- 非阻塞设计，不影响重定向延迟
- 缓冲区满时丢弃事件（非关键路径）
- 多 worker 并行写入

## 性能指标

| 指标 | 目标 |
|------|------|
| 读取 QPS | 500,000+ |
| 重定向 P99 延迟 | < 10ms |
| 创建 P99 延迟 | < 50ms |
| L1 缓存命中率 | > 90% |
| L2 缓存命中率 | > 99% |

## 错误处理

### 客户端错误 (4xx)
- `400 Bad Request`: 无效 URL 格式
- `409 Conflict`: 自定义短码已存在
- `410 Gone`: 短链接已过期
- `429 Too Many Requests`: 超出速率限制

### 服务端错误 (5xx)
- `500 Internal Server Error`: 意外错误
- `503 Service Unavailable`: MySQL 不可用

### 优雅降级策略

| 故障场景 | 处理方式 |
|----------|----------|
| Redis 故障 | 绕过 L2 缓存，直接查询 MySQL |
| MySQL 故障 | 写操作返回 503，读操作从缓存服务 |
| Kafka 故障 | 丢弃分析事件，继续正常重定向 |

## 相关文档

- 详细设计: `.kiro/specs/url-shortener-service/design.md`
- API 定义: `api/v1/shortener.proto`
- 服务代码: `apps/shortener-service/`
