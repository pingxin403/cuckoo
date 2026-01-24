package readreceipt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("successfully marks message as read", func(t *testing.T) {
		msgID := "msg-123"
		readerID := "user-001"
		senderID := "user-002"
		conversationID := "conv-123"
		conversationType := "private"
		deviceID := "device-001"

		// Expect transaction begin
		mock.ExpectBegin()

		// Expect insert into read_receipts
		mock.ExpectExec("INSERT INTO read_receipts").
			WithArgs(msgID, readerID, senderID, conversationID, conversationType, sqlmock.AnyArg(), deviceID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect update of offline_messages
		mock.ExpectExec("UPDATE offline_messages").
			WithArgs(sqlmock.AnyArg(), msgID, readerID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect commit
		mock.ExpectCommit()

		receipt, err := service.MarkAsRead(ctx, msgID, readerID, senderID, conversationID, conversationType, deviceID)

		assert.NoError(t, err)
		assert.NotNil(t, receipt)
		assert.Equal(t, msgID, receipt.MsgID)
		assert.Equal(t, readerID, receipt.ReaderID)
		assert.Equal(t, senderID, receipt.SenderID)
		assert.Equal(t, conversationID, receipt.ConversationID)
		assert.Equal(t, conversationType, receipt.ConversationType)
		assert.Equal(t, deviceID, receipt.DeviceID)
		assert.WithinDuration(t, time.Now(), receipt.ReadAt, 2*time.Second)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when msgID is empty", func(t *testing.T) {
		receipt, err := service.MarkAsRead(ctx, "", "user-001", "user-002", "conv-123", "private", "device-001")

		assert.Error(t, err)
		assert.Nil(t, receipt)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when readerID is empty", func(t *testing.T) {
		receipt, err := service.MarkAsRead(ctx, "msg-123", "", "user-002", "conv-123", "private", "device-001")

		assert.Error(t, err)
		assert.Nil(t, receipt)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when senderID is empty", func(t *testing.T) {
		receipt, err := service.MarkAsRead(ctx, "msg-123", "user-001", "", "conv-123", "private", "device-001")

		assert.Error(t, err)
		assert.Nil(t, receipt)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("handles database error on insert", func(t *testing.T) {
		msgID := "msg-123"
		readerID := "user-001"
		senderID := "user-002"
		conversationID := "conv-123"
		conversationType := "private"
		deviceID := "device-001"

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO read_receipts").
			WithArgs(msgID, readerID, senderID, conversationID, conversationType, sqlmock.AnyArg(), deviceID).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		receipt, err := service.MarkAsRead(ctx, msgID, readerID, senderID, conversationID, conversationType, deviceID)

		assert.Error(t, err)
		assert.Nil(t, receipt)
		assert.Contains(t, err.Error(), "failed to insert read receipt")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetReadReceipts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("successfully retrieves read receipts", func(t *testing.T) {
		msgID := "msg-123"
		readAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "msg_id", "reader_id", "sender_id", "conversation_id", "conversation_type", "read_at", "device_id"}).
			AddRow(1, msgID, "user-001", "user-002", "conv-123", "private", readAt, "device-001").
			AddRow(2, msgID, "user-003", "user-002", "conv-123", "private", readAt.Add(time.Minute), "device-002")

		mock.ExpectQuery("SELECT (.+) FROM read_receipts").
			WithArgs(msgID).
			WillReturnRows(rows)

		receipts, err := service.GetReadReceipts(ctx, msgID)

		assert.NoError(t, err)
		assert.Len(t, receipts, 2)
		assert.Equal(t, "user-001", receipts[0].ReaderID)
		assert.Equal(t, "user-003", receipts[1].ReaderID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no receipts found", func(t *testing.T) {
		msgID := "msg-999"

		rows := sqlmock.NewRows([]string{"id", "msg_id", "reader_id", "sender_id", "conversation_id", "conversation_type", "read_at", "device_id"})

		mock.ExpectQuery("SELECT (.+) FROM read_receipts").
			WithArgs(msgID).
			WillReturnRows(rows)

		receipts, err := service.GetReadReceipts(ctx, msgID)

		assert.NoError(t, err)
		assert.Empty(t, receipts)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles database error", func(t *testing.T) {
		msgID := "msg-123"

		mock.ExpectQuery("SELECT (.+) FROM read_receipts").
			WithArgs(msgID).
			WillReturnError(sql.ErrConnDone)

		receipts, err := service.GetReadReceipts(ctx, msgID)

		assert.Error(t, err)
		assert.Nil(t, receipts)
		assert.Contains(t, err.Error(), "failed to query read receipts")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetUnreadCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("successfully retrieves unread count", func(t *testing.T) {
		userID := "user-001"
		expectedCount := 5

		rows := sqlmock.NewRows([]string{"count"}).AddRow(expectedCount)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := service.GetUnreadCount(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns zero when no unread messages", func(t *testing.T) {
		userID := "user-002"

		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := service.GetUnreadCount(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, 0, count)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles database error", func(t *testing.T) {
		userID := "user-001"

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(userID).
			WillReturnError(sql.ErrConnDone)

		count, err := service.GetUnreadCount(ctx, userID)

		assert.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Contains(t, err.Error(), "failed to get unread count")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetUnreadMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("successfully retrieves unread messages", func(t *testing.T) {
		userID := "user-001"
		limit := 10
		offset := 0

		rows := sqlmock.NewRows([]string{"msg_id"}).
			AddRow("msg-001").
			AddRow("msg-002").
			AddRow("msg-003")

		mock.ExpectQuery("SELECT msg_id FROM offline_messages").
			WithArgs(userID, limit, offset).
			WillReturnRows(rows)

		msgIDs, err := service.GetUnreadMessages(ctx, userID, limit, offset)

		assert.NoError(t, err)
		assert.Len(t, msgIDs, 3)
		assert.Equal(t, "msg-001", msgIDs[0])
		assert.Equal(t, "msg-002", msgIDs[1])
		assert.Equal(t, "msg-003", msgIDs[2])

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when no unread messages", func(t *testing.T) {
		userID := "user-002"
		limit := 10
		offset := 0

		rows := sqlmock.NewRows([]string{"msg_id"})

		mock.ExpectQuery("SELECT msg_id FROM offline_messages").
			WithArgs(userID, limit, offset).
			WillReturnRows(rows)

		msgIDs, err := service.GetUnreadMessages(ctx, userID, limit, offset)

		assert.NoError(t, err)
		assert.Empty(t, msgIDs)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMarkConversationAsRead(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("successfully marks conversation as read", func(t *testing.T) {
		userID := "user-001"
		conversationID := "conv-123"

		mock.ExpectExec("UPDATE offline_messages").
			WithArgs(sqlmock.AnyArg(), userID, conversationID).
			WillReturnResult(sqlmock.NewResult(0, 5))

		rowsAffected, err := service.MarkConversationAsRead(ctx, userID, conversationID)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), rowsAffected)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns zero when no messages to mark", func(t *testing.T) {
		userID := "user-002"
		conversationID := "conv-456"

		mock.ExpectExec("UPDATE offline_messages").
			WithArgs(sqlmock.AnyArg(), userID, conversationID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		rowsAffected, err := service.MarkConversationAsRead(ctx, userID, conversationID)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles database error", func(t *testing.T) {
		userID := "user-001"
		conversationID := "conv-123"

		mock.ExpectExec("UPDATE offline_messages").
			WithArgs(sqlmock.AnyArg(), userID, conversationID).
			WillReturnError(sql.ErrConnDone)

		rowsAffected, err := service.MarkConversationAsRead(ctx, userID, conversationID)

		assert.Error(t, err)
		assert.Equal(t, int64(0), rowsAffected)
		assert.Contains(t, err.Error(), "failed to mark conversation as read")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetReadStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	t.Run("returns read status from offline_messages", func(t *testing.T) {
		msgID := "msg-123"
		userID := "user-001"
		readAt := time.Now()

		rows := sqlmock.NewRows([]string{"read_at"}).AddRow(readAt)

		mock.ExpectQuery("SELECT read_at FROM offline_messages").
			WithArgs(msgID, userID).
			WillReturnRows(rows)

		isRead, timestamp, err := service.GetReadStatus(ctx, msgID, userID)

		assert.NoError(t, err)
		assert.True(t, isRead)
		assert.NotNil(t, timestamp)
		assert.WithinDuration(t, readAt, *timestamp, time.Second)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns unread status when read_at is NULL", func(t *testing.T) {
		msgID := "msg-123"
		userID := "user-001"

		rows := sqlmock.NewRows([]string{"read_at"}).AddRow(nil)

		mock.ExpectQuery("SELECT read_at FROM offline_messages").
			WithArgs(msgID, userID).
			WillReturnRows(rows)

		isRead, timestamp, err := service.GetReadStatus(ctx, msgID, userID)

		assert.NoError(t, err)
		assert.False(t, isRead)
		assert.Nil(t, timestamp)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("checks read_receipts table when message not in offline storage", func(t *testing.T) {
		msgID := "msg-123"
		userID := "user-001"
		readAt := time.Now()

		// First query returns no rows (not in offline_messages)
		mock.ExpectQuery("SELECT read_at FROM offline_messages").
			WithArgs(msgID, userID).
			WillReturnError(sql.ErrNoRows)

		// Second query checks read_receipts table
		rows := sqlmock.NewRows([]string{"read_at"}).AddRow(readAt)
		mock.ExpectQuery("SELECT read_at FROM read_receipts").
			WithArgs(msgID, userID).
			WillReturnRows(rows)

		isRead, timestamp, err := service.GetReadStatus(ctx, msgID, userID)

		assert.NoError(t, err)
		assert.True(t, isRead)
		assert.NotNil(t, timestamp)
		assert.WithinDuration(t, readAt, *timestamp, time.Second)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when message not found in either table", func(t *testing.T) {
		msgID := "msg-999"
		userID := "user-001"

		// First query returns no rows
		mock.ExpectQuery("SELECT read_at FROM offline_messages").
			WithArgs(msgID, userID).
			WillReturnError(sql.ErrNoRows)

		// Second query also returns no rows
		mock.ExpectQuery("SELECT read_at FROM read_receipts").
			WithArgs(msgID, userID).
			WillReturnError(sql.ErrNoRows)

		isRead, timestamp, err := service.GetReadStatus(ctx, msgID, userID)

		assert.NoError(t, err)
		assert.False(t, isRead)
		assert.Nil(t, timestamp)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestNewReadReceiptServiceWithKafka tests service creation with Kafka producer
func TestNewReadReceiptServiceWithKafka(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Use invalid broker to test error handling
	service := NewReadReceiptServiceWithKafka(db, []string{"invalid:9092"}, "read_receipts")

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	// Service should still be created even if Kafka fails
	assert.False(t, service.kafkaEnabled)
}

// TestClose tests service cleanup
func TestClose(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)

	service := NewReadReceiptService(db)

	err = service.Close()
	assert.NoError(t, err)
}

// TestCloseWithKafka tests service cleanup with Kafka producer
func TestCloseWithKafka(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create service without Kafka (kafkaProducer will be nil)
	service := NewReadReceiptService(db)

	err = service.Close()
	assert.NoError(t, err)
}

// TestPublishReadReceiptEvent tests Kafka event publishing
func TestPublishReadReceiptEvent(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	t.Run("returns error when producer is nil", func(t *testing.T) {
		service := NewReadReceiptService(db)

		receipt := &ReadReceipt{
			ID:    1,
			MsgID: "msg-001",
		}

		// Should return error when producer is nil
		err := service.publishReadReceiptEvent(context.Background(), receipt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kafka producer not initialized")
	})
}

// TestMarkAsReadWithKafkaDisabled tests marking as read without Kafka
func TestMarkAsReadWithKafkaDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewReadReceiptService(db)
	ctx := context.Background()

	msgID := "msg-123"
	readerID := "user-001"
	senderID := "user-002"
	conversationID := "conv-123"
	conversationType := "private"
	deviceID := "device-001"

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect insert into read_receipts
	mock.ExpectExec("INSERT INTO read_receipts").
		WithArgs(msgID, readerID, senderID, conversationID, conversationType, sqlmock.AnyArg(), deviceID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect update of offline_messages
	mock.ExpectExec("UPDATE offline_messages").
		WithArgs(sqlmock.AnyArg(), msgID, readerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	receipt, err := service.MarkAsRead(ctx, msgID, readerID, senderID, conversationID, conversationType, deviceID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.NoError(t, mock.ExpectationsWereMet())
}
