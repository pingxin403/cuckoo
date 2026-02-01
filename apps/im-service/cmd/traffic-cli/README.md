# Traffic CLI - Multi-Region Traffic Switching Tool

A command-line tool for managing traffic distribution between regions in a multi-region active-active architecture.

## Features

- **Proportional Traffic Distribution**: Distribute traffic across regions with custom percentages (e.g., 90:10, 50:50)
- **Full Traffic Switch**: Switch 100% of traffic to a single region with one command
- **Dry-Run Mode**: Test traffic switches without applying changes
- **Event Logging**: Track all traffic switching events with detailed history
- **Route Testing**: Test which region a specific user would be routed to
- **Real-time Status**: View current traffic configuration and distribution

## Installation

Build the CLI tool:

```bash
cd apps/im-service
go build -o bin/traffic-cli ./cmd/traffic-cli
```

The binary will be created at `apps/im-service/bin/traffic-cli`.

## Usage

### Global Flags

All commands support these global flags:

- `--redis-addr`: Redis server address (default: `localhost:6379`)
- `--redis-password`: Redis password (default: empty)
- `--redis-db`: Redis database number (default: `0`)
- `--dry-run`: Perform a dry run without applying changes
- `--operator`: Operator performing the switch (default: current user)
- `--reason`: Reason for the traffic switch (required for switch commands)

### Commands

#### 1. View Current Status

Display the current traffic configuration:

```bash
./bin/traffic-cli status
```

Example output:
```
Current Traffic Configuration
=============================
Version:        5
Last Updated:   2024-01-15T10:30:00Z
Updated By:     ops-team
Default Region: region-a

Region Weights:
  region-a    90% █████████████████████████████████████████████
  region-b    10% █████
```

#### 2. Proportional Traffic Switch

Switch traffic with custom proportions:

```bash
# Switch 90% to region-a, 10% to region-b
./bin/traffic-cli switch proportional region-a:90 region-b:10 \
  --reason "Gradual migration to region-a"

# Switch 50% to each region
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Load balancing test"

# Dry run mode (test without applying)
./bin/traffic-cli switch proportional region-a:80 region-b:20 \
  --dry-run \
  --reason "Testing new configuration"
```

#### 3. Full Traffic Switch

Switch 100% of traffic to a single region:

```bash
# Switch all traffic to region-a
./bin/traffic-cli switch full region-a \
  --reason "Maintenance on region-b"

# Switch all traffic to region-b
./bin/traffic-cli switch full region-b \
  --reason "Failover due to region-a outage"

# Dry run mode
./bin/traffic-cli switch full region-a \
  --dry-run \
  --reason "Testing failover procedure"
```

#### 4. View Event History

Display recent traffic switching events:

```bash
# Show last 10 events (default)
./bin/traffic-cli events

# Show last 20 events
./bin/traffic-cli events --limit 20
```

Example output:
```
Recent Traffic Switching Events (showing 3)
===========================================

Event ID:   switch_1705315800000000000
Type:       proportional
Status:     completed
Timestamp:  2024-01-15T10:30:00Z
Operator:   ops-team
Reason:     Gradual migration to region-a
Duration:   2.5s
Changes:
  region-a: 80% → 90%
  region-b: 20% → 10%
--------------------------------------------------
```

#### 5. Test User Routing

Test which region a specific user would be routed to:

```bash
./bin/traffic-cli route user123
```

Example output:
```
Routing Information for User: user123
=====================================
Target Region: region-a

Current Configuration:
→ region-a    90%
  region-b    10%
```

## Common Scenarios

### Scenario 1: Gradual Migration

Gradually migrate traffic from region-b to region-a:

```bash
# Step 1: Start with 70:30
./bin/traffic-cli switch proportional region-a:70 region-b:30 \
  --reason "Migration phase 1"

# Step 2: Increase to 90:10
./bin/traffic-cli switch proportional region-a:90 region-b:10 \
  --reason "Migration phase 2"

# Step 3: Full switch to region-a
./bin/traffic-cli switch full region-a \
  --reason "Migration complete"
```

### Scenario 2: Emergency Failover

Quickly switch all traffic to a healthy region:

```bash
# Switch all traffic to region-b
./bin/traffic-cli switch full region-b \
  --reason "Emergency: region-a database outage" \
  --operator "oncall-engineer"
```

### Scenario 3: Maintenance Window

Drain traffic from a region for maintenance:

```bash
# Before maintenance: drain region-b
./bin/traffic-cli switch full region-a \
  --reason "Maintenance window for region-b"

# After maintenance: restore traffic
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Maintenance complete, restoring balance"
```

### Scenario 4: Testing Configuration

Test a configuration change without applying it:

```bash
# Dry run to see what would happen
./bin/traffic-cli switch proportional region-a:60 region-b:40 \
  --dry-run \
  --reason "Testing new load distribution"

# If satisfied, apply the change
./bin/traffic-cli switch proportional region-a:60 region-b:40 \
  --reason "Applying tested load distribution"
```

