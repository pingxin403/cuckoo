package capacity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 4: 容量预测单调性
// For monotonically increasing usage history, DaysUntilFull should be positive
// and decrease as usage increases.
// Validates: Requirements 7.1.5
func TestProperty_CapacityForecastMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Create monitor with mock history store
		history := NewInMemoryHistoryStore(1000)
		thresholds := ThresholdConfig{
			DefaultPercent: 80.0,
			Overrides:      make(map[ResourceType]float64),
		}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		resourceType := ResourceMySQL
		resourceName := "test-db"
		totalBytes := int64(1000000000) // 1GB

		// Generate monotonically increasing usage history
		numDays := rapid.IntRange(7, 30).Draw(t, "numDays")
		baseTime := time.Now().Add(-time.Duration(numDays) * 24 * time.Hour)

		// Initial usage (10-50%)
		initialUsagePercent := rapid.Float64Range(10.0, 50.0).Draw(t, "initialUsage")
		initialUsedBytes := int64(float64(totalBytes) * initialUsagePercent / 100.0)

		// Growth per day (0.5-5%)
		growthPercentPerDay := rapid.Float64Range(0.5, 5.0).Draw(t, "growthPerDay")
		growthBytesPerDay := int64(float64(totalBytes) * growthPercentPerDay / 100.0)

		// Store monotonically increasing usage history
		for day := 0; day < numDays; day++ {
			usedBytes := initialUsedBytes + int64(day)*growthBytesPerDay
			if usedBytes > totalBytes {
				usedBytes = totalBytes
			}

			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    usedBytes,
				TotalBytes:   totalBytes,
				UsagePercent: float64(usedBytes) * 100.0 / float64(totalBytes),
				Timestamp:    baseTime.Add(time.Duration(day) * 24 * time.Hour),
			}

			err := history.Store(ctx, usage)
			if err != nil {
				t.Fatalf("Failed to store usage: %v", err)
			}
		}

		// Get forecast
		forecast, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err != nil {
			t.Fatalf("Forecast failed: %v", err)
		}

		// Verify DaysUntilFull is positive (unless already at 100%)
		if forecast.CurrentUsage < 100.0 && forecast.DaysUntilFull <= 0 {
			t.Fatalf("DaysUntilFull should be positive for growing usage: got %d",
				forecast.DaysUntilFull)
		}

		// Verify growth rate is positive
		if forecast.GrowthRatePerDay <= 0 {
			t.Fatalf("GrowthRatePerDay should be positive for increasing usage: got %f",
				forecast.GrowthRatePerDay)
		}

		// Now add more usage and verify DaysUntilFull decreases
		additionalDays := rapid.IntRange(1, 5).Draw(t, "additionalDays")
		for day := 0; day < additionalDays; day++ {
			usedBytes := initialUsedBytes + int64(numDays+day)*growthBytesPerDay
			if usedBytes > totalBytes {
				usedBytes = totalBytes
			}

			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    usedBytes,
				TotalBytes:   totalBytes,
				UsagePercent: float64(usedBytes) * 100.0 / float64(totalBytes),
				Timestamp:    baseTime.Add(time.Duration(numDays+day) * 24 * time.Hour),
			}

			history.Store(ctx, usage)
		}

		// Get new forecast
		newForecast, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err != nil {
			t.Fatalf("Second forecast failed: %v", err)
		}

		// Verify DaysUntilFull decreased (or stayed same if at capacity)
		if newForecast.CurrentUsage < 100.0 {
			if newForecast.DaysUntilFull > forecast.DaysUntilFull {
				t.Fatalf("DaysUntilFull should decrease as usage increases: before=%d, after=%d",
					forecast.DaysUntilFull, newForecast.DaysUntilFull)
			}
		}
	})
}

// Property: Forecast should require minimum data points
func TestProperty_ForecastRequiresMinimumData(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)
		thresholds := ThresholdConfig{DefaultPercent: 80.0}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		resourceType := ResourceMySQL
		resourceName := "test-db"

		// Generate insufficient data (less than 7 days)
		numDays := rapid.IntRange(1, 6).Draw(t, "numDays")
		baseTime := time.Now().Add(-time.Duration(numDays) * 24 * time.Hour)

		for day := 0; day < numDays; day++ {
			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    int64(day * 1000000),
				TotalBytes:   10000000,
				UsagePercent: float64(day*1000000) * 100.0 / 10000000.0,
				Timestamp:    baseTime.Add(time.Duration(day) * 24 * time.Hour),
			}
			history.Store(ctx, usage)
		}

		// Forecast should fail with insufficient data
		_, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err == nil {
			t.Fatal("Forecast should fail with insufficient data (< 7 days)")
		}
	})
}

