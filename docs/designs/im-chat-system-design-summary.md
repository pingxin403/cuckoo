# IM 聊天系统设计摘要

**架构文档**: `docs/architecture/IM_CHAT_SYSTEM.md`

---

## 系统概述

IM 聊天系统是一个分布式即时通讯系统，支持单聊、群聊、离线消息、已读回执、消息去重等功能。

### 核心功能
- 单聊和群聊
- 实时消息推送（WebSocket）
- 离线消息存储和推送
- 已读回执
- 消息去重
- 内容过滤
- 多设备同步

---

## 架构设计

### 整体架构

```
┌──────────────┐
│   Clients    │
│ (Web/Mobile) │
└──────┬───────┘
       │ WebSocket
       ▼
┌─────────────────────────────────┐
│   IM Gateway Service            │
│  - WebSocket 连接管理           │
│  - 消息路由                     │
│  - 在线状态管理                 │
└──────────┬──────────────────────┘
           │ gRPC
           ▼
┌─────────────────────────────────┐
│   IM Service                    │
│  - 消息持久化                   │
│  - 离线消息管理                 │
│  - 已读回执                     │
│  - 消息去重                     │
│  - 内容过滤                     │
└──────────┬──────────────────────┘
           │
           ├─── MySQL (消息存储)
           ├─── Redis (在线状态、去重)
           └─── etcd (服务发现)
```

### 消息流程

#### 1. 在线消息流程
```
发送方 → Gateway → IM Service → 持久化 → IM Service → Gateway → 接收方
```

#### 2. 离线消息流程
```
发送方 → Gateway → IM Service → 持久化 → 离线队列
                                           ↓
接收方上线 ← Gateway ← IM Service ← 离线队列
```

---

## 核心组件

### 1. IM Gateway Service
**职责**: WebSocket 连接管理和消息路由

**关键特性**:
- WebSocket 长连接管理
- 心跳检测（30 秒）
- 连接注册和注销
- 消息路由到 IM Service
- 在线状态管理

**技术栈**:
- Go + gorilla/websocket
- gRPC 客户端
- Redis（在线状态）

### 2. IM Service
**职责**: 消息处理和持久化

**关键特性**:
- 消息持久化（MySQL）
- 离线消息管理
- 已读回执处理
- 消息去重（Redis）
- 内容过滤（敏感词）
- 消息序列号生成

**技术栈**:
- Go + gRPC
- MySQL（消息存储）
- Redis（去重、缓存）
- etcd（服务发现）

### 3. 消息去重 (Deduplication)
**目的**: 防止消息重复发送

**实现**:
- Redis SET 存储消息 ID
- TTL 5 分钟
- 幂等性保证

**代码位置**: `apps/im-service/dedup/`

### 4. 内容过滤 (Content Filter)
**目的**: 过滤敏感词和违规内容

**实现**:
- Aho-Corasick 算法
- 敏感词库（可配置）
- 替换策略（***）

**代码位置**: `apps/im-service/filter/`

### 5. 已读回执 (Read Receipt)
**目的**: 跟踪消息已读状态

**实现**:
- MySQL 存储已读记录
- 批量更新优化
- 已读状态推送

**代码位置**: `apps/im-service/readreceipt/`

---

## 数据模型

### 消息表 (messages)
```sql
CREATE TABLE messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    message_id VARCHAR(64) UNIQUE NOT NULL,
    conversation_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    receiver_id VARCHAR(64),
    content TEXT NOT NULL,
    message_type TINYINT NOT NULL,
    sequence_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_conversation_sequence (conversation_id, sequence_id),
    INDEX idx_receiver_created (receiver_id, created_at)
);
```

### 离线消息表 (offline_messages)
```sql
CREATE TABLE offline_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(64) NOT NULL,
    message_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_created (user_id, created_at)
);
```

### 已读回执表 (read_receipts)
```sql
CREATE TABLE read_receipts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    conversation_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    message_id VARCHAR(64) NOT NULL,
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_conversation_user_message (conversation_id, user_id, message_id)
);
```

---

## 性能优化

### 1. 连接管理
- 单个 Gateway 支持 10K+ 并发连接
- 心跳检测减少无效连接
- 连接池复用

### 2. 消息路由
- 基于 user_id 的路由
- 本地缓存在线状态
- 批量推送优化

### 3. 数据库优化
- 索引优化（conversation_id + sequence_id）
- 批量插入
- 读写分离

### 4. 缓存策略
- Redis 缓存在线状态
- 消息去重缓存（5 分钟）
- 离线消息队列

---

## 多区域支持

### 混合逻辑时钟 (HLC)
- 全局唯一时间戳
- 因果关系保证
- 冲突检测

### 跨区域同步
- 消息异步同步
- 冲突解决策略
- 最终一致性


---

## 监控指标

### 连接指标
- `im_gateway_connections_total` - 当前连接数
- `im_gateway_connection_duration_seconds` - 连接时长
- `im_gateway_heartbeat_timeout_total` - 心跳超时

### 消息指标
- `im_messages_sent_total` - 发送消息数
- `im_messages_received_total` - 接收消息数
- `im_offline_messages_total` - 离线消息数
- `im_message_latency_seconds` - 消息延迟

### 业务指标
- `im_read_receipts_total` - 已读回执数
- `im_duplicate_messages_total` - 重复消息数
- `im_filtered_messages_total` - 过滤消息数

---

## 安全特性

### 认证和授权
- JWT Token 认证
- 用户身份验证
- 权限检查

### 内容安全
- 敏感词过滤
- 消息审计日志
- 违规内容拦截

### 数据安全
- 消息加密传输（TLS）
- 数据库加密存储
- 访问控制

---

## 部署架构

### 单区域部署
```
Load Balancer
     │
     ├─── IM Gateway (3 replicas)
     │
     ├─── IM Service (3 replicas)
     │
     ├─── MySQL (Master-Slave)
     │
     ├─── Redis Cluster
     │
     └─── etcd Cluster
```

### 多区域部署
- 两地双活架构
- 跨区域消息同步
- 流量切换支持

**相关文档**: `deploy/docker/MULTI_REGION_DEPLOYMENT.md`

---

## 相关文档

- **架构文档**: `docs/architecture/IM_CHAT_SYSTEM.md`
- **Gateway 文档**: `apps/im-gateway-service/README.md`
- **Service 文档**: `apps/im-service/README.md`
- **部署指南**: `deploy/docker/README.md`

---

**最后更新**: 2026-02-03  
**维护者**: Backend Go Team
