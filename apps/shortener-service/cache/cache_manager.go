package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
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

// NullCacheEntry represents a cached "not found" entry to prevent cache penetration
type NullCacheEntry struct {
	ShortCode string
	CachedAt  time.Time
}

// IsNullEntry checks if a URL mapping represents a null cache entry
func IsNullEntry(mapping *URLMapping) bool {
	return mapping != nil && mapping.LongURL == "" && mapping.ShortCode != ""
}

// CreateNullEntry creates a null cache entry for a non-existent key
func CreateNullEntry(shortCode string) *URLMapping {
	return &URLMapping{
		ShortCode: shortCode,
		LongURL:   "", // Empty URL indicates null entry
		CreatedAt: time.Now(),
	}
}

// CacheManager manages multi-tier caching with enhanced singleflight
type CacheManager struct {
	l1       *L1Cache
	l2       *L2Cache
	storage  Storage
	loader   *CacheLoader
	pipeline *PipelineHelper
	sf       *EnhancedSingleflight
	obs      observability.Observability
}

// NewCacheManager creates a new cache manager
func NewCacheManager(l1 *L1Cache, l2 *L2Cache, storage Storage, obs observability.Observability) *CacheManager {
	return &CacheManager{
		l1:      l1,
		l2:      l2,
		storage: storage,
		sf:      NewEnhancedSingleflight(obs),
		obs:     obs,
	}
}

// NewCacheManagerWithLoader creates a new cache manager with CacheLoader
// This constructor enables SETNX-based cache loading to prevent cache stampede
func NewCacheManagerWithLoader(l1 *L1Cache, l2 *L2Cache, storage Storage, loader *CacheLoader, obs observability.Observability) *CacheManager {
	var pipeline *PipelineHelper
	if l2 != nil {
		pipeline = NewPipelineHelper(l2.client, obs)
	}

	return &CacheManager{
		l1:       l1,
		l2:       l2,
		storage:  storage,
		loader:   loader,
		pipeline: pipeline,
		sf:       NewEnhancedSingleflight(obs),
		obs:      obs,
	}
}

// Get retrieves a URL mapping with multi-tier fallback and enhanced singleflight
// Flow: L1 → L2 → DB, with backfilling on misses
func (cm *CacheManager) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Use enhanced singleflight to coalesce concurrent requests for the same key
	v, err := cm.sf.Do(ctx, shortCode, func() (interface{}, error) {
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
		cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L1"})
		cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "hit", "layer": "l1"})

		// Check if it's a null entry (cached 404)
		if IsNullEntry(mapping) {
			cm.obs.Metrics().IncrementCounter("shortener_cache_null_hits_total", map[string]string{"layer": "L1"})
			return nil, fmt.Errorf("mapping not found")
		}

		return mapping, nil
	}
	cm.obs.Metrics().IncrementCounter("shortener_cache_misses_total", map[string]string{"layer": "L1"})
	cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "miss", "layer": "l1"})

	// Try L2 cache
	if cm.l2 != nil {
		mapping, err := cm.l2.Get(ctx, shortCode)
		if err == nil && mapping != nil {
			cm.obs.Metrics().IncrementCounter("shortener_cache_hits_total", map[string]string{"layer": "L2"})
			cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "hit", "layer": "l2"})

			// Check if it's a null entry (cached 404)
			if IsNullEntry(mapping) {
				cm.obs.Metrics().IncrementCounter("shortener_cache_null_hits_total", map[string]string{"layer": "L2"})
				// Backfill L1 cache with null entry
				cm.l1.Set(mapping.ShortCode, "", mapping.CreatedAt)
				return nil, fmt.Errorf("mapping not found")
			}

			// Backfill L1 cache
			cm.l1.Set(mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
			return mapping, nil
		}
		cm.obs.Metrics().IncrementCounter("shortener_cache_misses_total", map[string]string{"layer": "L2"})
		cm.obs.Metrics().IncrementCounter("shortener_cache_operations_total", map[string]string{"operation": "miss", "layer": "l2"})
		// Continue to DB on L2 miss or error (graceful degradation)
	}

	// Fallback to database
	if cm.loader != nil && cm.l2 != nil {
		// Use CacheLoader with SETNX to prevent cache stampede
		// This ensures only one goroutine loads from DB when cache misses occur
		mapping, err := cm.loader.LoadWithLock(ctx, shortCode)
		if err != nil {
			// Check if it's a "not found" error
			if isNotFoundError(err) {
				// Cache null entry to prevent cache penetration
				nullEntry := CreateNullEntry(shortCode)
				cm.cacheNullEntry(ctx, nullEntry)
				cm.obs.Metrics().IncrementCounter("shortener_cache_null_entries_created_total", map[string]string{"source": "loader"})
				return nil, err
			}
			return nil, fmt.Errorf("failed to load with lock: %w", err)
		}

		// LoadWithLock already populated L2 cache and returns full mapping
		// Backfill L1 cache
		cm.l1.Set(mapping.ShortCode, mapping.LongURL, mapping.CreatedAt)
		return mapping, nil
	}

	// Fallback to direct database query (backward compatibility)
	storageMapping, err := cm.storage.Get(ctx, shortCode)
	if err != nil {
		// Check if it's a "not found" error
		if isNotFoundError(err) {
			// Cache null entry to prevent cache penetration
			nullEntry := CreateNullEntry(shortCode)
			cm.cacheNullEntry(ctx, nullEntry)
			cm.obs.Metrics().IncrementCounter("shortener_cache_null_entries_created_total", map[string]string{"source": "storage"})
			return nil, err
		}
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
		// Only backfill L2 if we didn't use LoadWithLock (which already populated it)
		if err := cm.l2.Set(ctx, mapping.ShortCode, mapping.LongURL, mapping.CreatedAt); err != nil {
			// Log error but don't fail the request (graceful degradation)
			cm.obs.Logger().Warn(ctx, "Failed to backfill L2 cache",
				"short_code", mapping.ShortCode,
				"error", err)
			cm.obs.Metrics().IncrementCounter("shortener_cache_backfill_errors_total",
				map[string]string{"layer": "L2"})
		}
	}

	return mapping, nil
}

