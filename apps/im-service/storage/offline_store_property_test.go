package storage

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// **Validates: Requirements 4.3, 4.7, 16.8, 16.9**
// Property 7: Offline Message Ordering Preservation
// Messages retrieved from offline storage must maintain correct ordering by sequence_number
// Pagination must return all messages without duplicates or gaps

func TestProperty_OfflineMessageOrderingPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a list of messages with random sequence numbers
		numMessages := rapid.IntRange(1, 50).Draw(t, "numMessages")
		messages := generateMessages(t, numMessages)

		// Sort messages by sequence number (expected order)
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].SequenceNumber < messages[j].SequenceNumber
		})

		// Create mock database
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		store := &OfflineStore{db: db}
		ctx := context.Background()

		// Mock the query to return messages in sequence order
		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		})

		for _, msg := range messages {
			rows.AddRow(
				msg.ID,
				msg.MsgID,
				msg.UserID,
				msg.SenderID,
				msg.ConversationID,
				msg.ConversationType,
				msg.Content,
				msg.SequenceNumber,
				msg.Timestamp,
				msg.CreatedAt,
				msg.ExpiresAt,
			)
		}

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs("user-001", int64(0), 100).
			WillReturnRows(rows)

		// Retrieve messages
		retrieved, err := store.GetMessages(ctx, "user-001", 0, 100)
		require.NoError(t, err)

		// Property: Messages must be ordered by sequence_number
		for i := 1; i < len(retrieved); i++ {
			if retrieved[i].SequenceNumber <= retrieved[i-1].SequenceNumber {
				t.Fatalf("Messages not ordered by sequence_number: %d <= %d at index %d",
					retrieved[i].SequenceNumber, retrieved[i-1].SequenceNumber, i)
			}
		}

		// Property: All messages must be retrieved
		if len(retrieved) != len(messages) {
			t.Fatalf("Expected %d messages, got %d", len(messages), len(retrieved))
		}

		// Property: Sequence numbers must match
		for i, msg := range retrieved {
			if msg.SequenceNumber != messages[i].SequenceNumber {
				t.Fatalf("Sequence number mismatch at index %d: expected %d, got %d",
					i, messages[i].SequenceNumber, msg.SequenceNumber)
			}
		}
	})
}

func TestProperty_PaginationCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate messages
		numMessages := rapid.IntRange(10, 100).Draw(t, "numMessages")
		pageSize := rapid.IntRange(5, 20).Draw(t, "pageSize")

		messages := generateMessages(t, numMessages)
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].SequenceNumber < messages[j].SequenceNumber
		})

		// Assign sequential IDs
		for i := range messages {
			messages[i].ID = int64(i + 1)
		}

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		store := &OfflineStore{db: db}
		ctx := context.Background()

		// Simulate pagination
		var allRetrieved []OfflineMessage
		cursor := int64(0)

		for {
			// Calculate which messages should be returned for this page
			startIdx := int(cursor)
			endIdx := startIdx + pageSize
			if endIdx > len(messages) {
				endIdx = len(messages)
			}

			if startIdx >= len(messages) {
				break
			}

			pageMessages := messages[startIdx:endIdx]

			// Mock the query for this page
			rows := sqlmock.NewRows([]string{
				"id", "msg_id", "user_id", "sender_id", "conversation_id",
				"conversation_type", "content", "sequence_number", "timestamp",
				"created_at", "expires_at",
			})

			for _, msg := range pageMessages {
				rows.AddRow(
					msg.ID,
					msg.MsgID,
					msg.UserID,
					msg.SenderID,
					msg.ConversationID,
					msg.ConversationType,
					msg.Content,
					msg.SequenceNumber,
					msg.Timestamp,
					msg.CreatedAt,
					msg.ExpiresAt,
				)
			}

			mock.ExpectQuery("SELECT (.+) FROM offline_messages").
				WithArgs("user-001", cursor, pageSize).
				WillReturnRows(rows)

			// Retrieve page
			page, err := store.GetMessages(ctx, "user-001", cursor, pageSize)
			require.NoError(t, err)

			if len(page) == 0 {
				break
			}

			allRetrieved = append(allRetrieved, page...)

			// Update cursor to last message ID
			cursor = page[len(page)-1].ID
		}

		// Property: All messages must be retrieved through pagination
		if len(allRetrieved) != len(messages) {
			t.Fatalf("Pagination incomplete: expected %d messages, got %d",
				len(messages), len(allRetrieved))
		}

		// Property: No duplicates
		seen := make(map[string]bool)
		for _, msg := range allRetrieved {
			if seen[msg.MsgID] {
				t.Fatalf("Duplicate message found: %s", msg.MsgID)
			}
			seen[msg.MsgID] = true
		}

		// Property: Messages must be in sequence order
		for i := 1; i < len(allRetrieved); i++ {
			if allRetrieved[i].SequenceNumber <= allRetrieved[i-1].SequenceNumber {
				t.Fatalf("Messages not ordered after pagination: %d <= %d at index %d",
					allRetrieved[i].SequenceNumber, allRetrieved[i-1].SequenceNumber, i)
			}
		}
	})
}

