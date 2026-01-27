# User Service - API Documentation

## Overview

The User Service provides user profile and group membership management APIs.

## gRPC Services

### Service Definition

**Proto File**: `api/v1/user.proto`

```protobuf
syntax = "proto3";

package user.v1;

service UserService {
  // Get user profile
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  
  // Batch get users
  rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);
  
  // Get group members
  rpc GetGroupMembers(GetGroupMembersRequest) returns (GetGroupMembersResponse);
  
  // Validate group membership
  rpc ValidateGroupMembership(ValidateGroupMembershipRequest) returns (ValidateGroupMembershipResponse);
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  User user = 1;
  string error_message = 2;
}

message User {
  string user_id = 1;
  string username = 2;
  string email = 3;
  string avatar_url = 4;
  int64 created_at = 5;
}

message BatchGetUsersRequest {
  repeated string user_ids = 1;
}

message BatchGetUsersResponse {
  repeated User users = 1;
  repeated string not_found = 2;
}

message GetGroupMembersRequest {
  string group_id = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message GetGroupMembersResponse {
  repeated string member_ids = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}

message ValidateGroupMembershipRequest {
  string group_id = 1;
  string user_id = 2;
}

message ValidateGroupMembershipResponse {
  bool is_member = 1;
  string role = 2;  // "owner", "admin", "member"
}
```

### gRPC Endpoints

#### 1. GetUser

**Purpose**: Get user profile by user ID

**Request**:
```json
{
  "user_id": "user_123"
}
```

**Response**:
```json
{
  "user": {
    "user_id": "user_123",
    "username": "john_doe",
    "email": "john@example.com",
    "avatar_url": "https://cdn.example.com/avatar123.jpg",
    "created_at": 1706180400000
  },
  "error_message": ""
}
```

**Example** (Go):
```go
import (
    pb "github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
    "google.golang.org/grpc"
)

conn, err := grpc.Dial("user-service:9096", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewUserServiceClient(conn)

req := &pb.GetUserRequest{
    UserId: "user_123",
}

resp, err := client.GetUser(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("User: %s (%s)\n", resp.User.Username, resp.User.Email)
```

#### 2. BatchGetUsers

**Purpose**: Get multiple user profiles in one request

**Request**:
```json
{
  "user_ids": ["user_123", "user_456", "user_789"]
}
```

**Response**:
```json
{
  "users": [
    {
      "user_id": "user_123",
      "username": "john_doe",
      "email": "john@example.com"
    },
    {
      "user_id": "user_456",
      "username": "jane_smith",
      "email": "jane@example.com"
    }
  ],
  "not_found": ["user_789"]
}
```

#### 3. GetGroupMembers

**Purpose**: Get group members with pagination

**Request**:
```json
{
  "group_id": "group_789",
  "page_size": 100,
  "page_token": ""
}
```

**Response**:
```json
{
  "member_ids": ["user_123", "user_456", "user_789"],
  "next_page_token": "token_abc123",
  "total_count": 1500
}
```

#### 4. ValidateGroupMembership

**Purpose**: Check if user is member of group

**Request**:
```json
{
  "group_id": "group_789",
  "user_id": "user_123"
}
```

**Response**:
```json
{
  "is_member": true,
  "role": "member"
}
```

**Role Values**:
- `owner`: Group owner
- `admin`: Group administrator
- `member`: Regular member

## REST API Endpoints

### Health Check

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy"
}
```

### Readiness Check

**Endpoint**: `GET /ready`

**Response**:
```json
{
  "status": "ready",
  "dependencies": {
    "mysql": "healthy",
    "redis": "healthy"
  }
}
```

### Metrics

**Endpoint**: `GET /metrics`

**Key Metrics**:
```
# User lookups
user_service_user_lookups_total{result="success"} 1234567
user_service_user_lookups_total{result="not_found"} 123

# Group member lookups
user_service_group_member_lookups_total 567890

# Cache hit rate
user_service_cache_hits_total{type="user"} 1000000
user_service_cache_misses_total{type="user"} 50000
```

## Error Codes

| Code | Message | Description |
|------|---------|-------------|
| `USER_NOT_FOUND` | User not found | User ID does not exist |
| `GROUP_NOT_FOUND` | Group not found | Group ID does not exist |
| `INVALID_PAGE_SIZE` | Invalid page size | Page size > max limit |
| `DATABASE_ERROR` | Database error | Database query failed |

## Usage Examples

### Go Client

```go
package main

import (
    "context"
    "log"

    pb "github.com/pingxin403/cuckoo/apps/user-service/gen/userpb"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("user-service:9096", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Get user
    userReq := &pb.GetUserRequest{
        UserId: "user_123",
    }

    userResp, err := client.GetUser(context.Background(), userReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User: %s\n", userResp.User.Username)

    // Batch get users
    batchReq := &pb.BatchGetUsersRequest{
        UserIds: []string{"user_123", "user_456", "user_789"},
    }

    batchResp, err := client.BatchGetUsers(context.Background(), batchReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d users\n", len(batchResp.Users))

    // Get group members
    membersReq := &pb.GetGroupMembersRequest{
        GroupId:   "group_789",
        PageSize:  100,
        PageToken: "",
    }

    membersResp, err := client.GetGroupMembers(context.Background(), membersReq)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Group has %d members\n", membersResp.TotalCount)

    // Validate membership
    validateReq := &pb.ValidateGroupMembershipRequest{
        GroupId: "group_789",
        UserId:  "user_123",
    }

    validateResp, err := client.ValidateGroupMembership(context.Background(), validateReq)
    if err != nil {
        log.Fatal(err)
    }

    if validateResp.IsMember {
        log.Printf("User is a %s\n", validateResp.Role)
    }
}
```

## Best Practices

### User Lookups
1. **Batch Operations**: Use BatchGetUsers for multiple users
2. **Caching**: Cache frequently accessed user profiles
3. **Error Handling**: Handle USER_NOT_FOUND gracefully

### Group Members
1. **Pagination**: Use pagination for large groups (>1000 members)
2. **Page Size**: Use appropriate page size (100-1000)
3. **Caching**: Cache group membership for small groups

### Performance
1. **Connection Pooling**: Reuse gRPC connections
2. **Timeout**: Set appropriate timeouts (3-5s)
3. **Monitoring**: Track cache hit rates

## References

- [Deployment Guide](./DEPLOYMENT.md)
- [Testing Guide](./TESTING.md)
- [gRPC Documentation](https://grpc.io/docs/)
