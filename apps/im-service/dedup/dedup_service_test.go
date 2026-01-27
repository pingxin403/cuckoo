package dedup

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a miniredis instance for testing
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *DedupService) {
	mr := miniredis.RunT(t)

	cfg := Config{
		RedisAddr: mr.Addr(),
		TTL:       7 * 24 * time.Hour,
	}

	service := NewDedupService(cfg)
	t.Cleanup(func() {
		_ = service.Close()
	})
	return mr, service
}

// Test duplicate detection
// Requirements: 8.1, 8.3
func TestDedupService_CheckDuplicate(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	msgID := "msg-12345"

	// First check - should not be duplicate
	isDup, err := service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.False(t, isDup, "Message should not be duplicate initially")

	// Mark as processed
	err = service.MarkProcessed(ctx, msgID)
	require.NoError(t, err)

	// Second check - should be duplicate
	isDup, err = service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.True(t, isDup, "Message should be duplicate after marking")
}

// Test marking message as processed
// Requirements: 8.2
func TestDedupService_MarkProcessed(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	msgID := "msg-67890"

	// Mark as processed
	err := service.MarkProcessed(ctx, msgID)
	require.NoError(t, err)

	// Verify it's marked
	isDup, err := service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.True(t, isDup, "Message should be marked as duplicate")
}

// Test TTL expiration
// Requirements: 8.2
func TestDedupService_TTLExpiration(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	// Use short TTL for testing
	service.ttl = 1 * time.Second

	ctx := context.Background()
	msgID := "msg-ttl-test"

	// Mark as processed
	err := service.MarkProcessed(ctx, msgID)
	require.NoError(t, err)

	// Should be duplicate immediately
	isDup, err := service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.True(t, isDup)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Should not be duplicate after TTL expiration
	isDup, err = service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.False(t, isDup, "Message should not be duplicate after TTL expiration")
}

// Test atomic check and mark
// Requirements: 8.1, 8.2, 8.3
func TestDedupService_CheckAndMark(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	msgID := "msg-atomic-test"

	// First call - should not be duplicate and mark it
	isDup, err := service.CheckAndMark(ctx, msgID)
	require.NoError(t, err)
	assert.False(t, isDup, "First call should not be duplicate")

	// Second call - should be duplicate
	isDup, err = service.CheckAndMark(ctx, msgID)
	require.NoError(t, err)
	assert.True(t, isDup, "Second call should be duplicate")

	// Verify it's marked
	isDup, err = service.CheckDuplicate(ctx, msgID)
	require.NoError(t, err)
	assert.True(t, isDup)
}

// Test concurrent access
// Requirements: 8.3
func TestDedupService_ConcurrentAccess(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	msgID := "msg-concurrent-test"

	// Run multiple goroutines trying to mark the same message
	const numGoroutines = 10
	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			isDup, err := service.CheckAndMark(ctx, msgID)
			require.NoError(t, err)
			results <- isDup
		}()
	}

	// Collect results
	duplicateCount := 0
	newCount := 0
	for i := 0; i < numGoroutines; i++ {
		isDup := <-results
		if isDup {
			duplicateCount++
		} else {
			newCount++
		}
	}

	// Exactly one should succeed (not duplicate), rest should be duplicates
	assert.Equal(t, 1, newCount, "Exactly one goroutine should mark as new")
	assert.Equal(t, numGoroutines-1, duplicateCount, "Rest should see as duplicate")
}

// Test Redis connection failure handling
// Requirements: 8.1
func TestDedupService_RedisConnectionFailure(t *testing.T) {
	mr, service := setupTestRedis(t)

	ctx := context.Background()
	msgID := "msg-connection-test"

	// Close Redis to simulate connection failure
	mr.Close()

	// CheckDuplicate should return error
	_, err := service.CheckDuplicate(ctx, msgID)
	assert.Error(t, err, "Should return error when Redis is down")

	// MarkProcessed should return error
	err = service.MarkProcessed(ctx, msgID)
	assert.Error(t, err, "Should return error when Redis is down")

	// CheckAndMark should return error
	_, err = service.CheckAndMark(ctx, msgID)
	assert.Error(t, err, "Should return error when Redis is down")
}

// Test Ping functionality
func TestDedupService_Ping(t *testing.T) {
	mr, service := setupTestRedis(t)

	ctx := context.Background()

	// Ping should succeed
	err := service.Ping(ctx)
	assert.NoError(t, err, "Ping should succeed when Redis is up")

	// Close Redis
	mr.Close()

	// Ping should fail
	err = service.Ping(ctx)
	assert.Error(t, err, "Ping should fail when Redis is down")
}

// Test default TTL configuration
// Requirements: 8.2
func TestDedupService_DefaultTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := Config{
		RedisAddr: mr.Addr(),
		// TTL not specified - should use default 7 days
	}

	service := NewDedupService(cfg)
	defer func() { _ = service.Close() }()

	assert.Equal(t, 7*24*time.Hour, service.ttl, "Should use default 7-day TTL")
}

// Test custom TTL configuration
// Requirements: 8.2
func TestDedupService_CustomTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	customTTL := 24 * time.Hour
	cfg := Config{
		RedisAddr: mr.Addr(),
		TTL:       customTTL,
	}

	service := NewDedupService(cfg)
	defer func() { _ = service.Close() }()

	assert.Equal(t, customTTL, service.ttl, "Should use custom TTL")
}

// Test dedup key format
func TestDedupService_DedupKeyFormat(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := Config{
		RedisAddr: mr.Addr(),
	}

	service := NewDedupService(cfg)
	defer func() { _ = service.Close() }()

	msgID := "test-msg-123"
	expectedKey := "dedup:msg:test-msg-123"
	actualKey := service.dedupKey(msgID)

	assert.Equal(t, expectedKey, actualKey, "Dedup key format should match")
}

// Test multiple different messages
// Requirements: 8.1, 8.3
func TestDedupService_MultipleDifferentMessages(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	messages := []string{"msg-1", "msg-2", "msg-3", "msg-4", "msg-5"}

	// Mark all messages as processed
	for _, msgID := range messages {
		err := service.MarkProcessed(ctx, msgID)
		require.NoError(t, err)
	}

	// Verify all are marked as duplicates
	for _, msgID := range messages {
		isDup, err := service.CheckDuplicate(ctx, msgID)
		require.NoError(t, err)
		assert.True(t, isDup, "Message %s should be duplicate", msgID)
	}

	// Verify a new message is not duplicate
	isDup, err := service.CheckDuplicate(ctx, "msg-new")
	require.NoError(t, err)
	assert.False(t, isDup, "New message should not be duplicate")
}

// Test Close functionality
func TestDedupService_Close(t *testing.T) {
	mr, service := setupTestRedis(t)
	defer mr.Close()

	// Close should succeed
	err := service.Close()
	assert.NoError(t, err, "Close should succeed")

	// Operations after close should fail
	ctx := context.Background()
	_, err = service.CheckDuplicate(ctx, "msg-test")
	assert.Error(t, err, "Operations should fail after close")
}
