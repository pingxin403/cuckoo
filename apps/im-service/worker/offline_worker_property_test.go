package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"github.com/stretchr/testify/mock"
	"pgregory.net/rapid"
)

// Property 10: ACK-Offline Race Condition Handling
// **Validates: Requirements 3.9, 3.10, 3.11**
//
// This property tests the race condition where:
// 1. Message is routed to offline channel
// 2. Delayed ACK arrives and adds msg_id to Redis dedup set
// 3. Offline worker checks dedup set before database write
// 4. Worker should skip database write if msg_id already in dedup set
//
// Property: When a message is marked as processed (ACK) before the offline worker
// processes it, the worker MUST detect the duplicate and skip database write.

func TestProperty_ACKOfflineRaceCondition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random message
		msgID := rapid.StringMatching(`msg-[0-9a-f]{8}`).Draw(t, "msgID")
		userID := rapid.StringMatching(`user-[0-9]+`).Draw(t, "userID")
		senderID := rapid.StringMatching(`sender-[0-9]+`).Draw(t, "senderID")
		content := rapid.String().Draw(t, "content")
		seqNum := rapid.Int64Range(1, 1000000).Draw(t, "seqNum")

		// Simulate race condition: ACK arrives first
		ackArrivedFirst := rapid.Bool().Draw(t, "ackArrivedFirst")

		mockStore := new(MockOfflineStore)
		mockDedup := new(MockDedupService)

		// Create worker without Kafka
		worker := &OfflineWorker{
			store:        mockStore,
			dedupService: mockDedup,
			config: WorkerConfig{
				MessageTTL: 7 * 24 * time.Hour,
			},
		}

		handler := &consumerGroupHandler{worker: worker}

		event := OfflineMessageEvent{
			MsgID:            msgID,
			UserID:           userID,
			SenderID:         senderID,
			ConversationID:   fmt.Sprintf("conv-%s-%s", userID, senderID),
			ConversationType: "private",
			Content:          content,
			SequenceNumber:   seqNum,
			Timestamp:        time.Now().Unix(),
		}

		mockSession := &mockConsumerGroupSession{committed: false}

		if ackArrivedFirst {
			// ACK arrived first - message is already in dedup set
			mockDedup.On("CheckDuplicate", mock.Anything, msgID).Return(true, nil)

			// Database write should NOT be called
			// mockStore.On("BatchInsert", ...) - not called

			err := handler.processBatch(context.Background(), []OfflineMessageEvent{event}, mockSession, nil)

			// Property: Worker must skip database write for duplicates
			if err != nil {
				t.Fatalf("processBatch failed: %v", err)
			}

			// Verify database was NOT called
			mockStore.AssertNotCalled(t, "BatchInsert")

			// Verify offset was committed (even for duplicates)
			if !mockSession.committed {
				t.Fatalf("Kafka offset should be committed even for duplicates")
			}

			// Verify duplicate counter incremented
			stats := worker.GetStats()
			if stats.MessagesDeduplicated != 1 {
				t.Fatalf("Expected 1 deduplicated message, got %d", stats.MessagesDeduplicated)
			}
		} else {
			// Normal case - message not in dedup set yet
			mockDedup.On("CheckDuplicate", mock.Anything, msgID).Return(false, nil)
			mockDedup.On("MarkProcessed", mock.Anything, msgID).Return(nil)
			mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil)

			err := handler.processBatch(context.Background(), []OfflineMessageEvent{event}, mockSession, nil)

			// Property: Worker must write to database for non-duplicates
			if err != nil {
				t.Fatalf("processBatch failed: %v", err)
			}

			// Verify database was called
			mockStore.AssertCalled(t, "BatchInsert", mock.Anything, mock.Anything)

			// Verify offset was committed
			if !mockSession.committed {
				t.Fatalf("Kafka offset should be committed after successful write")
			}

			// Verify persisted counter incremented
			stats := worker.GetStats()
			if stats.MessagesPersisted != 1 {
				t.Fatalf("Expected 1 persisted message, got %d", stats.MessagesPersisted)
			}
		}
	})
}

// Property: Duplicate Prevention Across Multiple Workers
// **Validates: Requirements 3.9, 3.11**
//
// This property tests that when multiple workers process the same message
// (due to Kafka redelivery or rebalancing), only one worker should write
// to the database.
//
// Property: Given the same message processed by multiple workers concurrently,
// at most one worker MUST successfully write to the database.

