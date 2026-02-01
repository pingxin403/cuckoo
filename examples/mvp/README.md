# MVP Simplified Components

This directory contains simplified implementations of infrastructure components for MVP and testing purposes.

## ⚠️ Warning

**These components are NOT suitable for production use.** They are simplified implementations designed for:
- Local development
- Testing
- Prototyping
- Learning

For production deployments, use proper infrastructure:
- Use **Kafka** instead of `queue/`
- Use **MySQL/PostgreSQL** instead of `storage/`

## Components

- **queue/** - Go channel-based message queue (replaces Kafka for MVP)
- **storage/** - SQLite-based local storage (replaces MySQL for MVP)

## Usage

These components are used by the multi-region examples and tests:

```go
import "github.com/cuckoo-org/cuckoo/examples/mvp/queue"
import "github.com/cuckoo-org/cuckoo/examples/mvp/storage"
```

## Migration to Production

When moving to production:

1. Replace `queue` with Kafka client
2. Replace `storage` with MySQL/PostgreSQL client
3. Update configuration and connection strings
4. Test thoroughly in staging environment
