package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestObservabilityForLoader creates a test observability instance
func createTestObservabilityForLoader() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	return obs
}

// createTestL2Cache creates a test L2Cache instance
func createTestL2Cache(t *testing.T) *L2Cache {
	t.Helper()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := NewL2Cache(l2Config, createTestObservabilityForLoader())
	if err != nil {
		t.Fatalf("Failed to create L2 cache: %v", err)
	}
	return l2
}

// MockStorage for testing
type MockStorageForLoader struct {
	mappings map[string]*StorageMapping
	getCalls int
}

func NewMockStorageForLoader() *MockStorageForLoader {
	return &MockStorageForLoader{
		mappings: make(map[string]*StorageMapping),
		getCalls: 0,
	}
}

func (m *MockStorageForLoader) Get(ctx context.Context, shortCode string) (*StorageMapping, error) {
	m.getCalls++
	if mapping, ok := m.mappings[shortCode]; ok {
		return mapping, nil
	}
	return nil, storage.ErrNotFound
}

func (m *MockStorageForLoader) Exists(ctx context.Context, shortCode string) (bool, error) {
	_, ok := m.mappings[shortCode]
	return ok, nil
}

func (m *MockStorageForLoader) GetCallCount() int {
	return m.getCalls
}

func (m *MockStorageForLoader) ResetCallCount() {
	m.getCalls = 0
}

// TestCacheLoader_LoadWithLock_LockAcquired tests successful lock acquisition and data loading
func TestCacheLoader_LoadWithLock_LockAcquired(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-lock-acquired"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, "url:"+testShortCode)
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	mockStorage.mappings[testShortCode] = &StorageMapping{
		ShortCode: testShortCode,
		LongURL:   "https://example.com/test",
		CreatedAt: time.Now(),
	}

	obs := createTestObservabilityForLoader()
	l2 := createTestL2Cache(t)
	loader := NewCacheLoader(client, mockStorage, l2, obs)

	// Test: Load data when lock is acquired
	result, err := loader.LoadWithLock(ctx, testShortCode)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "https://example.com/test", result.LongURL)
	assert.Equal(t, testShortCode, result.ShortCode)

	// Verify data was loaded from database
	assert.Equal(t, 1, mockStorage.GetCallCount())

	// Verify data was cached in L2 (using hash structure)
	cachedMapping, err := l2.Get(ctx, testShortCode)
	require.NoError(t, err)
	assert.NotNil(t, cachedMapping)
	assert.Equal(t, "https://example.com/test", cachedMapping.LongURL)

	// Verify lock was released
	lockExists, err := client.Exists(ctx, lockKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), lockExists, "Lock should be released")
}

// TestCacheLoader_LoadWithLock_NotFound tests handling of non-existent records
func TestCacheLoader_LoadWithLock_NotFound(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-not-found"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Test: Load non-existent data
	result, err := loader.LoadWithLock(ctx, testShortCode)

	// Verify
	assert.Equal(t, storage.ErrNotFound, err)
	assert.Nil(t, result)

	// Note: Empty placeholder is not cached in this implementation
	// The L2Cache.Set method is not called for non-existent records
}

// TestCacheLoader_LoadWithLock_Contention tests retry logic when lock is held
func TestCacheLoader_LoadWithLock_Contention(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-contention"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	mockStorage.mappings["test-contention"] = &StorageMapping{
		ShortCode: "test-contention",
		LongURL:   "https://example.com/contention",
		CreatedAt: time.Now(),
	}

	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Simulate lock being held by another goroutine
	err = client.Set(ctx, lockKey, "1", 5*time.Second).Err()
	require.NoError(t, err)

	// Pre-populate L2 cache (simulating another goroutine loading the data)
	err = l2Cache.Set(ctx, testShortCode, "https://example.com/contention", time.Now())
	require.NoError(t, err)

	// Test: Load data when lock is held (should retry and read from cache)
	result, err := loader.LoadWithLock(ctx, testShortCode)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-contention", result.ShortCode)
	assert.Equal(t, "https://example.com/contention", result.LongURL)

	// Verify database was NOT queried (data was read from cache)
	assert.Equal(t, 0, mockStorage.GetCallCount())
}

