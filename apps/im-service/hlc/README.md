# HLC (Hybrid Logical Clock) for IM Service

A Go implementation of Hybrid Logical Clock integrated into the IM service for multi-region active-active architecture.

## Overview

This HLC implementation provides global ordering and conflict resolution capabilities for the IM service across multiple regions.

## Key Features

- **Thread-Safe**: Concurrent ID generation with proper synchronization
- **High Performance**: ~139ns per ID generation
- **Clock Skew Resilient**: Handles backward clock jumps gracefully
- **Deterministic Ordering**: RegionID tiebreaker ensures consistent results

## Usage in IM Service

```go
// Initialize HLC in IM service
hlc := hlc.NewHLC(config.RegionID, config.NodeID)

// Generate global message ID
globalID := hlc.GenerateID()

// Update from remote region
err := hlc.UpdateFromRemote(remoteMessage.HLC)
```

## Integration Points

1. **Sequence Generator** (`apps/im-service/sequence/`) - Integrates HLC for global message IDs
2. **Message Sync** (`apps/im-service/sync/`) - Uses HLC for cross-region synchronization
3. **Conflict Resolution** - LWW strategy based on HLC timestamps

## Requirements Satisfied

- ✅ Requirement 2.1: HLC-based global transaction ID generation
- ✅ Format: `{region_id}-{hlc_timestamp}-{logical_counter}`
- ✅ Causal ordering across regions
