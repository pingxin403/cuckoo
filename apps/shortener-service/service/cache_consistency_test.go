package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConsistencyTest(t *testing.T) (*miniredis.Miniredis, *cache.CacheManager, observability.Observability) {
	mr := miniredis.RunT(t)

	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	// Create L1 cache
	l1Cache, err := cache.NewL1Cache()
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs: []string{mr.Addr()},
		DB:    0,
	}
	l2Cache, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	// Create a mock storage
	mockStorage := &MockStorageForConsistency{}

	cacheManager := cache.NewCacheManager(l1Cache, l2Cache, mockStorage, obs)

	return mr, cacheManager, obs
}

// MockStorageForConsistency for testing
type MockStorageForConsistency struct{}

func (m *MockStorageForConsistency) Get(ctx context.Context, shortCode string) (*cache.StorageMapping, error) {
	return nil, errors.New("not implemented")
}

func (m *MockStorageForConsistency) Exists(ctx context.Context, shortCode string) (bool, error) {
	return false, errors.New("not implemented")
}

func TestNewCacheConsistency(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)

	assert.NotNil(t, cc)
	assert.NotNil(t, cc.cacheManager)
	assert.NotNil(t, cc.obs)
	assert.Equal(t, 1*time.Second, cc.delayTime)
}

func TestCacheConsistency_UpdateWithConsistency_Success(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	// Use shorter delay for testing
	cc.SetDelayTime(100 * time.Millisecond)

	ctx := context.Background()
	shortCode := "test123"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://old-url.com")

	// Verify cache exists before update
	exists := mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Perform update with consistency
	updateCalled := false
	err := cc.UpdateWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		updateCalled = true
		// Simulate DB update
		return nil
	})

	require.NoError(t, err)
	assert.True(t, updateCalled)

	// Cache should be deleted immediately
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)

	// Wait for delayed delete to complete
	time.Sleep(200 * time.Millisecond)

	// Cache should still be deleted
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_UpdateWithConsistency_DBError(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	ctx := context.Background()
	shortCode := "test456"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://old-url.com")

	// Perform update with DB error
	dbError := errors.New("database connection failed")
	err := cc.UpdateWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		return dbError
	})

	// Should return the DB error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database update failed")
	assert.Contains(t, err.Error(), dbError.Error())

	// Cache should still be deleted (immediate delete happened)
	exists := mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_CreateWithConsistency_Success(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	cc.SetDelayTime(100 * time.Millisecond)

	ctx := context.Background()
	shortCode := "new123"

	// Perform create with consistency
	createCalled := false
	err := cc.CreateWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		createCalled = true
		// Simulate DB create
		return nil
	})

	require.NoError(t, err)
	assert.True(t, createCalled)

	// Wait for delayed delete to complete
	time.Sleep(200 * time.Millisecond)

	// Cache should be deleted (delayed delete)
	exists := mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_CreateWithConsistency_DBError(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	ctx := context.Background()
	shortCode := "new456"

	// Perform create with DB error
	dbError := errors.New("unique constraint violation")
	err := cc.CreateWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		return dbError
	})

	// Should return the DB error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database create failed")
	assert.Contains(t, err.Error(), dbError.Error())
}

