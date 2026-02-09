package connpool

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolManager_CacheWarmer_Integration(t *testing.T) {
	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get Redis pool
	redisClient, err := pm.GetRedis("test-redis", &redis.Options{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)
	assert.NotNil(t, redisClient)

	// Create cache warmer
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.Enabled = false // Don't start automatic warming
	warmer, err := pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
	require.NoError(t, err)
	assert.NotNil(t, warmer)

	// Warm cache with hot data
	ctx := context.Background()
	hotData := []HotDataItem{
		{Key: "user:1", Value: "Alice", AccessCount: 150, TTL: 1 * time.Hour},
		{Key: "user:2", Value: "Bob", AccessCount: 200, TTL: 1 * time.Hour},
		{Key: "product:1", Value: "Laptop", AccessCount: 180, TTL: 30 * time.Minute},
	}

	result := warmer.WarmCache(ctx, hotData)
	assert.Equal(t, 3, result.TotalItems)
	assert.Equal(t, 3, result.SuccessCount)
	assert.Equal(t, 0, result.FailureCount)

	// Verify data is in cache
	val, err := redisClient.Get(ctx, "user:1").Result()
	assert.NoError(t, err)
	assert.Equal(t, "Alice", val)

	// Get metrics
	metrics := warmer.GetMetrics()
	assert.Equal(t, int64(3), metrics.TotalWarmed)
}

func TestPoolManager_CacheWarmer_AutomaticWarming(t *testing.T) {
	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get Redis pool
	_, err = pm.GetRedis("test-redis", &redis.Options{
		Addr: mr.Addr(),
	})
	require.NoError(t, err)

	// Create cache warmer with automatic warming
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.Enabled = true
	warmerConfig.WarmInterval = 100 * time.Millisecond
	warmer, err := pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
	require.NoError(t, err)

	// Start pool manager (which starts cache warmer)
	err = pm.Start()
	require.NoError(t, err)

	// Wait for at least one warming cycle
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := warmer.GetMetrics()
	assert.False(t, metrics.LastWarmTime.IsZero())
}

func TestPoolManager_CacheWarmer_CrossRegionSync(t *testing.T) {
	// Setup local Redis
	localMr, err := miniredis.Run()
	require.NoError(t, err)
	defer localMr.Close()

	// Setup remote Redis
	remoteMr, err := miniredis.Run()
	require.NoError(t, err)
	defer remoteMr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get local Redis pool
	localClient, err := pm.GetRedis("local-redis", &redis.Options{
		Addr: localMr.Addr(),
	})
	require.NoError(t, err)

	// Get remote Redis pool
	remoteClient, err := pm.GetRedis("remote-redis", &redis.Options{
		Addr: remoteMr.Addr(),
	})
	require.NoError(t, err)

	// Create cache warmer for local Redis
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.CrossRegionSync = true
	warmer, err := pm.GetOrCreateCacheWarmer("local-redis", warmerConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// Set data in local cache
	keys := []string{"user:1", "user:2"}
	values := []string{"Alice", "Bob"}
	for i, key := range keys {
		err := localClient.Set(ctx, key, values[i], 1*time.Hour).Err()
		require.NoError(t, err)
	}

	// Sync to remote cache
	err = warmer.SyncCrossRegion(ctx, remoteClient, keys)
	assert.NoError(t, err)

	// Verify data is in remote cache
	for i, key := range keys {
		val, err := remoteClient.Get(ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, values[i], val)
	}
}

func TestPoolManager_CacheWarmer_InvalidationStrategies(t *testing.T) {
	strategies := []InvalidationStrategy{
		InvalidationTTL,
		InvalidationWrite,
		InvalidationLRU,
		InvalidationHybrid,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			// Setup Redis
			mr, err := miniredis.Run()
			require.NoError(t, err)
			defer mr.Close()

			// Create pool manager
			config := DefaultPoolConfig()
			pm := NewPoolManager(config)
			defer pm.Stop()

			// Get Redis pool
			redisClient, err := pm.GetRedis("test-redis", &redis.Options{
				Addr: mr.Addr(),
			})
			require.NoError(t, err)

			// Create cache warmer with specific strategy
			warmerConfig := DefaultCacheWarmerConfig()
			warmerConfig.InvalidationStrategy = strategy
			warmer, err := pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
			require.NoError(t, err)

			ctx := context.Background()

			// Set some keys
			keys := []string{"key1", "key2"}
			for _, key := range keys {
				err := redisClient.Set(ctx, key, "value", 1*time.Hour).Err()
				require.NoError(t, err)
			}

			// Invalidate keys
			err = warmer.InvalidateCache(ctx, keys)
			assert.NoError(t, err)

			// Verify behavior based on strategy
			switch strategy {
			case InvalidationTTL:
				// Keys should still exist (TTL-based is automatic)
				for _, key := range keys {
					exists, err := redisClient.Exists(ctx, key).Result()
					require.NoError(t, err)
					assert.Equal(t, int64(1), exists)
				}
			case InvalidationWrite:
				// Keys should be deleted
				for _, key := range keys {
					exists, err := redisClient.Exists(ctx, key).Result()
					require.NoError(t, err)
					assert.Equal(t, int64(0), exists)
				}
			case InvalidationLRU, InvalidationHybrid:
				// Keys should exist but with short TTL
				for _, key := range keys {
					exists, err := redisClient.Exists(ctx, key).Result()
					require.NoError(t, err)
					assert.Equal(t, int64(1), exists)

					ttl, err := redisClient.TTL(ctx, key).Result()
					require.NoError(t, err)
					assert.LessOrEqual(t, ttl, 2*time.Second)
				}
			}
		})
	}
}

