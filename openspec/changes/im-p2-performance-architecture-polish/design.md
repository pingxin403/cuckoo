# Design: im-p2-performance-architecture-polish

## Context

P2 聚焦“更快、更稳、更易演进”。经过 P0/P1 后，系统具备功能与稳定性基础，下一阶段需要通过性能工程和代码治理把系统带到可持续扩展状态。

## Goals / Non-Goals

### Goals
- 对关键链路建立容量模型与性能基线。
- 优化热点路径（群消息广播、跨网关转发、ACK 状态管理）。
- 降低模块耦合与重复逻辑，提升可测试性。

### Non-Goals
- 不做无数据支持的大规模重写。
- 不在缺乏压测证据时引入复杂新机制。

## Decisions

- 采用“基准→优化→回归基准”闭环，所有优化均需量化收益。
- 优先局部优化和模块收敛，避免一次性大改。
- 以 staging 压测结果作为进入生产优化的门禁。

## Module Boundary Notes (IM Gateway)

为收敛 `gateway_service / push_service / forwarder` 的职责边界，本阶段约定如下：

- **gateway_service**
  - 负责连接生命周期、ACK 状态、配置和服务装配。
  - 仅暴露必要入口（如 `PushMessage` / `PushReadReceipt`）给 gRPC 层与外部调用。

- **push_service**
  - 负责投递流程编排：本地连接投递、远端转发、失败分类、指标与 tracing 打点。
  - 通过公共流程函数收敛重复逻辑，避免 message/read-receipt 路径在业务流程上分叉失真。

- **remote_forwarder**
  - 负责跨网关传输细节：gRPC 连接池、超时、重试、熔断。
  - 不承担业务规则决策，返回可被 `push_service` 统一解释的传输结果。

本次在 `push_service` 内新增公共远端转发路径以减少重复代码，并补充“指定设备远端转发”回归测试，作为边界收敛的第一步。

## Validation Plan

- 设定基线指标：吞吐、P95/P99 延迟、超时率、错误率、资源占用。
- 引入热点群与多节点场景压测。
- 通过混沌测试验证优化后的鲁棒性不回退。
