package cache

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/dgraph-io/ristretto"
)

// L1Cache represents the in-memory cache layer using Ristretto

type L1Cache struct {
	cache *ristretto.Cache
}

// URLMapping represents a cached URL mapping
type URLMapping struct {
	ShortCode string
	LongURL   string
	CreatedAt time.Time
}

// NewL1Cache creates a new L1 cache instance with Ristretto
// Configuration:
// - MaxCost: 1GB (1 << 30 bytes)
// - NumCounters: 10M (10 * MaxCost)
// - BufferItems: 64 (default)

func NewL1Cache() (*L1Cache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10_000_000, // 10M counters (10 * MaxCost recommended)
		MaxCost:     1 << 30,    // 1GB max cache size
		BufferItems: 64,         // Number of keys per Get buffer
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}

	return &L1Cache{cache: cache}, nil
}

// Get retrieves a URL mapping from the cache
// Returns nil if the key is not found

func (c *L1Cache) Get(shortCode string) *URLMapping {
	value, found := c.cache.Get(shortCode)
	if !found {
		return nil
	}

	mapping, ok := value.(*URLMapping)
	if !ok {
		return nil
	}

	return mapping
}

// Set stores a URL mapping in the cache with TTL jitter
// TTL: 1 hour ±10% (54-66 minutes) to prevent thundering herd
// Cost: estimated size of the mapping in bytes

func (c *L1Cache) Set(shortCode string, longURL string, createdAt time.Time) bool {
	mapping := &URLMapping{
		ShortCode: shortCode,
		LongURL:   longURL,
		CreatedAt: createdAt,
	}

	// Calculate TTL with ±10% jitter to prevent thundering herd
	// Base TTL: 1 hour (3600 seconds)
	// Jitter range: 3240-3960 seconds (54-66 minutes)
	baseTTL := 3600 // 1 hour in seconds
	jitterPercent := 0.1
	jitterRange := int(float64(baseTTL) * jitterPercent)

	// Generate random jitter: -10% to +10%
	jitter := rand.Intn(2*jitterRange+1) - jitterRange // #nosec G404 - weak random is acceptable for cache TTL jitter
	ttlSeconds := baseTTL + jitter
	ttl := time.Duration(ttlSeconds) * time.Second

	// Estimate cost: shortCode + longURL + overhead
	cost := int64(len(shortCode) + len(longURL) + 100)

	return c.cache.SetWithTTL(shortCode, mapping, cost, ttl)
}

// Delete removes a URL mapping from the cache

func (c *L1Cache) Delete(shortCode string) {
	c.cache.Del(shortCode)
}

// Close closes the cache and releases resources
func (c *L1Cache) Close() {
	c.cache.Close()
}

// SetNilMarker caches a "not found" marker to prevent cache penetration
// TTL: 2 minutes (shorter than normal cache to allow for new creations)
func (c *L1Cache) SetNilMarker(shortCode string) bool {
	nilMapping := &URLMapping{
		ShortCode: "__NIL__",
		LongURL:   "",
		CreatedAt: time.Now(),
	}

	// Short TTL for nil markers (2 minutes)
	ttl := 2 * time.Minute
	cost := int64(len(shortCode) + 50) // Smaller cost for nil markers

	return c.cache.SetWithTTL(shortCode, nilMapping, cost, ttl)
}
