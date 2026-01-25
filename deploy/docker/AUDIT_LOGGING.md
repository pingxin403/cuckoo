# Audit Logging Implementation Guide

## Overview

This guide explains the audit logging system for the IM Chat System, which tracks all data access and modification events for security, compliance, and forensic purposes.

## Audit Logging Requirements

### Events to Log

**Data Access Events**:
- User login/logout
- Message retrieval
- User profile access
- Group membership queries
- Offline message retrieval

**Data Modification Events**:
- Message sent
- Message deleted
- User profile updated
- Group membership changed
- Settings modified

**Security Events**:
- Authentication failures
- Authorization failures
- Token refresh
- Password changes
- Account lockouts

**Administrative Events**:
- User account creation/deletion
- Permission changes
- Configuration changes
- System maintenance

## Audit Log Format

### Standard Fields

```json
{
  "timestamp": "2025-01-25T10:30:45.123Z",
  "event_id": "evt_abc123",
  "event_type": "message_sent",
  "event_category": "data_modification",
  "severity": "info",
  "user_id": "user123",
  "device_id": "device456",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_5 like Mac OS X)",
  "session_id": "sess_xyz789",
  "trace_id": "trace_abc123",
  "result": "success",
  "details": {
    "msg_id": "msg789",
    "recipient_id": "user456",
    "conversation_type": "private",
    "message_size_bytes": 1024
  }
}
```

### Event Categories

- `authentication`: Login, logout, token operations
- `authorization`: Permission checks, access denials
- `data_access`: Reading data
- `data_modification`: Creating, updating, deleting data
- `security`: Security-related events
- `administrative`: Admin operations
- `system`: System events

### Severity Levels

- `debug`: Detailed diagnostic information
- `info`: Normal operations
- `warn`: Warning conditions
- `error`: Error conditions
- `critical`: Critical security events

## Implementation

### Audit Logger Interface

**File**: `apps/im-service/audit/audit_logger.go`

```go
package audit

import (
    "context"
    "encoding/json"
    "time"
)

type AuditLogger struct {
    mysqlClient *sql.DB
    kafkaProducer *kafka.Producer
}

type AuditEvent struct {
    Timestamp     time.Time              `json:"timestamp"`
    EventID       string                 `json:"event_id"`
    EventType     string                 `json:"event_type"`
    EventCategory string                 `json:"event_category"`
    Severity      string                 `json:"severity"`
    UserID        string                 `json:"user_id"`
    DeviceID      string                 `json:"device_id,omitempty"`
    IPAddress     string                 `json:"ip_address"`
    UserAgent     string                 `json:"user_agent,omitempty"`
    SessionID     string                 `json:"session_id,omitempty"`
    TraceID       string                 `json:"trace_id,omitempty"`
    Result        string                 `json:"result"`
    Details       map[string]interface{} `json:"details,omitempty"`
}

func NewAuditLogger(mysqlClient *sql.DB, kafkaProducer *kafka.Producer) *AuditLogger {
    return &AuditLogger{
        mysqlClient:   mysqlClient,
        kafkaProducer: kafkaProducer,
    }
}

func (l *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
    // Set timestamp if not provided
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }

    // Generate event ID if not provided
    if event.EventID == "" {
        event.EventID = generateEventID()
    }

    // 1. Write to MySQL for long-term storage
    if err := l.writeToMySQL(ctx, event); err != nil {
        return fmt.Errorf("failed to write audit log to MySQL: %w", err)
    }

    // 2. Publish to Kafka for real-time processing
    if err := l.publishToKafka(ctx, event); err != nil {
        // Log error but don't fail (MySQL is primary storage)
        log.Error("failed to publish audit log to Kafka", "error", err)
    }

    return nil
}

func (l *AuditLogger) writeToMySQL(ctx context.Context, event *AuditEvent) error {
    detailsJSON, err := json.Marshal(event.Details)
    if err != nil {
        return err
    }

    query := `
        INSERT INTO audit_logs (
            event_id, timestamp, event_type, event_category, severity,
            user_id, device_id, ip_address, user_agent, session_id,
            trace_id, result, details
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    _, err = l.mysqlClient.ExecContext(ctx, query,
        event.EventID, event.Timestamp, event.EventType, event.EventCategory,
        event.Severity, event.UserID, event.DeviceID, event.IPAddress,
        event.UserAgent, event.SessionID, event.TraceID, event.Result,
        detailsJSON)

    return err
}

