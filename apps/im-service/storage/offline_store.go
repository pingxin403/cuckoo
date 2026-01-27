package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// OfflineMessage represents a message stored for offline delivery
type OfflineMessage struct {
	ID               int64             `json:"id"`
	MsgID            string            `json:"msg_id"`
	UserID           string            `json:"user_id"`
	SenderID         string            `json:"sender_id"`
	ConversationID   string            `json:"conversation_id"`
	ConversationType string            `json:"conversation_type"` // "private" or "group"
	Content          string            `json:"content"`
	SequenceNumber   int64             `json:"sequence_number"`
	Timestamp        int64             `json:"timestamp"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// OfflineStore handles offline message storage operations
type OfflineStore struct {
	db *sql.DB
}

// Config holds database configuration
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// NewOfflineStore creates a new offline message store
func NewOfflineStore(config Config) (*OfflineStore, error) {
	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &OfflineStore{db: db}, nil
}

// Close closes the database connection
func (s *OfflineStore) Close() error {
	return s.db.Close()
}

// GetDB returns the underlying database connection
// This is used by other services that need direct database access
func (s *OfflineStore) GetDB() *sql.DB {
	return s.db
}

// BatchInsert inserts multiple messages in a single transaction
// Maximum batch size is 100 messages per transaction
func (s *OfflineStore) BatchInsert(ctx context.Context, messages []OfflineMessage) error {
	if len(messages) == 0 {
		return nil
	}

	if len(messages) > 100 {
		return fmt.Errorf("batch size exceeds maximum of 100 messages")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO offline_messages (
			msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, expires_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, msg := range messages {
		// Convert metadata map to JSON string (simplified - in production use json.Marshal)
		var metadata interface{}
		if len(msg.Metadata) > 0 {
			metadata = msg.Metadata
		}

		_, err := stmt.ExecContext(ctx,
			msg.MsgID,
			msg.UserID,
			msg.SenderID,
			msg.ConversationID,
			msg.ConversationType,
			msg.Content,
			msg.SequenceNumber,
			msg.Timestamp,
			msg.ExpiresAt,
			metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to insert message %s: %w", msg.MsgID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMessages retrieves offline messages for a user with cursor-based pagination
// cursor is the last message ID from the previous page (0 for first page)
// limit is the maximum number of messages to return (max 100)
func (s *OfflineStore) GetMessages(ctx context.Context, userID string, cursor int64, limit int) ([]OfflineMessage, error) {
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	query := `
		SELECT 
			id, msg_id, user_id, sender_id, conversation_id, conversation_type,
			content, sequence_number, timestamp, created_at, expires_at
		FROM offline_messages
		WHERE user_id = ? AND id > ?
		ORDER BY sequence_number ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []OfflineMessage
	for rows.Next() {
		var msg OfflineMessage
		err := rows.Scan(
			&msg.ID,
			&msg.MsgID,
			&msg.UserID,
			&msg.SenderID,
			&msg.ConversationID,
			&msg.ConversationType,
			&msg.Content,
			&msg.SequenceNumber,
			&msg.Timestamp,
			&msg.CreatedAt,
			&msg.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessageCount returns the count of offline messages for a user
func (s *OfflineStore) GetMessageCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM offline_messages WHERE user_id = ?`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}
	return count, nil
}

// DeleteExpiredMessages deletes messages older than the TTL
// Returns the number of messages deleted
// Batch size is limited to prevent long-running transactions
func (s *OfflineStore) DeleteExpiredMessages(ctx context.Context, batchSize int) (int64, error) {
	if batchSize <= 0 || batchSize > 10000 {
		return 0, fmt.Errorf("batch size must be between 1 and 10000")
	}

	query := `DELETE FROM offline_messages WHERE expires_at < NOW() LIMIT ?`
	result, err := s.db.ExecContext(ctx, query, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetExpiredMessageCount returns the count of expired messages
func (s *OfflineStore) GetExpiredMessageCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM offline_messages WHERE expires_at < NOW()`
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count expired messages: %w", err)
	}
	return count, nil
}

// DeleteMessagesByUser deletes all messages for a specific user (for GDPR compliance)
func (s *OfflineStore) DeleteMessagesByUser(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM offline_messages WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetOldestExpiredMessage returns the timestamp of the oldest expired message
func (s *OfflineStore) GetOldestExpiredMessage(ctx context.Context) (*time.Time, error) {
	var expiresAt sql.NullTime
	query := `SELECT MIN(expires_at) FROM offline_messages WHERE expires_at < NOW()`
	err := s.db.QueryRowContext(ctx, query).Scan(&expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get oldest expired message: %w", err)
	}

	if !expiresAt.Valid {
		return nil, nil // No expired messages
	}

	return &expiresAt.Time, nil
}
