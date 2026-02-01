package hlc

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewHLC(t *testing.T) {
	regionID := "region-a"
	nodeID := "node-1"

	hlc := NewHLC(regionID, nodeID)

	if hlc.regionID != regionID {
		t.Errorf("Expected regionID %s, got %s", regionID, hlc.regionID)
	}

	if hlc.nodeID != nodeID {
		t.Errorf("Expected nodeID %s, got %s", nodeID, hlc.nodeID)
	}

	if hlc.logicalTime != 0 {
		t.Errorf("Expected initial logical time 0, got %d", hlc.logicalTime)
	}

	if hlc.sequence != 0 {
		t.Errorf("Expected initial sequence 0, got %d", hlc.sequence)
	}
}

func TestGenerateID_Monotonicity(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate multiple IDs and verify they are monotonically increasing
	var ids []GlobalID
	for i := 0; i < 100; i++ {
		id := hlc.GenerateID()
		ids = append(ids, id)

		// Verify region ID is correct
		if id.RegionID != "region-a" {
			t.Errorf("Expected region ID 'region-a', got %s", id.RegionID)
		}

		// Verify sequence is monotonic
		if id.Sequence != int64(i+1) {
			t.Errorf("Expected sequence %d, got %d", i+1, id.Sequence)
		}
	}

	// Verify all IDs are in order
	for i := 1; i < len(ids); i++ {
		if CompareGlobalID(ids[i-1], ids[i]) >= 0 {
			t.Errorf("IDs not in monotonic order: %s >= %s", ids[i-1], ids[i])
		}
	}
}

func TestGenerateID_ConcurrentSafety(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	const numGoroutines = 10
	const idsPerGoroutine = 100

	var wg sync.WaitGroup
	idChan := make(chan GlobalID, numGoroutines*idsPerGoroutine)

	// Generate IDs concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id := hlc.GenerateID()
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// Collect all IDs
	var ids []GlobalID
	for id := range idChan {
		ids = append(ids, id)
	}

	// Verify we got the expected number of IDs
	if len(ids) != numGoroutines*idsPerGoroutine {
		t.Errorf("Expected %d IDs, got %d", numGoroutines*idsPerGoroutine, len(ids))
	}

	// Verify all sequence numbers are unique
	seqMap := make(map[int64]bool)
	for _, id := range ids {
		if seqMap[id.Sequence] {
			t.Errorf("Duplicate sequence number: %d", id.Sequence)
		}
		seqMap[id.Sequence] = true
	}
}

func TestUpdateFromRemote_BasicSync(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate a local ID first
	localID := hlc.GenerateID()

	// Simulate receiving a remote timestamp from the future
	remoteHLC := "1234567890123-5" // Some future timestamp

	err := hlc.UpdateFromRemote(remoteHLC)
	if err != nil {
		t.Fatalf("UpdateFromRemote failed: %v", err)
	}

	// Generate another ID and verify it's after the remote timestamp
	newID := hlc.GenerateID()

	// The new ID should be greater than both local and remote
	remoteID := GlobalID{
		RegionID: "region-b",
		HLC:      remoteHLC,
		Sequence: 1,
	}

	if CompareGlobalID(newID, localID) <= 0 {
		t.Errorf("New ID should be greater than local ID")
	}

	if CompareGlobalID(newID, remoteID) <= 0 {
		t.Errorf("New ID should be greater than remote ID")
	}
}

func TestUpdateFromRemote_ClockSkew(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Simulate clock skew - remote clock is behind
	remoteHLC := "1234567890000-10" // Past timestamp with high logical counter

	err := hlc.UpdateFromRemote(remoteHLC)
	if err != nil {
		t.Fatalf("UpdateFromRemote failed: %v", err)
	}

	// Generate new ID - should still be monotonic
	id1 := hlc.GenerateID()
	id2 := hlc.GenerateID()

	if CompareGlobalID(id1, id2) >= 0 {
		t.Errorf("IDs should be monotonic even with clock skew")
	}
}

func TestUpdateFromRemote_InvalidFormat(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	testCases := []string{
		"invalid",
		"123",
		"123-",
		"-456",
		"abc-def",
		"123-456-789",
	}

	for _, tc := range testCases {
		err := hlc.UpdateFromRemote(tc)
		if err == nil {
			t.Errorf("Expected error for invalid HLC format: %s", tc)
		}
	}
}

