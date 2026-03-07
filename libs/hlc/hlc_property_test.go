//go:build property

package hlc

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Validates: Requirements 2.1**

// TestProperty_HLCMonotonicity tests Property 1: HLC 单调性保证
// This property ensures that HLC timestamps are always monotonically increasing
// within a single node, regardless of physical clock behavior.
func TestProperty_HLCMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random region and node IDs
		regionID := rapid.StringMatching("[a-z]+-[a-z]").Draw(t, "regionID")
		nodeID := rapid.StringMatching("node-[0-9]+").Draw(t, "nodeID")

		hlc := NewHLC(regionID, nodeID)

		// Generate a sequence of IDs
		numIDs := rapid.IntRange(2, 50).Draw(t, "numIDs")
		var ids []GlobalID

		for i := 0; i < numIDs; i++ {
			id := hlc.GenerateID()
			ids = append(ids, id)
		}

		// Property: All generated IDs must be in strictly monotonic order
		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i-1], ids[i]) >= 0 {
				t.Fatalf("Monotonicity violated: ID[%d] %s >= ID[%d] %s",
					i-1, ids[i-1], i, ids[i])
			}
		}

		// Additional check: Sequence numbers must be strictly increasing
		for i := 1; i < len(ids); i++ {
			if ids[i].Sequence <= ids[i-1].Sequence {
				t.Fatalf("Sequence monotonicity violated: seq[%d] %d <= seq[%d] %d",
					i, ids[i].Sequence, i-1, ids[i-1].Sequence)
			}
		}
	})
}

// TestProperty_CausalOrderingPreservation tests Property 2: 因果关系保序
// This property ensures that causal relationships are preserved across regions
// when HLC timestamps are synchronized.
func TestProperty_CausalOrderingPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create two HLC instances representing different regions
		regionA := "region-a"
		regionB := "region-b"
		hlcA := NewHLC(regionA, "node-1")
		hlcB := NewHLC(regionB, "node-1")

		// Generate a sequence of events with causal relationships
		numEvents := rapid.IntRange(3, 20).Draw(t, "numEvents")
		var events []GlobalID

		for i := 0; i < numEvents; i++ {
			// Randomly choose which region generates the event
			useRegionA := rapid.Bool().Draw(t, fmt.Sprintf("useRegionA_%d", i))

			var event GlobalID
			if useRegionA {
				event = hlcA.GenerateID()
				// Simulate sending this event to region B
				err := hlcB.UpdateFromRemote(event.HLC)
				if err != nil {
					t.Fatalf("Failed to update region B from region A: %v", err)
				}
			} else {
				event = hlcB.GenerateID()
				// Simulate sending this event to region A
				err := hlcA.UpdateFromRemote(event.HLC)
				if err != nil {
					t.Fatalf("Failed to update region A from region B: %v", err)
				}
			}

			events = append(events, event)
		}

		// Property: Events should be sortable by HLC timestamp in a way that
		// preserves causal ordering (events that happened-before should sort earlier)
		sortedEvents := make([]GlobalID, len(events))
		copy(sortedEvents, events)
		sort.Slice(sortedEvents, func(i, j int) bool {
			return CompareGlobalID(sortedEvents[i], sortedEvents[j]) < 0
		})

		// Verify that the sorted order respects causal relationships
		// If event A was used to update the clock before generating event B,
		// then A should sort before B
		for i := 0; i < len(events)-1; i++ {
			currentEvent := events[i]
			nextEvent := events[i+1]

			// Find positions in sorted array
			currentPos := findEventPosition(sortedEvents, currentEvent)
			nextPos := findEventPosition(sortedEvents, nextEvent)

			// If events are from the same region, they must maintain order
			if currentEvent.RegionID == nextEvent.RegionID {
				if currentPos >= nextPos {
					t.Fatalf("Causal ordering violated for same-region events: %s should come before %s",
						currentEvent, nextEvent)
				}
			}
		}

		// Additional property: After synchronization, new events from either region
		// should be greater than all previous events
		finalEventA := hlcA.GenerateID()
		finalEventB := hlcB.GenerateID()

		for _, event := range events {
			if CompareGlobalID(event, finalEventA) >= 0 {
				t.Fatalf("Final event A should be greater than all previous events: %s >= %s",
					event, finalEventA)
			}
			if CompareGlobalID(event, finalEventB) >= 0 {
				t.Fatalf("Final event B should be greater than all previous events: %s >= %s",
					event, finalEventB)
			}
		}
	})
}