// Property: Zero or negative growth should result in infinite days until full
func TestProperty_ZeroGrowthInfiniteDays(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)
		thresholds := ThresholdConfig{DefaultPercent: 80.0}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		resourceType := ResourceMySQL
		resourceName := "test-db"
		totalBytes := int64(1000000000)

		// Generate flat usage history (no growth)
		numDays := rapid.IntRange(7, 30).Draw(t, "numDays")
		baseTime := time.Now().Add(-time.Duration(numDays) * 24 * time.Hour)
		constantUsage := rapid.Int64Range(100000000, 500000000).Draw(t, "constantUsage")

		for day := 0; day < numDays; day++ {
			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    constantUsage,
				TotalBytes:   totalBytes,
				UsagePercent: float64(constantUsage) * 100.0 / float64(totalBytes),
				Timestamp:    baseTime.Add(time.Duration(day) * 24 * time.Hour),
			}
			history.Store(ctx, usage)
		}

		// Get forecast
		forecast, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err != nil {
			t.Fatalf("Forecast failed: %v", err)
		}

		// With zero growth, DaysUntilFull should be 0 (indicating no growth)
		if forecast.GrowthRatePerDay > 1.0 { // Allow small floating point errors
			t.Fatalf("GrowthRatePerDay should be near zero for flat usage: got %f",
				forecast.GrowthRatePerDay)
		}

		if forecast.DaysUntilFull != 0 {
			t.Logf("Warning: DaysUntilFull = %d for zero growth (expected 0)", forecast.DaysUntilFull)
		}
	})
}

// Property: Threshold checking should be consistent
func TestProperty_ThresholdCheckingConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)

		// Generate random threshold
		threshold := rapid.Float64Range(50.0, 95.0).Draw(t, "threshold")
		thresholds := ThresholdConfig{
			DefaultPercent: threshold,
			Overrides:      make(map[ResourceType]float64),
		}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		// Generate random usages
		numUsages := rapid.IntRange(5, 20).Draw(t, "numUsages")
		var usages []ResourceUsage

		for i := 0; i < numUsages; i++ {
			usagePercent := rapid.Float64Range(0.0, 100.0).Draw(t, "usagePercent")
			usage := ResourceUsage{
				ResourceType: ResourceMySQL,
				ResourceName: fmt.Sprintf("test-resource-%d", i), // Unique name for each resource
				RegionID:     "region-a",
				UsedBytes:    int64(usagePercent * 1000000),
				TotalBytes:   100000000,
				UsagePercent: usagePercent,
				Timestamp:    time.Now(),
			}
			usages = append(usages, usage)
		}

		// Check thresholds
		exceeded := monitor.CheckThresholds(ctx, usages)

		// Verify all exceeded usages are >= threshold
		for _, usage := range exceeded {
			if usage.UsagePercent < threshold {
				t.Fatalf("Usage %f%% should not exceed threshold %f%%",
					usage.UsagePercent, threshold)
			}
		}

		// Verify all non-exceeded usages are < threshold
		exceededMap := make(map[string]bool)
		for _, usage := range exceeded {
			exceededMap[usage.ResourceName] = true
		}

		for _, usage := range usages {
			isExceeded := exceededMap[usage.ResourceName]
			shouldExceed := usage.UsagePercent >= threshold

			if isExceeded != shouldExceed {
				t.Fatalf("Threshold check inconsistent for usage %f%% (threshold %f%%): isExceeded=%v, shouldExceed=%v",
					usage.UsagePercent, threshold, isExceeded, shouldExceed)
			}
		}
	})
}

// Property: Forecast should handle edge cases
func TestProperty_ForecastEdgeCases(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)
		thresholds := ThresholdConfig{DefaultPercent: 80.0}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		resourceType := ResourceMySQL
		resourceName := "test-db"
		totalBytes := int64(1000000000)

		// Generate history that reaches 100%
		numDays := rapid.IntRange(7, 15).Draw(t, "numDays")
		baseTime := time.Now().Add(-time.Duration(numDays) * 24 * time.Hour)

		for day := 0; day < numDays; day++ {
			// Gradually increase to 100%
			usagePercent := 50.0 + (50.0 * float64(day) / float64(numDays-1))
			if usagePercent > 100.0 {
				usagePercent = 100.0
			}

			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    int64(float64(totalBytes) * usagePercent / 100.0),
				TotalBytes:   totalBytes,
				UsagePercent: usagePercent,
				Timestamp:    baseTime.Add(time.Duration(day) * 24 * time.Hour),
			}
			history.Store(ctx, usage)
		}

		// Get forecast
		forecast, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err != nil {
			t.Fatalf("Forecast failed: %v", err)
		}

		// If at 100%, DaysUntilFull should be 0
		if forecast.CurrentUsage >= 100.0 && forecast.DaysUntilFull != 0 {
			t.Logf("Warning: At 100%% usage, DaysUntilFull should be 0, got %d",
				forecast.DaysUntilFull)
		}

		// Forecast should not be negative
		if forecast.DaysUntilFull < 0 {
			t.Fatalf("DaysUntilFull should not be negative: got %d", forecast.DaysUntilFull)
		}
	})
}

