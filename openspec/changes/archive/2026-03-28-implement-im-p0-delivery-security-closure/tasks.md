# Tasks: implement-im-p0-delivery-security-closure

## 1. IM Service 闭环

- [x] **1.1** 在 `apps/im-service/service/im_service.go` 实现群聊消息 Kafka 发布逻辑（`group_msg`）
- [x] **1.2** 在 `apps/im-service/service/im_service.go` 实现 `GetMessageStatus` 最小可用状态查询
- [x] **1.3** 为群聊路由与状态查询补齐结构化日志（成功、降级、失败）
- [x] **1.4** 为上述能力新增/更新单元测试

## 2. IM Gateway 运行与跨网关闭环

- [x] **2.1** 在 `apps/im-gateway-service/main.go` 完成 auth/registry/im client 初始化与注入
- [x] **2.2** 在 `apps/im-gateway-service/main.go` 补齐 gateway 启动与 Kafka 配置接线
- [x] **2.3** 在 `apps/im-gateway-service/service/push_service.go` 实现跨网关消息投递
- [x] **2.4** 在 `apps/im-gateway-service/service/push_service.go` 实现跨网关读回执投递
- [x] **2.5** 在 `apps/im-gateway-service/service/gateway_service.go` 完成 ACK 接收、关联、超时处理
- [x] **2.6** 在 `apps/im-gateway-service/service/kafka_consumer.go` 实现离线读回执最小持久化路径
- [x] **2.7** 为跨网关与 ACK 场景新增/更新单元测试

## 3. 安全基线（Origin）

- [x] **3.1** 在 `apps/im-gateway-service/service/gateway_service.go` 实现 Origin 白名单校验
- [x] **3.2** 在配置中新增/确认 Origin 校验相关项（允许列表、空 Origin 策略）
- [x] **3.3** 为 Origin 校验新增单元测试（允许、拒绝、灰度配置）

## 4. 集成验证

- [x] **4.1** 补充双网关拓扑下的跨节点消息投递集成测试
- [x] **4.2** 补充双网关拓扑下的跨节点读回执集成测试
- [x] **4.3** 验证 ACK 成功率与超时路径行为符合预期
- [x] **4.4** 回归离线消息路径，确保与现有行为兼容

## 5. 文档与交付

- [x] **5.1** 更新 `apps/im-service/README.md` 中与实际能力不一致的描述
- [x] **5.2** 更新 `apps/im-gateway-service/README.md`，同步 P0 完成项与剩余项
- [x] **5.3** 输出 P0 验证报告（功能覆盖、风险、回滚点）
