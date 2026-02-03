# Multi-Region Active-Active Operations Guide

## 概述

本目录包含多地域双活系统的运维文档，涵盖故障排查、容量规划、性能调优和监控告警等方面。

## 文档导航

### 核心运维文档

| 文档 | 描述 | 使用场景 |
|------|------|----------|
| [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | 故障排查手册 | 系统出现问题时的诊断和解决 |
| [CAPACITY_PLANNING.md](./CAPACITY_PLANNING.md) | 容量规划指南 | 规划系统容量和成本优化 |
| [PERFORMANCE_TUNING.md](./PERFORMANCE_TUNING.md) | 性能调优指南 | 优化系统性能和资源利用 |
| [MONITORING_ALERTING.md](./MONITORING_ALERTING.md) | 监控告警手册 | 配置监控指标和告警规则 |

## 快速参考

### 常见场景

#### 1. 系统健康检查

```bash
# 检查所有服务状态
docker ps | grep -E "im-service|im-gateway"

# 检查健康端点
curl http://localhost:8080/health  # Region A
curl http://localhost:8081/health  # Region B

# 查看 Grafana 面板
open http://localhost:3000/d/multi-region-overview
```

#### 2. 故障转移测试

```bash
# 停止 Region A
docker stop im-service-region-a im-gateway-region-a

# 等待自动故障转移（< 30秒）
# 验证流量切换到 Region B
curl http://im-gateway.example.com/stats | jq '.region'

# 恢复 Region A
docker start im-service-region-a im-gateway-region-a
```

#### 3. 性能监控

```bash
# 查看跨地域同步延迟
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile\(0.99,cross_region_sync_latency_ms\)

# 查看冲突率
curl -s http://localhost:9090/api/v1/query?query=rate\(cross_region_conflicts_total\[5m\]\)

# 查看活跃连接数
curl -s http://localhost:9090/api/v1/query?query=sum\(active_websocket_connections\)by\(region\)
```

#### 4. 流量切换

```bash
# 渐进式切换（50% 流量到 Region B）
cd apps/im-service/cmd/traffic-cli
go run main.go switch --from region-a --to region-b --ratio 50

# 全量切换
go run main.go switch --from region-a --to region-b --ratio 100

# 回滚
go run main.go switch --from region-b --to region-a --ratio 100
```

### 关键指标

| 指标 | 正常范围 | 告警阈值 | 严重阈值 |
|------|----------|----------|----------|
| **跨地域同步延迟 (P99)** | < 300ms | > 500ms | > 1000ms |
| **冲突率** | < 0.05% | > 0.1% | > 0.5% |
| **CPU 使用率** | 60-70% | > 80% | > 90% |
| **内存使用率** | 60-70% | > 85% | > 95% |
| **故障转移时间 (RTO)** | < 25s | > 30s | > 60s |
| **数据丢失 (RPO)** | < 0.5s | > 1s | > 5s |

### 告警级别

| 级别 | 响应时间 | 影响 | 示例 |
|------|----------|------|------|
| **P0 - 严重** | < 5 分钟 | 完全服务中断 | 地域宕机、数据丢失 |
| **P1 - 高** | < 15 分钟 | 显著性能下降 | 高同步延迟、故障转移失败 |
| **P2 - 中** | < 1 小时 | 轻微性能下降 | 高冲突率、缓存问题 |
| **P3 - 低** | < 4 小时 | 无用户影响 | 监控缺失、优化建议 |

## 运维工作流

### 日常运维

#### 每日检查清单

- [ ] 检查系统健康状态
- [ ] 查看告警历史
- [ ] 检查关键指标趋势
- [ ] 查看错误日志
- [ ] 验证备份完成

#### 每周检查清单

- [ ] 审查容量使用情况
- [ ] 分析性能趋势
- [ ] 检查冲突率变化
- [ ] 更新运维文档
- [ ] 团队知识分享

#### 每月检查清单

- [ ] 容量规划评审
- [ ] 成本优化分析
- [ ] 灾难恢复演练
- [ ] 安全审计
- [ ] 文档更新

### 故障响应流程

```
1. 告警触发
   ↓
2. 确认告警（< 2 分钟）
   ↓
3. 初步诊断（< 5 分钟）
   ↓
4. 升级决策
   ├─ 可自行解决 → 执行修复
   └─ 需要升级 → 联系高级工程师
   ↓
5. 问题解决
   ↓
6. 验证恢复
   ↓
7. 事后分析（Post-mortem）
   ↓
8. 更新文档和流程
```

### 变更管理

#### 变更类型

| 类型 | 审批 | 测试要求 | 回滚计划 |
|------|------|----------|----------|
| **紧急修复** | 技术主管 | 冒烟测试 | 必需 |
| **常规变更** | 团队 Lead | 完整测试 | 必需 |
| **重大变更** | 工程经理 | 完整测试 + 灰度 | 必需 |

#### 变更流程

1. **提交变更请求** - 填写变更单
2. **风险评估** - 评估影响范围
3. **审批** - 根据变更类型获取审批
4. **测试** - 在测试环境验证
5. **执行** - 在生产环境执行
6. **验证** - 验证变更效果
7. **文档** - 更新相关文档

### 容量管理

#### 容量监控

- **计算资源**: CPU、内存、网络
- **存储资源**: 磁盘空间、IOPS
- **网络资源**: 带宽、延迟
- **应用资源**: 连接数、消息队列深度

#### 扩容触发条件

- CPU 使用率 > 70% 持续 15 分钟
- 内存使用率 > 70% 持续 15 分钟
- 磁盘使用率 > 80%
- 网络带宽 > 70% 持续 10 分钟

#### 扩容流程

1. **评估需求** - 确定扩容规模
2. **制定计划** - 选择扩容方式（水平/垂直）
3. **测试验证** - 在测试环境验证
4. **执行扩容** - 在生产环境执行
5. **监控验证** - 验证扩容效果
6. **文档更新** - 更新容量记录

## 工具和脚本

### 健康检查脚本

```bash
#!/bin/bash
# scripts/health-check.sh

echo "=== Multi-Region Health Check ==="
echo "1. Service Status:"
docker ps --format "table {{.Names}}\t{{.Status}}" | grep -E "im-service|im-gateway"

echo "2. Cross-Region Connectivity:"
docker exec im-service-region-a ping -c 1 im-service-region-b

echo "3. Key Metrics:"
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile\(0.99,cross_region_sync_latency_ms\)
```

### 故障转移测试脚本

```bash
#!/bin/bash
# scripts/test-failover.sh

echo "=== Failover Test ==="
echo "1. Stopping Region A..."
docker stop im-service-region-a im-gateway-region-a

echo "2. Waiting for failover..."
start_time=$(date +%s)
while ! curl -s http://im-gateway.example.com/health | grep -q region-b; do
  sleep 1
done
end_time=$(date +%s)
failover_time=$((end_time - start_time))

echo "✓ Failover completed in ${failover_time} seconds"

echo "3. Restarting Region A..."
docker start im-service-region-a im-gateway-region-a
```

### 性能分析脚本

```bash
#!/bin/bash
# scripts/performance-analysis.sh

echo "=== Performance Analysis ==="
echo "1. Latency Metrics:"
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile\(0.99,cross_region_sync_latency_ms\)

echo "2. Throughput Metrics:"
curl -s http://localhost:9090/api/v1/query?query=sum\(rate\(messages_sent_total\[5m\]\)\)by\(region\)

echo "3. Resource Usage:"
docker stats --no-stream | grep -E "im-service|im-gateway"
```

## 最佳实践

### 监控最佳实践

1. **设置合理的告警阈值** - 避免告警疲劳
2. **使用多级告警** - 警告 → 严重 → 紧急
3. **定期审查告警** - 每月审查告警有效性
4. **自动化响应** - 对常见问题自动修复
5. **保留历史数据** - 至少保留 90 天

### 故障排查最佳实践

1. **先检查监控** - 查看 Grafana 面板
2. **查看日志** - 使用 Loki 聚合日志
3. **隔离问题** - 确定是哪个组件的问题
4. **记录过程** - 记录诊断步骤和发现
5. **分享经验** - 更新故障排查文档

### 容量规划最佳实践

1. **提前规划** - 至少提前 3 个月
2. **保留余量** - 目标 70% 利用率
3. **定期审查** - 每月审查容量使用
4. **成本优化** - 使用预留实例和 Spot 实例
5. **自动化扩容** - 配置自动扩缩容

### 性能优化最佳实践

1. **先测量后优化** - 基于数据做决策
2. **优化瓶颈** - 关注最慢的组件
3. **渐进式优化** - 一次优化一个方面
4. **验证效果** - 对比优化前后指标
5. **文档记录** - 记录优化方法和效果

## 紧急联系方式

### 值班联系

- **主要值班**: PagerDuty
- **备用值班**: #oncall-backup (Slack)
- **升级联系**: #incident-response (Slack)

### 专家联系

- **架构问题**: #architecture (Slack)
- **数据库问题**: #database-team (Slack)
- **网络问题**: #network-team (Slack)
- **安全问题**: #security-team (Slack)

### 供应商支持

- **云服务商**: support@cloud-provider.com
- **监控服务**: support@monitoring-vendor.com
- **数据库服务**: support@database-vendor.com

## 相关资源

### 内部文档

- [架构文档](../../architecture/MULTI_REGION_ACTIVE_ACTIVE.md)
- [部署指南](../../deployment/MULTI_REGION_DEPLOYMENT.md)

### 外部资源

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Docker 文档](https://docs.docker.com/)

### 培训材料

- [多地域架构培训](../../../docs/training/multi-region-architecture.md)
- [故障排查培训](../../../docs/training/troubleshooting.md)
- [监控告警培训](../../../docs/training/monitoring-alerting.md)

## 更新日志

### 2024-01

- ✅ 创建运维文档目录
- ✅ 发布故障排查手册
- ✅ 发布容量规划指南
- ✅ 发布性能调优指南

### 未来计划

- [ ] 添加监控告警手册
- [ ] 添加灾难恢复手册
- [ ] 添加安全运维指南
- [ ] 添加成本优化指南

---

**维护者**: Platform Engineering Team  
**最后更新**: 2024  
**联系方式**: #multi-region-ops (Slack)
