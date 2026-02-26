package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestRunner 压力测试运行器
type LoadTestRunner struct {
	config  *LoadTestConfig
	pool    *ConnectionPool
	limiter *RateLimiter

	// 统计
	totalSent     int64
	totalReceived int64
	totalFailed   int64

	// 故障转移跟踪
	failoverStart time.Time
	failoverEnd   time.Time
	failoverStats *FailoverImpact

	mu sync.Mutex
}

// NewLoadTestRunner 创建压力测试运行器
func NewLoadTestRunner(config *LoadTestConfig) *LoadTestRunner {
	return &LoadTestRunner{
		config:  config,
		pool:    NewConnectionPool(config),
		limiter: NewRateLimiter(config.MessageRate),
	}
}

// Run 运行压力测试
func (r *LoadTestRunner) Run(ctx context.Context) (*LoadTestResult, error) {
	// 1. 建立连接
	fmt.Printf("Connecting %d WebSocket connections...\n", r.config.TotalConnections)
	if err := r.pool.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	connStats := r.pool.GetConnectionStats()
	fmt.Printf("Connected: %d active, %d failed (Region A: %d, Region B: %d)\n",
		connStats.Active, connStats.Failed, connStats.RegionA, connStats.RegionB)

	// 2. 运行测试
	fmt.Printf("Running load test for %v...\n", r.config.Duration)
	testCtx, cancel := context.WithTimeout(ctx, r.config.Duration)
	defer cancel()

	startTime := time.Now()

	// 启动消息发送
	var wg sync.WaitGroup
	connections := r.pool.GetConnections()

	for _, conn := range connections {
		wg.Add(1)
		go func(c *WSConnection) {
			defer wg.Done()
			r.sendMessages(testCtx, c)
		}(conn)
	}

	// 等待测试完成
	wg.Wait()
	duration := time.Since(startTime)

	// 3. 收集统计
	result := r.collectResults(duration)

	// 4. 关闭连接
	fmt.Println("Closing connections...")
	if err := r.pool.CloseAll(); err != nil {
		fmt.Printf("Warning: error closing connections: %v\n", err)
	}

	return result, nil
}

// sendMessages 发送消息 (单个连接)
func (r *LoadTestRunner) sendMessages(ctx context.Context, conn *WSConnection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 速率控制
			if err := r.limiter.Wait(ctx); err != nil {
				return
			}

			// 生成测试消息
			msg := r.generateMessage(conn)

			// 发送消息
			_, err := conn.SendMessage(msg)
			if err != nil {
				atomic.AddInt64(&r.totalFailed, 1)
			} else {
				atomic.AddInt64(&r.totalSent, 1)
			}
		}
	}
}

// generateMessage 生成测试消息
func (r *LoadTestRunner) generateMessage(conn *WSConnection) []byte {
	// 简单的测试消息格式
	msg := map[string]interface{}{
		"type":      "chat",
		"sender_id": conn.ID,
		"content":   fmt.Sprintf("Load test message from %s at %d", conn.ID, time.Now().UnixNano()),
		"timestamp": time.Now().UnixMilli(),
	}

	// 随机决定是否为跨地域消息
	if rand.Float64() < r.config.CrossRegionRatio {
		// 跨地域消息: 发送到另一个地域的用户
		if conn.Region == "region-a" {
			msg["receiver_region"] = "region-b"
		} else {
			msg["receiver_region"] = "region-a"
		}
	}

	data, _ := json.Marshal(msg)
	return data
}

// collectResults 收集测试结果
func (r *LoadTestRunner) collectResults(duration time.Duration) *LoadTestResult {
	// 收集所有连接的统计
	connections := r.pool.GetConnections()
	allStats := make([]MessageStats, 0, len(connections))

	for _, conn := range connections {
		stats := conn.GetStats()
		allStats = append(allStats, stats)
	}

	// 聚合统计
	aggregated := AggregateMessageStats(allStats)

	// 计算延迟统计
	latencyStats := CalculateLatencyStats(aggregated.Latencies)

	// 计算跨地域延迟
	crossRegionLatencies := FilterCrossRegionLatencies(aggregated.Latencies, r.config.CrossRegionRatio)
	crossRegionStats := CalculateLatencyStats(crossRegionLatencies)

	// 计算成功率和吞吐量
	successRate := CalculateSuccessRate(aggregated.Sent, aggregated.Failed)
	throughput := CalculateThroughput(aggregated.Sent, duration)

	result := &LoadTestResult{
		TotalMessages:  aggregated.Sent,
		SuccessRate:    successRate,
		LatencyP50:     latencyStats.P50,
		LatencyP95:     latencyStats.P95,
		LatencyP99:     latencyStats.P99,
		CrossRegionP99: crossRegionStats.P99,
		Duration:       duration,
		Throughput:     throughput,
	}

	// 如果有故障转移统计，添加到结果
	if r.failoverStats != nil {
		result.FailoverImpact = r.failoverStats
	}

	return result
}

// TriggerFailover 触发故障转移 (用于故障转移测试)
func (r *LoadTestRunner) TriggerFailover() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failoverStart = time.Now()

	// 记录故障转移前的吞吐量
	if r.failoverStats == nil {
		r.failoverStats = &FailoverImpact{
			StartTime: r.failoverStart,
		}
	}
}

// CompleteFailover 完成故障转移
func (r *LoadTestRunner) CompleteFailover() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failoverStats != nil {
		r.failoverStats.EndTime = time.Now()
	}
}

// Cleanup 清理资源
func (r *LoadTestRunner) Cleanup() error {
	r.limiter.Stop()
	return r.pool.CloseAll()
}
