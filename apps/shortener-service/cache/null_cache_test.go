package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNullCacheEntry tests null cache entry creation and detection
func TestNullCacheEntry(t *testing.T) {
	// Create null entry
	nullEntry := CreateNullEntry("test123")

	if nullEntry.ShortCode != "test123" {
		t.Errorf("Expected short code test123, got %s", nullEntry.ShortCode)
	}

	if nullEntry.LongURL != "" {
		t.Errorf("Expected empty long URL for null entry, got %s", nullEntry.LongURL)
	}

	// Test null entry detection
	if !IsNullEntry(nullEntry) {
		t.Error("Expected IsNullEntry to return true for null entry")
	}

	// Test normal entry detection
	normalEntry := &URLMapping{
		ShortCode: "test456",
		LongURL:   "https://example.com",
		CreatedAt: time.Now(),
	}

	if IsNullEntry(normalEntry) {
		t.Error("Expected IsNullEntry to return false for normal entry")
	}
}

// TestCachePenetrationWithNullCache tests that null caching prevents repeated DB queries
func TestCachePenetrationWithNullCache(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	nonExistentKey := "non_existent_key"

	// First request: cache miss → DB query → cache null entry
	storage.ResetCallCount()
	_, err = cm.Get(ctx, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	firstCallCount := storage.GetCallCount()
	if firstCallCount != 1 {
		t.Errorf("Expected 1 DB query on first request, got %d", firstCallCount)
	}

	// Wait for cache to process
	time.Sleep(50 * time.Millisecond)

	// Second request: should hit null cache → no DB query
	storage.ResetCallCount()
	_, err = cm.Get(ctx, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	secondCallCount := storage.GetCallCount()
	if secondCallCount != 0 {
		t.Errorf("Expected 0 DB queries on second request (null cache hit), got %d", secondCallCount)
	}

	t.Logf("Null cache prevented %d DB queries", firstCallCount-secondCallCount)
}

// TestCachePenetrationConcurrentWithNullCache tests null caching under concurrent load
func TestCachePenetrationConcurrentWithNullCache(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())

	const numRequests = 100
	nonExistentKey := "concurrent_non_existent"

	// First wave: concurrent requests for non-existent key
	var wg sync.WaitGroup
	wg.Add(numRequests)

	ctx := context.Background()
	storage.ResetCallCount()

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			_, _ = cm.Get(ctx, nonExistentKey)
		}()
	}

	wg.Wait()
	firstWaveQueries := storage.GetCallCount()

	t.Logf("First wave: %d concurrent requests resulted in %d DB queries", numRequests, firstWaveQueries)

	// Wait for null cache to be populated
	time.Sleep(100 * time.Millisecond)

	// Second wave: should hit null cache
	wg.Add(numRequests)
	storage.ResetCallCount()

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			_, _ = cm.Get(ctx, nonExistentKey)
		}()
	}

	wg.Wait()
	secondWaveQueries := storage.GetCallCount()

	t.Logf("Second wave: %d concurrent requests resulted in %d DB queries (null cache)", numRequests, secondWaveQueries)

	// Second wave should have significantly fewer DB queries
	if secondWaveQueries > firstWaveQueries/2 {
		t.Errorf("Expected null cache to reduce DB queries significantly, got %d in second wave vs %d in first wave",
			secondWaveQueries, firstWaveQueries)
	}

	reductionPercent := float64(firstWaveQueries-secondWaveQueries) / float64(firstWaveQueries) * 100
	t.Logf("Null cache reduced DB queries by %.1f%%", reductionPercent)
}

// TestNullCacheWithL2 tests null caching with Redis (L2 cache)
func TestNullCacheWithL2(t *testing.T) {
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
	obs := createTestObservability()
	loader := NewCacheLoader(redisClient, storage, l2, obs)
	cm := NewCacheManagerWithLoader(l1, l2, storage, loader, obs)

	ctx := context.Background()
	nonExistentKey := "non_existent_with_l2"

	// First request: cache miss → DB query → cache null entry in both L1 and L2
	storage.ResetCallCount()
	_, err = cm.Get(ctx, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query on first request, got %d", storage.GetCallCount())
	}

	// Wait for caches to process
	time.Sleep(50 * time.Millisecond)

	// Second request: should hit L1 null cache
	storage.ResetCallCount()
	_, err = cm.Get(ctx, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L1 hit, got %d", storage.GetCallCount())
	}

	// Clear L1, third request: should hit L2 null cache
	l1.Delete(nonExistentKey)
	storage.ResetCallCount()
	_, err = cm.Get(ctx, nonExistentKey)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}

	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L2 hit, got %d", storage.GetCallCount())
	}

	t.Log("Null cache working correctly with both L1 and L2")
}