func (l *AuditLogger) publishToKafka(ctx context.Context, event *AuditEvent) error {
    eventJSON, err := json.Marshal(event)
    if err != nil {
        return err
    }

    topic := "audit_logs"
    return l.kafkaProducer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &topic,
            Partition: kafka.PartitionAny,
        },
        Key:   []byte(event.UserID),
        Value: eventJSON,
    }, nil)
}

// Helper methods for common audit events

func (l *AuditLogger) LogMessageSent(ctx context.Context, userID, msgID, recipientID string) error {
    return l.Log(ctx, &AuditEvent{
        EventType:     "message_sent",
        EventCategory: "data_modification",
        Severity:      "info",
        UserID:        userID,
        Result:        "success",
        Details: map[string]interface{}{
            "msg_id":       msgID,
            "recipient_id": recipientID,
        },
    })
}

func (l *AuditLogger) LogMessageDeleted(ctx context.Context, userID, msgID string) error {
    return l.Log(ctx, &AuditEvent{
        EventType:     "message_deleted",
        EventCategory: "data_modification",
        Severity:      "warn",
        UserID:        userID,
        Result:        "success",
        Details: map[string]interface{}{
            "msg_id": msgID,
        },
    })
}

func (l *AuditLogger) LogDataExport(ctx context.Context, userID string, recordCount int) error {
    return l.Log(ctx, &AuditEvent{
        EventType:     "data_export",
        EventCategory: "data_access",
        Severity:      "info",
        UserID:        userID,
        Result:        "success",
        Details: map[string]interface{}{
            "record_count": recordCount,
        },
    })
}

