# Chaos Engineering Tests (混沌工程测试)

多地域 IM 系统混沌工程测试套件，通过故障注入验证系统的容错能力和数据一致性。

## 功能特性

### 核心功能
- ✅ **网络故障注入**: 网络分区、延迟、丢包、带宽限制
- ✅ **时钟偏移注入**: 正向/负向时钟偏移、时钟回拨、跨地域时钟不同步
- ✅ **节点故障注入**: Pod 杀死、容器杀死、Pod 失败
- ✅ **自动化测试脚本**: 故障注入 → 等待恢复 → 验证数据一致性
- ✅ **数据一致性验证**: 基于 Merkle Tree 的数据对账

### 验证需求
- **需求 9.2.1**: 使用 Chaos Mesh 注入网络延迟、丢包和分区故障
- **需求 9.2.2**: 网络分区恢复后 60 秒内完成数据重新同步
- **需求 9.2.3**: 单个数据中心完全不可用时，在 RTO（30 秒）内完成故障转移
- **需求 9.2.4**: 注入时钟偏移（最大 5 秒），验证 HLC 算法的容错能力
- **需求 9.2.5**: 验证故障恢复后数据一致性（通过 Merkle Tree 对账）

## 快速开始

### 前置条件

1. **Kubernetes 集群**: 需要一个运行中的 Kubernetes 集群
2. **Chaos Mesh**: 安装 Chaos Mesh

```bash
# 安装 Chaos Mesh
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash

# 验证安装
kubectl get pods -n chaos-mesh
```

3. **kubectl**: 安装并配置 kubectl
4. **jq**: 安装 jq 用于 JSON 处理

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# CentOS/RHEL
sudo yum install jq
```

### 运行测试

#### 自动化测试脚本

```bash
# 运行所有混沌工程测试
cd tests/chaos
./run-chaos-tests.sh

# 使用自定义配置
NAMESPACE=default \
REGION_A_NAMESPACE=im-region-a \
REGION_B_NAMESPACE=im-region-b \
SYNC_TIMEOUT=60 \
RTO_TIMEOUT=30 \
./run-chaos-tests.sh
```

#### 手动测试

##### 1. 网络分区测试

```bash
# 应用网络分区配置
kubectl apply -f network-partition.yaml

# 等待 60 秒
sleep 60

# 删除网络分区配置（恢复网络）
kubectl delete -f network-partition.yaml

# 验证数据同步
# 检查两个地域的 Merkle Root 是否一致
```

##### 2. 时钟偏移测试

```bash
# 应用时钟偏移配置
kubectl apply -f clock-skew.yaml

# 等待 120 秒
sleep 120

# 验证 HLC 健康状态
curl http://region-a-service/api/v1/hlc/health

# 删除时钟偏移配置
kubectl delete -f clock-skew.yaml
```

##### 3. Pod 故障测试

```bash
# 应用 Pod Kill 配置
kubectl apply -f pod-kill.yaml

# 监控故障转移
kubectl get pods -n im-region-a -w
kubectl get pods -n im-region-b -w

# 删除 Pod Kill 配置
kubectl delete -f pod-kill.yaml
```

## 配置文件说明

### network-partition.yaml

网络故障注入配置，包含以下测试场景：

| 测试场景 | 描述 | 持续时间 | 验证需求 |
|---------|------|---------|---------|
| cross-region-partition | 跨地域网络分区 | 60s | 9.2.1, 9.2.2 |
| cross-region-latency | 跨地域网络延迟 (500ms) | 120s | 9.2.1 |
| cross-region-packet-loss | 网络丢包 (10%) | 90s | 9.2.1 |
| cross-region-bandwidth-limit | 带宽限制 (1 Mbps) | 120s | 9.2.1 |

**配置示例**:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: cross-region-partition
spec:
  action: partition
  mode: all
  selector:
    namespaces: ["im-region-a"]
    labelSelectors:
      app: im-service
  direction: both
  target:
    selector:
      namespaces: ["im-region-b"]
      labelSelectors:
        app: im-service
  duration: "60s"
```

### clock-skew.yaml

时钟偏移注入配置，包含以下测试场景：

| 测试场景 | 描述 | 时钟偏移 | 持续时间 | 验证需求 |
|---------|------|---------|---------|---------|
| clock-skew-positive | 正向时钟偏移 | +5s | 120s | 9.2.4 |
| clock-skew-negative | 负向时钟偏移 | -3s | 120s | 9.2.4 |
| clock-backward | 时钟回拨 | -10s | 60s | 9.2.4 |
| cross-region-clock-desync | 跨地域时钟不同步 | ±5s | 180s | 9.2.4 |
| clock-skew-extreme | 极端时钟偏移 | +30s | 60s | 9.2.4 |

