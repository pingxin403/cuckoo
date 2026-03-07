# Capacity Monitoring and Lifecycle Management

容量监控与数据生命周期管理组件，用于多地域架构的资源监控和数据归档。

## Features

### 容量监控 (Capacity Monitoring)

- **资源采集**: 支持 MySQL、Kafka、Redis、Network 等资源类型
- **容量预测**: 基于线性回归的容量预测，预测资源耗尽时间
- **阈值检查**: 可配置的资源使用阈值，超过阈值自动告警
- **Prometheus 集成**: 暴露资源使用指标到 Prometheus

### 数据生命周期管理 (Lifecycle Management)

- **自动归档**: 按配置的保留策略自动归档过期消息
- **批量处理**: 支持批量归档，避免对数据库造成过大压力
- **一致性保证**: 确保消息只存在于热存储或冷存储之一
- **重试机制**: 归档失败自动重试

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CapacityMonitor                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │   MySQL      │  │   Kafka      │  │   Network    │     │
│  │  Collector   │  │  Collector   │  │  Collector   │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                 │                 │              │
│         └─────────────────┴─────────────────┘              │
│                           │                                │
│                    ┌──────▼──────┐                         │
│                    │   History   │                         │
│                    │    Store    │                         │
│                    └─────────────┘                         │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                  LifecycleManager                           │
│  ┌──────────────┐         ┌──────────────┐                 │
│  │ Hot Storage  │────────►│ Cold Storage │                 │
│  │   (MySQL)    │ Archive │  (Archive)   │                 │
│  └──────────────┘         └──────────────┘                 │
│         │                                                   │
│         │ Retention Policies                                │
│         │ - Message Type                                    │
│         │ - Hot TTL                                         │
│         │ - Archive After                                   │
└─────────────────────────────────────────────────────────────┘
```

## Usage

### 容量监控

```go
import "github.com/pingxin403/cuckoo/libs/capacity"

// 创建历史存储
history := capacity.NewInMemoryHistoryStore(1000)

// 创建容量监控器
thresholds := capacity.ThresholdConfig{
    DefaultPercent: 80.0,
    Overrides: map[capacity.ResourceType]float64{
        capacity.ResourceMySQL: 85.0,
        capacity.ResourceKafka: 90.0,
    },
}

monitor := capacity.NewDefaultCapacityMonitor(thresholds, history)

// 注册采集器
metrics := capacity.NewCollectorMetrics(prometheus.DefaultRegisterer)
monitor.RegisterCollector(capacity.ResourceMySQL, 
    capacity.NewMySQLCollector(db, metrics))

// 采集资源使用情况
usages, err := monitor.CollectUsage(ctx, "region-a")
if err != nil {
    log.Fatal(err)
}

// 检查阈值
exceeded := monitor.CheckThresholds(ctx, usages)
for _, usage := range exceeded {
    log.Printf("Resource %s exceeded threshold: %.2f%%", 
        usage.ResourceName, usage.UsagePercent)
}

// 容量预测
forecast, err := monitor.Forecast(ctx, capacity.ResourceMySQL, "im_chat")
if err != nil {
    log.Fatal(err)
}
log.Printf("Days until full: %d", forecast.DaysUntilFull)
```

### 数据生命周期管理

```go
import "github.com/pingxin403/cuckoo/libs/capacity"

// 定义保留策略
policies := []capacity.RetentionPolicy{
    {
        MessageType:  "text",
        HotTTL:       30 * 24 * time.Hour,  // 30 天
        ArchiveAfter: 30 * 24 * time.Hour,
    },
    {
        MessageType:  "media",
        HotTTL:       7 * 24 * time.Hour,   // 7 天
        ArchiveAfter: 7 * 24 * time.Hour,
    },
}

// 创建生命周期管理器
manager := capacity.NewLifecycleManager("region-a", policies, hotDB, coldDB)

// 归档过期消息
result, err := manager.ArchiveExpiredMessages(ctx, 1000)
if err != nil {
    log.Fatal(err)
}

log.Printf("Archived: %d, Failed: %d, Duration: %v", 
    result.ArchivedCount, result.FailedCount, result.Duration)
```

## Configuration

### 阈值配置

```yaml
capacity:
  thresholds:
    default_percent: 80.0
    overrides:
      mysql: 85.0
      kafka: 90.0
      network: 75.0
```

### 保留策略配置

```yaml
lifecycle:
  policies:
    - message_type: "text"
      hot_ttl: "720h"      # 30 days
      archive_after: "720h"
    - message_type: "media"
      hot_ttl: "168h"      # 7 days
      archive_after: "168h"
    - message_type: "system"
      hot_ttl: "2160h"     # 90 days
      archive_after: "2160h"
```

## Metrics

### 容量监控指标

- `capacity_resource_usage_bytes{resource_type, region_id, resource_name}`: 资源使用字节数
- `capacity_resource_usage_percent{resource_type, region_id, resource_name}`: 资源使用百分比
- `capacity_collection_success_total{resource_type, region_id}`: 采集成功次数
- `capacity_collection_errors_total{resource_type, region_id}`: 采集失败次数
- `capacity_collection_duration_seconds{resource_type, region_id}`: 采集耗时

### 生命周期管理指标

- `lifecycle_archived_messages_total{region_id}`: 归档消息总数
- `lifecycle_archive_errors_total{region_id}`: 归档失败总数
- `lifecycle_archive_duration_seconds{region_id}`: 归档耗时

## Testing

```bash
# 运行单元测试
go test -v ./...

# 运行属性测试
go test -v -run TestCapacityForecastMonotonicity

# 运行集成测试
go test -v -tags=integration ./...
```

## Requirements

- Go 1.23+
- MySQL 5.7+ (用于数据归档)
- Prometheus (用于指标收集)

## Related

- [Multi-Region Active-Active Architecture](../../.kiro/specs/multi-region-active-active/)
- [Requirements 7.1: 容量监控](../../.kiro/specs/multi-region-active-active/requirements.md#71-容量监控)
- [Requirements 7.2: 数据生命周期管理](../../.kiro/specs/multi-region-active-active/requirements.md#72-数据生命周期管理)
