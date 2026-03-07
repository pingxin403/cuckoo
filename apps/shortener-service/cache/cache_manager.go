package cache

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"golang.org/x/sync/singleflight"
)

var (
	// shortCodeRegex validates shortCode format: 4-20 alphanumeric characters and hyphens
	shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{4,20}$`)
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

type CacheManager struct {
	l1      *L1Cache
	l2      *L2Cache
	storage Storage
	sf      singleflight.Group
	obs     observability.Observability
}

// NewCacheManager creates a new cache manager
func NewCacheManager(l1 *L1Cache, l2 *L2Cache, storage Storage, obs observability.Observability) *CacheManager {
	return &CacheManager{
		l1:      l1,
		l2:      l2,
		storage: storage,
		obs:     obs,
	}
}

// Get retrieves a URL mapping with multi-tier fallback and singleflight
// Flow: L1 → L2 → DB, with backfilling on misses
func (cm *CacheManager) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Validate shortCode format to prevent cache penetration attacks
	if !shortCodeRegex.MatchString(shortCode) {
		cm.obs.Metrics().IncrementCounter("shortener_invalid_shortcode_total", map[string]string{"reason": "invalid_format"})
		return nil, nil // Invalid format, return nil without querying
	}

	// Try L1 cache first (fast path, no singleflight needed)
	if mapping := cm.l1.Get(shortCode); mapping != nil {
		// Check if it's a nil marker (cached "not found")
		if mapping.ShortCode == "__NIL__" {
			cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L1", "type": "nil_marker"})
			return nil, nil
		}
		cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L1"})
		cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "hit", "layer": "l1"})
		return mapping, nil
	}
	cm.obs.Metrics().IncrementCounter("shortener_cache_misses_total", map[string]string{"layer": "L1"})
	cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "miss", "layer": "l1"})

	// Use singleflight only for L2 and DB lookups (cache miss scenario)
	v, err, _ := cm.sf.Do(shortCode, func() (interface{}, error) {
		return cm.getFromL2OrDB(ctx, shortCode)
	})

	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	return v.(*URLMapping), nil
}

// getFromL2OrDB implements L2 cache and database fallback logic
func (cm *CacheManager) getFromL2OrDB(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Try L2 cache
	if cm.l2 != nil {
		mapping, err := cm.l2.Get(ctx, shortCode)
		if err == nil && mapping != nil {
			// Check if it's a nil marker
			if mapping.ShortCode == "__NIL__" {
				cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L2", "type": "nil_marker"})
				// Backfill L1 cache with nil marker
				cm.l1.SetNilMarker(shortCode)
				return nil, nil
			}
			cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L2"})
			cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "hit", "layer": "l2"})
			// Backfill L1 cache
			cm.l1.Set(mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
			return mapping, nil
		}
		cm.obs.Metrics().IncrementCounter("shortener_cache_misses_total", map[string]string{"layer": "L2"})
		cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "miss", "layer": "l2"})
		// Continue to DB on L2 miss or error (graceful degradation)
	}

	// Fallback to database
	storageMapping, err := cm.storage.Get(ctx, shortCode)
	if err != nil {
		// If storage returns error (not found), cache nil marker
		cm.obs.Metrics().IncrementCounter("shortener_cache_nil_marker_set_total", map[string]string{"reason": "not_found"})
		cm.l1.SetNilMarker(shortCode)
		if cm.l2 != nil {
			_ = cm.l2.SetNilMarker(ctx, shortCode)
		}
		return nil, err
	}

	// If not found in database (nil without error), cache nil marker
	if storageMapping == nil {
		cm.obs.Metrics().IncrementCounter("shortener_cache_nil_marker_set_total", map[string]string{"reason": "not_found"})
		cm.l1.SetNilMarker(shortCode)
		if cm.l2 != nil {
			_ = cm.l2.SetNilMarker(ctx, shortCode)
		}
		return nil, nil
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
