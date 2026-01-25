package export

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// ExportService handles GDPR-compliant data export
type ExportService struct {
	mysqlClient *sql.DB
}

// UserExport represents the complete export of user data
type UserExport struct {
	UserID     string                 `json:"user_id"`
	ExportDate time.Time              `json:"export_date"`
	Data       map[string]interface{} `json:"data"`
}

// Message represents a message in the export
type Message struct {
	MsgID            string     `json:"msg_id"`
	ConversationType string     `json:"conversation_type"`
	ConversationID   string     `json:"conversation_id"`
	SenderID         string     `json:"sender_id"`
	RecipientID      string     `json:"recipient_id"`
	Content          string     `json:"content"`
	Timestamp        time.Time  `json:"timestamp"`
	SequenceNumber   int64      `json:"sequence_number"`
	Delivered        bool       `json:"delivered"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
}

// Statistics represents usage statistics
type Statistics struct {
	TotalMessagesSent     int `json:"total_messages_sent"`
	TotalMessagesReceived int `json:"total_messages_received"`
	TotalConversations    int `json:"total_conversations"`
}

// NewExportService creates a new export service
func NewExportService(mysqlClient *sql.DB) *ExportService {
	return &ExportService{
		mysqlClient: mysqlClient,
	}
}

// ExportUserData exports all data for a user in portable JSON format
// Validates: Requirements 13.5
func (s *ExportService) ExportUserData(ctx context.Context, userID string) (*UserExport, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	export := &UserExport{
		UserID:     userID,
		ExportDate: time.Now(),
		Data:       make(map[string]interface{}),
	}

	// 1. Export messages
	messages, err := s.exportMessages(ctx, userID)
	if err != nil {
		return nil, err
	}
	export.Data["messages"] = messages

	// 2. Export statistics
	stats, err := s.exportStatistics(ctx, userID)
	if err != nil {
		return nil, err
	}
	export.Data["statistics"] = stats

	return export, nil
}

// ExportToJSON exports user data as JSON string
func (s *ExportService) ExportToJSON(ctx context.Context, userID string) (string, error) {
	export, err := s.ExportUserData(ctx, userID)
	if err != nil {
		return "", err
	}

	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// exportMessages exports all messages for a user
func (s *ExportService) exportMessages(ctx context.Context, userID string) ([]Message, error) {
	query := `
		SELECT msg_id, conversation_type, conversation_id, sender_id, 
		       recipient_id, content, timestamp, sequence_number, 
		       delivered, read_at
		FROM offline_messages
		WHERE user_id = ? OR sender_id = ?
		ORDER BY timestamp DESC
		LIMIT 10000
	`

	rows, err := s.mysqlClient.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var messages []Message
	for rows.Next() {
		var msg Message
		var readAt sql.NullTime

		err := rows.Scan(
			&msg.MsgID,
			&msg.ConversationType,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.RecipientID,
			&msg.Content,
			&msg.Timestamp,
			&msg.SequenceNumber,
			&msg.Delivered,
			&readAt,
		)
		if err != nil {
			return nil, err
		}

		if readAt.Valid {
			msg.ReadAt = &readAt.Time
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// exportStatistics exports usage statistics for a user
func (s *ExportService) exportStatistics(ctx context.Context, userID string) (*Statistics, error) {
	stats := &Statistics{}

	// Count messages sent
	err := s.mysqlClient.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM offline_messages WHERE sender_id = ?`,
		userID,
	).Scan(&stats.TotalMessagesSent)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Count messages received
	err = s.mysqlClient.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM offline_messages WHERE user_id = ? AND sender_id != ?`,
		userID, userID,
	).Scan(&stats.TotalMessagesReceived)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Count unique conversations
	err = s.mysqlClient.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT conversation_id) FROM offline_messages WHERE user_id = ? OR sender_id = ?`,
		userID, userID,
	).Scan(&stats.TotalConversations)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return stats, nil
}

// Custom errors
var (
	ErrUserIDRequired = &ExportError{Code: "USER_ID_REQUIRED", Message: "user_id is required"}
)

// ExportError represents an export error
type ExportError struct {
	Code    string
	Message string
}

func (e *ExportError) Error() string {
	return e.Message
}