// TestProperty_RemoteSynchronizationConsistency tests Property 3: 远程同步一致性
// This property ensures that remote synchronization maintains consistency
// and convergence across different nodes.
func TestProperty_RemoteSynchronizationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create multiple HLC instances representing different nodes
		numNodes := rapid.IntRange(2, 5).Draw(t, "numNodes")
		var hlcs []*HLC

		for i := 0; i < numNodes; i++ {
			regionID := fmt.Sprintf("region-%d", i)
			nodeID := fmt.Sprintf("node-%d", i)
			hlcs = append(hlcs, NewHLC(regionID, nodeID))
		}

		// Generate events and synchronize them across all nodes
		numRounds := rapid.IntRange(3, 10).Draw(t, "numRounds")
		var allEvents []GlobalID

		for round := 0; round < numRounds; round++ {
			// Each node generates an event
			var roundEvents []GlobalID
			for i, hlc := range hlcs {
				event := hlc.GenerateID()
				roundEvents = append(roundEvents, event)
				allEvents = append(allEvents, event)

				// Synchronize this event with all other nodes
				for j, otherHLC := range hlcs {
					if i != j {
						err := otherHLC.UpdateFromRemote(event.HLC)
						if err != nil {
							t.Fatalf("Failed to sync event from node %d to node %d: %v", i, j, err)
						}
					}
				}
			}
		}

		// Property 1: After synchronization, all nodes should generate
		// events that are greater than all previous events
		var finalEvents []GlobalID
		for _, hlc := range hlcs {
			finalEvent := hlc.GenerateID()
			finalEvents = append(finalEvents, finalEvent)

			// This final event should be greater than all previous events
			for _, prevEvent := range allEvents {
				if CompareGlobalID(prevEvent, finalEvent) >= 0 {
					t.Fatalf("Final event should be greater than all previous events: %s >= %s",
						prevEvent, finalEvent)
				}
			}
		}

		// Property 2: All final events should be comparable and sortable
		sort.Slice(finalEvents, func(i, j int) bool {
			return CompareGlobalID(finalEvents[i], finalEvents[j]) < 0
		})

		// Verify the sorted order is consistent
		for i := 1; i < len(finalEvents); i++ {
			if CompareGlobalID(finalEvents[i-1], finalEvents[i]) >= 0 {
				t.Fatalf("Final events not properly sorted: %s >= %s",
					finalEvents[i-1], finalEvents[i])
			}
		}

		// Property 3: Convergence - after full synchronization,
		// the logical relationship between any two events should be deterministic
		for i := 0; i < len(allEvents); i++ {
			for j := i + 1; j < len(allEvents); j++ {
				cmp1 := CompareGlobalID(allEvents[i], allEvents[j])
				cmp2 := CompareGlobalID(allEvents[j], allEvents[i])

				// The comparison should be consistent and opposite
				if cmp1 == 0 && cmp2 != 0 {
					t.Fatalf("Inconsistent comparison: %s vs %s", allEvents[i], allEvents[j])
				}
				if cmp1 > 0 && cmp2 >= 0 {
					t.Fatalf("Inconsistent comparison: %s vs %s", allEvents[i], allEvents[j])
				}
				if cmp1 < 0 && cmp2 <= 0 {
					t.Fatalf("Inconsistent comparison: %s vs %s", allEvents[i], allEvents[j])
				}
			}
		}
	})
}

// TestProperty_ConcurrentSynchronizationSafety tests that concurrent
// synchronization operations maintain consistency
func TestProperty_ConcurrentSynchronizationSafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hlc := NewHLC("region-a", "node-1")

		// Generate random remote timestamps
		numTimestamps := rapid.IntRange(5, 20).Draw(t, "numTimestamps")
		var remoteTimestamps []string

		baseTime := time.Now().UnixMilli()
		for i := 0; i < numTimestamps; i++ {
			// Generate realistic timestamps around current time
			physicalTime := baseTime + rapid.Int64Range(-1000, 1000).Draw(t, fmt.Sprintf("physical_%d", i))
			logicalTime := rapid.Int64Range(0, 100).Draw(t, fmt.Sprintf("logical_%d", i))
			timestamp := fmt.Sprintf("%d-%d", physicalTime, logicalTime)
			remoteTimestamps = append(remoteTimestamps, timestamp)
		}

		// Apply remote updates concurrently
		var wg sync.WaitGroup
		errorChan := make(chan error, numTimestamps)

		for i, timestamp := range remoteTimestamps {
			wg.Add(1)
			go func(ts string, idx int) {
				defer wg.Done()

				// Add some small delay to increase chance of race conditions
				time.Sleep(time.Duration(idx%5) * time.Millisecond)

				err := hlc.UpdateFromRemote(ts)
				if err != nil {
					errorChan <- fmt.Errorf("update %d failed: %w", idx, err)
				}
			}(timestamp, i)
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		for err := range errorChan {
			t.Fatalf("Concurrent synchronization error: %v", err)
		}

		// Property: After all concurrent updates, the HLC should still
		// generate monotonic IDs
		var ids []GlobalID
		for i := 0; i < 10; i++ {
			id := hlc.GenerateID()
			ids = append(ids, id)
		}

		for i := 1; i < len(ids); i++ {
			if CompareGlobalID(ids[i-1], ids[i]) >= 0 {
				t.Fatalf("Monotonicity lost after concurrent sync: %s >= %s",
					ids[i-1], ids[i])
			}
		}
	})
}

