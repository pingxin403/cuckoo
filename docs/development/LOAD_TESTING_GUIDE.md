# 项目负载测试指南

## 概述

本指南提供了项目级别的负载测试标准和最佳实践，适用于所有微服务。我们使用 [Grafana k6](https://k6.io/) 作为负载测试工具，它是一个现代化的、开发者友好的性能测试工具。

## 目录

- [为什么选择 k6](#为什么选择-k6)
- [快速开始](#快速开始)
- [测试类型](#测试类型)
- [项目结构](#项目结构)
- [编写测试](#编写测试)
- [执行测试](#执行测试)
- [结果分析](#结果分析)
- [CI/CD 集成](#cicd-集成)
- [最佳实践](#最佳实践)
- [故障排查](#故障排查)

## 为什么选择 k6

k6 是我们选择的负载测试工具，原因如下：

- **开发者友好**：使用 JavaScript/TypeScript 编写测试脚本
- **高性能**：单机可生成大量负载（10万+ QPS）
- **现代化**：支持 HTTP/1.1、HTTP/2、WebSocket、gRPC
- **可扩展**：丰富的指标和自定义扩展
- **云原生**：易于集成到 CI/CD 和 Kubernetes
- **开源**：活跃的社区和丰富的文档

## 快速开始

### 安装 k6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
# Debian/Ubuntu
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


### 验证安装

```bash
k6 version
```

### 第一个测试

创建 `hello-world.js`：

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
};

export default function () {
  const res = http.get('https://test.k6.io');
  check(res, { 'status was 200': (r) => r.status == 200 });
  sleep(1);
}
```

运行测试：

```bash
k6 run hello-world.js
```

## 测试类型

根据不同的测试目标，我们定义了以下标准测试类型：

### 1. Smoke Test（冒烟测试）

**目的**：验证脚本正确性和服务基本功能

**特点**：
- 最小负载（1-10 VUs）
- 短时间（1-5 分钟）
- 快速反馈

**使用场景**：
- 开发阶段验证
- CI/CD 流水线
- 脚本调试

**示例配置**：
```javascript
export const options = {
  vus: 5,
  duration: '1m',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};
```

### 2. Load Test（负载测试）

**目的**：验证系统在预期负载下的性能

**特点**：
- 模拟正常业务负载
- 持续时间较长（10-30 分钟）
- 评估系统稳定性

**使用场景**：
- 性能基准测试
- 容量规划
- SLO 验证

**示例配置**：
```javascript
export const options = {
  stages: [
    { duration: '2m', target: 100 },  // 预热
    { duration: '10m', target: 100 }, // 稳定负载
    { duration: '2m', target: 0 },    // 降压
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200', 'p(99)<500'],
  },
};
```


### 3. Stress Test（压力测试）

**目的**：找到系统的性能极限

**特点**：
- 逐步增加负载
- 超过正常容量
- 识别瓶颈

**使用场景**：
- 容量规划
- 瓶颈分析
- 扩容决策

**示例配置**：
```javascript
export const options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 200 },
    { duration: '5m', target: 300 },
    { duration: '5m', target: 400 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<1000'],
  },
};
```

### 4. Spike Test（峰值测试）

**目的**：测试系统应对突发流量的能力

**特点**：
- 快速增加负载
- 短时间高峰
- 测试恢复能力

**使用场景**：
- 营销活动准备
- 突发事件应对
- 自动扩容验证

**示例配置**：
```javascript
export const options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '1m', target: 1000 },  // 快速冲击
    { duration: '3m', target: 1000 },  // 保持峰值
    { duration: '10s', target: 100 },
    { duration: '3m', target: 100 },   // 恢复观察
  ],
};
```

### 5. Soak Test（浸泡测试）

**目的**：验证系统长时间运行的稳定性

**特点**：
- 中等负载
- 长时间运行（数小时）
- 检测内存泄漏等问题

**使用场景**：
- 生产环境验证
- 内存泄漏检测
- 资源泄漏检测

**示例配置**：
```javascript
export const options = {
  stages: [
    { duration: '5m', target: 200 },
    { duration: '8h', target: 200 },  // 长时间稳定负载
    { duration: '5m', target: 0 },
  ],
};
```


## 项目结构

推荐的负载测试目录结构：

```
<service>/
├── load_test/
│   ├── config/
│   │   ├── environments.js    # 环境配置
│   │   ├── workloads.js       # 负载配置
│   │   └── thresholds.js      # 阈值配置
│   ├── lib/
│   │   ├── api-client.js      # API 客户端封装
│   │   ├── data-generator.js  # 测试数据生成
│   │   ├── helpers.js         # 工具函数
│   │   └── metrics.js         # 自定义指标
│   ├── scenarios/
│   │   ├── create-link.js     # 创建短链场景
│   │   ├── redirect.js        # 重定向场景
│   │   └── mixed-load.js      # 混合负载场景
│   ├── tests/
│   │   ├── smoke.js           # 冒烟测试
│   │   ├── load.js            # 负载测试
│   │   ├── stress.js          # 压力测试
│   │   ├── spike.js           # 峰值测试
│   │   └── soak.js            # 浸泡测试
│   ├── data/
│   │   └── test-codes.json    # 测试数据
│   ├── scripts/
│   │   └── prepare-data.sh    # 数据准备脚本
│   └── README.md              # 服务特定说明
```

### 关键目录说明

- **config/**: 配置文件，支持多环境和多负载模式
- **lib/**: 可复用的库和工具
- **scenarios/**: 测试场景（VU 逻辑），可在多个测试中复用
- **tests/**: 完整的测试脚本，组合场景和配置
- **data/**: 测试数据文件
- **scripts/**: 辅助脚本（数据准备、清理等）

## 编写测试

### 模块化配置

#### 环境配置 (config/environments.js)

```javascript
export const EnvironmentConfig = {
  local: {
    baseUrl: 'http://localhost:8080',
    grpcUrl: 'localhost:50051',
  },
  dev: {
    baseUrl: 'https://dev.example.com',
    grpcUrl: 'dev.example.com:50051',
  },
  staging: {
    baseUrl: 'https://staging.example.com',
    grpcUrl: 'staging.example.com:50051',
  },
  prod: {
    baseUrl: 'https://api.example.com',
    grpcUrl: 'api.example.com:50051',
  },
};

export function getEnvironment() {
  const env = __ENV.ENVIRONMENT || 'local';
  const config = EnvironmentConfig[env];
  
  if (!config) {
    throw new Error(`Unknown environment: ${env}`);
  }
  
  return config;
}
```


#### 负载配置 (config/workloads.js)

```javascript
export const WorkloadConfig = {
  smoke: {
    executor: 'constant-vus',
    vus: 5,
    duration: '1m',
  },
  load: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },
      { duration: '10m', target: 100 },
      { duration: '2m', target: 0 },
    ],
  },
  stress: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '2m', target: 100 },
      { duration: '5m', target: 200 },
      { duration: '5m', target: 300 },
      { duration: '5m', target: 400 },
      { duration: '2m', target: 0 },
    ],
  },
  spike: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 100 },
      { duration: '1m', target: 1000 },
      { duration: '3m', target: 1000 },
      { duration: '10s', target: 100 },
      { duration: '3m', target: 100 },
    ],
  },
  sustained: {
    executor: 'constant-arrival-rate',
    rate: 100000,
    timeUnit: '1s',
    duration: '10m',
    preAllocatedVUs: 1000,
    maxVUs: 2000,
  },
};