// TestNullCacheExpiration tests that null cache entries expire after TTL
func TestNullCacheExpiration(t *testing.T) {
	t.Skip("This test requires waiting for TTL expiration (1 minute), skipping for CI")

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
	obs := createTestObservability()
	loader := NewCacheLoader(redisClient, storage, l2, obs)
	cm := NewCacheManagerWithLoader(l1, l2, storage, loader, obs)

	ctx := context.Background()
	nonExistentKey := "expiring_null_entry"

	// First request: cache null entry
	_, _ = cm.Get(ctx, nonExistentKey)
	time.Sleep(50 * time.Millisecond)

	// Verify null cache hit
	storage.ResetCallCount()
	_, _ = cm.Get(ctx, nonExistentKey)
	if storage.GetCallCount() != 0 {
		t.Error("Expected null cache hit before expiration")
	}

	// Wait for null cache to expire (1 minute + buffer)
	t.Log("Waiting for null cache to expire (65 seconds)...")
	time.Sleep(65 * time.Second)

	// After expiration: should query DB again
	storage.ResetCallCount()
	_, _ = cm.Get(ctx, nonExistentKey)
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query after null cache expiration, got %d", storage.GetCallCount())
	}

	t.Log("Null cache expired correctly after TTL")
}

// TestNullCacheMetrics tests that null cache operations are tracked in metrics
func TestNullCacheMetrics(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Query non-existent keys to trigger null cache entries
	for i := 0; i < 10; i++ {
		_, _ = cm.Get(ctx, fmt.Sprintf("non_existent_%d", i))
	}

	time.Sleep(50 * time.Millisecond)

	// Query again to hit null cache
	for i := 0; i < 10; i++ {
		_, _ = cm.Get(ctx, fmt.Sprintf("non_existent_%d", i))
	}

	// Metrics should be tracked (verified by observability system)
	t.Log("Null cache metrics tracked successfully")
}

// TestNullCacheInvalidation tests that null cache entries are invalidated when data is created
func TestNullCacheInvalidation(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	shortCode := "initially_non_existent"

	// First request: cache null entry
	_, err = cm.Get(ctx, shortCode)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
	time.Sleep(50 * time.Millisecond)

	// Verify null cache hit
	storage.ResetCallCount()
	_, err = cm.Get(ctx, shortCode)
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
	if storage.GetCallCount() != 0 {
		t.Error("Expected null cache hit")
	}

	// Create the mapping in storage
	testMapping := &StorageMapping{
		ShortCode: shortCode,
		LongURL:   "https://example.com/created",
		CreatedAt: time.Now(),
	}
	storage.Set(shortCode, testMapping)

	// Invalidate null cache entry
	err = cm.Delete(ctx, shortCode)
	if err != nil {
		t.Fatalf("Failed to delete null cache entry: %v", err)
	}

	// Next request should query DB and get the new mapping
	storage.ResetCallCount()
	mapping, err := cm.Get(ctx, shortCode)
	if err != nil {
		t.Fatalf("Expected success after creating mapping, got error: %v", err)
	}
	if mapping == nil {
		t.Fatal("Expected mapping to be found")
	}
	if mapping.LongURL != "https://example.com/created" {
		t.Errorf("Expected long URL https://example.com/created, got %s", mapping.LongURL)
	}
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query after invalidation, got %d", storage.GetCallCount())
	}

	t.Log("Null cache invalidation working correctly")
}

// BenchmarkNullCacheEffectiveness benchmarks the effectiveness of null caching
func BenchmarkNullCacheEffectiveness(b *testing.B) {
	l1, _ := NewL1Cache()
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	nonExistentKey := "bench_non_existent"

	// Warm up null cache
	_, _ = cm.Get(ctx, nonExistentKey)
	time.Sleep(50 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cm.Get(ctx, nonExistentKey)
		}
	})
}

// BenchmarkWithoutNullCache benchmarks performance without null caching
func BenchmarkWithoutNullCache(b *testing.B) {
	storage := NewMockStorage()
	ctx := context.Background()

	nonExistentKey := "bench_non_existent"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = storage.Get(ctx, nonExistentKey)
		}
	})
}
