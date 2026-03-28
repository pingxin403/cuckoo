# Tasks: im-p1-reliability-observability-hardening

## 1. Reliability Hardening

- [ ] **1.1** 为关键外部调用统一超时与重试策略
- [ ] **1.2** 为关键 gRPC 调用引入连接池/连接复用策略
- [ ] **1.3** 为依赖调用引入熔断与恢复策略
- [ ] **1.4** 补齐失败分类与标准错误码映射

## 2. Observability Hardening

- [ ] **2.1** 增加消息投递关键指标（成功率、延迟、超时、重试）
- [ ] **2.2** 增加跨网关路径指标（转发成功率、失败原因）
- [ ] **2.3** 增加 ACK 全链路指标（pending、timeout、late-ack）
- [ ] **2.4** 补齐关键 span 与 trace 属性

## 3. Quality & Validation

- [ ] **3.1** 增加依赖抖动/超时场景集成测试
- [ ] **3.2** 增加故障注入与恢复验证
- [ ] **3.3** 定义并验证核心告警规则（ACK timeout、kafka lag、cross-gateway failure）

## 4. Documentation

- [ ] **4.1** 更新运维与监控文档
- [ ] **4.2** 输出 P1 稳定性验收报告
