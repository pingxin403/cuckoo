# Design: im-p1-reliability-observability-hardening

## Context

P0 解决“能闭环”，P1 解决“跑得稳、看得见、能定位”。系统进入多节点和真实流量后，核心问题会转向：
- 外部依赖抖动导致级联超时。
- 指标/追踪缺失导致排障效率低。
- 失败分类不清，难以定义有效 SLO。

## Goals / Non-Goals

### Goals
- 建立统一的 timeout/retry/circuit-breaker 策略。
- 补齐关键路径 metrics + tracing + structured logs。
- 提供可执行的告警信号（ACK timeout、cross-gateway fail、kafka lag）。

### Non-Goals
- 不进行协议重构。
- 不引入新的主消息链路基础设施。

## Decisions

- 采用“有限重试 + 熔断 + 指标驱动降级”策略，而非无限重试。
- 统一错误码与失败标签，保障日志/指标可聚合分析。
- 先覆盖最关键链路，再扩展到边缘路径。

## Validation Plan

- 增加故障注入测试（依赖超时、远端网关不可达、Kafka 暂时不可用）。
- 验证观测面：每个失败类均有 metric/log/trace 证据。
