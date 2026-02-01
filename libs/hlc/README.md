# HLC (Hybrid Logical Clock) Library

A Go implementation of Hybrid Logical Clock for multi-region active-active architecture, providing global ordering and conflict resolution capabilities.

## Overview

This library implements the Hybrid Logical Clock algorithm as described in the paper "Logical Physical Clocks and Consistent Snapshots in Globally Distributed Databases" by Kulkarni et al. It combines physical time with logical counters to provide:

- **Global Ordering**: Messages can be ordered across regions despite clock skew
- **Causal Consistency**: Maintains happens-before relationships
- **Conflict Resolution**: Deterministic ordering using RegionID tiebreaker
- **Clock Synchronization**: Handles remote clock updates for distributed systems

## Key Features

- ✅ **Thread-Safe**: Concurrent ID generation with proper synchronization
- ✅ **High Performance**: ~139ns per ID generation, ~116ns per comparison
- ✅ **Clock Skew Resilient**: Handles backward clock jumps gracefully
- ✅ **Deterministic Ordering**: RegionID tiebreaker ensures consistent results
- ✅ **Comprehensive Testing**: 13 unit tests + benchmarks + examples

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/cuckoo-org/cuckoo/libs/hlc"
)

func main() {
    // Create HLC for region-a
    clock := hlc.NewHLC("region-a", "node-1")
    
    // Generate global IDs
    id1 := clock.GenerateID()
    id2 := clock.GenerateID()
    
    fmt.Printf("Generated IDs: %s, %s\n", id1, id2)
    // Output: Generated IDs: region-a-1703123456789-0-1, region-a-1703123456789-1-2
}
```

### Multi-Region Synchronization

```go
// Region A generates a message
regionA := hlc.NewHLC("region-a", "node-1")
msgID := regionA.GenerateID()

// Region B receives the message and updates its clock
regionB := hlc.NewHLC("region-b", "node-1")
regionB.UpdateFromRemote(msgID.HLC)

// Region B's next ID will be causally after the received message
nextID := regionB.GenerateID()
```

### Global Ordering and Conflict Resolution

```go
// Compare two global IDs for ordering
id1 := hlc.GlobalID{RegionID: "region-a", HLC: "1703123456789-5", Sequence: 1}
id2 := hlc.GlobalID{RegionID: "region-b", HLC: "1703123456790-3", Sequence: 1}

result := hlc.CompareGlobalID(id1, id2)
if result < 0 {
    fmt.Println("id1 comes before id2")
} else if result > 0 {
    fmt.Println("id1 comes after id2")  
} else {
    fmt.Println("id1 and id2 are identical")
}
```

## API Reference

### Types

#### `HLC`
```go
type HLC struct {
    // Thread-safe hybrid logical clock
}
```

#### `GlobalID`
```go
type GlobalID struct {
    RegionID string `json:"region_id"` // Region identifier
    HLC      string `json:"hlc"`       // HLC timestamp "physical-logical"
    Sequence int64  `json:"sequence"`  // Local sequence number
}
```

### Functions

#### `NewHLC(regionID, nodeID string) *HLC`
Creates a new HLC instance for the specified region and node.

#### `(h *HLC) GenerateID() GlobalID`
Generates a new globally unique ID with HLC timestamp. Thread-safe.

#### `(h *HLC) UpdateFromRemote(remoteHLC string) error`
Updates the local clock based on a remote HLC timestamp to maintain causal ordering.

#### `(h *HLC) GetCurrentTimestamp() string`
Returns the current HLC timestamp as a string in format "physical-logical".

#### `CompareGlobalID(id1, id2 GlobalID) int`
Compares two GlobalIDs and returns:
- `-1` if id1 < id2
- `0` if id1 == id2  
- `1` if id1 > id2

Comparison order:
1. HLC physical time
2. HLC logical time
3. RegionID (tiebreaker)
4. Sequence number

## Design Decisions

### RegionID Tiebreaker
When two messages have identical HLC timestamps, the RegionID is used as a tiebreaker to ensure deterministic ordering. This is crucial for conflict resolution in multi-region active-active systems.

### Thread Safety
The HLC struct uses a read-write mutex to protect the physical and logical time fields, while the sequence counter uses atomic operations for optimal performance.

### Error Handling
The `UpdateFromRemote` function validates the HLC format and returns descriptive errors for invalid input, making debugging easier.

### Performance Optimization
- Atomic operations for sequence counter
- Efficient string parsing for HLC timestamps
- Minimal allocations in hot paths

## Performance

Benchmarks on Apple M2 Max:

```
BenchmarkGenerateID-12           7779439    139.2 ns/op
BenchmarkCompareGlobalID-12     10572872    115.7 ns/op  
BenchmarkUpdateFromRemote-12    13515187     90.91 ns/op
```

## Testing

Run the test suite:

```bash
cd libs/hlc
go test -v                    # Run all tests
go test -bench=.             # Run benchmarks
go test -race                # Test for race conditions
```

The test suite includes:
- Monotonicity guarantees
- Concurrent safety
- Clock synchronization
- Invalid input handling
- Comparison logic
- Performance benchmarks

## Integration with Multi-Region Architecture

This HLC implementation is designed for the multi-region active-active IM chat system:

1. **Message Ordering**: Each message gets a GlobalID for consistent ordering across regions
2. **Conflict Resolution**: LWW (Last Write Wins) using HLC timestamps with RegionID tiebreaker
3. **Causal Consistency**: Remote clock updates maintain happens-before relationships
4. **Monitoring**: HLC timestamps enable cross-region latency monitoring

## Requirements Satisfied

This implementation satisfies requirement **2.1** from the multi-region active-active specification:

- ✅ Uses Hybrid Logical Clock (HLC) for transaction ID generation
- ✅ Transaction ID format: `{region_id}-{hlc_timestamp}-{logical_counter}`
- ✅ Combines physical time and logical counter to prevent clock rollback issues
- ✅ Globally unique transaction IDs with causal relationship support

## License

MIT License - see project root for details.