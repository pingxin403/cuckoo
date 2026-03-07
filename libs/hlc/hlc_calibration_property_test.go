package hlc

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 7: HLC 校准后单调性保持
// For any sequence of drift adjustments and ID generations,
// the generated IDs should always be monotonically increasing.
// Validates: Requirements 8.2.4
func TestProperty_HLCCalibrationMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		regionID := rapid.StringMatching(`region-[a-z]`).Draw(t, "regionID")
		nodeID := rapid.StringMatching(`node-[0-9]+`).Draw(t, "nodeID")
		hlc := NewHLC(regionID, nodeID)

		// Generate random sequence of operations
		numOperations := rapid.IntRange(10, 100).Draw(t, "numOperations")
		var generatedIDs []GlobalID

		for i := 0; i < numOperations; i++ {
			// Randomly choose between drift adjustment and ID generation
			if rapid.Bool().Draw(t, "adjustDrift") {
				// Generate random offset (-1s to +1s)
				offsetMs := rapid.Int64Range(-1000, 1000).Draw(t, "offsetMs")
				offset := time.Duration(offsetMs) * time.Millisecond

				err := hlc.AdjustForDrift(offset)
				if err != nil {
					t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
				}
			}

			// Generate ID
			id := hlc.GenerateID()
			generatedIDs = append(generatedIDs, id)

			// Small delay to allow physical time to advance
			time.Sleep(time.Microsecond)
		}

		// Verify all IDs are monotonically increasing
		for i := 1; i < len(generatedIDs); i++ {
			cmp := CompareGlobalID(generatedIDs[i], generatedIDs[i-1])
			if cmp <= 0 {
				t.Fatalf("Monotonicity violated: ID[%d]=%v should be > ID[%d]=%v (cmp=%d)",
					i, generatedIDs[i], i-1, generatedIDs[i-1], cmp)
			}
		}
	})
}

// Property: Positive drift adjustment should increase logical time
func TestProperty_PositiveDriftIncreasesLogicalTime(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		// Generate initial ID to establish baseline
		hlc.GenerateID()
		initialLogical := hlc.GetLogicalTime()

		// Apply positive drift
		offsetMs := rapid.Int64Range(1, 1000).Draw(t, "offsetMs")
		offset := time.Duration(offsetMs) * time.Millisecond

		err := hlc.AdjustForDrift(offset)
		if err != nil {
			t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
		}

		// Logical time should increase
		newLogical := hlc.GetLogicalTime()
		if newLogical <= initialLogical {
			t.Fatalf("Logical time should increase after positive drift: before=%d, after=%d",
				initialLogical, newLogical)
		}

		// The increase should be proportional to the offset
		increase := newLogical - initialLogical
		expectedIncrease := offsetMs
		if increase < expectedIncrease {
			t.Fatalf("Logical time increase %d should be at least %d",
				increase, expectedIncrease)
		}
	})
}

// Property: Negative drift should not break monotonicity
func TestProperty_NegativeDriftMaintainsMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		// Generate IDs before and after negative drift
		id1 := hlc.GenerateID()

		// Apply negative drift
		offsetMs := rapid.Int64Range(-1000, -1).Draw(t, "offsetMs")
		offset := time.Duration(offsetMs) * time.Millisecond

		err := hlc.AdjustForDrift(offset)
		if err != nil {
			t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
		}

		// Generate ID after drift adjustment
		time.Sleep(time.Millisecond) // Ensure physical time advances
		id2 := hlc.GenerateID()

		// Monotonicity should be maintained
		if CompareGlobalID(id2, id1) <= 0 {
			t.Fatalf("Monotonicity violated after negative drift: id1=%v, id2=%v",
				id1, id2)
		}
	})
}

