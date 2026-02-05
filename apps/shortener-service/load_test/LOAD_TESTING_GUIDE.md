# 短链服务压测指南

## 概述

本指南提供了完整的压测流程，包括环境准备、测试数据创建、测试场景执行和结果分析。所有测试脚本都遵循 k6 最佳实践，确保测试结果的准确性和可重复性。

## 目录

- [环境准备](#环境准备)
- [测试数据准备](#测试数据准备)
- [测试场景](#测试场景)
- [执行流程](#执行流程)
- [结果分析](#结果分析)
- [故障排查](#故障排查)
- [性能基准](#性能基准)

---

## 环境准备

### 1. 安装 k6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows:**
```bash
choco install k6
```

### 2. 启动服务

**方式 1: 使用 Docker Compose（推荐）**

```bash
# 从项目根目录
cd apps/shortener-service

# 启动依赖服务（Redis, MySQL）
docker-compose -f integration_test/docker-compose.yml up -d

# 启动短链服务
go run main.go
```

**方式 2: 本地启动**

```bash
# 确保 Redis 和 MySQL 已启动
# 启动服务
cd apps/shortener-service
go run main.go
```

### 3. 验证服务状态

```bash
# 检查健康状态
curl http://localhost:8080/health

# 预期输出
{
  "status": "healthy",
  "checks": {
    "mysql": "ok",
    "redis": "ok"
  }
}
```

**服务端口：**
- HTTP: `8080`
- gRPC: `50051`

---

## 测试数据准备

### 自动创建测试数据

运行数据准备脚本，自动创建测试所需的短链：

```bash
cd apps/shortener-service/load_test
./prepare-test-data.sh
```

**脚本功能：**
- 创建 11 个测试短链（test001-test005 + 6 个随机短码）
- 验证所有短链可用性
- 输出测试数据摘要

**测试短码列表：**
```
test001 → https://example.com/page/1
test002 → https://example.com/page/2
test003 → https://example.com/page/3
test004 → https://example.com/page/4
test005 → https://example.com/page/5
ncll0yl → https://example.com/page/6
LqhWmMl → https://example.com/page/7
8eOIL5Z → https://example.com/page/8
0UCIIQf → https://example.com/page/9
oslpgO2 → https://example.com/page/10
YGuviUI → https://example.com/page/11
```

### 手动验证测试数据

```bash
# 验证单个短链
curl -I http://localhost:8080/test001

# 预期输出
HTTP/1.1 302 Found
Location: https://example.com/page/1
```

---

## 测试场景

### 1. Quick QPS Test - 快速性能验证

**用途：** 快速验证服务性能，适合日常开发和 CI/CD

**配置：**
- VUs: 100
- 持续时间: 30 秒
- 预期 QPS: 10K-20K

**执行：**
```bash
k6 run quick-qps-test.js
```

**成功标准：**
- P99 延迟 < 10ms
- 错误率 < 1%
- 缓存命中率 > 80%

---

### 2. Redirect QPS Test - 重定向性能测试

**用途：** 测试核心重定向功能的性能上限

**配置：**
- 阶段 1: 30s → 100 VUs（预热）
- 阶段 2: 1m → 500 VUs
- 阶段 3: 2m → 1000 VUs
- 阶段 4: 2m 保持 1000 VUs
- 阶段 5: 30s → 0 VUs（降压）

**执行：**
```bash
k6 run redirect-qps-test.js
```

**成功标准：**
- 单机 QPS: 150K-180K（高缓存命中）
- P95 延迟 < 10ms
- P99 延迟 < 20ms
- 错误率 < 1%
- 缓存命中率 > 90%

**流量分布：**
- 80% 请求命中前 5 个短码（模拟热点数据）
- 20% 请求随机分布

---

### 3. Sustained Load Test - 持续高负载测试

**用途：** 验证服务在持续高负载下的稳定性

**配置：**
- 目标 QPS: 100K
- 持续时间: 10 分钟
- 预分配 VUs: 1000
- 最大 VUs: 2000

**执行：**
```bash
k6 run sustained-load.js
```

**成功标准：**
- 持续 QPS: 100K
- P99 延迟 < 5ms
- 错误率 < 0.1%
- 缓存命中率 > 95%
- CPU 使用率 < 80%
- 内存使用率 < 80%

**流量分布：**
- 90% 请求命中前 5 个短码
- 10% 请求随机分布

---

### 4. Spike Test - 峰值冲击测试

**用途：** 测试服务应对突发流量的能力

**配置：**
- 阶段 1: 1m → 2000 VUs（快速冲击）
- 阶段 2: 2m 保持 2000 VUs
- 阶段 3: 1m → 0 VUs（恢复）

**执行：**
```bash
k6 run spike-test.js
```

**成功标准：**
- 峰值 QPS: 200K
- P99 延迟 < 10ms（峰值期间）
- 错误率 < 1%
- 系统恢复时间 < 30s

**流量分布：**
- 95% 请求命中前 5 个短码（峰值期间主要是缓存命中）
- 5% 请求随机分布

---

### 5. Cache Stampede Test - 缓存击穿测试

**用途：** 验证 Singleflight 机制防止缓存击穿

**配置：**
- VUs: 1000（并发）
- 总请求数: 1000
- 最大持续时间: 30s

**执行：**
```bash
k6 run cache-stampede.js
```

**成功标准：**
- DB 查询数 < 10（1000 个并发请求）
- DB 负载降低 > 90%
- P99 延迟 < 100ms
- 错误率 < 0.1%

**测试原理：**
- 所有 VUs 同时请求同一个短码
- Singleflight 确保只有一个请求查询 DB
- 其他请求等待并共享结果

---

### 6. Realistic QPS Test - 真实场景测试

**用途：** 模拟真实生产环境的流量模式

**配置：**
- 阶段 1: 30s → 100 VUs（预热）
- 阶段 2: 1m → 500 VUs
- 阶段 3: 2m → 1000 VUs
- 阶段 4: 2m 保持 1000 VUs
- 阶段 5: 30s → 0 VUs

**执行：**
```bash
k6 run realistic-qps-test.js
```

**成功标准：**
- 平均 QPS: 100K-120K
- P95 延迟 < 20ms
- P99 延迟 < 50ms
- 错误率 < 5%

**流量分布：**
- 80% 请求命中前 5 个短码
- 20% 请求随机分布

---

## 执行流程

### 完整测试流程

```bash
# 1. 准备环境
cd apps/shortener-service
docker-compose -f integration_test/docker-compose.yml up -d
go run main.go

# 2. 验证服务
curl http://localhost:8080/health

# 3. 准备测试数据
cd load_test
./prepare-test-data.sh

# 4. 执行测试（按顺序）
k6 run quick-qps-test.js           # 快速验证
k6 run redirect-qps-test.js        # 重定向性能
k6 run sustained-load.js           # 持续负载
k6 run spike-test.js               # 峰值冲击
k6 run cache-stampede.js           # 缓存击穿

# 5. 查看监控（可选）
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000
```

### 自定义测试参数

**修改 VUs 数量：**
```bash
k6 run --vus 2000 redirect-qps-test.js
```

**修改持续时间：**
```bash
k6 run --duration 5m sustained-load.js
```

**修改 Base URL：**
```bash
k6 run -e BASE_URL=http://192.168.1.100:8080 quick-qps-test.js
```

**输出结果到文件：**
```bash
k6 run --out json=results.json redirect-qps-test.js
```

**输出到 Prometheus：**
```bash
k6 run --out experimental-prometheus-rw sustained-load.js
```

---

## 结果分析

### k6 输出指标说明

**关键指标：**

| 指标 | 说明 | 目标值 |
|------|------|--------|
| `http_reqs` | 总请求数 | - |
| `http_req_duration` | 请求延迟 | P99 < 5ms |
| `http_req_failed` | 失败率 | < 0.1% |
| `vus` | 虚拟用户数 | - |
| `iterations` | 迭代次数 | - |

**延迟分位数：**
- `p(50)`: 中位数延迟（50% 请求）
- `p(95)`: 95% 请求的延迟
- `p(99)`: 99% 请求的延迟

**自定义指标：**
- `successful_redirects`: 成功重定向数
- `cache_hits_estimated`: 估计缓存命中数（< 5ms）
- `error_rate`: 错误率
- `requests_total`: 总请求数

### 示例输出

```
========================================
Redirect QPS Test Results
========================================
Total Requests: 1,234,567
Duration: 360.0s
Average QPS: 3,429
Successful Redirects: 1,234,000
Error Rate: 0.05%

Cache Performance:
  Estimated Cache Hits: 1,111,000 (90.03%)

Latency:
  P50: 2.15ms
  P95: 8.32ms
  P99: 15.67ms

========================================
```

### 性能分析

**1. QPS 分析**

```
实际 QPS = 总请求数 / 测试时长
```

**单机性能参考：**
- 纯重定向（高缓存命中）: 150K-180K QPS
- 混合负载（80读20写）: 100K-120K QPS
- 纯创建（写入密集）: 8K-10K QPS

**2. 延迟分析**

| 延迟范围 | 可能原因 |
|---------|---------|
| < 1ms | L1 缓存命中（Ristretto） |
| 1-5ms | L2 缓存命中（Redis） |
| 5-10ms | DB 查询（有索引） |
| > 10ms | 网络延迟或系统过载 |

**3. 缓存命中率分析**

```
缓存命中率 = 缓存命中数 / 总请求数 × 100%
```

**目标值：**
- 生产环境: > 95%
- 测试环境: > 90%

**4. 错误率分析**

```
错误率 = 失败请求数 / 总请求数 × 100%
```

**可接受范围：**
- 正常负载: < 0.1%
- 峰值负载: < 1%
- 压力测试: < 5%

---

## 故障排查

### 问题 1: 连接被拒绝

**症状：**
```
WARN[0001] Request Failed error="Get \"http://localhost:8080/test001\": dial tcp [::1]:8080: connect: connection refused"
```

**解决方案：**
1. 检查服务是否启动：
   ```bash
   curl http://localhost:8080/health
   ```

2. 检查端口是否被占用：
   ```bash
   lsof -i :8080
   ```

3. 查看服务日志：
   ```bash
   docker-compose logs shortener-service
   ```

---

### 问题 2: 高错误率

**症状：**
- 错误率 > 5%
- 大量 500 或 503 错误

**可能原因：**
1. 服务资源不足（CPU/内存）
2. 数据库连接池耗尽
3. Redis 连接池耗尽
4. 熔断器打开

**解决方案：**

**检查资源使用：**
```bash
# CPU 和内存
top

# 连接数
netstat -an | grep :8080 | wc -l
```

**调整连接池配置：**
```yaml
# config.yaml
redis:
  pool_size: 30
  min_idle_conns: 10

mysql:
  max_open_conns: 50
  max_idle_conns: 10
```

**检查熔断器状态：**
```bash
# 查看 Prometheus 指标
curl http://localhost:9090/api/v1/query?query=circuit_breaker_state
```

---

### 问题 3: 低吞吐量

**症状：**
- 实际 QPS 远低于预期
- VUs 数量不足

**可能原因：**
1. VUs 分配不足
2. 网络瓶颈
3. 服务资源限制

**解决方案：**

**增加 VUs：**
```bash
k6 run --vus 2000 --duration 5m redirect-qps-test.js
```

**增加预分配 VUs：**
```javascript
export const options = {
  scenarios: {
    sustained_load: {
      executor: 'constant-arrival-rate',
      rate: 100000,
      timeUnit: '1s',
      duration: '10m',
      preAllocatedVUs: 2000,  // 增加预分配
      maxVUs: 5000,           // 增加最大值
    },
  },
};
```

**检查资源限制：**
```bash
# 检查文件描述符限制
ulimit -n

# 增加限制
ulimit -n 65535
```

---

### 问题 4: 测试数据不存在

**症状：**
```
Error: No valid short codes found. Run prepare-test-data.sh first.
```

**解决方案：**
```bash
cd apps/shortener-service/load_test
./prepare-test-data.sh
```

---

### 问题 5: 缓存命中率低

**症状：**
- 缓存命中率 < 80%
- 延迟较高

**可能原因：**
1. 缓存 TTL 过短
2. 缓存容量不足
3. 流量分布不均

**解决方案：**

**调整缓存配置：**
```yaml
# config.yaml
cache:
  l1:
    max_size: 10000      # 增加 L1 缓存大小
    ttl: 300s            # 增加 TTL
  l2:
    ttl: 3600s           # 增加 Redis TTL
```

**优化流量分布：**
```javascript
// 增加热点数据比例
const index = Math.random() < 0.9  // 从 0.8 改为 0.9
  ? Math.floor(Math.random() * Math.min(5, data.codes.length))
  : Math.floor(Math.random() * data.codes.length);
```

---

## 性能基准

### 单机性能基准（8核16GB）

| 场景 | QPS | P99 延迟 | 缓存命中率 | 错误率 |
|------|-----|---------|-----------|--------|
| 纯重定向（高缓存） | 150K-180K | < 5ms | > 95% | < 0.1% |
| 混合负载（80读20写） | 100K-120K | < 10ms | > 90% | < 0.1% |
| 纯创建（写入密集） | 8K-10K | < 50ms | N/A | < 0.1% |
| 峰值冲击 | 200K | < 10ms | > 95% | < 1% |

### 水平扩展性能（5实例）

| 场景 | QPS | P99 延迟 | 备注 |
|------|-----|---------|------|
| 纯重定向 | 500K | < 5ms | Redis Cluster |
| 混合负载 | 500K | < 10ms | MySQL 主从 |
| 纯创建 | 50K | < 50ms | 写入瓶颈 |

### 性能优化建议

**达到 50 万 QPS 的方案：**

1. **水平扩展**（推荐）
   - 5 个服务实例
   - Redis Cluster（3主3从）
   - MySQL 主从（1主2从）
   - 负载均衡（Envoy/Nginx）

2. **缓存优化**
   - L1 缓存: 10GB（Ristretto）
   - L2 缓存: Redis Cluster
   - 缓存命中率 > 95%

3. **连接池优化**
   - Redis Pool: 30-50
   - MySQL Pool: 50-100
   - 连接复用

4. **网络优化**
   - 万兆网卡（10Gbps）
   - HTTP/2 或 gRPC
   - 响应压缩

详细方案参考：[单机性能分析文档](../docs/SINGLE_MACHINE_PERFORMANCE_ANALYSIS.md)

---

## 监控和可观测性

### Prometheus 指标

访问 Prometheus: `http://localhost:9090`

**关键查询：**

```promql
# 请求速率
rate(http_requests_total[1m])

# P99 延迟
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[1m]))

# 缓存命中率
rate(redis_cache_hits_total[1m]) / 
(rate(redis_cache_hits_total[1m]) + rate(redis_cache_misses_total[1m]))

# DB 查询速率
rate(db_queries_total[1m])

# 连接池使用率
redis_pool_active_connections / redis_pool_size
```

### Grafana 仪表板

访问 Grafana: `http://localhost:3000`

**关键面板：**
- 请求吞吐量
- 延迟分位数（P50, P95, P99）
- 缓存命中率
- 连接池状态
- 熔断器状态
- 错误率

---

## 最佳实践

### 1. 测试前准备

- ✅ 确保服务健康
- ✅ 准备测试数据
- ✅ 清理旧的测试数据
- ✅ 检查资源限制
- ✅ 配置监控

### 2. 测试执行

- ✅ 从小负载开始
- ✅ 逐步增加负载
- ✅ 观察系统指标
- ✅ 记录测试结果
- ✅ 保存测试日志

### 3. 结果分析

- ✅ 对比性能基准
- ✅ 分析瓶颈点
- ✅ 识别优化机会
- ✅ 记录改进建议
- ✅ 更新文档

### 4. 持续改进

- ✅ 定期执行压测
- ✅ 跟踪性能趋势
- ✅ 优化配置参数
- ✅ 更新测试脚本
- ✅ 分享最佳实践

---

## 参考资料

- [k6 官方文档](https://k6.io/docs/)
- [k6 最佳实践](https://k6.io/docs/testing-guides/test-types/)
- [单机性能分析](../docs/SINGLE_MACHINE_PERFORMANCE_ANALYSIS.md)
- [服务 README](../README.md)

---

**文档版本:** 2.0  
**最后更新:** 2026-02-04  
**维护者:** 短链服务团队
