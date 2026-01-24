package readreceipt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

// ReadReceipt represents a read receipt for a message
type ReadReceipt struct {
	ID               int64     `json:"id"`
	MsgID            string    `json:"msg_id"`
	ReaderID         string    `json:"reader_id"`
	SenderID         string    `json:"sender_id"`
	ConversationID   string    `json:"conversation_id"`
	ConversationType string    `json:"conversation_type"` // "private" or "group"
	ReadAt           time.Time `json:"read_at"`
	DeviceID         string    `json:"device_id,omitempty"`
}

// ReadReceiptService handles read receipt tracking and delivery
type ReadReceiptService struct {
	db            *sql.DB
	kafkaProducer sarama.SyncProducer
	kafkaTopic    string
	kafkaEnabled  bool
}

// NewReadReceiptService creates a new read receipt service
func NewReadReceiptService(db *sql.DB) *ReadReceiptService {
	return &ReadReceiptService{
		db:           db,
		kafkaEnabled: false,
	}
}

// NewReadReceiptServiceWithKafka creates a new read receipt service with Kafka support
func NewReadReceiptServiceWithKafka(db *sql.DB, kafkaBrokers []string, topic string) *ReadReceiptService {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all replicas
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Partitioner = sarama.NewHashPartitioner // Partition by key (sender_id)

	producer, err := sarama.NewSyncProducer(kafkaBrokers, config)
	if err != nil {
		// Log error but don't fail service initialization
		fmt.Printf("Warning: failed to create Kafka producer: %v\n", err)
		return &ReadReceiptService{
			db:           db,
			kafkaEnabled: false,
		}
	}

	return &ReadReceiptService{
		db:            db,
		kafkaProducer: producer,
		kafkaTopic:    topic,
		kafkaEnabled:  true,
	}
}

// Close closes the Kafka producer
func (s *ReadReceiptService) Close() error {
	if s.kafkaProducer != nil {
		return s.kafkaProducer.Close()
	}
	return nil
}

