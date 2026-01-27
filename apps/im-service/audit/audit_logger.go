package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditLogger handles audit logging for security and compliance
type AuditLogger struct {
	mysqlClient *sql.DB
}

// AuditEvent represents an audit log event
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

// Event categories
const (
	CategoryAuthentication   = "authentication"
	CategoryAuthorization    = "authorization"
	CategoryDataAccess       = "data_access"
	CategoryDataModification = "data_modification"
	CategorySecurity         = "security"
	CategoryAdministrative   = "administrative"
	CategorySystem           = "system"
)

// Severity levels
const (
	SeverityDebug    = "debug"
	SeverityInfo     = "info"
	SeverityWarn     = "warn"
	SeverityError    = "error"
	SeverityCritical = "critical"
)

// Result values
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// NewAuditLogger creates a new audit logger
func NewAuditLogger(mysqlClient *sql.DB) *AuditLogger {
	return &AuditLogger{
		mysqlClient: mysqlClient,
	}
}

// Log writes an audit event to storage
// Validates: Requirements 13.3, 13.4
func (l *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate event ID if not provided
	if event.EventID == "" {
		event.EventID = generateEventID()
	}

	// Validate required fields
	if err := l.validateEvent(event); err != nil {
		return err
	}

	// Write to MySQL for long-term storage
	if err := l.writeToMySQL(ctx, event); err != nil {
		return fmt.Errorf("failed to write audit log to MySQL: %w", err)
	}

	return nil
}

// LogMessageSent logs a message sent event
func (l *AuditLogger) LogMessageSent(ctx context.Context, userID, msgID, recipientID string) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "message_sent",
		EventCategory: CategoryDataModification,
		Severity:      SeverityInfo,
		UserID:        userID,
		Result:        ResultSuccess,
		Details: map[string]interface{}{
			"msg_id":       msgID,
			"recipient_id": recipientID,
		},
	})
}

// LogMessageDeleted logs a message deleted event
func (l *AuditLogger) LogMessageDeleted(ctx context.Context, userID, msgID string) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "message_deleted",
		EventCategory: CategoryDataModification,
		Severity:      SeverityWarn,
		UserID:        userID,
		Result:        ResultSuccess,
		Details: map[string]interface{}{
			"msg_id": msgID,
		},
	})
}

// LogDataExport logs a data export event
func (l *AuditLogger) LogDataExport(ctx context.Context, userID string, recordCount int) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "data_export",
		EventCategory: CategoryDataAccess,
		Severity:      SeverityInfo,
		UserID:        userID,
		Result:        ResultSuccess,
		Details: map[string]interface{}{
			"record_count": recordCount,
		},
	})
}

// LogDeletionRequest logs a data deletion request
func (l *AuditLogger) LogDeletionRequest(ctx context.Context, userID string) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "data_deletion_request",
		EventCategory: CategoryDataModification,
		Severity:      SeverityWarn,
		UserID:        userID,
		Result:        ResultSuccess,
	})
}

// LogDeletionComplete logs a data deletion completion
func (l *AuditLogger) LogDeletionComplete(ctx context.Context, userID string, details map[string]interface{}) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "data_deletion_complete",
		EventCategory: CategoryDataModification,
		Severity:      SeverityWarn,
		UserID:        userID,
		Result:        ResultSuccess,
		Details:       details,
	})
}

// LogAuthenticationFailure logs an authentication failure
func (l *AuditLogger) LogAuthenticationFailure(ctx context.Context, userID, reason, ipAddress string) error {
	return l.Log(ctx, &AuditEvent{
		EventType:     "authentication_failed",
		EventCategory: CategorySecurity,
		Severity:      SeverityWarn,
		UserID:        userID,
		IPAddress:     ipAddress,
		Result:        ResultFailure,
		Details: map[string]interface{}{
			"reason": reason,
		},
	})
}

// validateEvent validates required fields in an audit event
func (l *AuditLogger) validateEvent(event *AuditEvent) error {
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if event.EventCategory == "" {
		return fmt.Errorf("event_category is required")
	}
	if event.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	if event.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if event.Result == "" {
		return fmt.Errorf("result is required")
	}
	return nil
}

// writeToMySQL writes an audit event to MySQL
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

// generateEventID generates a unique event ID
func generateEventID() string {
	return "evt_" + uuid.New().String()
}