export function getWorkload() {
  const workload = __ENV.WORKLOAD || 'smoke';
  const config = WorkloadConfig[workload];
  
  if (!config) {
    throw new Error(`Unknown workload: ${workload}`);
  }
  
  return config;
}
```

#### 阈值配置 (config/thresholds.js)

```javascript
export const CommonThresholds = {
  // HTTP 请求失败率
  http_req_failed: ['rate<0.01'],
  
  // 请求延迟
  http_req_duration: ['p(95)<200', 'p(99)<500'],
  
  // 请求速率（可选）
  http_reqs: ['rate>100'],
};

export const StrictThresholds = {
  http_req_failed: ['rate<0.001'],
  http_req_duration: ['p(95)<100', 'p(99)<200'],
};

export const RelaxedThresholds = {
  http_req_failed: ['rate<0.05'],
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
};

export function getThresholds(type = 'common') {
  const thresholds = {
    common: CommonThresholds,
    strict: StrictThresholds,
    relaxed: RelaxedThresholds,
  };
  
  return thresholds[type] || CommonThresholds;
}
```


### API 客户端封装 (lib/api-client.js)

```javascript
import http from 'k6/http';
import { check } from 'k6';

export class APIClient {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
    this.headers = {
      'Content-Type': 'application/json',
    };
  }

  get(endpoint, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const res = http.get(url, {
      headers: this.headers,
      ...options,
    });
    
    return {
      success: res.status >= 200 && res.status < 300,
      data: this.parseJSON(res),
      response: res,
    };
  }

  post(endpoint, payload, options = {}) {
    const url = `${this.baseUrl}${endpoint}`;
    const res = http.post(url, JSON.stringify(payload), {
      headers: this.headers,
      ...options,
    });
    
    return {
      success: res.status >= 200 && res.status < 300,
      data: this.parseJSON(res),
      response: res,
    };
  }

  parseJSON(res) {
    try {
      return res.json();
    } catch (e) {
      return null;
    }
  }
}
```

### 可复用场景 (scenarios/redirect.js)

```javascript
import { ShortenerAPIClient } from '../lib/api-client.js';
import { selectWithWeight, isCacheHit } from '../lib/helpers.js';

