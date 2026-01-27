# Read Receipt Service

The Read Receipt Service handles tracking and delivery of read receipts for messages in the IM Chat System.

## Overview

Read receipts allow users to see when their messages have been read by recipients. The service supports:

- **Private chat read receipts**: Track when 1-on-1 messages are read
- **Group chat read receipts**: Track which members have read group messages
- **Multi-device support**: Track read status per device
- **Offline sender support**: Store read receipts for offline senders
- **Unread message tracking**: Query unread message counts and lists

## Requirements Validated

- **Requirement 5.1**: Send read receipt when recipient reads a message
- **Requirement 5.2**: Update message status to "read" with timestamp
- **Requirement 5.3**: Push read receipt to online sender in real-time
- **Requirement 5.4**: Store read receipt for offline sender
- **Requirement 5.5**: Support read receipts for both private and group chats

## Architecture

### Database Schema

**read_receipts table:**
```sql
CREATE TABLE read_receipts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    msg_id VARCHAR(36) NOT NULL,
    reader_id VARCHAR(64) NOT NULL,
    sender_id VARCHAR(64) NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    conversation_type ENUM('private', 'group') NOT NULL,
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    device_id VARCHAR(64),
    
    UNIQUE KEY idx_msg_reader_device (msg_id, reader_id, device_id),
    KEY idx_sender_conversation (sender_id, conversation_id),
    KEY idx_msg_id (msg_id),
    KEY idx_read_at (read_at)
);
```

**offline_messages table (updated):**
```sql
ALTER TABLE offline_messages 
ADD COLUMN read_at TIMESTAMP NULL DEFAULT NULL,
ADD KEY idx_user_read_status (user_id, read_at),
ADD KEY idx_user_unread (user_id, read_at, timestamp);
```

### Components

1. **ReadReceiptService**: Core service for read receipt operations
2. **HTTPHandler**: REST API endpoints for read receipts
3. **WebSocket Integration**: Real-time read receipt delivery (implemented in Gateway Service)

## API Endpoints

### Mark Message as Read

**POST** `/api/v1/messages/read`

Marks a message as read and creates a read receipt.

**Request Body:**
```json
{
  "msg_id": "msg-123",
  "reader_id": "user-001",
  "sender_id": "user-002",
  "conversation_id": "conv-123",
  "conversation_type": "private",
  "device_id": "device-001"
}
```

**Response:**
```json
{
  "success": true,
  "receipt": {
    "id": 1,
    "msg_id": "msg-123",
    "reader_id": "user-001",
    "sender_id": "user-002",
    "conversation_id": "conv-123",
    "conversation_type": "private",
    "read_at": "2024-01-24T10:30:00Z",
    "device_id": "device-001"
  }
}
```

### Get Unread Count

**GET** `/api/v1/messages/unread/count?user_id={user_id}`

Returns the count of unread messages for a user.

**Response:**
```json
{
  "user_id": "user-001",
  "count": 5
}
```

### Get Unread Messages

**GET** `/api/v1/messages/unread?user_id={user_id}&limit={limit}&offset={offset}`

Returns a list of unread message IDs for a user.

**Response:**
```json
{
  "user_id": "user-001",
  "msg_ids": ["msg-001", "msg-002", "msg-003"],
  "limit": 50,
  "offset": 0
}
```

### Get Read Receipts for Message

**GET** `/api/v1/messages/receipts?msg_id={msg_id}`

Returns all read receipts for a specific message (useful for group chats).

**Response:**
```json
{
  "msg_id": "msg-123",
  "receipts": [
    {
      "id": 1,
      "msg_id": "msg-123",
      "reader_id": "user-001",
      "sender_id": "user-002",
      "conversation_id": "group-001",
      "conversation_type": "group",
      "read_at": "2024-01-24T10:30:00Z",
      "device_id": "device-001"
    },
    {
      "id": 2,
      "msg_id": "msg-123",
      "reader_id": "user-003",
      "sender_id": "user-002",
      "conversation_id": "group-001",
      "conversation_type": "group",
      "read_at": "2024-01-24T10:31:00Z",
      "device_id": "device-002"
    }
  ]
}
```

