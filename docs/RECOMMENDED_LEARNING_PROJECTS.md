# 推荐学习/实践项目

基于项目现状和个人技术背景的学习路径规划。

## 项目现状总结

### 已完成的核心功能

- **基础架构**: Monorepo、Protobuf API、Docker/K8s 部署、CI/CD
- **业务服务**: IM 聊天系统(95%+)、URL 短链接、秒杀系统(100%)、认证/用户服务
- **可观测性**: OpenTelemetry、Prometheus、Grafana、Jaeger、Loki

---

## 🎯 精准推荐项目

### 1. 两地双活 / 多活架构 ⭐⭐⭐⭐⭐

**当前状态**: 单集群部署

**可实践内容**:
- 跨地域 IM 消息同步
- 全局事务 ID + LWW 冲突解决
- 流量切换与故障演练自动化
- CRDTs (Conflict-free Replicated Data Types) 探索
- 基于 Kafka MirrorMaker 2 的跨集群复制

**价值**: 高级架构师核心能力，面试加分项

---

### 2. 分布式事务 - Saga 编排器 ⭐⭐⭐⭐⭐

**当前状态**: 秒杀系统用 Kafka 异步补偿

**可实践内容**:
- 实现 Saga 编排器（Orchestrator 模式）
- 订单→库存→支付 跨服务事务
- 补偿事务状态机 + 可视化
- 对比 TCC vs Saga vs 本地消息表

**价值**: 支付幂等经验延伸，面试高频考点

---

### 3. Kubernetes Operator 开发 ⭐⭐⭐⭐

**可实践内容**:
- 开发 IM-Service Operator（自动扩缩容）
- 基于 Kubebuilder/Operator SDK
- 实现 CRD: IMCluster, IMGateway
- 自动化运维：故障自愈、配置热更新
- 与 Flagger 集成实现智能发布

**价值**: CKA + Operator 开发 = 云原生专家，稀缺能力

---

### 4. eBPF 可观测性增强 ⭐⭐⭐⭐

**可实践内容**:
- 基于 eBPF 的无侵入式追踪（Pixie/Cilium Hubble）
- 网络层延迟分析（TCP 重传、连接建立时间）
- 系统调用级别的性能分析
- 与现有 OTel 体系集成

**价值**: 可观测性前沿方向

---

### 5. AI Agent 工程化 ⭐⭐⭐⭐

**可实践内容**:
- IM 智能客服 Agent（复用 RAG 经验）
- 代码审查 Agent（集成到 CI/CD）
- 运维 Agent（故障诊断、自动修复）
- Multi-Agent 协作（Orchestrator-Experts）
- Agent 评估体系（准确率、延迟、成本）

**价值**: AI + 后端趋势方向

---

### 6. 性能工程体系化 ⭐⭐⭐

**可实践内容**:
- 全链路压测平台（流量录制回放）
- 性能基线管理（自动检测性能回退）
- 火焰图自动分析（pprof + 自动化报告）
- 容量规划模型（基于历史数据预测）
- 性能 SLO 体系（P99 预算管理）

---

## 📊 优先级排序

| 优先级 | 项目 | 原因 |
|--------|------|------|
| 🥇 | 两地双活深化 | 核心竞争力，可复现并深化 |
| 🥇 | Saga 编排器 | 支付经验延伸，面试高频 |
| 🥈 | Operator 开发 | CKA 加持，差异化能力 |
| 🥈 | AI Agent 工程化 | Langgraph 经验延伸，趋势方向 |
| 🥉 | eBPF 可观测性 | OTel 经验延伸，前沿技术 |
| 🥉 | 性能工程体系 | 系统化优化经验 |

---

## 🚀 推荐学习路径

### 第一阶段（2周）：两地双活架构
- 设计跨地域同步方案
- 实现冲突检测与解决
- 故障切换演练
- 文档化经验

### 第二阶段（2周）：Saga 编排器
- 设计状态机
- 实现订单-库存-支付流程
- 补偿事务测试
- 可视化面板

### 第三阶段（2周）：Operator 开发
- 学习 Kubebuilder
- 实现 IMCluster CRD
- 自动扩缩容逻辑
- 与 Flagger 集成

---

## 相关 Spec 目录

