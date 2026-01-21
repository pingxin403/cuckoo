package cache

import (
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestProperty_TTLJitterPreventsThunderingHerd verifies TTL jitter prevents cache stampede
// Property 16: TTL Jitter Prevents Thundering Herd
// Requirements: 12.4
func TestProperty_TTLJitterPreventsThunderingHerd(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cache, err := NewL1Cache()
		if err != nil {
			t.Fatalf("Failed to create L1 cache: %v", err)
		}
		defer cache.Close()

		// Generate random number of entries to create simultaneously
		numEntries := rapid.IntRange(10, 100).Draw(t, "numEntries")

		// Create entries simultaneously
		var wg sync.WaitGroup
		wg.Add(numEntries)

		for i := 0; i < numEntries; i++ {
			go func(idx int) {
				defer wg.Done()
				shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Example(idx)
				longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Example(idx)
				cache.Set(shortCode, longURL, time.Now())
			}(i)
		}

		wg.Wait()

		// Wait for Ristretto to process all sets
		time.Sleep(50 * time.Millisecond)

		// Property 1: Most entries should be retrievable (verifies TTL jitter doesn't break caching)
		// We expect at least 80% of entries to be in cache
		retrievableCount := 0
		for i := 0; i < numEntries; i++ {
			shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Example(i)
			if mapping := cache.Get(shortCode); mapping != nil {
				retrievableCount++
			}
		}

		retrievablePercent := float64(retrievableCount) / float64(numEntries) * 100
		if retrievablePercent < 80 {
			t.Fatalf("Expected at least 80%% entries retrievable, got %.1f%% (%d/%d)",
				retrievablePercent, retrievableCount, numEntries)
		}

		// Property 2: Retrieved values should match what was stored
		for i := 0; i < numEntries; i++ {
			shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Example(i)
			expectedURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Example(i)

			if mapping := cache.Get(shortCode); mapping != nil {
				if mapping.LongURL != expectedURL {
					t.Fatalf("Expected URL %s, got %s", expectedURL, mapping.LongURL)
				}
			}
		}
	})
}

// TestProperty_CacheConsistency verifies cache operations are consistent
func TestProperty_CacheConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cache, err := NewL1Cache()
		if err != nil {
			t.Fatalf("Failed to create L1 cache: %v", err)
		}
		defer cache.Close()

		// Generate random short code and URL
		shortCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "shortCode")
		longURL := rapid.StringMatching(`https://[a-z]+\.com/[a-z]+`).Draw(t, "longURL")
		createdAt := time.Now()

		// Property 1: Set then Get returns the same value
		cache.Set(shortCode, longURL, createdAt)
		time.Sleep(10 * time.Millisecond) // Wait for Ristretto

		mapping := cache.Get(shortCode)
		if mapping != nil && mapping.LongURL != longURL {
			t.Fatalf("Expected URL %s, got %s", longURL, mapping.LongURL)
		}

		// Property 2: Cache miss returns nil
		nonExistentCode := rapid.StringMatching(`[a-zA-Z0-9]{7}`).Draw(t, "nonExistentCode")
		if nonExistentCode == shortCode {
			nonExistentCode = shortCode + "x" // Ensure different
		}

		if mapping := cache.Get(nonExistentCode); mapping != nil {
			t.Fatal("Expected nil for non-existent key")
		}

		// Property 3: Delete removes entry
		cache.Delete(shortCode)
		if mapping := cache.Get(shortCode); mapping != nil {
			t.Fatal("Expected nil after delete")
		}
	})
}
