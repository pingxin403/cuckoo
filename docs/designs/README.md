# 设计文档索引


---

## 📋 文档说明

- **设计摘要**: 本目录下的文档是各功能模块设计的简要说明，便于快速了解系统架构和设计决策
- **实现代码**: 各服务的实现代码位于 `apps/` 和 `libs/` 目录

---

## 🏗️ 核心系统设计

### 1. IM 聊天系统 (IM Chat System)
**简要说明**: 分布式即时通讯系统，支持单聊、群聊、离线消息、已读回执等功能

**关键特性**:
- WebSocket 长连接网关
- 消息路由和持久化
- 离线消息推送
- 已读回执
- 消息去重和过滤

**相关文档**:
- 架构文档: `docs/architecture/IM_CHAT_SYSTEM.md`
- 服务文档: `apps/im-service/README.md`, `apps/im-gateway-service/README.md`

---

### 2. URL 短链服务 (URL Shortener Service)
**简要说明**: 高性能 URL 短链生成和重定向服务

**关键特性**:
- 短链生成和自定义
- 高性能缓存（L1 + L2）
- 缓存保护机制（穿透、击穿、雪崩、延时双删）
- 点击统计和分析
- Redis 集群支持

**相关文档**:
- 架构文档: `docs/architecture/URL_SHORTENER_SERVICE.md`
- 服务文档: `apps/shortener-service/README.md`
- Redis 优化: `apps/shortener-service/docs/REDIS_OPTIMIZATION_COMPLETE.md`
- 缓存保护: `apps/shortener-service/docs/CACHE_PROTECTION_SUMMARY.md`

---

### 3. 秒杀系统 (Flash Sale System)
**简要说明**: 高并发秒杀系统，支持库存管理、订单处理、防刷机制

**关键特性**:
- 高并发库存扣减
- 分布式锁
- 消息队列异步处理
- 防刷和风控
- 数据一致性保证

**相关文档**:
- 服务文档: `apps/flash-sale-service/README.md`

---

### 4. 多区域双活架构 (Multi-Region Active-Active)
**简要说明**: 两地双活架构，支持跨区域消息同步和故障切换

**关键特性**:
- 混合逻辑时钟 (HLC)
- 跨区域消息同步
- 冲突检测和解决
- 流量切换
- 区域故障恢复

**相关文档**:
- 部署文档: `deploy/docker/MULTI_REGION_DEPLOYMENT.md`
- 演示文档: `docs/multi-region-demo/README.md`

---

## 🔧 基础设施和工具

### 5. 可观测性集成 (Observability Integration)
**简要说明**: 统一的可观测性框架，集成日志、指标、追踪

**关键特性**:
- OpenTelemetry 集成
- Prometheus 指标
- Jaeger 分布式追踪
- 结构化日志
- 线程安全保证

**相关文档**:
- 库文档: `libs/observability/README.md`
- 架构文档: `docs/architecture/OBSERVABILITY_SYSTEM.md`

---

### 6. Redis 优化 (Redis Optimization)
**简要说明**: Redis 性能优化和最佳实践

**关键特性**:
- 连接池优化
- Pipeline 批量操作
- Lua 脚本原子操作
- 集群支持
- 缓存保护机制

**相关文档**:
- 实现文档: `apps/shortener-service/docs/REDIS_OPTIMIZATION_COMPLETE.md`

---

### 7. 健康检查标准化 (Health Check Standardization)
**简要说明**: 统一的健康检查框架和标准

**关键特性**:
- 统一健康检查接口
- 依赖检查（数据库、Redis、Kafka）
- 优雅启动和关闭
- Kubernetes 就绪探针

**相关文档**:
- 库文档: `libs/health/README.md`

---

### 8. 中心化 Proto 生成 (Centralized Proto Generation)
**简要说明**: 统一的 Protocol Buffers 代码生成流程

**关键特性**:
- 中心化 proto 定义
- 多语言代码生成（Go, Java, TypeScript）
- 版本管理
- CI/CD 集成

**相关文档**:
- 架构文档: `docs/architecture/PROTO_GENERATION.md`

---

## 📚 如何使用这些文档

### 快速了解系统
1. 阅读本文档的简要说明
2. 查看 `docs/architecture/` 下的架构文档
3. 查看各服务的 README.md

### 深入了解设计

### 参与开发
2. 阅读相关服务的 `TESTING.md` 了解测试要求
3. 参考 `docs/development/` 下的开发指南

---

## 🔗 相关文档

- [项目架构](../architecture/ARCHITECTURE.md)
- [开发指南](../development/)
- [部署指南](../deployment/)
- [运维指南](../operations/)

---

**最后更新**: 2026-02-03  
**维护者**: Backend Team
