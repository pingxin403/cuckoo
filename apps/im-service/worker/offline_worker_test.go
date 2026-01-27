package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/pingxin403/cuckoo/apps/im-service/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOfflineStore is a mock implementation of storage.OfflineStore
type MockOfflineStore struct {
	mock.Mock
}

func (m *MockOfflineStore) BatchInsert(ctx context.Context, messages []storage.OfflineMessage) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *MockOfflineStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockDedupService is a mock implementation of dedup.DedupService
type MockDedupService struct {
	mock.Mock
}

func (m *MockDedupService) CheckDuplicate(ctx context.Context, msgID string) (bool, error) {
	args := m.Called(ctx, msgID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDedupService) MarkProcessed(ctx context.Context, msgID string) error {
	args := m.Called(ctx, msgID)
	return args.Error(0)
}

func (m *MockDedupService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDedupService) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestNewOfflineWorker tests worker creation
func TestNewOfflineWorker(t *testing.T) {
	tests := []struct {
		name        string
		config      WorkerConfig
		expectError bool
	}{
		{
			name: "valid config with defaults",
			config: WorkerConfig{
				KafkaBrokers:  []string{"localhost:9092"},
				ConsumerGroup: "test-group",
				Topic:         "test-topic",
			},
			expectError: false,
		},
		{
			name: "valid config with custom values",
			config: WorkerConfig{
				KafkaBrokers:  []string{"localhost:9092"},
				ConsumerGroup: "test-group",
				Topic:         "test-topic",
				BatchSize:     50,
				BatchTimeout:  3 * time.Second,
				MaxRetries:    3,
				MessageTTL:    24 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "invalid kafka brokers",
			config: WorkerConfig{
				KafkaBrokers:  []string{},
				ConsumerGroup: "test-group",
				Topic:         "test-topic",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockOfflineStore)
			mockDedup := new(MockDedupService)

			worker, err := NewOfflineWorker(tt.config, mockStore, mockDedup)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, worker)
			} else {
				// Skip tests that require Kafka connection
				// In production, we'd use testcontainers or mock the consumer group
				if err != nil {
					t.Skipf("Skipping test due to Kafka connection requirement: %v", err)
				} else {
					assert.NotNil(t, worker)
					assert.Equal(t, tt.config.Topic, worker.config.Topic)
					// Clean up
					_ = worker.Stop()
				}
			}
		})
	}
}

// TestWorkerStats tests statistics tracking
func TestWorkerStats(t *testing.T) {
	// Create worker without Kafka for testing stats
	worker := &OfflineWorker{
		config: WorkerConfig{
			BatchSize:    100,
			BatchTimeout: 5 * time.Second,
			MessageTTL:   7 * 24 * time.Hour,
		},
	}

	// Test initial stats
	stats := worker.GetStats()
	assert.Equal(t, int64(0), stats.MessagesProcessed)
	assert.Equal(t, int64(0), stats.MessagesDeduplicated)
	assert.Equal(t, int64(0), stats.MessagesPersisted)
	assert.Equal(t, int64(0), stats.BatchWrites)
	assert.Equal(t, int64(0), stats.Errors)
	assert.Equal(t, 0.0, stats.AvgBatchSize)

	// Increment counters
	worker.incrementProcessed()
	worker.incrementProcessed()
	worker.incrementDeduplicated()
	worker.incrementPersisted(5)
	worker.incrementBatchWrites()
	worker.incrementErrors()

	// Test updated stats
	stats = worker.GetStats()
	assert.Equal(t, int64(2), stats.MessagesProcessed)
	assert.Equal(t, int64(1), stats.MessagesDeduplicated)
	assert.Equal(t, int64(5), stats.MessagesPersisted)
	assert.Equal(t, int64(1), stats.BatchWrites)
	assert.Equal(t, int64(1), stats.Errors)
	assert.Equal(t, 5.0, stats.AvgBatchSize)
}