func TestPoolManager_GetAllCacheWarmerMetrics(t *testing.T) {
	// Setup Redis instances
	mr1, err := miniredis.Run()
	require.NoError(t, err)
	defer mr1.Close()

	mr2, err := miniredis.Run()
	require.NoError(t, err)
	defer mr2.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get Redis pools
	_, err = pm.GetRedis("redis-1", &redis.Options{Addr: mr1.Addr()})
	require.NoError(t, err)

	_, err = pm.GetRedis("redis-2", &redis.Options{Addr: mr2.Addr()})
	require.NoError(t, err)

	// Create cache warmers
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.Enabled = false

	_, err = pm.GetOrCreateCacheWarmer("redis-1", warmerConfig)
	require.NoError(t, err)

	_, err = pm.GetOrCreateCacheWarmer("redis-2", warmerConfig)
	require.NoError(t, err)

	// Get all metrics
	allMetrics := pm.GetAllCacheWarmerMetrics()
	assert.Len(t, allMetrics, 2)
	assert.Contains(t, allMetrics, "redis-1")
	assert.Contains(t, allMetrics, "redis-2")
}

func TestPoolManager_CacheWarmer_GetExisting(t *testing.T) {
	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get Redis pool
	_, err = pm.GetRedis("test-redis", &redis.Options{Addr: mr.Addr()})
	require.NoError(t, err)

	// Create cache warmer
	warmerConfig := DefaultCacheWarmerConfig()
	warmer1, err := pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
	require.NoError(t, err)
	assert.NotNil(t, warmer1)

	// Get existing cache warmer
	warmer2, exists := pm.GetCacheWarmer("test-redis")
	assert.True(t, exists)
	assert.Equal(t, warmer1, warmer2)

	// Try to get non-existent cache warmer
	_, exists = pm.GetCacheWarmer("non-existent")
	assert.False(t, exists)
}

func TestPoolManager_CacheWarmer_ErrorHandling(t *testing.T) {
	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Try to create cache warmer for non-existent Redis pool
	warmerConfig := DefaultCacheWarmerConfig()
	_, err := pm.GetOrCreateCacheWarmer("non-existent", warmerConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPoolManager_CacheWarmer_Lifecycle(t *testing.T) {
	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)

	// Get Redis pool
	_, err = pm.GetRedis("test-redis", &redis.Options{Addr: mr.Addr()})
	require.NoError(t, err)

	// Create cache warmer
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.Enabled = true
	warmerConfig.WarmInterval = 100 * time.Millisecond
	_, err = pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
	require.NoError(t, err)

	// Start pool manager
	err = pm.Start()
	require.NoError(t, err)

	// Wait for warming cycles
	time.Sleep(250 * time.Millisecond)

	// Stop pool manager (should stop cache warmer)
	err = pm.Stop()
	assert.NoError(t, err)

	// Verify cache warmer is stopped
	warmer, exists := pm.GetCacheWarmer("test-redis")
	assert.False(t, exists)
	assert.Nil(t, warmer)
}

// Benchmark tests
func BenchmarkPoolManager_CacheWarmer_WarmCache(b *testing.B) {
	// Setup Redis
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	// Create pool manager
	config := DefaultPoolConfig()
	pm := NewPoolManager(config)
	defer pm.Stop()

	// Get Redis pool
	_, err = pm.GetRedis("test-redis", &redis.Options{Addr: mr.Addr()})
	if err != nil {
		b.Fatal(err)
	}

	// Create cache warmer
	warmerConfig := DefaultCacheWarmerConfig()
	warmerConfig.Enabled = false
	warmer, err := pm.GetOrCreateCacheWarmer("test-redis", warmerConfig)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	// Prepare hot data
	hotData := make([]HotDataItem, 100)
	for i := 0; i < 100; i++ {
		hotData[i] = HotDataItem{
			Key:         string(rune(i)),
			Value:       i,
			AccessCount: 100,
			TTL:         1 * time.Hour,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warmer.WarmCache(ctx, hotData)
	}
}