export function redirectScenario(env, data, metrics) {
  const client = new ShortenerAPIClient(env.baseUrl);
  
  // Weighted selection: 80% hot data
  const code = selectWithWeight(data.codes, 0.8, 5);
  
  // Execute request
  const result = client.getRedirect(code);
  
  // Track metrics
  if (result.success) {
    metrics.successfulRedirects.add(1);
    if (isCacheHit(result.response.timings.duration)) {
      metrics.cacheHitsEstimated.add(1);
    }
  }
  
  metrics.errorRate.add(!result.success);
  metrics.latencyTrend.add(result.response.timings.duration);
}
```

### 完整测试示例 (tests/redirect-test.js)

```javascript
import { getEnvironment } from '../config/environments.js';
import { getWorkload } from '../config/workloads.js';
import { getThresholds } from '../config/thresholds.js';
import { redirectScenario } from '../scenarios/redirect.js';
import { createMetrics, formatSummary } from '../lib/helpers.js';

const env = getEnvironment();
const workload = getWorkload();

export const options = {
  scenarios: {
    redirect_test: workload,
  },
  thresholds: getThresholds('redirect'),
};

const metrics = createMetrics();

export function setup() {
  // Prepare test data
  return { codes: ['test001', 'test002', 'test003'] };
}

export default function(data) {
  redirectScenario(env, data, metrics);
}

export function handleSummary(data) {
  console.log(formatSummary(data, 'Redirect Test'));
  return { 'stdout': '' };
}
```


## 执行测试

### 基本用法

```bash
# 使用默认配置（local 环境，smoke 负载）
k6 run tests/redirect-test.js

# 指定环境
k6 run -e ENVIRONMENT=staging tests/redirect-test.js

# 指定负载类型
k6 run -e WORKLOAD=load tests/redirect-test.js

# 组合使用
k6 run -e ENVIRONMENT=prod -e WORKLOAD=stress tests/redirect-test.js
```

### 高级选项

```bash
# 覆盖 VUs 数量
k6 run --vus 100 tests/redirect-test.js

# 覆盖持续时间
k6 run --duration 5m tests/redirect-test.js

# 输出结果到文件
k6 run --out json=results.json tests/redirect-test.js

# 输出到多个目标
k6 run --out json=results.json --out influxdb=http://localhost:8086 tests/redirect-test.js

# 使用标签过滤
k6 run --tag testid=123 --tag env=prod tests/redirect-test.js

# 安静模式（减少输出）
k6 run --quiet tests/redirect-test.js

# 详细模式（更多调试信息）
k6 run --verbose tests/redirect-test.js
```

### 输出格式

k6 支持多种输出格式：

```bash
# JSON 格式
k6 run --out json=results.json tests/redirect-test.js

# CSV 格式
k6 run --out csv=results.csv tests/redirect-test.js

# InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 tests/redirect-test.js

# Prometheus Remote Write
k6 run --out experimental-prometheus-rw tests/redirect-test.js