**配置示例**:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: TimeChaos
metadata:
  name: clock-skew-positive
spec:
  mode: one
  selector:
    namespaces: ["im-region-a"]
    labelSelectors:
      app: im-service
  timeOffset: "5s"
  duration: "120s"
  clockIds:
    - CLOCK_REALTIME
```

### pod-kill.yaml

Pod 故障注入配置，包含以下测试场景：

| 测试场景 | 描述 | 模式 | 调度 | 验证需求 |
|---------|------|------|------|---------|
| im-service-pod-kill | 杀死单个 IM 服务 Pod | one | @every 5m | 9.2.3 |
| im-service-region-failure | 杀死整个地域的 IM 服务 | all | 手动 | 9.2.3 |
| im-service-container-kill | 杀死容器 | one | @every 10m | 9.2.3 |
| im-service-pod-failure | Pod 失败 | one | 60s | 9.2.3 |
| mysql-pod-kill | 杀死 MySQL Pod | one | @every 30m | 9.2.3 |
| redis-pod-kill | 杀死 Redis Pod | one | @every 20m | 9.2.3 |
| kafka-pod-kill | 杀死 Kafka Pod | one | @every 15m | 9.2.3 |
| multi-component-failure | 多组件同时故障 | fixed(2) | 手动 | 9.2.3 |

**配置示例**:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: im-service-pod-kill
spec:
  action: pod-kill
  mode: one
  selector:
    namespaces: ["im-region-a"]
    labelSelectors:
      app: im-service
  scheduler:
    cron: "@every 5m"
```

## 测试脚本说明

### run-chaos-tests.sh

自动化混沌工程测试脚本，执行以下测试流程：

1. **网络分区恢复测试** (test_network_partition_recovery)
   - 注入网络分区故障
   - 等待 60 秒
   - 恢复网络
   - 验证数据在 60 秒内完成重新同步
   - 验证需求: 9.2.1, 9.2.2

2. **地域故障转移测试** (test_region_failover)
   - 杀死 Region A 所有 IM 服务 Pod
   - 监控 Region B 接管流量
   - 验证故障转移在 30 秒内完成
   - 恢复 Region A
   - 验证需求: 9.2.3

3. **时钟偏移容错测试** (test_clock_skew_tolerance)
   - 注入时钟偏移 (+5s)
   - 等待 120 秒
   - 验证 HLC 仍然正确工作
   - 验证需求: 9.2.4

4. **数据一致性验证测试** (test_data_consistency_verification)
   - 验证故障恢复后数据一致性
   - 使用 Merkle Tree 对账
   - 验证需求: 9.2.5

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| NAMESPACE | Chaos 实验命名空间 | default |
| REGION_A_NAMESPACE | Region A 命名空间 | im-region-a |
| REGION_B_NAMESPACE | Region B 命名空间 | im-region-b |
| CHAOS_MESH_NAMESPACE | Chaos Mesh 命名空间 | chaos-mesh |
| SYNC_TIMEOUT | 数据同步超时时间（秒） | 60 |
| RTO_TIMEOUT | 故障转移超时时间（秒） | 30 |
| VERIFICATION_INTERVAL | 验证间隔（秒） | 5 |

### 输出示例

```
[INFO] 2024-01-15 10:00:00 - ==========================================
[INFO] 2024-01-15 10:00:00 - 混沌工程测试开始
[INFO] 2024-01-15 10:00:00 - ==========================================
[INFO] 2024-01-15 10:00:00 - 检查依赖...
[SUCCESS] 2024-01-15 10:00:01 - 依赖检查通过
[INFO] 2024-01-15 10:00:01 - ==========================================
[INFO] 2024-01-15 10:00:01 - 测试: 网络分区恢复 (需求 9.2.1, 9.2.2)
[INFO] 2024-01-15 10:00:01 - ==========================================
[INFO] 2024-01-15 10:00:01 - 应用 Chaos 配置: tests/chaos/network-partition.yaml
[INFO] 2024-01-15 10:00:03 - 网络分区已注入，等待 60 秒...
[INFO] 2024-01-15 10:01:03 - 删除 Chaos 配置: tests/chaos/network-partition.yaml
[SUCCESS] 2024-01-15 10:01:05 - Chaos 配置已删除
[INFO] 2024-01-15 10:01:05 - 网络分区已恢复，开始验证数据同步...
[INFO] 2024-01-15 10:01:10 - 验证数据一致性（Merkle Tree 对账）...
[INFO] 2024-01-15 10:01:10 - Region A Merkle Root: abc123...
[INFO] 2024-01-15 10:01:10 - Region B Merkle Root: abc123...
[SUCCESS] 2024-01-15 10:01:10 - 数据一致性验证通过
[SUCCESS] 2024-01-15 10:01:10 - 数据重新同步完成，耗时: 5s
[SUCCESS] 2024-01-15 10:01:10 - ✓ 需求 9.2.2 验证通过: 数据在 5s 内完成重新同步 (要求 < 60s)
[SUCCESS] 2024-01-15 10:01:10 - 网络分区恢复测试完成，总耗时: 69s

...

[INFO] 2024-01-15 10:05:00 - ==========================================
[INFO] 2024-01-15 10:05:00 - 混沌工程测试完成
[INFO] 2024-01-15 10:05:00 - ==========================================
[INFO] 2024-01-15 10:05:00 - 总测试数: 4
[SUCCESS] 2024-01-15 10:05:00 - 通过: 4
[INFO] 2024-01-15 10:05:00 - 失败: 0
[SUCCESS] 2024-01-15 10:05:00 - 所有测试通过！
```

