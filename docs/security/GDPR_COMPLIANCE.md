# GDPR Compliance Guide for IM Chat System

## Overview

This guide explains how the IM Chat System implements GDPR (General Data Protection Regulation) compliance, including the right to erasure (Article 17) and the right to data portability (Article 20).

## GDPR Requirements

### Right to Erasure (Article 17)

Users have the right to request deletion of their personal data:
- All messages sent or received
- User profile information
- Connection logs
- Audit logs (after retention period)

### Right to Data Portability (Article 20)

Users have the right to receive their personal data in a portable format:
- All messages in JSON format
- User profile data
- Message metadata (timestamps, recipients, etc.)

### Data Retention

- **Messages**: 7 days in offline storage (configurable)
- **Audit Logs**: 90 days minimum
- **Deduplication Data**: 7 days
- **User Data**: Until deletion requested

## Message Deletion Implementation

### API Endpoint

**DELETE /api/v1/messages**

Delete all messages for a user across all storage systems.

**Request**:
```http
DELETE /api/v1/messages HTTP/1.1
Host: gateway.example.com
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "user_id": "user123",
  "confirmation": "DELETE_ALL_DATA"
}
```

**Response**:
```json
{
  "status": "success",
  "deleted": {
    "offline_messages": 1234,
    "kafka_messages": 56,
    "redis_dedup_keys": 789,
    "audit_logs": 0
  },
  "timestamp": "2025-01-25T10:30:45Z",
  "request_id": "req_abc123"
}
```

### Cascade Deletion Process

```
┌─────────────────────────────────────────────────────────┐
│              Message Deletion Request                    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  1. Verify User Identity (JWT Token)                    │
│  2. Require Explicit Confirmation                        │
│  3. Log Deletion Request (Audit)                         │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│  MySQL   │  │  Kafka   │  │  Redis   │
│ Offline  │  │ Messages │  │  Dedup   │
│ Messages │  │ (Recent) │  │   Keys   │
└──────────┘  └──────────┘  └──────────┘
        │            │            │
        └────────────┼────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│  4. Return Deletion Summary                              │
│  5. Log Deletion Completion (Audit)                      │
└─────────────────────────────────────────────────────────┘
```

### Implementation Details

#### 1. MySQL Offline Messages

```sql
-- Delete all offline messages for user
DELETE FROM offline_messages 
WHERE user_id = ? OR sender_id = ?;

-- Verify deletion
SELECT COUNT(*) FROM offline_messages 
WHERE user_id = ? OR sender_id = ?;
```

#### 2. Kafka Messages

**Note**: Kafka messages cannot be selectively deleted. Options:

**Option A**: Wait for retention period (7 days)
```go
// Log that Kafka messages will be deleted after retention
log.Info("Kafka messages will be deleted after retention period",
    "user_id", userID,
    "retention_days", 7)
```

**Option B**: Tombstone messages (Kafka Compaction)
```go
// Publish tombstone message (null value)
producer.Produce(&kafka.Message{
    TopicPartition: kafka.TopicPartition{
        Topic:     &topic,
        Partition: kafka.PartitionAny,
    },
    Key:   []byte(userID),
    Value: nil, // Tombstone
})
```

#### 3. Redis Deduplication Keys

```go
// Delete all dedup keys for user's messages
pattern := fmt.Sprintf("dedup:*:%s:*", userID)
keys, err := redisClient.Keys(ctx, pattern).Result()
if err != nil {
    return err
}

if len(keys) > 0 {
    deleted, err := redisClient.Del(ctx, keys...).Result()
    if err != nil {
        return err
    }
}
```

#### 4. Registry (etcd)

```go
// Delete user's device registrations
prefix := fmt.Sprintf("/registry/users/%s/", userID)
_, err := etcdClient.Delete(ctx, prefix, clientv3.WithPrefix())
if err != nil {
    return err
}
```

### Deletion Service Implementation

**File**: `apps/im-service/deletion/deletion_service.go`

