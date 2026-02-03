package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL2Cache_SetEmpty tests the empty cache functionality
func TestL2Cache_SetEmpty(t *testing.T) {
	// Setup mini Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2Cache, err := NewL2Cache(config, obs)
	require.NoError(t, err)
	defer l2Cache.Close()

	ctx := context.Background()
	shortCode := "empty001"

	// Set empty value
	err = l2Cache.SetEmpty(ctx, shortCode)
	require.NoError(t, err)

	// Verify empty value is cached
	key := "url:" + shortCode
	result, err := l2Cache.client.HGetAll(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, shortCode, result["short_code"])
	assert.Equal(t, "__EMPTY__", result["long_url"])

	// Verify TTL is set (should be 5 minutes)
	ttl, err := l2Cache.client.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 4*time.Minute)
	assert.LessOrEqual(t, ttl, 5*time.Minute)
}

// TestL2Cache_GetEmpty tests retrieving empty cached values
func TestL2Cache_GetEmpty(t *testing.T) {
	// Setup mini Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2Cache, err := NewL2Cache(config, obs)
	require.NoError(t, err)
	defer l2Cache.Close()

	ctx := context.Background()
	shortCode := "empty002"

	// Set empty value
	err = l2Cache.SetEmpty(ctx, shortCode)
	require.NoError(t, err)

	// Get should return nil for empty cached values
	mapping, err := l2Cache.Get(ctx, shortCode)
	require.NoError(t, err)
	assert.Nil(t, mapping, "Empty cache should return nil mapping")
}

// TestL2Cache_EmptyVsNormal tests that empty cache doesn't interfere with normal cache
func TestL2Cache_EmptyVsNormal(t *testing.T) {
	// Setup mini Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2Cache, err := NewL2Cache(config, obs)
	require.NoError(t, err)
	defer l2Cache.Close()

	ctx := context.Background()

	// Set empty value for one short code
	emptyCode := "empty003"
	err = l2Cache.SetEmpty(ctx, emptyCode)
	require.NoError(t, err)

	// Set normal value for another short code
	normalCode := "normal001"
	err = l2Cache.Set(ctx, normalCode, "https://example.com", time.Now())
	require.NoError(t, err)

	// Get empty - should return nil
	emptyMapping, err := l2Cache.Get(ctx, emptyCode)
	require.NoError(t, err)
	assert.Nil(t, emptyMapping)

	// Get normal - should return the mapping
	normalMapping, err := l2Cache.Get(ctx, normalCode)
	require.NoError(t, err)
	require.NotNil(t, normalMapping)
	assert.Equal(t, normalCode, normalMapping.ShortCode)
	assert.Equal(t, "https://example.com", normalMapping.LongURL)
}

// TestL2Cache_EmptyExpiration tests that empty cache expires after TTL
func TestL2Cache_EmptyExpiration(t *testing.T) {
	// Setup mini Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2Cache, err := NewL2Cache(config, obs)
	require.NoError(t, err)
	defer l2Cache.Close()

	ctx := context.Background()
	shortCode := "empty004"

	// Set empty value
	err = l2Cache.SetEmpty(ctx, shortCode)
	require.NoError(t, err)

	// Verify it exists
	mapping, err := l2Cache.Get(ctx, shortCode)
	require.NoError(t, err)
	assert.Nil(t, mapping)

	// Fast forward time in miniredis (5 minutes + 1 second)
	mr.FastForward(5*time.Minute + time.Second)

	// Verify it's expired
	key := "url:" + shortCode
	exists, err := l2Cache.client.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "Empty cache should expire after TTL")
}

// TestL2Cache_EmptyOverwrite tests overwriting empty cache with real data
func TestL2Cache_EmptyOverwrite(t *testing.T) {
	// Setup mini Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2Cache, err := NewL2Cache(config, obs)
	require.NoError(t, err)
	defer l2Cache.Close()

	ctx := context.Background()
	shortCode := "test005"

	// First, set empty value
	err = l2Cache.SetEmpty(ctx, shortCode)
	require.NoError(t, err)

	// Verify empty
	mapping, err := l2Cache.Get(ctx, shortCode)
	require.NoError(t, err)
	assert.Nil(t, mapping)

	// Now set real data (simulating URL creation)
	err = l2Cache.Set(ctx, shortCode, "https://example.com", time.Now())
	require.NoError(t, err)

	// Verify real data is returned
	mapping, err = l2Cache.Get(ctx, shortCode)
	require.NoError(t, err)
	require.NotNil(t, mapping)
	assert.Equal(t, shortCode, mapping.ShortCode)
	assert.Equal(t, "https://example.com", mapping.LongURL)
}