func TestCompareGlobalID_HLCOrdering(t *testing.T) {
	testCases := []struct {
		name     string
		id1      GlobalID
		id2      GlobalID
		expected int
	}{
		{
			name:     "Different physical time",
			id1:      GlobalID{RegionID: "region-a", HLC: "1000-0", Sequence: 1},
			id2:      GlobalID{RegionID: "region-a", HLC: "2000-0", Sequence: 1},
			expected: -1,
		},
		{
			name:     "Same physical, different logical",
			id1:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			id2:      GlobalID{RegionID: "region-a", HLC: "1000-10", Sequence: 1},
			expected: -1,
		},
		{
			name:     "Same HLC, different region (tiebreaker)",
			id1:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			id2:      GlobalID{RegionID: "region-b", HLC: "1000-5", Sequence: 1},
			expected: -1,
		},
		{
			name:     "Same HLC and region, different sequence",
			id1:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			id2:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 2},
			expected: -1,
		},
		{
			name:     "Identical IDs",
			id1:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			id2:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			expected: 0,
		},
		{
			name:     "Reverse order",
			id1:      GlobalID{RegionID: "region-b", HLC: "2000-10", Sequence: 5},
			id2:      GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1},
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CompareGlobalID(tc.id1, tc.id2)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d for comparison of %s vs %s",
					tc.expected, result, tc.id1, tc.id2)
			}
		})
	}
}

func TestCompareGlobalID_InvalidHLC(t *testing.T) {
	// Test with invalid HLC format - should fall back to string comparison
	id1 := GlobalID{RegionID: "region-a", HLC: "invalid-format", Sequence: 1}
	id2 := GlobalID{RegionID: "region-a", HLC: "1000-5", Sequence: 1}

	result := CompareGlobalID(id1, id2)
	// Should not panic and should return some consistent result
	if result == 0 {
		t.Errorf("Expected non-zero result for invalid HLC comparison")
	}
}

func TestParseHLC(t *testing.T) {
	testCases := []struct {
		input    string
		expected HLCTimestamp
		hasError bool
	}{
		{
			input:    "1234567890123-456",
			expected: HLCTimestamp{Physical: 1234567890123, Logical: 456},
			hasError: false,
		},
		{
			input:    "0-0",
			expected: HLCTimestamp{Physical: 0, Logical: 0},
			hasError: false,
		},
		{
			input:    "invalid",
			hasError: true,
		},
		{
			input:    "123-456-789",
			hasError: true,
		},
		{
			input:    "abc-def",
			hasError: true,
		},
		{
			input:    "123-",
			hasError: true,
		},
		{
			input:    "-456",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseHLC(tc.input)

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				}
				if result.Physical != tc.expected.Physical {
					t.Errorf("Expected physical %d, got %d", tc.expected.Physical, result.Physical)
				}
				if result.Logical != tc.expected.Logical {
					t.Errorf("Expected logical %d, got %d", tc.expected.Logical, result.Logical)
				}
			}
		})
	}
}

func TestHLC_String(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")
	hlc.GenerateID() // Generate one ID to increment sequence

	str := hlc.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}

	// Should contain region and node info
	if !contains(str, "region-a") {
		t.Error("String() should contain region ID")
	}
	if !contains(str, "node-1") {
		t.Error("String() should contain node ID")
	}
}

func TestGlobalID_String(t *testing.T) {
	id := GlobalID{
		RegionID: "region-a",
		HLC:      "1234567890123-456",
		Sequence: 789,
	}

	str := id.String()
	expected := "region-a-1234567890123-456-789"

	if str != expected {
		t.Errorf("Expected %s, got %s", expected, str)
	}
}

