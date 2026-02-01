package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalStore_MemoryMode(t *testing.T) {
	config := Config{
		RegionID:   "region-a",
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := NewLocalStore(config)
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.True(t, store.memoryMode)
	assert.Equal(t, "region-a", store.regionID)

	err = store.Close()
	assert.NoError(t, err)
}

func TestNewLocalStore_SQLiteMode(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := Config{
		DatabasePath: dbPath,
		RegionID:     "region-b",
		WALMode:      true,
		MemoryMode:   false,
		TTL:          7 * 24 * time.Hour,
	}

	store, err := NewLocalStore(config)
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.False(t, store.memoryMode)
	assert.Equal(t, "region-b", store.regionID)

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestLocalStore_Insert_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	message := createTestMessage("msg-1", "user-1", "sender-1")

	err := store.Insert(ctx, message)
	assert.NoError(t, err)

	// Verify message was inserted
	retrieved, err := store.GetMessageByID(ctx, "msg-1")
	require.NoError(t, err)
	assert.Equal(t, message.MsgID, retrieved.MsgID)
	assert.Equal(t, message.UserID, retrieved.UserID)
	assert.Equal(t, message.Content, retrieved.Content)
}

func TestLocalStore_Insert_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	message := createTestMessage("msg-1", "user-1", "sender-1")

	err := store.Insert(ctx, message)
	assert.NoError(t, err)

	// Verify message was inserted
	retrieved, err := store.GetMessageByID(ctx, "msg-1")
	require.NoError(t, err)
	assert.Equal(t, message.MsgID, retrieved.MsgID)
	assert.Equal(t, message.UserID, retrieved.UserID)
	assert.Equal(t, message.Content, retrieved.Content)
}

func TestLocalStore_Insert_Duplicate(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	message := createTestMessage("msg-1", "user-1", "sender-1")

	// Insert first time
	err := store.Insert(ctx, message)
	assert.NoError(t, err)

	// Insert duplicate should fail
	err = store.Insert(ctx, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLocalStore_BatchInsert_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	messages := []LocalMessage{
		createTestMessage("msg-1", "user-1", "sender-1"),
		createTestMessage("msg-2", "user-1", "sender-2"),
		createTestMessage("msg-3", "user-2", "sender-1"),
	}

	err := store.BatchInsert(ctx, messages)
	assert.NoError(t, err)

	// Verify all messages were inserted
	for _, msg := range messages {
		retrieved, err := store.GetMessageByID(ctx, msg.MsgID)
		require.NoError(t, err)
		assert.Equal(t, msg.MsgID, retrieved.MsgID)
	}
}

func TestLocalStore_BatchInsert_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	messages := []LocalMessage{
		createTestMessage("msg-1", "user-1", "sender-1"),
		createTestMessage("msg-2", "user-1", "sender-2"),
		createTestMessage("msg-3", "user-2", "sender-1"),
	}

	err := store.BatchInsert(ctx, messages)
	assert.NoError(t, err)

	// Verify all messages were inserted
	for _, msg := range messages {
		retrieved, err := store.GetMessageByID(ctx, msg.MsgID)
		require.NoError(t, err)
		assert.Equal(t, msg.MsgID, retrieved.MsgID)
	}
}

func TestLocalStore_GetMessages_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert test messages with different sequence numbers
	messages := []LocalMessage{
		createTestMessageWithSeq("msg-1", "user-1", "sender-1", 1),
		createTestMessageWithSeq("msg-2", "user-1", "sender-2", 2),
		createTestMessageWithSeq("msg-3", "user-1", "sender-3", 3),
		createTestMessageWithSeq("msg-4", "user-2", "sender-1", 1), // Different user
	}

	for _, msg := range messages {
		err := store.Insert(ctx, msg)
		require.NoError(t, err)
	}

	// Get messages for user-1
	retrieved, err := store.GetMessages(ctx, "user-1", 0, 10)
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)

	// Verify messages are sorted by sequence number
	assert.Equal(t, int64(1), retrieved[0].SequenceNumber)
	assert.Equal(t, int64(2), retrieved[1].SequenceNumber)
	assert.Equal(t, int64(3), retrieved[2].SequenceNumber)
}

