# Local Storage for Multi-Region Active-Active Architecture

This package provides a simplified local storage implementation for the multi-region active-active IM chat system MVP. It serves as a replacement for MySQL during development and testing phases.

## Features

- **Dual Storage Modes**: SQLite with WAL mode or in-memory storage
- **Conflict Detection**: Built-in conflict detection for multi-region scenarios
- **CRUD Operations**: Complete Create, Read, Update, Delete operations
- **Batch Operations**: Efficient batch insert for high throughput
- **TTL Support**: Automatic expiration of old messages
- **Monitoring**: Conflict tracking and storage statistics

## Architecture

### Storage Modes

#### SQLite Mode (Production-like)
- Uses SQLite with WAL (Write-Ahead Logging) mode
- Persistent storage with ACID guarantees
- Optimized for concurrent read/write operations
- Suitable for MVP deployment

#### Memory Mode (Testing)
- In-memory storage using Go maps
- Fast operations for unit testing
- No persistence (data lost on restart)
- Thread-safe with mutex protection

### Data Model

```go
type LocalMessage struct {
    ID               int64             // Auto-increment ID
    MsgID            string            // Unique message identifier
    UserID           string            // Recipient user ID
    SenderID         string            // Sender user ID
    ConversationID   string            // Conversation identifier
    ConversationType string            // "private" or "group"
    Content          string            // Message content
    SequenceNumber   int64             // Message sequence in conversation
    Timestamp        int64             // Unix timestamp in milliseconds
    CreatedAt        time.Time         // Creation timestamp
    ExpiresAt        time.Time         // Expiration timestamp
    Metadata         map[string]string // Additional metadata
    RegionID         string            // Source region identifier
    GlobalID         string            // HLC-based global identifier
    Version          int64             // Version for conflict resolution
}
```

## Usage

### Basic Setup

```go
import "github.com/pingxin403/cuckoo/storage"

// SQLite mode
config := storage.Config{
    DatabasePath: "./data/region-a.db",
    RegionID:     "region-a",
    WALMode:      true,
    MemoryMode:   false,
    TTL:          7 * 24 * time.Hour,
}

store, err := storage.NewLocalStore(config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Memory mode (for testing)
config := storage.Config{
    RegionID:   "region-a",
    MemoryMode: true,
    TTL:        7 * 24 * time.Hour,
}

store, err := storage.NewLocalStore(config)
```

### Message Operations

```go
ctx := context.Background()

// Insert single message
message := storage.LocalMessage{
    MsgID:            "msg-123",
    UserID:           "user-456",
    SenderID:         "user-789",
    ConversationID:   "conv-abc",
    ConversationType: "private",
    Content:          "Hello, world!",
    SequenceNumber:   1,
    Timestamp:        time.Now().UnixMilli(),
    ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
    RegionID:         "region-a",
    GlobalID:         "region-a-1234567890-1",
    Version:          1,
}

err := store.Insert(ctx, message)

// Batch insert
messages := []storage.LocalMessage{message1, message2, message3}
err := store.BatchInsert(ctx, messages)

// Retrieve messages with pagination
messages, err := store.GetMessages(ctx, "user-456", 0, 50)

// Get specific message
message, err := store.GetMessageByID(ctx, "msg-123")
```

### Conflict Detection

```go
// Detect conflicts between local and remote messages
remoteMessage := storage.LocalMessage{
    MsgID:    "msg-123",
    Version:  2,
    RegionID: "region-b",
    GlobalID: "region-b-1234567891-1",
    // ... other fields
}

conflict, err := store.DetectConflict(ctx, remoteMessage)
if conflict != nil {
    // Handle conflict
    fmt.Printf("Conflict detected: %s wins\n", conflict.Resolution)
    
    // Record conflict for monitoring
    err := store.RecordConflict(ctx, *conflict)
}
```

### Maintenance Operations

