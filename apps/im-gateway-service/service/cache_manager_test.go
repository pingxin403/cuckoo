package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheManager_GetUserGateway_CacheMiss tests cache miss scenario
func TestCacheManager_GetUserGateway_CacheMiss(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Register a user
	_ = registryClient.RegisterUser(context.Background(), "user123", "device456", "gateway-1")

	// Get user gateway (cache miss)
	locations, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, "gateway-1", locations[0].GatewayNode)
	assert.Equal(t, "device456", locations[0].DeviceID)
}

// TestCacheManager_GetUserGateway_CacheHit tests cache hit scenario
func TestCacheManager_GetUserGateway_CacheHit(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Register a user
	_ = registryClient.RegisterUser(context.Background(), "user123", "device456", "gateway-1")

	// First call - cache miss
	locations1, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations1, 1)

	// Second call - cache hit
	locations2, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations2, 1)
	assert.Equal(t, locations1[0].GatewayNode, locations2[0].GatewayNode)
}

// TestCacheManager_InvalidateUserCache tests cache invalidation
func TestCacheManager_InvalidateUserCache(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Register a user
	_ = registryClient.RegisterUser(context.Background(), "user123", "device456", "gateway-1")

	// Get user gateway (cache miss)
	locations1, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations1, 1)

	// Invalidate cache
	cacheManager.InvalidateUserCache("user123")

	// Update registry
	_ = registryClient.UnregisterUser(context.Background(), "user123", "device456")
	_ = registryClient.RegisterUser(context.Background(), "user123", "device789", "gateway-2")

	// Get user gateway again (should query registry)
	locations2, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations2, 1)
	assert.Equal(t, "gateway-2", locations2[0].GatewayNode)
	assert.Equal(t, "device789", locations2[0].DeviceID)
}

// TestCacheManager_GetGroupMembers tests group member retrieval
func TestCacheManager_GetGroupMembers(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Get group members (will return empty list)
	members, err := cacheManager.GetGroupMembers(context.Background(), "group123")
	require.NoError(t, err)
	assert.Empty(t, members)
}

// TestCacheManager_InvalidateGroupCache tests group cache invalidation
func TestCacheManager_InvalidateGroupCache(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Invalidate group cache (should not crash)
	cacheManager.InvalidateGroupCache("group123")
}

// TestCacheManager_StartStop tests cache manager lifecycle
func TestCacheManager_StartStop(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Start cache manager
	err := cacheManager.Start()
	require.NoError(t, err)

	// Give it some time to run
	time.Sleep(100 * time.Millisecond)

	// Stop cache manager
	err = cacheManager.Stop()
	require.NoError(t, err)
}

// TestCacheManager_ExpiredEntries tests cache entry expiration
func TestCacheManager_ExpiredEntries(t *testing.T) {
	registryClient := newMockRegistryClient()
	// Use very short TTL for testing
	cacheManager := NewCacheManager(nil, registryClient, 100*time.Millisecond, 100*time.Millisecond)

	// Register a user
	_ = registryClient.RegisterUser(context.Background(), "user123", "device456", "gateway-1")

	// Get user gateway (cache miss)
	locations1, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations1, 1)

	// Wait for cache to expire
	time.Sleep(200 * time.Millisecond)

	// Get user gateway again (should query registry because cache expired)
	locations2, err := cacheManager.GetUserGateway(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations2, 1)
}
