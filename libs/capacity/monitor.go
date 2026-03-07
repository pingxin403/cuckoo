package capacity

import (
	"context"
	"fmt"
	"math"
	"time"
)

// CapacityMonitor 容量监控器接口
type CapacityMonitor interface {
	// CollectUsage 采集指定地域的资源使用情况
	CollectUsage(ctx context.Context, regionID string) ([]ResourceUsage, error)

	// Forecast 基于历史数据预测资源容量
	Forecast(ctx context.Context, resourceType ResourceType, name string) (*CapacityForecast, error)

	// CheckThresholds 检查资源使用是否超过阈值，返回超过阈值的资源列表
	CheckThresholds(ctx context.Context, usages []ResourceUsage) []ResourceUsage
}

// DefaultCapacityMonitor 默认容量监控器实现
type DefaultCapacityMonitor struct {
	thresholds ThresholdConfig
	collectors map[ResourceType]ResourceCollector
	history    HistoryStore
}

// ResourceCollector 资源采集器接口
type ResourceCollector interface {
	Collect(ctx context.Context, regionID string) ([]ResourceUsage, error)
}

// HistoryStore 历史数据存储接口
type HistoryStore interface {
	Store(ctx context.Context, usage ResourceUsage) error
	Query(ctx context.Context, resourceType ResourceType, name string, since time.Time) ([]ResourceUsage, error)
}

// NewDefaultCapacityMonitor 创建默认容量监控器
func NewDefaultCapacityMonitor(thresholds ThresholdConfig, history HistoryStore) *DefaultCapacityMonitor {
	return &DefaultCapacityMonitor{
		thresholds: thresholds,
		collectors: make(map[ResourceType]ResourceCollector),
		history:    history,
	}
}

// RegisterCollector 注册资源采集器
func (m *DefaultCapacityMonitor) RegisterCollector(resourceType ResourceType, collector ResourceCollector) {
	m.collectors[resourceType] = collector
}

// CollectUsage 采集指定地域的资源使用情况
func (m *DefaultCapacityMonitor) CollectUsage(ctx context.Context, regionID string) ([]ResourceUsage, error) {
	var allUsages []ResourceUsage

	for resourceType, collector := range m.collectors {
		usages, err := collector.Collect(ctx, regionID)
		if err != nil {
			return nil, fmt.Errorf("failed to collect %s usage: %w", resourceType, err)
		}

		// 存储到历史记录
		for _, usage := range usages {
			if err := m.history.Store(ctx, usage); err != nil {
				// 记录错误但继续处理
				fmt.Printf("failed to store usage history: %v\n", err)
			}
		}

		allUsages = append(allUsages, usages...)
	}

	return allUsages, nil
}

// Forecast 基于历史数据预测资源容量
func (m *DefaultCapacityMonitor) Forecast(ctx context.Context, resourceType ResourceType, name string) (*CapacityForecast, error) {
	// 查询最近 30 天的历史数据
	since := time.Now().Add(-30 * 24 * time.Hour)
	history, err := m.history.Query(ctx, resourceType, name, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}

	if len(history) < 7 {
		return nil, fmt.Errorf("insufficient data: need at least 7 days of history, got %d samples", len(history))
	}

	// 使用线性回归计算增长率
	growthRate := calculateGrowthRate(history)

	// 获取最新使用量
	latest := history[len(history)-1]
	currentUsage := latest.UsagePercent

	// 计算达到 100% 的天数
	daysUntilFull := 0
	if growthRate > 0 {
		remainingPercent := 100.0 - currentUsage
		daysUntilFull = int(math.Ceil(remainingPercent / (growthRate * 100.0 / float64(latest.TotalBytes))))
	}

	return &CapacityForecast{
		ResourceType:     resourceType,
		ResourceName:     name,
		CurrentUsage:     currentUsage,
		GrowthRatePerDay: growthRate,
		DaysUntilFull:    daysUntilFull,
		ForecastDate:     time.Now(),
	}, nil
}

// CheckThresholds 检查资源使用是否超过阈值
func (m *DefaultCapacityMonitor) CheckThresholds(ctx context.Context, usages []ResourceUsage) []ResourceUsage {
	var exceeded []ResourceUsage

	for _, usage := range usages {
		threshold := m.thresholds.DefaultPercent
		if override, ok := m.thresholds.Overrides[usage.ResourceType]; ok {
			threshold = override
		}

		if usage.UsagePercent >= threshold {
			exceeded = append(exceeded, usage)
		}
	}

	return exceeded
}

// calculateGrowthRate 使用线性回归计算增长率（字节/天）
func calculateGrowthRate(history []ResourceUsage) float64 {
	if len(history) < 2 {
		return 0
	}

	// 简单线性回归: y = mx + b
	// 其中 x 是天数，y 是使用字节数
	n := float64(len(history))
	var sumX, sumY, sumXY, sumX2 float64

	baseTime := history[0].Timestamp
	for i, usage := range history {
		x := float64(i)
		y := float64(usage.UsedBytes)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// 计算斜率 m = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	// 将斜率转换为每天的增长率
	// 假设历史数据是均匀分布的
	if len(history) > 1 {
		totalDays := history[len(history)-1].Timestamp.Sub(baseTime).Hours() / 24
		if totalDays > 0 {
			slope = slope * (float64(len(history)-1) / totalDays)
		}
	}

	return slope
}
