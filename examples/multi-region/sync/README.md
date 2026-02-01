# Cross-Region Message Synchronizer

This package implements the cross-region message synchronizer for the multi-region active-active architecture.

## Features

### Core Functionality
- **Asynchronous Message Sync**: High-throughput message replication across regions
- **Synchronous Message Sync**: Critical business operations with acknowledgment
- **HLC Integration**: Hybrid Logical Clock for global ordering and causality
- **Conflict Resolution**: Last Write Wins (LWW) strategy with RegionID tiebreaker
- **Message Integrity**: SHA-256 checksum verification
- **Deduplication**: Prevents duplicate message processing

### Architecture Components

#### MessageSyncer
The main component that orchestrates cross-region synchronization:
- Manages Kafka producers/consumers for async and sync topics
- Integrates with HLC for global ID generation
- Handles conflict detection and resolution
- Provides metrics and monitoring

#### SyncMessage
Represents a message being synchronized across regions:
```go
type SyncMessage struct {
    ID               string            // Unique sync operation ID
    Type             string            // "async" or "sync"
    SourceRegion     string            // Origin region
    TargetRegion     string            // Destination region
    MessageID        string            // Original message ID
    GlobalID         hlc.GlobalID      // HLC-based global identifier
    ConversationID   string            // Chat conversation ID
    Content          string            // Message content
    SequenceNumber   int64             // Message sequence in conversation
    Checksum         string            // SHA-256 integrity checksum
    RequiresAck      bool              // Whether acknowledgment is required
    IsCritical       bool              // Critical business operation flag
}
```

#### SyncAck
Acknowledgment for synchronous operations:
```go
type SyncAck struct {
    MessageID    string    // Original message ID
    Status       string    // "success", "error", "conflict"
    Error        string    // Error details if failed
    ProcessTime  int64     // Processing time in milliseconds
}
```

## Usage

### Basic Setup
```go
// Create HLC clock
hlcClock := hlc.NewHLC("region-a", "node-1")

// Create message syncer
config := sync.DefaultConfig("region-a")
syncer, err := sync.NewMessageSyncer(
    "region-a",
    hlcClock,
    localQueue,
    localStorage,
    config,
    logger,
)

// Start the syncer
err = syncer.Start()
```

### Asynchronous Sync
```go
// Sync message asynchronously (fire-and-forget)
err := syncer.SyncMessageAsync(ctx, "region-b", message)
```

### Synchronous Sync
```go
// Sync message synchronously (wait for acknowledgment)
err := syncer.SyncMessageSync(ctx, "region-b", criticalMessage)
```

### Metrics
```go
metrics := syncer.GetMetrics()
fmt.Printf("Async syncs: %d, Conflicts: %d, Avg latency: %dms",
    metrics["async_sync_count"],
    metrics["conflict_count"],
    metrics["avg_sync_latency_ms"])
```

## Configuration

### Default Configuration
```go
Config{
    RegionID:            "region-a",
    AsyncTopic:          "cross_region_async",
    SyncTopic:           "cross_region_sync",
    AckTopic:            "cross_region_ack",
    SyncTimeout:         5 * time.Second,
    MaxRetries:          3,
    BatchSize:           100,
    FlushInterval:       100 * time.Millisecond,
    EnableChecksum:      true,
    EnableDeduplication: true,
    MetricsInterval:     30 * time.Second,
}
```

### Customization
```go
config := sync.DefaultConfig("region-a")
config.SyncTimeout = 10 * time.Second  // Longer timeout
config.MaxRetries = 5                  // More retries
config.EnableChecksum = false          // Disable checksums for performance
```

## Message Flow

### Async Flow
1. **Producer**: `SyncMessageAsync()` → Kafka async topic
2. **Consumer**: Receives message → Updates HLC → Detects conflicts → Stores message
3. **Metrics**: Updates async sync count and conflict metrics

### Sync Flow
1. **Producer**: `SyncMessageSync()` → Kafka sync topic → Wait for ACK
2. **Consumer**: Receives message → Processes → Sends ACK
3. **Producer**: Receives ACK → Returns success/error
4. **Metrics**: Updates sync latency and success/error counts

