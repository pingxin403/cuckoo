package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	event := &AuditEvent{
		EventType:     "message_sent",
		EventCategory: CategoryDataModification,
		Severity:      SeverityInfo,
		UserID:        "user123",
		IPAddress:     "192.168.1.100",
		Result:        ResultSuccess,
		Details: map[string]interface{}{
			"msg_id":       "msg789",
			"recipient_id": "user456",
		},
	}

	mock.ExpectExec("INSERT INTO audit_logs").
		WithArgs(
			sqlmock.AnyArg(), // event_id (generated)
			sqlmock.AnyArg(), // timestamp (generated)
			"message_sent",
			CategoryDataModification,
			SeverityInfo,
			"user123",
			"",
			"192.168.1.100",
			"",
			"",
			"",
			ResultSuccess,
			sqlmock.AnyArg(), // details JSON
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.Log(ctx, event)

	require.NoError(t, err)
	assert.NotEmpty(t, event.EventID)
	assert.False(t, event.Timestamp.IsZero())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLog_MissingRequiredFields(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	tests := []struct {
		name  string
		event *AuditEvent
		error string
	}{
		{
			name: "missing event_type",
			event: &AuditEvent{
				EventCategory: CategoryDataModification,
				Severity:      SeverityInfo,
				UserID:        "user123",
				Result:        ResultSuccess,
			},
			error: "event_type is required",
		},
		{
			name: "missing event_category",
			event: &AuditEvent{
				EventType: "message_sent",
				Severity:  SeverityInfo,
				UserID:    "user123",
				Result:    ResultSuccess,
			},
			error: "event_category is required",
		},
		{
			name: "missing severity",
			event: &AuditEvent{
				EventType:     "message_sent",
				EventCategory: CategoryDataModification,
				UserID:        "user123",
				Result:        ResultSuccess,
			},
			error: "severity is required",
		},
		{
			name: "missing user_id",
			event: &AuditEvent{
				EventType:     "message_sent",
				EventCategory: CategoryDataModification,
				Severity:      SeverityInfo,
				Result:        ResultSuccess,
			},
			error: "user_id is required",
		},
		{
			name: "missing result",
			event: &AuditEvent{
				EventType:     "message_sent",
				EventCategory: CategoryDataModification,
				Severity:      SeverityInfo,
				UserID:        "user123",
			},
			error: "result is required",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logger.Log(ctx, tt.event)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.error)
		})
	}
}

func TestLog_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	event := &AuditEvent{
		EventType:     "message_sent",
		EventCategory: CategoryDataModification,
		Severity:      SeverityInfo,
		UserID:        "user123",
		Result:        ResultSuccess,
	}

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnError(assert.AnError)

	ctx := context.Background()
	err = logger.Log(ctx, event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write audit log to MySQL")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogMessageSent_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.LogMessageSent(ctx, "user123", "msg789", "user456")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogMessageDeleted_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.LogMessageDeleted(ctx, "user123", "msg789")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogDataExport_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.LogDataExport(ctx, "user123", 100)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogDeletionRequest_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.LogDeletionRequest(ctx, "user123")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogDeletionComplete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	details := map[string]interface{}{
		"offline_messages": 100,
		"redis_keys":       50,
	}
	err = logger.LogDeletionComplete(ctx, "user123", details)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogAuthenticationFailure_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	mock.ExpectExec("INSERT INTO audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.LogAuthenticationFailure(ctx, "user123", "invalid_token", "192.168.1.100")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWriteToMySQL_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	logger := NewAuditLogger(db)

	event := &AuditEvent{
		EventID:       "evt_123",
		Timestamp:     time.Now(),
		EventType:     "message_sent",
		EventCategory: CategoryDataModification,
		Severity:      SeverityInfo,
		UserID:        "user123",
		DeviceID:      "device456",
		IPAddress:     "192.168.1.100",
		UserAgent:     "Mozilla/5.0",
		SessionID:     "sess_789",
		TraceID:       "trace_abc",
		Result:        ResultSuccess,
		Details: map[string]interface{}{
			"msg_id": "msg789",
		},
	}

	detailsJSON, _ := json.Marshal(event.Details)

	mock.ExpectExec("INSERT INTO audit_logs").
		WithArgs(
			event.EventID,
			event.Timestamp,
			event.EventType,
			event.EventCategory,
			event.Severity,
			event.UserID,
			event.DeviceID,
			event.IPAddress,
			event.UserAgent,
			event.SessionID,
			event.TraceID,
			event.Result,
			detailsJSON,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	err = logger.writeToMySQL(ctx, event)

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	id2 := generateEventID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "evt_")
	assert.Contains(t, id2, "evt_")
}

func TestValidateEvent_AllFieldsValid(t *testing.T) {
	logger := NewAuditLogger(nil)

	event := &AuditEvent{
		EventType:     "message_sent",
		EventCategory: CategoryDataModification,
		Severity:      SeverityInfo,
		UserID:        "user123",
		Result:        ResultSuccess,
	}

	err := logger.validateEvent(event)
	assert.NoError(t, err)
}