## Configuration

### Redis Connection

The CLI tool connects to Redis to read and update traffic configuration. Configure the connection using flags:

```bash
./bin/traffic-cli status \
  --redis-addr "redis.example.com:6379" \
  --redis-password "secret" \
  --redis-db 2
```

Or set environment variables:

```bash
export REDIS_ADDR="redis.example.com:6379"
export REDIS_PASSWORD="secret"
export REDIS_DB="2"
```

## Traffic Distribution Algorithm

The CLI tool uses a hash-based routing algorithm to distribute traffic:

1. **User ID Hashing**: Each user ID is hashed to a consistent value
2. **Weight-Based Selection**: The hash is mapped to a region based on configured weights
3. **Consistency**: The same user always routes to the same region (unless weights change)

Example with 90:10 distribution:
- Users with hash % 100 < 90 → region-a
- Users with hash % 100 >= 90 → region-b

## Event Logging

All traffic switches are logged with the following information:

- **Event ID**: Unique identifier for the event
- **Type**: `proportional` or `full_switch`
- **Status**: `started`, `completed`, `failed`, or `dry_run`
- **Timestamp**: When the switch occurred
- **Operator**: Who performed the switch
- **Reason**: Why the switch was performed
- **Duration**: How long the switch took
- **Configuration Changes**: Before and after weights

Events are stored in Redis and can be queried using the `events` command.

## Safety Features

### 1. Distributed Locking

The CLI tool uses Redis distributed locks to prevent concurrent traffic switches:

- Only one switch operation can run at a time
- Lock timeout: 30 seconds
- Automatic lock release on completion or failure

### 2. Validation

All traffic switches are validated before applying:

- Region names must be valid (`region-a` or `region-b`)
- Weights must be between 0 and 100
- Total weights must equal 100
- Reason must be provided (for audit trail)

### 3. Dry-Run Mode

Test any configuration change without applying it:

```bash
./bin/traffic-cli switch proportional region-a:80 region-b:20 \
  --dry-run \
  --reason "Testing configuration"
```

Dry-run mode shows:
- What the new configuration would be
- Estimated switch duration
- No actual changes are applied

## Troubleshooting

### Connection Issues

If you see "failed to connect to Redis":

1. Check Redis is running: `redis-cli ping`
2. Verify the address: `--redis-addr localhost:6379`
3. Check authentication: `--redis-password <password>`

### Lock Acquisition Failed

If you see "Another traffic switch operation is in progress":

1. Wait for the current operation to complete (max 30 seconds)
2. Check if a previous operation is stuck
3. Manually release the lock if needed (use Redis CLI)

### Invalid Weights

If you see "total weight must equal 100":

1. Ensure all weights sum to exactly 100
2. Example: `region-a:90 region-b:10` ✓
3. Example: `region-a:90 region-b:20` ✗ (sums to 110)

## Integration with Monitoring

The traffic switcher emits metrics that can be monitored:

- `traffic_switch_events_total`: Total number of traffic switches
- `traffic_switch_duration_seconds`: Duration of traffic switches
- `traffic_config_version`: Current configuration version

Use these metrics to:
- Track traffic switch frequency
- Monitor switch performance
- Alert on configuration changes

## Best Practices

1. **Always Provide a Reason**: Use `--reason` to document why you're switching traffic
2. **Use Dry-Run First**: Test configuration changes with `--dry-run` before applying
3. **Gradual Changes**: For planned migrations, change weights gradually (e.g., 70:30 → 80:20 → 90:10)
4. **Monitor After Switch**: Check metrics and logs after switching traffic
5. **Document in Runbooks**: Include CLI commands in your incident response runbooks

## Examples

### Example 1: Planned Maintenance

```bash
# Check current status
./bin/traffic-cli status

# Dry run the switch
./bin/traffic-cli switch full region-a --dry-run \
  --reason "Maintenance on region-b"

# Apply the switch
./bin/traffic-cli switch full region-a \
  --reason "Maintenance on region-b" \
  --operator "ops-team"

# Verify the switch
./bin/traffic-cli status

# After maintenance, restore balance
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Maintenance complete"
```

### Example 2: Load Testing

```bash
# Gradually increase load on region-b
./bin/traffic-cli switch proportional region-a:80 region-b:20 \
  --reason "Load test phase 1"

# Monitor metrics, then increase further
./bin/traffic-cli switch proportional region-a:60 region-b:40 \
  --reason "Load test phase 2"

# If successful, balance the load
./bin/traffic-cli switch proportional region-a:50 region-b:50 \
  --reason "Load test successful, balancing"
```

## Related Documentation

- [Multi-Region Architecture Design](../../.kiro/specs/multi-region-active-active/design.md)
- [Traffic Switcher Implementation](../../traffic/traffic_switcher.go)
- [Requirements](../../.kiro/specs/multi-region-active-active/requirements.md)

## Support

For issues or questions:
1. Check the event log: `./bin/traffic-cli events`
2. Review Redis logs
3. Check application metrics
4. Contact the platform team
