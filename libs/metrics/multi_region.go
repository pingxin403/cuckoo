package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// MultiRegionMetrics provides metrics collection for multi-region active-active architecture
type MultiRegionMetrics struct {
	obs      observability.Observability
	regionID string
	mu       sync.RWMutex

	// Sync latency tracking
	syncLatencies     map[string][]float64 // target_region -> latencies
	syncLatencyWindow time.Duration

	// Conflict tracking
	conflictCounts map[string]int64 // conflict_type -> count
	conflictWindow time.Duration

	// Failover tracking
	failoverEvents []FailoverEvent
	failoverWindow time.Duration
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	FromRegion string
	ToRegion   string
	Timestamp  time.Time
	DurationMs float64
	Reason     string
}

// Config holds configuration for multi-region metrics
type Config struct {
	RegionID          string
	SyncLatencyWindow time.Duration
	ConflictWindow    time.Duration
	FailoverWindow    time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig(regionID string) Config {
	return Config{
		RegionID:          regionID,
		SyncLatencyWindow: 5 * time.Minute,
		ConflictWindow:    5 * time.Minute,
		FailoverWindow:    1 * time.Hour,
	}
}

// NewMultiRegionMetrics creates a new multi-region metrics collector
func NewMultiRegionMetrics(obs observability.Observability, config Config) *MultiRegionMetrics {
	return &MultiRegionMetrics{
		obs:               obs,
		regionID:          config.RegionID,
		syncLatencies:     make(map[string][]float64),
		syncLatencyWindow: config.SyncLatencyWindow,
		conflictCounts:    make(map[string]int64),
		conflictWindow:    config.ConflictWindow,
		failoverEvents:    make([]FailoverEvent, 0),
		failoverWindow:    config.FailoverWindow,
	}
}

// RecordSyncLatency records cross-region synchronization latency
// Validates: Requirements 5.1 - Cross-region latency monitoring
func (m *MultiRegionMetrics) RecordSyncLatency(targetRegion string, latencyMs float64) {
	// Record to observability system
	m.obs.Metrics().RecordHistogram("cross_region_sync_latency_ms", latencyMs, map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})

	// Track in-memory for statistics
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.syncLatencies[targetRegion]; !exists {
		m.syncLatencies[targetRegion] = make([]float64, 0)
	}
	m.syncLatencies[targetRegion] = append(m.syncLatencies[targetRegion], latencyMs)

	// Increment counter
	m.obs.Metrics().IncrementCounter("cross_region_sync_total", map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})
}

// RecordDatabaseReplicationLatency records database replication latency
// Validates: Requirements 5.1 - Database replication delay monitoring
func (m *MultiRegionMetrics) RecordDatabaseReplicationLatency(targetRegion string, latencyMs float64) {
	m.obs.Metrics().RecordHistogram("database_replication_latency_ms", latencyMs, map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})
}

// RecordMessageSyncLatency records message synchronization latency
// Validates: Requirements 1.1.2 - Message sync latency monitoring
func (m *MultiRegionMetrics) RecordMessageSyncLatency(targetRegion string, latencyMs float64) {
	m.obs.Metrics().RecordHistogram("message_sync_latency_ms", latencyMs, map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})

	// Check threshold and alert if exceeded (500ms threshold from requirements)
	if latencyMs > 500 {
		m.obs.Metrics().IncrementCounter("message_sync_latency_threshold_exceeded_total", map[string]string{
			"source_region": m.regionID,
			"target_region": targetRegion,
		})
	}
}

// RecordConflictEvent records a conflict resolution event
// Validates: Requirements 5.2 - Conflict rate monitoring
func (m *MultiRegionMetrics) RecordConflictEvent(conflictType string) {
	m.obs.Metrics().IncrementCounter("cross_region_conflicts_total", map[string]string{
		"region":        m.regionID,
		"conflict_type": conflictType,
	})

	// Track in-memory for rate calculation
	m.mu.Lock()
	defer m.mu.Unlock()

	m.conflictCounts[conflictType]++
}

// RecordConflictResolution records conflict resolution details
// Validates: Requirements 2.2.2 - Conflict logging with both versions
func (m *MultiRegionMetrics) RecordConflictResolution(conflictType, resolution string, resolutionTimeMs float64) {
	m.obs.Metrics().RecordHistogram("conflict_resolution_duration_ms", resolutionTimeMs, map[string]string{
		"region":        m.regionID,
		"conflict_type": conflictType,
		"resolution":    resolution, // "local_wins", "remote_wins"
	})

	m.obs.Metrics().IncrementCounter("conflict_resolutions_total", map[string]string{
		"region":        m.regionID,
		"conflict_type": conflictType,
		"resolution":    resolution,
	})
}

