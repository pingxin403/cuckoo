package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cuckoo-org/cuckoo/tests/loadtest"
)

var (
	// 基础配置
	connections      = flag.Int("connections", 1000, "Total number of WebSocket connections")
	regionAPercent   = flag.Int("region-a-percent", 50, "Percentage of connections to Region A (0-100)")
	messageRate      = flag.Int("rate", 100, "Message sending rate (messages/second)")
	duration         = flag.Duration("duration", 1*time.Minute, "Test duration")
	rampUpTime       = flag.Duration("rampup", 10*time.Second, "Ramp-up time for connections")
	crossRegionRatio = flag.Float64("cross-region", 0.3, "Cross-region message ratio (0.0-1.0)")

	// 端点配置
	regionAEndpoint = flag.String("region-a", "ws://localhost:8080/ws", "Region A WebSocket endpoint")
	regionBEndpoint = flag.String("region-b", "ws://localhost:8081/ws", "Region B WebSocket endpoint")
	authToken       = flag.String("token", "", "Authentication token")

	// 故障转移配置
	failoverTest     = flag.Bool("failover", false, "Run failover test")
	failoverDelay    = flag.Duration("failover-delay", 30*time.Second, "Delay before triggering failover")
	failoverRecovery = flag.Duration("failover-recovery", 5*time.Second, "Failover recovery time")
	failedRegion     = flag.String("failed-region", "region-a", "Region to fail (region-a or region-b)")

	// 输出配置
	outputFile = flag.String("output", "", "Output file for results (JSON format)")
	verbose    = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	// 验证参数
	if err := validateFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// 运行测试
	var result *loadtest.LoadTestResult
	var err error

	if *failoverTest {
		result, err = runFailoverTest(ctx)
	} else {
		result, err = runBasicTest(ctx)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Test failed: %v\n", err)
		os.Exit(1)
	}

	// 打印结果
	printResults(result)

	// 保存结果到文件
	if *outputFile != "" {
		if err := saveResults(result, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save results: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nResults saved to: %s\n", *outputFile)
	}
}

func validateFlags() error {
	if *connections <= 0 {
		return fmt.Errorf("connections must be positive")
	}
	if *regionAPercent < 0 || *regionAPercent > 100 {
		return fmt.Errorf("region-a-percent must be between 0 and 100")
	}
	if *messageRate <= 0 {
		return fmt.Errorf("rate must be positive")
	}
	if *duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if *crossRegionRatio < 0 || *crossRegionRatio > 1 {
		return fmt.Errorf("cross-region must be between 0.0 and 1.0")
	}
	if *failoverTest {
		if *failedRegion != "region-a" && *failedRegion != "region-b" {
			return fmt.Errorf("failed-region must be 'region-a' or 'region-b'")
		}
	}
	return nil
}

func runBasicTest(ctx context.Context) (*loadtest.LoadTestResult, error) {
	config := &loadtest.LoadTestConfig{
		TotalConnections: *connections,
		RegionAPercent:   *regionAPercent,
		MessageRate:      *messageRate,
		Duration:         *duration,
		RampUpTime:       *rampUpTime,
		CrossRegionRatio: *crossRegionRatio,
		RegionAEndpoint:  *regionAEndpoint,
		RegionBEndpoint:  *regionBEndpoint,
		AuthToken:        *authToken,
	}

	printConfig(config)

	runner := loadtest.NewLoadTestRunner(config)
	defer runner.Cleanup()

	return runner.Run(ctx)
}

func runFailoverTest(ctx context.Context) (*loadtest.LoadTestResult, error) {
	config := &loadtest.FailoverTestConfig{
		LoadTestConfig: loadtest.LoadTestConfig{
			TotalConnections: *connections,
			RegionAPercent:   *regionAPercent,
			MessageRate:      *messageRate,
			Duration:         *duration,
			RampUpTime:       *rampUpTime,
			CrossRegionRatio: *crossRegionRatio,
			RegionAEndpoint:  *regionAEndpoint,
			RegionBEndpoint:  *regionBEndpoint,
			AuthToken:        *authToken,
		},
		FailoverTriggerDelay: *failoverDelay,
		FailoverRecoveryTime: *failoverRecovery,
		FailedRegion:         *failedRegion,
	}

	printFailoverConfig(config)

	runner := loadtest.NewFailoverTestRunner(config)
	defer runner.Cleanup()

	result, err := runner.Run(ctx)
	if err != nil {
		return nil, err
	}

	// 验证无消息丢失
	if err := runner.VerifyNoMessageLoss(); err != nil {
		fmt.Printf("\n⚠️  Warning: %v\n", err)
	} else {
		fmt.Printf("\n✅ Message loss verification passed\n")
	}

	return result, nil
}

func printConfig(config *loadtest.LoadTestConfig) {
	fmt.Println("=== Load Test Configuration ===")
	fmt.Printf("Total Connections: %d\n", config.TotalConnections)
	fmt.Printf("Region A: %d%% (%d connections)\n",
		config.RegionAPercent,
		config.TotalConnections*config.RegionAPercent/100)
	fmt.Printf("Region B: %d%% (%d connections)\n",
		100-config.RegionAPercent,
		config.TotalConnections*(100-config.RegionAPercent)/100)
	fmt.Printf("Message Rate: %d msg/s\n", config.MessageRate)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Ramp-up Time: %v\n", config.RampUpTime)
	fmt.Printf("Cross-Region Ratio: %.1f%%\n", config.CrossRegionRatio*100)
	fmt.Printf("Region A Endpoint: %s\n", config.RegionAEndpoint)
	fmt.Printf("Region B Endpoint: %s\n", config.RegionBEndpoint)
	fmt.Println()
}

func printFailoverConfig(config *loadtest.FailoverTestConfig) {
	printConfig(&config.LoadTestConfig)
	fmt.Println("=== Failover Test Configuration ===")
	fmt.Printf("Failover Trigger Delay: %v\n", config.FailoverTriggerDelay)
	fmt.Printf("Failover Recovery Time: %v\n", config.FailoverRecoveryTime)
	fmt.Printf("Failed Region: %s\n", config.FailedRegion)
	fmt.Println()
}

func printResults(result *loadtest.LoadTestResult) {
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Messages: %d\n", result.TotalMessages)
	fmt.Printf("Success Rate: %.2f%%\n", result.SuccessRate)
	fmt.Printf("Throughput: %.2f msg/s\n", result.Throughput)
	fmt.Println()

	fmt.Println("=== Latency Statistics ===")
	fmt.Printf("P50: %v\n", result.LatencyP50)
	fmt.Printf("P95: %v\n", result.LatencyP95)
	fmt.Printf("P99: %v\n", result.LatencyP99)
	fmt.Printf("Cross-Region P99: %v\n", result.CrossRegionP99)
	fmt.Println()

	// 打印故障转移影响
	if result.FailoverImpact != nil {
		printFailoverImpact(result.FailoverImpact)
	}

	// 性能评估
	printPerformanceAssessment(result)
}

func printFailoverImpact(impact *loadtest.FailoverImpact) {
	fmt.Println("=== Failover Impact ===")

	duration := impact.EndTime.Sub(impact.StartTime)
	fmt.Printf("Failover Duration: %v\n", duration)

	if duration > 30*time.Second {
		fmt.Printf("⚠️  RTO exceeded target (30s): %v\n", duration)
	} else {
		fmt.Printf("✅ RTO within target: %v < 30s\n", duration)
	}

	fmt.Printf("Messages During Failover: %d\n", impact.MessagesDuringFailover)
	fmt.Printf("Failed Messages: %d\n", impact.FailedMessages)
	fmt.Println()

	fmt.Println("=== Throughput Impact ===")
	fmt.Printf("Before Failover: %.2f msg/s\n", impact.ThroughputBefore)
	fmt.Printf("During Failover: %.2f msg/s\n", impact.ThroughputDuring)
	fmt.Printf("After Failover: %.2f msg/s\n", impact.ThroughputAfter)

	if impact.ThroughputBefore > 0 {
		drop := (impact.ThroughputBefore - impact.ThroughputDuring) / impact.ThroughputBefore * 100
		recovery := (impact.ThroughputAfter / impact.ThroughputBefore) * 100
		fmt.Printf("Throughput Drop: %.2f%%\n", drop)
		fmt.Printf("Recovery Rate: %.2f%%\n", recovery)
	}

	if impact.LatencyIncrease > 0 {
		fmt.Printf("Latency Increase: %v\n", impact.LatencyIncrease)
	}
	fmt.Println()
}

func printPerformanceAssessment(result *loadtest.LoadTestResult) {
	fmt.Println("=== Performance Assessment ===")

	passed := true

	// 检查成功率
	if result.SuccessRate >= 99.0 {
		fmt.Printf("✅ Success Rate: %.2f%% (target: ≥99%%)\n", result.SuccessRate)
	} else {
		fmt.Printf("❌ Success Rate: %.2f%% (target: ≥99%%)\n", result.SuccessRate)
		passed = false
	}

	// 检查 P99 延迟
	if result.LatencyP99 <= 500*time.Millisecond {
		fmt.Printf("✅ P99 Latency: %v (target: ≤500ms)\n", result.LatencyP99)
	} else {
		fmt.Printf("❌ P99 Latency: %v (target: ≤500ms)\n", result.LatencyP99)
		passed = false
	}

	// 检查跨地域延迟
	if result.CrossRegionP99 <= 500*time.Millisecond {
		fmt.Printf("✅ Cross-Region P99: %v (target: ≤500ms)\n", result.CrossRegionP99)
	} else {
		fmt.Printf("⚠️  Cross-Region P99: %v (target: ≤500ms)\n", result.CrossRegionP99)
	}

	fmt.Println()
	if passed {
		fmt.Println("🎉 All performance targets met!")
	} else {
		fmt.Println("⚠️  Some performance targets not met")
	}
}

func saveResults(result *loadtest.LoadTestResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