func TestGetCurrentTimestamp(t *testing.T) {
	hlc := NewHLC("region-a", "node-1")

	// Generate an ID to advance the clock
	hlc.GenerateID()

	timestamp := hlc.GetCurrentTimestamp()

	// Should be in format "physical-logical"
	_, err := parseHLC(timestamp)
	if err != nil {
		t.Errorf("GetCurrentTimestamp() returned invalid format: %s", timestamp)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkGenerateID(b *testing.B) {
	hlc := NewHLC("region-a", "node-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hlc.GenerateID()
	}
}

func BenchmarkCompareGlobalID(b *testing.B) {
	id1 := GlobalID{RegionID: "region-a", HLC: "1234567890123-456", Sequence: 1}
	id2 := GlobalID{RegionID: "region-b", HLC: "1234567890124-457", Sequence: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CompareGlobalID(id1, id2)
	}
}

func BenchmarkUpdateFromRemote(b *testing.B) {
	hlc := NewHLC("region-a", "node-1")
	remoteHLC := "1234567890123-456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hlc.UpdateFromRemote(remoteHLC)
	}
}

// Additional comprehensive tests for task 1.2 requirements

// TestClockRollbackHandling tests how HLC handles various clock rollback scenarios
func TestClockRollbackHandling(t *testing.T) {
	t.Run("LogicalTimeIncrement", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Generate multiple IDs in quick succession to test logical time increment
		// When physical time doesn't advance, logical time should increment
		var ids []GlobalID
		for i := 0; i < 5; i++ {
			id := hlc.GenerateID()
			ids = append(ids, id)
		}

		// Verify all IDs are monotonic
		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i-1], ids[i]) >= 0 {
				t.Errorf("IDs not monotonic at position %d: %s >= %s",
					i, ids[i-1], ids[i])
			}
		}

		// Check that logical time increments when physical time is the same
		hlc1, _ := parseHLC(ids[0].HLC)
		hlc2, _ := parseHLC(ids[1].HLC)

		if hlc1.Physical == hlc2.Physical {
			if hlc2.Logical <= hlc1.Logical {
				t.Errorf("Logical time should increment when physical time is same: %d <= %d",
					hlc2.Logical, hlc1.Logical)
			}
		}
	})

	t.Run("RemoteClockFromPast", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Generate some local IDs first
		for i := 0; i < 5; i++ {
			hlc.GenerateID()
		}

		// Simulate receiving a timestamp from far in the past
		pastTimestamp := "1000000000000-100" // Very old timestamp

		err := hlc.UpdateFromRemote(pastTimestamp)
		if err != nil {
			t.Fatalf("UpdateFromRemote should handle past timestamps: %v", err)
		}

		// Generate new ID - should still be monotonic
		newID := hlc.GenerateID()

		// The new ID should be greater than the past timestamp
		pastID := GlobalID{RegionID: "region-b", HLC: pastTimestamp, Sequence: 1}
		if CompareGlobalID(newID, pastID) <= 0 {
			t.Errorf("New ID should be greater than past remote timestamp")
		}
	})

	t.Run("ClockSkewResilience", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Test that HLC maintains monotonicity even with various remote timestamps
		// that might represent clock skew scenarios
		remoteTimestamps := []string{
			"1703123456789-5",  // Normal timestamp
			"1703123456788-10", // Slightly behind in physical time, ahead in logical
			"1703123456790-2",  // Ahead in physical time, behind in logical
			"1703123456789-15", // Same physical time, high logical time
		}

		var allIDs []GlobalID

		for _, remoteTS := range remoteTimestamps {
			// Update from remote
			err := hlc.UpdateFromRemote(remoteTS)
			if err != nil {
				t.Fatalf("UpdateFromRemote failed for %s: %v", remoteTS, err)
			}

			// Generate local ID after each remote update
			localID := hlc.GenerateID()
			allIDs = append(allIDs, localID)
		}

		// Verify all local IDs are monotonic
		for i := 1; i < len(allIDs); i++ {
			if CompareGlobalID(allIDs[i-1], allIDs[i]) >= 0 {
				t.Errorf("Local IDs not monotonic after remote updates at position %d: %s >= %s",
					i, allIDs[i-1], allIDs[i])
			}
		}
	})
}

