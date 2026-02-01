# Multi-Region Active-Active Examples

This directory contains demonstration and testing implementations of multi-region active-active components.

## ⚠️ Important Note

These are **example implementations** for demonstration, testing, and learning purposes. 

**Production implementations** are integrated into the services:
- `apps/im-service/sync/` - Production sync implementation
- `apps/im-gateway-service/routing/` - Production routing implementation

## Components

- **arbiter/** - Distributed coordination and split-brain prevention
- **failover/** - Automatic failover management
- **health/** - Multi-dimensional health checking
- **routing/** - Geographic routing with health-aware failover
- **sync/** - Cross-region message synchronization
- **monitoring/** - Web-based monitoring dashboard

## Usage

Each component can be run independently for testing and demonstration:

```bash
# Run routing example
go run examples/multi-region/routing/cmd/example/main.go

# Run monitoring dashboard
go run examples/multi-region/monitoring/cmd/dashboard/main.go

# Run sync example
go run examples/multi-region/sync/cmd/example/main.go

# Run health check example
go run examples/multi-region/health/cmd/example/main.go

# Run arbiter example
go run examples/multi-region/arbiter/cmd/example/main.go
```

## Testing

Run tests for all components:

```bash
# Test all multi-region components
go test ./examples/multi-region/...

# Test specific component
go test ./examples/multi-region/sync/...
```

## Documentation

See the README in each component directory for detailed documentation.
