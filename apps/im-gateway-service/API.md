# IM Gateway Service - API Documentation

## Overview

The IM Gateway Service provides WebSocket and gRPC APIs for real-time messaging. This document covers all available APIs, protocols, and usage examples.

## Table of Contents

1. [WebSocket Protocol](#websocket-protocol)
2. [gRPC Services](#grpc-services)
3. [REST API Endpoints](#rest-api-endpoints)
4. [Message Formats](#message-formats)
5. [Error Codes](#error-codes)
6. [Usage Examples](#usage-examples)

## WebSocket Protocol

### Connection

**Endpoint**: `ws://gateway.example.com:8080/ws` or `wss://gateway.example.com:8080/ws`

**Headers**:
```
Authorization: Bearer <JWT_TOKEN>
Upgrade: websocket
Connection: Upgrade
```

**Connection Flow**:
1. Client initiates WebSocket handshake with JWT token
2. Gateway validates token via Auth Service
3. Gateway extracts `user_id` and `device_id` from token
4. Gateway registers user in Registry (etcd)
5. Connection established

**Example** (JavaScript):
```javascript
const ws = new WebSocket('wss://gateway.example.com:8080/ws', {
  headers: {
    'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
  }
});

ws.onopen = () => {
  console.log('Connected to IM Gateway');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = (event) => {
  console.log('Connection closed:', event.code, event.reason);
};
```

### Message Types

#### 1. Authentication Response

**Direction**: Server → Client  
**Sent**: Immediately after connection established

**Format**:
```json
{
  "type": "auth_response",
  "success": true,
  "user_id": "user_12345",
  "device_id": "device_abc123",
  "timestamp": 1706180400000
}
```

**Fields**:
- `type`: Always "auth_response"
- `success`: Boolean indicating authentication success
- `user_id`: Authenticated user ID
- `device_id`: Device ID from JWT token
- `timestamp`: Server timestamp (milliseconds since epoch)

#### 2. Send Message

**Direction**: Client → Server  
**Purpose**: Send a private or group message

**Format**:
```json
{
  "type": "send_msg",
  "msg_id": "msg_user123_1706180400000_abc",
  "recipient_id": "user_67890",
  "recipient_type": "user",
  "content": "Hello, how are you?",
  "timestamp": 1706180400000
}
```

**Fields**:
- `type`: Always "send_msg"
- `msg_id`: Unique message ID (client-generated, used for deduplication)
- `recipient_id`: User ID (for private) or Group ID (for group)
- `recipient_type`: "user" or "group"
- `content`: Message content (max 10KB)
- `timestamp`: Client timestamp (milliseconds since epoch)

**Response** (ACK):
```json
{
  "type": "ack",
  "msg_id": "msg_user123_1706180400000_abc",
  "status": "delivered",
  "sequence_number": 12345,
  "timestamp": 1706180400100
}
```

#### 3. Receive Message

**Direction**: Server → Client  
**Purpose**: Deliver incoming message

**Format**:
```json
{
  "type": "message",
  "msg_id": "msg_user456_1706180400000_xyz",
  "sender_id": "user_456",
  "recipient_id": "user_123",
  "recipient_type": "user",
  "content": "Hi! I'm doing great, thanks!",
  "sequence_number": 12346,
  "timestamp": 1706180400200
}
```

**Fields**:
- `type`: Always "message"
- `msg_id`: Unique message ID
- `sender_id`: Sender user ID
- `recipient_id`: Recipient user ID or group ID
- `recipient_type`: "user" or "group"
- `content`: Message content
- `sequence_number`: Monotonically increasing sequence number
- `timestamp`: Server timestamp

**Client ACK**:
```json
{
  "type": "ack",
  "msg_id": "msg_user456_1706180400000_xyz",
  "status": "received",
  "timestamp": 1706180400250
}
```

#### 4. Heartbeat (Ping/Pong)

**Direction**: Bidirectional  
**Purpose**: Keep connection alive

**Client → Server (Ping)**:
```json
{
  "type": "heartbeat",
  "timestamp": 1706180400000
}
```

**Server → Client (Pong)**:
```json
{
  "type": "heartbeat_response",
  "timestamp": 1706180400050
}
```

**Interval**: Every 30 seconds  
**Timeout**: 90 seconds (connection closed if no response)

#### 5. Read Receipt

**Direction**: Server → Client  
**Purpose**: Notify sender that message was read

**Format**:
```json
{
  "type": "read_receipt",
  "msg_id": "msg_user123_1706180400000_abc",
  "reader_id": "user_67890",
  "read_at": 1706180500000,
  "timestamp": 1706180500050
}
```

**Fields**:
- `type`: Always "read_receipt"
- `msg_id`: Original message ID
- `reader_id`: User who read the message
- `read_at`: Timestamp when message was read
- `timestamp`: Server timestamp

#### 6. Group Membership Change

**Direction**: Server → Client  
**Purpose**: Notify group members of membership changes

**Format**:
```json
{
  "type": "membership_change",
  "group_id": "group_789",
  "change_type": "join",
  "user_id": "user_999",
  "timestamp": 1706180600000
}
```

**Fields**:
- `type`: Always "membership_change"
- `group_id`: Group ID
- `change_type`: "join" or "leave"
- `user_id`: User who joined/left
- `timestamp`: Server timestamp

#### 7. Error Message

**Direction**: Server → Client  
**Purpose**: Notify client of errors

**Format**:
```json
{
  "type": "error",
  "error_code": "INVALID_RECIPIENT",
  "error_message": "Recipient user_99999 not found",
  "msg_id": "msg_user123_1706180400000_abc",
  "timestamp": 1706180400100
}
```

**Fields**:
- `type`: Always "error"
- `error_code`: Error code (see [Error Codes](#error-codes))
- `error_message`: Human-readable error message
- `msg_id`: Related message ID (if applicable)
- `timestamp`: Server timestamp

### Connection Lifecycle

```
Client                          Gateway
  |                                |
  |--- WebSocket Handshake ------->|
  |    (Authorization: Bearer JWT) |
  |                                |
  |<--- auth_response -------------|
  |    (success: true)             |
  |                                |
  |--- send_msg ------------------>|
  |                                |
  |<--- ack (delivered) -----------|
  |                                |
  |<--- message -------------------|
  |    (incoming message)          |
  |                                |
  |--- ack (received) ------------>|
  |                                |
  |--- heartbeat ----------------->|
  |                                |
  |<--- heartbeat_response --------|
  |                                |
  |    ... (30s interval) ...      |
  |                                |
  |--- close -------------------->|
  |                                |
  |<--- close ---------------------|
```

## gRPC Services

### Service Definition

**Proto File**: `api/v1/im_gateway_service.proto`

```protobuf
syntax = "proto3";

package im.gateway.v1;

service IMGatewayService {
  // Push message to connected client
  rpc PushMessage(PushMessageRequest) returns (PushMessageResponse);
  
  // Get connection status
  rpc GetConnectionStatus(GetConnectionStatusRequest) returns (GetConnectionStatusResponse);
  
  // Close connection
  rpc CloseConnection(CloseConnectionRequest) returns (CloseConnectionResponse);
}

message PushMessageRequest {
  string user_id = 1;
  string device_id = 2;  // Optional, if empty push to all devices
  string msg_id = 3;
  string sender_id = 4;
  string content = 5;
  int64 sequence_number = 6;
  int64 timestamp = 7;
}

message PushMessageResponse {
  bool success = 1;
  string status = 2;  // "delivered", "offline", "failed"
  string error_message = 3;
}

message GetConnectionStatusRequest {
  string user_id = 1;
  string device_id = 2;  // Optional
}

message GetConnectionStatusResponse {
  bool online = 1;
  repeated DeviceStatus devices = 2;
}

message DeviceStatus {
  string device_id = 1;
  bool online = 2;
  string gateway_node = 3;
  int64 connected_at = 4;
}

message CloseConnectionRequest {
  string user_id = 1;
  string device_id = 2;  // Optional, if empty close all devices
  string reason = 3;
}

message CloseConnectionResponse {
  bool success = 1;
  int32 closed_count = 2;
}
```

### gRPC Endpoints

#### 1. PushMessage

**Purpose**: Push message to connected client (called by IM Service)

**Request**:
```json
{
  "user_id": "user_123",
  "device_id": "device_abc",
  "msg_id": "msg_456_1706180400000_xyz",
  "sender_id": "user_456",
  "content": "Hello from IM Service",
  "sequence_number": 12345,
  "timestamp": 1706180400000
}
```

**Response**:
```json
{
  "success": true,
  "status": "delivered",
  "error_message": ""
}
```

**Status Values**:
- `delivered`: Message delivered to WebSocket connection
- `offline`: User offline, message queued
- `failed`: Delivery failed

**Example** (Go):
```go
import (
    pb "github.com/pingxin403/cuckoo/apps/im-gateway-service/gen/im_gateway_servicepb"
    "google.golang.org/grpc"
)

conn, err := grpc.Dial("gateway:9093", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewIMGatewayServiceClient(conn)

req := &pb.PushMessageRequest{
    UserId:         "user_123",
    DeviceId:       "device_abc",
    MsgId:          "msg_456_1706180400000_xyz",
    SenderId:       "user_456",
    Content:        "Hello from IM Service",
    SequenceNumber: 12345,
    Timestamp:      time.Now().UnixMilli(),
}

resp, err := client.PushMessage(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", resp.Status)
```

#### 2. GetConnectionStatus

**Purpose**: Check if user is online

**Request**:
```json
{
  "user_id": "user_123",
  "device_id": ""
}
```

**Response**:
```json
{
  "online": true,
  "devices": [
    {
      "device_id": "device_abc",
      "online": true,
      "gateway_node": "gateway-1",
      "connected_at": 1706180000000
    },
    {
      "device_id": "device_xyz",
      "online": true,
      "gateway_node": "gateway-2",
      "connected_at": 1706180100000
    }
  ]
}
```

#### 3. CloseConnection

**Purpose**: Force close user connection (admin operation)

**Request**:
```json
{
  "user_id": "user_123",
  "device_id": "device_abc",
  "reason": "Admin requested disconnect"
}
```

**Response**:
```json
{
  "success": true,
  "closed_count": 1
}
```

## REST API Endpoints

### Health Check

**Endpoint**: `GET /health`  
**Purpose**: Liveness probe

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2026-01-25T10:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Service is healthy
- `503 Service Unavailable`: Service is unhealthy

### Readiness Check

**Endpoint**: `GET /ready`  
**Purpose**: Readiness probe

**Response**:
```json
{
  "status": "ready",
  "dependencies": {
    "etcd": "healthy",
    "redis": "healthy",
    "kafka": "healthy",
    "auth_service": "healthy",
    "user_service": "healthy",
    "im_service": "healthy"
  }
}
```

**Status Codes**:
- `200 OK`: Service is ready
- `503 Service Unavailable`: Service is not ready

### Metrics

**Endpoint**: `GET /metrics`  
**Purpose**: Prometheus metrics

**Response**: Prometheus text format

**Key Metrics**:
```
# Active connections
im_gateway_active_connections 85234

# Messages sent
im_gateway_messages_sent_total 1234567

# Messages delivered
im_gateway_messages_delivered_total 1234500

# Message latency histogram
im_gateway_message_latency_bucket{le="50"} 1000000
im_gateway_message_latency_bucket{le="100"} 1200000
im_gateway_message_latency_bucket{le="200"} 1230000
im_gateway_message_latency_bucket{le="+Inf"} 1234500

# Connection errors
im_gateway_connection_errors_total 67
```

### Statistics

**Endpoint**: `GET /stats`  
**Purpose**: Connection statistics

**Response**:
```json
{
  "active_connections": 85234,
  "total_messages_sent": 1234567,
  "total_messages_delivered": 1234500,
  "uptime_seconds": 86400,
  "memory_usage_bytes": 6871947673,
  "goroutines": 85500,
  "cache_stats": {
    "user_mapping_size": 85234,
    "group_membership_size": 1234,
    "hit_rate": 0.95
  }
}
```

## Message Formats

### Private Message

**Client → Server**:
```json
{
  "type": "send_msg",
  "msg_id": "msg_user123_1706180400000_abc",
  "recipient_id": "user_456",
  "recipient_type": "user",
  "content": "Hello!",
  "timestamp": 1706180400000
}
```

**Server → Recipient**:
```json
{
  "type": "message",
  "msg_id": "msg_user123_1706180400000_abc",
  "sender_id": "user_123",
  "recipient_id": "user_456",
  "recipient_type": "user",
  "content": "Hello!",
  "sequence_number": 12345,
  "timestamp": 1706180400100
}
```

### Group Message

**Client → Server**:
```json
{
  "type": "send_msg",
  "msg_id": "msg_user123_1706180400000_xyz",
  "recipient_id": "group_789",
  "recipient_type": "group",
  "content": "Hello everyone!",
  "timestamp": 1706180400000
}
```

**Server → Group Members**:
```json
{
  "type": "message",
  "msg_id": "msg_user123_1706180400000_xyz",
  "sender_id": "user_123",
  "recipient_id": "group_789",
  "recipient_type": "group",
  "content": "Hello everyone!",
  "sequence_number": 67890,
  "timestamp": 1706180400100
}
```

### Message with Metadata

**Extended Format**:
```json
{
  "type": "message",
  "msg_id": "msg_user123_1706180400000_abc",
  "sender_id": "user_123",
  "recipient_id": "user_456",
  "recipient_type": "user",
  "content": "Check out this link!",
  "sequence_number": 12345,
  "timestamp": 1706180400100,
  "metadata": {
    "content_type": "text/plain",
    "reply_to": "msg_user456_1706180300000_xyz",
    "mentions": ["user_789"],
    "attachments": [
      {
        "type": "image",
        "url": "https://cdn.example.com/image123.jpg",
        "size": 102400,
        "mime_type": "image/jpeg"
      }
    ]
  }
}
```

## Error Codes

### Authentication Errors

| Code | Message | Description |
|------|---------|-------------|
| `AUTH_FAILED` | Authentication failed | Invalid or expired JWT token |
| `AUTH_REQUIRED` | Authentication required | No JWT token provided |
| `INVALID_TOKEN` | Invalid token format | Malformed JWT token |
| `TOKEN_EXPIRED` | Token expired | JWT token has expired |
| `DEVICE_LIMIT_EXCEEDED` | Device limit exceeded | User has > 5 devices connected |

### Message Errors

| Code | Message | Description |
|------|---------|-------------|
| `INVALID_RECIPIENT` | Invalid recipient | Recipient user/group not found |
| `MESSAGE_TOO_LARGE` | Message too large | Message content > 10KB |
| `INVALID_MESSAGE_FORMAT` | Invalid message format | Malformed JSON message |
| `DUPLICATE_MESSAGE` | Duplicate message | Message with same msg_id already processed |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded | Too many messages sent |

### Connection Errors

| Code | Message | Description |
|------|---------|-------------|
| `CONNECTION_CLOSED` | Connection closed | WebSocket connection closed |
| `HEARTBEAT_TIMEOUT` | Heartbeat timeout | No heartbeat received for 90s |
| `MAX_CONNECTIONS_EXCEEDED` | Max connections exceeded | Gateway at capacity |
| `INTERNAL_ERROR` | Internal server error | Unexpected server error |

### Service Errors

| Code | Message | Description |
|------|---------|-------------|
| `SERVICE_UNAVAILABLE` | Service unavailable | Dependency service unavailable |
| `REGISTRY_ERROR` | Registry error | etcd registry error |
| `CACHE_ERROR` | Cache error | Redis cache error |
| `ROUTING_ERROR` | Routing error | Message routing failed |

## Usage Examples

### JavaScript/TypeScript Client

```typescript
class IMClient {
  private ws: WebSocket;
  private token: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(gatewayUrl: string, token: string) {
    this.token = token;
    this.connect(gatewayUrl);
  }

  private connect(url: string) {
    this.ws = new WebSocket(url, {
      headers: {
        'Authorization': `Bearer ${this.token}`
      }
    });

    this.ws.onopen = () => {
      console.log('Connected to IM Gateway');
      this.reconnectAttempts = 0;
      this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = (event) => {
      console.log('Connection closed:', event.code, event.reason);
      this.stopHeartbeat();
      this.reconnect(url);
    };
  }

  private handleMessage(message: any) {
    switch (message.type) {
      case 'auth_response':
        console.log('Authenticated:', message.user_id);
        break;
      case 'message':
        console.log('Received message:', message);
        this.sendAck(message.msg_id);
        break;
      case 'ack':
        console.log('Message delivered:', message.msg_id);
        break;
      case 'read_receipt':
        console.log('Message read:', message.msg_id);
        break;
      case 'error':
        console.error('Error:', message.error_code, message.error_message);
        break;
    }
  }

  sendMessage(recipientId: string, content: string, recipientType: 'user' | 'group' = 'user') {
    const message = {
      type: 'send_msg',
      msg_id: `msg_${Date.now()}_${Math.random()}`,
      recipient_id: recipientId,
      recipient_type: recipientType,
      content: content,
      timestamp: Date.now()
    };
    this.ws.send(JSON.stringify(message));
  }

  private sendAck(msgId: string) {
    const ack = {
      type: 'ack',
      msg_id: msgId,
      status: 'received',
      timestamp: Date.now()
    };
    this.ws.send(JSON.stringify(ack));
  }

  private heartbeatInterval: any;

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      const heartbeat = {
        type: 'heartbeat',
        timestamp: Date.now()
      };
      this.ws.send(JSON.stringify(heartbeat));
    }, 30000); // 30 seconds
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }
  }

  private reconnect(url: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
      console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
      setTimeout(() => this.connect(url), delay);
    } else {
      console.error('Max reconnect attempts reached');
    }
  }

  close() {
    this.stopHeartbeat();
    this.ws.close();
  }
}

// Usage
const client = new IMClient('wss://gateway.example.com:8080/ws', 'your-jwt-token');

// Send private message
client.sendMessage('user_456', 'Hello!');

// Send group message
client.sendMessage('group_789', 'Hello everyone!', 'group');
```

### Go Client

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/gorilla/websocket"
)

type IMClient struct {
    conn  *websocket.Conn
    token string
}

type Message struct {
    Type          string `json:"type"`
    MsgID         string `json:"msg_id,omitempty"`
    RecipientID   string `json:"recipient_id,omitempty"`
    RecipientType string `json:"recipient_type,omitempty"`
    Content       string `json:"content,omitempty"`
    Timestamp     int64  `json:"timestamp,omitempty"`
}

func NewIMClient(gatewayURL, token string) (*IMClient, error) {
    header := make(map[string][]string)
    header["Authorization"] = []string{fmt.Sprintf("Bearer %s", token)}

    conn, _, err := websocket.DefaultDialer.Dial(gatewayURL, header)
    if err != nil {
        return nil, err
    }

    client := &IMClient{
        conn:  conn,
        token: token,
    }

    go client.readMessages()
    go client.heartbeat()

    return client, nil
}

func (c *IMClient) readMessages() {
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            return
        }

        var msg Message
        if err := json.Unmarshal(message, &msg); err != nil {
            log.Println("Unmarshal error:", err)
            continue
        }

        c.handleMessage(msg)
    }
}

func (c *IMClient) handleMessage(msg Message) {
    switch msg.Type {
    case "auth_response":
        log.Println("Authenticated")
    case "message":
        log.Printf("Received message: %s\n", msg.Content)
        c.sendAck(msg.MsgID)
    case "ack":
        log.Printf("Message delivered: %s\n", msg.MsgID)
    case "error":
        log.Printf("Error: %+v\n", msg)
    }
}

func (c *IMClient) SendMessage(recipientID, content, recipientType string) error {
    msg := Message{
        Type:          "send_msg",
        MsgID:         fmt.Sprintf("msg_%d_%d", time.Now().UnixMilli(), time.Now().Nanosecond()),
        RecipientID:   recipientID,
        RecipientType: recipientType,
        Content:       content,
        Timestamp:     time.Now().UnixMilli(),
    }

    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *IMClient) sendAck(msgID string) error {
    ack := Message{
        Type:      "ack",
        MsgID:     msgID,
        Timestamp: time.Now().UnixMilli(),
    }

    data, err := json.Marshal(ack)
    if err != nil {
        return err
    }

    return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *IMClient) heartbeat() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        hb := Message{
            Type:      "heartbeat",
            Timestamp: time.Now().UnixMilli(),
        }

        data, err := json.Marshal(hb)
        if err != nil {
            log.Println("Heartbeat marshal error:", err)
            continue
        }

        if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
            log.Println("Heartbeat send error:", err)
            return
        }
    }
}

func (c *IMClient) Close() error {
    return c.conn.Close()
}

func main() {
    client, err := NewIMClient("wss://gateway.example.com:8080/ws", "your-jwt-token")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Send private message
    if err := client.SendMessage("user_456", "Hello!", "user"); err != nil {
        log.Fatal(err)
    }

    // Send group message
    if err := client.SendMessage("group_789", "Hello everyone!", "group"); err != nil {
        log.Fatal(err)
    }

    // Keep running
    select {}
}
```

## Rate Limiting

### Per-User Limits
- **Messages**: 100 messages/minute
- **Connections**: 5 devices per user
- **Heartbeats**: 1 per 30 seconds (enforced by server)

### Enforcement
- Rate limit exceeded returns `RATE_LIMIT_EXCEEDED` error
- Connection closed after repeated violations
- Exponential backoff recommended for retries

## Best Practices

### Client Implementation
1. **Implement Reconnection**: Use exponential backoff (1s, 2s, 4s, 8s, max 30s)
2. **Handle Heartbeats**: Send heartbeat every 30 seconds
3. **Implement ACKs**: Always send ACK for received messages
4. **Deduplicate Messages**: Track received msg_ids to avoid duplicates
5. **Handle Errors**: Implement proper error handling for all error codes
6. **Use TLS**: Always use `wss://` in production
7. **Token Refresh**: Refresh JWT token before expiration

### Message Design
1. **Keep Messages Small**: Max 10KB per message
2. **Use Unique msg_id**: Include timestamp and random component
3. **Include Metadata**: Use metadata field for rich content
4. **Compress Large Content**: Use gzip for large text content
5. **Batch Operations**: Group multiple operations when possible

### Performance
1. **Connection Pooling**: Reuse connections when possible
2. **Lazy Loading**: Load offline messages on demand
3. **Local Caching**: Cache frequently accessed data
4. **Optimize Payloads**: Minimize JSON payload size
5. **Monitor Latency**: Track P99 latency < 200ms

## OpenAPI Specification

See [openapi.yaml](./openapi.yaml) for complete OpenAPI 3.0 specification of REST endpoints.

## References

- [WebSocket Protocol (RFC 6455)](https://tools.ietf.org/html/rfc6455)
- [gRPC Documentation](https://grpc.io/docs/)
- [JWT (RFC 7519)](https://tools.ietf.org/html/rfc7519)
- [Prometheus Metrics](https://prometheus.io/docs/concepts/metric_types/)
