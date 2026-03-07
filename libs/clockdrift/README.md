# Clock Drift Detection and Calibration

时钟偏移检测与校准库，用于多地域 active-active 架构中的 HLC 时钟同步。

## 功能特性

- **定期 NTP 检测**: 定期查询 NTP 服务器检测本地时钟偏移
- **自动校准**: 当偏移超过阈值时自动校准 HLC
- **历史记录**: 使用环形缓冲区记录偏移历史
- **Prometheus 指标**: 暴露偏移量、校准次数等指标
- **并发安全**: 所有操作都是线程安全的

## 核心组件

### RingBuffer

环形缓冲区，用于存储时钟偏移采样历史。

```go
// 创建容量为 100 的环形缓冲区
rb := clockdrift.NewRingBuffer(100)

// 添加采样
sample := clockdrift.ClockSample{
    Timestamp: time.Now(),
    Offset:    50 * time.Millisecond,
    Source:    "pool.ntp.org",
}
rb.Push(sample)

// 获取所有采样（按时间顺序）
samples := rb.GetAll()

// 获取最近 1 小时的采样
recentSamples := rb.GetSince(time.Now().Add(-time.Hour))
```

### DriftDetector

时钟偏移检测器，定期检测偏移并触发校准。

```go
// 配置
cfg := clockdrift.DefaultConfig()
cfg.NTPServer = "pool.ntp.org"
cfg.CheckInterval = 30 * time.Second
cfg.Threshold = 500 * time.Millisecond
cfg.MaxOffset = 10 * time.Second

// 创建 HLC 实例
hlc := hlc.NewHLC("region-a", "node-1")

// 创建检测器，传入 HLC 校准函数
detector := clockdrift.NewDriftDetector(cfg, func(offset time.Duration) error {
    return hlc.AdjustForDrift(offset)
})

// 启动定期检测
ctx := context.Background()
go detector.Start(ctx)

// 查询偏移历史
history := detector.GetHistory(24 * time.Hour)
for _, sample := range history {
    fmt.Printf("Time: %v, Offset: %v\n", sample.Timestamp, sample.Offset)
}

// 获取最近一次偏移
offset, checkTime := detector.GetLastOffset()
fmt.Printf("Last offset: %v at %v\n", offset, checkTime)
```

## HLC 校准

HLC 支持基于检测到的时钟偏移进行校准：

```go
hlc := hlc.NewHLC("region-a", "node-1")

// 当检测到正偏移（本地时钟快）时，增加逻辑计数器
offset := 100 * time.Millisecond
err := hlc.AdjustForDrift(offset)

// 校准后生成的 ID 仍然保持单调性
id := hlc.GenerateID()
```

### 校准策略

- **正偏移（本地时钟快）**: 增加逻辑计数器步长，补偿物理时钟偏差
- **负偏移（本地时钟慢）**: 不做调整，HLC 算法会在下次 GenerateID 时自然使用更高的墙上时钟
- **零偏移**: 不做任何调整

## Prometheus 指标

DriftDetector 暴露以下指标：

- `clock_drift_current_offset_ms`: 当前时钟偏移（毫秒）
- `clock_drift_calibration_total`: HLC 校准总次数
- `clock_drift_threshold_breach_total`: 偏移超过阈值的总次数
- `clock_drift_check_latency_ms`: NTP 检查延迟（毫秒）

```go
// 获取 Prometheus registry
registry := detector.GetRegistry()

// 注册到全局 registry 或使用自定义 HTTP handler
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

## 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `NTPServer` | `pool.ntp.org` | NTP 服务器地址 |
| `CheckInterval` | `30s` | 检测间隔 |
| `Threshold` | `500ms` | 触发校准的阈值 |
| `MaxOffset` | `10s` | 最大可接受偏移（超过则仅告警） |
| `HistoryDuration` | `24h` | 历史记录保留时长 |
| `HistoryCapacity` | `2880` | 环形缓冲区容量 |

## 使用示例

### 集成到 IM Service

```go
package main

import (
    "context"
    "log"
    
    "github.com/yourusername/im-platform/libs/clockdrift"
    "github.com/yourusername/im-platform/libs/hlc"
)

func main() {
    // 创建 HLC
    hlc := hlc.NewHLC("region-a", "node-1")
    
    // 创建时钟偏移检测器
    cfg := clockdrift.DefaultConfig()
    detector := clockdrift.NewDriftDetector(cfg, func(offset time.Duration) error {
        log.Printf("Calibrating HLC with offset: %v", offset)
        return hlc.AdjustForDrift(offset)
    })
    
    // 启动检测
    ctx := context.Background()
    go func() {
        if err := detector.Start(ctx); err != nil {
            log.Printf("Drift detector stopped: %v", err)
        }
    }()
    
    // 使用 HLC 生成 ID
    id := hlc.GenerateID()
    log.Printf("Generated ID: %v", id)
    
    // 查询偏移历史
    history := detector.GetHistory(1 * time.Hour)
    log.Printf("Recent drift history: %d samples", len(history))
}
```

## 测试

运行单元测试：

```bash
cd libs/clockdrift
go test -v
```

运行特定测试：

```bash
go test -v -run TestRingBuffer
go test -v -run TestDriftDetector
```

## 需求满足

该实现满足以下需求：

- ✅ 8.2.1: 定期检测本地时钟与 NTP 服务器的偏移量（默认 30 秒）
- ✅ 8.2.2: 偏移超过阈值时触发告警（默认 500ms）
- ✅ 8.2.3: 记录时钟偏移历史数据（支持查询最近 24 小时）
- ✅ 8.2.4: 偏移超过阈值时 HLC 增加逻辑计数器步长补偿
- ✅ 8.2.5: 通过 Prometheus 指标暴露偏移量和校准次数

## 架构决策

### 为什么使用环形缓冲区？

- 固定内存占用，不会无限增长
- O(1) 写入性能
- 自动淘汰旧数据

### 为什么正偏移增加逻辑计数器？

- 保持 HLC 单调性：即使物理时钟被校准回退，逻辑计数器确保生成的 ID 仍然递增
- 避免 ID 冲突：不同节点在相同物理时间生成的 ID 通过逻辑计数器区分

### 为什么负偏移不调整？

- HLC 算法天然处理：下次 GenerateID 会使用更高的墙上时钟
- 避免复杂性：不需要额外的补偿逻辑

## 相关文档

- [HLC 实现文档](../hlc/README.md)
- [多地域架构设计](../../.kiro/specs/multi-region-active-active/design.md)
- [需求文档](../../.kiro/specs/multi-region-active-active/requirements.md)
