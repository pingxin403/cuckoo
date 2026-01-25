# Observability Stack Deployment - Complete ✅

## 概述

已成功为 monorepo 创建完整的可观测性栈部署配置，包括 Docker Compose（本地开发）和 Kubernetes（生产环境）两种部署方式。

## 已创建的文件

### Docker Compose 配置（本地开发）

```
deploy/docker/
├── docker-compose.observability.yml    # 主配置文件
├── otel-collector-config.yaml          # OpenTelemetry Collector 配置
├── prometheus.yml                      # Prometheus 配置
├── loki-config.yaml                    # Loki 配置
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   └── datasources.yml        # 数据源自动配置
│   │   └── dashboards/
│   │       └── dashboards.yml         # 仪表板自动配置
│   └── dashboards/
│       └── service-overview.json      # 示例仪表板
├── OBSERVABILITY.md                    # 完整文档
├── QUICK_START_OBSERVABILITY.md        # 快速入门指南
└── README.md                           # 已更新
```

### Kubernetes 配置（生产环境）

```
deploy/k8s/observability/
├── namespace.yaml                      # 命名空间
├── otel-collector.yaml                 # OpenTelemetry Collector 部署
├── jaeger.yaml                         # Jaeger 部署
├── prometheus.yaml                     # Prometheus 部署
├── grafana.yaml                        # Grafana 部署
├── loki.yaml                           # Loki 部署
└── README.md                           # Kubernetes 部署指南
```

### 脚本和工具

```
scripts/
└── test-observability-stack.sh         # 测试脚本

Makefile                                 # 已添加可观测性目标
```

### 文档

```
deploy/
├── OBSERVABILITY_DEPLOYMENT_SUMMARY.md  # 部署总结
└── OBSERVABILITY_COMPLETE.md            # 本文件
```

## 组件架构

```
┌─────────────────────────────────────────────────────────────┐
│                      应用服务                                 │
│  (shortener-service, im-service, im-gateway-service, etc.)  │
└────────────────────┬────────────────────────────────────────┘
                     │ OTLP (gRPC: 4317, HTTP: 4318)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              OpenTelemetry Collector                         │
│  - 接收: Traces, Metrics, Logs                              │
│  - 处理: 批处理, 过滤, 增强                                  │
│  - 导出: 到多个后端                                          │
└─────┬──────────────┬──────────────┬─────────────────────────┘
      │              │              │
      ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│  Jaeger  │  │Prometheus│  │   Loki   │
│ (链路追踪)│  │ (指标)   │  │  (日志)  │
│  :16686  │  │  :9090   │  │  :3100   │
└──────────┘  └──────────┘  └──────────┘
      │              │              │
      └──────────────┴──────────────┘
                     │
                     ▼
              ┌──────────┐
              │ Grafana  │
              │ (可视化) │
              │   :3000  │
              └──────────┘
```

## 快速开始

### 1. 启动可观测性栈

```bash
# 使用 Makefile（推荐）
make observability-up

# 或直接使用 docker compose
docker compose -f deploy/docker/docker-compose.observability.yml up -d
```

### 2. 验证部署

```bash
# 检查所有服务状态
make observability-status

# 运行测试脚本
./scripts/test-observability-stack.sh
```

### 3. 访问 UI

- **Grafana**: http://localhost:3000
  - 用户名: `admin`
  - 密码: `admin`

- **Jaeger**: http://localhost:16686

- **Prometheus**: http://localhost:9090

### 4. 配置服务发送遥测数据

#### Go 服务（使用 observability 库）

```go
import "github.com/pingxin403/cuckoo/libs/observability"

config := observability.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Environment:    "development",
    
    // OpenTelemetry 配置
    OTLPEndpoint:   "localhost:4317",
    UseOTelMetrics: true,
    UseOTelLogs:    true,
    EnableTracing:  true,
    EnableMetrics:  true,
}

obs, err := observability.New(config)
if err != nil {
    log.Fatal(err)
}
defer obs.Shutdown(context.Background())
```

#### 环境变量配置

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_SERVICE_NAME="my-service"
export OTEL_RESOURCE_ATTRIBUTES="deployment.environment=development"
```

## Makefile 命令

```bash
# 启动可观测性栈
make observability-up

# 停止可观测性栈
make observability-down

# 重启可观测性栈
make observability-restart

# 查看日志
make observability-logs

# 检查状态
make observability-status

# 清理数据（警告：删除所有数据！）
make observability-clean
```

## 组件说明

### OpenTelemetry Collector

**端口**:
- 4317: OTLP gRPC 接收器
- 4318: OTLP HTTP 接收器
- 8888: Collector 自身指标
- 8889: Prometheus 导出器
- 13133: 健康检查

**功能**:
- 接收来自服务的遥测数据
- 批处理和内存限制
- 资源属性增强
- 导出到 Jaeger、Prometheus、Loki

### Jaeger

**端口**: 16686 (UI)

**功能**:
- 分布式链路追踪
- 服务依赖图
- 链路搜索和过滤
- Span 详情和日志

### Prometheus

**端口**: 9090 (UI/API)

**功能**:
- 指标存储和查询
- PromQL 查询语言
- 15 天数据保留
- 服务发现（Kubernetes）

### Grafana

**端口**: 3000 (UI)

**功能**:
- 预配置数据源（Prometheus、Jaeger、Loki）
- 示例仪表板
- 链路到日志关联
- 日志到链路关联

### Loki

**端口**: 3100 (API)

**功能**:
- 日志聚合
- LogQL 查询语言
- 7 天数据保留
- 基于标签的索引

## 常用查询

### Prometheus (指标)

```promql
# 请求速率
rate(http_requests_total[5m])

