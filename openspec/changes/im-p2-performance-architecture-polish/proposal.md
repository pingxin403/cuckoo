# Change: im-p2-performance-architecture-polish

## Why

在完成 P0/P1 后，IM 系统将具备可上线与可运维能力。P2 目标是进一步提升性能上限、降低复杂度，并收敛技术债，确保后续演进成本可控。

## What Changes

- 面向高并发场景进行性能优化（热点群、跨网关高扇出、ACK 高并发）。
- 收敛历史 TODO/重复逻辑，推进模块边界清晰化。
- 提升自动化验证能力（压测、混沌、基准对比、容量模型）。
- 明确架构演进路径（保守增强 vs 渐进重构）。

## Impact

- Affected specs: `im-chat-system`（新增）
- Affected code: `apps/im-service/*`, `apps/im-gateway-service/*`, `load_test/*`, `docs/operations/*`
- Breaking changes: 原则上避免，若涉及将单独标记
- Database migrations: 可能（仅在性能收益明确且可回滚时）
- API changes: 仅兼容增强
