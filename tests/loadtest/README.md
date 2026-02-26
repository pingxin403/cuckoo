# Load Test Suite (压力测试套件)

多地域 IM 系统压力测试套件，用于验证系统在高并发场景下的性能和稳定性。

## 功能特性

### 核心功能
- ✅ **WebSocket 连接池**: 模拟大量并发 WebSocket 连接
- ✅ **速率控制**: 精确控制消息发送速率
- ✅ **延迟测量**: 实时测量消息延迟 (P50/P95/P99)
- ✅ **统计分析**: 吞吐量、成功率、延迟分布统计
- ✅ **故障转移测试**: 模拟地域故障并测量影响
- ✅ **跨地域测试**: 支持跨地域消息发送和延迟测量

### 验证需求
- **需求 9.1.1**: 模拟至少 10 万并发 WebSocket 连接，分布在两个地域
- **需求 9.1.2**: 测量跨地域消息吞吐量并输出 P50/P95/P99 延迟
- **需求 9.1.3**: 测量故障转移对吞吐量和延迟的影响，验证无消息丢失
- **需求 9.1.4**: 支持持续运行至少 24 小时的稳定性测试

## 快速开始

### 安装依赖

```bash
cd tests/loadtest
go mod download
```

### 基础压力测试

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/cuckoo-org/cuckoo/tests/loadtest"
)

func main() {
    config := &loadtest.LoadTestConfig{
        TotalConnections: 10000,
        RegionAPercent:   50,
        MessageRate:      1000,
        Duration:         5 * time.Minute,
        RampUpTime:       30 * time.Second,
        CrossRegionRatio: 0.3,
        RegionAEndpoint:  "ws://region-a.example.com/ws",
        RegionBEndpoint:  "ws://region-b.example.com/ws",
        AuthToken:        "your-auth-token",
    }
    
    runner := loadtest.NewLoadTestRunner(config)
    defer runner.Cleanup()
    
    result, err := runner.Run(context.Background())
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Total Messages: %d\n", result.TotalMessages)
    fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
    fmt.Printf("Throughput: %.2f msg/s\n", result.Throughput)
    fmt.Printf("P99 Latency: %v\n", result.LatencyP99)
}
```

### 故障转移测试

```go
config := &loadtest.FailoverTestConfig{
    LoadTestConfig: loadtest.LoadTestConfig{
        TotalConnections: 10000,
        RegionAPercent:   50,
        MessageRate:      1000,
        Duration:         10 * time.Minute,
        RampUpTime:       30 * time.Second,
        CrossRegionRatio: 0.3,
        RegionAEndpoint:  "ws://region-a.example.com/ws",
        RegionBEndpoint:  "ws://region-b.example.com/ws",
        AuthToken:        "your-auth-token",
    },
    FailoverTriggerDelay: 3 * time.Minute,
    FailoverRecoveryTime: 30 * time.Second,
    FailedRegion:         "region-a",
}

runner := loadtest.NewFailoverTestRunner(config)
defer runner.Cleanup()

result, err := runner.Run(context.Background())
if err != nil {
    panic(err)
}

// 验证故障转移影响
if result.FailoverImpact != nil {
    fmt.Printf("Failover Duration: %v\n", 
        result.FailoverImpact.EndTime.Sub(result.FailoverImpact.StartTime))
    fmt.Printf("Throughput Drop: %.2f%%\n",
        (result.FailoverImpact.ThroughputBefore - result.FailoverImpact.ThroughputDuring) /
        result.FailoverImpact.ThroughputBefore * 100)
}

// 验证无消息丢失
if err := runner.VerifyNoMessageLoss(); err != nil {
    fmt.Printf("Message loss detected: %v\n", err)
}
```

## 运行测试

### 单元测试

```bash
# 运行所有测试
go test -v

# 运行特定测试
go test -v -run TestBasicLoadTest

# 跳过长时间测试
go test -v -short
```

### 压力测试

```bash
# 基础压力测试 (1000 连接)
go test -v -run TestBasicLoadTest -timeout 2m

# 故障转移测试
go test -v -run TestFailoverLoadTest -timeout 5m

