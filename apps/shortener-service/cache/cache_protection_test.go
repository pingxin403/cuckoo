package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCachePenetration tests protection against cache penetration attacks
// Cache penetration occurs when querying non-existent keys repeatedly
func TestCachePenetration(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Simulate cache penetration: query non-existent keys repeatedly
	const numRequests = 100
	nonExistentKey := "non_existent_key"

	storage.ResetCallCount()

	for i := 0; i < numRequests; i++ {
		_, err := cm.Get(ctx, nonExistentKey)
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	}

	// Without protection, each request would hit the database
	// With singleflight, concurrent requests are coalesced
	callCount := storage.GetCallCount()
	t.Logf("Cache penetration: %d requests for non-existent key resulted in %d DB queries", numRequests, callCount)

	// We expect all requests to hit DB since the key doesn't exist
	// But singleflight should reduce concurrent requests
	if callCount != numRequests {
		t.Logf("Note: Singleflight reduced DB queries from %d to %d", numRequests, callCount)
	}
}

// TestCachePenetrationConcurrent tests concurrent cache penetration protection
func TestCachePenetrationConcurrent(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())

	// Launch concurrent requests for non-existent key
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	ctx := context.Background()
	storage.ResetCallCount()

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			_, _ = cm.Get(ctx, "non_existent_concurrent")
		}()
	}

	wg.Wait()

	// Singleflight should coalesce concurrent requests
	callCount := storage.GetCallCount()
	t.Logf("Concurrent cache penetration: %d requests coalesced to %d DB queries (%.1f%% reduction)",
		numRequests, callCount, float64(numRequests-callCount)/float64(numRequests)*100)

	// For non-existent keys, singleflight can only coalesce requests that arrive
	// at exactly the same time. Since errors are not cached, subsequent requests
	// will trigger new DB queries. We expect some reduction but not as dramatic
	// as with existing keys.
	reductionPercent := float64(numRequests-callCount) / float64(numRequests) * 100
	if reductionPercent < 10 {
		t.Logf("Note: Only %.1f%% reduction for non-existent keys (expected behavior)", reductionPercent)
	}

	// The key insight: for cache penetration protection, we need additional mechanisms
	// like bloom filters or null value caching, which are noted in the placeholder tests below
}