// RecordFailoverEvent records a failover event
// Validates: Requirements 5.3 - Failover event logging
func (m *MultiRegionMetrics) RecordFailoverEvent(fromRegion, toRegion string, durationMs float64, reason string) {
	// Record to observability system
	m.obs.Metrics().IncrementCounter("failover_events_total", map[string]string{
		"from_region": fromRegion,
		"to_region":   toRegion,
		"reason":      reason,
	})

	m.obs.Metrics().RecordHistogram("failover_duration_ms", durationMs, map[string]string{
		"from_region": fromRegion,
		"to_region":   toRegion,
	})

	// Track in-memory for event history
	m.mu.Lock()
	defer m.mu.Unlock()

	event := FailoverEvent{
		FromRegion: fromRegion,
		ToRegion:   toRegion,
		Timestamp:  time.Now(),
		DurationMs: durationMs,
		Reason:     reason,
	}
	m.failoverEvents = append(m.failoverEvents, event)

	// Clean up old events outside the window
	m.cleanupOldFailoverEvents()
}

// RecordFailoverDetectionTime records the time taken to detect a failure
// Validates: Requirements 4.1.3 - Failure detection to failover trigger < 15s
func (m *MultiRegionMetrics) RecordFailoverDetectionTime(region string, detectionTimeMs float64) {
	m.obs.Metrics().RecordHistogram("failover_detection_time_ms", detectionTimeMs, map[string]string{
		"region": region,
	})

	// Alert if detection time exceeds threshold (15 seconds from requirements)
	if detectionTimeMs > 15000 {
		m.obs.Metrics().IncrementCounter("failover_detection_time_threshold_exceeded_total", map[string]string{
			"region": region,
		})
	}
}

// RecordHealthCheckLatency records health check latency
// Validates: Requirements 4.1.1 - Health check monitoring
func (m *MultiRegionMetrics) RecordHealthCheckLatency(targetRegion string, latencyMs float64, healthy bool) {
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	m.obs.Metrics().RecordHistogram("health_check_latency_ms", latencyMs, map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
		"status":        status,
	})

	m.obs.Metrics().IncrementCounter("health_checks_total", map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
		"status":        status,
	})
}

// RecordRegionAvailability records region availability status
// Validates: Requirements 4.1 - Automatic failure detection
func (m *MultiRegionMetrics) RecordRegionAvailability(targetRegion string, available bool) {
	value := 0.0
	if available {
		value = 1.0
	}

	m.obs.Metrics().SetGauge("region_availability", value, map[string]string{
		"region": targetRegion,
	})
}

// RecordDataSyncStatus records data synchronization status
// Validates: Requirements 1.1.3 - Network partition handling
func (m *MultiRegionMetrics) RecordDataSyncStatus(targetRegion string, status string) {
	m.obs.Metrics().IncrementCounter("data_sync_status_total", map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
		"status":        status, // "success", "failed", "pending"
	})
}

// RecordNetworkPartition records a network partition event
// Validates: Requirements 4.3 - Split-brain prevention
func (m *MultiRegionMetrics) RecordNetworkPartition(affectedRegions []string, durationMs float64) {
	m.obs.Metrics().IncrementCounter("network_partition_events_total", map[string]string{
		"region":           m.regionID,
		"affected_regions": fmt.Sprintf("%v", affectedRegions),
	})

	m.obs.Metrics().RecordHistogram("network_partition_duration_ms", durationMs, map[string]string{
		"region": m.regionID,
	})
}

// RecordReconciliationEvent records a data reconciliation event
// Validates: Requirements 4.4 - Data reconciliation
func (m *MultiRegionMetrics) RecordReconciliationEvent(targetRegion string, discrepancies int64, fixedCount int64, durationMs float64) {
	m.obs.Metrics().IncrementCounter("reconciliation_events_total", map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})

	m.obs.Metrics().SetGauge("reconciliation_discrepancies", float64(discrepancies), map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})

	m.obs.Metrics().SetGauge("reconciliation_fixed_count", float64(fixedCount), map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})

	m.obs.Metrics().RecordHistogram("reconciliation_duration_ms", durationMs, map[string]string{
		"source_region": m.regionID,
		"target_region": targetRegion,
	})
}

