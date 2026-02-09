package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// createTestObservability creates a test observability instance
func createTestObservability() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "shortener-service-test",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	return obs
}

// createTestRedisClient creates a test Redis client for integration tests
// Returns the client and a cleanup function
func createTestRedisClient(t *testing.T) (redis.UniversalClient, func()) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for tests to avoid conflicts
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	cleanup := func() {
		// Clean up test keys
		_ = client.FlushDB(ctx)
		_ = client.Close()
	}

	return client, cleanup
}

// MockStorage implements Storage interface for testing
type MockStorage struct {
	data      map[string]*StorageMapping
	mu        sync.RWMutex
	callCount int32
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]*StorageMapping),
	}
}

func (m *MockStorage) Get(ctx context.Context, shortCode string) (*StorageMapping, error) {
	atomic.AddInt32(&m.callCount, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping, ok := m.data[shortCode]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return mapping, nil
}

func (m *MockStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data[shortCode]
	return ok, nil
}

func (m *MockStorage) Set(shortCode string, mapping *StorageMapping) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[shortCode] = mapping
}

func (m *MockStorage) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func (m *MockStorage) ResetCallCount() {
	atomic.StoreInt32(&m.callCount, 0)
}

// TestCacheManagerSingleflight verifies singleflight request coalescing
func TestCacheManagerSingleflight(t *testing.T) {
	// Create cache manager with mock storage
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()

	// Add test data to storage
	testMapping := &StorageMapping{
		ShortCode: "test123",
		LongURL:   "https://example.com",
		CreatedAt: time.Now(),
	}
	storage.Set("test123", testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())

	// Launch 100 concurrent requests for the same key
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	ctx := context.Background()
	results := make([]*URLMapping, numRequests)
	errors := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			mapping, err := cm.Get(ctx, "test123")
			results[idx] = mapping
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// Verify all results are consistent
	for i, mapping := range results {
		if mapping == nil {
			t.Errorf("Request %d returned nil mapping", i)
			continue
		}
		if mapping.ShortCode != "test123" {
			t.Errorf("Request %d returned wrong short code: %s", i, mapping.ShortCode)
		}
		if mapping.LongURL != "https://example.com" {
			t.Errorf("Request %d returned wrong long URL: %s", i, mapping.LongURL)
		}
	}

	// Verify singleflight coalesced requests
	// Due to timing sensitivity, we allow 1-3 DB queries instead of exactly 1
	// The key property is that 100 concurrent requests result in significantly fewer DB queries
	callCount := storage.GetCallCount()
	t.Logf("Singleflight coalesced %d concurrent requests to %d DB queries (%.1f%% reduction)",
		numRequests, callCount, float64(numRequests-callCount)/float64(numRequests)*100)

	if callCount < 1 {
		t.Errorf("Expected at least 1 DB query, got %d", callCount)
	}
	if callCount > 3 {
		t.Errorf("Expected at most 3 DB queries due to timing, got %d (singleflight may not be working)", callCount)
	}
}

// TestCacheManagerFallback verifies multi-tier cache fallback
func TestCacheManagerFallback(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	testMapping := &StorageMapping{
		ShortCode: "test456",
		LongURL:   "https://example.com/test",
		CreatedAt: time.Now(),
	}
	storage.Set("test456", testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// First request: L1 miss → DB query → backfill L1
	storage.ResetCallCount()
	mapping, err := cm.Get(ctx, "test456")
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}
	if mapping.ShortCode != "test456" {
		t.Errorf("Expected short code test456, got %s", mapping.ShortCode)
	}
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query on L1 miss, got %d", storage.GetCallCount())
	}

	// Wait for Ristretto to process the backfill
	time.Sleep(50 * time.Millisecond)

	// Second request: L1 hit → no DB query
	storage.ResetCallCount()
	mapping, err = cm.Get(ctx, "test456")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}
	if mapping.ShortCode != "test456" {
		t.Errorf("Expected short code test456, got %s", mapping.ShortCode)
	}
	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L1 hit, got %d", storage.GetCallCount())
	}
}

// TestCacheManagerDelete verifies cache invalidation
func TestCacheManagerDelete(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Set a value in L1
	l1.Set("test789", "https://example.com", time.Now())

	// Wait for Ristretto to process the set
	time.Sleep(50 * time.Millisecond)

	// Verify it's in cache
	if mapping := l1.Get("test789"); mapping == nil {
		t.Fatal("Expected mapping in L1 cache")
	}

	// Delete from cache
	err = cm.Delete(ctx, "test789")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's removed from L1
	if mapping := l1.Get("test789"); mapping != nil {
		t.Error("Expected mapping to be removed from L1 cache")
	}
}