// Property: Multiple drift adjustments should maintain monotonicity
func TestProperty_MultipleDriftAdjustmentsMaintainMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		numAdjustments := rapid.IntRange(5, 20).Draw(t, "numAdjustments")
		var ids []GlobalID

		for i := 0; i < numAdjustments; i++ {
			// Random drift adjustment
			offsetMs := rapid.Int64Range(-500, 500).Draw(t, "offsetMs")
			offset := time.Duration(offsetMs) * time.Millisecond

			err := hlc.AdjustForDrift(offset)
			if err != nil {
				t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
			}

			// Generate ID
			id := hlc.GenerateID()
			ids = append(ids, id)

			time.Sleep(time.Microsecond)
		}

		// Verify monotonicity
		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i], ids[i-1]) <= 0 {
				t.Fatalf("Monotonicity violated at index %d: ids[%d]=%v, ids[%d]=%v",
					i, i, ids[i], i-1, ids[i-1])
			}
		}
	})
}

// Property: Zero drift should not affect HLC state
func TestProperty_ZeroDriftNoEffect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		// Generate initial ID
		hlc.GenerateID()
		logicalBefore := hlc.GetLogicalTime()

		// Apply zero drift
		err := hlc.AdjustForDrift(0)
		if err != nil {
			t.Fatalf("AdjustForDrift(0) failed: %v", err)
		}

		logicalAfter := hlc.GetLogicalTime()

		// Logical time should not change
		if logicalAfter != logicalBefore {
			t.Fatalf("Zero drift should not change logical time: before=%d, after=%d",
				logicalBefore, logicalAfter)
		}
	})
}

// Property: Drift adjustment should not affect region/node ID
func TestProperty_DriftDoesNotAffectIdentifiers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		regionID := rapid.StringMatching(`region-[a-z]`).Draw(t, "regionID")
		nodeID := rapid.StringMatching(`node-[0-9]+`).Draw(t, "nodeID")
		hlc := NewHLC(regionID, nodeID)

		// Apply random drift adjustments
		numAdjustments := rapid.IntRange(1, 10).Draw(t, "numAdjustments")
		for i := 0; i < numAdjustments; i++ {
			offsetMs := rapid.Int64Range(-1000, 1000).Draw(t, "offsetMs")
			offset := time.Duration(offsetMs) * time.Millisecond
			hlc.AdjustForDrift(offset)
		}

		// Generate ID
		id := hlc.GenerateID()

		// Verify region identifier is unchanged
		if id.RegionID != regionID {
			t.Fatalf("RegionID changed: expected %s, got %s", regionID, id.RegionID)
		}

		// Verify HLC timestamp is valid (non-empty)
		if id.HLC == "" {
			t.Fatal("Generated ID has empty HLC timestamp")
		}
	})
}

// Property: Large positive drift should not cause overflow
func TestProperty_LargeDriftNoOverflow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		// Apply large positive drift (up to 1 hour)
		offsetMs := rapid.Int64Range(1000, 3600000).Draw(t, "offsetMs")
		offset := time.Duration(offsetMs) * time.Millisecond

		err := hlc.AdjustForDrift(offset)
		if err != nil {
			t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
		}

		// Should still be able to generate valid IDs
		id1 := hlc.GenerateID()
		time.Sleep(time.Millisecond)
		id2 := hlc.GenerateID()

		// Verify monotonicity
		if CompareGlobalID(id2, id1) <= 0 {
			t.Fatalf("Monotonicity violated after large drift: id1=%v, id2=%v",
				id1, id2)
		}

		// Verify IDs are valid (have non-empty HLC timestamps)
		if id1.HLC == "" || id2.HLC == "" {
			t.Fatal("Generated IDs have empty HLC timestamps")
		}
	})
}