# Grafana Cloud k6
k6 run --out cloud tests/redirect-test.js
```

### 环境变量

除了 `-e` 标志，还可以使用环境变量：

```bash
export ENVIRONMENT=staging
export WORKLOAD=load
k6 run tests/redirect-test.js
```

### 测试数据准备

在运行测试前，确保准备好测试数据：

```bash
# 运行数据准备脚本
cd apps/shortener-service/load_test
./scripts/prepare-data.sh

# 或手动创建测试数据
grpcurl -plaintext -d '{"long_url": "https://example.com", "custom_code": "test001"}' \
  localhost:50051 api.v1.ShortenerService/CreateShortLink
```


## 结果分析

### k6 输出指标

k6 提供了丰富的内置指标：

#### HTTP 指标

| 指标 | 说明 | 目标值 |
|------|------|--------|
| `http_reqs` | 总请求数 | - |
| `http_req_duration` | 请求延迟 | P95 < 200ms |
| `http_req_failed` | 失败率 | < 1% |
| `http_req_blocked` | 等待连接时间 | < 1ms |
| `http_req_connecting` | 建立连接时间 | < 1ms |
| `http_req_sending` | 发送数据时间 | < 1ms |
| `http_req_waiting` | 等待响应时间 | - |
| `http_req_receiving` | 接收数据时间 | < 1ms |

#### 通用指标

| 指标 | 说明 |
|------|------|
| `vus` | 当前虚拟用户数 |
| `vus_max` | 最大虚拟用户数 |
| `iterations` | 迭代次数 |
| `iteration_duration` | 迭代持续时间 |
| `data_received` | 接收数据量 |
| `data_sent` | 发送数据量 |

### 延迟分析

理解延迟分位数：

- **P50 (中位数)**：50% 的请求延迟低于此值
- **P95**：95% 的请求延迟低于此值
- **P99**：99% 的请求延迟低于此值
- **P99.9**：99.9% 的请求延迟低于此值

**延迟目标参考**：

| 服务类型 | P95 | P99 | P99.9 |
|---------|-----|-----|-------|
| 缓存读取 | < 10ms | < 20ms | < 50ms |
| 数据库查询 | < 100ms | < 200ms | < 500ms |
| API 调用 | < 200ms | < 500ms | < 1000ms |
| 批处理 | < 1s | < 2s | < 5s |

### 性能瓶颈识别

#### 1. 高延迟

**症状**：P95/P99 延迟超过目标值

**可能原因**：
- 数据库查询慢
- 缓存未命中
- 网络延迟
- CPU/内存不足
- 锁竞争

**排查方法**：
```bash
# 查看详细的请求时间分解
k6 run --summary-trend-stats="avg,min,med,max,p(90),p(95),p(99)" tests/redirect-test.js

# 启用详细日志
k6 run --http-debug="full" tests/redirect-test.js
```

#### 2. 高错误率

**症状**：`http_req_failed` > 1%

**可能原因**：
- 服务过载
- 连接池耗尽
- 超时设置不当
- 熔断器触发

**排查方法**：
- 检查服务日志
- 查看监控指标（CPU、内存、连接数）
- 分析错误类型分布

#### 3. 低吞吐量

**症状**：实际 QPS 远低于预期

**可能原因**：
- VUs 数量不足
- 网络带宽限制
- 客户端资源不足
- 服务端限流

**解决方案**：
```bash
# 增加 VUs
k6 run --vus 2000 tests/redirect-test.js

# 使用 constant-arrival-rate executor
# 在 workloads.js 中配置
```


### 自定义指标

创建和使用自定义指标：

```javascript
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';

// 计数器 - 累加值
const cacheHits = new Counter('cache_hits');
cacheHits.add(1);

// 比率 - 百分比
const errorRate = new Rate('error_rate');
errorRate.add(true);  // 错误
errorRate.add(false); // 成功

// 趋势 - 统计分布
const latency = new Trend('custom_latency');
latency.add(100);

