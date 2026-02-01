# Message Synchronization Property-Based Tests

This document describes the property-based tests implemented for the message synchronization system.

## Overview

Property-based tests validate universal properties that should hold across all possible inputs for the message synchronization and conflict resolution components. These tests use the `pgregory.net/rapid` framework to generate random test scenarios.

## Running Property Tests

```bash
# Run property tests with default iterations (100)
go test -tags=property -v ./sync/

# Run with more iterations for thorough testing
go test -tags=property -v -rapid.checks=1000 ./sync/

# Run specific property test
go test -tags=property -v -run TestProperty_MessageEventualConsistency ./sync/
```

## Implemented Properties

### Property 4: Message Eventual Consistency (消息最终一致性)
**Validates: Requirements 1.1**

**Test**: `TestProperty_MessageEventualConsistency`

**Property**: Messages eventually reach consistency across regions regardless of network delays or failures.

**Verification**:
- Creates 2-4 regions with message syncers
- Generates 3-15 messages in random regions
- Simulates network delays (10-100ms) during synchronization
- Verifies all messages eventually exist in all regions
- Ensures message ordering is consistent across regions
- Validates convergence to identical final state

**Key Assertions**:
- All messages must be retrievable from all regions after eventual consistency period
- Message content and metadata must be identical across regions
- Message ordering (based on HLC timestamps) must be consistent across regions
- All regions must converge to the same final state

### Property 5: Conflict Resolution Determinism (冲突解决确定性)
**Validates: Requirements 2.2**

**Test**: `TestProperty_ConflictResolutionDeterminism`

**Property**: Conflict resolution is deterministic - the same conflict always resolves the same way across all regions.

**Verification**:
- Creates 2-4 regions with message syncers
- Generates 2-8 conflict groups (messages with same ID but different content)
- Creates 2-3 conflicting versions per group with different timestamps
- Syncs conflicting messages to all regions
- Verifies all regions resolve conflicts to the same winner
- Validates LWW (Last Write Wins) strategy based on HLC timestamps

**Key Assertions**:
- All regions must choose the same winner for each conflict
- Winner must be the message with the latest HLC timestamp (LWW strategy)
- Conflict resolution must be consistent across multiple regions
- Conflict metrics should indicate conflicts were detected and resolved

### Property 6: Message Sync Idempotency
**Test**: `TestProperty_MessageSyncIdempotency`

**Property**: Message synchronization is idempotent - multiple sync operations of the same message should not create duplicates.

**Verification**:
- Creates two regions
- Syncs the same message multiple times (2-10 attempts)
- Verifies only one copy exists in the target region
- Ensures content integrity is maintained
- Validates error handling for duplicate sync attempts

**Key Assertions**:
- Only one copy of the message should exist after multiple sync attempts
- Message content must remain uncorrupted
- Error count should be reasonable (allowing for deduplication logic)

### Property 7: Cross-Region Causal Consistency
**Test**: `TestProperty_CrossRegionCausalConsistency`

**Property**: Causal relationships are preserved across regions during synchronization.

**Verification**:
- Creates three regions to test complex causal chains
- Generates causal chains: A → B → C (messages with happens-before relationships)
- Syncs messages across regions maintaining causal dependencies
- Verifies causal ordering is preserved in all regions
- Validates sequence numbers maintain causal relationships

**Key Assertions**:
- Messages in causal chains must maintain their ordering across all regions
- Sequence numbers must be monotonically increasing within causal chains
- HLC timestamps must respect causal relationships
- All regions must have identical causal ordering

## Test Coverage

The property-based tests complement existing unit tests by:

1. **Distributed System Properties**: Testing properties that only emerge in multi-region scenarios
2. **Network Simulation**: Simulating realistic network delays and failures
3. **Conflict Generation**: Automatically generating realistic conflict scenarios
4. **Stress Testing**: Running thousands of iterations with random parameters
5. **Edge Case Discovery**: Finding edge cases in synchronization and conflict resolution

## Key Benefits

1. **Eventual Consistency Validation**: Ensures the system converges to consistent state
2. **Deterministic Conflict Resolution**: Validates that conflicts resolve consistently
3. **Causal Consistency**: Ensures causal relationships are preserved
4. **Idempotency Guarantees**: Validates that operations can be safely retried
5. **Regression Prevention**: Catches subtle distributed system bugs

## Integration with Requirements

### Requirement 1.1: 消息跨地域复制
- **Property 4** validates that messages are eventually consistent across regions
- **Property 6** ensures sync operations are idempotent and safe to retry
- **Property 7** validates that causal relationships are preserved

### Requirement 2.2: LWW 冲突解决
- **Property 5** validates that conflict resolution is deterministic
- Tests ensure LWW strategy based on HLC timestamps works correctly
- Verifies that all regions resolve conflicts to the same winner

## Test Parameters

The property tests use configurable parameters to balance thoroughness with execution time:

- **Number of regions**: 2-4 (realistic multi-region scenarios)
- **Number of messages**: 3-15 (sufficient to test ordering and conflicts)
- **Network delays**: 10-100ms (realistic cross-region latencies)
- **Conflict groups**: 2-8 (adequate conflict resolution testing)
- **Sync attempts**: 2-10 (idempotency testing)

## Failure Analysis

When property tests fail, they provide detailed information:

1. **Shrinking**: Rapid automatically finds minimal failing cases
2. **Detailed Logging**: Tests log intermediate states and decisions
3. **Assertion Context**: Clear error messages indicate which property was violated
4. **Reproducible Seeds**: Failed tests can be reproduced with specific seeds

## Performance Considerations

Property tests are designed to be:

- **Fast**: Each test completes in seconds, not minutes
- **Deterministic**: Results are reproducible given the same seed
- **Scalable**: Can be run with different iteration counts based on CI/local needs
- **Isolated**: Each test creates fresh instances to avoid interference

## Future Enhancements

Potential additions to the property test suite:

1. **Network Partition Testing**: Simulate network splits and recovery
2. **Byzantine Fault Tolerance**: Test behavior with malicious nodes
3. **Performance Properties**: Validate latency and throughput properties
4. **Garbage Collection**: Test message cleanup and TTL behavior
5. **Schema Evolution**: Test backward compatibility during upgrades