# 长时间稳定性测试 (需要更长超时)
go test -v -run TestLongRunningStabilityTest -timeout 30m
```

### 性能基准测试

```bash
go test -bench=. -benchmem -benchtime=10s
```

## 配置说明

### LoadTestConfig

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| TotalConnections | int | 总连接数 | 必填 |
| RegionAPercent | int | Region A 连接占比 (0-100) | 必填 |
| MessageRate | int | 消息发送速率 (msg/s) | 必填 |
| Duration | time.Duration | 测试持续时间 | 必填 |
| RampUpTime | time.Duration | 预热时间 | 0 |
| CrossRegionRatio | float64 | 跨地域消息比例 (0.0-1.0) | 0.0 |
| RegionAEndpoint | string | Region A WebSocket 端点 | 必填 |
| RegionBEndpoint | string | Region B WebSocket 端点 | 必填 |
| AuthToken | string | 认证令牌 | "" |

### FailoverTestConfig

继承 `LoadTestConfig`，额外字段：

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| FailoverTriggerDelay | time.Duration | 故障转移触发延迟 | 必填 |
| FailoverRecoveryTime | time.Duration | 故障转移恢复时间 | 必填 |
| FailedRegion | string | 故障地域 ("region-a" 或 "region-b") | 必填 |

## 测试结果

### LoadTestResult

```go
type LoadTestResult struct {
    TotalMessages  int64         // 总消息数
    SuccessRate    float64       // 成功率 (%)
    LatencyP50     time.Duration // P50 延迟
    LatencyP95     time.Duration // P95 延迟
    LatencyP99     time.Duration // P99 延迟
    CrossRegionP99 time.Duration // 跨地域 P99 延迟
    Duration       time.Duration // 测试持续时间
    Throughput     float64       // 吞吐量 (msg/s)
    FailoverImpact *FailoverImpact // 故障转移影响 (可选)
}
```

### FailoverImpact

```go
type FailoverImpact struct {
    StartTime              time.Time     // 故障转移开始时间
    EndTime                time.Time     // 故障转移完成时间
    MessagesDuringFailover int64         // 故障转移期间消息数
    FailedMessages         int64         // 失败消息数
    ThroughputBefore       float64       // 故障转移前吞吐量
    ThroughputDuring       float64       // 故障转移期间吞吐量
    ThroughputAfter        float64       // 故障转移后吞吐量
    LatencyIncrease        time.Duration // 延迟增加
}
```

## 架构设计

### 组件结构

```
tests/loadtest/
├── types.go              # 类型定义
├── connection_pool.go    # WebSocket 连接池
├── rate_limiter.go       # 速率控制器
├── statistics.go         # 统计分析
├── runner.go             # 测试运行器
├── failover_test.go      # 故障转移测试
├── example_test.go       # 示例测试
└── README.md             # 文档
```

### 核心流程

```
1. 建立连接
   ├── 创建连接池
   ├── 按比例分配地域
   └── 预热 (逐步建立连接)

2. 发送消息
   ├── 速率控制
   ├── 延迟测量
   └── 统计收集

3. 故障转移 (可选)
   ├── 关闭故障地域连接
   ├── 等待恢复时间
   └── 重连到健康地域

4. 收集结果
   ├── 聚合统计
   ├── 计算延迟分布
   └── 分析故障转移影响
```

## 性能指标

### 目标指标

- **连接数**: 支持 10 万+ 并发连接
- **吞吐量**: 10,000+ msg/s
- **延迟**: P99 < 500ms
- **成功率**: > 99%
- **RTO**: < 30 秒
- **消息丢失率**: < 0.1%

### 实际测试结果

运行测试后查看实际性能指标，并与目标对比。

## 故障排查

### 连接失败

```
Error: failed to connect: dial tcp: connection refused
```

**解决方案**:
1. 检查 WebSocket 端点是否正确
2. 确认服务是否运行
3. 检查网络连接和防火墙

### 认证失败

```
Error: WebSocket upgrade failed: 401 Unauthorized
```

**解决方案**:
1. 检查 AuthToken 是否正确
2. 确认令牌未过期
3. 验证认证服务是否正常

### 内存不足

```
Error: cannot allocate memory
```

**解决方案**:
1. 减少 TotalConnections
2. 增加系统内存
3. 调整 ulimit 限制

## 最佳实践

### 1. 逐步增加负载

```go
config := &LoadTestConfig{
    TotalConnections: 100000,
    RampUpTime:       5 * time.Minute, // 5 分钟预热
    // ...
}
```

### 2. 监控系统资源

- CPU 使用率
- 内存使用率
- 网络带宽
- 文件描述符数量

### 3. 分阶段测试

1. **小规模测试** (1,000 连接): 验证功能
2. **中等规模测试** (10,000 连接): 验证性能
3. **大规模测试** (100,000 连接): 验证稳定性

### 4. 记录测试结果

```go
result, _ := runner.Run(ctx)

// 保存结果到文件
data, _ := json.MarshalIndent(result, "", "  ")
os.WriteFile("loadtest-result.json", data, 0644)
```

## 扩展开发

### 自定义消息格式

修改 `runner.go` 中的 `generateMessage` 方法：

```go
func (r *LoadTestRunner) generateMessage(conn *WSConnection) []byte {
    msg := YourCustomMessage{
        // 自定义字段
    }
    data, _ := json.Marshal(msg)
    return data
}
```

### 添加自定义指标

扩展 `LoadTestResult` 结构体：

```go
type LoadTestResult struct {
    // 现有字段...
    
    // 自定义指标
    CustomMetric1 float64 `json:"custom_metric_1"`
    CustomMetric2 int64   `json:"custom_metric_2"`
}
```

## 参考资料

- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Go Testing Package](https://pkg.go.dev/testing)

## 许可证

MIT License
