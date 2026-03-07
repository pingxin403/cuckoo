package hlc

import (
	"sync"
	"testing"
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
		_ = hlc.UpdateFromRemote(remoteHLC)
	}
}