// 仪表 - 当前值
const activeConnections = new Gauge('active_connections');
activeConnections.add(50);
```

### 结果可视化

#### 使用 Grafana

1. 配置 InfluxDB 输出：
```bash
k6 run --out influxdb=http://localhost:8086/k6 tests/redirect-test.js
```

2. 在 Grafana 中创建仪表板，查询 InfluxDB 数据

#### 使用 Grafana Cloud k6

```bash
# 设置 API token
export K6_CLOUD_TOKEN=your_token

# 运行测试并上传结果
k6 run --out cloud tests/redirect-test.js
```

#### 生成 HTML 报告

使用 k6-reporter 扩展：

```bash
# 安装
npm install -g k6-reporter

# 生成报告
k6 run --out json=results.json tests/redirect-test.js
k6-reporter results.json
```

## CI/CD 集成

### GitHub Actions

创建 `.github/workflows/load-test.yml`：

```yaml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨 2 点
  workflow_dispatch:      # 手动触发

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
            --keyserver hkp://keyserver.ubuntu.com:80 \
            --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
            sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Run smoke test
        run: |
          cd apps/shortener-service/load_test
          k6 run -e ENVIRONMENT=staging -e WORKLOAD=smoke tests/redirect-test.js
      
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: results.json
```

### GitLab CI

创建 `.gitlab-ci.yml`：

```yaml
load-test:
  image: grafana/k6:latest
  stage: test
  script:
    - cd apps/shortener-service/load_test
    - k6 run -e ENVIRONMENT=staging -e WORKLOAD=smoke tests/redirect-test.js
  artifacts:
    paths:
      - results.json
    expire_in: 1 week
  only:
    - schedules
```


### Jenkins

创建 Jenkinsfile：

```groovy
pipeline {
    agent any
    
    stages {
        stage('Install k6') {
            steps {
                sh '''
                    if ! command -v k6 &> /dev/null; then
                        sudo apt-get update
                        sudo apt-get install -y k6
                    fi
                '''
            }
        }
        
        stage('Run Load Test') {
            steps {
                dir('apps/shortener-service/load_test') {
                    sh 'k6 run -e ENVIRONMENT=staging -e WORKLOAD=load tests/redirect-test.js'
                }
            }
        }
        
        stage('Archive Results') {
            steps {
                archiveArtifacts artifacts: 'results.json', fingerprint: true
            }
        }
    }
    
    post {
        always {
            junit 'results.xml'
        }
    }
}
```

## 最佳实践

### 1. 测试设计原则

#### 从小开始，逐步增加

```javascript
// ❌ 错误：直接高负载
export const options = {
  vus: 10000,
  duration: '1h',
};

// ✅ 正确：逐步增加
export const options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 500 },
    { duration: '5m', target: 1000 },
    { duration: '2m', target: 0 },
  ],
};
```

#### 使用真实数据分布

```javascript
// ❌ 错误：均匀分布
const code = codes[Math.floor(Math.random() * codes.length)];

// ✅ 正确：模拟热点数据（80/20 规则）
const code = Math.random() < 0.8
  ? codes[Math.floor(Math.random() * Math.min(5, codes.length))]
  : codes[Math.floor(Math.random() * codes.length)];
```

#### 设置合理的阈值

```javascript
// ❌ 错误：过于严格或宽松
export const options = {
  thresholds: {
    http_req_duration: ['p(99)<1'],  // 太严格
    http_req_failed: ['rate<0.5'],   // 太宽松
  },
};

// ✅ 正确：基于 SLO 设置
export const options = {
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
};
```

### 2. 代码组织

#### 模块化配置

```javascript
// ✅ 好的做法
import { getEnvironment } from './config/environments.js';
import { getWorkload } from './config/workloads.js';

const env = getEnvironment();
const workload = getWorkload();
```

#### 复用场景

```javascript
// ✅ 好的做法
import { redirectScenario } from './scenarios/redirect.js';

export default function(data) {
  redirectScenario(env, data, metrics);
}
```

#### 封装 API 调用

```javascript
// ✅ 好的做法
const client = new APIClient(baseUrl);
const result = client.getRedirect(code);
```

### 3. 性能优化

#### 避免不必要的 sleep

```javascript
// ❌ 错误：固定 sleep
export default function() {
  http.get(url);
  sleep(1);  // 限制了 QPS
}