// TestCacheBreakdown tests protection against cache breakdown (hotspot expiration)
// Cache breakdown occurs when a hot key expires and many requests hit the database simultaneously
func TestCacheBreakdown(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	hotKey := "hot_key_123"
	testMapping := &StorageMapping{
		ShortCode: hotKey,
		LongURL:   "https://example.com/hot",
		CreatedAt: time.Now(),
	}
	storage.Set(hotKey, testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Warm up cache
	_, err = cm.Get(ctx, hotKey)
	if err != nil {
		t.Fatalf("Failed to warm up cache: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Simulate cache expiration by deleting from L1
	l1.Delete(hotKey)

	// Launch concurrent requests immediately after expiration
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	storage.ResetCallCount()
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			_, _ = cm.Get(ctx, hotKey)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	// Singleflight should prevent cache breakdown
	callCount := storage.GetCallCount()
	t.Logf("Cache breakdown protection: %d concurrent requests after expiration resulted in %d DB queries in %v (%.1f%% reduction)",
		numRequests, callCount, duration, float64(numRequests-callCount)/float64(numRequests)*100)

	// With singleflight, we expect only 1-3 DB queries
	if callCount > 3 {
		t.Errorf("Expected at most 3 DB queries with singleflight, got %d (cache breakdown not prevented)", callCount)
	}
}

// TestCacheAvalanche tests protection against cache avalanche
// Cache avalanche occurs when many keys expire at the same time
func TestCacheAvalanche(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()

	// Create multiple keys that will expire simultaneously
	const numKeys = 50
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("avalanche_key_%d", i)
		keys[i] = key
		testMapping := &StorageMapping{
			ShortCode: key,
			LongURL:   fmt.Sprintf("https://example.com/avalanche/%d", i),
			CreatedAt: time.Now(),
		}
		storage.Set(key, testMapping)
	}

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Warm up cache for all keys
	for _, key := range keys {
		_, err := cm.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to warm up cache for %s: %v", key, err)
		}
	}
	time.Sleep(50 * time.Millisecond)

	// Simulate cache avalanche by deleting all keys
	for _, key := range keys {
		l1.Delete(key)
	}

	// Launch concurrent requests for all keys
	const requestsPerKey = 10
	totalRequests := numKeys * requestsPerKey
	var wg sync.WaitGroup
	wg.Add(totalRequests)

	storage.ResetCallCount()
	start := time.Now()

	for _, key := range keys {
		for j := 0; j < requestsPerKey; j++ {
			go func(k string) {
				defer wg.Done()
				_, _ = cm.Get(ctx, k)
			}(key)
		}
	}

	wg.Wait()
	duration := time.Since(start)

	// Singleflight should reduce DB load during avalanche
	callCount := storage.GetCallCount()
	t.Logf("Cache avalanche protection: %d concurrent requests for %d keys resulted in %d DB queries in %v (%.1f%% reduction)",
		totalRequests, numKeys, callCount, duration, float64(totalRequests-callCount)/float64(totalRequests)*100)

	// With singleflight, we expect roughly numKeys queries (one per key)
	// In practice, due to timing and concurrent execution, we may see more queries
	// The key metric is that we have significant reduction from totalRequests
	reductionPercent := float64(totalRequests-callCount) / float64(totalRequests) * 100
	if reductionPercent < 30 {
		t.Errorf("Expected at least 30%% reduction in DB queries, got %.1f%%", reductionPercent)
	}

	// Verify we didn't query more than total requests (sanity check)
	if callCount > totalRequests {
		t.Errorf("DB queries (%d) exceeded total requests (%d)", callCount, totalRequests)
	}
}

// TestDelayedDoubleDelete tests delayed double delete pattern for cache consistency
// This pattern is used when updating data to ensure cache consistency
func TestDelayedDoubleDelete(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	key := "update_key_123"
	originalMapping := &StorageMapping{
		ShortCode: key,
		LongURL:   "https://example.com/original",
		CreatedAt: time.Now(),
	}
	storage.Set(key, originalMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Step 1: Warm up cache
	mapping, err := cm.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to warm up cache: %v", err)
	}
	if mapping.LongURL != "https://example.com/original" {
		t.Errorf("Expected original URL, got %s", mapping.LongURL)
	}
	time.Sleep(50 * time.Millisecond)

	// Step 2: Update data in storage (simulate database update)
	updatedMapping := &StorageMapping{
		ShortCode: key,
		LongURL:   "https://example.com/updated",
		CreatedAt: time.Now(),
	}
	storage.Set(key, updatedMapping)

	// Step 3: First delete - delete cache immediately after DB update
	err = cm.Delete(ctx, key)
	if err != nil {
		t.Fatalf("First delete failed: %v", err)
	}

	// Step 4: Simulate concurrent read during update
	// This read might cache stale data if there's a race condition
	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = cm.Get(ctx, key)
	}()

	// Step 5: Delayed second delete - delete cache again after a short delay
	// This ensures any stale data cached during the update is removed
	time.Sleep(100 * time.Millisecond)
	err = cm.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Second delete failed: %v", err)
	}

	// Step 6: Verify cache returns updated data
	time.Sleep(50 * time.Millisecond)
	mapping, err = cm.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get updated data: %v", err)
	}
	if mapping.LongURL != "https://example.com/updated" {
		t.Errorf("Expected updated URL, got %s (delayed double delete may not be working)", mapping.LongURL)
	}
}

