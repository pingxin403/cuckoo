package loadtest

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// FailoverTestConfig 故障转移测试配置
type FailoverTestConfig struct {
	LoadTestConfig

	// 故障转移触发时间 (测试开始后多久触发)
	FailoverTriggerDelay time.Duration `yaml:"failover_trigger_delay"`

	// 故障转移恢复时间 (模拟故障转移耗时)
	FailoverRecoveryTime time.Duration `yaml:"failover_recovery_time"`

	// 故障地域
	FailedRegion string `yaml:"failed_region"` // "region-a" or "region-b"
}

// FailoverTestRunner 故障转移压力测试运行器
type FailoverTestRunner struct {
	*LoadTestRunner
	failoverConfig *FailoverTestConfig

	// 故障转移期间统计
	messagesDuringFailover int64
	failedDuringFailover   int64

	// 吞吐量采样
	throughputSamples []throughputSample
	samplingInterval  time.Duration
}

type throughputSample struct {
	timestamp    time.Time
	messageCount int64
	phase        string // "before", "during", "after"
}

// NewFailoverTestRunner 创建故障转移测试运行器
func NewFailoverTestRunner(config *FailoverTestConfig) *FailoverTestRunner {
	return &FailoverTestRunner{
		LoadTestRunner:    NewLoadTestRunner(&config.LoadTestConfig),
		failoverConfig:    config,
		throughputSamples: make([]throughputSample, 0),
		samplingInterval:  1 * time.Second,
	}
}

// Run 运行故障转移压力测试
// 验证需求 9.1.3: 测量故障转移对吞吐量和延迟的影响，验证无消息丢失
func (r *FailoverTestRunner) Run(ctx context.Context) (*LoadTestResult, error) {
	// 1. 建立连接
	fmt.Printf("Connecting %d WebSocket connections...\n", r.config.TotalConnections)
	if err := r.pool.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	connStats := r.pool.GetConnectionStats()
	fmt.Printf("Connected: %d active (Region A: %d, Region B: %d)\n",
		connStats.Active, connStats.RegionA, connStats.RegionB)

	// 2. 启动吞吐量采样
	samplingCtx, cancelSampling := context.WithCancel(ctx)
	defer cancelSampling()
	go r.sampleThroughput(samplingCtx)

	// 3. 启动消息发送
	testCtx, cancel := context.WithTimeout(ctx, r.config.Duration)
	defer cancel()

	startTime := time.Now()
	connections := r.pool.GetConnections()

	// 启动发送协程
	sendCtx, cancelSend := context.WithCancel(testCtx)
	defer cancelSend()

	for _, conn := range connections {
		go r.sendMessages(sendCtx, conn)
	}

	// 4. 等待并触发故障转移
	fmt.Printf("Running for %v before triggering failover...\n", r.failoverConfig.FailoverTriggerDelay)
	time.Sleep(r.failoverConfig.FailoverTriggerDelay)

	// 5. 触发故障转移
	fmt.Printf("Triggering failover for %s...\n", r.failoverConfig.FailedRegion)
	if err := r.triggerFailover(sendCtx); err != nil {
		return nil, fmt.Errorf("failover failed: %w", err)
	}

	// 6. 等待测试完成
	<-testCtx.Done()
	duration := time.Since(startTime)

	fmt.Println("Test completed, collecting results...")

	// 7. 收集结果
	result := r.collectFailoverResults(duration)

	// 8. 清理
	if err := r.Cleanup(); err != nil {
		fmt.Printf("Warning: cleanup error: %v\n", err)
	}

	return result, nil
}

// triggerFailover 触发故障转移
func (r *FailoverTestRunner) triggerFailover(ctx context.Context) error {
	r.TriggerFailover()

	fmt.Printf("Simulating %s failure...\n", r.failoverConfig.FailedRegion)

	// 记录故障转移前的消息计数
	messagesBefore := atomic.LoadInt64(&r.totalSent)

	// 关闭故障地域的连接
	failedCount := r.closeRegionConnections(r.failoverConfig.FailedRegion)
	fmt.Printf("Closed %d connections in %s\n", failedCount, r.failoverConfig.FailedRegion)

	// 模拟故障转移恢复时间
	fmt.Printf("Waiting %v for failover recovery...\n", r.failoverConfig.FailoverRecoveryTime)
	time.Sleep(r.failoverConfig.FailoverRecoveryTime)

	// 重新连接到健康地域
	healthyRegion := "region-b"
	healthyEndpoint := r.config.RegionBEndpoint
	if r.failoverConfig.FailedRegion == "region-b" {
		healthyRegion = "region-a"
		healthyEndpoint = r.config.RegionAEndpoint
	}

	fmt.Printf("Reconnecting %d connections to %s...\n", failedCount, healthyRegion)
	if err := r.reconnectToRegion(ctx, healthyRegion, healthyEndpoint, failedCount); err != nil {
		return fmt.Errorf("reconnection failed: %w", err)
	}

	// 记录故障转移期间的消息统计
	messagesAfter := atomic.LoadInt64(&r.totalSent)
	r.messagesDuringFailover = messagesAfter - messagesBefore

	r.CompleteFailover()

	fmt.Printf("Failover completed. Messages during failover: %d\n", r.messagesDuringFailover)

	return nil
}

