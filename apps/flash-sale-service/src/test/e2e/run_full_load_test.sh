# Full链路压测测试计划 (End-to-End Load Test Plan)

## 1. 测试目标

验证 Flash Sale System 在 100K QPS 压力下的性能表现

## 2. 测试环境

### 2.1 基础设施配置

| 组件 | 配置 | 数量 |
|------|------|------|
| Redis | r6g.xlarge (4 vCPU, 30GB) | 3 (1 master + 2 replica) |
| Kafka | m6i.xlarge (4 vCPU, 16GB) | 5 (100 partitions) |
| MySQL | r6g.2xlarge (8 vCPU, 64GB) | 2 (1 master + 1 replica) |
| Higress | c6i.xlarge (4 vCPU, 8GB) | 2 |
| Application | c6i.2xlarge (8 vCPU, 16GB) | 4 |

### 2.2 测试工具

- JMeter: `apps/flash-sale-service/src/test/jmeter/flash_sale_load_test.jmx`
- Redis Benchmark: `apps/flash-sale-service/src/test/benchmark/redis_benchmark.sh`
- Chaos Control: `apps/flash-sale-service/src/test/chaos/chaos_control.sh`

## 3. 测试场景

### 3.1 场景1: 基线性能测试

```bash
# 1. Redis 基线
./redis_benchmark.sh

# 2. 单服务基线
jmeter -n -t flash_sale_load_test.jmx -l results.jtl -e -o report
```

### 3.2 场景2: 峰值流量测试

```bash
# 100K QPS, 持续 30 分钟
jmeter -n \
  -t flash_sale_load_test.jmx \
  -JTHREADS=1000 \
  -JRAMP_UP=60 \
  -JDURATION=1800 \
  -l peak_test.jtl
```

### 3.3 场景3: 混沌工程测试

```bash
# 测试 Redis 故障恢复
./chaos_control.sh redis kill
# 观察系统降级行为
# 恢复后验证数据一致性

# 测试 Kafka 故障
./chaos_control.sh kafka kill
# 验证消息堆积和恢复

# 测试网络延迟
./chaos_control.sh redis latency 1000
# 验证超时处理
```

### 3.4 场景4: 全链路压测

```bash
#!/bin/bash

set -e

BASE_URL="${BASE_URL:-http://localhost:8084}"
DURATION="${DURATION:-600}"
CONCURRENT_USERS="${CONCURRENT_USERS:-1000}"

echo "=== Full链路压测开始 ==="
echo "Target: $BASE_URL"
echo "Duration: ${DURATION}s"
echo "Concurrent Users: $CONCURRENT_USERS"

# 预热
echo "预热阶段..."
for i in {1..10}; do
  curl -s "$BASE_URL/actuator/health" > /dev/null || true
  sleep 1
done

# 压测
echo "开始压测..."
jmeter -n \
  -t flash_sale_load_test.jmx \
  -JTHREADS=$CONCURRENT_USERS \
  -JRAMP_UP=60 \
  -JDURATION=$DURATION \
  -l full_test.jtl

# 生成报告
echo "生成报告..."
jmeter -g full_test.jtl -o test-report

echo "=== 压测完成 ==="
echo "Report: test-report/index.html"