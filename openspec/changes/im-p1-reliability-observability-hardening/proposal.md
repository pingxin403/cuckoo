# Change: im-p1-reliability-observability-hardening

## Why

在 P0 完成基础闭环后，系统将进入“可用但需增强”的阶段。当前风险集中在：
- 分布式场景下故障隔离与降级策略不足。
- 可观测性指标和追踪覆盖不足，问题定位成本高。
- 外部依赖调用缺少连接池/熔断/超时统一策略。

P1 目标是提升稳定性、可维护性和可运营性。

## What Changes

- 增强 Gateway/IM 对关键外部依赖（Auth/User/IM gRPC、Kafka、Registry）的连接管理与容错。
- 完善指标与链路追踪，覆盖消息收发、ACK、跨网关转发、失败分类。
- 增强错误分类与告警信号，减少“失败但不可观测”的盲区。
- 补齐关键路径集成测试与故障场景验证（超时、依赖抖动、部分失败）。

## Impact

- Affected specs: `im-chat-system`（新增）
- Affected code:
  - `apps/im-gateway-service/service/*`
  - `apps/im-service/service/*`
  - `apps/im-gateway-service/metrics/*`
  - `libs/observability/*`
- Breaking changes: 否（原则上兼容）
- Database migrations: 视离线状态增强需求决定（默认否）
- API changes: 可能新增可选字段（错误码/状态细分）
