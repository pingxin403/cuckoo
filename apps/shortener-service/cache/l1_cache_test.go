package cache

import (
	"testing"
	"time"
)

// TestL1CacheBasicOperations tests basic cache operations
func TestL1CacheBasicOperations(t *testing.T) {
	cache, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer cache.Close()

	// Test Set and Get
	shortCode := "test123"
	longURL := "https://example.com"
	createdAt := time.Now()

	success := cache.Set(shortCode, longURL, createdAt)
	if !success {
		t.Error("Failed to set value in cache")
	}

	// Wait for Ristretto to process the set operation
	time.Sleep(10 * time.Millisecond)

	mapping := cache.Get(shortCode)
	if mapping == nil {
		t.Fatal("Expected mapping to be in cache")
	}
	if mapping.ShortCode != shortCode {
		t.Errorf("Expected short code %s, got %s", shortCode, mapping.ShortCode)
	}
	if mapping.LongURL != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, mapping.LongURL)
	}
}

// TestL1CacheMiss tests cache miss behavior
func TestL1CacheMiss(t *testing.T) {
	cache, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer cache.Close()

	// Get non-existent key
	mapping := cache.Get("nonexistent")
	if mapping != nil {
		t.Error("Expected nil for non-existent key")
	}
}

// TestL1CacheDelete tests cache deletion
func TestL1CacheDelete(t *testing.T) {
	cache, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer cache.Close()

	// Set a value
	shortCode := "test456"
	cache.Set(shortCode, "https://example.com", time.Now())
	time.Sleep(10 * time.Millisecond)

	// Verify it's in cache
	if mapping := cache.Get(shortCode); mapping == nil {
		t.Fatal("Expected mapping to be in cache")
	}

	// Delete it
	cache.Delete(shortCode)

	// Verify it's removed
	if mapping := cache.Get(shortCode); mapping != nil {
		t.Error("Expected mapping to be removed from cache")
	}
}

// TestL1CacheTTL tests TTL expiration
func TestL1CacheTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	cache, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer cache.Close()

	// Note: This test would require waiting for TTL expiration (54-66 minutes)
	// which is impractical for unit tests. TTL behavior is verified by property tests.
	t.Skip("TTL expiration test skipped - requires 54-66 minutes wait time")
}
