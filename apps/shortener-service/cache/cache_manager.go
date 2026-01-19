package cache

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

// Storage interface for database operations
type Storage interface {
	Get(ctx context.Context, shortCode string) (*StorageMapping, error)
	Exists(ctx context.Context, shortCode string) (bool, error)
}

// StorageMapping represents a URL mapping from storage
type StorageMapping struct {
	ShortCode string
	LongURL   string
	CreatedAt time.Time
	ExpiresAt *time.Time
	CreatorIP string
}

// CacheManager manages multi-tier caching with singleflight
// Requirements: 3.2, 3.3, 4.4, 12.1, 12.2, 12.5
type CacheManager struct {
	l1      *L1Cache
	l2      *L2Cache
	storage Storage
	sf      singleflight.Group
}

// NewCacheManager creates a new cache manager
func NewCacheManager(l1 *L1Cache, l2 *L2Cache, storage Storage) *CacheManager {
	return &CacheManager{
		l1:      l1,
		l2:      l2,
		storage: storage,
	}
}

// Get retrieves a URL mapping with multi-tier fallback and singleflight
// Flow: L1 → L2 → DB, with backfilling on misses
// Requirements: 3.2, 3.3, 4.4, 12.1, 12.2, 12.5
func (cm *CacheManager) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Use singleflight to coalesce concurrent requests for the same key
	v, err, _ := cm.sf.Do(shortCode, func() (interface{}, error) {
		return cm.getWithFallback(ctx, shortCode)
	})

	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	return v.(*URLMapping), nil
}

// getWithFallback implements the multi-tier cache fallback logic
func (cm *CacheManager) getWithFallback(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Try L1 cache first
	if mapping := cm.l1.Get(shortCode); mapping != nil {
		return mapping, nil
	}

	// Try L2 cache
	if cm.l2 != nil {
		mapping, err := cm.l2.Get(ctx, shortCode)
		if err == nil && mapping != nil {
			// Backfill L1 cache
			cm.l1.Set(mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
			return mapping, nil
		}
		// Continue to DB on L2 miss or error (graceful degradation)
	}

	// Fallback to database
	storageMapping, err := cm.storage.Get(ctx, shortCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get from storage: %w", err)
	}

	// Convert storage mapping to cache mapping
	mapping := &URLMapping{
		ShortCode: storageMapping.ShortCode,
		LongURL:   storageMapping.LongURL,
		CreatedAt: storageMapping.CreatedAt,
	}

	// Backfill both cache layers
	cm.l1.Set(mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
	if cm.l2 != nil {
		_ = cm.l2.Set(ctx, mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
	}

	return mapping, nil
}

// Delete removes a URL mapping from all cache layers
// Requirements: 4.6
func (cm *CacheManager) Delete(ctx context.Context, shortCode string) error {
	// Delete from L1
	cm.l1.Delete(shortCode)

	// Delete from L2
	if cm.l2 != nil {
		if err := cm.l2.Delete(ctx, shortCode); err != nil {
			return fmt.Errorf("failed to delete from L2: %w", err)
		}
	}

	return nil
}

// Set stores a URL mapping in all cache layers
// Requirements: 4.3
func (cm *CacheManager) Set(ctx context.Context, shortCode string, longURL string, createdAt time.Time) error {
	// Set in L1
	cm.l1.Set(shortCode, longURL, createdAt)

	// Set in L2
	if cm.l2 != nil {
		if err := cm.l2.Set(ctx, shortCode, longURL, createdAt); err != nil {
			return fmt.Errorf("failed to set in L2: %w", err)
		}
	}

	return nil
}
