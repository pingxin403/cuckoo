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

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestNewCacheWarmer(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	warmer := NewCacheWarmer(client, config)

	assert.NotNil(t, warmer)
	assert.Equal(t, client, warmer.redis)
	assert.Equal(t, config, warmer.config)
}

func TestCacheWarmer_WarmCache(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.PreloadBatchSize = 10
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Prepare hot data
	hotData := []HotDataItem{
		{Key: "user:1", Value: "Alice", AccessCount: 150, TTL: 1 * time.Hour},
		{Key: "user:2", Value: "Bob", AccessCount: 200, TTL: 1 * time.Hour},
		{Key: "user:3", Value: "Charlie", AccessCount: 120, TTL: 1 * time.Hour},
		{Key: "product:1", Value: "Laptop", AccessCount: 180, TTL: 30 * time.Minute},
		{Key: "product:2", Value: "Phone", AccessCount: 250, TTL: 30 * time.Minute},
	}

	// Warm cache
	result := warmer.WarmCache(ctx, hotData)

	// Verify results
	assert.Equal(t, 5, result.TotalItems)
	assert.Equal(t, 5, result.SuccessCount)
	assert.Equal(t, 0, result.FailureCount)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Empty(t, result.Errors)

	// Verify data is in cache
	val, err := client.Get(ctx, "user:1").Result()
	assert.NoError(t, err)
	assert.Equal(t, "Alice", val)

	val, err = client.Get(ctx, "product:2").Result()
	assert.NoError(t, err)
	assert.Equal(t, "Phone", val)

	// Verify metrics
	metrics := warmer.GetMetrics()
	assert.Equal(t, int64(5), metrics.TotalWarmed)
	assert.Equal(t, int64(0), metrics.TotalFailed)
}

func TestCacheWarmer_WarmCache_EmptyData(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Warm cache with empty data
	result := warmer.WarmCache(ctx, []HotDataItem{})

	assert.Equal(t, 0, result.TotalItems)
	assert.Equal(t, 0, result.SuccessCount)
	assert.Equal(t, 0, result.FailureCount)
}

func TestCacheWarmer_WarmCache_Batching(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.PreloadBatchSize = 2 // Small batch size to test batching
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Prepare hot data (more than batch size)
	hotData := make([]HotDataItem, 5)
	for i := 0; i < 5; i++ {
		hotData[i] = HotDataItem{
			Key:         string(rune('A' + i)),
			Value:       i,
			AccessCount: 100,
			TTL:         1 * time.Hour,
		}
	}

	// Warm cache
	result := warmer.WarmCache(ctx, hotData)

	// Verify all items were processed
	assert.Equal(t, 5, result.TotalItems)
	assert.Equal(t, 5, result.SuccessCount)
	assert.Equal(t, 0, result.FailureCount)

	// Verify all data is in cache
	for i := 0; i < 5; i++ {
		val, err := client.Get(ctx, string(rune('A'+i))).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, val)
	}
}

func TestCacheWarmer_WarmCache_WithTimeout(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	warmer := NewCacheWarmer(client, config)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Prepare large hot data
	hotData := make([]HotDataItem, 1000)
	for i := 0; i < 1000; i++ {
		hotData[i] = HotDataItem{
			Key:         string(rune(i)),
			Value:       i,
			AccessCount: 100,
			TTL:         1 * time.Hour,
		}
	}

	// Warm cache (should timeout)
	result := warmer.WarmCache(ctx, hotData)

	// Should have context error
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[len(result.Errors)-1].Error(), "context")
}

func TestCacheWarmer_InvalidateCache_Write(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.InvalidationStrategy = InvalidationWrite
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Set some keys
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		err := client.Set(ctx, key, "value", 1*time.Hour).Err()
		require.NoError(t, err)
	}

	// Verify keys exist
	for _, key := range keys {
		exists, err := client.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)
	}

	// Invalidate keys
	err := warmer.InvalidateCache(ctx, keys)
	assert.NoError(t, err)

	// Verify keys are deleted
	for _, key := range keys {
		exists, err := client.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(0), exists)
	}

	// Verify metrics
	metrics := warmer.GetMetrics()
	assert.Equal(t, int64(3), metrics.TotalInvalidated)
}

func TestCacheWarmer_InvalidateCache_TTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.InvalidationStrategy = InvalidationTTL
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Set some keys
	keys := []string{"key1", "key2"}
	for _, key := range keys {
		err := client.Set(ctx, key, "value", 1*time.Hour).Err()
		require.NoError(t, err)
	}

	// Invalidate keys (should be no-op for TTL strategy)
	err := warmer.InvalidateCache(ctx, keys)
	assert.NoError(t, err)

	// Keys should still exist (TTL-based invalidation is automatic)
	for _, key := range keys {
		exists, err := client.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)
	}
}

func TestCacheWarmer_InvalidateCache_Hybrid(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.InvalidationStrategy = InvalidationHybrid
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Set some keys
	keys := []string{"key1", "key2"}
	for _, key := range keys {
		err := client.Set(ctx, key, "value", 1*time.Hour).Err()
		require.NoError(t, err)
	}

	// Invalidate keys (should mark for invalidation)
	err := warmer.InvalidateCache(ctx, keys)
	assert.NoError(t, err)

	// Keys should still exist but with short TTL
	for _, key := range keys {
		exists, err := client.Exists(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), exists)

		ttl, err := client.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.LessOrEqual(t, ttl, 2*time.Second)
	}

	// Verify metrics
	metrics := warmer.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalInvalidated)
}

