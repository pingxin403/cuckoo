package storage_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
)

// ExampleLocalStore_basic demonstrates basic usage of LocalStore
func ExampleLocalStore_basic() {
	// Create a memory-based store for this example
	config := storage.Config{
		RegionID:   "region-a",
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create a test message
	message := storage.LocalMessage{
		MsgID:            "msg-123",
		UserID:           "user-456",
		SenderID:         "user-789",
		ConversationID:   "conv-abc",
		ConversationType: "private",
		Content:          "Hello, world!",
		SequenceNumber:   1,
		Timestamp:        time.Now().UnixMilli(),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		RegionID:         "region-a",
		GlobalID:         "region-a-1234567890-1",
		Version:          1,
	}

	// Insert the message
	err = store.Insert(ctx, message)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the message
	retrieved, err := store.GetMessageByID(ctx, "msg-123")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Retrieved message: %s\n", retrieved.Content)
	// Output: Retrieved message: Hello, world!
}

// ExampleLocalStore_conflictDetection demonstrates conflict detection
func ExampleLocalStore_conflictDetection() {
	config := storage.Config{
		RegionID:   "region-a",
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert a local message
	localMessage := storage.LocalMessage{
		MsgID:     "msg-123",
		UserID:    "user-456",
		Content:   "Local version",
		Version:   1,
		RegionID:  "region-a",
		GlobalID:  "region-a-1000-1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	err = store.Insert(ctx, localMessage)
	if err != nil {
		log.Fatal(err)
	}

	// Simulate a conflicting remote message
	remoteMessage := storage.LocalMessage{
		MsgID:     "msg-123",
		UserID:    "user-456",
		Content:   "Remote version",
		Version:   2,
		RegionID:  "region-b",
		GlobalID:  "region-b-1001-1", // Higher timestamp
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	// Detect conflict
	conflict, err := store.DetectConflict(ctx, remoteMessage)
	if err != nil {
		log.Fatal(err)
	}

	if conflict != nil {
		fmt.Printf("Conflict detected: %s\n", conflict.Resolution)

		// Record the conflict
		err = store.RecordConflict(ctx, *conflict)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Output: Conflict detected: remote_wins
}

// ExampleLocalStore_batchOperations demonstrates batch operations
func ExampleLocalStore_batchOperations() {
	config := storage.Config{
		RegionID:   "region-a",
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create multiple messages
	messages := []storage.LocalMessage{
		{
			MsgID:          "msg-1",
			UserID:         "user-1",
			Content:        "First message",
			SequenceNumber: 1,
			Timestamp:      time.Now().UnixMilli(),
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
			RegionID:       "region-a",
			GlobalID:       "region-a-1000-1",
			Version:        1,
		},
		{
			MsgID:          "msg-2",
			UserID:         "user-1",
			Content:        "Second message",
			SequenceNumber: 2,
			Timestamp:      time.Now().UnixMilli(),
			ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
			RegionID:       "region-a",
			GlobalID:       "region-a-1001-1",
			Version:        1,
		},
	}

	// Batch insert
	err = store.BatchInsert(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve messages for user
	retrieved, err := store.GetMessages(ctx, "user-1", 0, 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Retrieved %d messages\n", len(retrieved))
	// Output: Retrieved 2 messages
}

// ExampleLocalStore_statistics demonstrates getting storage statistics
func ExampleLocalStore_statistics() {
	config := storage.Config{
		RegionID:   "region-a",
		MemoryMode: true,
		TTL:        7 * 24 * time.Hour,
	}

	store, err := storage.NewLocalStore(config)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert a test message
	message := storage.LocalMessage{
		MsgID:     "msg-1",
		UserID:    "user-1",
		Content:   "Test message",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		RegionID:  "region-a",
		GlobalID:  "region-a-1000-1",
		Version:   1,
	}

	err = store.Insert(ctx, message)
	if err != nil {
		log.Fatal(err)
	}

	// Get statistics
	stats, err := store.GetStats(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Region: %s, Messages: %v\n", stats["region_id"], stats["total_messages"])
	// Output: Region: region-a, Messages: 1
}