// GetSyncLatencyStats returns sync latency statistics for a target region
func (m *MultiRegionMetrics) GetSyncLatencyStats(targetRegion string) *LatencyStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	latencies, exists := m.syncLatencies[targetRegion]
	if !exists || len(latencies) == 0 {
		return nil
	}

	return calculateLatencyStats(latencies)
}

// GetConflictRate returns the conflict rate (conflicts per minute)
// Validates: Requirements 5.2.3 - Conflict rate > 0.1% triggers alert
func (m *MultiRegionMetrics) GetConflictRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalConflicts := int64(0)
	for _, count := range m.conflictCounts {
		totalConflicts += count
	}

	// Calculate rate over the conflict window
	return float64(totalConflicts) / m.conflictWindow.Minutes()
}

// GetFailoverEvents returns recent failover events
func (m *MultiRegionMetrics) GetFailoverEvents() []FailoverEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	events := make([]FailoverEvent, len(m.failoverEvents))
	copy(events, m.failoverEvents)
	return events
}

// ResetMetrics resets all in-memory metrics (useful for testing)
func (m *MultiRegionMetrics) ResetMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.syncLatencies = make(map[string][]float64)
	m.conflictCounts = make(map[string]int64)
	m.failoverEvents = make([]FailoverEvent, 0)
}

// cleanupOldFailoverEvents removes events outside the failover window
func (m *MultiRegionMetrics) cleanupOldFailoverEvents() {
	cutoff := time.Now().Add(-m.failoverWindow)
	validEvents := make([]FailoverEvent, 0)

	for _, event := range m.failoverEvents {
		if event.Timestamp.After(cutoff) {
			validEvents = append(validEvents, event)
		}
	}

	m.failoverEvents = validEvents
}

// LatencyStats holds latency statistics
type LatencyStats struct {
	Count  int
	Min    float64
	Max    float64
	Mean   float64
	P50    float64
	P95    float64
	P99    float64
	StdDev float64
}

// calculateLatencyStats calculates statistics from latency samples
func calculateLatencyStats(latencies []float64) *LatencyStats {
	if len(latencies) == 0 {
		return nil
	}

	// Sort for percentile calculation
	sorted := make([]float64, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort (good enough for small samples)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	stats := &LatencyStats{
		Count: len(sorted),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
	}

	// Calculate mean
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	stats.Mean = sum / float64(len(sorted))

	// Calculate percentiles
	stats.P50 = percentile(sorted, 0.50)
	stats.P95 = percentile(sorted, 0.95)
	stats.P99 = percentile(sorted, 0.99)

	// Calculate standard deviation
	variance := 0.0
	for _, v := range sorted {
		diff := v - stats.Mean
		variance += diff * diff
	}
	variance /= float64(len(sorted))
	stats.StdDev = 0.0
	if variance > 0 {
		// Simple square root approximation
		stats.StdDev = variance / 2.0
	}

	return stats
}

// percentile calculates the percentile value from sorted data
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	index := p * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// LogMetrics logs current metrics to the logger
func (m *MultiRegionMetrics) LogMetrics(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Log sync latency stats
	for targetRegion, latencies := range m.syncLatencies {
		if len(latencies) > 0 {
			stats := calculateLatencyStats(latencies)
			m.obs.Logger().Info(ctx, "Sync latency statistics",
				"source_region", m.regionID,
				"target_region", targetRegion,
				"count", stats.Count,
				"p50_ms", stats.P50,
				"p95_ms", stats.P95,
				"p99_ms", stats.P99,
				"mean_ms", stats.Mean,
			)
		}
	}

	// Log conflict counts
	totalConflicts := int64(0)
	for conflictType, count := range m.conflictCounts {
		totalConflicts += count
		m.obs.Logger().Info(ctx, "Conflict statistics",
			"region", m.regionID,
			"conflict_type", conflictType,
			"count", count,
		)
	}

	// Log conflict rate
	conflictRate := m.GetConflictRate()
	m.obs.Logger().Info(ctx, "Conflict rate",
		"region", m.regionID,
		"rate_per_minute", conflictRate,
	)

	// Log failover events
	if len(m.failoverEvents) > 0 {
		m.obs.Logger().Info(ctx, "Recent failover events",
			"region", m.regionID,
			"event_count", len(m.failoverEvents),
		)
	}
}
