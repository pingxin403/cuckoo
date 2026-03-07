package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
)

// CacheConsistency implements the delayed double delete strategy
// to maintain cache-database consistency during updates
//
// Strategy:
// 1. Delete cache immediately before DB update (prevents stale reads)
// 2. Update database
// 3. Delete cache again after 1 second delay (handles replication lag)
//
// This ensures eventual consistency even with read replicas and replication lag
type CacheConsistency struct {
	cacheManager *cache.CacheManager
	obs          observability.Observability
	delayTime    time.Duration // Default: 1 second
}

// NewCacheConsistency creates a new CacheConsistency instance
func NewCacheConsistency(cacheManager *cache.CacheManager, obs observability.Observability) *CacheConsistency {
	return &CacheConsistency{
		cacheManager: cacheManager,
		obs:          obs,
	}
}

// UpdateWithConsistency performs a database update with cache consistency guarantees
// This method implements the delayed double delete strategy:
// 1. Immediate cache delete (before DB update)
// 2. DB update operation
// 3. Delayed cache delete (after 1 second)
func (cc *CacheConsistency) UpdateWithConsistency(
	ctx context.Context,
	shortCode string,
	updateFunc func(context.Context) error,
) error {
	// Step 1: Delete cache immediately before DB update
	if err := cc.immediateDelete(ctx, shortCode); err != nil {
		// Log error but continue - cache delete failure shouldn't block DB update
		cc.obs.Logger().Error(ctx, "immediate cache delete failed",
			"short_code", shortCode,
			"error", err.Error(),
		)
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_errors_total", map[string]string{
			"type": "immediate",
		})
	} else {
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_total", map[string]string{
			"type": "immediate",
		})
	}

	// Step 2: Execute the database update
	if err := updateFunc(ctx); err != nil {
		return fmt.Errorf("database update failed: %w", err)
	}

	// Step 3: Schedule delayed cache delete
	go cc.delayedDelete(shortCode)

	return nil
}

// CreateWithConsistency performs a database create operation with cache consistency
// For create operations, we only need to invalidate cache after the operation
// (no need for immediate delete since the key doesn't exist yet)
func (cc *CacheConsistency) CreateWithConsistency(
	ctx context.Context,
	shortCode string,
	createFunc func(context.Context) error,
) error {
	// Execute the database create
	if err := createFunc(ctx); err != nil {
		return fmt.Errorf("database create failed: %w", err)
	}

	// For create operations, we don't need immediate delete
	// Just schedule delayed delete to ensure consistency
	go cc.delayedDelete(shortCode)

	return nil
}

// DeleteWithConsistency performs a database delete operation with cache consistency
// For delete operations, we use the same delayed double delete strategy
func (cc *CacheConsistency) DeleteWithConsistency(
	ctx context.Context,
	shortCode string,
	deleteFunc func(context.Context) error,
) error {
	// Step 1: Delete cache immediately before DB delete
	if err := cc.immediateDelete(ctx, shortCode); err != nil {
		cc.obs.Logger().Error(ctx, "immediate cache delete failed",
			"short_code", shortCode,
			"error", err.Error(),
		)
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_errors_total", map[string]string{
			"type": "immediate",
		})
	} else {
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_total", map[string]string{
			"type": "immediate",
		})
	}

	// Step 2: Execute the database delete
	if err := deleteFunc(ctx); err != nil {
		return fmt.Errorf("database delete failed: %w", err)
	}

	// Step 3: Schedule delayed cache delete
	go cc.delayedDelete(shortCode)

	return nil
}

// immediateDelete deletes cache entry immediately
func (cc *CacheConsistency) immediateDelete(ctx context.Context, shortCode string) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		cc.obs.Metrics().RecordHistogram("redis_cache_delete_duration_seconds", duration, map[string]string{
			"type": "immediate",
		})
	}()

	// Delete from cache manager (handles both L1 and L2)
	if err := cc.cacheManager.Delete(ctx, shortCode); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// delayedDelete deletes cache entry after a delay
// This runs in a separate goroutine to avoid blocking the main operation
func (cc *CacheConsistency) delayedDelete(shortCode string) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		cc.obs.Metrics().RecordHistogram("redis_cache_delete_duration_seconds", duration, map[string]string{
			"type": "delayed",
		})
	}()

	// Wait for the configured delay
	time.Sleep(cc.delayTime)

	// Create a new context with timeout for the delayed delete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete from cache manager
	if err := cc.cacheManager.Delete(ctx, shortCode); err != nil {
		// Log error but don't fail - this is a best-effort operation
		cc.obs.Logger().Error(ctx, "delayed cache delete failed",
			"short_code", shortCode,
			"error", err.Error(),
		)
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_errors_total", map[string]string{
			"type": "delayed",
		})
	} else {
		cc.obs.Metrics().IncrementCounter("redis_cache_delete_total", map[string]string{
			"type": "delayed",
		})
	}
}

// SetDelayTime allows customizing the delay time for testing or tuning
// Default is 1 second, but can be adjusted based on replication lag characteristics
func (cc *CacheConsistency) SetDelayTime(delay time.Duration) {
	cc.delayTime = delay
}

// GetDelayTime returns the current delay time
func (cc *CacheConsistency) GetDelayTime() time.Duration {
	return cc.delayTime
}

// WarmCacheAfterUpdate warms the cache after a database update
// This is optional and can be used to proactively populate cache
// after consistency operations complete
func (cc *CacheConsistency) WarmCacheAfterUpdate(ctx context.Context, shortCode string, mapping *storage.URLMapping) error {
	// Wait for delayed delete to complete
	time.Sleep(cc.delayTime + 100*time.Millisecond)

	// Warm the cache with fresh data
	if err := cc.cacheManager.Set(ctx, shortCode, mapping.LongURL, mapping.CreatedAt, mapping.ExpiresAt); err != nil {
		return fmt.Errorf("failed to warm cache: %w", err)
	}

	return nil
}
