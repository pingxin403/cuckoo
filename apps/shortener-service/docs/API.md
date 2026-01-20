# URL Shortener Service API Documentation

This document provides comprehensive API documentation for the URL Shortener Service, including gRPC and HTTP endpoints, request/response formats, error codes, and usage examples.

## Table of Contents

- [Overview](#overview)
- [gRPC API](#grpc-api)
  - [CreateShortLink](#createshortlink)
  - [GetLinkInfo](#getlinkinfo)
  - [DeleteShortLink](#deleteshortlink)
- [HTTP API](#http-api)
  - [Redirect Endpoint](#redirect-endpoint)
  - [Health Check Endpoints](#health-check-endpoints)
- [Error Codes](#error-codes)
- [Usage Examples](#usage-examples)
- [Rate Limiting](#rate-limiting)
- [Best Practices](#best-practices)

## Overview

The URL Shortener Service provides two APIs:

1. **gRPC API** (Port 9092): For creating, retrieving, and managing short links
2. **HTTP API** (Port 8080): For redirecting short codes to long URLs

### Base URLs

- **gRPC**: `localhost:9092` (development) / `shortener-service:9092` (Kubernetes)
- **HTTP**: `http://localhost:8080` (development) / `http://shortener-service:8080` (Kubernetes)

### Authentication

Currently, the service does not require authentication. In production, consider adding:
- API keys for gRPC endpoints
- Rate limiting per IP/user
- OAuth2 for administrative operations

## gRPC API

The gRPC API is defined in [`api/v1/shortener.proto`](../../../api/v1/shortener.proto).

### Service Definition

```protobuf
service ShortenerService {
  rpc CreateShortLink(CreateShortLinkRequest) returns (CreateShortLinkResponse);
  rpc GetLinkInfo(GetLinkInfoRequest) returns (GetLinkInfoResponse);
  rpc DeleteShortLink(DeleteShortLinkRequest) returns (DeleteShortLinkResponse);
}
```

---

### CreateShortLink

Creates a new short link for a given long URL.

#### Request

```protobuf
message CreateShortLinkRequest {
  string long_url = 1;      // Required: URL to shorten (max 2048 chars)
  string custom_code = 2;   // Optional: Custom short code (4-20 chars)
  int64 expires_at = 3;     // Optional: Unix timestamp for expiration
}
```

**Field Descriptions:**

- `long_url` (required): The long URL to shorten
  - Must be a valid HTTP/HTTPS URL
  - Maximum length: 2048 characters
  - Must not contain malicious patterns (javascript:, data:, etc.)

- `custom_code` (optional): User-defined short code
  - Length: 4-20 characters
  - Allowed characters: alphanumeric (a-z, A-Z, 0-9) and hyphen (-)
  - Must not be a reserved keyword (api, admin, health, ready, metrics)
  - Must be unique (returns error if already exists)

- `expires_at` (optional): Expiration timestamp
  - Unix timestamp in seconds
  - If not provided, link never expires
  - Must be in the future

#### Response

```protobuf
message CreateShortLinkResponse {
  string short_code = 1;    // Generated or custom short code
  string short_url = 2;     // Full short URL
  int64 created_at = 3;     // Creation timestamp (Unix seconds)
  int64 expires_at = 4;     // Expiration timestamp (0 if no expiration)
}
```

**Field Descriptions:**

- `short_code`: The 7-character Base62 code (or custom code if provided)
- `short_url`: Complete short URL (e.g., `http://localhost:8080/abc1234`)
- `created_at`: When the link was created (Unix timestamp)
- `expires_at`: When the link expires (0 if never expires)

#### Examples

**Basic Usage:**

```bash
grpcurl -plaintext -d '{
  "long_url": "https://example.com/very/long/url/path"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink
```

**Response:**
```json
{
  "short_code": "abc1234",
  "short_url": "http://localhost:8080/abc1234",
  "created_at": 1737360000,
  "expires_at": 0
}
```

**With Custom Code:**

```bash
grpcurl -plaintext -d '{
  "long_url": "https://example.com/page",
  "custom_code": "my-link"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink
```

**Response:**
```json
{
  "short_code": "my-link",
  "short_url": "http://localhost:8080/my-link",
  "created_at": 1737360000,
  "expires_at": 0
}
```

**With Expiration (7 days):**

```bash
# Calculate expiration: current_time + 7 days
EXPIRES_AT=$(date -v+7d +%s)

grpcurl -plaintext -d "{
  \"long_url\": \"https://example.com/temp\",
  \"expires_at\": $EXPIRES_AT
}" localhost:9092 api.v1.ShortenerService/CreateShortLink
```

**Response:**
```json
{
  "short_code": "xyz9876",
  "short_url": "http://localhost:8080/xyz9876",
  "created_at": 1737360000,
  "expires_at": 1737964800
}
```

#### Error Cases

| Error | Status Code | Description |
|-------|-------------|-------------|
| INVALID_URL | InvalidArgument | URL is invalid, too long, or malicious |
| INVALID_CUSTOM_CODE | InvalidArgument | Custom code format is invalid |
| CUSTOM_CODE_RESERVED | InvalidArgument | Custom code is a reserved keyword |
| SHORT_CODE_EXISTS | AlreadyExists | Custom code already in use |
| STORAGE_ERROR | Internal | Database operation failed |

---

### GetLinkInfo

Retrieves information about a short link, including metadata and statistics.

#### Request

```protobuf
message GetLinkInfoRequest {
  string short_code = 1;    // Required: Short code to look up
}
```

**Field Descriptions:**

- `short_code` (required): The short code to retrieve information for

#### Response

```protobuf
message GetLinkInfoResponse {
  string short_code = 1;    // Short code
  string long_url = 2;      // Original long URL
  int64 created_at = 3;     // Creation timestamp
  int64 expires_at = 4;     // Expiration timestamp (0 if no expiration)
  int32 click_count = 5;    // Number of redirects (future feature)
  bool is_deleted = 6;      // Whether the link has been deleted
}
```

**Field Descriptions:**

- `short_code`: The short code
- `long_url`: The original long URL
- `created_at`: When the link was created
- `expires_at`: When the link expires (0 if never)
- `click_count`: Number of times the link has been accessed (currently always 0)
- `is_deleted`: Whether the link has been soft-deleted

#### Examples

**Basic Usage:**

```bash
grpcurl -plaintext -d '{
  "short_code": "abc1234"
}' localhost:9092 api.v1.ShortenerService/GetLinkInfo
```

**Response:**
```json
{
  "short_code": "abc1234",
  "long_url": "https://example.com/very/long/url/path",
  "created_at": 1737360000,
  "expires_at": 0,
  "click_count": 0,
  "is_deleted": false
}
```

#### Error Cases

| Error | Status Code | Description |
|-------|-------------|-------------|
| SHORT_CODE_NOT_FOUND | NotFound | Short code does not exist |
| STORAGE_ERROR | Internal | Database operation failed |

---

### DeleteShortLink

Soft deletes a short link. The link will no longer redirect, but the data is retained in the database.

#### Request

```protobuf
message DeleteShortLinkRequest {
  string short_code = 1;    // Required: Short code to delete
}
```

**Field Descriptions:**

- `short_code` (required): The short code to delete

#### Response

```protobuf
message DeleteShortLinkResponse {
  bool success = 1;         // Whether the deletion was successful
}
```

**Field Descriptions:**

- `success`: Always `true` if the operation succeeds (otherwise returns error)

#### Examples

**Basic Usage:**

```bash
grpcurl -plaintext -d '{
  "short_code": "abc1234"
}' localhost:9092 api.v1.ShortenerService/DeleteShortLink
```

**Response:**
```json
{
  "success": true
}
```

#### Error Cases

| Error | Status Code | Description |
|-------|-------------|-------------|
| SHORT_CODE_NOT_FOUND | NotFound | Short code does not exist |
| STORAGE_ERROR | Internal | Database operation failed |

---

## HTTP API

The HTTP API provides redirect functionality and health checks.

### Redirect Endpoint

Redirects a short code to its corresponding long URL.

#### Request

```
GET /:code
```

**Parameters:**

- `code` (path parameter): The short code to redirect

#### Response

**Success (302 Found):**

```http
HTTP/1.1 302 Found
Location: https://example.com/very/long/url/path
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
```

**Not Found (404):**

```http
HTTP/1.1 404 Not Found
Content-Type: text/plain

Short code not found
```

**Expired/Deleted (410 Gone):**

```http
HTTP/1.1 410 Gone
Content-Type: text/plain

Short link has expired or been deleted
```

#### Examples

**Using curl (follow redirect):**

```bash
curl -L http://localhost:8080/abc1234
```

**Using curl (see redirect without following):**

```bash
curl -I http://localhost:8080/abc1234
```

**Response:**
```http
HTTP/1.1 302 Found
Location: https://example.com/very/long/url/path
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Date: Mon, 20 Jan 2025 12:00:00 GMT
Content-Length: 0
```

**Using browser:**

Simply navigate to `http://localhost:8080/abc1234` and you'll be redirected automatically.

#### Security Headers

All redirect responses include security headers:

- `X-Content-Type-Options: nosniff`: Prevents MIME type sniffing
- `X-Frame-Options: DENY`: Prevents clickjacking attacks

---

### Health Check Endpoints

#### Liveness Probe

Checks if the service is alive.

```
GET /health
```

**Response:**

```http
HTTP/1.1 200 OK
Content-Type: text/plain

OK
```

**Example:**

```bash
curl http://localhost:8080/health
```

#### Readiness Probe

Checks if the service is ready to accept traffic (verifies MySQL and Redis connectivity).

```
GET /ready
```

**Response (Ready):**

```http
HTTP/1.1 200 OK
Content-Type: text/plain

Ready
```

**Response (Not Ready):**

```http
HTTP/1.1 503 Service Unavailable
Content-Type: text/plain

Not ready: MySQL connection failed
```

**Example:**

```bash
curl http://localhost:8080/ready
```

---

## Error Codes

The service uses standard gRPC status codes with custom error codes in the error message.

### gRPC Error Codes

| Error Code | gRPC Status | HTTP Equivalent | Description |
|------------|-------------|-----------------|-------------|
| INVALID_URL | InvalidArgument | 400 | URL is invalid, too long, or contains malicious patterns |
| URL_TOO_LONG | InvalidArgument | 400 | URL exceeds 2048 characters |
| INVALID_CUSTOM_CODE | InvalidArgument | 400 | Custom code format is invalid (length or characters) |
| CUSTOM_CODE_RESERVED | InvalidArgument | 400 | Custom code is a reserved keyword |
| SHORT_CODE_NOT_FOUND | NotFound | 404 | Short code does not exist in the database |
| SHORT_CODE_EXISTS | AlreadyExists | 409 | Custom code is already in use |
| SHORT_CODE_EXPIRED | FailedPrecondition | 410 | Short link has expired |
| SHORT_CODE_DELETED | FailedPrecondition | 410 | Short link has been deleted |
| STORAGE_ERROR | Internal | 500 | Database operation failed |
| GENERATION_FAILED | Internal | 500 | Failed to generate unique short code |

### HTTP Status Codes

| Status Code | Description |
|-------------|-------------|
| 200 OK | Health check successful |
| 302 Found | Successful redirect to long URL |
| 404 Not Found | Short code does not exist |
| 410 Gone | Short link has expired or been deleted |
| 503 Service Unavailable | Service is not ready (readiness check failed) |

---

## Usage Examples

### Complete Workflow

```bash
# 1. Create a short link
RESPONSE=$(grpcurl -plaintext -d '{
  "long_url": "https://example.com/my/long/url"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink)

# Extract short code from response
SHORT_CODE=$(echo $RESPONSE | jq -r '.short_code')
echo "Created short code: $SHORT_CODE"

# 2. Get link information
grpcurl -plaintext -d "{
  \"short_code\": \"$SHORT_CODE\"
}" localhost:9092 api.v1.ShortenerService/GetLinkInfo

# 3. Test redirect
curl -I http://localhost:8080/$SHORT_CODE

# 4. Delete the link
grpcurl -plaintext -d "{
  \"short_code\": \"$SHORT_CODE\"
}" localhost:9092 api.v1.ShortenerService/DeleteShortLink

# 5. Verify deletion (should return 410 Gone)
curl -I http://localhost:8080/$SHORT_CODE
```

### Batch Creation

```bash
# Create multiple short links
for url in \
  "https://example.com/page1" \
  "https://example.com/page2" \
  "https://example.com/page3"
do
  grpcurl -plaintext -d "{
    \"long_url\": \"$url\"
  }" localhost:9092 api.v1.ShortenerService/CreateShortLink
done
```

### Custom Codes for Marketing

```bash
# Create branded short links
grpcurl -plaintext -d '{
  "long_url": "https://example.com/summer-sale",
  "custom_code": "summer2025"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink

# Result: http://localhost:8080/summer2025
```

### Temporary Links

```bash
# Create a link that expires in 24 hours
EXPIRES_AT=$(date -v+1d +%s)

grpcurl -plaintext -d "{
  \"long_url\": \"https://example.com/temp-content\",
  \"expires_at\": $EXPIRES_AT
}" localhost:9092 api.v1.ShortenerService/CreateShortLink
```

---

## Rate Limiting

**Note**: Rate limiting is not yet implemented in the MVP version.

Future implementation will include:
- Per-IP rate limiting: 100 requests per minute
- HTTP 429 (Too Many Requests) response with Retry-After header
- Configurable limits per API key/user

---

## Best Practices

### URL Validation

Always validate URLs before shortening:

```bash
# Good: Valid HTTPS URL
grpcurl -plaintext -d '{
  "long_url": "https://example.com/page"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink

# Bad: Invalid protocol (will be rejected)
grpcurl -plaintext -d '{
  "long_url": "javascript:alert(1)"
}' localhost:9092 api.v1.ShortenerService/CreateShortLink
```

### Custom Code Guidelines

1. **Use descriptive codes**: `summer-sale` instead of `abc123`
2. **Keep it short**: 4-10 characters is ideal
3. **Avoid ambiguous characters**: Don't use `0` and `O`, `1` and `l` together
4. **Check availability**: Handle `SHORT_CODE_EXISTS` errors gracefully

### Expiration Strategy

1. **Temporary content**: Set expiration for time-sensitive links
2. **Permanent links**: Omit `expires_at` for evergreen content
3. **Cleanup**: Periodically delete expired links to free up codes

### Error Handling

Always handle errors appropriately:

```go
// Example in Go
resp, err := client.CreateShortLink(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if !ok {
        // Handle non-gRPC error
        return err
    }
    
    switch st.Code() {
    case codes.InvalidArgument:
        // Handle validation error
        log.Printf("Invalid input: %s", st.Message())
    case codes.AlreadyExists:
        // Handle duplicate custom code
        log.Printf("Code already exists: %s", st.Message())
    case codes.Internal:
        // Handle server error
        log.Printf("Server error: %s", st.Message())
    default:
        log.Printf("Unexpected error: %s", st.Message())
    }
    return err
}
```

### Performance Optimization

1. **Use caching**: The service uses multi-tier caching (L1 + L2 + MySQL)
2. **Batch operations**: Create multiple links in parallel
3. **Monitor metrics**: Check Prometheus metrics for cache hit rates
4. **Connection pooling**: Reuse gRPC connections

### Security Considerations

1. **Validate all URLs**: The service validates URLs, but client-side validation is also recommended
2. **Monitor for abuse**: Track creation patterns to detect spam
3. **Use HTTPS**: Always use HTTPS for production deployments
4. **Implement authentication**: Add API keys or OAuth2 for production

---

## Monitoring

### Prometheus Metrics

The service exposes metrics on port 9090:

```bash
curl http://localhost:9090/metrics
```

**Key Metrics:**

- `shortener_requests_total`: Total requests by method and status
- `shortener_request_duration_seconds`: Request latency histogram
- `shortener_cache_hits_total`: Cache hits by layer (L1/L2)
- `shortener_cache_misses_total`: Cache misses by layer
- `shortener_errors_total`: Total errors by type
- `shortener_redirects_total`: Total redirects by status code

### Logging

The service uses structured logging (JSON format in production):

```json
{
  "level": "info",
  "ts": 1737360000,
  "msg": "Short link created",
  "short_code": "abc1234",
  "long_url": "https://example.com/page",
  "source_ip": "192.168.1.100"
}
```

---

## Support

For questions or issues:
- Check the [main README](../README.md)
- Review the [design document](../../../.kiro/specs/url-shortener-service/design.md)
- Contact the backend-go-team

## Related Documentation

- [Service README](../README.md) - Service overview and setup
- [Protocol Buffer Definition](../../../api/v1/shortener.proto) - API contract
- [Design Document](../../../.kiro/specs/url-shortener-service/design.md) - Architecture and design decisions
- [Requirements Document](../../../.kiro/specs/url-shortener-service/requirements.md) - Feature requirements
