# Auth Service - API Documentation

## Overview

The Auth Service provides JWT token validation and refresh functionality.

## gRPC Services

### Service Definition

**Proto File**: `api/v1/auth.proto`

```protobuf
syntax = "proto3";

package auth.v1;

service AuthService {
  // Validate JWT token
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  
  // Refresh JWT token
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
}

message ValidateTokenRequest {
  string token = 1;
}

message ValidateTokenResponse {
  bool valid = 1;
  string user_id = 2;
  string device_id = 3;
  int64 expires_at = 4;
  string error_message = 5;
}

message RefreshTokenRequest {
  string refresh_token = 1;
}

message RefreshTokenResponse {
  bool success = 1;
  string access_token = 2;
  string refresh_token = 3;
  int64 expires_at = 4;
  string error_message = 5;
}
```

### gRPC Endpoints

#### 1. ValidateToken

**Purpose**: Validate JWT token and extract claims

**Request**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response**:
```json
{
  "valid": true,
  "user_id": "user_123",
  "device_id": "device_abc",
  "expires_at": 1706266800000,
  "error_message": ""
}
```

**Example** (Go):
```go
import (
    pb "github.com/pingxin403/cuckoo/apps/auth-service/gen/authpb"
    "google.golang.org/grpc"
)

conn, err := grpc.Dial("auth-service:9095", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewAuthServiceClient(conn)

req := &pb.ValidateTokenRequest{
    Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
}

resp, err := client.ValidateToken(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

if resp.Valid {
    fmt.Printf("User: %s, Device: %s\n", resp.UserId, resp.DeviceId)
}
```

#### 2. RefreshToken

**Purpose**: Refresh expired access token

**Request**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response**:
```json
{
  "success": true,
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": 1706266800000,
  "error_message": ""
}
```

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
  "status": "ready"
}
```

### Metrics

**Endpoint**: `GET /metrics`

**Key Metrics**:
```
# Token validations
auth_service_token_validations_total{result="valid"} 1234567
auth_service_token_validations_total{result="invalid"} 123

# Token refreshes
auth_service_token_refreshes_total{result="success"} 12345
auth_service_token_refreshes_total{result="failed"} 12

# Validation latency
auth_service_validation_latency_bucket{le="10"} 1000000
auth_service_validation_latency_bucket{le="50"} 1234000
```

## Error Codes

| Code | Message | Description |
|------|---------|-------------|
| `INVALID_TOKEN` | Invalid token | Malformed JWT token |
| `TOKEN_EXPIRED` | Token expired | JWT token has expired |
| `INVALID_SIGNATURE` | Invalid signature | JWT signature verification failed |
| `DEVICE_LIMIT_EXCEEDED` | Device limit exceeded | User has > 5 devices |

## Usage Examples

### Go Client

```go
package main

import (
    "context"
    "log"

    pb "github.com/pingxin403/cuckoo/apps/auth-service/gen/authpb"
    "google.golang.org/grpc"
)

func main() {
    conn, err := grpc.Dial("auth-service:9095", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewAuthServiceClient(conn)

    // Validate token
    validateReq := &pb.ValidateTokenRequest{
        Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    }

    validateResp, err := client.ValidateToken(context.Background(), validateReq)
    if err != nil {
        log.Fatal(err)
    }

    if validateResp.Valid {
        log.Printf("Token valid for user: %s\n", validateResp.UserId)
    } else {
        log.Printf("Token invalid: %s\n", validateResp.ErrorMessage)
    }

    // Refresh token
    refreshReq := &pb.RefreshTokenRequest{
        RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    }

    refreshResp, err := client.RefreshToken(context.Background(), refreshReq)
    if err != nil {
        log.Fatal(err)
    }

    if refreshResp.Success {
        log.Printf("New access token: %s\n", refreshResp.AccessToken)
    }
}
```

## References

- [Deployment Guide](./DEPLOYMENT.md)
- [Testing Guide](./TESTING.md)
- [JWT RFC 7519](https://tools.ietf.org/html/rfc7519)