```go
package deletion

import (
    "context"
    "fmt"
    "time"
)

type DeletionService struct {
    mysqlClient  *sql.DB
    redisClient  *redis.Client
    kafkaProducer *kafka.Producer
    etcdClient   *clientv3.Client
    auditLogger  *AuditLogger
}

type DeletionResult struct {
    OfflineMessages int64
    KafkaMessages   int64
    RedisKeys       int64
    RegistryKeys    int64
    Timestamp       time.Time
}

func (s *DeletionService) DeleteUserData(ctx context.Context, userID string) (*DeletionResult, error) {
    result := &DeletionResult{
        Timestamp: time.Now(),
    }

    // Audit log: deletion started
    s.auditLogger.LogDeletionRequest(userID)

    // 1. Delete offline messages from MySQL
    offlineCount, err := s.deleteOfflineMessages(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to delete offline messages: %w", err)
    }
    result.OfflineMessages = offlineCount

    // 2. Tombstone Kafka messages (if using compaction)
    kafkaCount, err := s.tombstoneKafkaMessages(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to tombstone kafka messages: %w", err)
    }
    result.KafkaMessages = kafkaCount

    // 3. Delete Redis dedup keys
    redisCount, err := s.deleteRedisKeys(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to delete redis keys: %w", err)
    }
    result.RedisKeys = redisCount

    // 4. Delete registry entries
    registryCount, err := s.deleteRegistryEntries(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to delete registry entries: %w", err)
    }
    result.RegistryKeys = registryCount

    // Audit log: deletion completed
    s.auditLogger.LogDeletionComplete(userID, result)

    return result, nil
}

func (s *DeletionService) deleteOfflineMessages(ctx context.Context, userID string) (int64, error) {
    query := `DELETE FROM offline_messages WHERE user_id = ? OR sender_id = ?`
    result, err := s.mysqlClient.ExecContext(ctx, query, userID, userID)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

func (s *DeletionService) deleteRedisKeys(ctx context.Context, userID string) (int64, error) {
    pattern := fmt.Sprintf("dedup:*:%s:*", userID)
    keys, err := s.redisClient.Keys(ctx, pattern).Result()
    if err != nil {
        return 0, err
    }

    if len(keys) == 0 {
        return 0, nil
    }

    deleted, err := s.redisClient.Del(ctx, keys...).Result()
    return deleted, err
}

func (s *DeletionService) tombstoneKafkaMessages(ctx context.Context, userID string) (int64, error) {
    // Publish tombstone message for user
    topic := "user_deletions"
    err := s.kafkaProducer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &topic,
            Partition: kafka.PartitionAny,
        },
        Key:   []byte(userID),
        Value: nil, // Tombstone
    }, nil)
    
    if err != nil {
        return 0, err
    }
    
    return 1, nil
}

func (s *DeletionService) deleteRegistryEntries(ctx context.Context, userID string) (int64, error) {
    prefix := fmt.Sprintf("/registry/users/%s/", userID)
    resp, err := s.etcdClient.Delete(ctx, prefix, clientv3.WithPrefix())
    if err != nil {
        return 0, err
    }
    return resp.Deleted, nil
}
```

## Data Export Implementation

### API Endpoint

**GET /api/v1/export**

Export all user data in portable JSON format.

**Request**:
```http
GET /api/v1/export HTTP/1.1
Host: gateway.example.com
Authorization: Bearer <jwt_token>
```

**Response**:
```json
{
  "user_id": "user123",
  "export_date": "2025-01-25T10:30:45Z",
  "data": {
    "profile": {
      "user_id": "user123",
      "username": "john_doe",
      "email": "john@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    },
    "messages": [
      {
        "msg_id": "msg789",
        "conversation_type": "private",
        "conversation_id": "conv123",
        "sender_id": "user123",
        "recipient_id": "user456",
        "content": "Hello, world!",
        "timestamp": "2025-01-25T10:00:00Z",
        "sequence_number": 12345,
        "delivered": true,
        "read": true
      }
    ],
    "statistics": {
      "total_messages_sent": 1234,
      "total_messages_received": 5678,
      "total_conversations": 42
    }
  }
}
```

### Export Service Implementation

**File**: `apps/im-service/export/export_service.go`