func TestCacheConsistency_DeleteWithConsistency_Success(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	cc.SetDelayTime(100 * time.Millisecond)

	ctx := context.Background()
	shortCode := "del123"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://to-delete.com")

	// Verify cache exists before delete
	exists := mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Perform delete with consistency
	deleteCalled := false
	err := cc.DeleteWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		deleteCalled = true
		// Simulate DB delete
		return nil
	})

	require.NoError(t, err)
	assert.True(t, deleteCalled)

	// Cache should be deleted immediately
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)

	// Wait for delayed delete to complete
	time.Sleep(200 * time.Millisecond)

	// Cache should still be deleted
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_DeleteWithConsistency_DBError(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	ctx := context.Background()
	shortCode := "del456"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://to-delete.com")

	// Perform delete with DB error
	dbError := errors.New("foreign key constraint")
	err := cc.DeleteWithConsistency(ctx, shortCode, func(ctx context.Context) error {
		return dbError
	})

	// Should return the DB error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database delete failed")
	assert.Contains(t, err.Error(), dbError.Error())

	// Cache should still be deleted (immediate delete happened)
	exists := mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_ImmediateDelete(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	ctx := context.Background()
	shortCode := "imm123"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://immediate.com")

	// Verify cache exists
	exists := mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Perform immediate delete
	err := cc.immediateDelete(ctx, shortCode)
	require.NoError(t, err)

	// Cache should be deleted
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_DelayedDelete(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	cc.SetDelayTime(100 * time.Millisecond)

	shortCode := "delay123"

	// Pre-populate cache
	mr.HSet("url:"+shortCode, "short_code", shortCode)
	mr.HSet("url:"+shortCode, "long_url", "https://delayed.com")

	// Verify cache exists
	exists := mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Trigger delayed delete
	go cc.delayedDelete(shortCode)

	// Cache should still exist immediately
	exists = mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Wait for delay
	time.Sleep(50 * time.Millisecond)
	// Still should exist (delay not complete)
	exists = mr.Exists("url:" + shortCode)
	assert.True(t, exists)

	// Wait for delay to complete
	time.Sleep(100 * time.Millisecond)
	// Now should be deleted
	exists = mr.Exists("url:" + shortCode)
	assert.False(t, exists)
}

func TestCacheConsistency_SetDelayTime(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)

	// Default delay
	assert.Equal(t, 1*time.Second, cc.GetDelayTime())

	// Set custom delay
	cc.SetDelayTime(500 * time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, cc.GetDelayTime())

	// Set another delay
	cc.SetDelayTime(2 * time.Second)
	assert.Equal(t, 2*time.Second, cc.GetDelayTime())
}

func TestCacheConsistency_DelayedDeleteTiming(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)

	// Test with different delay times
	delays := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	for i, delay := range delays {
		shortCode := fmt.Sprintf("timing%d", i)
		cc.SetDelayTime(delay)

		// Pre-populate cache
		mr.HSet("url:"+shortCode, "short_code", shortCode)
		mr.HSet("url:"+shortCode, "long_url", "https://timing.com")

		start := time.Now()
		go cc.delayedDelete(shortCode)

		// Wait for delay + buffer
		time.Sleep(delay + 50*time.Millisecond)

		elapsed := time.Since(start)

		// Cache should be deleted
		exists := mr.Exists("url:" + shortCode)
		assert.False(t, exists, "Cache should be deleted for delay %v", delay)

		// Verify timing is approximately correct (within 100ms tolerance)
		assert.GreaterOrEqual(t, elapsed, delay, "Elapsed time should be at least the delay")
		assert.Less(t, elapsed, delay+150*time.Millisecond, "Elapsed time should not exceed delay + 150ms")
	}
}

func TestCacheConsistency_ConcurrentOperations(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	cc.SetDelayTime(50 * time.Millisecond)

	ctx := context.Background()
	numOperations := 10

	// Perform concurrent updates
	done := make(chan bool, numOperations)
	for i := 0; i < numOperations; i++ {
		shortCode := fmt.Sprintf("concurrent%d", i)

		// Pre-populate cache
		mr.HSet("url:"+shortCode, "short_code", shortCode)
		mr.HSet("url:"+shortCode, "long_url", "https://concurrent.com")

		go func(code string) {
			err := cc.UpdateWithConsistency(ctx, code, func(ctx context.Context) error {
				// Simulate DB update
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if err != nil {
				t.Errorf("Update failed for %s: %v", code, err)
			}
			done <- true
		}(shortCode)
	}

	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		<-done
	}

	// Wait for all delayed deletes to complete
	time.Sleep(150 * time.Millisecond)

	// Verify all caches are deleted
	for i := 0; i < numOperations; i++ {
		shortCode := fmt.Sprintf("concurrent%d", i)
		exists := mr.Exists("url:" + shortCode)
		assert.False(t, exists, "Cache should be deleted for %s", shortCode)
	}
}

func TestCacheConsistency_ImmediateDeleteError(t *testing.T) {
	mr, cacheManager, obs := setupConsistencyTest(t)
	defer mr.Close()

	cc := NewCacheConsistency(cacheManager, obs)
	ctx := context.Background()

	// Close Redis to simulate error
	mr.Close()

	shortCode := "error123"

	// Immediate delete should fail gracefully
	err := cc.immediateDelete(ctx, shortCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete from cache")
}