// ✅ 正确：使用 arrival-rate executor
export const options = {
  scenarios: {
    test: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
    },
  },
};
```

#### 复用连接

```javascript
// ✅ k6 默认复用 HTTP 连接
// 确保不要在每次请求时创建新的客户端
```

#### 批量操作

```javascript
// ✅ 使用 batch 并行请求
import { batch } from 'k6/http';

export default function() {
  const responses = batch([
    ['GET', 'https://api.example.com/endpoint1'],
    ['GET', 'https://api.example.com/endpoint2'],
    ['GET', 'https://api.example.com/endpoint3'],
  ]);
}
```


### 4. 监控和可观测性

#### 连接测试结果与监控

```javascript
// 在测试中添加 trace ID
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export default function() {
  const traceId = randomString(16);
  
  http.get(url, {
    headers: {
      'X-Trace-ID': traceId,
    },
  });
}
```

#### 使用标签分类请求

```javascript
export default function() {
  http.get(url, {
    tags: {
      name: 'redirect',
      type: 'read',
      cache: 'hit',
    },
  });
}
```

#### 记录关键事件

```javascript
import { Counter } from 'k6/metrics';

const criticalErrors = new Counter('critical_errors');

export default function() {
  const res = http.get(url);
  
  if (res.status === 500) {
    criticalErrors.add(1);
    console.error(`Critical error: ${res.body}`);
  }
}
```

### 5. 测试数据管理

#### 使用 SharedArray 共享数据

```javascript
import { SharedArray } from 'k6/data';

// ✅ 好的做法：所有 VUs 共享数据，节省内存
const data = new SharedArray('test codes', function() {
  return JSON.parse(open('./data/codes.json'));
});

export default function() {
  const code = data[Math.floor(Math.random() * data.length)];
  http.get(`${baseUrl}/${code}`);
}
```

#### 动态生成测试数据

```javascript
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export default function() {
  const longUrl = `https://example.com/${randomString(10)}`;
  http.post(url, JSON.stringify({ long_url: longUrl }));
}
```

### 6. 错误处理

#### 优雅处理错误

```javascript
export default function() {
  const res = http.get(url);
  
  if (!check(res, { 'status is 200': (r) => r.status === 200 })) {
    console.error(`Request failed: ${res.status} ${res.body}`);
    return;  // 继续下一次迭代
  }
  
  // 处理成功响应
}
```

#### 区分错误类型

```javascript
import { Counter } from 'k6/metrics';

const clientErrors = new Counter('client_errors');  // 4xx
const serverErrors = new Counter('server_errors');  // 5xx
const timeoutErrors = new Counter('timeout_errors');

export default function() {
  const res = http.get(url, { timeout: '5s' });
  
  if (res.status >= 400 && res.status < 500) {
    clientErrors.add(1);
  } else if (res.status >= 500) {
    serverErrors.add(1);
  } else if (res.error_code === 1050) {  // timeout
    timeoutErrors.add(1);
  }
}
```

## 故障排查

### 常见问题

#### 1. 连接被拒绝

**错误信息**：
```
WARN[0001] Request Failed error="dial tcp: connect: connection refused"
```

**解决方案**：
- 检查服务是否启动
- 验证 URL 和端口是否正确
- 检查防火墙设置

```bash
# 验证服务可访问
curl http://localhost:8080/health

# 检查端口监听
lsof -i :8080
```

#### 2. 文件描述符不足

**错误信息**：
```
WARN[0010] Request Failed error="dial tcp: socket: too many open files"
```

**解决方案**：
```bash
# 检查当前限制
ulimit -n

# 临时增加限制
ulimit -n 65535

# 永久修改（Linux）
echo "* soft nofile 65535" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65535" | sudo tee -a /etc/security/limits.conf
```


#### 3. VUs 不足

**症状**：实际 QPS 低于目标

**解决方案**：
```javascript
// 增加 preAllocatedVUs 和 maxVUs
export const options = {
  scenarios: {
    test: {
      executor: 'constant-arrival-rate',
      rate: 100000,
      timeUnit: '1s',
      preAllocatedVUs: 2000,  // 增加
      maxVUs: 5000,           // 增加
    },
  },
};
```

#### 4. 内存不足

**症状**：k6 进程被 OOM killer 终止

**解决方案**：
- 使用 SharedArray 共享数据
- 减少 VUs 数量
- 使用分布式测试

```javascript
// ✅ 使用 SharedArray
import { SharedArray } from 'k6/data';