// cacheNullEntry caches a null entry with short TTL to prevent cache penetration
func (cm *CacheManager) cacheNullEntry(ctx context.Context, nullEntry *URLMapping) {
	// Cache in L1 with short TTL (handled by Ristretto's cost-based eviction)
	cm.l1.Set(nullEntry.ShortCode, "", nullEntry.CreatedAt)

	// Cache in L2 with explicit short TTL (1 minute)
	if cm.l2 != nil {
		// Use a special method for null entries with short TTL
		if err := cm.l2.SetWithTTL(ctx, nullEntry.ShortCode, "", nullEntry.CreatedAt, 1*time.Minute); err != nil {
			cm.obs.Logger().Warn(ctx, "Failed to cache null entry in L2",
				"short_code", nullEntry.ShortCode,
				"error", err)
		}
	}
}

// isNotFoundError checks if an error indicates a "not found" condition
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "mapping not found" ||
		errStr == "not found" ||
		errStr == "record not found" ||
		errStr == "sql: no rows in result set"
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

// WarmCache preloads multiple URL mappings into cache using Pipeline
// This is useful for cache warming scenarios where multiple keys need to be loaded
func (cm *CacheManager) WarmCache(ctx context.Context, mappings []*URLMapping) error {
	if cm.l2 == nil || cm.pipeline == nil {
		return fmt.Errorf("L2 cache or pipeline not available")
	}

	if len(mappings) == 0 {
		return nil
	}

	// Prepare entries for batch set
	entries := make(map[string]string, len(mappings))
	for _, mapping := range mappings {
		key := fmt.Sprintf("url:%s", mapping.ShortCode)
		// Store as JSON for simplicity in batch operations
		value := fmt.Sprintf(`{"short_code":"%s","long_url":"%s","created_at":"%s"}`,
			mapping.ShortCode, mapping.LongURL, mapping.CreatedAt.Format(time.RFC3339))
		entries[key] = value
	}

	// Use Pipeline to batch set all entries
	// TTL: 7 days (jitter is handled by L2Cache.Set for individual operations)
	ttl := 7 * 24 * time.Hour
	if err := cm.pipeline.BatchSet(ctx, entries, ttl); err != nil {
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	cm.obs.Metrics().IncrementCounter("shortener_cache_warm_total", map[string]string{
		"count": fmt.Sprintf("%d", len(mappings)),
	})

	return nil
}

// BatchGet retrieves multiple URL mappings using Pipeline
func (cm *CacheManager) BatchGet(ctx context.Context, shortCodes []string) (map[string]*URLMapping, error) {
	if cm.l2 == nil {
		return nil, fmt.Errorf("L2 cache not available")
	}

	// Use L2Cache's BatchGet which already uses Pipeline
	return cm.l2.BatchGet(ctx, shortCodes)
}

// BatchDelete removes multiple URL mappings from all cache layers
func (cm *CacheManager) BatchDelete(ctx context.Context, shortCodes []string) error {
	// Delete from L1
	for _, shortCode := range shortCodes {
		cm.l1.Delete(shortCode)
	}

	// Delete from L2 using batch operation
	if cm.l2 != nil {
		if err := cm.l2.BatchDelete(ctx, shortCodes); err != nil {
			return fmt.Errorf("failed to batch delete from L2: %w", err)
		}
	}

	return nil
}
