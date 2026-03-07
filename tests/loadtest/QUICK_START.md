# Quick Start Guide - Load Test Suite

快速开始使用压力测试套件。

## 前置条件

1. **Go 1.21+** 已安装
2. **运行中的 IM 服务**:
   - Region A: `ws://localhost:8080/ws`
   - Region B: `ws://localhost:8081/ws`

## 5 分钟快速开始

### 1. 安装依赖

```bash
cd tests/loadtest
go mod download
```

### 2. 构建 CLI 工具

```bash
make build
```

### 3. 运行基础测试

```bash
./bin/loadtest \
  -connections 100 \
  -rate 10 \
  -duration 30s \
  -region-a ws://localhost:8080/ws \
  -region-b ws://localhost:8081/ws
```

**预期输出**:
```
=== Load Test Configuration ===
Total Connections: 100
Region A: 50% (50 connections)
Region B: 50% (50 connections)
Message Rate: 10 msg/s
Duration: 30s

Connecting 100 WebSocket connections...
Connected: 100 active (Region A: 50, Region B: 50)
Running load test for 30s...
Test completed, collecting results...

=== Load Test Results ===
Duration: 30s
Total Messages: 300
Success Rate: 99.67%
Throughput: 10.00 msg/s

=== Latency Statistics ===
P50: 45ms
P95: 120ms
P99: 180ms
Cross-Region P99: 200ms

✅ Success Rate: 99.67% (target: ≥99%)
✅ P99 Latency: 180ms (target: ≤500ms)
✅ Cross-Region P99: 200ms (target: ≤500ms)

🎉 All performance targets met!
```

## 常用测试场景

### 场景 1: 小规模功能验证

```bash
make run
```

**用途**: 验证基本功能是否正常

### 场景 2: 故障转移测试

```bash
make run-failover
```

**用途**: 验证故障转移机制

### 场景 3: 中等规模性能测试

```bash
./bin/loadtest \
  -connections 10000 \
  -rate 1000 \
  -duration 5m \
  -rampup 1m \
  -output results.json
```

**用途**: 验证系统在中等负载下的性能

### 场景 4: 大规模压力测试

```bash
make prod-large
```

**用途**: 验证系统在 10 万连接下的表现

## 使用 Go 测试框架

### 运行所有测试

```bash
make test
```

### 运行特定测试

```bash
# 基础压力测试
go test -v -run TestBasicLoadTest -timeout 5m

# 故障转移测试
go test -v -run TestFailoverLoadTest -timeout 10m
```

### 跳过长时间测试

```bash
go test -v -short
```

## 自定义配置

### 方法 1: 命令行参数

```bash
./bin/loadtest \
  -connections 5000 \
  -region-a-percent 60 \
  -rate 500 \
  -duration 2m \
  -rampup 20s \
  -cross-region 0.4 \
  -region-a ws://region-a.example.com/ws \
  -region-b ws://region-b.example.com/ws \
  -token "your-auth-token" \
  -output custom-results.json
```

### 方法 2: Go 代码

```go
package main

import (
    "context"
    "time"
    "github.com/cuckoo-org/cuckoo/tests/loadtest"
)

func main() {
    config := &loadtest.LoadTestConfig{
        TotalConnections: 5000,
        RegionAPercent:   60,
        MessageRate:      500,
        Duration:         2 * time.Minute,
        RampUpTime:       20 * time.Second,
        CrossRegionRatio: 0.4,
        RegionAEndpoint:  "ws://region-a.example.com/ws",
        RegionBEndpoint:  "ws://region-b.example.com/ws",
        AuthToken:        "your-auth-token",
    }
    
    runner := loadtest.NewLoadTestRunner(config)
    defer runner.Cleanup()
    
    result, _ := runner.Run(context.Background())
    // 处理结果...
}
```

## 结果分析

### 查看 JSON 结果

```bash
cat results.json | jq '.'
```

### 关键指标解读

| 指标 | 说明 | 目标值 |
|------|------|--------|
| `success_rate` | 消息发送成功率 | ≥ 99% |
| `latency_p99` | 99% 消息的延迟 | ≤ 500ms |
| `throughput` | 每秒消息数 | 根据配置 |
| `cross_region_p99` | 跨地域消息 P99 延迟 | ≤ 500ms |

### 故障转移指标

| 指标 | 说明 | 目标值 |
|------|------|--------|
| `failover_duration` | 故障转移耗时 | < 30s |
| `throughput_drop` | 吞吐量下降比例 | < 50% |
| `failed_messages` | 失败消息数 | < 0.1% |

## 故障排查

### 问题 1: 连接失败

**错误**: `failed to connect: dial tcp: connection refused`

**解决**:
```bash
# 检查服务是否运行
curl http://localhost:8080/health

# 检查端口是否监听
netstat -an | grep 8080
```

### 问题 2: 认证失败

**错误**: `WebSocket upgrade failed: 401 Unauthorized`

**解决**:
```bash
# 使用正确的认证令牌
./bin/loadtest -token "valid-token" ...
```

### 问题 3: 内存不足

**错误**: `cannot allocate memory`

**解决**:
```bash
# 减少连接数
./bin/loadtest -connections 1000 ...

# 或增加系统限制
ulimit -n 65536
```

## 性能调优建议

### 1. 系统配置

```bash
# 增加文件描述符限制
ulimit -n 100000

# 调整 TCP 参数
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_fin_timeout=30
```

### 2. 测试参数

- **预热时间**: 大规模测试建议设置 5-10 分钟预热
- **速率控制**: 根据服务器能力调整消息速率
- **连接分布**: 根据实际部署调整地域比例

### 3. 监控建议

在运行测试时，同时监控：
- 服务器 CPU 使用率
- 内存使用率
- 网络带宽
- 数据库连接数

## 下一步

1. **阅读完整文档**: [README.md](README.md)
2. **查看实现细节**: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
3. **运行生产测试**: 使用 `make prod-*` 命令
4. **集成到 CI/CD**: 将测试加入持续集成流程

## 获取帮助

```bash
# 查看所有可用命令
make help

# 查看 CLI 帮助
./bin/loadtest -h
```

## 示例输出

### 成功的测试

```
🎉 All performance targets met!
```

### 需要优化的测试

```
⚠️  Some performance targets not met
❌ Success Rate: 95.23% (target: ≥99%)
❌ P99 Latency: 650ms (target: ≤500ms)
```

## 常见问题

**Q: 测试需要多长时间？**
A: 取决于配置，基础测试 30 秒，生产测试 5-30 分钟，稳定性测试 24 小时。

**Q: 可以在生产环境运行吗？**
A: 可以，但建议先在测试环境验证，并在低峰期运行。

**Q: 如何模拟真实用户行为？**
A: 调整 `CrossRegionRatio` 和 `MessageRate` 参数，模拟实际使用模式。

**Q: 测试会影响生产服务吗？**
A: 大规模测试会产生负载，建议在独立环境或低峰期运行。

---

**准备好了吗？** 运行你的第一个测试：

```bash
make run
```
