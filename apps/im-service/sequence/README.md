# Sequence Generator

The sequence generator provides monotonically increasing sequence numbers for messages in the IM chat system. It ensures messages are properly ordered within conversations.

## Features

- **Redis-based sequence generation**: Uses Redis INCR for atomic, distributed sequence generation
- **Conversation-specific sequences**: Separate sequences for each conversation (private or group)
- **User ID sorting**: Automatically sorts user IDs for private chats to ensure consistent conversation IDs
- **MySQL backup**: Periodic snapshots to MySQL for disaster recovery
- **Concurrent-safe**: Thread-safe operations with proper locking

## Architecture

### Sequence Generator

The `SequenceGenerator` uses Redis INCR to atomically increment sequence numbers. Each conversation has its own sequence starting from 1.

**Key Format**: `seq:{conversation_type}:{conversation_id}`

- Private chat: `seq:private:user001:user002` (user IDs sorted alphabetically)
- Group chat: `seq:group:group001`

### Sequence Backup

The `SequenceBackup` component periodically saves snapshots to MySQL for disaster recovery.

**Default Snapshot Interval**: Every 10,000 messages

**Trade-off**: In case of Redis failure, up to 10,000 messages may have duplicate sequence numbers after recovery. This is acceptable because:
1. Client-side deduplication prevents duplicate display
2. Message IDs (msg_id) are unique and used for final deduplication
3. The probability of Redis failure is low

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/redis/go-redis/v9"
    "github.com/pingxin403/cuckoo/apps/im-service/sequence"
)

// Create Redis client
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Create sequence generator
sg := sequence.NewSequenceGenerator(redisClient)

// Generate sequence for private chat
seq, err := sg.GeneratePrivateChatSequence(ctx, "user001", "user002")
if err != nil {
    // Handle error
}

// Generate sequence for group chat
seq, err = sg.GenerateGroupChatSequence(ctx, "group001")
if err != nil {
    // Handle error
}
```

### With MySQL Backup

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

// Create MySQL connection
db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/im_chat")
if err != nil {
    // Handle error
}

// Create backup manager (snapshot every 10,000 messages)
backup := sequence.NewSequenceBackup(db, sg, 10000)

// Generate sequence
seq, err := sg.GenerateGroupChatSequence(ctx, "group001")
if err != nil {
    // Handle error
}

// Check if snapshot is needed
if backup.ShouldSnapshot(seq) {
    err = backup.SaveSnapshot(ctx, sequence.ConversationTypeGroup, "group001", seq)
    if err != nil {
        // Handle error (log but don't fail message delivery)
    }
}
```

### Recovery from MySQL

```go
// Recover sequence after Redis failure
recoveredSeq, err := backup.RecoverSequence(ctx, sequence.ConversationTypeGroup, "group001")
if err != nil {
    // Handle error
}

// Note: You'll need to initialize Redis with this value
// This requires a SET operation which is not part of the current interface
```

## Requirements Validation

This implementation validates the following requirements:

- **Requirement 16.1**: Monotonically increasing sequence numbers ✅
- **Requirement 16.2**: Sequence assignment for private chats ✅
- **Requirement 16.3**: Sequence assignment for group chats ✅
- **Requirement 16.6**: Distributed sequence generator using Redis ✅
- **Requirement 16.7**: MySQL backup for disaster recovery ✅

## Testing

The package includes comprehensive unit tests and property-based tests:

- **Unit Tests**: 19 tests covering all basic functionality
- **Property-Based Tests**: 6 properties tested with 100 iterations each
  - Property 1: Sequence monotonicity
  - Property 2: User ID sorting consistency
  - Property 3: Conversation independence
  - Property 4: Concurrent sequence generation
  - Property 5: GetCurrentSequence idempotence
  - Property 6: Empty input validation

Run tests:
```bash
go test ./sequence/... -v
```

Run with coverage:
```bash
go test ./sequence/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Performance

- **Redis INCR**: O(1) time complexity
- **Throughput**: Limited only by Redis performance (~100K ops/sec per instance)
- **Latency**: Typically <1ms for sequence generation
- **Scalability**: Horizontally scalable with Redis cluster

## Error Handling

- **Redis connection failure**: Returns error, caller should retry or route to offline channel
- **MySQL backup failure**: Logged but doesn't block message delivery
- **Empty input validation**: Returns descriptive errors for invalid inputs

## Future Enhancements

1. **Redis SET support**: Add ability to initialize Redis keys from MySQL snapshots
2. **Automatic recovery**: Detect Redis failures and auto-recover from MySQL
3. **Configurable snapshot intervals**: Per-conversation snapshot policies
4. **Metrics**: Expose Prometheus metrics for sequence generation rate and backup status