// TestCacheManagerWithLoader verifies CacheLoader integration
func TestCacheManagerWithLoader(t *testing.T) {
	// Setup test environment
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	// Create test Redis client (miniredis)
	redisClient, cleanup := createTestRedisClient(t)
	defer cleanup()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := NewL2Cache(l2Config, createTestObservability())
	if err != nil {
		t.Fatalf("Failed to create L2 cache: %v", err)
	}

	storage := NewMockStorage()
	testMapping := &StorageMapping{
		ShortCode: "loader123",
		LongURL:   "https://example.com/loader",
		CreatedAt: time.Now(),
	}
	storage.Set("loader123", testMapping)

	obs := createTestObservability()
	loader := NewCacheLoader(redisClient, storage, l2, obs)
	cm := NewCacheManagerWithLoader(l1, l2, storage, loader, obs)

	ctx := context.Background()

	// First request: L1 miss → L2 miss → LoadWithLock → DB query
	storage.ResetCallCount()
	mapping, err := cm.Get(ctx, "loader123")
	if err != nil {
		t.Fatalf("Get with loader failed: %v", err)
	}
	if mapping.ShortCode != "loader123" {
		t.Errorf("Expected short code loader123, got %s", mapping.ShortCode)
	}
	if mapping.LongURL != "https://example.com/loader" {
		t.Errorf("Expected long URL https://example.com/loader, got %s", mapping.LongURL)
	}

	// Verify LoadWithLock was used (should have 1 DB query)
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query with LoadWithLock, got %d", storage.GetCallCount())
	}

	// Wait for caches to process
	time.Sleep(50 * time.Millisecond)

	// Second request: L1 hit → no DB query
	storage.ResetCallCount()
	_, err = cm.Get(ctx, "loader123")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}
	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L1 hit, got %d", storage.GetCallCount())
	}

	// Clear L1, third request: L2 hit → no DB query
	l1.Delete("loader123")
	storage.ResetCallCount()
	_, err = cm.Get(ctx, "loader123")
	if err != nil {
		t.Fatalf("Third Get failed: %v", err)
	}
	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L2 hit, got %d", storage.GetCallCount())
	}
}

// TestCacheManagerWithLoaderConcurrent verifies LoadWithLock prevents cache stampede
func TestCacheManagerWithLoaderConcurrent(t *testing.T) {
	// Setup test environment
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	// Create test Redis client
	redisClient, cleanup := createTestRedisClient(t)
	defer cleanup()

	l2Config := L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := NewL2Cache(l2Config, createTestObservability())
	if err != nil {
		t.Fatalf("Failed to create L2 cache: %v", err)
	}

	storage := NewMockStorage()
	testMapping := &StorageMapping{
		ShortCode: "concurrent123",
		LongURL:   "https://example.com/concurrent",
		CreatedAt: time.Now(),
	}
	storage.Set("concurrent123", testMapping)

	obs := createTestObservability()
	loader := NewCacheLoader(redisClient, storage, l2, obs)
	cm := NewCacheManagerWithLoader(l1, l2, storage, loader, obs)

	// Launch 100 concurrent requests for the same key (cache miss)
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	ctx := context.Background()
	results := make([]*URLMapping, numRequests)
	errors := make([]error, numRequests)

	storage.ResetCallCount()

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			mapping, err := cm.Get(ctx, "concurrent123")
			results[idx] = mapping
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// Verify all results are consistent
	for i, mapping := range results {
		if mapping == nil {
			t.Errorf("Request %d returned nil mapping", i)
			continue
		}
		if mapping.ShortCode != "concurrent123" {
			t.Errorf("Request %d returned wrong short code: %s", i, mapping.ShortCode)
		}
		if mapping.LongURL != "https://example.com/concurrent" {
			t.Errorf("Request %d returned wrong long URL: %s", i, mapping.LongURL)
		}
	}

	// Verify LoadWithLock prevented cache stampede
	// With SETNX, only one goroutine should load from DB
	// Due to timing and singleflight, we allow 1-3 DB queries
	callCount := storage.GetCallCount()
	t.Logf("LoadWithLock prevented cache stampede: %d concurrent requests resulted in %d DB queries (%.1f%% reduction)",
		numRequests, callCount, float64(numRequests-callCount)/float64(numRequests)*100)

	if callCount < 1 {
		t.Errorf("Expected at least 1 DB query, got %d", callCount)
	}
	if callCount > 3 {
		t.Errorf("Expected at most 3 DB queries with LoadWithLock, got %d (SETNX may not be working)", callCount)
	}
}

// TestCacheManagerBackwardCompatibility verifies backward compatibility without CacheLoader
func TestCacheManagerBackwardCompatibility(t *testing.T) {
	// Create cache manager WITHOUT CacheLoader (using old constructor)
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	testMapping := &StorageMapping{
		ShortCode: "compat123",
		LongURL:   "https://example.com/compat",
		CreatedAt: time.Now(),
	}
	storage.Set("compat123", testMapping)

	// Use old constructor (no CacheLoader)
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Verify it still works with direct DB queries
	storage.ResetCallCount()
	mapping, err := cm.Get(ctx, "compat123")
	if err != nil {
		t.Fatalf("Get without loader failed: %v", err)
	}
	if mapping.ShortCode != "compat123" {
		t.Errorf("Expected short code compat123, got %s", mapping.ShortCode)
	}
	if mapping.LongURL != "https://example.com/compat" {
		t.Errorf("Expected long URL https://example.com/compat, got %s", mapping.LongURL)
	}

	// Verify direct DB query was used
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query without loader, got %d", storage.GetCallCount())
	}

	// Wait for cache to process
	time.Sleep(50 * time.Millisecond)

	// Second request should hit L1 cache
	storage.ResetCallCount()
	_, err = cm.Get(ctx, "compat123")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}
	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L1 hit, got %d", storage.GetCallCount())
	}
}