// Property: Alternating positive and negative drifts maintain monotonicity
func TestProperty_AlternatingDriftsMaintainMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		numCycles := rapid.IntRange(5, 15).Draw(t, "numCycles")
		var ids []GlobalID

		for i := 0; i < numCycles; i++ {
			// Positive drift
			posOffsetMs := rapid.Int64Range(10, 100).Draw(t, "posOffsetMs")
			posOffset := time.Duration(posOffsetMs) * time.Millisecond
			hlc.AdjustForDrift(posOffset)

			id1 := hlc.GenerateID()
			ids = append(ids, id1)
			time.Sleep(time.Microsecond)

			// Negative drift
			negOffsetMs := rapid.Int64Range(-100, -10).Draw(t, "negOffsetMs")
			negOffset := time.Duration(negOffsetMs) * time.Millisecond
			hlc.AdjustForDrift(negOffset)

			id2 := hlc.GenerateID()
			ids = append(ids, id2)
			time.Sleep(time.Microsecond)
		}

		// Verify all IDs are monotonically increasing
		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i], ids[i-1]) <= 0 {
				t.Fatalf("Monotonicity violated at index %d after alternating drifts", i)
			}
		}
	})
}

// Property: Concurrent drift adjustments and ID generation maintain monotonicity
func TestProperty_ConcurrentDriftAndGenerationMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		numGoroutines := rapid.IntRange(2, 5).Draw(t, "numGoroutines")
		opsPerGoroutine := rapid.IntRange(10, 50).Draw(t, "opsPerGoroutine")

		type idWithTimestamp struct {
			id        GlobalID
			timestamp time.Time
		}

		results := make(chan idWithTimestamp, numGoroutines*opsPerGoroutine)
		done := make(chan bool)

		// Start goroutines
		for g := 0; g < numGoroutines; g++ {
			go func(goroutineID int) {
				for i := 0; i < opsPerGoroutine; i++ {
					// Randomly adjust drift or generate ID
					if i%2 == 0 {
						offsetMs := int64(goroutineID*100 + i)
						offset := time.Duration(offsetMs) * time.Microsecond
						hlc.AdjustForDrift(offset)
					}

					// Generate ID
					id := hlc.GenerateID()
					results <- idWithTimestamp{
						id:        id,
						timestamp: time.Now(),
					}

					time.Sleep(time.Microsecond)
				}
				done <- true
			}(g)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
		close(results)

		// Collect all IDs
		var allResults []idWithTimestamp
		for result := range results {
			allResults = append(allResults, result)
		}

		// Sort by generation timestamp
		for i := 0; i < len(allResults)-1; i++ {
			for j := i + 1; j < len(allResults); j++ {
				if allResults[i].timestamp.After(allResults[j].timestamp) {
					allResults[i], allResults[j] = allResults[j], allResults[i]
				}
			}
		}

		// Verify IDs generated later are greater
		// Note: Due to concurrency, we can't guarantee strict ordering,
		// but IDs should generally increase over time
		violations := 0
		for i := 1; i < len(allResults); i++ {
			if CompareGlobalID(allResults[i].id, allResults[i-1].id) <= 0 {
				violations++
			}
		}

		// Allow some violations due to concurrency, but not too many
		maxAllowedViolations := len(allResults) / 10 // 10% tolerance
		if violations > maxAllowedViolations {
			t.Fatalf("Too many monotonicity violations: %d out of %d (max allowed: %d)",
				violations, len(allResults), maxAllowedViolations)
		}
	})
}

// Property: Drift adjustment magnitude should correlate with logical time increase
func TestProperty_DriftMagnitudeCorrelation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-test", "node-test")

		// Generate baseline
		hlc.GenerateID()
		logicalBefore := hlc.GetLogicalTime()

		// Apply positive drift
		offsetMs := rapid.Int64Range(100, 1000).Draw(t, "offsetMs")
		offset := time.Duration(offsetMs) * time.Millisecond

		err := hlc.AdjustForDrift(offset)
		if err != nil {
			t.Fatalf("AdjustForDrift(%v) failed: %v", offset, err)
		}

		logicalAfter := hlc.GetLogicalTime()
		increase := logicalAfter - logicalBefore

		// Increase should be at least proportional to offset
		// (may be larger due to physical time advancement)
		if increase < offsetMs {
			t.Fatalf("Logical time increase %d should be at least %d (offset magnitude)",
				increase, offsetMs)
		}
	})
}
