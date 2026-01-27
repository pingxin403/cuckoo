# 可观测性系统架构设计

## 概述

本文档描述了 Monorepo 中统一可观测性库 (`libs/observability`) 的架构设计。该库提供基于 OpenTelemetry 的指标、日志和追踪能力，同时支持 Prometheus 指标导出和 pprof 性能分析端点。

## 系统架构

### 高层架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         Services Layer                           │
├─────────────┬─────────────┬─────────────┬─────────────┬────────┤
│ auth-service│ user-service│ todo-service│  im-service │shortener│
│             │             │             │             │ -service│
└──────┬──────┴──────┬──────┴──────┬──────┴──────┬──────┴────┬───┘
       │             │             │             │            │
       └─────────────┴─────────────┴─────────────┴────────────┘
                              │
                    ┌─────────▼──────────┐
                    │  Observability Lib │
                    │  (libs/observability)│
                    └─────────┬──────────┘
                              │
       ┌──────────────────────┼──────────────────────┐
       │                      │                      │
┌──────▼──────┐      ┌───────▼────────┐    ┌───────▼────────┐
│  Prometheus │      │ OTLP Collector │    │  pprof Endpoints│
│  (pull)     │      │  (push)        │    │  (on-demand)   │
└─────────────┘      └────────┬───────┘    └────────────────┘
                              │
                    ┌─────────┴──────────┐
                    │                    │
            ┌───────▼────────┐  ┌───────▼────────┐
            │    Jaeger      │  │   Prometheus   │
            │   (traces)     │  │   (metrics)    │
            └────────────────┘  └────────────────┘
```

### 组件交互

1. **服务初始化**: 每个服务在启动时初始化可观测性库
2. **指标收集**: 服务通过库的指标接口发送指标
3. **日志发送**: 服务通过库的结构化日志器记录日志
4. **追踪创建**: 服务为操作创建追踪 span
5. **双重导出**: 指标同时导出到 Prometheus (pull) 和 OTLP (push)
6. **优雅关闭**: 服务在终止前刷新遥测数据

## OpenTelemetry 集成架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Observability Library                     │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Metrics    │  │   Logging    │  │   Tracing    │      │
│  │  (Collector) │  │   (Logger)   │  │   (Tracer)   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│         ▼                  ▼                  ▼              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ OTel Metrics │  │  OTel Logs   │  │ OTel Traces  │      │
│  │     SDK      │  │     SDK      │  │     SDK      │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│         ├──────────────────┼──────────────────┤              │
│         │                  │                  │              │
│         ▼                  ▼                  ▼              │
│  ┌──────────────────────────────────────────────────┐       │
│  │          OpenTelemetry Resource                  │       │
│  │  (service.name, service.version, environment)    │       │
│  └──────────────────────────────────────────────────┘       │
│         │                  │                  │              │
│         ▼                  ▼                  ▼              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ OTLP Metrics │  │  OTLP Logs   │  │ OTLP Traces  │      │
│  │   Exporter   │  │   Exporter   │  │   Exporter   │      │
│  └──────┬───────┘  └──────────────┘  └──────────────┘      │
│         │                                                    │
│         ▼                                                    │
│  ┌──────────────┐                                           │
│  │  Prometheus  │                                           │
│  │   Exporter   │                                           │
│  └──────────────┘                                           │
│                                                               │
│  ┌──────────────────────────────────────────────────┐       │
│  │              HTTP Server                          │       │
│  │  - /metrics (Prometheus)                         │       │
│  │  - /debug/pprof/* (pprof endpoints)              │       │
│  │  - /health                                        │       │
│  └──────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

## 核心接口

### Observability 主接口

```go
type Observability interface {
    Metrics() metrics.Collector
    Tracer() tracing.Tracer
    Logger() logging.Logger
    Shutdown(ctx context.Context) error
}
```

### Metrics 接口

```go
type Collector interface {
    IncrementCounter(name string, labels map[string]string)
    SetGauge(name string, value float64, labels map[string]string)
    RecordHistogram(name string, value float64, labels map[string]string)
    RecordDuration(name string, duration time.Duration, labels map[string]string)
    Handler() http.Handler
}
```

### Tracing 接口

```go
type Tracer interface {
    StartSpan(ctx context.Context, name string) (context.Context, Span)
    Shutdown(ctx context.Context) error
}
```

### Logging 接口

```go
type Logger interface {
    Debug(ctx context.Context, msg string, keysAndValues ...interface{})
    Info(ctx context.Context, msg string, keysAndValues ...interface{})
    Warn(ctx context.Context, msg string, keysAndValues ...interface{})
    Error(ctx context.Context, msg string, keysAndValues ...interface{})
    With(keysAndValues ...interface{}) Logger
    Sync() error
}
```

## 配置

### 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVICE_NAME` | 服务特定 | 服务名称 |
| `SERVICE_VERSION` | "1.0.0" | 服务版本 |
| `DEPLOYMENT_ENVIRONMENT` | "development" | 部署环境 |
| `ENABLE_METRICS` | true | 启用指标 |
| `METRICS_PORT` | 9090 | 指标端口 |
| `USE_OTEL_METRICS` | false | 使用 OTel Metrics SDK |
| `PROMETHEUS_ENABLED` | true | 启用 Prometheus 导出 |
| `ENABLE_TRACING` | false | 启用追踪 |
| `TRACING_ENDPOINT` | "" | OTLP 追踪端点 |
| `LOG_LEVEL` | "info" | 日志级别 |
| `LOG_FORMAT` | "json" | 日志格式 |
| `USE_OTEL_LOGS` | false | 使用 OTel Logs SDK |
| `ENABLE_PPROF` | false | 启用 pprof |
| `OTLP_ENDPOINT` | "" | 统一 OTLP 端点 |