// TestRemoteClockSyncLogic tests comprehensive remote clock synchronization scenarios
func TestRemoteClockSyncLogic(t *testing.T) {
	t.Run("RemoteClockAhead", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Generate baseline
		localID := hlc.GenerateID()

		// Remote clock is ahead
		futureTimestamp := "9999999999999-5"
		err := hlc.UpdateFromRemote(futureTimestamp)
		if err != nil {
			t.Fatalf("UpdateFromRemote failed: %v", err)
		}

		// Next local ID should be after the remote timestamp
		nextID := hlc.GenerateID()
		remoteID := GlobalID{RegionID: "region-b", HLC: futureTimestamp, Sequence: 1}

		if CompareGlobalID(nextID, remoteID) <= 0 {
			t.Errorf("Local ID should be after remote timestamp")
		}

		// Should also be after the original local ID
		if CompareGlobalID(nextID, localID) <= 0 {
			t.Errorf("New ID should be after original local ID")
		}
	})

	t.Run("RemoteClockSamePhysicalTime", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Get current timestamp
		currentTimestamp := hlc.GetCurrentTimestamp()
		currentHLC, _ := parseHLC(currentTimestamp)

		// Create remote timestamp with same physical time but different logical time
		remoteTimestamp := fmt.Sprintf("%d-%d", currentHLC.Physical, currentHLC.Logical+10)

		err := hlc.UpdateFromRemote(remoteTimestamp)
		if err != nil {
			t.Fatalf("UpdateFromRemote failed: %v", err)
		}

		// Next ID should have logical time greater than remote
		nextID := hlc.GenerateID()
		nextHLC, _ := parseHLC(nextID.HLC)

		if nextHLC.Physical == currentHLC.Physical && nextHLC.Logical <= currentHLC.Logical+10 {
			t.Errorf("Logical time should be greater than remote logical time")
		}
	})

	t.Run("MultipleRemoteUpdates", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		// Simulate receiving updates from multiple remote regions
		remoteTimestamps := []string{
			"1703123456789-5",
			"1703123456790-3",
			"1703123456788-10",
			"1703123456791-1",
		}

		var localIDs []GlobalID

		for i, remoteTS := range remoteTimestamps {
			err := hlc.UpdateFromRemote(remoteTS)
			if err != nil {
				t.Fatalf("UpdateFromRemote failed for %s: %v", remoteTS, err)
			}

			// Generate local ID after each remote update
			localID := hlc.GenerateID()
			localIDs = append(localIDs, localID)

			// Verify local ID is greater than the remote timestamp
			remoteID := GlobalID{
				RegionID: fmt.Sprintf("region-%d", i),
				HLC:      remoteTS,
				Sequence: 1,
			}

			if CompareGlobalID(localID, remoteID) <= 0 {
				t.Errorf("Local ID should be greater than remote timestamp %s", remoteTS)
			}
		}

		// Verify all local IDs are monotonic
		for i := 1; i < len(localIDs); i++ {
			if CompareGlobalID(localIDs[i-1], localIDs[i]) >= 0 {
				t.Errorf("Local IDs not monotonic after remote updates")
			}
		}
	})

	t.Run("ConcurrentRemoteUpdates", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		const numGoroutines = 5
		const updatesPerGoroutine = 20

		var wg sync.WaitGroup
		errorChan := make(chan error, numGoroutines*updatesPerGoroutine)

		// Concurrent remote updates
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < updatesPerGoroutine; j++ {
					// Create unique remote timestamp
					remoteTS := fmt.Sprintf("%d-%d",
						1703123456789+int64(goroutineID*1000+j),
						j%10)

					err := hlc.UpdateFromRemote(remoteTS)
					if err != nil {
						errorChan <- err
						return
					}

					// Generate ID after update
					hlc.GenerateID()
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		for err := range errorChan {
			t.Errorf("Concurrent remote update failed: %v", err)
		}
	})
}

