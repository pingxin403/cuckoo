# Design: implement-im-p0-delivery-security-closure

## Context

当前系统的核心风险不是“功能完全缺失”，而是“关键链路未闭环”：
- IM Service 在群聊发布、状态查询等核心路径存在 TODO。
- IM Gateway 在多节点场景的跨网关投递与 ACK 闭环不完整。
- WebSocket Origin 校验仍为占位实现，存在安全基线风险。

目标是在不引入大规模重构的前提下，完成上线必需闭环。

## Goals / Non-Goals

### Goals
- 完成 IM Service 群聊发布与状态查询最小闭环。
- 完成 IM Gateway 启动 wiring、跨网关消息与读回执投递、ACK 闭环。
- 完成 Origin 白名单校验，形成可配置安全基线。
- 保持对现有接口和部署方式的兼容。

### Non-Goals
- 不引入新的消息中间件或替换现有 Kafka 架构。
- 不进行大规模架构迁移（如 etcd → Redis Cluster）。
- 不在 P0 阶段引入复杂性能优化（Bloom Filter、深度背压策略等）。

## Decisions

### D1: 先闭环，再优化
优先补齐生产关键链路，避免在 P0 阶段同时进行架构级改造，降低交付风险。

### D2: 复用现有协议与模型
在现有 gRPC/WebSocket 消息结构内补齐字段与状态，不做破坏式变更。

### D3: 安全默认拒绝
Origin 校验采用可配置白名单，默认拒绝非法来源，并通过配置支持灰度上线。

## Technical Approach

### 1) IM Service
- 在 `RouteGroupMessage` 中接入 Kafka `group_msg` 发布逻辑。
- 在 `GetMessageStatus` 中实现状态读取（优先现有存储与状态映射）。
- 增加结构化日志：msg_id、conversation_id、path(fast/slow)、error_code。

### 2) IM Gateway
- 在 `main.go` 完成依赖初始化并统一注入 `GatewayService`/`PushService`。
- 在 `PushService` 实现跨网关转发：
  - 本地在线设备：直接 WebSocket 推送
  - 远端设备：通过 gateway 间 gRPC 转发
- 在 `gateway_service.go` 完成 ACK 关联与超时管理。
- 在 `kafka_consumer.go` 完成读回执离线落存的最小实现。

### 3) Origin 校验
- 在 WebSocket Upgrade 前执行 Origin 规则判断。
- 配置项支持：允许列表、是否允许空 Origin（兼容原生客户端）。
- 记录拒绝原因与来源，支持审计与排障。

## Risks / Trade-offs

- 跨网关转发增加网络跳数，P99 可能小幅上升；但能换取多节点正确性。
- ACK 闭环补齐后可能暴露历史“伪成功”路径，短期会看到更多真实失败指标。
- Origin 默认拒绝可能影响历史弱校验客户端，需要灰度与白名单回填。

## Validation Plan

- 单测：路由、ACK、Origin、跨网关分支。
- 集成：双网关拓扑下私聊/群聊/读回执跨节点验证。
- 回归：离线消息、重连同步、已有 API 契约。

## Rollout Plan

1. 先在本地与集成环境启用完整链路。
2. staging 灰度启用 Origin 严格校验与跨网关路径。
3. 观察 ACK 超时率、投递成功率、跨节点延迟后再全量。


---

## Revisions

| 日期 | 类型 | 变更描述 | 原因 | 影响 API |
|------|------|----------|------|----------|
| 2026-03-28 | behavior | Cross-gateway read-receipt remote forwarding remains unsupported in current protocol; remote forwarder returns explicit not-supported error while local persistence/fallback path is implemented. | Current im-gateway gRPC contract only exposes PushMessage RPC and lacks a dedicated remote read-receipt forwarding RPC. | im-gateway remote forwarding |
