# IM Service

IM (Instant Messaging) Service provides core message routing functionality for the chat system.

## Components

- **Registry**: User-to-gateway mapping with etcd backend
- **Sequence Generator**: Monotonic sequence number generation with Redis
- **Sequence Backup**: MySQL-based backup for sequence recovery

## Quick Start

### Testing

```bash
# Fast unit tests (1 second) - recommended for development
./scripts/test-coverage.sh

# Full test suite including property tests (7 minutes)
./scripts/test-coverage.sh --with-property

# Or use make from root
make test APP=im
```

**Note**: Property-based tests are slow due to TTL waits. See [TESTING.md](TESTING.md) for details.

### Development

```bash
# Run fast unit tests
go test ./... -run "^Test[^P]" -v

# Run linter
golangci-lint run ./...

# Build
go build -o bin/im-service .
```

## Architecture

### Registry Service
- Manages user-to-gateway mappings in etcd
- Supports multi-device connections
- 90-second TTL with heartbeat renewal
- Watch mechanism for cache invalidation

See [registry/README.md](registry/README.md) for details.

### Sequence Generator
- Generates monotonic sequence numbers using Redis INCR
- Supports private chat and group chat
- MySQL backup every 10,000 messages
- Recovery on Redis failure

See [sequence/README.md](sequence/README.md) for details.

## Requirements Validated

- **7.1**: Registry with 90-second TTL
- **7.2**: Lease renewal every 30 seconds
- **7.6**: etcd cluster (3 or 5 nodes)
- **7.9**: Watch mechanism for Registry changes
- **15.1**: Multi-device support
- **15.2**: Device ID in Registry
- **16.1**: Monotonic sequence numbers
- **16.2**: Redis-based sequence generation
- **16.6**: Conversation-specific sequences
- **16.7**: MySQL backup for sequences
- **17.3**: Watch-based cache invalidation

## Test Coverage

- **Unit Tests**: 48 tests (fast, < 1 second)
- **Property Tests**: 14 tests (slow, ~7 minutes)
- **Total**: 62 tests validating correctness

## Dependencies

- etcd v3.6.7 (Registry)
- Redis (Sequence Generator)
- MySQL (Sequence Backup)
- pgregory.net/rapid (Property-based testing)
