package loadtest

import (
	"math"
	"sort"
	"time"
)

// LatencyStats 延迟统计结果
type LatencyStats struct {
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
	Min time.Duration
	Max time.Duration
	Avg time.Duration
}

// CalculateLatencyStats 计算延迟统计 (P50/P95/P99)
func CalculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// 复制并排序
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	stats := LatencyStats{
		Min: sorted[0],
		Max: sorted[len(sorted)-1],
	}

	// 计算百分位数
	stats.P50 = percentile(sorted, 0.50)
	stats.P95 = percentile(sorted, 0.95)
	stats.P99 = percentile(sorted, 0.99)

	// 计算平均值
	var sum time.Duration
	for _, lat := range sorted {
		sum += lat
	}
	stats.Avg = sum / time.Duration(len(sorted))

	return stats
}

// percentile 计算百分位数
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	index := int(math.Ceil(float64(len(sorted)) * p))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// AggregateMessageStats 聚合多个连接的消息统计
func AggregateMessageStats(stats []MessageStats) MessageStats {
	aggregated := MessageStats{
		Latencies: make([]time.Duration, 0),
	}

	for _, s := range stats {
		aggregated.Sent += s.Sent
		aggregated.Received += s.Received
		aggregated.Failed += s.Failed
		aggregated.Latencies = append(aggregated.Latencies, s.Latencies...)
	}

	return aggregated
}

// CalculateSuccessRate 计算成功率
func CalculateSuccessRate(sent, failed int64) float64 {
	if sent == 0 {
		return 0.0
	}

	success := sent - failed
	return float64(success) / float64(sent) * 100.0
}

// CalculateThroughput 计算吞吐量 (messages/second)
func CalculateThroughput(messageCount int64, duration time.Duration) float64 {
	if duration == 0 {
		return 0.0
	}

	seconds := duration.Seconds()
	return float64(messageCount) / seconds
}

// FilterCrossRegionLatencies 过滤跨地域消息延迟
// 注意: 这是一个简化实现，实际需要根据消息元数据判断
func FilterCrossRegionLatencies(latencies []time.Duration, ratio float64) []time.Duration {
	if ratio <= 0 || ratio >= 1 {
		return latencies
	}

	// 简化: 假设延迟较高的消息是跨地域消息
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// 取高延迟部分
	startIndex := int(float64(len(sorted)) * (1 - ratio))
	if startIndex < 0 {
		startIndex = 0
	}

	return sorted[startIndex:]
}
