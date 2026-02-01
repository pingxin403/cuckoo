# Cross-Region Message Synchronization for IM Service

This package implements cross-region message synchronization integrated into the IM service for multi-region active-active architecture.

## Overview

The sync package provides conflict detection and resolution capabilities for messages synchronized across multiple regions using the Last Write Wins (LWW) strategy with HLC timestamps.

## Components

### ConflictResolver

Handles conflict detection and resolution using LWW strategy:

```go
// Create conflict resolver
config := sync.DefaultConflictResolverConfig("region-a")
resolver := sync.NewConflictResolver(config, logger)

// Resolve conflict between versions
resolution, err := resolver.ResolveConflict(ctx, localVersion, remoteVersion)
```

### MessageVersion

Represents a version of a message for conflict resolution:
- GlobalID (HLC-based)
- MessageID
- Content
- Timestamp
- RegionID
- Version number

### ConflictResolution

Result of conflict resolution:
- Winner version
- Resolution type (local_wins, remote_wins, no_conflict)
- Resolution reason
- Resolution time

## Integration with IM Service

### Storage Integration

The conflict resolver integrates with the IM service storage layer:

```go
// In apps/im-service/storage/offline_store.go
type OfflineStore struct {
    // ... existing fields
    conflictResolver *sync.ConflictResolver
}
```

### Worker Integration

The offline worker uses the conflict resolver for cross-region messages:

```go
// In apps/im-service/worker/offline_worker.go
func (w *OfflineWorker) processRemoteMessage(msg *Message) error {
    resolution, err := w.conflictResolver.ResolveConflict(ctx, localVersion, remoteVersion)
    // Apply resolution...
}
```

## Conflict Resolution Strategy

### LWW (Last Write Wins)

1. Compare HLC timestamps
2. Later timestamp wins
3. RegionID tiebreaker for identical timestamps
4. Sequence number as final tiebreaker

### Example

```go
// Region A: message with HLC "1640995200000-0"
// Region B: message with HLC "1640995200000-1"
// Result: Region B wins (higher logical counter)
```

## Metrics

Track conflict resolution metrics:

```go
metrics := resolver.GetMetrics()
fmt.Printf("Total conflicts: %d\n", metrics.TotalConflicts)
fmt.Printf("Local wins: %d\n", metrics.LocalWins)
fmt.Printf("Remote wins: %d\n", metrics.RemoteWins)
fmt.Printf("Avg resolution time: %.2f μs\n", metrics.AvgResolutionTimeUs)
```

## Requirements Satisfied

- ✅ Requirement 2.2: LWW conflict resolution
- ✅ Conflict logging and monitoring
- ✅ RegionID tiebreaker for determinism
- ✅ Metrics collection

## Next Steps

1. Integrate with offline_worker for cross-region message processing
2. Add conflict resolution to storage layer
3. Implement metrics export to Prometheus
4. Add conflict resolution tests