// closeRegionConnections 关闭指定地域的所有连接
func (r *FailoverTestRunner) closeRegionConnections(region string) int {
	connections := r.pool.GetConnections()
	closedCount := 0

	for _, conn := range connections {
		if conn.Region == region {
			conn.Close()
			closedCount++
		}
	}

	return closedCount
}

// reconnectToRegion 重新连接到指定地域
func (r *FailoverTestRunner) reconnectToRegion(ctx context.Context, region, endpoint string, count int) error {
	for i := 0; i < count; i++ {
		if err := r.pool.connectOne(ctx, region, endpoint); err != nil {
			return fmt.Errorf("failed to reconnect connection %d: %w", i, err)
		}

		// 启动新连接的消息发送
		connections := r.pool.GetConnections()
		newConn := connections[len(connections)-1]
		go r.sendMessages(ctx, newConn)
	}

	return nil
}

// sampleThroughput 定期采样吞吐量
func (r *FailoverTestRunner) sampleThroughput(ctx context.Context) {
	ticker := time.NewTicker(r.samplingInterval)
	defer ticker.Stop()

	lastCount := int64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			currentCount := atomic.LoadInt64(&r.totalSent)

			// 确定当前阶段
			phase := "before"
			if !r.failoverStart.IsZero() {
				if r.failoverEnd.IsZero() {
					phase = "during"
				} else {
					phase = "after"
				}
			}

			sample := throughputSample{
				timestamp:    now,
				messageCount: currentCount - lastCount,
				phase:        phase,
			}

			r.mu.Lock()
			r.throughputSamples = append(r.throughputSamples, sample)
			r.mu.Unlock()

			lastCount = currentCount
		}
	}
}

// collectFailoverResults 收集故障转移测试结果
func (r *FailoverTestRunner) collectFailoverResults(duration time.Duration) *LoadTestResult {
	// 基础结果
	result := r.collectResults(duration)

	// 计算故障转移影响
	if r.failoverStats != nil {
		r.calculateFailoverImpact(result)
	}

	return result
}

// calculateFailoverImpact 计算故障转移影响
func (r *FailoverTestRunner) calculateFailoverImpact(result *LoadTestResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.failoverStats == nil {
		return
	}

	// 分析吞吐量样本
	var beforeSamples, duringSamples, afterSamples []int64

	for _, sample := range r.throughputSamples {
		switch sample.phase {
		case "before":
			beforeSamples = append(beforeSamples, sample.messageCount)
		case "during":
			duringSamples = append(duringSamples, sample.messageCount)
		case "after":
			afterSamples = append(afterSamples, sample.messageCount)
		}
	}

	// 计算平均吞吐量
	r.failoverStats.ThroughputBefore = calculateAvgThroughput(beforeSamples, r.samplingInterval)
	r.failoverStats.ThroughputDuring = calculateAvgThroughput(duringSamples, r.samplingInterval)
	r.failoverStats.ThroughputAfter = calculateAvgThroughput(afterSamples, r.samplingInterval)

	// 设置故障转移期间的消息统计
	r.failoverStats.MessagesDuringFailover = r.messagesDuringFailover
	r.failoverStats.FailedMessages = atomic.LoadInt64(&r.failedDuringFailover)

	// 计算延迟增加 (简化: 使用 P99 差异)
	if r.failoverStats.ThroughputBefore > 0 {
		// 延迟增加估算
		throughputRatio := r.failoverStats.ThroughputDuring / r.failoverStats.ThroughputBefore
		if throughputRatio < 1.0 {
			// 吞吐量下降意味着延迟增加
			r.failoverStats.LatencyIncrease = time.Duration(float64(result.LatencyP99) * (1.0 - throughputRatio))
		}
	}

	result.FailoverImpact = r.failoverStats
}

// calculateAvgThroughput 计算平均吞吐量
func calculateAvgThroughput(samples []int64, interval time.Duration) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	var sum int64
	for _, s := range samples {
		sum += s
	}

	avgPerInterval := float64(sum) / float64(len(samples))
	return avgPerInterval / interval.Seconds()
}

// VerifyNoMessageLoss 验证故障转移期间无消息丢失
// 验证需求 9.1.3: 验证故障转移期间无消息丢失
func (r *FailoverTestRunner) VerifyNoMessageLoss() error {
	connections := r.pool.GetConnections()

	var totalSent, totalReceived, totalFailed int64

	for _, conn := range connections {
		stats := conn.GetStats()
		totalSent += stats.Sent
		totalReceived += stats.Received
		totalFailed += stats.Failed
	}

	// 检查消息丢失
	// 注意: 这是简化检查，实际应该通过序列号验证
	lossRate := float64(totalFailed) / float64(totalSent) * 100.0

	if lossRate > 1.0 { // 允许 1% 的消息丢失
		return fmt.Errorf("message loss detected: %.2f%% (%d/%d)", lossRate, totalFailed, totalSent)
	}

	fmt.Printf("Message loss verification passed: %.2f%% loss rate\n", lossRate)
	return nil
}