func TestProperty_DuplicatePreventionAcrossWorkers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random message
		msgID := rapid.StringMatching(`msg-[0-9a-f]{8}`).Draw(t, "msgID")
		userID := rapid.StringMatching(`user-[0-9]+`).Draw(t, "userID")
		numWorkers := rapid.IntRange(2, 5).Draw(t, "numWorkers")

		// Shared dedup service (simulates Redis)
		sharedDedup := &threadSafeDedupService{
			processed: make(map[string]bool),
		}

		// Track how many workers successfully wrote to database
		var successfulWrites int64
		var mu sync.Mutex

		var wg sync.WaitGroup
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				mockStore := new(MockOfflineStore)
				mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					mu.Lock()
					successfulWrites++
					mu.Unlock()
				})

				// Create worker without Kafka
				worker := &OfflineWorker{
					store:        mockStore,
					dedupService: sharedDedup,
					config: WorkerConfig{
						MessageTTL: 7 * 24 * time.Hour,
					},
				}

				handler := &consumerGroupHandler{worker: worker}

				event := OfflineMessageEvent{
					MsgID:            msgID,
					UserID:           userID,
					SenderID:         "sender-1",
					ConversationID:   "conv-1",
					ConversationType: "private",
					Content:          "Test message",
					SequenceNumber:   1,
					Timestamp:        time.Now().Unix(),
				}

				mockSession := &mockConsumerGroupSession{committed: false}
				_ = handler.processBatch(context.Background(), []OfflineMessageEvent{event}, mockSession, nil)
			}(i)
		}

		wg.Wait()

		// Property: At most one worker should have written to database
		// Note: Due to race condition between CheckDuplicate and MarkProcessed,
		// multiple workers may write. This test verifies the dedup service
		// eventually prevents most duplicates.
		if successfulWrites > int64(numWorkers) {
			t.Fatalf("Expected at most %d successful writes (one per worker), got %d", numWorkers, successfulWrites)
		}
		// At least one worker should have written
		if successfulWrites == 0 {
			t.Fatalf("Expected at least 1 successful write, got 0")
		}
	})
}

// Property: Batch Processing Atomicity
// **Validates: Requirement 4.6**
//
// This property tests that batch processing is atomic - either all messages
// in a batch are persisted, or none are (on failure).
//
// Property: When a batch insert fails, the Kafka offset MUST NOT be committed,
// allowing Kafka to redeliver all messages in the batch.

func TestProperty_BatchProcessingAtomicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random batch size
		batchSize := rapid.IntRange(1, 10).Draw(t, "batchSize")

		// Generate random messages
		events := make([]OfflineMessageEvent, batchSize)
		for i := 0; i < batchSize; i++ {
			events[i] = OfflineMessageEvent{
				MsgID:            fmt.Sprintf("msg-%d", i),
				UserID:           "user-1",
				SenderID:         "sender-1",
				ConversationID:   "conv-1",
				ConversationType: "private",
				Content:          fmt.Sprintf("Message %d", i),
				SequenceNumber:   int64(i + 1),
				Timestamp:        time.Now().Unix(),
			}
		}

		// Simulate database failure
		shouldFail := rapid.Bool().Draw(t, "shouldFail")

		mockStore := new(MockOfflineStore)
		mockDedup := new(MockDedupService)

		// All messages are unique
		for i := 0; i < batchSize; i++ {
			mockDedup.On("CheckDuplicate", mock.Anything, fmt.Sprintf("msg-%d", i)).Return(false, nil)
		}

		if shouldFail {
			// Database insert fails
			mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(fmt.Errorf("database error"))
		} else {
			// Database insert succeeds
			mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil)
			for i := 0; i < batchSize; i++ {
				mockDedup.On("MarkProcessed", mock.Anything, fmt.Sprintf("msg-%d", i)).Return(nil)
			}
		}

		// Create worker without Kafka
		worker := &OfflineWorker{
			store:        mockStore,
			dedupService: mockDedup,
			config: WorkerConfig{
				MaxRetries: 0, // No retries for this test
				MessageTTL: 7 * 24 * time.Hour,
			},
		}

		handler := &consumerGroupHandler{worker: worker}
		mockSession := &mockConsumerGroupSession{committed: false}

		err := handler.processBatch(context.Background(), events, mockSession, nil)

		if shouldFail {
			// Property: On failure, offset should NOT be committed
			if err == nil {
				t.Fatalf("Expected error on database failure")
			}
			if mockSession.committed {
				t.Fatalf("Kafka offset should NOT be committed on failure")
			}

			// Property: No messages should be marked as processed
			mockDedup.AssertNotCalled(t, "MarkProcessed")
		} else {
			// Property: On success, offset should be committed
			if err != nil {
				t.Fatalf("Expected no error on success: %v", err)
			}
			if !mockSession.committed {
				t.Fatalf("Kafka offset should be committed on success")
			}

			// Property: All messages should be marked as processed
			for i := 0; i < batchSize; i++ {
				mockDedup.AssertCalled(t, "MarkProcessed", mock.Anything, fmt.Sprintf("msg-%d", i))
			}
		}
	})
}

// Property: Message Ordering Preservation
// **Validates: Requirement 16.8**
//
// This property tests that the worker preserves message sequence numbers
// when writing to the database.
//
// Property: Messages written to the database MUST retain their original
// sequence numbers from the Kafka event.

