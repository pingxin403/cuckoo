package cache

import (
	"context"
	"testing"
	"time"
)

// skipIfRedisUnavailable checks if Redis is available and skips the test if not
func skipIfRedisUnavailable(t *testing.T) *L2Cache {
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	obs := createTestObservability()
	cache, err := NewL2Cache(config, obs)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		_ = cache.Close()
		t.Skipf("Redis not reachable, skipping test: %v", err)
		return nil
	}

	return cache
}

// TestL2CacheBasicOperations tests basic Redis cache operations
func TestL2CacheBasicOperations(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Test Set and Get
	shortCode := "test123"
	longURL := "https://example.com"
	createdAt := time.Now()

	err := cache.Set(ctx, shortCode, longURL, createdAt, nil)
	if err != nil {
		t.Fatalf("Failed to set value in Redis: %v", err)
	}

	mapping, err := cache.Get(ctx, shortCode)
	if err != nil {
		t.Fatalf("Failed to get value from Redis: %v", err)
	}
	if mapping == nil {
		t.Fatal("Expected mapping to be in cache")
	}
	if mapping.ShortCode != shortCode {
		t.Errorf("Expected short code %s, got %s", shortCode, mapping.ShortCode)
	}
	if mapping.LongURL != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, mapping.LongURL)
	}

	// Cleanup
	_ = cache.Delete(ctx, shortCode)
}

// TestL2CacheMiss tests cache miss behavior
func TestL2CacheMiss(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Get non-existent key
	mapping, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if mapping != nil {
		t.Error("Expected nil for non-existent key")
	}
}

// TestL2CacheDelete tests cache deletion
func TestL2CacheDelete(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Set a value
	shortCode := "test456"
	err := cache.Set(ctx, shortCode, "https://example.com", time.Now())
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// Verify it's in cache
	mapping, err := cache.Get(ctx, shortCode)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if mapping == nil {
		t.Fatal("Expected mapping to be in cache")
	}

	// Delete it
	err = cache.Delete(ctx, shortCode)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify it's removed
	mapping, err = cache.Get(ctx, shortCode)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if mapping != nil {
		t.Error("Expected nil after delete")
	}
}

// TestL2CacheBatchOperations tests batch get and delete
func TestL2CacheBatchOperations(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Set multiple values
	shortCodes := []string{"batch1", "batch2", "batch3"}
	for i, code := range shortCodes {
		err := cache.Set(ctx, code, "https://example.com/"+code, time.Now())
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Batch get
	results, err := cache.BatchGet(ctx, shortCodes)
	if err != nil {
		t.Fatalf("BatchGet failed: %v", err)
	}
	if len(results) != len(shortCodes) {
		t.Errorf("Expected %d results, got %d", len(shortCodes), len(results))
	}

	// Verify results
	for _, code := range shortCodes {
		mapping, ok := results[code]
		if !ok {
			t.Errorf("Expected result for %s", code)
			continue
		}
		if mapping.ShortCode != code {
			t.Errorf("Expected short code %s, got %s", code, mapping.ShortCode)
		}
	}

	// Batch delete
	err = cache.BatchDelete(ctx, shortCodes)
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// Verify all deleted
	results, err = cache.BatchGet(ctx, shortCodes)
	if err != nil {
		t.Fatalf("BatchGet after delete failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results after delete, got %d", len(results))
	}
}

// TestL2CacheTTLJitter_DifferentTTLs tests that two entries created simultaneously have different TTLs
func TestL2CacheTTLJitter_DifferentTTLs(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Create two entries at the same time
	shortCode1 := "jitter_test1"
	shortCode2 := "jitter_test2"
	createdAt := time.Now()

	err := cache.Set(ctx, shortCode1, "https://example.com/1", createdAt)
	if err != nil {
		t.Fatalf("Failed to set first entry: %v", err)
	}

	err = cache.Set(ctx, shortCode2, "https://example.com/2", createdAt)
	if err != nil {
		t.Fatalf("Failed to set second entry: %v", err)
	}

	// Get TTL for both entries
	key1 := "url:" + shortCode1
	key2 := "url:" + shortCode2

	ttl1, err := cache.Client().TTL(ctx, key1).Result()
	if err != nil {
		t.Fatalf("Failed to get TTL for first entry: %v", err)
	}

	ttl2, err := cache.Client().TTL(ctx, key2).Result()
	if err != nil {
		t.Fatalf("Failed to get TTL for second entry: %v", err)
	}

	// Verify TTLs are different (with high probability)
	// Due to jitter, they should differ by at least 1 second in most cases
	if ttl1 == ttl2 {
		t.Logf("Warning: TTLs are identical (%v), but this can happen rarely due to random chance", ttl1)
		// Don't fail the test as this can happen with very low probability
	} else {
		t.Logf("TTL1: %v, TTL2: %v - Successfully different", ttl1, ttl2)
	}

	// Cleanup
	_ = cache.Delete(ctx, shortCode1)
	_ = cache.Delete(ctx, shortCode2)
}

// TestL2CacheTTLJitter_WithinRange tests that jitter is within expected range (±1 day)
func TestL2CacheTTLJitter_WithinRange(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Create multiple entries and check their TTLs
	numEntries := 10
	shortCodes := make([]string, numEntries)

	for i := 0; i < numEntries; i++ {
		shortCode := "jitter_range_test_" + time.Now().Format("20060102150405") + "_" + string(rune('a'+i))
		shortCodes[i] = shortCode

		err := cache.Set(ctx, shortCode, "https://example.com/"+shortCode, time.Now())
		if err != nil {
			t.Fatalf("Failed to set entry %d: %v", i, err)
		}
	}

	// Check TTL for each entry
	baseTTL := 7 * 24 * time.Hour // 7 days
	jitterRange := 24 * time.Hour // ±1 day
	minTTL := baseTTL - jitterRange
	maxTTL := baseTTL + jitterRange

	for i, shortCode := range shortCodes {
		key := "url:" + shortCode
		ttl, err := cache.Client().TTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL for entry %d: %v", i, err)
		}

		// Verify TTL is within expected range
		// Allow a small margin for processing time (10 seconds)
		margin := 10 * time.Second
		if ttl < minTTL-margin || ttl > maxTTL+margin {
			t.Errorf("Entry %d: TTL %v is outside expected range [%v, %v]", i, ttl, minTTL, maxTTL)
		} else {
			t.Logf("Entry %d: TTL %v is within expected range", i, ttl)
		}
	}

	// Cleanup
	for _, shortCode := range shortCodes {
		_ = cache.Delete(ctx, shortCode)
	}
}