const data = new SharedArray('codes', function() {
  return JSON.parse(open('./data/codes.json'));
});
```

#### 5. 测试结果不稳定

**可能原因**：
- 预热时间不足
- 测试时间太短
- 外部因素干扰

**解决方案**：
```javascript
export const options = {
  stages: [
    { duration: '5m', target: 100 },   // 充分预热
    { duration: '30m', target: 100 },  // 足够长的稳定期
    { duration: '2m', target: 0 },
  ],
};
```

### 调试技巧

#### 启用详细日志

```bash
# HTTP 请求详情
k6 run --http-debug tests/redirect-test.js

# 完整 HTTP 请求/响应
k6 run --http-debug="full" tests/redirect-test.js

# 详细输出
k6 run --verbose tests/redirect-test.js
```

#### 使用 console.log

```javascript
export default function() {
  const res = http.get(url);
  
  // 调试输出
  console.log(`Status: ${res.status}`);
  console.log(`Duration: ${res.timings.duration}ms`);
  console.log(`Body: ${res.body.substring(0, 100)}`);
}
```

#### 检查指标

```javascript
import { Counter } from 'k6/metrics';

const debugCounter = new Counter('debug_counter');

export default function() {
  debugCounter.add(1);
  
  if (__ITER % 100 === 0) {
    console.log(`Iteration: ${__ITER}, VU: ${__VU}`);
  }
}
```

## 参考资料

### 官方文档

- [k6 官方文档](https://k6.io/docs/)
- [k6 API 参考](https://k6.io/docs/javascript-api/)
- [k6 最佳实践](https://k6.io/docs/testing-guides/test-types/)
- [k6 示例](https://k6.io/docs/examples/)

### 社区资源

- [k6 GitHub](https://github.com/grafana/k6)
- [k6 社区论坛](https://community.grafana.com/c/grafana-k6/)
- [Awesome k6](https://github.com/grafana/awesome-k6)

### 相关文档

- [项目负载测试指南](./LOAD_TESTING_GUIDE.md)（本文档）
- [性能测试指南](./TESTING_GUIDE.md)
- [监控和告警指南](../operations/MONITORING_ALERTING_GUIDE.md)

### 工具和扩展

- [k6-reporter](https://github.com/benc-uk/k6-reporter) - HTML 报告生成
- [k6-to-junit](https://github.com/smockle/k6-to-junit) - JUnit XML 转换
- [xk6](https://github.com/grafana/xk6) - k6 扩展构建工具

---

**文档版本**: 1.0  
**最后更新**: 2026-02-04  
**维护者**: 平台工程团队

## 附录

### A. 测试类型对比

| 测试类型 | 负载模式 | 持续时间 | 主要目的 | 使用场景 |
|---------|---------|---------|---------|---------|
| Smoke | 最小 | 1-5分钟 | 验证功能 | 开发、CI/CD |
| Load | 正常 | 10-30分钟 | 性能基准 | 容量规划、SLO验证 |
| Stress | 递增 | 20-60分钟 | 找到极限 | 瓶颈分析、扩容决策 |
| Spike | 突增 | 5-15分钟 | 突发应对 | 营销活动、自动扩容 |
| Soak | 中等 | 数小时 | 长期稳定性 | 内存泄漏检测 |

### B. 常用 k6 命令速查

```bash
# 基本运行
k6 run script.js

# 指定 VUs 和持续时间
k6 run --vus 100 --duration 5m script.js

# 使用环境变量
k6 run -e KEY=value script.js

# 输出到文件
k6 run --out json=results.json script.js

# 查看版本
k6 version

# 查看帮助
k6 run --help
```

### C. 性能目标参考

| 指标 | 优秀 | 良好 | 可接受 | 需改进 |
|------|------|------|--------|--------|
| P95 延迟 | < 100ms | < 200ms | < 500ms | > 500ms |
| P99 延迟 | < 200ms | < 500ms | < 1000ms | > 1000ms |
| 错误率 | < 0.1% | < 1% | < 5% | > 5% |
| 可用性 | > 99.99% | > 99.9% | > 99% | < 99% |