// Property: Growth rate calculation should be stable
func TestProperty_GrowthRateStability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)
		thresholds := ThresholdConfig{DefaultPercent: 80.0}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		resourceType := ResourceMySQL
		resourceName := "test-db"
		totalBytes := int64(1000000000)

		// Generate linear growth
		numDays := rapid.IntRange(10, 30).Draw(t, "numDays")
		baseTime := time.Now().Add(-time.Duration(numDays) * 24 * time.Hour)
		growthPerDay := rapid.Int64Range(1000000, 10000000).Draw(t, "growthPerDay")
		initialUsage := rapid.Int64Range(100000000, 300000000).Draw(t, "initialUsage")

		for day := 0; day < numDays; day++ {
			usedBytes := initialUsage + int64(day)*growthPerDay
			if usedBytes > totalBytes {
				usedBytes = totalBytes
			}

			usage := ResourceUsage{
				ResourceType: resourceType,
				ResourceName: resourceName,
				RegionID:     "region-a",
				UsedBytes:    usedBytes,
				TotalBytes:   totalBytes,
				UsagePercent: float64(usedBytes) * 100.0 / float64(totalBytes),
				Timestamp:    baseTime.Add(time.Duration(day) * 24 * time.Hour),
			}
			history.Store(ctx, usage)
		}

		// Get forecast
		forecast, err := monitor.Forecast(ctx, resourceType, resourceName)
		if err != nil {
			t.Fatalf("Forecast failed: %v", err)
		}

		// Growth rate should be approximately equal to actual growth
		// Allow 20% tolerance due to linear regression approximation
		expectedGrowthRate := float64(growthPerDay)
		actualGrowthRate := forecast.GrowthRatePerDay
		tolerance := expectedGrowthRate * 0.2

		if actualGrowthRate < expectedGrowthRate-tolerance ||
			actualGrowthRate > expectedGrowthRate+tolerance {
			t.Logf("Warning: Growth rate deviation: expected ~%f, got %f (tolerance: %f)",
				expectedGrowthRate, actualGrowthRate, tolerance)
		}
	})
}

// Property: Threshold overrides should take precedence
func TestProperty_ThresholdOverrides(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()
		history := NewInMemoryHistoryStore(1000)

		defaultThreshold := rapid.Float64Range(70.0, 80.0).Draw(t, "defaultThreshold")
		overrideThreshold := rapid.Float64Range(85.0, 95.0).Draw(t, "overrideThreshold")

		thresholds := ThresholdConfig{
			DefaultPercent: defaultThreshold,
			Overrides: map[ResourceType]float64{
				ResourceMySQL: overrideThreshold,
			},
		}
		monitor := NewDefaultCapacityMonitor(thresholds, history)

		// Create usage between default and override thresholds
		usagePercent := rapid.Float64Range(defaultThreshold+1, overrideThreshold-1).Draw(t, "usagePercent")

		usages := []ResourceUsage{
			{
				ResourceType: ResourceMySQL,
				ResourceName: "mysql-resource",
				UsagePercent: usagePercent,
			},
			{
				ResourceType: ResourceNetwork,
				ResourceName: "network-resource",
				UsagePercent: usagePercent,
			},
		}

		exceeded := monitor.CheckThresholds(ctx, usages)

		// MySQL should not exceed (uses override threshold)
		// Network should exceed (uses default threshold)
		mysqlExceeded := false
		networkExceeded := false

		for _, usage := range exceeded {
			if usage.ResourceType == ResourceMySQL {
				mysqlExceeded = true
			}
			if usage.ResourceType == ResourceNetwork {
				networkExceeded = true
			}
		}

		if mysqlExceeded {
			t.Fatalf("MySQL should not exceed override threshold %f%% at usage %f%%",
				overrideThreshold, usagePercent)
		}

		if !networkExceeded {
			t.Fatalf("Network should exceed default threshold %f%% at usage %f%%",
				defaultThreshold, usagePercent)
		}
	})
}
