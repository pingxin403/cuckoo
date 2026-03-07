package loadtest

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestBasicLoadTest 基础压力测试示例
// 验证需求 9.1.1: 模拟至少 10 万并发 WebSocket 连接
// 验证需求 9.1.2: 测量跨地域消息吞吐量并输出 P50/P95/P99 延迟
func TestBasicLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := &LoadTestConfig{
		TotalConnections: 1000, // 降低用于测试，生产环境使用 100000
		RegionAPercent:   50,
		MessageRate:      100,
		Duration:         30 * time.Second,
		RampUpTime:       5 * time.Second,
		CrossRegionRatio: 0.3,
		RegionAEndpoint:  "ws://localhost:8080/ws",
		RegionBEndpoint:  "ws://localhost:8081/ws",
		AuthToken:        "test-token",
	}

	runner := NewLoadTestRunner(config)
	defer runner.Cleanup()

	ctx := context.Background()
	result, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// 打印结果
	printLoadTestResult(t, result)

	// 验证结果
	if result.SuccessRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%%", result.SuccessRate)
	}

	if result.LatencyP99 > 500*time.Millisecond {
		t.Errorf("P99 latency too high: %v", result.LatencyP99)
	}
}

// TestFailoverLoadTest 故障转移压力测试示例
// 验证需求 9.1.3: 测量故障转移对吞吐量和延迟的影响
func TestFailoverLoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failover test in short mode")
	}

	config := &FailoverTestConfig{
		LoadTestConfig: LoadTestConfig{
			TotalConnections: 1000,
			RegionAPercent:   50,
			MessageRate:      100,
			Duration:         60 * time.Second,
			RampUpTime:       5 * time.Second,
			CrossRegionRatio: 0.3,
			RegionAEndpoint:  "ws://localhost:8080/ws",
			RegionBEndpoint:  "ws://localhost:8081/ws",
			AuthToken:        "test-token",
		},
		FailoverTriggerDelay: 20 * time.Second,
		FailoverRecoveryTime: 5 * time.Second,
		FailedRegion:         "region-a",
	}

	runner := NewFailoverTestRunner(config)
	defer runner.Cleanup()

	ctx := context.Background()
	result, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Failover test failed: %v", err)
	}

	// 打印结果
	printLoadTestResult(t, result)

	// 验证故障转移影响
	if result.FailoverImpact != nil {
		printFailoverImpact(t, result.FailoverImpact)

		// 验证 RTO < 30秒
		failoverDuration := result.FailoverImpact.EndTime.Sub(result.FailoverImpact.StartTime)
		if failoverDuration > 30*time.Second {
			t.Errorf("Failover RTO too high: %v (expected < 30s)", failoverDuration)
		}

		// 验证吞吐量影响
		throughputDrop := (result.FailoverImpact.ThroughputBefore - result.FailoverImpact.ThroughputDuring) /
			result.FailoverImpact.ThroughputBefore * 100
		t.Logf("Throughput drop during failover: %.2f%%", throughputDrop)
	}

	// 验证无消息丢失
	if err := runner.VerifyNoMessageLoss(); err != nil {
		t.Errorf("Message loss verification failed: %v", err)
	}
}

// TestLongRunningStabilityTest 长时间稳定性测试示例
// 验证需求 9.1.4: 支持持续运行至少 24 小时的稳定性测试
func TestLongRunningStabilityTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// 注意: 实际 24 小时测试应在生产环境运行
	// 这里使用较短时间用于演示
	config := &LoadTestConfig{
		TotalConnections: 10000,
		RegionAPercent:   50,
		MessageRate:      1000,
		Duration:         5 * time.Minute, // 实际应为 24 * time.Hour
		RampUpTime:       30 * time.Second,
		CrossRegionRatio: 0.3,
		RegionAEndpoint:  "ws://localhost:8080/ws",
		RegionBEndpoint:  "ws://localhost:8081/ws",
		AuthToken:        "test-token",
	}

	runner := NewLoadTestRunner(config)
	defer runner.Cleanup()

	ctx := context.Background()
	result, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Stability test failed: %v", err)
	}

	printLoadTestResult(t, result)

	// 验证稳定性指标
	if result.SuccessRate < 99.0 {
		t.Errorf("Success rate too low for stability test: %.2f%%", result.SuccessRate)
	}
}

// printLoadTestResult 打印测试结果
func printLoadTestResult(t *testing.T, result *LoadTestResult) {
	t.Logf("=== Load Test Results ===")
	t.Logf("Duration: %v", result.Duration)
	t.Logf("Total Messages: %d", result.TotalMessages)
	t.Logf("Success Rate: %.2f%%", result.SuccessRate)
	t.Logf("Throughput: %.2f msg/s", result.Throughput)
	t.Logf("Latency P50: %v", result.LatencyP50)
	t.Logf("Latency P95: %v", result.LatencyP95)
	t.Logf("Latency P99: %v", result.LatencyP99)
	t.Logf("Cross-Region P99: %v", result.CrossRegionP99)
}

// printFailoverImpact 打印故障转移影响
func printFailoverImpact(t *testing.T, impact *FailoverImpact) {
	t.Logf("=== Failover Impact ===")
	t.Logf("Start Time: %v", impact.StartTime)
	t.Logf("End Time: %v", impact.EndTime)
	t.Logf("Duration: %v", impact.EndTime.Sub(impact.StartTime))
	t.Logf("Messages During Failover: %d", impact.MessagesDuringFailover)
	t.Logf("Failed Messages: %d", impact.FailedMessages)
	t.Logf("Throughput Before: %.2f msg/s", impact.ThroughputBefore)
	t.Logf("Throughput During: %.2f msg/s", impact.ThroughputDuring)
	t.Logf("Throughput After: %.2f msg/s", impact.ThroughputAfter)
	t.Logf("Latency Increase: %v", impact.LatencyIncrease)

	if impact.ThroughputBefore > 0 {
		drop := (impact.ThroughputBefore - impact.ThroughputDuring) / impact.ThroughputBefore * 100
		t.Logf("Throughput Drop: %.2f%%", drop)
	}
}

// BenchmarkMessageSending 消息发送性能基准测试
func BenchmarkMessageSending(b *testing.B) {
	config := &LoadTestConfig{
		TotalConnections: 100,
		RegionAPercent:   50,
		MessageRate:      1000,
		Duration:         10 * time.Second,
		RampUpTime:       1 * time.Second,
		CrossRegionRatio: 0.3,
		RegionAEndpoint:  "ws://localhost:8080/ws",
		RegionBEndpoint:  "ws://localhost:8081/ws",
		AuthToken:        "test-token",
	}

	runner := NewLoadTestRunner(config)
	defer runner.Cleanup()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.Run(ctx)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// ExampleLoadTestRunner 使用示例
func ExampleLoadTestRunner() {
	config := &LoadTestConfig{
		TotalConnections: 1000,
		RegionAPercent:   50,
		MessageRate:      100,
		Duration:         30 * time.Second,
		RampUpTime:       5 * time.Second,
		CrossRegionRatio: 0.3,
		RegionAEndpoint:  "ws://localhost:8080/ws",
		RegionBEndpoint:  "ws://localhost:8081/ws",
		AuthToken:        "test-token",
	}

	runner := NewLoadTestRunner(config)
	defer runner.Cleanup()

	ctx := context.Background()
	result, err := runner.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Total Messages: %d\n", result.TotalMessages)
	fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
	fmt.Printf("Throughput: %.2f msg/s\n", result.Throughput)
	fmt.Printf("P99 Latency: %v\n", result.LatencyP99)
}
