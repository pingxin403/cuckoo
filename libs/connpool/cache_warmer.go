package connpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheWarmer manages cache warming and preloading strategies
type CacheWarmer struct {
	redis  *redis.Client
	config CacheWarmerConfig

	// Metrics
	mu                sync.RWMutex
	totalWarmed       int64
	totalFailed       int64
	totalInvalidated  int64
	lastWarmTime      time.Time
	warmDuration      time.Duration
	hitRateBeforeWarm float64
	hitRateAfterWarm  float64

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// CacheWarmerConfig holds cache warming configuration
type CacheWarmerConfig struct {
	// Enabled indicates if cache warming is enabled
	Enabled bool
	// WarmInterval is the interval between cache warming operations
	WarmInterval time.Duration
	// WarmTimeout is the timeout for warming operations
	WarmTimeout time.Duration
	// HotDataThreshold is the access count threshold for hot data
	HotDataThreshold int64
	// HotDataTTL is the TTL for hot data in cache
	HotDataTTL time.Duration
	// PreloadBatchSize is the batch size for preloading
	PreloadBatchSize int
	// CrossRegionSync enables cross-region cache synchronization
	CrossRegionSync bool
	// InvalidationStrategy defines the cache invalidation strategy
	InvalidationStrategy InvalidationStrategy
	// MaxCacheSize is the maximum cache size in bytes (0 = unlimited)
	MaxCacheSize int64
	// EvictionPolicy defines the eviction policy when cache is full
	EvictionPolicy EvictionPolicy
}

// InvalidationStrategy defines cache invalidation strategies
type InvalidationStrategy string

const (
	// InvalidationTTL uses TTL-based invalidation
	InvalidationTTL InvalidationStrategy = "ttl"
	// InvalidationLRU uses LRU-based invalidation
	InvalidationLRU InvalidationStrategy = "lru"
	// InvalidationWrite invalidates on write
	InvalidationWrite InvalidationStrategy = "write"
	// InvalidationHybrid uses a combination of strategies
	InvalidationHybrid InvalidationStrategy = "hybrid"
)

// EvictionPolicy defines cache eviction policies
type EvictionPolicy string

const (
	// EvictionLRU evicts least recently used items
	EvictionLRU EvictionPolicy = "lru"
	// EvictionLFU evicts least frequently used items
	EvictionLFU EvictionPolicy = "lfu"
	// EvictionRandom evicts random items
	EvictionRandom EvictionPolicy = "random"
	// EvictionTTL evicts items based on TTL
	EvictionTTL EvictionPolicy = "ttl"
)

// HotDataItem represents a hot data item to be cached
type HotDataItem struct {
	Key         string
	Value       interface{}
	AccessCount int64
	LastAccess  time.Time
	TTL         time.Duration
	Priority    int // Higher priority = more important
}

// CacheWarmingResult holds the result of a cache warming operation
type CacheWarmingResult struct {
	TotalItems    int
	SuccessCount  int
	FailureCount  int
	Duration      time.Duration
	HitRateBefore float64
	HitRateAfter  float64
	BytesWarmed   int64
	Errors        []error
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(redis *redis.Client, config CacheWarmerConfig) *CacheWarmer {
	ctx, cancel := context.WithCancel(context.Background())

	return &CacheWarmer{
		redis:  redis,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the cache warmer
func (cw *CacheWarmer) Start() error {
	if !cw.config.Enabled {
		return nil
	}

	cw.wg.Add(1)
	go func() {
		defer cw.wg.Done()
		cw.warmingLoop()
	}()

	return nil
}

// Stop stops the cache warmer
func (cw *CacheWarmer) Stop() error {
	cw.cancel()
	cw.wg.Wait()
	return nil
}

// warmingLoop runs the periodic cache warming
func (cw *CacheWarmer) warmingLoop() {
	ticker := time.NewTicker(cw.config.WarmInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cw.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(cw.ctx, cw.config.WarmTimeout)
			result := cw.WarmCache(ctx, nil)
			cancel()

			cw.mu.Lock()
			cw.lastWarmTime = time.Now()
			cw.warmDuration = result.Duration
			cw.hitRateBeforeWarm = result.HitRateBefore
			cw.hitRateAfterWarm = result.HitRateAfter
			cw.mu.Unlock()
		}
	}
}

// WarmCache warms the cache with hot data
func (cw *CacheWarmer) WarmCache(ctx context.Context, hotData []HotDataItem) CacheWarmingResult {
	startTime := time.Now()
	result := CacheWarmingResult{}

	// Get hit rate before warming
	result.HitRateBefore = cw.getHitRate(ctx)

	// If no hot data provided, identify hot data from access patterns
	if hotData == nil {
		var err error
		hotData, err = cw.identifyHotData(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to identify hot data: %w", err))
			return result
		}
	}

	result.TotalItems = len(hotData)

	// Preload hot data in batches
	for i := 0; i < len(hotData); i += cw.config.PreloadBatchSize {
		end := i + cw.config.PreloadBatchSize
		if end > len(hotData) {
			end = len(hotData)
		}

		batch := hotData[i:end]
		success, failed, bytesWarmed, errs := cw.preloadBatch(ctx, batch)
		result.SuccessCount += success
		result.FailureCount += failed
		result.BytesWarmed += bytesWarmed
		result.Errors = append(result.Errors, errs...)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, ctx.Err())
			result.Duration = time.Since(startTime)
			return result
		default:
		}
	}

	// Get hit rate after warming
	result.HitRateAfter = cw.getHitRate(ctx)

	result.Duration = time.Since(startTime)

	// Update metrics
	cw.mu.Lock()
	cw.totalWarmed += int64(result.SuccessCount)
	cw.totalFailed += int64(result.FailureCount)
	cw.mu.Unlock()

	return result
}

// identifyHotData identifies hot data based on access patterns
func (cw *CacheWarmer) identifyHotData(ctx context.Context) ([]HotDataItem, error) {
	// In a real implementation, this would analyze access logs or metrics
	// For now, we'll return an empty list
	// This should be implemented based on your specific use case

	// Example: Query access count from a separate tracking system
	// or use Redis OBJECT FREQ command for LFU eviction policy

	return []HotDataItem{}, nil
}

// preloadBatch preloads a batch of hot data items
func (cw *CacheWarmer) preloadBatch(ctx context.Context, batch []HotDataItem) (success, failed int, bytesWarmed int64, errs []error) {
	pipe := cw.redis.Pipeline()

	// Prepare pipeline commands
	cmds := make([]*redis.StatusCmd, len(batch))
	for i, item := range batch {
		ttl := item.TTL
		if ttl == 0 {
			ttl = cw.config.HotDataTTL
		}
		cmds[i] = pipe.Set(ctx, item.Key, item.Value, ttl)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		// If pipeline fails, try individual sets
		for _, item := range batch {
			ttl := item.TTL
			if ttl == 0 {
				ttl = cw.config.HotDataTTL
			}
			err := cw.redis.Set(ctx, item.Key, item.Value, ttl).Err()
			if err != nil {
				failed++
				errs = append(errs, fmt.Errorf("failed to set key %s: %w", item.Key, err))
			} else {
				success++
				// Estimate bytes (key + value)
				bytesWarmed += int64(len(item.Key) + estimateValueSize(item.Value))
			}
		}
		return
	}

	// Check individual command results
	for i, cmd := range cmds {
		if cmd.Err() != nil {
			failed++
			errs = append(errs, fmt.Errorf("failed to set key %s: %w", batch[i].Key, cmd.Err()))
		} else {
			success++
			bytesWarmed += int64(len(batch[i].Key) + estimateValueSize(batch[i].Value))
		}
	}

	return
}

// InvalidateCache invalidates cache entries based on the configured strategy
func (cw *CacheWarmer) InvalidateCache(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	switch cw.config.InvalidationStrategy {
	case InvalidationTTL:
		// TTL-based invalidation is automatic
		return nil
	case InvalidationWrite:
		// Delete keys immediately
		return cw.invalidateKeys(ctx, keys)
	case InvalidationLRU, InvalidationHybrid:
		// Mark keys for invalidation but don't delete immediately
		return cw.markForInvalidation(ctx, keys)
	default:
		return fmt.Errorf("unknown invalidation strategy: %s", cw.config.InvalidationStrategy)
	}
}

// invalidateKeys deletes keys from cache
func (cw *CacheWarmer) invalidateKeys(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Use pipeline for batch deletion
	pipe := cw.redis.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to invalidate keys: %w", err)
	}

	cw.mu.Lock()
	cw.totalInvalidated += int64(len(keys))
	cw.mu.Unlock()

	return nil
}

