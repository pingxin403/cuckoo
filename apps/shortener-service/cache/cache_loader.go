package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// CacheLoader implements SETNX-based cache loading to prevent cache stampede
// When multiple goroutines request the same cache miss simultaneously, only one
// will load from the database while others wait and retry reading from cache.
type CacheLoader struct {
	client     redis.UniversalClient
	storage    Storage
	l2Cache    *L2Cache // L2 cache for proper cache population
	obs        observability.Observability
	lockTTL    time.Duration // Default: 5 seconds
	maxRetries int           // Default: 3
	retryDelay time.Duration // Default: 50ms (base for exponential backoff)
}

// NewCacheLoader creates a new CacheLoader with default settings
func NewCacheLoader(client redis.UniversalClient, storage Storage, l2Cache *L2Cache, obs observability.Observability) *CacheLoader {
	return &CacheLoader{
		client:     client,
		storage:    storage,
		l2Cache:    l2Cache,
		obs:        obs,
		lockTTL:    5 * time.Second,
		maxRetries: 3,
		retryDelay: 50 * time.Millisecond,
	}
}

// LoadWithLock attempts to load data from cache or database using SETNX for coordination
// This method implements the following flow:
// 1. Try to acquire a lock using SETNX
// 2. If lock acquired: load from database, populate cache, release lock
// 3. If lock not acquired: wait briefly and retry reading from cache (exponential backoff)
func (cl *CacheLoader) LoadWithLock(ctx context.Context, shortCode string) (*URLMapping, error) {
	lockKey := "lock:" + shortCode

	// Try to acquire lock using SETNX
	acquired, err := cl.client.SetNX(ctx, lockKey, "1", cl.lockTTL).Result()
	if err != nil {
		cl.obs.Metrics().IncrementCounter("redis_setnx_errors_total", nil)
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if acquired {
		// Lock acquired - this goroutine loads data from database
		cl.obs.Metrics().IncrementCounter("redis_setnx_lock_acquired_total", nil)

		// Ensure lock is released even if we panic
		defer func() {
			// Release lock
			if delErr := cl.client.Del(ctx, lockKey).Err(); delErr != nil {
				cl.obs.Metrics().IncrementCounter("redis_setnx_lock_release_errors_total", nil)
			}
		}()

		// Load from database
		data, err := cl.storage.Get(ctx, shortCode)
		if err != nil {
			if err == storage.ErrNotFound {
				if cl.l2Cache != nil {
					if setErr := cl.l2Cache.SetEmpty(ctx, shortCode); setErr != nil {
						cl.obs.Logger().Warn(ctx, "Failed to cache empty value",
							"short_code", shortCode,
							"error", setErr)
						cl.obs.Metrics().IncrementCounter("redis_empty_cache_set_errors_total", nil)
					}
				}
				return nil, err
			}
			cl.obs.Metrics().IncrementCounter("redis_setnx_db_load_errors_total", nil)
			return nil, fmt.Errorf("failed to load from storage: %w", err)
		}

		// Set in L2 cache using proper L2Cache.Set method
		// This ensures the data is stored in the correct format (hash) with TTL jitter
		if cl.l2Cache != nil {
			if setErr := cl.l2Cache.Set(ctx, data.ShortCode, data.LongURL, data.CreatedAt); setErr != nil {
				cl.obs.Metrics().IncrementCounter("redis_setnx_cache_set_errors_total", nil)
				// Return data even if cache set fails (graceful degradation)
			}
		}

		// Return the full mapping
		return &URLMapping{
			ShortCode: data.ShortCode,
			LongURL:   data.LongURL,
			CreatedAt: data.CreatedAt,
		}, nil
	}

	// Lock not acquired - another goroutine is loading the data
	cl.obs.Metrics().IncrementCounter("redis_setnx_lock_contention_total", nil)

	for i := 0; i < cl.maxRetries; i++ {
		waitStart := time.Now()

		// Exponential backoff: 50ms, 100ms, 200ms
		backoffDelay := cl.retryDelay * time.Duration(1<<uint(i))
		time.Sleep(backoffDelay)

		waitDuration := time.Since(waitStart).Seconds()
		cl.obs.Metrics().RecordHistogram("redis_setnx_lock_wait_duration_seconds", waitDuration, nil)

		// Try to read from L2 cache
		if cl.l2Cache != nil {
			mapping, err := cl.l2Cache.Get(ctx, shortCode)
			if err == nil && mapping != nil {
				// Cache hit - data was loaded by another goroutine
				return mapping, nil
			}
		}

		// Cache miss - continue retrying
	}

	// All retries exhausted
	cl.obs.Metrics().IncrementCounter("redis_setnx_retry_exhausted_total", nil)
	return nil, fmt.Errorf("failed to load after %d retries", cl.maxRetries)
}
