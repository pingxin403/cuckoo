package hlc

import (
	"fmt"
	"time"
)

// Example demonstrates basic HLC usage for multi-region message ordering
func ExampleHLC_GenerateID() {
	// Create HLC instances for two regions
	regionA := NewHLC("region-a", "node-1")
	regionB := NewHLC("region-b", "node-1")

	// Generate IDs in region A
	idA1 := regionA.GenerateID()
	idA2 := regionA.GenerateID()

	// Generate ID in region B
	idB1 := regionB.GenerateID()

	// Simulate region B receiving a message from region A
	// This updates region B's clock to maintain causal ordering
	regionB.UpdateFromRemote(idA2.HLC)

	// Generate another ID in region B after sync
	idB2 := regionB.GenerateID()

	fmt.Printf("Region A IDs: %s, %s\n", idA1, idA2)
	fmt.Printf("Region B IDs: %s, %s\n", idB1, idB2)

	// Demonstrate global ordering
	ids := []GlobalID{idA1, idA2, idB1, idB2}
	fmt.Println("\nGlobal ordering:")
	for i := 0; i < len(ids)-1; i++ {
		for j := i + 1; j < len(ids); j++ {
			cmp := CompareGlobalID(ids[i], ids[j])
			if cmp < 0 {
				fmt.Printf("%s < %s\n", ids[i], ids[j])
			} else if cmp > 0 {
				fmt.Printf("%s > %s\n", ids[i], ids[j])
			} else {
				fmt.Printf("%s = %s\n", ids[i], ids[j])
			}
		}
	}
}

// Example demonstrates conflict resolution using RegionID tiebreaker
func ExampleCompareGlobalID() {
	// Two messages with identical HLC timestamps from different regions
	msgA := GlobalID{
		RegionID: "region-a",
		HLC:      "1234567890123-5",
		Sequence: 1,
	}

	msgB := GlobalID{
		RegionID: "region-b",
		HLC:      "1234567890123-5", // Same HLC timestamp
		Sequence: 1,
	}

	// RegionID acts as tiebreaker for deterministic ordering
	result := CompareGlobalID(msgA, msgB)
	if result < 0 {
		fmt.Printf("%s comes before %s (region-a < region-b)\n", msgA, msgB)
	}

	// Output: region-a-1234567890123-5-1 comes before region-b-1234567890123-5-1 (region-a < region-b)
}

// Example demonstrates HLC clock synchronization
func ExampleHLC_UpdateFromRemote() {
	hlc := NewHLC("region-a", "node-1")

	// Initial state
	fmt.Printf("Initial: %s\n", hlc.GetCurrentTimestamp())

	// Simulate receiving a message from future
	futureHLC := fmt.Sprintf("%d-0", time.Now().UnixMilli()+1000)
	hlc.UpdateFromRemote(futureHLC)

	// Clock jumps forward to maintain causal ordering
	fmt.Printf("After remote sync: %s\n", hlc.GetCurrentTimestamp())

	// Next generated ID will be after the remote timestamp
	id := hlc.GenerateID()
	fmt.Printf("Next ID: %s\n", id)
}