// markForInvalidation marks keys for future invalidation
func (cw *CacheWarmer) markForInvalidation(ctx context.Context, keys []string) error {
	// Set a short TTL on keys to mark them for invalidation
	pipe := cw.redis.Pipeline()
	for _, key := range keys {
		pipe.Expire(ctx, key, 1*time.Second)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark keys for invalidation: %w", err)
	}

	cw.mu.Lock()
	cw.totalInvalidated += int64(len(keys))
	cw.mu.Unlock()

	return nil
}

// SyncCrossRegion synchronizes cache entries across regions
func (cw *CacheWarmer) SyncCrossRegion(ctx context.Context, remoteRedis *redis.Client, keys []string) error {
	if !cw.config.CrossRegionSync {
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	// Get values from local cache
	pipe := cw.redis.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
		ttlCmds[i] = pipe.TTL(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get local cache values: %w", err)
	}

	// Set values in remote cache
	remotePipe := remoteRedis.Pipeline()
	for i, cmd := range cmds {
		if cmd.Err() == nil {
			val := cmd.Val()
			ttl := ttlCmds[i].Val()
			if ttl < 0 {
				ttl = cw.config.HotDataTTL
			}
			remotePipe.Set(ctx, keys[i], val, ttl)
		}
	}

	_, err = remotePipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync to remote cache: %w", err)
	}

	return nil
}

