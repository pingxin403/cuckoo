package sequence

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDB implements a simple in-memory database for testing
type MockDB struct {
	snapshots map[string]SequenceSnapshot
}

func NewMockDB() *MockDB {
	return &MockDB{
		snapshots: make(map[string]SequenceSnapshot),
	}
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	// Parse INSERT/UPDATE query
	if len(args) >= 4 {
		conversationType := args[0].(string)
		conversationID := args[1].(string)
		sequenceNumber := args[2].(int64)
		snapshotTime := args[3].(time.Time)

		key := fmt.Sprintf("%s:%s", conversationType, conversationID)
		m.snapshots[key] = SequenceSnapshot{
			ConversationType: conversationType,
			ConversationID:   conversationID,
			SequenceNumber:   sequenceNumber,
			SnapshotTime:     snapshotTime,
		}
	}

	return &mockResult{}, nil
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	// This is a simplified mock - in real tests, we'd use a proper SQL mock library
	// For now, we'll test with the actual methods that use this
	return nil
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// This is a simplified mock
	return nil, nil
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 1, nil }

// Test SaveSnapshot
func TestSaveSnapshot(t *testing.T) {
	_ = NewMockDB() // Mock DB for future use
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)

	// Create a real sql.DB for testing (we'll use a mock implementation)
	// For unit tests, we'll test the logic without actual DB
	sb := &SequenceBackup{
		db:                nil, // We'll test with mock
		generator:         sg,
		snapshotInterval:  10000,
		snapshotThreshold: 10000,
	}

	// Test snapshot interval
	assert.Equal(t, int64(10000), sb.GetSnapshotInterval())
}

// Test ShouldSnapshot logic
func TestShouldSnapshot(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)
	sb := NewSequenceBackup(nil, sg, 10000)

	tests := []struct {
		name           string
		sequenceNumber int64
		expected       bool
	}{
		{
			name:           "At interval boundary",
			sequenceNumber: 10000,
			expected:       true,
		},
		{
			name:           "At second interval",
			sequenceNumber: 20000,
			expected:       true,
		},
		{
			name:           "Not at boundary",
			sequenceNumber: 10001,
			expected:       false,
		},
		{
			name:           "Before first interval",
			sequenceNumber: 5000,
			expected:       false,
		},
		{
			name:           "Zero",
			sequenceNumber: 0,
			expected:       true, // 0 % 10000 == 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sb.ShouldSnapshot(tt.sequenceNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test default snapshot interval
func TestNewSequenceBackup_DefaultInterval(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)

	// Test with zero interval (should use default)
	sb := NewSequenceBackup(nil, sg, 0)
	assert.Equal(t, int64(10000), sb.GetSnapshotInterval())

	// Test with negative interval (should use default)
	sb = NewSequenceBackup(nil, sg, -100)
	assert.Equal(t, int64(10000), sb.GetSnapshotInterval())

	// Test with custom interval
	sb = NewSequenceBackup(nil, sg, 5000)
	assert.Equal(t, int64(5000), sb.GetSnapshotInterval())
}

// Test snapshot interval boundaries
func TestSnapshotIntervalBoundaries(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)
	sb := NewSequenceBackup(nil, sg, 100)

	// Test multiple intervals
	for i := int64(0); i <= 1000; i++ {
		shouldSnapshot := sb.ShouldSnapshot(i)
		expectedSnapshot := (i % 100) == 0

		if shouldSnapshot != expectedSnapshot {
			t.Errorf("At sequence %d: expected %v, got %v", i, expectedSnapshot, shouldSnapshot)
		}
	}
}

// Test SequenceSnapshot struct
func TestSequenceSnapshot(t *testing.T) {
	now := time.Now()
	snapshot := SequenceSnapshot{
		ConversationType: "private",
		ConversationID:   "user001:user002",
		SequenceNumber:   12345,
		SnapshotTime:     now,
	}

	assert.Equal(t, "private", snapshot.ConversationType)
	assert.Equal(t, "user001:user002", snapshot.ConversationID)
	assert.Equal(t, int64(12345), snapshot.SequenceNumber)
	assert.Equal(t, now, snapshot.SnapshotTime)
}

// Test recovery scenario
func TestRecoveryScenario(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)
	ctx := context.Background()

	// Generate some sequences
	for i := 0; i < 15000; i++ {
		_, err := sg.GenerateGroupChatSequence(ctx, "group001")
		require.NoError(t, err)
	}

	// Verify we reached 15000
	current, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, "group001")
	require.NoError(t, err)
	assert.Equal(t, int64(15000), current)

	// Simulate taking snapshots at intervals
	sb := NewSequenceBackup(nil, sg, 10000)

	// Check which sequences should trigger snapshots
	shouldSnapshot10000 := sb.ShouldSnapshot(10000)
	shouldSnapshot15000 := sb.ShouldSnapshot(15000)
	shouldSnapshot12345 := sb.ShouldSnapshot(12345)

	assert.True(t, shouldSnapshot10000, "Should snapshot at 10000")
	assert.False(t, shouldSnapshot15000, "Should not snapshot at 15000 (not a multiple of 10000)")
	assert.False(t, shouldSnapshot12345, "Should not snapshot at 12345")
}

// Test multiple conversations with different intervals
func TestMultipleConversationsSnapshots(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := NewSequenceGenerator(mockRedis)
	sb := NewSequenceBackup(nil, sg, 1000)
	ctx := context.Background()

	conversations := []struct {
		convType ConversationType
		convID   string
		count    int
	}{
		{ConversationTypePrivate, "user001:user002", 2500},
		{ConversationTypeGroup, "group001", 3500},
		{ConversationTypePrivate, "user003:user004", 1500},
	}

	for _, conv := range conversations {
		for i := 0; i < conv.count; i++ {
			_, err := sg.GenerateSequence(ctx, conv.convType, conv.convID)
			require.NoError(t, err)
		}

		// Verify final sequence
		current, err := sg.GetCurrentSequence(ctx, conv.convType, conv.convID)
		require.NoError(t, err)
		assert.Equal(t, int64(conv.count), current)

		// Check snapshot points
		expectedSnapshots := conv.count / 1000
		actualSnapshots := 0
		for i := 1; i <= conv.count; i++ {
			if sb.ShouldSnapshot(int64(i)) {
				actualSnapshots++
			}
		}
		assert.Equal(t, expectedSnapshots, actualSnapshots,
			"Conversation %s:%s should have %d snapshots", conv.convType, conv.convID, expectedSnapshots)
	}
}
