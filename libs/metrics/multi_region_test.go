package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

func TestNewMultiRegionMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false, // Disable for testing
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	if metrics == nil {
		t.Fatal("Expected non-nil metrics")
	}

	if metrics.regionID != "region-a" {
		t.Errorf("Expected regionID 'region-a', got '%s'", metrics.regionID)
	}

	if metrics.syncLatencyWindow != 5*time.Minute {
		t.Errorf("Expected syncLatencyWindow 5m, got %v", metrics.syncLatencyWindow)
	}
}

func TestRecordSyncLatency(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record some latencies
	metrics.RecordSyncLatency("region-b", 100.0)
	metrics.RecordSyncLatency("region-b", 200.0)
	metrics.RecordSyncLatency("region-b", 150.0)

	// Get stats
	stats := metrics.GetSyncLatencyStats("region-b")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.Count != 3 {
		t.Errorf("Expected count 3, got %d", stats.Count)
	}

	if stats.Min != 100.0 {
		t.Errorf("Expected min 100.0, got %f", stats.Min)
	}

	if stats.Max != 200.0 {
		t.Errorf("Expected max 200.0, got %f", stats.Max)
	}

	expectedMean := 150.0
	if stats.Mean != expectedMean {
		t.Errorf("Expected mean %f, got %f", expectedMean, stats.Mean)
	}
}

func TestRecordMessageSyncLatency(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Test normal latency (below threshold)
	metrics.RecordMessageSyncLatency("region-b", 300.0)

	// Test high latency (above 500ms threshold)
	metrics.RecordMessageSyncLatency("region-b", 600.0)

	// No errors should occur
}

func TestRecordConflictEvent(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record conflicts
	metrics.RecordConflictEvent("message_conflict")
	metrics.RecordConflictEvent("message_conflict")
	metrics.RecordConflictEvent("session_conflict")

	// Check conflict counts
	metrics.mu.RLock()
	messageConflicts := metrics.conflictCounts["message_conflict"]
	sessionConflicts := metrics.conflictCounts["session_conflict"]
	metrics.mu.RUnlock()

	if messageConflicts != 2 {
		t.Errorf("Expected 2 message conflicts, got %d", messageConflicts)
	}

	if sessionConflicts != 1 {
		t.Errorf("Expected 1 session conflict, got %d", sessionConflicts)
	}
}

func TestRecordConflictResolution(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record conflict resolutions
	metrics.RecordConflictResolution("message_conflict", "local_wins", 5.0)
	metrics.RecordConflictResolution("message_conflict", "remote_wins", 3.0)

	// No errors should occur
}

func TestRecordFailoverEvent(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record failover events
	metrics.RecordFailoverEvent("region-a", "region-b", 25000.0, "health_check_failed")
	metrics.RecordFailoverEvent("region-b", "region-a", 30000.0, "manual_failover")

	// Get events
	events := metrics.GetFailoverEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 failover events, got %d", len(events))
	}

	// Check first event
	if events[0].FromRegion != "region-a" {
		t.Errorf("Expected from_region 'region-a', got '%s'", events[0].FromRegion)
	}

	if events[0].ToRegion != "region-b" {
		t.Errorf("Expected to_region 'region-b', got '%s'", events[0].ToRegion)
	}

	if events[0].DurationMs != 25000.0 {
		t.Errorf("Expected duration 25000.0, got %f", events[0].DurationMs)
	}

	if events[0].Reason != "health_check_failed" {
		t.Errorf("Expected reason 'health_check_failed', got '%s'", events[0].Reason)
	}
}

func TestRecordFailoverDetectionTime(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Test normal detection time (below 15s threshold)
	metrics.RecordFailoverDetectionTime("region-b", 10000.0)

	// Test slow detection time (above 15s threshold)
	metrics.RecordFailoverDetectionTime("region-b", 20000.0)

	// No errors should occur
}

func TestRecordHealthCheckLatency(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record healthy check
	metrics.RecordHealthCheckLatency("region-b", 50.0, true)

	// Record unhealthy check
	metrics.RecordHealthCheckLatency("region-b", 5000.0, false)

	// No errors should occur
}

func TestRecordRegionAvailability(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record availability
	metrics.RecordRegionAvailability("region-b", true)
	metrics.RecordRegionAvailability("region-c", false)

	// No errors should occur
}

func TestRecordDataSyncStatus(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record sync statuses
	metrics.RecordDataSyncStatus("region-b", "success")
	metrics.RecordDataSyncStatus("region-b", "failed")
	metrics.RecordDataSyncStatus("region-b", "pending")

	// No errors should occur
}

func TestRecordNetworkPartition(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record network partition
	affectedRegions := []string{"region-a", "region-b"}
	metrics.RecordNetworkPartition(affectedRegions, 60000.0)

	// No errors should occur
}

func TestRecordReconciliationEvent(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record reconciliation event
	metrics.RecordReconciliationEvent("region-b", 10, 8, 5000.0)

	// No errors should occur
}

func TestGetConflictRate(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	config.ConflictWindow = 1 * time.Minute
	metrics := NewMultiRegionMetrics(obs, config)

	// Record conflicts
	for i := 0; i < 10; i++ {
		metrics.RecordConflictEvent("message_conflict")
	}

	// Get conflict rate
	rate := metrics.GetConflictRate()
	expectedRate := 10.0 // 10 conflicts per minute

	if rate != expectedRate {
		t.Errorf("Expected conflict rate %f, got %f", expectedRate, rate)
	}
}

func TestResetMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record some data
	metrics.RecordSyncLatency("region-b", 100.0)
	metrics.RecordConflictEvent("message_conflict")
	metrics.RecordFailoverEvent("region-a", "region-b", 25000.0, "test")

	// Reset
	metrics.ResetMetrics()

	// Verify reset
	stats := metrics.GetSyncLatencyStats("region-b")
	if stats != nil {
		t.Error("Expected nil stats after reset")
	}

	rate := metrics.GetConflictRate()
	if rate != 0.0 {
		t.Errorf("Expected conflict rate 0.0 after reset, got %f", rate)
	}

	events := metrics.GetFailoverEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 failover events after reset, got %d", len(events))
	}
}

func TestCalculateLatencyStats(t *testing.T) {
	tests := []struct {
		name      string
		latencies []float64
		wantNil   bool
		wantCount int
		wantMin   float64
		wantMax   float64
		wantMean  float64
	}{
		{
			name:      "empty slice",
			latencies: []float64{},
			wantNil:   true,
		},
		{
			name:      "single value",
			latencies: []float64{100.0},
			wantCount: 1,
			wantMin:   100.0,
			wantMax:   100.0,
			wantMean:  100.0,
		},
		{
			name:      "multiple values",
			latencies: []float64{100.0, 200.0, 150.0, 300.0, 250.0},
			wantCount: 5,
			wantMin:   100.0,
			wantMax:   300.0,
			wantMean:  200.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateLatencyStats(tt.latencies)

			if tt.wantNil {
				if stats != nil {
					t.Error("Expected nil stats")
				}
				return
			}

			if stats == nil {
				t.Fatal("Expected non-nil stats")
			}

			if stats.Count != tt.wantCount {
				t.Errorf("Count: got %d, want %d", stats.Count, tt.wantCount)
			}

			if stats.Min != tt.wantMin {
				t.Errorf("Min: got %f, want %f", stats.Min, tt.wantMin)
			}

			if stats.Max != tt.wantMax {
				t.Errorf("Max: got %f, want %f", stats.Max, tt.wantMax)
			}

			if stats.Mean != tt.wantMean {
				t.Errorf("Mean: got %f, want %f", stats.Mean, tt.wantMean)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name   string
		data   []float64
		p      float64
		want   float64
		approx bool // Allow approximate comparison
	}{
		{
			name: "empty slice",
			data: []float64{},
			p:    0.5,
			want: 0.0,
		},
		{
			name: "single value",
			data: []float64{100.0},
			p:    0.5,
			want: 100.0,
		},
		{
			name: "p50 of sorted data",
			data: []float64{100.0, 200.0, 300.0, 400.0, 500.0},
			p:    0.5,
			want: 300.0,
		},
		{
			name:   "p95 of sorted data",
			data:   []float64{100.0, 200.0, 300.0, 400.0, 500.0},
			p:      0.95,
			want:   500.0,
			approx: true,
		},
		{
			name:   "p99 of sorted data",
			data:   []float64{100.0, 200.0, 300.0, 400.0, 500.0},
			p:      0.99,
			want:   500.0,
			approx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.data, tt.p)

			if tt.approx {
				// Allow 10% difference for approximate comparisons
				diff := got - tt.want
				if diff < 0 {
					diff = -diff
				}
				if diff > tt.want*0.1 {
					t.Errorf("percentile() = %f, want approximately %f", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("percentile() = %f, want %f", got, tt.want)
				}
			}
		})
	}
}

func TestLogMetrics(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
		LogLevel:      "info",
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record some data
	metrics.RecordSyncLatency("region-b", 100.0)
	metrics.RecordSyncLatency("region-b", 200.0)
	metrics.RecordConflictEvent("message_conflict")
	metrics.RecordFailoverEvent("region-a", "region-b", 25000.0, "test")

	// Log metrics (should not panic)
	ctx := context.Background()
	metrics.LogMetrics(ctx)
}

func TestCleanupOldFailoverEvents(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	config.FailoverWindow = 100 * time.Millisecond
	metrics := NewMultiRegionMetrics(obs, config)

	// Record an event
	metrics.RecordFailoverEvent("region-a", "region-b", 25000.0, "test")

	// Wait for event to expire
	time.Sleep(150 * time.Millisecond)

	// Record another event (this will trigger cleanup)
	metrics.RecordFailoverEvent("region-b", "region-a", 30000.0, "test2")

	// Should only have the recent event
	events := metrics.GetFailoverEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 failover event after cleanup, got %d", len(events))
	}

	if events[0].Reason != "test2" {
		t.Errorf("Expected recent event with reason 'test2', got '%s'", events[0].Reason)
	}
}

func TestConcurrentMetricsRecording(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:   "test-service",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	config := DefaultConfig("region-a")
	metrics := NewMultiRegionMetrics(obs, config)

	// Record metrics concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metrics.RecordSyncLatency("region-b", float64(j))
				metrics.RecordConflictEvent("message_conflict")
				metrics.RecordFailoverEvent("region-a", "region-b", 25000.0, "test")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify data was recorded
	stats := metrics.GetSyncLatencyStats("region-b")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.Count != 1000 {
		t.Errorf("Expected count 1000, got %d", stats.Count)
	}

	rate := metrics.GetConflictRate()
	if rate == 0.0 {
		t.Error("Expected non-zero conflict rate")
	}

	events := metrics.GetFailoverEvents()
	if len(events) == 0 {
		t.Error("Expected some failover events")
	}
}
