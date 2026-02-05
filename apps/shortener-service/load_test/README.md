# 短链服务负载测试

本目录包含短链服务的 k6 负载测试脚本，遵循项目级别的负载测试标准。

## 快速开始

### 1. 安装 k6

```bash
# macOS
brew install k6

# Linux
sudo apt-get install k6

# Windows
choco install k6
```

### 2. 准备测试数据

```bash
# 启动服务
cd apps/shortener-service
go run main.go

# 准备测试数据
cd load_test
./scripts/prepare-data.sh
```

### 3. 运行测试

```bash
# 冒烟测试（默认）
k6 run tests/redirect-test.js

# 负载测试
k6 run -e WORKLOAD=load tests/redirect-test.js

# 压力测试
k6 run -e WORKLOAD=stress tests/redirect-test.js

# 指定环境
k6 run -e ENVIRONMENT=staging -e WORKLOAD=load tests/redirect-test.js
```

## 目录结构

```
load_test/
├── config/              # 配置文件
│   ├── environments.js  # 环境配置（local, dev, staging, prod）
│   ├── workloads.js     # 负载配置（smoke, load, stress, spike, sustained）
│   └── thresholds.js    # 阈值配置
├── lib/                 # 可复用库
│   ├── api-client.js    # API 客户端封装
│   └── helpers.js       # 工具函数
├── tests/               # 测试脚本
│   └── redirect-test.js # 重定向性能测试
├── scripts/             # 辅助脚本
│   └── prepare-data.sh  # 数据准备脚本
└── README.md           # 本文件
```

## 可用的测试

### redirect-test.js - 重定向性能测试

测试短链重定向功能的性能。

**使用方法**：
```bash
# 冒烟测试（5 VUs, 1分钟）
k6 run tests/redirect-test.js

# 负载测试（100 VUs, 14分钟）
k6 run -e WORKLOAD=load tests/redirect-test.js

# 压力测试（逐步增加到 400 VUs）
k6 run -e WORKLOAD=stress tests/redirect-test.js

# 峰值测试（快速冲击到 1000 VUs）
k6 run -e WORKLOAD=spike tests/redirect-test.js

# 持续高负载（100K QPS, 10分钟）
k6 run -e WORKLOAD=sustained tests/redirect-test.js
```

**性能目标**：
- P95 延迟: < 10ms
- P99 延迟: < 20ms
- 错误率: < 1%
- 缓存命中率: > 90%

## 配置说明

### 环境配置

在 `config/environments.js` 中定义：

- `local`: 本地开发环境（默认）
- `dev`: 开发环境
- `staging`: 预发布环境
- `prod`: 生产环境

### 负载配置

在 `config/workloads.js` 中定义：

- `smoke`: 冒烟测试（5 VUs, 1分钟）
- `load`: 负载测试（100 VUs, 14分钟）
- `stress`: 压力测试（100-400 VUs, 19分钟）
- `spike`: 峰值测试（100-1000 VUs, 7分钟）
- `sustained`: 持续高负载（100K QPS, 10分钟）

### 阈值配置

在 `config/thresholds.js` 中定义：

- `common`: 通用阈值
- `strict`: 严格阈值
- `relaxed`: 宽松阈值
- `redirect`: 重定向专用阈值
- `create`: 创建链接专用阈值

## 高级用法

### 自定义参数

```bash
# 覆盖 VUs 数量
k6 run --vus 200 tests/redirect-test.js

# 覆盖持续时间
k6 run --duration 10m tests/redirect-test.js

# 组合使用
k6 run -e ENVIRONMENT=prod -e WORKLOAD=load --vus 500 tests/redirect-test.js
```

### 输出结果

```bash
# 输出到 JSON 文件
k6 run --out json=results.json tests/redirect-test.js

# 输出到 InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 tests/redirect-test.js

# 输出到 Grafana Cloud
k6 run --out cloud tests/redirect-test.js
```

### 调试

```bash
# 详细 HTTP 日志
k6 run --http-debug tests/redirect-test.js

# 完整 HTTP 请求/响应
k6 run --http-debug="full" tests/redirect-test.js

# 详细输出
k6 run --verbose tests/redirect-test.js
```

## 性能基准

### 单机性能（8核16GB）

| 场景 | QPS | P99 延迟 | 缓存命中率 | 错误率 |
|------|-----|---------|-----------|--------|
| 纯重定向（高缓存） | 150K-180K | < 5ms | > 95% | < 0.1% |
| 混合负载（80读20写） | 100K-120K | < 10ms | > 90% | < 0.1% |

## 故障排查

### 连接被拒绝

```bash
# 检查服务状态
curl http://localhost:8080/health

# 检查端口
lsof -i :8080
```

### 测试数据不存在

```bash
# 重新准备数据
./scripts/prepare-data.sh
```

### 低吞吐量

```bash
# 增加 VUs
k6 run --vus 2000 tests/redirect-test.js

# 检查资源限制
ulimit -n
ulimit -n 65535
```

## 参考文档

- [项目负载测试指南](../../../docs/development/LOAD_TESTING_GUIDE.md)
- [k6 官方文档](https://k6.io/docs/)
- [短链服务 README](../README.md)

---

**维护者**: 短链服务团队  
**最后更新**: 2026-02-04
