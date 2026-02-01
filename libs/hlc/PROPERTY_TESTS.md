# HLC Property-Based Tests

This document describes the property-based tests implemented for the Hybrid Logical Clock (HLC) library.

## Overview

Property-based tests validate universal properties that should hold across all possible inputs, complementing the unit tests that verify specific examples. These tests use the `pgregory.net/rapid` framework to generate random test cases.

## Running Property Tests

```bash
# Run property tests with default iterations (100)
go test -tags=property -v ./...

# Run with more iterations for thorough testing
go test -tags=property -v -rapid.checks=1000 ./...
```

## Implemented Properties

### Property 1: HLC Monotonicity Guarantee (HLC 单调性保证)
**Validates: Requirements 2.1**

**Test**: `TestProperty_HLCMonotonicity`

**Property**: HLC timestamps must be monotonically increasing within a single node, regardless of physical clock behavior.

**Verification**:
- Generates sequences of 2-50 IDs from a single HLC instance
- Verifies all IDs are in strictly monotonic order using `CompareGlobalID`
- Verifies sequence numbers are strictly increasing
- Tests with random region and node IDs

### Property 2: Causal Ordering Preservation (因果关系保序)
**Validates: Requirements 2.1**

**Test**: `TestProperty_CausalOrderingPreservation`

**Property**: Causal relationships are preserved across regions when HLC timestamps are synchronized.

**Verification**:
- Creates two HLC instances representing different regions
- Generates 3-20 events with causal relationships (cross-region synchronization)
- Verifies events can be sorted by HLC timestamp in causal order
- Ensures same-region events maintain their original order
- Verifies final events from both regions are greater than all previous events

### Property 3: Remote Synchronization Consistency (远程同步一致性)
**Validates: Requirements 2.1**

**Test**: `TestProperty_RemoteSynchronizationConsistency`

**Property**: Remote synchronization maintains consistency and convergence across different nodes.

**Verification**:
- Creates 2-5 HLC instances representing different nodes
- Performs 3-10 rounds of event generation and cross-node synchronization
- Verifies all final events are greater than all previous events
- Ensures all events are comparable and sortable
- Tests convergence: comparison relationships are deterministic and consistent

### Property 4: Concurrent Synchronization Safety
**Test**: `TestProperty_ConcurrentSynchronizationSafety`

**Property**: Concurrent synchronization operations maintain consistency without race conditions.

**Verification**:
- Generates 5-20 random remote timestamps
- Applies updates concurrently from multiple goroutines
- Verifies no errors occur during concurrent updates
- Ensures HLC still generates monotonic IDs after concurrent operations

### Property 5: HLC Timestamp Parsing Consistency
**Test**: `TestProperty_HLCTimestampParsing`

**Property**: HLC timestamp parsing is consistent and reversible.

**Verification**:
- Generates random valid physical and logical time components
- Creates HLC timestamp strings and parses them back
- Verifies parsing is error-free and components match
- Ensures reconstructed strings match originals

### Property 6: Global ID Comparison Properties
**Test**: `TestProperty_GlobalIDComparison`

**Property**: GlobalID comparison is reflexive, antisymmetric, and transitive.

**Verification**:
- Generates three random GlobalIDs
- Tests reflexivity: `a == a` for all IDs
- Tests antisymmetry: if `a < b`, then `b > a`
- Tests transitivity: if `a < b` and `b < c`, then `a < c`

## Test Coverage

The property-based tests complement the existing unit tests by:

1. **Stress Testing**: Running thousands of iterations with random inputs
2. **Edge Case Discovery**: Automatically finding edge cases that manual tests might miss
3. **Invariant Verification**: Ensuring mathematical properties hold universally
4. **Concurrency Testing**: Validating thread safety under various race conditions

## Key Benefits

1. **Comprehensive Coverage**: Tests properties across the entire input space
2. **Automatic Shrinking**: When failures occur, rapid automatically finds minimal failing cases
3. **Regression Prevention**: Catches subtle bugs that might be introduced during refactoring
4. **Documentation**: Properties serve as executable specifications of expected behavior

## Integration with Requirements

All property tests are linked to **Requirements 2.1** which specifies:
- HLC-based global transaction ID generation
- Causal relationship preservation
- Prevention of clock rollback issues
- Support for cross-region synchronization

The tests ensure these requirements are met under all possible conditions, not just the specific scenarios covered by unit tests.