// TestProperty_HLCTimestampParsing tests that HLC timestamp parsing
// is consistent and handles edge cases properly
func TestProperty_HLCTimestampParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid HLC timestamp components
		physicalTime := rapid.Int64Range(0, 9999999999999).Draw(t, "physicalTime")
		logicalTime := rapid.Int64Range(0, 999999).Draw(t, "logicalTime")

		// Create HLC timestamp string
		hlcStr := fmt.Sprintf("%d-%d", physicalTime, logicalTime)

		// Property: Parsing should be consistent and reversible
		parsed, err := parseHLC(hlcStr)
		if err != nil {
			t.Fatalf("Failed to parse valid HLC timestamp %s: %v", hlcStr, err)
		}

		if parsed.Physical != physicalTime {
			t.Fatalf("Physical time mismatch: expected %d, got %d", physicalTime, parsed.Physical)
		}

		if parsed.Logical != logicalTime {
			t.Fatalf("Logical time mismatch: expected %d, got %d", logicalTime, parsed.Logical)
		}

		// Property: Reconstructed string should match original
		reconstructed := fmt.Sprintf("%d-%d", parsed.Physical, parsed.Logical)
		if reconstructed != hlcStr {
			t.Fatalf("Timestamp reconstruction failed: expected %s, got %s", hlcStr, reconstructed)
		}
	})
}

// TestProperty_GlobalIDComparison tests that GlobalID comparison
// is transitive, consistent, and deterministic
func TestProperty_GlobalIDComparison(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate three random GlobalIDs
		ids := make([]GlobalID, 3)
		for i := 0; i < 3; i++ {
			ids[i] = GlobalID{
				RegionID: rapid.SampledFrom([]string{"region-a", "region-b", "region-c"}).Draw(t, fmt.Sprintf("region_%d", i)),
				HLC:      generateRandomHLC(t),
				Sequence: rapid.Int64Range(1, 1000000).Draw(t, fmt.Sprintf("sequence_%d", i)),
			}
		}

		// Property 1: Comparison is reflexive (a == a)
		for i, id := range ids {
			if CompareGlobalID(id, id) != 0 {
				t.Fatalf("Reflexivity violated for ID[%d]: %s", i, id)
			}
		}

		// Property 2: Comparison is antisymmetric (if a < b, then b > a)
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				cmp1 := CompareGlobalID(ids[i], ids[j])
				cmp2 := CompareGlobalID(ids[j], ids[i])

				if cmp1 > 0 && cmp2 >= 0 {
					t.Fatalf("Antisymmetry violated: %s vs %s (cmp1=%d, cmp2=%d)",
						ids[i], ids[j], cmp1, cmp2)
				}
				if cmp1 < 0 && cmp2 <= 0 {
					t.Fatalf("Antisymmetry violated: %s vs %s (cmp1=%d, cmp2=%d)",
						ids[i], ids[j], cmp1, cmp2)
				}
				if cmp1 == 0 && cmp2 != 0 {
					t.Fatalf("Antisymmetry violated: %s vs %s (cmp1=%d, cmp2=%d)",
						ids[i], ids[j], cmp1, cmp2)
				}
			}
		}

		// Property 3: Comparison is transitive (if a < b and b < c, then a < c)
		cmp01 := CompareGlobalID(ids[0], ids[1])
		cmp12 := CompareGlobalID(ids[1], ids[2])
		cmp02 := CompareGlobalID(ids[0], ids[2])

		if cmp01 < 0 && cmp12 < 0 && cmp02 >= 0 {
			t.Fatalf("Transitivity violated: %s < %s < %s but %s >= %s",
				ids[0], ids[1], ids[2], ids[0], ids[2])
		}
		if cmp01 > 0 && cmp12 > 0 && cmp02 <= 0 {
			t.Fatalf("Transitivity violated: %s > %s > %s but %s <= %s",
				ids[0], ids[1], ids[2], ids[0], ids[2])
		}
	})
}

// Helper function to find the position of an event in a sorted slice
func findEventPosition(sortedEvents []GlobalID, event GlobalID) int {
	for i, e := range sortedEvents {
		if e.RegionID == event.RegionID && e.HLC == event.HLC && e.Sequence == event.Sequence {
			return i
		}
	}
	return -1
}

// Helper function to generate a random HLC timestamp string
func generateRandomHLC(t *rapid.T) string {
	physicalTime := rapid.Int64Range(1000000000000, 9999999999999).Draw(t, "physical")
	logicalTime := rapid.Int64Range(0, 1000).Draw(t, "logical")
	return fmt.Sprintf("%d-%d", physicalTime, logicalTime)
}