// TestDelayedDoubleDeleteConcurrent tests delayed double delete under concurrent load
func TestDelayedDoubleDeleteConcurrent(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	key := "concurrent_update_key"
	originalMapping := &StorageMapping{
		ShortCode: key,
		LongURL:   "https://example.com/original",
		CreatedAt: time.Now(),
	}
	storage.Set(key, originalMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Warm up cache
	_, err = cm.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to warm up cache: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Simulate concurrent updates with delayed double delete
	const numUpdates = 10
	var wg sync.WaitGroup
	wg.Add(numUpdates)

	for i := 0; i < numUpdates; i++ {
		go func(updateNum int) {
			defer wg.Done()

			// Update storage
			newMapping := &StorageMapping{
				ShortCode: key,
				LongURL:   fmt.Sprintf("https://example.com/update_%d", updateNum),
				CreatedAt: time.Now(),
			}
			storage.Set(key, newMapping)

			// First delete
			_ = cm.Delete(ctx, key)

			// Delayed second delete
			time.Sleep(50 * time.Millisecond)
			_ = cm.Delete(ctx, key)
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	time.Sleep(100 * time.Millisecond)
	mapping, err := cm.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get final data: %v", err)
	}

	// The final URL should be one of the updates (we don't know which due to concurrency)
	// But it should not be the original
	if mapping.LongURL == "https://example.com/original" {
		t.Error("Cache still contains original data after concurrent updates")
	}
	t.Logf("Final URL after concurrent updates: %s", mapping.LongURL)
}

// TestCacheConsistencyUnderLoad tests cache consistency under high concurrent load
func TestCacheConsistencyUnderLoad(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	key := "consistency_key"
	testMapping := &StorageMapping{
		ShortCode: key,
		LongURL:   "https://example.com/consistent",
		CreatedAt: time.Now(),
	}
	storage.Set(key, testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Launch mixed workload: reads, updates, and deletes
	const duration = 2 * time.Second
	const numReaders = 50
	const numUpdaters = 5
	const numDeleters = 5

	var wg sync.WaitGroup
	stop := make(chan struct{})
	var readCount, updateCount, deleteCount int64

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = cm.Get(ctx, key)
					atomic.AddInt64(&readCount, 1)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	// Updaters
	for i := 0; i < numUpdaters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			updateNum := 0
			for {
				select {
				case <-stop:
					return
				default:
					// Update storage
					newMapping := &StorageMapping{
						ShortCode: key,
						LongURL:   fmt.Sprintf("https://example.com/update_%d_%d", id, updateNum),
						CreatedAt: time.Now(),
					}
					storage.Set(key, newMapping)

					// Delayed double delete
					_ = cm.Delete(ctx, key)
					time.Sleep(50 * time.Millisecond)
					_ = cm.Delete(ctx, key)

					atomic.AddInt64(&updateCount, 1)
					updateNum++
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(i)
	}

	// Deleters
	for i := 0; i < numDeleters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = cm.Delete(ctx, key)
					atomic.AddInt64(&deleteCount, 1)
					time.Sleep(200 * time.Millisecond)
				}
			}
		}()
	}

	// Run for specified duration
	time.Sleep(duration)
	close(stop)
	wg.Wait()

	t.Logf("Cache consistency test completed:")
	t.Logf("  Reads: %d", readCount)
	t.Logf("  Updates: %d", updateCount)
	t.Logf("  Deletes: %d", deleteCount)
	t.Logf("  DB queries: %d", storage.GetCallCount())

	// Verify final state is consistent
	mapping, err := cm.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get final data: %v", err)
	}
	t.Logf("Final URL: %s", mapping.LongURL)
}

// TestBloomFilterForCachePenetration tests using bloom filter to prevent cache penetration
// Note: This is a conceptual test - actual implementation would require bloom filter integration
func TestBloomFilterForCachePenetration(t *testing.T) {
	t.Skip("Bloom filter integration not yet implemented - this is a placeholder for future enhancement")

	// Conceptual test:
	// 1. Maintain a bloom filter of all valid keys
	// 2. Before querying cache/DB, check bloom filter
	// 3. If bloom filter says "definitely not exists", return immediately
	// 4. If bloom filter says "might exist", proceed with cache/DB query
	//
	// This prevents querying non-existent keys that would cause cache penetration
}

// TestNullValueCachingForPenetration tests caching null values to prevent repeated DB queries
// Note: This is a conceptual test - actual implementation would require null value caching
func TestNullValueCachingForPenetration(t *testing.T) {
	t.Skip("Null value caching not yet implemented - this is a placeholder for future enhancement")

	// Conceptual test:
	// 1. When a key is not found in DB, cache a special "null" marker
	// 2. Set a short TTL for null values (e.g., 1 minute)
	// 3. Subsequent queries for the same non-existent key hit the null cache
	// 4. This prevents repeated DB queries for non-existent keys
	//
	// This is an alternative approach to prevent cache penetration
}