// TestProcessBatch_AllUnique tests batch processing with all unique messages
func TestProcessBatch_AllUnique(t *testing.T) {
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

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
		{
			MsgID:            "msg-2",
			UserID:           "user-1",
			SenderID:         "sender-2",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "World",
			SequenceNumber:   2,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service - all messages are unique
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, nil)
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-2").Return(false, nil)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-1").Return(nil)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-2").Return(nil)

	// Mock store - successful batch insert
	mockStore.On("BatchInsert", mock.Anything, mock.MatchedBy(func(msgs []storage.OfflineMessage) bool {
		return len(msgs) == 2
	})).Return(nil)

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, mockSession.committed)
	mockDedup.AssertExpectations(t)
	mockStore.AssertExpectations(t)

	// Check stats
	stats := worker.GetStats()
	assert.Equal(t, int64(2), stats.MessagesPersisted)
	assert.Equal(t, int64(1), stats.BatchWrites)
	assert.Equal(t, int64(0), stats.MessagesDeduplicated)
}

// TestProcessBatch_WithDuplicates tests batch processing with duplicate messages
func TestProcessBatch_WithDuplicates(t *testing.T) {
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

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
		{
			MsgID:            "msg-2",
			UserID:           "user-1",
			SenderID:         "sender-2",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "World",
			SequenceNumber:   2,
			Timestamp:        time.Now().Unix(),
		},
		{
			MsgID:            "msg-3",
			UserID:           "user-1",
			SenderID:         "sender-3",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Duplicate",
			SequenceNumber:   3,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service - msg-2 is duplicate
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, nil)
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-2").Return(true, nil) // Duplicate!
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-3").Return(false, nil)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-1").Return(nil)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-3").Return(nil)

	// Mock store - only 2 messages should be inserted (msg-1 and msg-3)
	mockStore.On("BatchInsert", mock.Anything, mock.MatchedBy(func(msgs []storage.OfflineMessage) bool {
		return len(msgs) == 2 && msgs[0].MsgID == "msg-1" && msgs[1].MsgID == "msg-3"
	})).Return(nil)

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, mockSession.committed)
	mockDedup.AssertExpectations(t)
	mockStore.AssertExpectations(t)

	// Check stats
	stats := worker.GetStats()
	assert.Equal(t, int64(2), stats.MessagesPersisted)
	assert.Equal(t, int64(1), stats.MessagesDeduplicated)
	assert.Equal(t, int64(1), stats.BatchWrites)
}

// TestProcessBatch_AllDuplicates tests batch processing when all messages are duplicates
func TestProcessBatch_AllDuplicates(t *testing.T) {
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

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service - all messages are duplicates
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(true, nil)

	// Mock store should NOT be called
	// mockStore.On("BatchInsert", ...) - not called

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, mockSession.committed)
	mockDedup.AssertExpectations(t)
	mockStore.AssertNotCalled(t, "BatchInsert")

	// Check stats
	stats := worker.GetStats()
	assert.Equal(t, int64(0), stats.MessagesPersisted)
	assert.Equal(t, int64(1), stats.MessagesDeduplicated)
	assert.Equal(t, int64(0), stats.BatchWrites)
}

// TestProcessBatch_DatabaseError tests batch processing with database error
func TestProcessBatch_DatabaseError(t *testing.T) {
	mockStore := new(MockOfflineStore)
	mockDedup := new(MockDedupService)

	// Create worker without Kafka
	worker := &OfflineWorker{
		store:        mockStore,
		dedupService: mockDedup,
		config: WorkerConfig{
			MaxRetries:   2,
			RetryBackoff: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond},
			MessageTTL:   7 * 24 * time.Hour,
		},
	}

	handler := &consumerGroupHandler{worker: worker}

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, nil)

	// Mock store - return error on all attempts
	dbError := errors.New("database connection failed")
	mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(dbError)

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert batch after")
	assert.False(t, mockSession.committed) // Should NOT commit on error
	mockDedup.AssertExpectations(t)
	mockStore.AssertExpectations(t)

	// Should have tried MaxRetries + 1 times (initial + 2 retries = 3 total)
	mockStore.AssertNumberOfCalls(t, "BatchInsert", 3)
}

