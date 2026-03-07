package hlc

import (
	"testing"
	"time"
)

func TestHLC_AdjustForDrift_PositiveOffset(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate initial ID
	id1 := hlc.GenerateID()

	// Simulate positive drift (local clock ahead by 100ms)
	offset := 100 * time.Millisecond
	err := hlc.AdjustForDrift(offset)
	if err != nil {
		t.Fatalf("AdjustForDrift() error = %v", err)
	}

	// Generate ID after calibration
	id2 := hlc.GenerateID()

	// Verify monotonicity is maintained
	if CompareGlobalID(id2, id1) <= 0 {
		t.Errorf("ID after calibration should be greater than before: id1=%v, id2=%v", id1, id2)
	}

	// Verify logical time increased
	if hlc.GetLogicalTime() <= 0 {
		t.Errorf("Logical time should increase after positive drift adjustment, got %d", hlc.GetLogicalTime())
	}
}

func TestHLC_AdjustForDrift_NegativeOffset(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate initial ID
	id1 := hlc.GenerateID()
	initialLogical := hlc.GetLogicalTime()

	// Simulate negative drift (local clock behind by 100ms)
	offset := -100 * time.Millisecond
	err := hlc.AdjustForDrift(offset)
	if err != nil {
		t.Fatalf("AdjustForDrift() error = %v", err)
	}

	// For negative offset, no adjustment is made
	// The HLC algorithm will naturally handle this on next GenerateID
	if hlc.GetLogicalTime() != initialLogical {
		t.Errorf("Logical time should not change for negative offset, got %d, want %d",
			hlc.GetLogicalTime(), initialLogical)
	}

	// Generate ID after calibration
	id2 := hlc.GenerateID()

	// Verify monotonicity is still maintained
	if CompareGlobalID(id2, id1) <= 0 {
		t.Errorf("ID after calibration should be greater than before: id1=%v, id2=%v", id1, id2)
	}
}

func TestHLC_AdjustForDrift_ZeroOffset(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	initialLogical := hlc.GetLogicalTime()

	// Zero offset should not change anything
	err := hlc.AdjustForDrift(0)
	if err != nil {
		t.Fatalf("AdjustForDrift() error = %v", err)
	}

	if hlc.GetLogicalTime() != initialLogical {
		t.Errorf("Logical time should not change for zero offset, got %d, want %d",
			hlc.GetLogicalTime(), initialLogical)
	}
}

func TestHLC_AdjustForDrift_Monotonicity(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate multiple IDs with drift adjustments
	var ids []GlobalID
	offsets := []time.Duration{
		50 * time.Millisecond,
		-30 * time.Millisecond,
		100 * time.Millisecond,
		0,
		-50 * time.Millisecond,
		200 * time.Millisecond,
	}

	for _, offset := range offsets {
		// Adjust for drift
		if err := hlc.AdjustForDrift(offset); err != nil {
			t.Fatalf("AdjustForDrift(%v) error = %v", offset, err)
		}

		// Generate ID
		id := hlc.GenerateID()
		ids = append(ids, id)

		// Small delay to ensure physical time advances
		time.Sleep(time.Millisecond)
	}

	// Verify all IDs are monotonically increasing
	for i := 1; i < len(ids); i++ {
		if CompareGlobalID(ids[i], ids[i-1]) <= 0 {
			t.Errorf("IDs not monotonic: ids[%d]=%v should be > ids[%d]=%v",
				i, ids[i], i-1, ids[i-1])
		}
	}
}

func TestHLC_AdjustForDrift_LargeOffset(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate initial ID
	id1 := hlc.GenerateID()

	// Simulate large positive drift (5 seconds)
	offset := 5 * time.Second
	err := hlc.AdjustForDrift(offset)
	if err != nil {
		t.Fatalf("AdjustForDrift() error = %v", err)
	}

	// Logical time should increase significantly
	logicalTime := hlc.GetLogicalTime()
	expectedIncrease := offset.Milliseconds()
	if logicalTime < expectedIncrease {
		t.Errorf("Logical time = %d, should be at least %d after large offset adjustment",
			logicalTime, expectedIncrease)
	}

	// Generate ID after calibration
	id2 := hlc.GenerateID()

	// Verify monotonicity
	if CompareGlobalID(id2, id1) <= 0 {
		t.Errorf("ID after large drift calibration should be greater: id1=%v, id2=%v", id1, id2)
	}
}

func TestHLC_AdjustForDrift_ConcurrentAccess(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")
	done := make(chan bool)

	// Concurrent drift adjustments
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				offset := time.Duration(id*10+j) * time.Millisecond
				hlc.AdjustForDrift(offset)
			}
			done <- true
		}(i)
	}

	// Concurrent ID generation
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				hlc.GenerateID()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// HLC should still be in valid state
	id := hlc.GenerateID()
	if id.RegionID != "region-a" {
		t.Errorf("RegionID = %s, want region-a", id.RegionID)
	}
}