func TestLocalStore_GetMessages_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert test messages with different sequence numbers
	messages := []LocalMessage{
		createTestMessageWithSeq("msg-1", "user-1", "sender-1", 1),
		createTestMessageWithSeq("msg-2", "user-1", "sender-2", 2),
		createTestMessageWithSeq("msg-3", "user-1", "sender-3", 3),
		createTestMessageWithSeq("msg-4", "user-2", "sender-1", 1), // Different user
	}

	err := store.BatchInsert(ctx, messages)
	require.NoError(t, err)

	// Get messages for user-1
	retrieved, err := store.GetMessages(ctx, "user-1", 0, 10)
	require.NoError(t, err)
	assert.Len(t, retrieved, 3)

	// Verify messages are sorted by sequence number
	assert.Equal(t, int64(1), retrieved[0].SequenceNumber)
	assert.Equal(t, int64(2), retrieved[1].SequenceNumber)
	assert.Equal(t, int64(3), retrieved[2].SequenceNumber)
}

func TestLocalStore_GetMessages_Pagination(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert 5 messages
	for i := 1; i <= 5; i++ {
		msg := createTestMessageWithSeq(
			fmt.Sprintf("msg-%d", i),
			"user-1",
			"sender-1",
			int64(i),
		)
		msg.ID = int64(i) // Set ID for cursor-based pagination
		err := store.Insert(ctx, msg)
		require.NoError(t, err)
	}

	// Get first page (limit 2)
	page1, err := store.GetMessages(ctx, "user-1", 0, 2)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page using cursor
	page2, err := store.GetMessages(ctx, "user-1", page1[len(page1)-1].ID, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify no overlap
	assert.NotEqual(t, page1[0].MsgID, page2[0].MsgID)
	assert.NotEqual(t, page1[1].MsgID, page2[1].MsgID)
}

func TestLocalStore_GetMessages_InvalidLimit(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Test invalid limits
	_, err := store.GetMessages(ctx, "user-1", 0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be between 1 and 100")

	_, err = store.GetMessages(ctx, "user-1", 0, 101)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be between 1 and 100")
}

func TestLocalStore_GetMessageByID_NotFound(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	_, err := store.GetMessageByID(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLocalStore_DetectConflict_NoConflict(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	remoteMessage := createTestMessage("msg-1", "user-1", "sender-1")

	// No local message exists, should not conflict
	conflict, err := store.DetectConflict(ctx, remoteMessage)
	require.NoError(t, err)
	assert.Nil(t, conflict)
}

func TestLocalStore_DetectConflict_WithConflict(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert local message
	localMessage := createTestMessage("msg-1", "user-1", "sender-1")
	localMessage.Version = 1
	localMessage.RegionID = "region-a"
	localMessage.GlobalID = "region-a-1000-1"

	err := store.Insert(ctx, localMessage)
	require.NoError(t, err)

	// Create conflicting remote message
	remoteMessage := createTestMessage("msg-1", "user-1", "sender-1")
	remoteMessage.Version = 2
	remoteMessage.RegionID = "region-b"
	remoteMessage.GlobalID = "region-b-1001-1" // Higher timestamp

	conflict, err := store.DetectConflict(ctx, remoteMessage)
	require.NoError(t, err)
	require.NotNil(t, conflict)

	assert.Equal(t, "msg-1", conflict.MessageID)
	assert.Equal(t, int64(1), conflict.LocalVersion)
	assert.Equal(t, int64(2), conflict.RemoteVersion)
	assert.Equal(t, "region-a", conflict.LocalRegion)
	assert.Equal(t, "region-b", conflict.RemoteRegion)
	assert.Equal(t, "remote_wins", conflict.Resolution) // Higher global ID wins
}

func TestLocalStore_RecordConflict_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	conflict := ConflictInfo{
		MessageID:     "msg-1",
		LocalVersion:  1,
		RemoteVersion: 2,
		LocalRegion:   "region-a",
		RemoteRegion:  "region-b",
		ConflictTime:  time.Now(),
		Resolution:    "remote_wins",
	}

	err := store.RecordConflict(ctx, conflict)
	assert.NoError(t, err)

	// Verify conflict was recorded
	conflicts, err := store.GetConflicts(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "msg-1", conflicts[0].MessageID)
}

func TestLocalStore_RecordConflict_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()
	conflict := ConflictInfo{
		MessageID:     "msg-1",
		LocalVersion:  1,
		RemoteVersion: 2,
		LocalRegion:   "region-a",
		RemoteRegion:  "region-b",
		ConflictTime:  time.Now(),
		Resolution:    "remote_wins",
	}

	err := store.RecordConflict(ctx, conflict)
	assert.NoError(t, err)

	// Verify conflict was recorded
	conflicts, err := store.GetConflicts(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "msg-1", conflicts[0].MessageID)
}

func TestLocalStore_DeleteExpiredMessages_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert expired and non-expired messages
	expiredMsg := createTestMessage("expired", "user-1", "sender-1")
	expiredMsg.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired

	validMsg := createTestMessage("valid", "user-1", "sender-1")
	validMsg.ExpiresAt = time.Now().Add(1 * time.Hour) // Not expired

	err := store.Insert(ctx, expiredMsg)
	require.NoError(t, err)
	err = store.Insert(ctx, validMsg)
	require.NoError(t, err)

	// Delete expired messages
	deleted, err := store.DeleteExpiredMessages(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify expired message was deleted
	_, err = store.GetMessageByID(ctx, "expired")
	assert.Error(t, err)

	// Verify valid message still exists
	_, err = store.GetMessageByID(ctx, "valid")
	assert.NoError(t, err)
}

func TestLocalStore_DeleteExpiredMessages_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert expired and non-expired messages
	expiredMsg := createTestMessage("expired", "user-1", "sender-1")
	expiredMsg.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired

	validMsg := createTestMessage("valid", "user-1", "sender-1")
	validMsg.ExpiresAt = time.Now().Add(1 * time.Hour) // Not expired

	err := store.BatchInsert(ctx, []LocalMessage{expiredMsg, validMsg})
	require.NoError(t, err)

	// Delete expired messages
	deleted, err := store.DeleteExpiredMessages(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify expired message was deleted
	_, err = store.GetMessageByID(ctx, "expired")
	assert.Error(t, err)

	// Verify valid message still exists
	_, err = store.GetMessageByID(ctx, "valid")
	assert.NoError(t, err)
}

func TestLocalStore_GetStats_Memory(t *testing.T) {
	store := createTestMemoryStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert some test data
	message := createTestMessage("msg-1", "user-1", "sender-1")
	err := store.Insert(ctx, message)
	require.NoError(t, err)

	conflict := ConflictInfo{
		MessageID:     "msg-1",
		LocalVersion:  1,
		RemoteVersion: 2,
		Resolution:    "remote_wins",
	}
	err = store.RecordConflict(ctx, conflict)
	require.NoError(t, err)

	// Get stats
	stats, err := store.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, "region-a", stats["region_id"])
	assert.Equal(t, true, stats["memory_mode"])
	assert.Equal(t, 1, stats["total_messages"])
	assert.Equal(t, 1, stats["total_conflicts"])
}

func TestLocalStore_GetStats_SQLite(t *testing.T) {
	store := createTestSQLiteStore(t, "region-a")
	defer store.Close()

	ctx := context.Background()

	// Insert some test data
	message := createTestMessage("msg-1", "user-1", "sender-1")
	err := store.Insert(ctx, message)
	require.NoError(t, err)

	conflict := ConflictInfo{
		MessageID:     "msg-1",
		LocalVersion:  1,
		RemoteVersion: 2,
		Resolution:    "remote_wins",
	}
	err = store.RecordConflict(ctx, conflict)
	require.NoError(t, err)

	// Get stats
	stats, err := store.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, "region-a", stats["region_id"])
	assert.Equal(t, false, stats["memory_mode"])
	assert.Equal(t, int64(1), stats["total_messages"])
	assert.Equal(t, int64(1), stats["total_conflicts"])
	assert.Equal(t, int64(0), stats["expired_messages"])
}