```go
// Clean up expired messages
deleted, err := store.DeleteExpiredMessages(ctx, 1000)
fmt.Printf("Deleted %d expired messages\n", deleted)

// Get storage statistics
stats, err := store.GetStats(ctx)
fmt.Printf("Total messages: %v\n", stats["total_messages"])
fmt.Printf("Total conflicts: %v\n", stats["total_conflicts"])

// Get recent conflicts for monitoring
conflicts, err := store.GetConflicts(ctx, 100)
for _, conflict := range conflicts {
    fmt.Printf("Conflict: %s at %v\n", conflict.MessageID, conflict.ConflictTime)
}
```

## Configuration

### SQLite Configuration

The SQLite mode is optimized for multi-region scenarios:

```sql
PRAGMA foreign_keys = ON;           -- Enable foreign key constraints
PRAGMA synchronous = NORMAL;        -- Balance safety and performance
PRAGMA cache_size = -64000;         -- 64MB cache
PRAGMA temp_store = MEMORY;         -- Store temp tables in memory
PRAGMA mmap_size = 268435456;       -- 256MB memory map
PRAGMA journal_mode = WAL;          -- Write-Ahead Logging
PRAGMA wal_autocheckpoint = 1000;   -- Auto-checkpoint every 1000 pages
```

### Database Schema

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    msg_id TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    conversation_type TEXT NOT NULL,
    content TEXT NOT NULL,
    sequence_number INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    metadata TEXT, -- JSON string
    region_id TEXT NOT NULL,
    global_id TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE conflicts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    local_version INTEGER NOT NULL,
    remote_version INTEGER NOT NULL,
    local_region TEXT NOT NULL,
    remote_region TEXT NOT NULL,
    conflict_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolution TEXT NOT NULL
);
```

## Conflict Resolution

The storage implements Last Write Wins (LWW) conflict resolution:

1. **Detection**: Compare message versions and region IDs
2. **Resolution**: Use Global ID (HLC-based) for deterministic ordering
3. **Recording**: Log all conflicts for monitoring and analysis

### Conflict Resolution Logic

```go
if remoteMessage.GlobalID > localMessage.GlobalID {
    conflict.Resolution = "remote_wins"
} else {
    conflict.Resolution = "local_wins"
}
```

## Performance Characteristics

### SQLite Mode
- **Insert**: ~1000 messages/second (single thread)
- **Batch Insert**: ~10,000 messages/second (100 per batch)
- **Query**: ~50ms for user message retrieval
- **Storage**: ~1KB per message average

### Memory Mode
- **Insert**: ~100,000 messages/second
- **Query**: ~1ms for message retrieval
- **Memory**: ~2KB per message (including Go overhead)

## Testing

The package includes comprehensive unit tests covering:

- Both storage modes (SQLite and memory)
- CRUD operations
- Batch operations
- Conflict detection and resolution
- Pagination
- Error handling
- TTL cleanup

Run tests:

```bash
cd storage
go test -v
go test -race -v  # Test for race conditions
go test -cover    # Test coverage
```

## Integration with Multi-Region Architecture

This storage component is designed to integrate with:

1. **HLC (Hybrid Logical Clock)**: Uses HLC-generated Global IDs
2. **Message Syncer**: Provides conflict detection interface
3. **Monitoring**: Exposes conflict metrics and statistics
4. **Cleanup Jobs**: Supports TTL-based message expiration

## Migration Path

The simplified storage can be easily migrated to production systems:

1. **SQLite → MySQL**: Schema is compatible, data can be exported/imported
2. **Memory → Redis**: In-memory operations can be adapted to Redis
3. **Local → Distributed**: Conflict resolution logic remains the same

## Limitations

As a simplified MVP implementation:

- No sharding or partitioning
- Limited to single-node deployment
- No read replicas
- Basic conflict resolution (LWW only)
- No data compression
- No backup/restore functionality

## Future Enhancements

- **Sharding**: Distribute data across multiple SQLite files
- **Compression**: Compress message content for storage efficiency
- **Encryption**: Encrypt sensitive message data
- **Replication**: Built-in master-slave replication
- **Advanced Conflicts**: Support for custom conflict resolution strategies