```go
package export

import (
    "context"
    "encoding/json"
    "time"
)

type ExportService struct {
    mysqlClient *sql.DB
    userService *UserService
}

type UserExport struct {
    UserID     string                 `json:"user_id"`
    ExportDate time.Time              `json:"export_date"`
    Data       map[string]interface{} `json:"data"`
}

func (s *ExportService) ExportUserData(ctx context.Context, userID string) (*UserExport, error) {
    export := &UserExport{
        UserID:     userID,
        ExportDate: time.Now(),
        Data:       make(map[string]interface{}),
    }

    // 1. Export user profile
    profile, err := s.exportProfile(ctx, userID)
    if err != nil {
        return nil, err
    }
    export.Data["profile"] = profile

    // 2. Export messages
    messages, err := s.exportMessages(ctx, userID)
    if err != nil {
        return nil, err
    }
    export.Data["messages"] = messages

    // 3. Export statistics
    stats, err := s.exportStatistics(ctx, userID)
    if err != nil {
        return nil, err
    }
    export.Data["statistics"] = stats

    return export, nil
}

func (s *ExportService) exportMessages(ctx context.Context, userID string) ([]map[string]interface{}, error) {
    query := `
        SELECT msg_id, conversation_type, conversation_id, sender_id, 
               recipient_id, content, timestamp, sequence_number, 
               delivered, read_at
        FROM offline_messages
        WHERE user_id = ? OR sender_id = ?
        ORDER BY timestamp DESC
    `
    
    rows, err := s.mysqlClient.QueryContext(ctx, query, userID, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var messages []map[string]interface{}
    for rows.Next() {
        var msg Message
        err := rows.Scan(&msg.MsgID, &msg.ConversationType, &msg.ConversationID,
            &msg.SenderID, &msg.RecipientID, &msg.Content, &msg.Timestamp,
            &msg.SequenceNumber, &msg.Delivered, &msg.ReadAt)
        if err != nil {
            return nil, err
        }
        
        messages = append(messages, map[string]interface{}{
            "msg_id":            msg.MsgID,
            "conversation_type": msg.ConversationType,
            "conversation_id":   msg.ConversationID,
            "sender_id":         msg.SenderID,
            "recipient_id":      msg.RecipientID,
            "content":           msg.Content,
            "timestamp":         msg.Timestamp,
            "sequence_number":   msg.SequenceNumber,
            "delivered":         msg.Delivered,
            "read":              msg.ReadAt != nil,
        })
    }

    return messages, nil
}
```

## Audit Logging

### Audit Log Format

```json
{
  "timestamp": "2025-01-25T10:30:45Z",
  "event_type": "data_deletion_request",
  "user_id": "user123",
  "requester_id": "user123",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "result": "success",
  "details": {
    "offline_messages_deleted": 1234,
    "redis_keys_deleted": 789,
    "kafka_tombstones": 56
  }
}
```

### Audit Events

- `data_deletion_request`: User requests data deletion
- `data_deletion_complete`: Data deletion completed
- `data_export_request`: User requests data export
- `data_export_complete`: Data export completed
- `data_access`: User accesses their data
- `data_modification`: User modifies their data

### Audit Log Retention

- **Minimum**: 90 days
- **Recommended**: 1 year
- **Maximum**: 7 years (depending on jurisdiction)

## Testing

### Test Message Deletion

```bash
# Request deletion
curl -X DELETE https://gateway.example.com/api/v1/messages \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "confirmation": "DELETE_ALL_DATA"
  }'

# Verify deletion
curl https://gateway.example.com/api/v1/messages \
  -H "Authorization: Bearer $JWT_TOKEN"
# Should return empty array
```

### Test Data Export

```bash
# Request export
curl https://gateway.example.com/api/v1/export \
  -H "Authorization: Bearer $JWT_TOKEN" \
  > user_data_export.json

# Verify export format
jq . user_data_export.json
```

## Compliance Checklist

- [ ] Implement message deletion API
- [ ] Cascade deletion across all storage systems
- [ ] Implement data export API
- [ ] Export data in portable format (JSON)
- [ ] Implement audit logging for all data operations
- [ ] Set up audit log retention (90 days minimum)
- [ ] Document data retention policies
- [ ] Implement user consent management
- [ ] Create privacy policy
- [ ] Implement data breach notification process
- [ ] Train staff on GDPR compliance
- [ ] Appoint Data Protection Officer (if required)

## Privacy Policy

Users must be informed about:
- What data is collected
- How data is used
- How long data is retained
- User rights (access, deletion, portability)
- How to exercise rights
- Contact information for privacy inquiries

## Support

For GDPR compliance questions:
- Email: privacy@example.com
- DPO: dpo@example.com
- Documentation: https://wiki.example.com/privacy/gdpr