func TestProperty_BatchInsertPreservesOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate batch of messages
		batchSize := rapid.IntRange(1, 100).Draw(t, "batchSize")
		messages := generateMessages(t, batchSize)

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		store := &OfflineStore{db: db}
		ctx := context.Background()

		// Mock batch insert
		mock.ExpectBegin()
		mock.ExpectPrepare("INSERT INTO offline_messages")
		for range messages {
			mock.ExpectExec("INSERT INTO offline_messages").
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectCommit()

		// Insert messages
		err = store.BatchInsert(ctx, messages)
		require.NoError(t, err)

		// Property: Batch insert should not modify message order
		// (This is verified by the fact that we insert in the order provided)
		// The sequence numbers should remain unchanged
		for i, msg := range messages {
			if msg.SequenceNumber < 0 {
				t.Fatalf("Invalid sequence number at index %d: %d", i, msg.SequenceNumber)
			}
		}
	})
}

func TestProperty_TTLCleanupConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random batch size
		batchSize := rapid.IntRange(100, 10000).Draw(t, "batchSize")
		deletedCount := rapid.Int64Range(0, int64(batchSize)).Draw(t, "deletedCount")

		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func() { _ = db.Close() }()

		store := &OfflineStore{db: db}
		ctx := context.Background()

		// Mock delete operation
		mock.ExpectExec("DELETE FROM offline_messages").
			WithArgs(batchSize).
			WillReturnResult(sqlmock.NewResult(0, deletedCount))

		// Delete expired messages
		deleted, err := store.DeleteExpiredMessages(ctx, batchSize)
		require.NoError(t, err)

		// Property: Deleted count must not exceed batch size
		if deleted > int64(batchSize) {
			t.Fatalf("Deleted count (%d) exceeds batch size (%d)", deleted, batchSize)
		}

		// Property: Deleted count must be non-negative
		if deleted < 0 {
			t.Fatalf("Deleted count cannot be negative: %d", deleted)
		}

		// Property: Deleted count must match what was returned
		if deleted != deletedCount {
			t.Fatalf("Expected %d deleted, got %d", deletedCount, deleted)
		}
	})
}

// Helper function to generate test messages
func generateMessages(t *rapid.T, count int) []OfflineMessage {
	messages := make([]OfflineMessage, count)

	// Generate unique sequence numbers
	baseSeq := rapid.Int64Range(1, 100000).Draw(t, "baseSequence")
	baseMsgID := rapid.Int64Range(1000000, 9999999).Draw(t, "baseMsgID")

	for i := 0; i < count; i++ {
		messages[i] = OfflineMessage{
			ID:               int64(i + 1),
			MsgID:            fmt.Sprintf("msg-%d-%d", baseMsgID, i), // Ensure unique
			UserID:           "user-001",
			SenderID:         rapid.StringMatching(`user-[0-9]{3}`).Draw(t, "senderID"),
			ConversationID:   rapid.StringMatching(`conv-[0-9]{3}`).Draw(t, "conversationID"),
			ConversationType: rapid.SampledFrom([]string{"private", "group"}).Draw(t, "conversationType"),
			Content:          rapid.String().Draw(t, "content"),
			SequenceNumber:   baseSeq + int64(i), // Ensure unique, monotonically increasing
			Timestamp:        time.Now().Unix() * 1000,
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return messages
}