// MarkAsRead marks a message as read and creates a read receipt
// This implements Requirements 5.1 and 5.2
func (s *ReadReceiptService) MarkAsRead(ctx context.Context, msgID, readerID, senderID, conversationID, conversationType, deviceID string) (*ReadReceipt, error) {
	if msgID == "" || readerID == "" || senderID == "" {
		return nil, fmt.Errorf("msgID, readerID, and senderID are required")
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	readAt := time.Now()

	// Insert read receipt (or update if already exists)
	query := `
		INSERT INTO read_receipts (msg_id, reader_id, sender_id, conversation_id, conversation_type, read_at, device_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE read_at = VALUES(read_at)
	`
	result, err := tx.ExecContext(ctx, query, msgID, readerID, senderID, conversationID, conversationType, readAt, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert read receipt: %w", err)
	}

	// Update offline_messages table if the message exists there
	updateQuery := `
		UPDATE offline_messages
		SET read_at = ?
		WHERE msg_id = ? AND user_id = ? AND read_at IS NULL
	`
	_, err = tx.ExecContext(ctx, updateQuery, readAt, msgID, readerID)
	if err != nil {
		return nil, fmt.Errorf("failed to update offline message read status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get the inserted/updated receipt ID
	receiptID, err := result.LastInsertId()
	if err != nil {
		// If we can't get the ID, it might be an update, so fetch it
		receiptID, err = s.getReceiptID(ctx, msgID, readerID, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get receipt ID: %w", err)
		}
	}

	receipt := &ReadReceipt{
		ID:               receiptID,
		MsgID:            msgID,
		ReaderID:         readerID,
		SenderID:         senderID,
		ConversationID:   conversationID,
		ConversationType: conversationType,
		ReadAt:           readAt,
		DeviceID:         deviceID,
	}

	// Publish read receipt event to Kafka for real-time delivery
	// Validates: Requirements 5.3, 5.4
	if s.kafkaEnabled {
		if err := s.publishReadReceiptEvent(ctx, receipt); err != nil {
			// Log error but don't fail the operation
			// The read receipt is already persisted in the database
			fmt.Printf("Warning: failed to publish read receipt event: %v\n", err)
		}
	}

	return receipt, nil
}

// getReceiptID retrieves the ID of an existing read receipt
func (s *ReadReceiptService) getReceiptID(ctx context.Context, msgID, readerID, deviceID string) (int64, error) {
	var id int64
	query := `SELECT id FROM read_receipts WHERE msg_id = ? AND reader_id = ? AND device_id = ?`
	err := s.db.QueryRowContext(ctx, query, msgID, readerID, deviceID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get receipt ID: %w", err)
	}
	return id, nil
}

// GetReadReceipts retrieves read receipts for a specific message
// This is used to check who has read a message (useful for group chats)
func (s *ReadReceiptService) GetReadReceipts(ctx context.Context, msgID string) ([]*ReadReceipt, error) {
	query := `
		SELECT id, msg_id, reader_id, sender_id, conversation_id, conversation_type, read_at, device_id
		FROM read_receipts
		WHERE msg_id = ?
		ORDER BY read_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query read receipts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var receipts []*ReadReceipt
	for rows.Next() {
		var receipt ReadReceipt
		var deviceID sql.NullString
		err := rows.Scan(
			&receipt.ID,
			&receipt.MsgID,
			&receipt.ReaderID,
			&receipt.SenderID,
			&receipt.ConversationID,
			&receipt.ConversationType,
			&receipt.ReadAt,
			&deviceID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan read receipt: %w", err)
		}
		if deviceID.Valid {
			receipt.DeviceID = deviceID.String
		}
		receipts = append(receipts, &receipt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating read receipts: %w", err)
	}

	return receipts, nil
}

// GetUnreadCount returns the count of unread messages for a user
func (s *ReadReceiptService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM offline_messages
		WHERE user_id = ? AND read_at IS NULL
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// GetUnreadMessages retrieves unread messages for a user with pagination
func (s *ReadReceiptService) GetUnreadMessages(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	query := `
		SELECT msg_id
		FROM offline_messages
		WHERE user_id = ? AND read_at IS NULL
		ORDER BY timestamp ASC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgIDs []string
	for rows.Next() {
		var msgID string
		if err := rows.Scan(&msgID); err != nil {
			return nil, fmt.Errorf("failed to scan message ID: %w", err)
		}
		msgIDs = append(msgIDs, msgID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unread messages: %w", err)
	}

	return msgIDs, nil
}

// MarkConversationAsRead marks all messages in a conversation as read
// This is useful for "mark all as read" functionality
func (s *ReadReceiptService) MarkConversationAsRead(ctx context.Context, userID, conversationID string) (int64, error) {
	query := `
		UPDATE offline_messages
		SET read_at = ?
		WHERE user_id = ? AND conversation_id = ? AND read_at IS NULL
	`

	readAt := time.Now()
	result, err := s.db.ExecContext(ctx, query, readAt, userID, conversationID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark conversation as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetReadStatus checks if a specific message has been read by a user
func (s *ReadReceiptService) GetReadStatus(ctx context.Context, msgID, userID string) (bool, *time.Time, error) {
	query := `
		SELECT read_at
		FROM offline_messages
		WHERE msg_id = ? AND user_id = ?
	`

	var readAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, msgID, userID).Scan(&readAt)
	if err == sql.ErrNoRows {
		// Message not in offline storage, check read_receipts table
		receiptQuery := `
			SELECT read_at
			FROM read_receipts
			WHERE msg_id = ? AND reader_id = ?
			LIMIT 1
		`
		err = s.db.QueryRowContext(ctx, receiptQuery, msgID, userID).Scan(&readAt)
		if err == sql.ErrNoRows {
			return false, nil, nil
		}
		if err != nil {
			return false, nil, fmt.Errorf("failed to query read receipt: %w", err)
		}
	} else if err != nil {
		return false, nil, fmt.Errorf("failed to query offline message: %w", err)
	}

	if readAt.Valid {
		return true, &readAt.Time, nil
	}
	return false, nil, nil
}

// ReadReceiptEvent represents a read receipt event for Kafka
type ReadReceiptEvent struct {
	MsgID          string `json:"msg_id"`
	SenderID       string `json:"sender_id"`
	ReaderID       string `json:"reader_id"`
	ConversationID string `json:"conversation_id"`
	ReadAt         int64  `json:"read_at"`
	DeviceID       string `json:"device_id,omitempty"`
}

// publishReadReceiptEvent publishes a read receipt event to Kafka
// Validates: Requirements 5.3, 5.4
func (s *ReadReceiptService) publishReadReceiptEvent(_ context.Context, receipt *ReadReceipt) error {
	if s.kafkaProducer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	// Create event
	event := ReadReceiptEvent{
		MsgID:          receipt.MsgID,
		SenderID:       receipt.SenderID,
		ReaderID:       receipt.ReaderID,
		ConversationID: receipt.ConversationID,
		ReadAt:         receipt.ReadAt.Unix(),
		DeviceID:       receipt.DeviceID,
	}

	// Marshal to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal read receipt event: %w", err)
	}

	// Create Kafka message
	// Use sender_id as key for partitioning (ensures all receipts for same sender go to same partition)
	msg := &sarama.ProducerMessage{
		Topic: s.kafkaTopic,
		Key:   sarama.StringEncoder(receipt.SenderID),
		Value: sarama.ByteEncoder(eventData),
	}

	// Send message synchronously
	_, _, err = s.kafkaProducer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message to kafka: %w", err)
	}

	return nil
}