// Helper functions

func createTestMemoryStore(t *testing.T, regionID string) *LocalStore {
	config := Config{
		RegionID:   regionID,
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := NewLocalStore(config)
	require.NoError(t, err)
	return store
}

func createTestSQLiteStore(t *testing.T, regionID string) *LocalStore {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := Config{
		DatabasePath: dbPath,
		RegionID:     regionID,
		WALMode:      true,
		MemoryMode:   false,
		TTL:          7 * 24 * time.Hour,
	}

	store, err := NewLocalStore(config)
	require.NoError(t, err)
	return store
}

func createTestMessage(msgID, userID, senderID string) LocalMessage {
	return LocalMessage{
		MsgID:            msgID,
		UserID:           userID,
		SenderID:         senderID,
		ConversationID:   fmt.Sprintf("conv-%s-%s", userID, senderID),
		ConversationType: "private",
		Content:          fmt.Sprintf("Test message %s", msgID),
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		Metadata:         map[string]string{"test": "value"},
		RegionID:         "region-a",
		GlobalID:         fmt.Sprintf("region-a-%d-1", time.Now().UnixMilli()),
		Version:          1,
	}
}

func createTestMessageWithSeq(msgID, userID, senderID string, seq int64) LocalMessage {
	msg := createTestMessage(msgID, userID, senderID)
	msg.SequenceNumber = seq
	return msg
}
