package loadtest

import (
	"time"
)

// LoadTestConfig 压力测试配置
type LoadTestConfig struct {
	// 总连接数
	TotalConnections int `yaml:"total_connections"`
	// Region A 连接占比 (0-100)
	RegionAPercent int `yaml:"region_a_percent"`
	// 消息发送速率 (messages/second)
	MessageRate int `yaml:"message_rate"`
	// 测试持续时间
	Duration time.Duration `yaml:"duration"`
	// 预热时间 (逐步增加连接)
	RampUpTime time.Duration `yaml:"ramp_up_time"`
	// 跨地域消息比例 (0.0-1.0)
	CrossRegionRatio float64 `yaml:"cross_region_ratio"`

	// WebSocket 端点
	RegionAEndpoint string `yaml:"region_a_endpoint"`
	RegionBEndpoint string `yaml:"region_b_endpoint"`

	// 认证令牌 (测试用)
	AuthToken string `yaml:"auth_token"`
}

// LoadTestResult 压力测试结果
type LoadTestResult struct {
	// 总消息数
	TotalMessages int64 `json:"total_messages"`
	// 成功率
	SuccessRate float64 `json:"success_rate"`
	// 延迟统计
	LatencyP50 time.Duration `json:"latency_p50"`
	LatencyP95 time.Duration `json:"latency_p95"`
	LatencyP99 time.Duration `json:"latency_p99"`
	// 跨地域延迟
	CrossRegionP99 time.Duration `json:"cross_region_p99"`
	// 故障转移影响 (可选)
	FailoverImpact *FailoverImpact `json:"failover_impact,omitempty"`
	// 测试持续时间
	Duration time.Duration `json:"duration"`
	// 吞吐量 (messages/second)
	Throughput float64 `json:"throughput"`
}

// FailoverImpact 故障转移影响统计
type FailoverImpact struct {
	// 故障转移开始时间
	StartTime time.Time `json:"start_time"`
	// 故障转移完成时间
	EndTime time.Time `json:"end_time"`
	// 故障转移期间消息数
	MessagesDuringFailover int64 `json:"messages_during_failover"`
	// 故障转移期间失败消息数
	FailedMessages int64 `json:"failed_messages"`
	// 故障转移前吞吐量
	ThroughputBefore float64 `json:"throughput_before"`
	// 故障转移期间吞吐量
	ThroughputDuring float64 `json:"throughput_during"`
	// 故障转移后吞吐量
	ThroughputAfter float64 `json:"throughput_after"`
	// 延迟影响
	LatencyIncrease time.Duration `json:"latency_increase"`
}

// MessageStats 消息统计
type MessageStats struct {
	Sent      int64
	Received  int64
	Failed    int64
	Latencies []time.Duration
}

// ConnectionStats 连接统计
type ConnectionStats struct {
	Active     int
	Failed     int
	Reconnects int
	RegionA    int
	RegionB    int
}