### Mark Conversation as Read

**POST** `/api/v1/conversations/read`

Marks all messages in a conversation as read (bulk operation).

**Request Body:**
```json
{
  "user_id": "user-001",
  "conversation_id": "conv-123"
}
```

**Response:**
```json
{
  "success": true,
  "rows_affected": 10
}
```

## Usage Example

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    "github.com/pingxin403/cuckoo/apps/im-service/readreceipt"
)

func main() {
    // Initialize database connection
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/im_chat")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create read receipt service
    service := readreceipt.NewReadReceiptService(db)
    
    // Mark a message as read
    receipt, err := service.MarkAsRead(
        context.Background(),
        "msg-123",           // msgID
        "user-001",          // readerID
        "user-002",          // senderID
        "conv-123",          // conversationID
        "private",           // conversationType
        "device-001",        // deviceID
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Message marked as read at: %v", receipt.ReadAt)
    
    // Get unread count
    count, err := service.GetUnreadCount(context.Background(), "user-001")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User has %d unread messages", count)
}
```

## Integration with Gateway Service

The Gateway Service is responsible for delivering read receipts to online senders in real-time via WebSocket.

**Flow:**
1. Client reads a message and calls `POST /api/v1/messages/read`
2. Read Receipt Service creates receipt and updates database
3. Read Receipt Service publishes event to message bus (Kafka)
4. Gateway Service consumes event and pushes to sender's WebSocket
5. If sender is offline, receipt is stored for later retrieval

**WebSocket Message Format:**
```json
{
  "type": "read_receipt",
  "data": {
    "msg_id": "msg-123",
    "reader_id": "user-001",
    "read_at": "2024-01-24T10:30:00Z",
    "device_id": "device-001"
  }
}
```

## Multi-Device Support

Read receipts are tracked per device to support multi-device scenarios:

- Each device generates a unique `device_id` (UUID v4)
- Read receipts are stored with `device_id`
- When a message is read on one device, other devices receive the read receipt
- Unique constraint: `(msg_id, reader_id, device_id)`

## Group Chat Read Receipts

For group chats, read receipts track which members have read the message:

- Each member's read status is tracked independently
- Sender can query all read receipts for a message
- UI can display "Read by 5 of 10 members"
- Individual read timestamps are available

## Performance Considerations

### Indexes

The service uses several indexes for optimal performance:

- `idx_msg_reader_device`: Unique constraint and fast duplicate detection
- `idx_sender_conversation`: Fast queries for sender's conversation receipts
- `idx_msg_id`: Fast lookup of all receipts for a message
- `idx_user_read_status`: Fast unread count queries
- `idx_user_unread`: Optimized unread message list queries

### Scalability

- Read receipts are stored in MySQL for durability
- Unread counts can be cached in Redis for faster access
- Read receipt delivery uses Kafka for reliable async processing
- Database partitioning can be applied if needed (by user_id)

## Testing

The service includes comprehensive unit tests:

```bash
# Run unit tests
go test ./readreceipt/...

# Run with coverage
go test -cover ./readreceipt/...

# Run specific test
go test -run TestMarkAsRead ./readreceipt/...
```

**Test Coverage:**
- ✅ Mark message as read
- ✅ Get read receipts for message
- ✅ Get unread count
- ✅ Get unread messages
- ✅ Mark conversation as read
- ✅ Get read status
- ✅ Error handling
- ✅ Database transaction handling

## Future Enhancements

- [ ] Read receipt aggregation for large groups (>1,000 members)
- [ ] Read receipt analytics (average read time, read rate)
- [ ] Configurable read receipt privacy settings
- [ ] Read receipt expiration (auto-delete after 30 days)
- [ ] Batch read receipt operations for performance

## References

- [Requirements Document](../../.kiro/specs/im-chat-system/requirements.md) - Requirement 5
- [Design Document](../../.kiro/specs/im-chat-system/design.md) - Read Receipt Design
- [Task List](../../.kiro/specs/im-chat-system/tasks.md) - Task 13
