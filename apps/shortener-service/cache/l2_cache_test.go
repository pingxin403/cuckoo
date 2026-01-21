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

	cache, err := NewL2Cache(config)
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

	err := cache.Set(ctx, shortCode, longURL, createdAt)
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