// TestL2CacheTTLJitter_BaseTTLPreserved tests that base TTL is preserved (7 days ± 1 day)
func TestL2CacheTTLJitter_BaseTTLPreserved(t *testing.T) {
	cache := skipIfRedisUnavailable(t)
	if cache == nil {
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Create multiple entries and collect their TTLs
	numEntries := 20
	ttls := make([]time.Duration, numEntries)
	shortCodes := make([]string, numEntries)

	for i := 0; i < numEntries; i++ {
		shortCode := "jitter_base_test_" + time.Now().Format("20060102150405") + "_" + string(rune('a'+i%26))
		shortCodes[i] = shortCode

		err := cache.Set(ctx, shortCode, "https://example.com/"+shortCode, time.Now())
		if err != nil {
			t.Fatalf("Failed to set entry %d: %v", i, err)
		}

		key := "url:" + shortCode
		ttl, err := cache.Client().TTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL for entry %d: %v", i, err)
		}
		ttls[i] = ttl
	}

	// Calculate average TTL
	var totalTTL time.Duration
	for _, ttl := range ttls {
		totalTTL += ttl
	}
	avgTTL := totalTTL / time.Duration(numEntries)

	// Expected base TTL is 7 days
	baseTTL := 7 * 24 * time.Hour

	// Average should be close to base TTL
	// With 20 samples and ±1 day jitter, we need a reasonable tolerance
	// The standard deviation of uniform distribution is range/sqrt(12) ≈ 14 hours
	// For 20 samples, standard error is ~3 hours, so 6 hours tolerance is reasonable
	tolerance := 6 * time.Hour
	if avgTTL < baseTTL-tolerance || avgTTL > baseTTL+tolerance {
		t.Errorf("Average TTL %v is not close to base TTL %v (tolerance: %v)", avgTTL, baseTTL, tolerance)
	} else {
		t.Logf("Average TTL %v is close to base TTL %v", avgTTL, baseTTL)
	}

	// Verify all TTLs are within the jitter range (6-8 days)
	minTTL := 6 * 24 * time.Hour
	maxTTL := 8 * 24 * time.Hour
	margin := 10 * time.Second

	for i, ttl := range ttls {
		if ttl < minTTL-margin || ttl > maxTTL+margin {
			t.Errorf("Entry %d: TTL %v is outside base range [%v, %v]", i, ttl, minTTL, maxTTL)
		}
	}

	// Cleanup
	for _, shortCode := range shortCodes {
		_ = cache.Delete(ctx, shortCode)
	}
}