// TestCacheLoader_LoadWithLock_ExponentialBackoff tests exponential backoff timing
func TestCacheLoader_LoadWithLock_ExponentialBackoff(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-backoff"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Simulate lock being held
	err = client.Set(ctx, lockKey, "1", 5*time.Second).Err()
	require.NoError(t, err)

	// Test: Measure retry timing
	start := time.Now()
	_, err = loader.LoadWithLock(ctx, testShortCode)
	duration := time.Since(start)

	// Verify: Should fail after 3 retries with exponential backoff
	// Expected delays: 50ms, 100ms, 200ms = 350ms total
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load after 3 retries")

	// Allow some tolerance for timing (300ms to 500ms)
	assert.GreaterOrEqual(t, duration, 300*time.Millisecond, "Should wait at least 300ms")
	assert.LessOrEqual(t, duration, 500*time.Millisecond, "Should not wait more than 500ms")
}

// TestCacheLoader_LoadWithLock_LockRelease tests that lock is always released
func TestCacheLoader_LoadWithLock_LockRelease(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-lock-release"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	mockStorage.mappings["test-lock-release"] = &StorageMapping{
		ShortCode: "test-lock-release",
		LongURL:   "https://example.com/release",
		CreatedAt: time.Now(),
	}

	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Test: Load data
	_, err = loader.LoadWithLock(ctx, testShortCode)
	require.NoError(t, err)

	// Verify: Lock should be released immediately
	lockExists, err := client.Exists(ctx, lockKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), lockExists, "Lock should be released after loading")
}

// TestCacheLoader_LoadWithLock_StorageError tests handling of storage errors
func TestCacheLoader_LoadWithLock_StorageError(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()

	// Clean up test keys
	testShortCode := "test-storage-error"
	lockKey := "lock:" + testShortCode
	defer func() {
		client.Del(ctx, lockKey)
	}()

	// Create mock storage that returns an error
	mockStorage := &MockStorageWithError{
		err: errors.New("database connection failed"),
	}

	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Test: Load data when storage fails
	result, err := loader.LoadWithLock(ctx, testShortCode)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load from storage")
	assert.Nil(t, result)

	// Verify lock was released even on error
	lockExists, err := client.Exists(ctx, lockKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), lockExists, "Lock should be released even on error")
}

// TestCacheLoader_ShortCodeHandling tests short code handling
func TestCacheLoader_ShortCodeHandling(t *testing.T) {
	// This test verifies that LoadWithLock correctly handles short codes
	// The function now takes shortCode directly instead of a Redis key

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	ctx := context.Background()
	testShortCode := "abc123"
	lockKey := "lock:" + testShortCode

	defer func() {
		client.Del(ctx, lockKey)
	}()

	mockStorage := NewMockStorageForLoader()
	mockStorage.mappings[testShortCode] = &StorageMapping{
		ShortCode: testShortCode,
		LongURL:   "https://example.com/test",
		CreatedAt: time.Now(),
	}

	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Test: Load with short code
	result, err := loader.LoadWithLock(ctx, testShortCode)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testShortCode, result.ShortCode)
	assert.Equal(t, "https://example.com/test", result.LongURL)
}

// MockStorageWithError for testing error handling
type MockStorageWithError struct {
	err error
}

func (m *MockStorageWithError) Get(ctx context.Context, shortCode string) (*StorageMapping, error) {
	return nil, m.err
}

func (m *MockStorageWithError) Exists(ctx context.Context, shortCode string) (bool, error) {
	return false, m.err
}

// TestCacheLoader_NewCacheLoader tests constructor with default values
func TestCacheLoader_NewCacheLoader(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	mockStorage := NewMockStorageForLoader()
	obs := createTestObservabilityForLoader()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2Cache, err := NewL2Cache(l2Config, obs)
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	loader := NewCacheLoader(client, mockStorage, l2Cache, obs)

	// Verify default values
	assert.NotNil(t, loader)
	assert.Equal(t, 5*time.Second, loader.lockTTL, "Default lock TTL should be 5 seconds")
	assert.Equal(t, 3, loader.maxRetries, "Default max retries should be 3")
	assert.Equal(t, 50*time.Millisecond, loader.retryDelay, "Default retry delay should be 50ms")
}