# 错误率
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])

# P95 延迟
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### LogQL (日志)

```logql
# 服务的所有日志
{service_name="my-service"}

# 仅错误日志
{service_name="my-service"} |= "error"

# 带特定 trace ID 的日志
{service_name="my-service"} | json | trace_id="abc123"
```

## 存储配置

### Docker Compose

- Prometheus: `prometheus-data` 卷
- Grafana: `grafana-data` 卷
- Loki: `loki-data` 卷

### Kubernetes

- Prometheus: 50Gi PersistentVolumeClaim
- Grafana: 10Gi PersistentVolumeClaim
- Loki: 50Gi PersistentVolumeClaim

## 资源需求

### Docker Compose（最小）

- CPU: 2 核心
- 内存: 4GB
- 磁盘: 20GB

### Kubernetes（生产）

| 组件 | CPU 请求 | 内存请求 | CPU 限制 | 内存限制 |
|------|----------|----------|----------|----------|
| OTel Collector | 200m | 512Mi | 1000m | 2Gi |
| Jaeger | 100m | 256Mi | 500m | 1Gi |
| Prometheus | 500m | 2Gi | 2000m | 8Gi |
| Grafana | 100m | 256Mi | 500m | 1Gi |
| Loki | 200m | 512Mi | 1000m | 2Gi |

## 故障排查

### 服务无法启动

```bash
# 查看日志
make observability-logs

# 检查容器状态
docker compose -f deploy/docker/docker-compose.observability.yml ps

# 检查端口占用
lsof -i :3000  # Grafana
lsof -i :16686 # Jaeger
lsof -i :9090  # Prometheus
```

### 没有数据显示

1. **检查 OpenTelemetry Collector 是否接收数据**:
   ```bash
   docker logs otel-collector
   ```

2. **验证服务发送到正确的端点**:
   ```bash
   curl -v http://localhost:4318/v1/traces
   ```

3. **检查服务日志中的错误**

### 运行测试脚本

```bash
./scripts/test-observability-stack.sh
```

## Kubernetes 部署

### 部署到 Kubernetes

```bash
# 创建命名空间
kubectl apply -f deploy/k8s/observability/namespace.yaml

# 部署所有组件
kubectl apply -f deploy/k8s/observability/

# 检查状态
kubectl get pods -n observability

# 端口转发访问
kubectl port-forward -n observability svc/grafana 3000:80
kubectl port-forward -n observability svc/jaeger-ui 16686:80
kubectl port-forward -n observability svc/prometheus 9090:9090
```

### 服务配置（Kubernetes）

```yaml
# 在 Deployment 中配置环境变量
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://otel-collector.observability.svc.cluster.local:4317"
  - name: OTEL_SERVICE_NAME
    value: "my-service"
```

## 安全考虑

### Docker Compose（开发环境）

- ⚠️ 使用默认密码（生产环境需更改！）
- ⚠️ 无 TLS（仅用于本地开发）
- ⚠️ 大多数服务无认证

### Kubernetes（生产环境）

- ✅ 更改默认 Grafana 密码
- ✅ 为所有服务启用 TLS
- ✅ 配置 RBAC
- ✅ 使用 Secrets 存储凭据
- ✅ 启用认证
- ✅ 配置网络策略

## 生产检查清单

- [ ] 更改默认密码
- [ ] 为所有服务启用 TLS
- [ ] 配置持久化存储
- [ ] 设置备份和恢复
- [ ] 配置资源限制
- [ ] 启用认证
- [ ] 设置告警
- [ ] 配置日志保留策略
- [ ] 启用 RBAC（Kubernetes）
- [ ] 监控可观测性栈本身
- [ ] 编写运维手册
- [ ] 测试灾难恢复流程

## 文档链接

- [Docker Compose 指南](./docker/OBSERVABILITY.md)
- [快速入门指南](./docker/QUICK_START_OBSERVABILITY.md)
- [Kubernetes 指南](./k8s/observability/README.md)
- [Observability 库](../libs/observability/README.md)
- [OpenTelemetry 指南](../libs/observability/OPENTELEMETRY_GUIDE.md)
- [迁移指南](../libs/observability/MIGRATION_GUIDE.md)

## 下一步

1. ✅ 启动可观测性栈: `make observability-up`
2. ✅ 运行测试脚本: `./scripts/test-observability-stack.sh`
3. ✅ 配置服务发送遥测数据
4. ✅ 访问 UI 并探索数据
5. ✅ 创建自定义仪表板
6. ✅ 设置告警规则
7. ✅ 部署到 Kubernetes（生产环境）

## 总结

✅ **完成！** 可观测性栈已完全配置并可用于：

- **本地开发**: 使用 Docker Compose
- **生产环境**: 使用 Kubernetes
- **完整的遥测**: 指标、链路追踪、日志
- **统一收集**: OpenTelemetry Collector
- **强大的可视化**: Grafana 仪表板
- **分布式追踪**: Jaeger
- **指标存储**: Prometheus
- **日志聚合**: Loki

可观测性栈提供了全面的监控、追踪和日志记录能力，使用业界标准的开源工具。

---

**创建时间**: 2026-01-24  
**状态**: ✅ 完成  
**维护者**: DevOps Team