### 配置示例

```go
config := observability.Config{
    ServiceName:       "my-service",
    ServiceVersion:    "1.0.0",
    Environment:       "production",
    EnableMetrics:     true,
    MetricsPort:       9090,
    UseOTelMetrics:    true,
    PrometheusEnabled: true,
    EnableTracing:     true,
    TracingEndpoint:   "otel-collector:4317",
    LogLevel:          "info",
    LogFormat:         "json",
    EnablePprof:       false,
}
```

## 追踪-日志关联

当在活跃的 span 上下文中创建日志条目时，日志记录会自动包含 `trace_id` 和 `span_id`：

```go
func (l *OTelLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
    record := log.Record{}
    record.SetTimestamp(time.Now())
    record.SetSeverity(log.SeverityInfo)
    record.SetBody(log.StringValue(msg))
    
    // 如果可用，添加追踪上下文
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        spanCtx := span.SpanContext()
        record.SetTraceID(spanCtx.TraceID())
        record.SetSpanID(spanCtx.SpanID())
        record.SetTraceFlags(spanCtx.TraceFlags())
    }
    
    // 添加键值对作为属性
    attrs := l.fields
    attrs = append(attrs, kvPairsToAttributes(keysAndValues...)...)
    record.SetAttributes(attrs...)
    
    l.logger.Emit(ctx, record)
}
```

## pprof 性能分析

### 可用端点

| 端点 | 说明 |
|------|------|
| `/debug/pprof/` | pprof 索引页 |
| `/debug/pprof/profile` | CPU 分析 |
| `/debug/pprof/heap` | 堆内存分析 |
| `/debug/pprof/goroutine` | Goroutine 分析 |
| `/debug/pprof/block` | 阻塞分析 |
| `/debug/pprof/mutex` | 互斥锁分析 |

### 使用示例

```bash
# CPU 分析 (30秒)
go tool pprof http://localhost:9090/debug/pprof/profile?seconds=30

# 堆内存分析
go tool pprof http://localhost:9090/debug/pprof/heap

# Goroutine 分析
go tool pprof http://localhost:9090/debug/pprof/goroutine
```

## 错误处理

### 初始化错误

当可观测性初始化失败时：
1. 将错误记录到 stderr
2. 创建 no-op 可观测性实例
3. 服务继续运行（降级模式）
4. 不会崩溃或服务失败

### 导出错误

当指标/日志/追踪导出失败时：
1. 通过可观测性日志器记录错误
2. 继续在内存中收集数据（有限制）
3. 在下一个间隔重试导出
4. 如果超出内存限制则丢弃旧数据

### 关闭错误

当可观测性关闭超时时：
1. 记录超时警告
2. 强制关闭可观测性组件
3. 服务继续关闭流程
4. 不会阻塞或挂起

## 相关文档

- 可观测性库: `libs/observability/`
- OTel 增强设计: `.kiro/specs/observability-otel-enhancement/design.md`
- 集成设计: `.kiro/specs/observability-integration/design.md`
- 监控告警指南: `docs/operations/MONITORING_ALERTING_GUIDE.md`