// TestConcurrentSafetyStress provides more comprehensive concurrent safety testing
func TestConcurrentSafetyStress(t *testing.T) {
	t.Run("HighConcurrencyIDGeneration", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		const numGoroutines = 50
		const idsPerGoroutine = 200

		var wg sync.WaitGroup
		idChan := make(chan GlobalID, numGoroutines*idsPerGoroutine)

		// High concurrency ID generation
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id := hlc.GenerateID()
					idChan <- id
				}
			}()
		}

		wg.Wait()
		close(idChan)

		// Collect and verify IDs
		var ids []GlobalID
		seqMap := make(map[int64]bool)

		for id := range idChan {
			ids = append(ids, id)

			// Check for duplicate sequences
			if seqMap[id.Sequence] {
				t.Errorf("Duplicate sequence number: %d", id.Sequence)
			}
			seqMap[id.Sequence] = true
		}

		if len(ids) != numGoroutines*idsPerGoroutine {
			t.Errorf("Expected %d IDs, got %d", numGoroutines*idsPerGoroutine, len(ids))
		}
	})

	t.Run("MixedOperationsConcurrency", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		const duration = 2 * time.Second
		const numGenerators = 10
		const numUpdaters = 5

		var wg sync.WaitGroup
		done := make(chan struct{})

		// Start ID generators
		for i := 0; i < numGenerators; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-done:
						return
					default:
						hlc.GenerateID()
					}
				}
			}()
		}

		// Start remote updaters
		for i := 0; i < numUpdaters; i++ {
			wg.Add(1)
			go func(updaterID int) {
				defer wg.Done()
				counter := 0
				for {
					select {
					case <-done:
						return
					default:
						remoteTS := fmt.Sprintf("%d-%d",
							time.Now().UnixMilli()+int64(updaterID*1000),
							counter%100)
						hlc.UpdateFromRemote(remoteTS)
						counter++
						time.Sleep(10 * time.Millisecond)
					}
				}
			}(i)
		}

		// Run for specified duration
		time.Sleep(duration)
		close(done)
		wg.Wait()

		// Verify HLC is still in a consistent state
		id1 := hlc.GenerateID()
		id2 := hlc.GenerateID()

		if CompareGlobalID(id1, id2) >= 0 {
			t.Errorf("HLC lost monotonicity after stress test")
		}
	})

	t.Run("RaceConditionDetection", func(t *testing.T) {
		// This test is designed to catch race conditions with -race flag
		hlc := NewHLC("region-a", "node-1")

		const numOperations = 1000
		var wg sync.WaitGroup

		// Mix of read and write operations
		for i := 0; i < numOperations; i++ {
			wg.Add(3)

			// ID generation
			go func() {
				defer wg.Done()
				hlc.GenerateID()
			}()

			// Remote update
			go func(i int) {
				defer wg.Done()
				remoteTS := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), i%50)
				hlc.UpdateFromRemote(remoteTS)
			}(i)

			// Timestamp reading
			go func() {
				defer wg.Done()
				hlc.GetCurrentTimestamp()
			}()
		}

		wg.Wait()
	})
}

// TestClockMonotonicity provides additional monotonicity guarantees testing
func TestClockMonotonicity(t *testing.T) {
	t.Run("MonotonicityWithFastGeneration", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		var ids []GlobalID

		// Generate IDs rapidly to test logical time increment behavior
		// When physical time doesn't advance, logical time should increment
		for i := 0; i < 20; i++ {
			id := hlc.GenerateID()
			ids = append(ids, id)
		}

		// Verify strict monotonicity
		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i-1], ids[i]) >= 0 {
				t.Errorf("Monotonicity violated at position %d: %s >= %s",
					i, ids[i-1], ids[i])
			}
		}

		// Verify that when physical time is the same, logical time increments
		for i := 1; i < len(ids); i++ {
			hlc1, _ := parseHLC(ids[i-1].HLC)
			hlc2, _ := parseHLC(ids[i].HLC)

			if hlc1.Physical == hlc2.Physical {
				if hlc2.Logical <= hlc1.Logical {
					t.Errorf("Logical time should increment when physical time is same at position %d: %d <= %d",
						i, hlc2.Logical, hlc1.Logical)
				}
			}
		}
	})

	t.Run("MonotonicityAcrossRemoteSync", func(t *testing.T) {
		hlc := NewHLC("region-a", "node-1")

		var allIDs []GlobalID

		// Generate some local IDs
		for i := 0; i < 5; i++ {
			id := hlc.GenerateID()
			allIDs = append(allIDs, id)
		}

		// Sync with remote timestamps
		remoteTimestamps := []string{
			"1703123456800-10",
			"1703123456750-5",
			"1703123456900-15",
		}

		for _, remoteTS := range remoteTimestamps {
			hlc.UpdateFromRemote(remoteTS)

			// Generate local ID after sync
			id := hlc.GenerateID()
			allIDs = append(allIDs, id)
		}

		// Generate more local IDs
		for i := 0; i < 5; i++ {
			id := hlc.GenerateID()
			allIDs = append(allIDs, id)
		}

		// Verify monotonicity is maintained throughout
		for i := 1; i < len(allIDs); i++ {
			if CompareGlobalID(allIDs[i-1], allIDs[i]) >= 0 {
				t.Errorf("Monotonicity violated after remote sync at position %d: %s >= %s",
					i, allIDs[i-1], allIDs[i])
			}
		}
	})
}
