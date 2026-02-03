package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLuaTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func createTestObservabilityForLua() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	return obs
}

func TestNewLuaScriptManager(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.client)
	assert.NotNil(t, manager.obs)
	assert.NotNil(t, manager.scripts)
	assert.NotNil(t, manager.scriptSHAs)

	// Verify scripts are registered
	scripts := manager.ListScripts()
	assert.Contains(t, scripts, "cache_load")
	assert.Contains(t, scripts, "increment_expire")
	assert.Contains(t, scripts, "set_ttl_jitter")
}

func TestLuaScriptManager_PreloadScripts(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()
	err := manager.PreloadScripts(ctx)
	require.NoError(t, err)

	// Verify scripts are loaded
	for _, name := range manager.ListScripts() {
		sha, ok := manager.GetScriptSHA(name)
		assert.True(t, ok)
		assert.NotEmpty(t, sha)
	}
}

func TestLuaScriptManager_ExecuteCacheLoad_Hit(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()

	// Set up cache data
	cacheKey := "url:test123"
	mr.HSet(cacheKey, "short_code", "test123")
	mr.HSet(cacheKey, "long_url", "https://example.com")
	mr.HSet(cacheKey, "created_at", "2026-02-03T00:00:00Z")

	// Execute cache load script
	status, cacheValue, err := manager.ExecuteCacheLoad(ctx, cacheKey, "lock:test123", 5)
	require.NoError(t, err)
	assert.Equal(t, "HIT", status)
	assert.NotNil(t, cacheValue)
	assert.Equal(t, "test123", cacheValue["short_code"])
	assert.Equal(t, "https://example.com", cacheValue["long_url"])
}

func TestLuaScriptManager_ExecuteCacheLoad_Locked(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()

	// Cache key doesn't exist
	cacheKey := "url:test456"
	lockKey := "lock:test456"

	// Execute cache load script - should acquire lock
	status, cacheValue, err := manager.ExecuteCacheLoad(ctx, cacheKey, lockKey, 5)
	require.NoError(t, err)
	assert.Equal(t, "LOCKED", status)
	assert.Nil(t, cacheValue)

	// Verify lock was set
	exists := mr.Exists(lockKey)
	assert.True(t, exists)

	// Verify lock has TTL
	ttl := mr.TTL(lockKey)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 5*time.Second)
}

func TestLuaScriptManager_ExecuteCacheLoad_Contention(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()

	// Cache key doesn't exist
	cacheKey := "url:test789"
	lockKey := "lock:test789"

	// Set lock manually (simulate another process holding the lock)
	mr.Set(lockKey, "1")
	mr.SetTTL(lockKey, 5*time.Second)

	// Execute cache load script - should detect contention
	status, cacheValue, err := manager.ExecuteCacheLoad(ctx, cacheKey, lockKey, 5)
	require.NoError(t, err)
	assert.Equal(t, "CONTENTION", status)
	assert.Nil(t, cacheValue)
}

func TestLuaScriptManager_ExecuteIncrementAndExpire(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()
	key := "counter:test"

	// First increment
	value, err := manager.ExecuteIncrementAndExpire(ctx, key, 1, 60)
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)

	// Verify key exists and has TTL
	exists := mr.Exists(key)
	assert.True(t, exists)
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 60*time.Second)

	// Second increment
	value, err = manager.ExecuteIncrementAndExpire(ctx, key, 5, 60)
	require.NoError(t, err)
	assert.Equal(t, int64(6), value)

	// Third increment with different TTL
	value, err = manager.ExecuteIncrementAndExpire(ctx, key, 10, 120)
	require.NoError(t, err)
	assert.Equal(t, int64(16), value)

	// Verify TTL was updated
	ttl = mr.TTL(key)
	assert.Greater(t, ttl, 60*time.Second)
	assert.LessOrEqual(t, ttl, 120*time.Second)
}

func TestLuaScriptManager_ExecuteSetWithTTLJitter(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()
	key := "url:jitter_test"

	fields := map[string]string{
		"short_code": "abc123",
		"long_url":   "https://example.com/very/long/url",
		"created_at": "2026-02-03T00:00:00Z",
	}

	baseTTL := 7 * 24 * 3600 // 7 days
	jitterRange := 24 * 3600 // ±1 day

	// Execute set with TTL jitter
	err := manager.ExecuteSetWithTTLJitter(ctx, key, fields, baseTTL, jitterRange)
	require.NoError(t, err)

	// Verify hash fields were set
	shortCode := mr.HGet(key, "short_code")
	assert.Equal(t, "abc123", shortCode)

	longURL := mr.HGet(key, "long_url")
	assert.Equal(t, "https://example.com/very/long/url", longURL)

	// Verify TTL was set (should be within range)
	exists := mr.Exists(key)
	assert.True(t, exists)
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))

	// TTL should be within base ± jitter range
	minTTL := time.Duration(baseTTL-jitterRange) * time.Second
	maxTTL := time.Duration(baseTTL+jitterRange) * time.Second
	assert.GreaterOrEqual(t, ttl, minTTL)
	assert.LessOrEqual(t, ttl, maxTTL)
}

func TestLuaScriptManager_ScriptCacheFallback(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()

	// Don't preload scripts - this will trigger EVAL fallback
	key := "counter:fallback_test"

	// Execute increment - should fall back to EVAL
	value, err := manager.ExecuteIncrementAndExpire(ctx, key, 1, 60)
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)

	// Verify counter was incremented
	exists := mr.Exists(key)
	assert.True(t, exists)
}

func TestLuaScriptManager_ConcurrentExecution(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	ctx := context.Background()
	key := "counter:concurrent"

	// Preload scripts
	err := manager.PreloadScripts(ctx)
	require.NoError(t, err)

	// Execute multiple increments concurrently
	const numGoroutines = 10
	const incrementsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < incrementsPerGoroutine; j++ {
				_, err := manager.ExecuteIncrementAndExpire(ctx, key, 1, 60)
				if err != nil {
					t.Errorf("Increment failed: %v", err)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final count
	finalValue, err := client.Get(ctx, key).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*incrementsPerGoroutine), finalValue)
}

func TestLuaScriptManager_GetScriptSHA(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	// Get SHA for existing script
	sha, ok := manager.GetScriptSHA("cache_load")
	assert.True(t, ok)
	assert.NotEmpty(t, sha)
	assert.Len(t, sha, 40) // SHA1 is 40 hex characters

	// Get SHA for non-existent script
	sha, ok = manager.GetScriptSHA("non_existent")
	assert.False(t, ok)
	assert.Empty(t, sha)
}

func TestLuaScriptManager_ListScripts(t *testing.T) {
	mr, client := setupLuaTestRedis(t)
	defer mr.Close()
	defer client.Close()

	obs := createTestObservabilityForLua()
	manager := NewLuaScriptManager(client, obs)

	scripts := manager.ListScripts()
	assert.Len(t, scripts, 3)
	assert.Contains(t, scripts, "cache_load")
	assert.Contains(t, scripts, "increment_expire")
	assert.Contains(t, scripts, "set_ttl_jitter")
}

func TestIsNoScriptError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "NOSCRIPT error",
			err:      redis.Nil,
			expected: false,
		},
		{
			name:     "other error",
			err:      context.DeadlineExceeded,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNoScriptError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