## Conflict Resolution

### Detection
Conflicts are detected when:
- Same message ID exists with different versions
- Different regions have different content for same message
- HLC timestamps indicate concurrent updates

### Resolution Strategy
1. **Compare Global IDs**: Use HLC timestamp comparison
2. **RegionID Tiebreaker**: If HLC timestamps are identical, use lexicographic region comparison
3. **LWW Application**: Later timestamp wins, update local storage
4. **Conflict Logging**: Record conflict details for monitoring

### Example
```go
// Region A: message-001 with HLC "1640995200000-0"
// Region B: message-001 with HLC "1640995200000-1"
// Resolution: Region B wins (higher logical counter)

conflict := &ConflictInfo{
    MessageID:     "message-001",
    LocalVersion:  1,
    RemoteVersion: 1,
    LocalRegion:   "region-a",
    RemoteRegion:  "region-b",
    Resolution:    "remote_wins",  // Based on HLC comparison
}
```

## Integration Points

### HLC Library
- Generates global IDs with causal ordering
- Updates local clock from remote timestamps
- Provides deterministic conflict resolution

### Local Queue
- Kafka-like message queue for cross-region communication
- Supports async/sync message patterns
- Handles message persistence and retry logic

### Local Storage
- SQLite-based message storage
- Conflict detection and recording
- Message deduplication and TTL management

## Monitoring and Metrics

### Key Metrics
- `async_sync_count`: Number of async messages synchronized
- `sync_sync_count`: Number of sync messages synchronized
- `conflict_count`: Number of conflicts detected and resolved
- `error_count`: Number of sync errors
- `avg_sync_latency_ms`: Average synchronous sync latency

### Health Checks
- Producer/consumer connectivity
- HLC clock synchronization
- Storage availability
- Queue buffer utilization

## Error Handling

### Retry Logic
- Exponential backoff for failed syncs
- Maximum retry limits per message
- Dead letter queue for permanently failed messages

### Timeout Handling
- Configurable sync timeouts
- Graceful degradation on timeout
- Async fallback for critical operations

### Network Partitions
- Message buffering during outages
- Automatic resync on reconnection
- Conflict resolution for partition healing

## Performance Considerations

### Throughput Optimization
- Batch message processing
- Configurable buffer sizes
- Parallel consumer threads

### Latency Optimization
- Direct region-to-region channels
- Minimal serialization overhead
- Efficient checksum algorithms

### Memory Management
- Message deduplication caching
- TTL-based cleanup
- Bounded queue sizes

## Testing

### Unit Tests
- Checksum calculation and verification
- Message conversion and serialization
- Configuration validation
- Metrics collection

### Integration Tests
- End-to-end message sync flows
- Conflict resolution scenarios
- HLC clock synchronization
- Network failure simulation

### Property-Based Tests
- Message ordering properties
- Conflict resolution determinism
- HLC causality preservation

## Requirements Validation

This implementation satisfies the following requirements:

### 1.1 消息跨地域复制
- ✅ 500ms sync latency (configurable timeout)
- ✅ Monitoring and alerting integration
- ✅ Network partition handling with local buffering
- ✅ Message deduplication

### 2.1 全局事务 ID 生成（基于 HLC）
- ✅ HLC-based global ID generation
- ✅ Format: {region_id}-{hlc_timestamp}-{logical_counter}
- ✅ Causal ordering preservation
- ✅ Clock skew protection

### 2.2 LWW 冲突解决
- ✅ Last Write Wins strategy
- ✅ Conflict logging and monitoring
- ✅ RegionID tiebreaker for determinism

## Future Enhancements

### Phase 1 (P1)
- Real Kafka integration (replace local queue)
- MySQL replication support
- Redis CRDT integration
- Prometheus metrics export

### Phase 2 (P2)
- Merkle tree-based reconciliation
- Batch conflict resolution
- Advanced retry strategies
- Performance profiling and optimization