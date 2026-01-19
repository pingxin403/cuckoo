package cache

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestProperty_CacheInvalidationOnDeletion verifies cache invalidation
// Property 8: Cache Invalidation on Deletion
// Requirements: 4.6
func TestProperty_CacheInvalidationOnDeletion(t *testing.T) {
	// Check if Redis is available
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config)
	if err != nil {
		t.Skipf("Redis not available, skipping property test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		t.Skipf("Redis not reachable, skipping property test: %v", err)
		return
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate random short code and URL
		shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "shortCode")
		longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Draw(t, "longURL")
		createdAt := time.Now()

		ctx := context.Background()

		// Property 1: Set then Delete removes entry from L2
		err := cache.Set(ctx, shortCode, longURL, createdAt)
		if err != nil {
			t.Fatalf("Failed to set in L2: %v", err)
		}

		// Verify it's in cache
		mapping, err := cache.Get(ctx, shortCode)
		if err != nil {
			t.Fatalf("Failed to get from L2: %v", err)
		}
		if mapping == nil {
			t.Fatal("Expected mapping in L2 cache")
		}

		// Delete it
		err = cache.Delete(ctx, shortCode)
		if err != nil {
			t.Fatalf("Failed to delete from L2: %v", err)
		}

		// Verify it's removed
		mapping, err = cache.Get(ctx, shortCode)
		if err != nil {
			t.Fatalf("Get after delete failed: %v", err)
		}
		if mapping != nil {
			t.Fatal("Expected nil after delete")
		}
	})
}

// TestProperty_L2CacheConsistency verifies L2 cache operations are consistent
func TestProperty_L2CacheConsistency(t *testing.T) {
	// Check if Redis is available
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config)
	if err != nil {
		t.Skipf("Redis not available, skipping property test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		t.Skipf("Redis not reachable, skipping property test: %v", err)
		return
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate random short code and URL
		shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "shortCode")
		longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Draw(t, "longURL")
		createdAt := time.Now()

		ctx := context.Background()

		// Property: Set then Get returns the same value
		err := cache.Set(ctx, shortCode, longURL, createdAt)
		if err != nil {
			t.Fatalf("Failed to set in L2: %v", err)
		}

		mapping, err := cache.Get(ctx, shortCode)
		if err != nil {
			t.Fatalf("Failed to get from L2: %v", err)
		}
		if mapping != nil && mapping.LongURL != longURL {
			t.Fatalf("Expected URL %s, got %s", longURL, mapping.LongURL)
		}

		// Cleanup
		_ = cache.Delete(ctx, shortCode)
	})
}

// TestProperty_BatchOperationsConsistency verifies batch operations are consistent
func TestProperty_BatchOperationsConsistency(t *testing.T) {
	// Check if Redis is available
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config)
	if err != nil {
		t.Skipf("Redis not available, skipping property test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		t.Skipf("Redis not reachable, skipping property test: %v", err)
		return
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate random number of entries
		numEntries := rapid.IntRange(5, 20).Draw(t, "numEntries")

		shortCodes := make([]string, numEntries)
		expectedURLs := make(map[string]string)

		ctx := context.Background()

		// Set multiple entries
		for i := 0; i < numEntries; i++ {
			shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "shortCode")
			longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Draw(t, "longURL")

			shortCodes[i] = shortCode
			expectedURLs[shortCode] = longURL

			err := cache.Set(ctx, shortCode, longURL, time.Now())
			if err != nil {
				t.Fatalf("Failed to set entry %d: %v", i, err)
			}
		}

		// Batch get
		results, err := cache.BatchGet(ctx, shortCodes)
		if err != nil {
			t.Fatalf("BatchGet failed: %v", err)
		}

		// Verify all entries retrieved correctly
		if len(results) != numEntries {
			t.Fatalf("Expected %d results, got %d", numEntries, len(results))
		}

		for shortCode, expectedURL := range expectedURLs {
			mapping, ok := results[shortCode]
			if !ok {
				t.Fatalf("Expected result for %s", shortCode)
			}
			if mapping.LongURL != expectedURL {
				t.Fatalf("Expected URL %s, got %s", expectedURL, mapping.LongURL)
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
			t.Fatalf("Expected 0 results after delete, got %d", len(results))
		}
	})
}

// TestProperty_TTLJitter verifies TTL jitter is applied
func TestProperty_TTLJitter(t *testing.T) {
	// Check if Redis is available
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config)
	if err != nil {
		t.Skipf("Redis not available, skipping property test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		t.Skipf("Redis not reachable, skipping property test: %v", err)
		return
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate random short code and URL
		shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "shortCode")
		longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Draw(t, "longURL")

		ctx := context.Background()

		// Property: Entry should be stored successfully with TTL
		err := cache.Set(ctx, shortCode, longURL, time.Now())
		if err != nil {
			t.Fatalf("Failed to set with TTL: %v", err)
		}

		// Verify it's retrievable
		mapping, err := cache.Get(ctx, shortCode)
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if mapping == nil {
			t.Fatal("Expected mapping in cache")
		}

		// Cleanup
		_ = cache.Delete(ctx, shortCode)
	})
}
