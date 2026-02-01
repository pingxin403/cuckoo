# Traffic CLI - Quick Start Guide

Get started with the Traffic CLI tool in 5 minutes.

## Prerequisites

- Go 1.24+ installed
- Redis running (default: `localhost:6379`)
- Access to the im-service codebase

## Step 1: Build the CLI Tool

```bash
cd apps/im-service
go build -o bin/traffic-cli ./cmd/traffic-cli
```

## Step 2: Start Redis (if not running)

Using Docker:
```bash
docker run -d -p 6379:6379 redis:latest
```

Or using docker-compose:
```bash
cd deploy/docker
docker-compose up -d redis
```

## Step 3: Check Current Status

```bash
./bin/traffic-cli status
```

Expected output:
```
Current Traffic Configuration
=============================
Version:        1
Last Updated:   2024-01-15T10:00:00Z
Updated By:     system
Default Region: region-a

Region Weights:
  region-a   100% ██████████████████████████████████████████████████
  region-b     0% 
```

## Step 4: Try a Dry Run

Test a traffic switch without applying changes:

```bash
./bin/traffic-cli switch proportional region-a:80 region-b:20 \
  --dry-run \
  --reason "Testing traffic distribution"
```

## Step 5: Apply a Traffic Switch

Switch traffic to 80:20 distribution:

```bash
./bin/traffic-cli switch proportional region-a:80 region-b:20 \
  --reason "Load balancing test"
```

## Step 6: Verify the Change

```bash
./bin/traffic-cli status
```

You should see:
```
Region Weights:
  region-a    80% ████████████████████████████████████████
  region-b    20% ██████████
```

## Step 7: Test User Routing

See which region a user would be routed to:

```bash
./bin/traffic-cli route user123
```

## Step 8: View Event History

```bash
./bin/traffic-cli events
```

## Common Commands Cheat Sheet

### View Status
```bash
./bin/traffic-cli status
```

### Proportional Switch
```bash
# 90:10 split
./bin/traffic-cli switch proportional region-a:90 region-b:10 \
  --reason "Your reason here"

# 50:50 split
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Your reason here"
```

### Full Switch
```bash
# All traffic to region-a
./bin/traffic-cli switch full region-a --reason "Your reason here"

# All traffic to region-b
./bin/traffic-cli switch full region-b --reason "Your reason here"
```

### Dry Run (Test First)
```bash
./bin/traffic-cli switch proportional region-a:70 region-b:30 \
  --dry-run \
  --reason "Testing configuration"
```

### View Events
```bash
# Last 10 events
./bin/traffic-cli events

# Last 20 events
./bin/traffic-cli events --limit 20
```

### Test Routing
```bash
./bin/traffic-cli route user123
```

## Real-World Scenarios

### Scenario 1: Gradual Migration

Migrate traffic from region-b to region-a gradually:

```bash
# Phase 1: 70:30
./bin/traffic-cli switch proportional region-a:70 region-b:30 \
  --reason "Migration phase 1"

# Wait and monitor...

# Phase 2: 90:10
./bin/traffic-cli switch proportional region-a:90 region-b:10 \
  --reason "Migration phase 2"

# Wait and monitor...

# Phase 3: Complete migration
./bin/traffic-cli switch full region-a \
  --reason "Migration complete"
```

### Scenario 2: Emergency Failover

Quickly switch all traffic to a healthy region:

```bash
./bin/traffic-cli switch full region-b \
  --reason "Emergency: region-a database outage" \
  --operator "oncall-engineer"
```

### Scenario 3: Maintenance Window

Drain traffic before maintenance:

```bash
# Before maintenance
./bin/traffic-cli switch full region-a \
  --reason "Maintenance on region-b"

# Perform maintenance...

# After maintenance
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Maintenance complete"
```

## Configuration Options

### Redis Connection

```bash
# Custom Redis address
./bin/traffic-cli status --redis-addr "redis.example.com:6379"

# With password
./bin/traffic-cli status \
  --redis-addr "redis.example.com:6379" \
  --redis-password "secret"

# Custom database
./bin/traffic-cli status --redis-db 2
```

### Operator Identification

```bash
# Specify operator
./bin/traffic-cli switch full region-a \
  --reason "Planned maintenance" \
  --operator "ops-team"
```

## Troubleshooting

### Redis Connection Failed

```bash
# Test Redis connection
redis-cli ping

# Check if Redis is running
docker ps | grep redis

# Start Redis if needed
docker-compose up -d redis
```

### Invalid Weights Error

Make sure weights sum to exactly 100:
```bash
# ✗ Wrong (sums to 90)
./bin/traffic-cli switch proportional region-a:60 region-b:30

# ✓ Correct (sums to 100)
./bin/traffic-cli switch proportional region-a:60 region-b:40
```

### Lock Acquisition Failed

Another operation is in progress. Wait 30 seconds or check:
```bash
# Check Redis for stuck locks
redis-cli GET traffic:lock

# Manually release if needed (use with caution!)
redis-cli DEL traffic:lock
```

## Next Steps

- Read the full [README](README.md) for detailed documentation
- Run the [integration tests](integration_test.sh)
- Check the [multi-region architecture design](../../../.kiro/specs/multi-region-active-active/design.md)

## Getting Help

```bash
# Show help for all commands
./bin/traffic-cli --help

# Show help for a specific command
./bin/traffic-cli switch --help
./bin/traffic-cli switch proportional --help
```

## Tips

1. **Always use dry-run first**: Test your configuration before applying
2. **Provide meaningful reasons**: Use `--reason` to document why you're switching
3. **Monitor after switching**: Check metrics and logs after traffic changes
4. **Gradual changes**: For planned migrations, change weights gradually
5. **Keep event history**: Use `events` command to track all changes

Happy traffic switching! 🚦