// getHitRate calculates the current cache hit rate
func (cw *CacheWarmer) getHitRate(ctx context.Context) float64 {
	stats := cw.redis.PoolStats()
	totalRequests := stats.Hits + stats.Misses
	if totalRequests == 0 {
		return 0.0
	}
	return float64(stats.Hits) / float64(totalRequests)
}

// GetMetrics returns cache warmer metrics
func (cw *CacheWarmer) GetMetrics() CacheWarmerMetrics {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	return CacheWarmerMetrics{
		TotalWarmed:        cw.totalWarmed,
		TotalFailed:        cw.totalFailed,
		TotalInvalidated:   cw.totalInvalidated,
		LastWarmTime:       cw.lastWarmTime,
		WarmDuration:       cw.warmDuration,
		HitRateBefore:      cw.hitRateBeforeWarm,
		HitRateAfter:       cw.hitRateAfterWarm,
		HitRateImprovement: cw.hitRateAfterWarm - cw.hitRateBeforeWarm,
	}
}

// CacheWarmerMetrics holds cache warmer metrics
type CacheWarmerMetrics struct {
	TotalWarmed        int64
	TotalFailed        int64
	TotalInvalidated   int64
	LastWarmTime       time.Time
	WarmDuration       time.Duration
	HitRateBefore      float64
	HitRateAfter       float64
	HitRateImprovement float64
}

// estimateValueSize estimates the size of a value in bytes
func estimateValueSize(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case int, int8, int16, int32, int64:
		return 8
	case uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case bool:
		return 1
	default:
		// Rough estimate for complex types
		return 100
	}
}

// DefaultCacheWarmerConfig returns a default cache warmer configuration
func DefaultCacheWarmerConfig() CacheWarmerConfig {
	return CacheWarmerConfig{
		Enabled:              true,
		WarmInterval:         5 * time.Minute,
		WarmTimeout:          30 * time.Second,
		HotDataThreshold:     100, // Items accessed 100+ times
		HotDataTTL:           1 * time.Hour,
		PreloadBatchSize:     100,
		CrossRegionSync:      true,
		InvalidationStrategy: InvalidationHybrid,
		MaxCacheSize:         1024 * 1024 * 1024, // 1GB
		EvictionPolicy:       EvictionLRU,
	}
}
