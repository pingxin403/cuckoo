# Change: implement-im-p0-delivery-security-closure

## Why

当前 IM Service 与 IM Gateway 已具备基础能力，但关键链路仍存在未闭环项：
- IM Service 群聊发布与消息状态查询仍是 TODO。
- IM Gateway 的启动 wiring、跨网关投递、ACK 处理与 Origin 校验未完全落地。
- 多节点场景下存在“服务可启动但消息不可达/不一致”的风险。

为保障“可上线的最小完整闭环”，需要先完成 P0：消息投递闭环 + 基础安全链路。

## What Changes

### 1) IM Service 功能闭环
- 完成群聊消息发布到 `group_msg` 的实现。
- 完成消息状态查询（`GetMessageStatus`）最小可用实现。
- 补齐关键路径结构化日志（路由结果、降级、失败原因）。

### 2) IM Gateway 运行与投递闭环
- 完成 `main.go` 生产级 wiring（auth/registry/im client、gateway 启动、Kafka 配置接入）。
- 完成跨网关消息投递与跨网关读回执投递。
- 完成 ACK 处理闭环（接收、关联、超时与状态回传）。

### 3) 安全基线补齐（P0 范围）
- 实现 WebSocket Origin 白名单校验（可配置）。
- 默认拒绝非法 Origin，支持灰度开关与可观测日志。

### 4) 文档与契约同步
- 更新 IM / Gateway README，使其与代码能力一致。
- 明确多节点与跨网关行为约束，避免“文档已支持、代码未支持”偏差。

## Impact

- Affected specs: `im-chat-system`（新增）
- Affected code:
  - `apps/im-service/service/im_service.go`
  - `apps/im-gateway-service/main.go`
  - `apps/im-gateway-service/service/gateway_service.go`
  - `apps/im-gateway-service/service/push_service.go`
  - `apps/im-gateway-service/service/kafka_consumer.go`
- Breaking changes: 否（默认向后兼容）
- Database migrations: 否（P0 不引入新表，仅允许复用现有结构）
- API changes: 可能包含兼容性增强（状态字段补齐，不删除既有字段）
