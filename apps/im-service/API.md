# IM Service - API Documentation

## Overview

The IM Service provides gRPC APIs for message routing, offline message management, and read receipts.

## Table of Contents

1. [gRPC Services](#grpc-services)
2. [REST API Endpoints](#rest-api-endpoints)
3. [Message Flow](#message-flow)
4. [Error Codes](#error-codes)
5. [Usage Examples](#usage-examples)

## gRPC Services

### Service Definition

**Proto File**: `api/v1/im.proto`

```protobuf
syntax = "proto3";

package im.v1;

service IMService {
  // Route private message
  rpc RoutePrivateMessage(RoutePrivateMessageRequest) returns (RoutePrivateMessageResponse);
  
  // Route group message
  rpc RouteGroupMessage(RouteGroupMessageRequest) returns (RouteGroupMessageResponse);
  
  // Get message status
  rpc GetMessageStatus(GetMessageStatusRequest) returns (GetMessageStatusResponse);
}

message RoutePrivateMessageRequest {
  string msg_id = 1;
  string sender_id = 2;
  string recipient_id = 3;
  string content = 4;
  int64 timestamp = 5;
  map<string, string> metadata = 6;
}

message RoutePrivateMessageResponse {
  bool success = 1;
  string status = 2;  // "delivered", "offline", "failed"
  int64 sequence_number = 3;
  string error_message = 4;
}

message RouteGroupMessageRequest {
  string msg_id = 1;
  string sender_id = 2;
  string group_id = 3;
  string content = 4;
  int64 timestamp = 5;
  map<string, string> metadata = 6;
}

message RouteGroupMessageResponse {
  bool success = 1;
  string status = 2;  // "published", "failed"
  int64 sequence_number = 3;
  int32 member_count = 4;
  string error_message = 5;
}

message GetMessageStatusRequest {
  string msg_id = 1;
}

message GetMessageStatusResponse {
  string status = 1;  // "pending", "delivered", "read", "failed"
  int64 delivered_at = 2;
  int64 read_at = 3;
  repeated string delivered_devices = 4;
}
```

### gRPC Endpoints

#### 1. RoutePrivateMessage

**Purpose**: Route private message to recipient

**Request**:
```json
{
  "msg_id": "msg_user123_1706180400000_abc",
  "sender_id": "user_123",
  "recipient_id": "user_456",
  "content": "Hello!",
  "timestamp": 1706180400000,
  "metadata": {
    "content_type": "text/plain"
  }
}
```

**Response**:
```json
{
  "success": true,
  "status": "delivered",
  "sequence_number": 12345,
  "error_message": ""
}
```

**Status Values**:
- `delivered`: Message delivered to online user
- `offline`: User offline, message stored
- `failed`: Delivery failed

**Example** (Go):
```go
import (
    pb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"
    "google.golang.org/grpc"
)

conn, err := grpc.Dial("im-service:9094", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewIMServiceClient(conn)

req := &pb.RoutePrivateMessageRequest{
    MsgId:       "msg_user123_1706180400000_abc",
    SenderId:    "user_123",
    RecipientId: "user_456",
    Content:     "Hello!",
    Timestamp:   time.Now().UnixMilli(),
}

resp, err := client.RoutePrivateMessage(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s, Sequence: %d\n", resp.Status, resp.SequenceNumber)
```

#### 2. RouteGroupMessage

**Purpose**: Route group message to all members

**Request**:
```json
{
  "msg_id": "msg_user123_1706180400000_xyz",
  "sender_id": "user_123",
  "group_id": "group_789",
  "content": "Hello everyone!",
  "timestamp": 1706180400000,
  "metadata": {
    "content_type": "text/plain"
  }
}
```

**Response**:
```json
{
  "success": true,
  "status": "published",
  "sequence_number": 67890,
  "member_count": 150,
  "error_message": ""
}
```

**Status Values**:
- `published`: Message published to Kafka
- `failed`: Publish failed

#### 3. GetMessageStatus

**Purpose**: Get message delivery status

**Request**:
```json
{
  "msg_id": "msg_user123_1706180400000_abc"
}
```

**Response**:
```json
{
  "status": "delivered",
  "delivered_at": 1706180400100,
  "read_at": 1706180500000,
  "delivered_devices": ["device_abc", "device_xyz"]
}
```

**Status Values**:
- `pending`: Message in transit
- `delivered`: Message delivered
- `read`: Message read by recipient
- `failed`: Delivery failed

## REST API Endpoints

### 1. Get Offline Messages

**Endpoint**: `GET /api/v1/offline`

**Query Parameters**:
- `cursor`: Pagination cursor (last message ID)
- `limit`: Number of messages to return (default: 50, max: 100)

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
```

**Response**:
```json
{
  "messages": [
    {
      "msg_id": "msg_user456_1706180400000_xyz",
      "sender_id": "user_456",
      "recipient_id": "user_123",
      "content": "Hi there!",
      "sequence_number": 12346,
      "timestamp": 1706180400200,
      "created_at": 1706180400250
    }
  ],
  "next_cursor": "msg_user789_1706180500000_abc",
  "has_more": true,
  "total_count": 250
}
```

### 2. Get Offline Message Count

**Endpoint**: `GET /api/v1/offline/count`

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
```

**Response**:
```json
{
  "count": 250,
  "oldest_timestamp": 1706180400000,
  "newest_timestamp": 1706190000000
}
```

### 3. Mark Message as Read

**Endpoint**: `POST /api/v1/messages/read`

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

**Request Body**:
```json
{
  "msg_id": "msg_user456_1706180400000_xyz",
  "read_at": 1706180500000
}
```

**Response**:
```json
{
  "success": true,
  "message": "Read receipt recorded"
}
```

### 4. Delete Message (GDPR)

**Endpoint**: `DELETE /api/v1/messages/{msg_id}`

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
```

**Response**:
```json
{
  "success": true,
  "message": "Message deleted from all systems"
}
```

### 5. Export User Data (GDPR)

**Endpoint**: `GET /api/v1/export`

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
```

**Response**:
```json
{
  "user_id": "user_123",
  "export_date": "2026-01-25T10:00:00Z",
  "messages": [
    {
      "msg_id": "msg_user123_1706180400000_abc",
      "type": "sent",
      "recipient_id": "user_456",
      "content": "Hello!",
      "timestamp": 1706180400000
    }
  ],
  "total_messages": 1500
}
```

### 6. Health Check

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-25T10:00:00Z"
}
```

### 7. Readiness Check

**Endpoint**: `GET /ready`

**Response**:
```json
{
  "status": "ready",
  "dependencies": {
    "mysql": "healthy",
    "redis": "healthy",
    "kafka": "healthy",
    "etcd": "healthy",
    "gateway_service": "healthy",
    "user_service": "healthy"
  }
}
```

### 8. Metrics

**Endpoint**: `GET /metrics`

**Response**: Prometheus text format

**Key Metrics**:
```
# Messages routed
im_service_messages_routed_total{type="private"} 1234567
im_service_messages_routed_total{type="group"} 567890

# Message routing latency
im_service_routing_latency_bucket{le="50"} 1000000
im_service_routing_latency_bucket{le="100"} 1200000
im_service_routing_latency_bucket{le="200"} 1230000

# Offline messages stored
im_service_offline_messages_stored_total 123456

# Sequence numbers generated
im_service_sequence_numbers_generated_total 1234567
```

## Message Flow

### Private Message Flow

```
Client → Gateway → IM Service → Registry (lookup) → Gateway → Recipient
                              ↓
                         Offline Storage (if offline)
```

**Steps**:
1. Client sends message to Gateway
2. Gateway forwards to IM Service
3. IM Service assigns sequence number
4. IM Service applies sensitive word filter
5. IM Service encrypts content (if enabled)
6. IM Service looks up recipient in Registry
7. If online: IM Service calls Gateway to push message
8. If offline: IM Service publishes to Kafka `offline_msg` topic
9. Offline Worker consumes and stores in MySQL

### Group Message Flow

```
Client → Gateway → IM Service → Kafka (group_msg) → Gateway Nodes → Members
```

**Steps**:
1. Client sends message to Gateway
2. Gateway forwards to IM Service
3. IM Service assigns sequence number
4. IM Service applies sensitive word filter
5. IM Service encrypts content (if enabled)
6. IM Service publishes to Kafka `group_msg` topic
7. All Gateway nodes consume message
8. Each Gateway filters for locally-connected members
9. Gateway pushes to connected members
10. Offline members handled by Offline Worker

## Error Codes

### Message Routing Errors

| Code | Message | Description |
|------|---------|-------------|
| `INVALID_RECIPIENT` | Invalid recipient | Recipient user/group not found |
| `MESSAGE_TOO_LARGE` | Message too large | Message content > 10KB |
| `DUPLICATE_MESSAGE` | Duplicate message | Message with same msg_id exists |
| `ROUTING_FAILED` | Routing failed | Message routing failed |
| `SEQUENCE_ERROR` | Sequence error | Sequence number generation failed |

### Storage Errors

| Code | Message | Description |
|------|---------|-------------|
| `STORAGE_ERROR` | Storage error | Database write failed |
| `RETRIEVAL_ERROR` | Retrieval error | Database read failed |
| `QUOTA_EXCEEDED` | Quota exceeded | User offline message quota exceeded |

### Service Errors

| Code | Message | Description |
|------|---------|-------------|
| `SERVICE_UNAVAILABLE` | Service unavailable | Dependency service unavailable |
| `REGISTRY_ERROR` | Registry error | etcd registry error |
| `KAFKA_ERROR` | Kafka error | Kafka publish/consume error |
| `ENCRYPTION_ERROR` | Encryption error | Message encryption failed |

## Usage Examples

### Go Client

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"
    "google.golang.org/grpc"
)

func main() {
    // Connect to IM Service
    conn, err := grpc.Dial("im-service:9094", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewIMServiceClient(conn)

    // Send private message
    privateReq := &pb.RoutePrivateMessageRequest{
        MsgId:       "msg_user123_1706180400000_abc",
        SenderId:    "user_123",
        RecipientId: "user_456",
        Content:     "Hello!",
        Timestamp:   time.Now().UnixMilli(),
        Metadata: map[string]string{
            "content_type": "text/plain",
        },
    }

    privateResp, err := client.RoutePrivateMessage(context.Background(), privateReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Private message: status=%s, seq=%d\n", 
        privateResp.Status, privateResp.SequenceNumber)

    // Send group message
    groupReq := &pb.RouteGroupMessageRequest{
        MsgId:     "msg_user123_1706180400000_xyz",
        SenderId:  "user_123",
        GroupId:   "group_789",
        Content:   "Hello everyone!",
        Timestamp: time.Now().UnixMilli(),
    }

    groupResp, err := client.RouteGroupMessage(context.Background(), groupReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Group message: status=%s, members=%d\n", 
        groupResp.Status, groupResp.MemberCount)

    // Get message status
    statusReq := &pb.GetMessageStatusRequest{
        MsgId: "msg_user123_1706180400000_abc",
    }

    statusResp, err := client.GetMessageStatus(context.Background(), statusReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Message status: %s, delivered_at=%d\n", 
        statusResp.Status, statusResp.DeliveredAt)
}
```

### REST API Client (JavaScript)

```javascript
class IMServiceClient {
  constructor(baseURL, token) {
    this.baseURL = baseURL;
    this.token = token;
  }

  async getOfflineMessages(cursor = '', limit = 50) {
    const url = `${this.baseURL}/api/v1/offline?cursor=${cursor}&limit=${limit}`;
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.json();
  }

  async getOfflineMessageCount() {
    const url = `${this.baseURL}/api/v1/offline/count`;
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.json();
  }

  async markMessageAsRead(msgId, readAt = Date.now()) {
    const url = `${this.baseURL}/api/v1/messages/read`;
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        msg_id: msgId,
        read_at: readAt
      })
    });
    return response.json();
  }

  async deleteMessage(msgId) {
    const url = `${this.baseURL}/api/v1/messages/${msgId}`;
    const response = await fetch(url, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.json();
  }

  async exportUserData() {
    const url = `${this.baseURL}/api/v1/export`;
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });
    return response.json();
  }
}

// Usage
const client = new IMServiceClient('http://im-service:8094', 'your-jwt-token');

// Get offline messages
const messages = await client.getOfflineMessages('', 50);
console.log(`Received ${messages.messages.length} offline messages`);

// Mark message as read
await client.markMessageAsRead('msg_user456_1706180400000_xyz');

// Get offline message count
const count = await client.getOfflineMessageCount();
console.log(`Total offline messages: ${count.count}`);
```

## Best Practices

### Message Routing
1. **Idempotency**: Always use unique msg_id for deduplication
2. **Retry Logic**: Implement exponential backoff for retries
3. **Timeout Handling**: Set appropriate timeouts for gRPC calls
4. **Error Handling**: Handle all error codes gracefully

### Offline Messages
1. **Pagination**: Use cursor-based pagination for large result sets
2. **Batch Retrieval**: Fetch messages in batches (50-100)
3. **Lazy Loading**: Load messages on demand
4. **Cleanup**: Delete read messages to save storage

### Performance
1. **Connection Pooling**: Reuse gRPC connections
2. **Batch Operations**: Group multiple operations when possible
3. **Caching**: Cache frequently accessed data
4. **Monitoring**: Track P99 latency < 200ms

## References

- [Deployment Guide](./DEPLOYMENT.md)
- [Integration Testing Guide](./integration_test/README.md)
- [gRPC Documentation](https://grpc.io/docs/)
