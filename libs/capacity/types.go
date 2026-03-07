package capacity

import (
	"time"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceMySQL   ResourceType = "mysql"
	ResourceKafka   ResourceType = "kafka"
	ResourceRedis   ResourceType = "redis"
	ResourceNetwork ResourceType = "network"
	ResourceCPU     ResourceType = "cpu"
	ResourceMemory  ResourceType = "memory"
)

// ResourceUsage 资源使用快照
type ResourceUsage struct {
	RegionID     string            `json:"region_id"`
	ResourceType ResourceType      `json:"resource_type"`
	ResourceName string            `json:"resource_name"`
	UsedBytes    int64             `json:"used_bytes"`
	TotalBytes   int64             `json:"total_bytes"`
	UsagePercent float64           `json:"usage_percent"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata"`
}

// CapacityForecast 容量预测结果
type CapacityForecast struct {
	ResourceType     ResourceType `json:"resource_type"`
	ResourceName     string       `json:"resource_name"`
	CurrentUsage     float64      `json:"current_usage_percent"`
	GrowthRatePerDay float64      `json:"growth_rate_per_day_bytes"`
	DaysUntilFull    int          `json:"days_until_full"`
	ForecastDate     time.Time    `json:"forecast_date"`
}

// ThresholdConfig 阈值配置
type ThresholdConfig struct {
	DefaultPercent float64                  `yaml:"default_percent"` // 默认 80%
	Overrides      map[ResourceType]float64 `yaml:"overrides"`
}
