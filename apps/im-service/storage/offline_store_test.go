package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOfflineStore(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		store := &OfflineStore{db: db}
		assert.NotNil(t, store)
	})

	t.Run("connection failure", func(t *testing.T) {
		// Test with invalid DSN
		config := Config{
			DSN:             "invalid:dsn",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		}
		_, err := NewOfflineStore(config)
		assert.Error(t, err)
	})
}

func TestBatchInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful batch insert", func(t *testing.T) {
		messages := []OfflineMessage{
			{
				MsgID:            "msg-001",
				UserID:           "user-001",
				SenderID:         "user-002",
				ConversationID:   "conv-001",
				ConversationType: "private",
				Content:          "Hello",
				SequenceNumber:   1,
				Timestamp:        time.Now().Unix() * 1000,
				ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
			},
			{
				MsgID:            "msg-002",
				UserID:           "user-001",
				SenderID:         "user-003",
				ConversationID:   "conv-002",
				ConversationType: "group",
				Content:          "World",
				SequenceNumber:   2,
				Timestamp:        time.Now().Unix() * 1000,
				ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
			},
		}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO offline_messages")
		for range messages {
			mock.ExpectExec("INSERT INTO offline_messages").
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectCommit()

		err := store.BatchInsert(ctx, messages)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty batch", func(t *testing.T) {
		err := store.BatchInsert(ctx, []OfflineMessage{})
		assert.NoError(t, err)
	})

	t.Run("batch size exceeds maximum", func(t *testing.T) {
		messages := make([]OfflineMessage, 101)
		err := store.BatchInsert(ctx, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("transaction begin failure", func(t *testing.T) {
		messages := []OfflineMessage{
			{
				MsgID:  "msg-001",
				UserID: "user-001",
			},
		}

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		err := store.BatchInsert(ctx, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("prepare statement failure", func(t *testing.T) {
		messages := []OfflineMessage{
			{
				MsgID:  "msg-001",
				UserID: "user-001",
			},
		}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO offline_messages").
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		err := store.BatchInsert(ctx, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to prepare statement")
	})

	t.Run("insert failure", func(t *testing.T) {
		messages := []OfflineMessage{
			{
				MsgID:            "msg-001",
				UserID:           "user-001",
				SenderID:         "user-002",
				ConversationID:   "conv-001",
				ConversationType: "private",
				Content:          "Hello",
				SequenceNumber:   1,
				Timestamp:        time.Now().Unix() * 1000,
				ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
			},
		}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO offline_messages")
		mock.ExpectExec("INSERT INTO offline_messages").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := store.BatchInsert(ctx, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert message")
	})

	t.Run("commit failure", func(t *testing.T) {
		messages := []OfflineMessage{
			{
				MsgID:            "msg-001",
				UserID:           "user-001",
				SenderID:         "user-002",
				ConversationID:   "conv-001",
				ConversationType: "private",
				Content:          "Hello",
				SequenceNumber:   1,
				Timestamp:        time.Now().Unix() * 1000,
				ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
			},
		}

		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO offline_messages")
		mock.ExpectExec("INSERT INTO offline_messages").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(sql.ErrTxDone)

		err := store.BatchInsert(ctx, messages)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}

func TestGetMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		userID := "user-001"
		cursor := int64(0)
		limit := 10

		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		}).
			AddRow(1, "msg-001", "user-001", "user-002", "conv-001",
				"private", "Hello", 1, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour)).
			AddRow(2, "msg-002", "user-001", "user-003", "conv-002",
				"group", "World", 2, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour))

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs(userID, cursor, limit).
			WillReturnRows(rows)

		messages, err := store.GetMessages(ctx, userID, cursor, limit)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "msg-001", messages[0].MsgID)
		assert.Equal(t, "msg-002", messages[1].MsgID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid limit", func(t *testing.T) {
		_, err := store.GetMessages(ctx, "user-001", 0, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between")

		_, err = store.GetMessages(ctx, "user-001", 0, 101)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between")
	})

	t.Run("query failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs("user-001", int64(0), 10).
			WillReturnError(sql.ErrConnDone)

		_, err := store.GetMessages(ctx, "user-001", 0, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query messages")
	})

	t.Run("scan failure", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		}).
			AddRow(1, "msg-001", "user-001", "user-002", "conv-001",
				"private", "Hello", "invalid-number", time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour))

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs("user-001", int64(0), 10).
			WillReturnRows(rows)

		_, err := store.GetMessages(ctx, "user-001", 0, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan message")
	})

	t.Run("empty result", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs("user-001", int64(0), 10).
			WillReturnRows(rows)

		messages, err := store.GetMessages(ctx, "user-001", 0, 10)
		assert.NoError(t, err)
		assert.Empty(t, messages)
	})
}

func TestGetMessageCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		userID := "user-001"
		expectedCount := int64(42)

		rows := sqlmock.NewRows([]string{"count"}).AddRow(expectedCount)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages").
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := store.GetMessageCount(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages").
			WithArgs("user-001").
			WillReturnError(sql.ErrConnDone)

		_, err := store.GetMessageCount(ctx, "user-001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count messages")
	})

	t.Run("zero count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages").
			WithArgs("user-001").
			WillReturnRows(rows)

		count, err := store.GetMessageCount(ctx, "user-001")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestDeleteExpiredMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		batchSize := 1000
		expectedDeleted := int64(500)

		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(batchSize).
			WillReturnResult(sqlmock.NewResult(0, expectedDeleted))

		deleted, err := store.DeleteExpiredMessages(ctx, batchSize)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeleted, deleted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invalid batch size", func(t *testing.T) {
		_, err := store.DeleteExpiredMessages(ctx, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch size must be between")

		_, err = store.DeleteExpiredMessages(ctx, 10001)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch size must be between")
	})

	t.Run("deletion failure", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnError(sql.ErrConnDone)

		_, err := store.DeleteExpiredMessages(ctx, 1000)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete expired messages")
	})

	t.Run("no messages to delete", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(1000).
			WillReturnResult(sqlmock.NewResult(0, 0))

		deleted, err := store.DeleteExpiredMessages(ctx, 1000)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})
}

func TestGetExpiredMessageCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		expectedCount := int64(123)

		rows := sqlmock.NewRows([]string{"count"}).AddRow(expectedCount)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		count, err := store.GetExpiredMessageCount(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages WHERE expires_at").
			WillReturnError(sql.ErrConnDone)

		_, err := store.GetExpiredMessageCount(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count expired messages")
	})
}

func TestDeleteMessagesByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		userID := "user-001"
		expectedDeleted := int64(10)

		mock.ExpectExec("DELETE FROM offline_messages WHERE user_id").
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, expectedDeleted))

		deleted, err := store.DeleteMessagesByUser(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedDeleted, deleted)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("deletion failure", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM offline_messages WHERE user_id").
			WithArgs("user-001").
			WillReturnError(sql.ErrConnDone)

		_, err := store.DeleteMessagesByUser(ctx, "user-001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete user messages")
	})
}

func TestGetOldestExpiredMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	store := &OfflineStore{db: db}
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		expectedTime := time.Now().Add(-8 * 24 * time.Hour)

		rows := sqlmock.NewRows([]string{"expires_at"}).AddRow(expectedTime)
		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		oldest, err := store.GetOldestExpiredMessage(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, oldest)
		assert.WithinDuration(t, expectedTime, *oldest, time.Second)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no expired messages", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"expires_at"}).AddRow(nil)
		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnRows(rows)

		oldest, err := store.GetOldestExpiredMessage(ctx)
		assert.NoError(t, err)
		assert.Nil(t, oldest)
	})

	t.Run("query failure", func(t *testing.T) {
		mock.ExpectQuery("SELECT MIN\\(expires_at\\) FROM offline_messages WHERE expires_at").
			WillReturnError(sql.ErrConnDone)

		_, err := store.GetOldestExpiredMessage(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get oldest expired message")
	})
}

func TestClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	store := &OfflineStore{db: db}
	mock.ExpectClose()
	err = store.Close()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