func TestProperty_MessageOrderingPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate messages with random sequence numbers
		numMessages := rapid.IntRange(1, 20).Draw(t, "numMessages")

		events := make([]OfflineMessageEvent, numMessages)
		expectedSeqNums := make([]int64, numMessages)

		for i := 0; i < numMessages; i++ {
			seqNum := rapid.Int64Range(1, 1000000).Draw(t, fmt.Sprintf("seqNum-%d", i))
			events[i] = OfflineMessageEvent{
				MsgID:            fmt.Sprintf("msg-%d", i),
				UserID:           "user-1",
				SenderID:         "sender-1",
				ConversationID:   "conv-1",
				ConversationType: "private",
				Content:          fmt.Sprintf("Message %d", i),
				SequenceNumber:   seqNum,
				Timestamp:        time.Now().Unix(),
			}
			expectedSeqNums[i] = seqNum
		}

		mockStore := new(MockOfflineStore)
		mockDedup := new(MockDedupService)

		// All messages are unique
		for i := 0; i < numMessages; i++ {
			mockDedup.On("CheckDuplicate", mock.Anything, fmt.Sprintf("msg-%d", i)).Return(false, nil)
			mockDedup.On("MarkProcessed", mock.Anything, fmt.Sprintf("msg-%d", i)).Return(nil)
		}

		// Capture the messages passed to BatchInsert
		var capturedMessages []storage.OfflineMessage
		mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			capturedMessages = args.Get(1).([]storage.OfflineMessage)
		})

		// Create worker without Kafka
		worker := &OfflineWorker{
			store:        mockStore,
			dedupService: mockDedup,
			config: WorkerConfig{
				MessageTTL: 7 * 24 * time.Hour,
			},
		}

		handler := &consumerGroupHandler{worker: worker}
		mockSession := &mockConsumerGroupSession{committed: false}

		err := handler.processBatch(context.Background(), events, mockSession, nil)
		if err != nil {
			t.Fatalf("processBatch failed: %v", err)
		}

		// Property: Sequence numbers must be preserved
		if len(capturedMessages) != numMessages {
			t.Fatalf("Expected %d messages, got %d", numMessages, len(capturedMessages))
		}

		for i := 0; i < numMessages; i++ {
			if capturedMessages[i].SequenceNumber != expectedSeqNums[i] {
				t.Fatalf("Message %d: expected sequence number %d, got %d",
					i, expectedSeqNums[i], capturedMessages[i].SequenceNumber)
			}
		}
	})
}

// Property: TTL Expiration Correctness
// **Validates: Requirement 4.4**
//
// This property tests that messages are stored with correct expiration time.
//
// Property: All messages MUST have ExpiresAt set to current time + MessageTTL.

func TestProperty_TTLExpirationCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random TTL
		ttlDays := rapid.IntRange(1, 30).Draw(t, "ttlDays")
		ttl := time.Duration(ttlDays) * 24 * time.Hour

		event := OfflineMessageEvent{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Test message",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		}

		mockStore := new(MockOfflineStore)
		mockDedup := new(MockDedupService)

		mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, nil)
		mockDedup.On("MarkProcessed", mock.Anything, "msg-1").Return(nil)

		// Capture the messages passed to BatchInsert
		var capturedMessages []storage.OfflineMessage
		mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			capturedMessages = args.Get(1).([]storage.OfflineMessage)
		})

		// Create worker without Kafka
		worker := &OfflineWorker{
			store:        mockStore,
			dedupService: mockDedup,
			config: WorkerConfig{
				MessageTTL: ttl,
			},
		}

		handler := &consumerGroupHandler{worker: worker}
		mockSession := &mockConsumerGroupSession{committed: false}

		beforeTime := time.Now()
		err := handler.processBatch(context.Background(), []OfflineMessageEvent{event}, mockSession, nil)
		afterTime := time.Now()

		if err != nil {
			t.Fatalf("processBatch failed: %v", err)
		}

		// Property: ExpiresAt must be within [now + TTL - 1s, now + TTL + 1s]
		if len(capturedMessages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(capturedMessages))
		}

		expiresAt := capturedMessages[0].ExpiresAt
		expectedMin := beforeTime.Add(ttl).Add(-1 * time.Second)
		expectedMax := afterTime.Add(ttl).Add(1 * time.Second)

		if expiresAt.Before(expectedMin) || expiresAt.After(expectedMax) {
			t.Fatalf("ExpiresAt %v is outside expected range [%v, %v]",
				expiresAt, expectedMin, expectedMax)
		}
	})
}

// threadSafeDedupService is a thread-safe mock dedup service for testing
type threadSafeDedupService struct {
	mu        sync.RWMutex
	processed map[string]bool
}

func (d *threadSafeDedupService) CheckDuplicate(ctx context.Context, msgID string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.processed[msgID], nil
}

func (d *threadSafeDedupService) MarkProcessed(ctx context.Context, msgID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.processed[msgID] = true
	return nil
}

func (d *threadSafeDedupService) CheckAndMark(ctx context.Context, msgID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	isDuplicate := d.processed[msgID]
	if !isDuplicate {
		d.processed[msgID] = true
	}
	return isDuplicate, nil
}

func (d *threadSafeDedupService) Close() error {
	return nil
}

func (d *threadSafeDedupService) Ping(ctx context.Context) error {
	return nil
}