func (l *AuditLogger) LogAuthenticationFailure(ctx context.Context, userID, reason string, ipAddress string) error {
    return l.Log(ctx, &AuditEvent{
        EventType:     "authentication_failed",
        EventCategory: "security",
        Severity:      "warn",
        UserID:        userID,
        IPAddress:     ipAddress,
        Result:        "failure",
        Details: map[string]interface{}{
            "reason": reason,
        },
    })
}
```

### MySQL Schema

```sql
CREATE TABLE audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_id VARCHAR(64) NOT NULL UNIQUE,
    timestamp TIMESTAMP(3) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    event_category VARCHAR(32) NOT NULL,
    severity VARCHAR(16) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    device_id VARCHAR(64),
    ip_address VARCHAR(45),
    user_agent TEXT,
    session_id VARCHAR(64),
    trace_id VARCHAR(64),
    result VARCHAR(16) NOT NULL,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_event_type (event_type),
    INDEX idx_event_category (event_category),
    INDEX idx_severity (severity),
    INDEX idx_trace_id (trace_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Partition by month for better performance
ALTER TABLE audit_logs PARTITION BY RANGE (UNIX_TIMESTAMP(timestamp)) (
    PARTITION p202501 VALUES LESS THAN (UNIX_TIMESTAMP('2025-02-01')),
    PARTITION p202502 VALUES LESS THAN (UNIX_TIMESTAMP('2025-03-01')),
    PARTITION p202503 VALUES LESS THAN (UNIX_TIMESTAMP('2025-04-01')),
    -- Add more partitions as needed
    PARTITION p_future VALUES LESS THAN MAXVALUE
);
```

## Audit Log Retention

### Retention Policy

- **Active Storage**: 90 days in MySQL
- **Archive Storage**: 1-7 years in S3/cold storage
- **Deletion**: After retention period expires

### Archival Process

**Script**: `scripts/archive-audit-logs.sh`

```bash
#!/bin/bash
set -e

# Archive audit logs older than 90 days
ARCHIVE_DATE=$(date -d '90 days ago' +%Y-%m-%d)

echo "Archiving audit logs older than $ARCHIVE_DATE..."

# Export to JSON
mysql -u root -p im_chat -e "
    SELECT * FROM audit_logs 
    WHERE timestamp < '$ARCHIVE_DATE'
    INTO OUTFILE '/tmp/audit_logs_archive.json'
    FIELDS TERMINATED BY ',' 
    ENCLOSED BY '\"'
    LINES TERMINATED BY '\n'
"

# Upload to S3
aws s3 cp /tmp/audit_logs_archive.json \
    s3://audit-logs-archive/$(date +%Y/%m)/audit_logs_$ARCHIVE_DATE.json

# Delete from MySQL
mysql -u root -p im_chat -e "
    DELETE FROM audit_logs 
    WHERE timestamp < '$ARCHIVE_DATE'
"

echo "Archive complete"
```

## Audit Log Search

### Search API

**GET /api/v1/audit/search**

Search audit logs with filters.

**Request**:
```http
GET /api/v1/audit/search?user_id=user123&event_type=message_sent&start_date=2025-01-01&end_date=2025-01-31 HTTP/1.1
Host: gateway.example.com
Authorization: Bearer <admin_token>
```

**Response**:
```json
{
  "total": 1234,
  "page": 1,
  "page_size": 100,
  "events": [
    {
      "timestamp": "2025-01-25T10:30:45Z",
      "event_id": "evt_abc123",
      "event_type": "message_sent",
      "user_id": "user123",
      "result": "success",
      "details": {
        "msg_id": "msg789",
        "recipient_id": "user456"
      }
    }
  ]
}
```

### Search Implementation

```go
func (s *AuditService) SearchLogs(ctx context.Context, filters *SearchFilters) (*SearchResults, error) {
    query := `
        SELECT event_id, timestamp, event_type, event_category, severity,
               user_id, device_id, ip_address, result, details
        FROM audit_logs
        WHERE 1=1
    `
    args := []interface{}{}

    if filters.UserID != "" {
        query += " AND user_id = ?"
        args = append(args, filters.UserID)
    }

    if filters.EventType != "" {
        query += " AND event_type = ?"
        args = append(args, filters.EventType)
    }

    if !filters.StartDate.IsZero() {
        query += " AND timestamp >= ?"
        args = append(args, filters.StartDate)
    }

    if !filters.EndDate.IsZero() {
        query += " AND timestamp <= ?"
        args = append(args, filters.EndDate)
    }

    query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
    args = append(args, filters.PageSize, filters.Offset)

    rows, err := s.mysqlClient.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Parse results...
    return results, nil
}
```

## Monitoring Audit Logs

### Metrics

- Audit log write rate (logs/sec)
- Audit log write latency
- Audit log storage size
- Failed audit log writes

### Alerts

**High Audit Log Write Failures**:
```yaml
- alert: HighAuditLogWriteFailures
  expr: rate(audit_log_write_failures_total[5m]) > 10
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High rate of audit log write failures"
```

**Suspicious Activity**:
```yaml
- alert: SuspiciousAuthenticationFailures
  expr: |
    sum by (user_id) (
      rate(audit_logs{event_type="authentication_failed"}[5m])
    ) > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Suspicious authentication failures for user {{ $labels.user_id }}"
```

## Security Best Practices

### DO:
- Log all data access and modifications
- Include sufficient context (user_id, IP, etc.)
- Use structured logging (JSON)
- Implement tamper-proof logging
- Retain logs for compliance period
- Monitor audit log health
- Restrict access to audit logs

### DON'T:
- Log sensitive data (passwords, tokens, PII)
- Allow users to delete their own audit logs
- Store audit logs in same database as application data
- Ignore audit log write failures
- Allow modification of audit logs
- Share audit logs with unauthorized parties

## Compliance

### GDPR Requirements

- Log all data access and modifications
- Retain audit logs for 90 days minimum
- Provide audit trail for data deletion requests
- Allow users to request their audit logs

### SOC 2 Requirements

- Comprehensive audit logging
- Tamper-proof log storage
- Regular log review
- Incident response based on audit logs

### PCI DSS Requirements

- Log all access to cardholder data
- Retain logs for 1 year minimum
- Daily log review
- Automated alerting on suspicious activity

## Testing

### Test Audit Logging

```bash
# Send message (should create audit log)
curl -X POST https://gateway.example.com/api/v1/messages \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"recipient_id": "user456", "content": "Hello"}'

# Search audit logs
curl "https://gateway.example.com/api/v1/audit/search?user_id=user123&event_type=message_sent" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Support

For audit logging questions:
- Email: security@example.com
- Slack: #security-team
- Documentation: https://wiki.example.com/security/audit-logging
