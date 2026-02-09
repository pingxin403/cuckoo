package metrics_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/pingxin403/cuckoo/libs/metrics"
)

// Example demonstrates basic usage of multi-region metrics
func Example() {
	// Initialize observability
	obs, err := observability.New(observability.Config{
		ServiceName:   "im-service",
		EnableMetrics: false, // Disable for example
		LogLevel:      "info",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer obs.Shutdown(context.Background())

	// Create multi-region metrics
	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record sync latency
	mrMetrics.RecordSyncLatency("region-b", 150.0)
	mrMetrics.RecordSyncLatency("region-b", 200.0)
	mrMetrics.RecordSyncLatency("region-b", 180.0)

	// Get statistics
	stats := mrMetrics.GetSyncLatencyStats("region-b")
	if stats != nil {
		fmt.Printf("Sync latency stats: P50=%.0fms, P95=%.0fms, P99=%.0fms\n",
			stats.P50, stats.P95, stats.P99)
	}

	// Output:
	// Sync latency stats: P50=180ms, P95=198ms, P99=200ms
}

// ExampleMultiRegionMetrics_RecordSyncLatency demonstrates recording sync latency
func ExampleMultiRegionMetrics_RecordSyncLatency() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record cross-region sync latency
	mrMetrics.RecordSyncLatency("region-b", 150.0)

	// Record message-specific sync latency
	mrMetrics.RecordMessageSyncLatency("region-b", 300.0)

	// Record database replication latency
	mrMetrics.RecordDatabaseReplicationLatency("region-b", 800.0)

	fmt.Println("Sync latencies recorded")
	// Output: Sync latencies recorded
}

// ExampleMultiRegionMetrics_RecordConflictEvent demonstrates conflict tracking
func ExampleMultiRegionMetrics_RecordConflictEvent() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	config.ConflictWindow = 1 * time.Minute
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record conflicts
	mrMetrics.RecordConflictEvent("message_conflict")
	mrMetrics.RecordConflictEvent("message_conflict")
	mrMetrics.RecordConflictEvent("session_conflict")

	// Record conflict resolution
	mrMetrics.RecordConflictResolution("message_conflict", "local_wins", 5.0)

	// Get conflict rate
	rate := mrMetrics.GetConflictRate()
	fmt.Printf("Conflict rate: %.1f conflicts/minute\n", rate)

	// Output: Conflict rate: 3.0 conflicts/minute
}

// ExampleMultiRegionMetrics_RecordFailoverEvent demonstrates failover tracking
func ExampleMultiRegionMetrics_RecordFailoverEvent() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record failover event
	mrMetrics.RecordFailoverEvent(
		"region-a",
		"region-b",
		25000.0,
		"health_check_failed",
	)

	// Record detection time
	mrMetrics.RecordFailoverDetectionTime("region-a", 12000.0)

	// Get failover events
	events := mrMetrics.GetFailoverEvents()
	if len(events) > 0 {
		event := events[0]
		fmt.Printf("Failover: %s -> %s (%.0fms)\n",
			event.FromRegion, event.ToRegion, event.DurationMs)
	}

	// Output: Failover: region-a -> region-b (25000ms)
}

// ExampleMultiRegionMetrics_RecordHealthCheckLatency demonstrates health monitoring
func ExampleMultiRegionMetrics_RecordHealthCheckLatency() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record healthy check
	mrMetrics.RecordHealthCheckLatency("region-b", 50.0, true)

	// Record unhealthy check
	mrMetrics.RecordHealthCheckLatency("region-c", 5000.0, false)

	// Update availability
	mrMetrics.RecordRegionAvailability("region-b", true)
	mrMetrics.RecordRegionAvailability("region-c", false)

	fmt.Println("Health checks recorded")
	// Output: Health checks recorded
}

// ExampleMultiRegionMetrics_GetSyncLatencyStats demonstrates statistics retrieval
func ExampleMultiRegionMetrics_GetSyncLatencyStats() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record multiple latencies
	latencies := []float64{100, 150, 200, 250, 300, 350, 400, 450, 500}
	for _, latency := range latencies {
		mrMetrics.RecordSyncLatency("region-b", latency)
	}

	// Get statistics
	stats := mrMetrics.GetSyncLatencyStats("region-b")
	if stats != nil {
		fmt.Printf("Count: %d\n", stats.Count)
		fmt.Printf("Min: %.0fms\n", stats.Min)
		fmt.Printf("Max: %.0fms\n", stats.Max)
		fmt.Printf("Mean: %.0fms\n", stats.Mean)
		fmt.Printf("P50: %.0fms\n", stats.P50)
	}

	// Output:
	// Count: 9
	// Min: 100ms
	// Max: 500ms
	// Mean: 300ms
	// P50: 300ms
}

// ExampleMultiRegionMetrics_RecordReconciliationEvent demonstrates reconciliation tracking
func ExampleMultiRegionMetrics_RecordReconciliationEvent() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record reconciliation event
	discrepancies := int64(10)
	fixedCount := int64(8)
	durationMs := 5000.0

	mrMetrics.RecordReconciliationEvent("region-b", discrepancies, fixedCount, durationMs)

	fmt.Printf("Reconciliation: %d discrepancies, %d fixed\n", discrepancies, fixedCount)
	// Output: Reconciliation: 10 discrepancies, 8 fixed
}

// ExampleMultiRegionMetrics_RecordNetworkPartition demonstrates network partition tracking
func ExampleMultiRegionMetrics_RecordNetworkPartition() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record network partition
	affectedRegions := []string{"region-a", "region-b"}
	durationMs := 60000.0 // 1 minute

	mrMetrics.RecordNetworkPartition(affectedRegions, durationMs)

	fmt.Printf("Network partition: %v (%.0fs)\n", affectedRegions, durationMs/1000)
	// Output: Network partition: [region-a region-b] (60s)
}

// ExampleMultiRegionMetrics_LogMetrics demonstrates metrics logging
func ExampleMultiRegionMetrics_LogMetrics() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
		LogLevel:      "error", // Suppress logs for example
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record some metrics
	mrMetrics.RecordSyncLatency("region-b", 150.0)
	mrMetrics.RecordConflictEvent("message_conflict")

	// Log all metrics
	ctx := context.Background()
	mrMetrics.LogMetrics(ctx)

	fmt.Println("Metrics logged")
	// Output: Metrics logged
}

// ExampleMultiRegionMetrics_ResetMetrics demonstrates metrics reset
func ExampleMultiRegionMetrics_ResetMetrics() {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	defer obs.Shutdown(context.Background())

	config := metrics.DefaultConfig("region-a")
	mrMetrics := metrics.NewMultiRegionMetrics(obs, config)

	// Record some metrics
	mrMetrics.RecordSyncLatency("region-b", 150.0)
	mrMetrics.RecordConflictEvent("message_conflict")

	// Reset all metrics
	mrMetrics.ResetMetrics()

	// Verify reset
	stats := mrMetrics.GetSyncLatencyStats("region-b")
	rate := mrMetrics.GetConflictRate()

	fmt.Printf("Stats after reset: %v\n", stats)
	fmt.Printf("Conflict rate after reset: %.1f\n", rate)

	// Output:
	// Stats after reset: <nil>
	// Conflict rate after reset: 0.0
}
