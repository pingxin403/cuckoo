# Centralized Logging Guide

## Overview

This guide explains the centralized logging system for the IM Chat System, using the Loki + Grafana stack integrated with OpenTelemetry.

## Architecture

```
┌─────────────────────┐
│  IM Gateway Service │
│                     │
│  ┌───────────────┐ │
│  │ Structured    │ │
│  │ Logging       │ │
│  │ (JSON)        │ │
│  └───────┬───────┘ │
└──────────┼─────────┘
           │
           ▼
    ┌─────────────┐
    │ OTLP        │
    │ Exporter    │
    │ (Logs)      │
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ OTEL        │
    │ Collector   │
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Loki        │
    │ (Storage)   │
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Grafana     │
    │ (Query UI)  │
    └─────────────┘
```

## Log Levels

- **DEBUG**: Detailed diagnostic information (disabled in production)
- **INFO**: General informational messages
- **WARN**: Warning messages for potentially harmful situations
- **ERROR**: Error messages for failures that don't stop the service
- **FATAL**: Critical errors that cause service shutdown

## Structured Log Format

All logs use JSON format with standard fields:

```json
{
  "timestamp": "2025-01-25T10:30:45.123Z",
  "level": "INFO",
  "service": "im-gateway-service",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "message": "Message delivered successfully",
  "user_id": "user123",
  "device_id": "device456",
  "msg_id": "msg789",
  "latency_ms": 45,
  "component": "message-delivery"
}
```

## Log Categories

### 1. Message Delivery Events

**Success**:
```json
{
  "level": "INFO",
  "event": "message_delivered",
  "user_id": "user123",
  "device_id": "device456",
  "msg_id": "msg789",
  "conversation_type": "private",
  "conversation_id": "conv123",
  "sequence_number": 12345,
  "latency_ms": 45,
  "retry_count": 0
}
```

**Failure**:
```json
{
  "level": "ERROR",
  "event": "message_delivery_failed",
  "user_id": "user123",
  "msg_id": "msg789",
  "error": "connection timeout",
  "retry_count": 3,
  "will_retry": false,
  "routed_to_offline": true
}
```

### 2. Connection Events

**Connection Established**:
```json
{
  "level": "INFO",
  "event": "connection_established",
  "user_id": "user123",
  "device_id": "device456",
  "remote_addr": "192.168.1.100",
  "user_agent": "iOS/14.5"
}
```

**Connection Closed**:
```json
{
  "level": "INFO",
  "event": "connection_closed",
  "user_id": "user123",
  "device_id": "device456",
  "reason": "client_disconnect",
  "duration_seconds": 3600
}
```

### 3. Error Events

**Authentication Failure**:
```json
{
  "level": "WARN",
  "event": "auth_failed",
  "remote_addr": "192.168.1.100",
  "error": "invalid_token",
  "token_expired": true
}
```

**Database Error**:
```json
{
  "level": "ERROR",
  "event": "database_error",
  "operation": "insert_offline_message",
  "error": "connection pool exhausted",
  "retry_count": 2
}
```

### 4. Performance Events

**Slow Query**:
```json
{
  "level": "WARN",
  "event": "slow_query",
  "query": "SELECT * FROM offline_messages WHERE user_id = ?",
  "duration_ms": 1500,
  "threshold_ms": 1000
}
```

**High Latency**:
```json
{
  "level": "WARN",
  "event": "high_latency",
  "operation": "message_delivery",
  "latency_ms": 800,
  "threshold_ms": 500
}
```

## Log Retention Policies

- **Development**: 7 days
- **Staging**: 30 days
- **Production**: 90 days

## Querying Logs in Grafana

### Access Grafana Explore

1. Open Grafana: http://localhost:3000
2. Navigate to Explore (compass icon)
3. Select "Loki" as data source

### Example Queries

**All logs from im-gateway-service**:
```logql
{service_name="im-gateway-service"}
```

**Error logs only**:
```logql
{service_name="im-gateway-service"} |= "ERROR"
```

**Message delivery failures**:
```logql
{service_name="im-gateway-service"} | json | event="message_delivery_failed"
```

**Logs for specific user**:
```logql
{service_name="im-gateway-service"} | json | user_id="user123"
```

**High latency events**:
```logql
{service_name="im-gateway-service"} | json | event="high_latency" | latency_ms > 500
```

**Logs with trace ID** (for distributed tracing):
```logql
{service_name="im-gateway-service"} | json | trace_id="abc123..."
```

## Log Aggregation Patterns

### Count errors by type
```logql
sum by (error) (count_over_time({service_name="im-gateway-service"} |= "ERROR" [5m]))
```

### Average latency over time
```logql
avg_over_time({service_name="im-gateway-service"} | json | latency_ms != "" | unwrap latency_ms [5m])
```

### Top users by message count
```logql
topk(10, sum by (user_id) (count_over_time({service_name="im-gateway-service"} | json | event="message_delivered" [1h])))
```

## Integration with Tracing

Logs are automatically correlated with traces using `trace_id` and `span_id` fields:

1. Find a slow request in Jaeger
2. Copy the trace ID
3. Query logs in Grafana:
   ```logql
   {service_name="im-gateway-service"} | json | trace_id="<trace_id>"
   ```

## Best Practices

### DO:
- Use structured logging (JSON format)
- Include trace_id and span_id for correlation
- Log at appropriate levels
- Include context (user_id, msg_id, etc.)
- Log errors with stack traces
- Use consistent field names

### DON'T:
- Log sensitive data (passwords, tokens, PII)
- Log at DEBUG level in production
- Log inside tight loops
- Use string concatenation for log messages
- Log without context

## Configuration

### Environment Variables

```bash
# Log level (DEBUG, INFO, WARN, ERROR, FATAL)
LOG_LEVEL=INFO

# Log format (json, text)
LOG_FORMAT=json

# Enable log sampling (reduce log volume)
LOG_SAMPLING_ENABLED=true
LOG_SAMPLING_INITIAL=100
LOG_SAMPLING_THEREAFTER=100

# OTLP endpoint for logs
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

### Loki Configuration

Edit `loki-config.yaml` to adjust retention:

```yaml
limits_config:
  retention_period: 90d  # Keep logs for 90 days
  
chunk_store_config:
  max_look_back_period: 90d
```

## Troubleshooting

### Logs not appearing in Grafana

1. Check OTel Collector is receiving logs:
   ```bash
   docker logs otel-collector | grep "logs"
   ```

2. Check Loki is healthy:
   ```bash
   curl http://localhost:3100/ready
   ```

3. Verify log export in service:
   ```bash
   docker logs im-gateway-service | grep "log export"
   ```

### High log volume

1. Enable log sampling in production
2. Reduce DEBUG logs
3. Use log aggregation instead of individual logs
4. Adjust retention period

## Monitoring Log Health

### Metrics to Monitor

- Log ingestion rate (logs/sec)
- Log storage size (GB)
- Log query latency (ms)
- Failed log exports

### Grafana Dashboard

Create a dashboard to monitor:
- Log volume by service
- Error rate over time
- Top error types
- Log ingestion lag

## Support

For questions or issues with logging:
- Slack: #observability-support
- Documentation: https://grafana.com/docs/loki/
- OpenTelemetry Logs: https://opentelemetry.io/docs/specs/otel/logs/
