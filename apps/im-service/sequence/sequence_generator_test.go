package sequence

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRedisClient implements a simple in-memory Redis client for testing
type MockRedisClient struct {
	data map[string]int64
	mu   sync.Mutex
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]int64),
	}
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key]++
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(m.data[key])
	return cmd
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := redis.NewStringCmd(ctx)
	if val, ok := m.data[key]; ok {
		cmd.SetVal(fmt.Sprintf("%d", val))
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

// Test monotonic increment for private chat
func TestGeneratePrivateChatSequence_MonotonicIncrement(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()
	userID1 := "user001"
	userID2 := "user002"

	// Generate multiple sequences and verify they are strictly increasing
	var prevSeq int64
	for i := 0; i < 10; i++ {
		seq, err := sg.GeneratePrivateChatSequence(ctx, userID1, userID2)
		require.NoError(t, err)
		assert.Greater(t, seq, prevSeq, "Sequence should be strictly increasing")
		prevSeq = seq
	}

	// Verify final sequence is 10
	assert.Equal(t, int64(10), prevSeq)
}

// Test monotonic increment for group chat
func TestGenerateGroupChatSequence_MonotonicIncrement(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()
	groupID := "group001"

	// Generate multiple sequences and verify they are strictly increasing
	var prevSeq int64
	for i := 0; i < 10; i++ {
		seq, err := sg.GenerateGroupChatSequence(ctx, groupID)
		require.NoError(t, err)
		assert.Greater(t, seq, prevSeq, "Sequence should be strictly increasing")
		prevSeq = seq
	}

	// Verify final sequence is 10
	assert.Equal(t, int64(10), prevSeq)
}

// Test key format for private chat
func TestGeneratePrivateChatSequence_KeyFormat(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()
	userID1 := "user001"
	userID2 := "user002"

	_, err := sg.GeneratePrivateChatSequence(ctx, userID1, userID2)
	require.NoError(t, err)

	// Verify the key format: seq:private:user001:user002 (sorted)
	expectedKey := "seq:private:user001:user002"
	assert.Contains(t, mockRedis.data, expectedKey)
}

// Test key format for group chat
func TestGenerateGroupChatSequence_KeyFormat(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()
	groupID := "group001"

	_, err := sg.GenerateGroupChatSequence(ctx, groupID)
	require.NoError(t, err)

	// Verify the key format: seq:group:group001
	expectedKey := "seq:group:group001"
	assert.Contains(t, mockRedis.data, expectedKey)
}

// Test user ID sorting for private chat
func TestGeneratePrivateChatSequence_UserIDSorting(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	// Generate sequence with user001 -> user002
	seq1, err := sg.GeneratePrivateChatSequence(ctx, "user001", "user002")
	require.NoError(t, err)

	// Generate sequence with user002 -> user001 (reversed order)
	seq2, err := sg.GeneratePrivateChatSequence(ctx, "user002", "user001")
	require.NoError(t, err)

	// Both should use the same key and increment the same sequence
	assert.Equal(t, int64(1), seq1)
	assert.Equal(t, int64(2), seq2)
}

// Test empty user ID validation
func TestGeneratePrivateChatSequence_EmptyUserID(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	// Test empty userID1
	_, err := sg.GeneratePrivateChatSequence(ctx, "", "user002")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user IDs cannot be empty")

	// Test empty userID2
	_, err = sg.GeneratePrivateChatSequence(ctx, "user001", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user IDs cannot be empty")

	// Test both empty
	_, err = sg.GeneratePrivateChatSequence(ctx, "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user IDs cannot be empty")
}

// Test empty group ID validation
func TestGenerateGroupChatSequence_EmptyGroupID(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	_, err := sg.GenerateGroupChatSequence(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "group ID cannot be empty")
}

// Test GetCurrentSequence for existing key
func TestGetCurrentSequence_ExistingKey(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()
	groupID := "group001"

	// Generate some sequences
	for i := 0; i < 5; i++ {
		_, err := sg.GenerateGroupChatSequence(ctx, groupID)
		require.NoError(t, err)
	}

	// Get current sequence without incrementing
	current, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, groupID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), current)

	// Verify it didn't increment
	current2, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, groupID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), current2)
}

// Test GetCurrentSequence for non-existent key
func TestGetCurrentSequence_NonExistentKey(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	// Get current sequence for non-existent key
	current, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), current)
}

// Test multiple conversations are independent
func TestGenerateSequence_MultipleConversations(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	// Generate sequences for different conversations
	seq1, err := sg.GeneratePrivateChatSequence(ctx, "user001", "user002")
	require.NoError(t, err)
	assert.Equal(t, int64(1), seq1)

	seq2, err := sg.GeneratePrivateChatSequence(ctx, "user003", "user004")
	require.NoError(t, err)
	assert.Equal(t, int64(1), seq2)

	seq3, err := sg.GenerateGroupChatSequence(ctx, "group001")
	require.NoError(t, err)
	assert.Equal(t, int64(1), seq3)

	// Generate more for first conversation
	seq4, err := sg.GeneratePrivateChatSequence(ctx, "user001", "user002")
	require.NoError(t, err)
	assert.Equal(t, int64(2), seq4)

	// Verify other conversations are unaffected
	seq5, err := sg.GeneratePrivateChatSequence(ctx, "user003", "user004")
	require.NoError(t, err)
	assert.Equal(t, int64(2), seq5)
}

// Test sortAndJoinUserIDs helper function
func TestSortAndJoinUserIDs(t *testing.T) {
	tests := []struct {
		name     string
		userID1  string
		userID2  string
		expected string
	}{
		{
			name:     "Already sorted",
			userID1:  "user001",
			userID2:  "user002",
			expected: "user001:user002",
		},
		{
			name:     "Needs sorting",
			userID1:  "user002",
			userID2:  "user001",
			expected: "user001:user002",
		},
		{
			name:     "Alphabetically different",
			userID1:  "bob",
			userID2:  "alice",
			expected: "alice:bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortAndJoinUserIDs(tt.userID1, tt.userID2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GenerateSequence with empty conversation ID
func TestGenerateSequence_EmptyConversationID(t *testing.T) {
	mockRedis := NewMockRedisClient()
	sg := &SequenceGenerator{redis: mockRedis}

	ctx := context.Background()

	_, err := sg.GenerateSequence(ctx, ConversationTypePrivate, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conversation ID cannot be empty")
}