func TestCacheWarmer_SyncCrossRegion(t *testing.T) {
	// Setup local Redis
	localClient, localMr := setupTestRedis(t)
	defer localMr.Close()
	defer localClient.Close()

	// Setup remote Redis
	remoteClient, remoteMr := setupTestRedis(t)
	defer remoteMr.Close()
	defer remoteClient.Close()

	config := DefaultCacheWarmerConfig()
	config.CrossRegionSync = true
	warmer := NewCacheWarmer(localClient, config)

	ctx := context.Background()

	// Set data in local cache
	keys := []string{"user:1", "user:2", "product:1"}
	values := []string{"Alice", "Bob", "Laptop"}
	for i, key := range keys {
		err := localClient.Set(ctx, key, values[i], 1*time.Hour).Err()
		require.NoError(t, err)
	}

	// Sync to remote cache
	err := warmer.SyncCrossRegion(ctx, remoteClient, keys)
	assert.NoError(t, err)

	// Verify data is in remote cache
	for i, key := range keys {
		val, err := remoteClient.Get(ctx, key).Result()
		assert.NoError(t, err)
		assert.Equal(t, values[i], val)
	}
}

func TestCacheWarmer_SyncCrossRegion_Disabled(t *testing.T) {
	localClient, localMr := setupTestRedis(t)
	defer localMr.Close()
	defer localClient.Close()

	remoteClient, remoteMr := setupTestRedis(t)
	defer remoteMr.Close()
	defer remoteClient.Close()

	config := DefaultCacheWarmerConfig()
	config.CrossRegionSync = false // Disabled
	warmer := NewCacheWarmer(localClient, config)

	ctx := context.Background()

	// Set data in local cache
	keys := []string{"key1"}
	err := localClient.Set(ctx, keys[0], "value", 1*time.Hour).Err()
	require.NoError(t, err)

	// Sync should be no-op
	err = warmer.SyncCrossRegion(ctx, remoteClient, keys)
	assert.NoError(t, err)

	// Remote cache should be empty
	exists, err := remoteClient.Exists(ctx, keys[0]).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestCacheWarmer_StartStop(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.Enabled = true
	config.WarmInterval = 100 * time.Millisecond
	warmer := NewCacheWarmer(client, config)

	// Start warmer
	err := warmer.Start()
	assert.NoError(t, err)

	// Wait for at least one warming cycle
	time.Sleep(200 * time.Millisecond)

	// Stop warmer
	err = warmer.Stop()
	assert.NoError(t, err)

	// Verify metrics were updated
	metrics := warmer.GetMetrics()
	assert.False(t, metrics.LastWarmTime.IsZero())
}

func TestCacheWarmer_StartStop_Disabled(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.Enabled = false
	warmer := NewCacheWarmer(client, config)

	// Start should be no-op
	err := warmer.Start()
	assert.NoError(t, err)

	// Stop should be no-op
	err = warmer.Stop()
	assert.NoError(t, err)
}

func TestCacheWarmer_GetMetrics(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Initial metrics
	metrics := warmer.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalWarmed)
	assert.Equal(t, int64(0), metrics.TotalFailed)
	assert.Equal(t, int64(0), metrics.TotalInvalidated)

	// Warm cache
	hotData := []HotDataItem{
		{Key: "key1", Value: "value1", TTL: 1 * time.Hour},
		{Key: "key2", Value: "value2", TTL: 1 * time.Hour},
	}
	warmer.WarmCache(ctx, hotData)

	// Check metrics after warming
	metrics = warmer.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalWarmed)
	assert.Equal(t, int64(0), metrics.TotalFailed)

	// Invalidate cache
	warmer.InvalidateCache(ctx, []string{"key1"})

	// Check metrics after invalidation
	metrics = warmer.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalInvalidated)
}

func TestEstimateValueSize(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected int
	}{
		{"string", "hello", 5},
		{"bytes", []byte("world"), 5},
		{"int", 42, 8},
		{"int64", int64(100), 8},
		{"float64", 3.14, 8},
		{"bool", true, 1},
		{"complex", map[string]string{"key": "value"}, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := estimateValueSize(tt.value)
			assert.Equal(t, tt.expected, size)
		})
	}
}

func TestDefaultCacheWarmerConfig(t *testing.T) {
	config := DefaultCacheWarmerConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 5*time.Minute, config.WarmInterval)
	assert.Equal(t, 30*time.Second, config.WarmTimeout)
	assert.Equal(t, int64(100), config.HotDataThreshold)
	assert.Equal(t, 1*time.Hour, config.HotDataTTL)
	assert.Equal(t, 100, config.PreloadBatchSize)
	assert.True(t, config.CrossRegionSync)
	assert.Equal(t, InvalidationHybrid, config.InvalidationStrategy)
	assert.Equal(t, int64(1024*1024*1024), config.MaxCacheSize)
	assert.Equal(t, EvictionLRU, config.EvictionPolicy)
}

// Benchmark tests
func BenchmarkCacheWarmer_WarmCache(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	warmer := NewCacheWarmer(client, config)

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

func BenchmarkCacheWarmer_InvalidateCache(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()
	defer client.Close()

	config := DefaultCacheWarmerConfig()
	config.InvalidationStrategy = InvalidationWrite
	warmer := NewCacheWarmer(client, config)

	ctx := context.Background()

	// Prepare keys
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = string(rune(i))
		client.Set(ctx, keys[i], i, 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		warmer.InvalidateCache(ctx, keys)
		// Re-populate for next iteration
		for j, key := range keys {
			client.Set(ctx, key, j, 1*time.Hour)
		}
	}
}
