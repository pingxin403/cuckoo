# User Service

User Service provides user profile and group membership management for the IM Chat System.

## Overview

The User Service is a gRPC-based microservice that manages:
- User profiles (username, display name, avatar, status)
- Group metadata (name, creator, member count)
- Group membership (roles, join dates, mute status)

## Features

- **User Profile Management**: Retrieve single or batch user profiles
- **Group Membership**: Query group members with cursor-based pagination
- **Membership Validation**: Fast validation of user-group membership
- **Large Group Support**: Efficient pagination for groups with >1,000 members
- **MySQL Storage**: Persistent storage with connection pooling

## API

### gRPC Service

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc BatchGetUsers(BatchGetUsersRequest) returns (BatchGetUsersResponse);
  rpc GetGroupMembers(GetGroupMembersRequest) returns (GetGroupMembersResponse);
  rpc ValidateGroupMembership(ValidateGroupMembershipRequest) returns (ValidateGroupMembershipResponse);
}
```

### Key Operations

#### GetUser
Retrieves a single user's profile by user_id.

```bash
grpcurl -plaintext -d '{"user_id": "user001"}' localhost:9096 user.v1.UserService/GetUser
```

#### BatchGetUsers
Retrieves multiple users' profiles in a single request (max 100 users).

```bash
grpcurl -plaintext -d '{"user_ids": ["user001", "user002", "user003"]}' localhost:9096 user.v1.UserService/BatchGetUsers
```

#### GetGroupMembers
Retrieves group members with cursor-based pagination (default 100, max 1000 per page).

```bash
grpcurl -plaintext -d '{"group_id": "group001", "limit": 100}' localhost:9096 user.v1.UserService/GetGroupMembers
```

#### ValidateGroupMembership
Checks if a user is a member of a specific group.

```bash
grpcurl -plaintext -d '{"user_id": "user001", "group_id": "group001"}' localhost:9096 user.v1.UserService/ValidateGroupMembership
```

## Configuration

### Environment Variables

#### Service Configuration
- `PORT`: gRPC server port (default: 9096)
- `MYSQL_DSN`: MySQL connection string (default: `im_service:im_password@tcp(localhost:3306)/im_chat?parseTime=true`)

#### Observability Configuration
- `SERVICE_NAME`: Service name for observability (default: "user-service")
- `SERVICE_VERSION`: Service version (default: "1.0.0")
- `DEPLOYMENT_ENVIRONMENT`: Deployment environment (default: "development")
- `ENABLE_METRICS`: Enable metrics collection (default: true)
- `METRICS_PORT`: Metrics server port (default: 9090)
- `LOG_LEVEL`: Log level - debug, info, warn, error (default: "info")
- `LOG_FORMAT`: Log format - json, text (default: "json")
- `ENABLE_OTEL_METRICS`: Enable OpenTelemetry metrics export (default: false)
- `ENABLE_OTEL_LOGS`: Enable OpenTelemetry logs export (default: false)
- `ENABLE_OTEL_TRACING`: Enable distributed tracing (default: false)
- `OTLP_ENDPOINT`: OpenTelemetry collector endpoint (default: "")
- `ENABLE_PPROF`: Enable pprof profiling endpoints (default: false)

### MySQL Connection Pool

- Max open connections: 25
- Max idle connections: 5
- Connection max lifetime: 5 minutes

## Database Schema

### Tables

#### users
Stores user profile information.

```sql
CREATE TABLE users (
    user_id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(256) NOT NULL UNIQUE,
    display_name VARCHAR(256) NOT NULL,
    avatar_url VARCHAR(512),
    status INT NOT NULL DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### groups
Stores group metadata.

```sql
CREATE TABLE groups (
    group_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    creator_id VARCHAR(64) NOT NULL,
    member_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### group_members
Stores group membership information.

```sql
CREATE TABLE group_members (
    group_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    role INT NOT NULL DEFAULT 1,
    group_display_name VARCHAR(256),
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (group_id, user_id)
);
```

## Development

### Running Locally

```bash
# Set up MySQL database
# Database migrations are now in apps/im-service/migrations/
docker compose -f deploy/docker/docker-compose.infra.yml up liquibase

# Run the service
export MYSQL_DSN="im_service:im_password@tcp(localhost:3306)/im_chat?parseTime=true"
go run main.go
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run only unit tests
go test -v -run Test[^P] ./service/...

# Run only property-based tests
go test -v -run TestProperty ./service/...
```

### Test Coverage

- **Service package**: 90.7% coverage
- **Total tests**: 24 (16 unit + 8 property-based)
- **Property test iterations**: 100 per property (800 total)

## Deployment

### Kubernetes

The service is deployed using Kustomize manifests in `deploy/k8s/services/user-service/`.

```bash
# Deploy to development
kubectl apply -k deploy/k8s/overlays/development

# Deploy to production
kubectl apply -k deploy/k8s/overlays/production
```

### Docker

```bash
# Build image
docker build -t user-service:latest .

# Run container
docker run -p 9096:9096 \
  -e MYSQL_DSN="im_service:im_password@tcp(mysql:3306)/im_chat?parseTime=true" \
  user-service:latest
```

## Performance

### Benchmarks

- **GetUser**: ~0.5ms per request
- **BatchGetUsers**: ~2ms for 100 users
- **GetGroupMembers**: ~1ms per page (100 members)
- **ValidateGroupMembership**: ~0.3ms per request

### Scalability

- Supports 100+ concurrent requests per instance
- Horizontal scaling via Kubernetes replicas
- Connection pooling prevents database overload
- Cursor-based pagination for large groups (>1,000 members)

## Integration

### With IM Service

The IM Service calls User Service to:
- Validate group membership before routing group messages
- Retrieve user profiles for message metadata
- Query group members for message broadcasting

### With Auth Service

User Service does not directly integrate with Auth Service. Authentication is handled at the gateway level.

## Monitoring

### Observability

The service integrates with the unified observability library providing:
- **Metrics**: Prometheus-compatible metrics on port 9090
- **Structured Logging**: JSON-formatted logs with trace correlation
- **Distributed Tracing**: OpenTelemetry traces (when enabled)
- **Performance Profiling**: pprof endpoints (when enabled)

### Metrics

The service exposes the following metrics on `http://localhost:9090/metrics`:

#### User Operation Metrics
- `user_operations_total{operation, status}`: Total user operations (get, batch_get, get_group_members, validate_membership)
  - Labels: `operation` (get, batch_get, get_group_members, validate_membership), `status` (success, failure, not_found, not_member)

#### Database Operation Metrics
- `user_db_operations_total{operation, status}`: Total database operations
  - Labels: `operation` (get, batch_get, get_group_members, validate_membership), `status` (success, failure, not_found, not_member)
- `user_db_operation_duration_seconds{operation}`: Database operation duration histogram
  - Labels: `operation` (get, batch_get, get_group_members, validate_membership)

#### gRPC Metrics
- `user_grpc_requests_total{method}`: Total gRPC requests
  - Labels: `method` (GetUser, BatchGetUsers, GetGroupMembers, ValidateGroupMembership)
- `user_grpc_request_duration_seconds{method}`: gRPC request duration histogram
  - Labels: `method` (GetUser, BatchGetUsers, GetGroupMembers, ValidateGroupMembership)

### Structured Logging

All logs are structured with the following fields:
- `timestamp`: ISO 8601 timestamp
- `level`: Log level (debug, info, warn, error)
- `service`: Service name (user-service)
- `message`: Log message
- Additional context fields (port, error, etc.)

Example log entry:
```json
{
  "timestamp": "2024-01-25T10:30:00Z",
  "level": "info",
  "service": "user-service",
  "message": "user-service listening",
  "port": "9096"
}
```

### Distributed Tracing

When tracing is enabled (`ENABLE_OTEL_TRACING=true`), the service creates trace spans for:
- gRPC method calls
- Database operations

Traces are exported to the configured OTLP endpoint and can be viewed in Jaeger or other tracing backends.

### Performance Profiling

When pprof is enabled (`ENABLE_PPROF=true`), profiling endpoints are available on the metrics port:
- `http://localhost:9090/debug/pprof/`: Profile index
- `http://localhost:9090/debug/pprof/heap`: Memory heap profile
- `http://localhost:9090/debug/pprof/goroutine`: Goroutine profile
- `http://localhost:9090/debug/pprof/profile`: CPU profile (30s)

### Health Checks

The service exposes gRPC health checks compatible with Kubernetes probes.

## Error Handling

### Error Codes

- `USER_ERROR_CODE_USER_NOT_FOUND`: User does not exist
- `USER_ERROR_CODE_GROUP_NOT_FOUND`: Group does not exist
- `USER_ERROR_CODE_INVALID_REQUEST`: Invalid request parameters
- `USER_ERROR_CODE_DATABASE_ERROR`: Database operation failed
- `USER_ERROR_CODE_TOO_MANY_IDS`: Batch request exceeds 100 users
- `USER_ERROR_CODE_INTERNAL_ERROR`: Internal server error

## Contributing

See the main [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

See [LICENSE](../../LICENSE) for details.