## 数据一致性验证

### Merkle Tree 对账

混沌工程测试使用 Merkle Tree 对账来验证数据一致性：

1. **Merkle Root 计算**: 每个地域计算其数据的 Merkle Root 哈希值
2. **哈希值比较**: 比较两个地域的 Merkle Root 是否一致
3. **差异检测**: 如果哈希值不一致，说明数据存在差异
4. **自动修复**: 系统应自动修复数据差异

### API 端点

```bash
# 获取 Region A 的 Merkle Root
curl http://region-a-service/api/v1/reconcile/merkle-root

# 响应示例
{
  "hash": "abc123def456...",
  "timestamp": 1705305600,
  "region_id": "region-a"
}

# 获取 Region B 的 Merkle Root
curl http://region-b-service/api/v1/reconcile/merkle-root

# 响应示例
{
  "hash": "abc123def456...",
  "timestamp": 1705305600,
  "region_id": "region-b"
}
```

## 故障排查

### Chaos Mesh 未安装

```
[ERROR] Chaos Mesh 未安装，请先安装 Chaos Mesh
```

**解决方案**:

```bash
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash
```

### Pod 未就绪

```
[ERROR] 等待 Pod 就绪超时 (60s)
```

**解决方案**:
1. 检查 Pod 状态: `kubectl get pods -n im-region-a`
2. 查看 Pod 日志: `kubectl logs -n im-region-a <pod-name>`
3. 检查资源限制: `kubectl describe pod -n im-region-a <pod-name>`

### 数据一致性验证失败

```
[ERROR] 数据一致性验证失败: 两个地域的 Merkle Root 不一致
```

**解决方案**:
1. 检查数据同步状态
2. 查看同步日志
3. 手动触发数据对账
4. 检查网络连接

### 故障转移超时

```
[ERROR] 故障转移超时 (30s)
```

**解决方案**:
1. 检查健康检查配置
2. 检查 Region B 的资源是否充足
3. 查看故障转移日志
4. 增加 RTO_TIMEOUT 值

## 最佳实践

### 1. 逐步增加故障强度

从轻微故障开始，逐步增加故障强度：

1. **轻微故障**: 网络延迟、单个 Pod 故障
2. **中等故障**: 网络丢包、多个 Pod 故障
3. **严重故障**: 网络分区、整个地域故障

### 2. 监控系统指标

在测试期间监控以下指标：

- CPU 使用率
- 内存使用率
- 网络带宽
- 消息延迟
- 错误率
- 故障转移时间

### 3. 定期执行测试

建议定期执行混沌工程测试：

- **每日**: 轻微故障测试
- **每周**: 中等故障测试
- **每月**: 严重故障测试

### 4. 记录测试结果

保存测试结果用于分析和改进：

```bash
# 保存测试日志
./run-chaos-tests.sh 2>&1 | tee chaos-test-$(date +%Y%m%d-%H%M%S).log

# 保存测试指标
kubectl top pods -n im-region-a > metrics-region-a.txt
kubectl top pods -n im-region-b > metrics-region-b.txt
```

## 扩展开发

### 添加新的 Chaos 场景

1. 创建新的 YAML 配置文件
2. 在 `run-chaos-tests.sh` 中添加测试函数
3. 更新 README 文档

### 自定义验证逻辑

修改 `verify_data_consistency` 函数：

```bash
verify_data_consistency() {
    local region_a_endpoint=$1
    local region_b_endpoint=$2
    
    # 自定义验证逻辑
    # ...
    
    return 0
}
```

## 参考资料

- [Chaos Mesh 官方文档](https://chaos-mesh.org/docs/)
- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [混沌工程原则](https://principlesofchaos.org/)
- [Merkle Tree 数据结构](https://en.wikipedia.org/wiki/Merkle_tree)

## 许可证

MIT License