// TestProcessBatch_DedupError tests batch processing with deduplication error
func TestProcessBatch_DedupError(t *testing.T) {
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

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service - return error on check
	dedupError := errors.New("redis connection failed")
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, dedupError)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-1").Return(nil)

	// Mock store - should still be called (dedup error doesn't fail batch)
	mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil)

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err) // Dedup error doesn't fail the batch
	assert.True(t, mockSession.committed)
	mockDedup.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}

// TestProcessBatch_EmptyBatch tests batch processing with empty batch
func TestProcessBatch_EmptyBatch(t *testing.T) {
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

	// Empty batch
	events := []OfflineMessageEvent{}

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err)
	assert.False(t, mockSession.committed) // Should not commit for empty batch
	mockDedup.AssertNotCalled(t, "CheckDuplicate")
	mockStore.AssertNotCalled(t, "BatchInsert")
}

// TestProcessBatch_MarkProcessedError tests when marking as processed fails
func TestProcessBatch_MarkProcessedError(t *testing.T) {
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

	// Create test events
	events := []OfflineMessageEvent{
		{
			MsgID:            "msg-1",
			UserID:           "user-1",
			SenderID:         "sender-1",
			ConversationID:   "conv-1",
			ConversationType: "private",
			Content:          "Hello",
			SequenceNumber:   1,
			Timestamp:        time.Now().Unix(),
		},
	}

	// Mock dedup service
	mockDedup.On("CheckDuplicate", mock.Anything, "msg-1").Return(false, nil)
	mockDedup.On("MarkProcessed", mock.Anything, "msg-1").Return(errors.New("redis error"))

	// Mock store
	mockStore.On("BatchInsert", mock.Anything, mock.Anything).Return(nil)

	// Create mock session
	mockSession := &mockConsumerGroupSession{
		committed: false,
	}

	// Process batch
	err := handler.processBatch(context.Background(), events, mockSession, nil)

	// Assertions
	assert.NoError(t, err) // MarkProcessed error doesn't fail the batch
	assert.True(t, mockSession.committed)
	mockDedup.AssertExpectations(t)
	mockStore.AssertExpectations(t)
}

// mockConsumerGroupSession is a mock implementation of sarama.ConsumerGroupSession
type mockConsumerGroupSession struct {
	mock.Mock
	committed bool
}

func (m *mockConsumerGroupSession) Claims() map[string][]int32 {
	return nil
}

func (m *mockConsumerGroupSession) MemberID() string {
	return "test-member"
}

func (m *mockConsumerGroupSession) GenerationID() int32 {
	return 1
}

func (m *mockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) Commit() {
	m.committed = true
}

func (m *mockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
}

func (m *mockConsumerGroupSession) Context() context.Context {
	return context.Background()
}

// TestOfflineMessageEventSerialization tests JSON serialization
func TestOfflineMessageEventSerialization(t *testing.T) {
	event := OfflineMessageEvent{
		MsgID:            "msg-123",
		UserID:           "user-456",
		SenderID:         "sender-789",
		ConversationID:   "conv-abc",
		ConversationType: "private",
		Content:          "Hello, World!",
		SequenceNumber:   42,
		Timestamp:        1704067200,
	}

	// Serialize
	data, err := json.Marshal(event)
	assert.NoError(t, err)

	// Deserialize
	var decoded OfflineMessageEvent
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	// Compare
	assert.Equal(t, event.MsgID, decoded.MsgID)
	assert.Equal(t, event.UserID, decoded.UserID)
	assert.Equal(t, event.SenderID, decoded.SenderID)
	assert.Equal(t, event.ConversationID, decoded.ConversationID)
	assert.Equal(t, event.ConversationType, decoded.ConversationType)
	assert.Equal(t, event.Content, decoded.Content)
	assert.Equal(t, event.SequenceNumber, decoded.SequenceNumber)
	assert.Equal(t, event.Timestamp, decoded.Timestamp)
}
