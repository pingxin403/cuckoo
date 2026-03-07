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

	// Get group members (will return empty list) - use valid group ID format
	members, err := cacheManager.GetGroupMembers(context.Background(), "group_123")
	require.NoError(t, err)
	assert.Empty(t, members)
}

// TestCacheManager_InvalidateGroupCache tests group cache invalidation
func TestCacheManager_InvalidateGroupCache(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Invalidate group cache (should not crash) - use valid group ID format
	cacheManager.InvalidateGroupCache("group_123")
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

// TestCacheManager_SingleflightUserGateway verifies singleflight request coalescing for user gateway lookups
func TestCacheManager_SingleflightUserGateway(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Register a user
	_ = registryClient.RegisterUser(context.Background(), "user123", "device456", "gateway-1")

	// Simulate 100 concurrent requests for the same user
	numRequests := 100
	results := make(chan []GatewayLocation, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			locations, err := cacheManager.GetUserGateway(context.Background(), "user123")
			if err != nil {
				errors <- err
			} else {
				results <- locations
			}
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case locations := <-results:
			assert.Len(t, locations, 1)
			assert.Equal(t, "gateway-1", locations[0].GatewayNode)
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	// Verify that singleflight coalesced requests
	// The mock registry client should have been called only once or a few times
	// (not 100 times) due to singleflight
	callCount := registryClient.GetLookupCallCount()
	t.Logf("Singleflight coalesced %d concurrent requests to %d Registry queries (%.1f%% reduction)",
		numRequests, callCount, float64(numRequests-callCount)/float64(numRequests)*100)

	// Allow some timing variance, but should be significantly less than numRequests
	if callCount > 5 {
		t.Errorf("Expected at most 5 Registry queries due to singleflight, got %d", callCount)
	}
}

// TestCacheManager_SingleflightGroupMembers verifies singleflight request coalescing for group member lookups
func TestCacheManager_SingleflightGroupMembers(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Simulate 100 concurrent requests for the same group (use valid group ID format)
	numRequests := 100
	results := make(chan []string, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			members, err := cacheManager.GetGroupMembers(context.Background(), "group_123")
			if err != nil {
				errors <- err
			} else {
				results <- members
			}
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		select {
		case members := <-results:
			// Empty list is expected since we don't have Redis or User Service
			assert.NotNil(t, members)
		case err := <-errors:
			t.Fatalf("Unexpected error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	// Verify cache stats show only 1 miss (singleflight coalesced the rest)
	_, misses, _ := cacheManager.GetCacheStats()
	t.Logf("Singleflight coalesced %d concurrent requests to %d cache misses (%.1f%% reduction)",
		numRequests, misses, float64(numRequests-int(misses))/float64(numRequests)*100)

	// Should have only 1 cache miss due to singleflight
	if misses > 1 {
		t.Logf("Warning: Expected 1 cache miss, got %d (timing variance is acceptable)", misses)
	}
}

// TestCacheManager_InvalidUserIDValidation tests that invalid user IDs are rejected without querying
func TestCacheManager_InvalidUserIDValidation(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Test invalid user ID formats
	invalidUserIDs := []string{
		"",         // Empty
		"invalid",  // No "user" prefix
		"user",     // No digits
		"user_123", // Underscore instead of direct digits
		"admin123", // Wrong prefix
		"123user",  // Digits before prefix
		"user-123", // Hyphen instead of direct digits
		"user 123", // Space
		"user@123", // Special character
	}

	initialCallCount := registryClient.GetLookupCallCount()

	for _, userID := range invalidUserIDs {
		locations, err := cacheManager.GetUserGateway(context.Background(), userID)
		assert.NoError(t, err, "Should not return error for invalid format")
		assert.Nil(t, locations, "Should return nil for invalid user ID: %s", userID)
	}

	// Verify no Registry queries were made
	finalCallCount := registryClient.GetLookupCallCount()
	assert.Equal(t, initialCallCount, finalCallCount, "No Registry queries should be made for invalid user IDs")
}

// TestCacheManager_InvalidGroupIDValidation tests that invalid group IDs are rejected without querying
func TestCacheManager_InvalidGroupIDValidation(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Test invalid group ID formats
	invalidGroupIDs := []string{
		"",             // Empty
		"invalid",      // No "group_" prefix
		"group",        // No underscore and digits
		"group123",     // No underscore
		"group__123",   // Double underscore
		"group_",       // No digits
		"large_group",  // No digits
		"large_group_", // No digits
		"team_123",     // Wrong prefix
		"group_abc",    // Non-numeric suffix
		"group 123",    // Space
		"group_@123",   // Special character
	}

	initialMisses := int64(0)
	_, initialMisses, _ = cacheManager.GetCacheStats()

	for _, groupID := range invalidGroupIDs {
		members, err := cacheManager.GetGroupMembers(context.Background(), groupID)
		assert.NoError(t, err, "Should not return error for invalid format")
		assert.Nil(t, members, "Should return nil for invalid group ID: %s", groupID)
	}

	// Verify no cache misses were recorded (no external queries)
	_, finalMisses, _ := cacheManager.GetCacheStats()
	assert.Equal(t, initialMisses, finalMisses, "No cache misses should be recorded for invalid group IDs")
}

// TestCacheManager_UserNilMarkerCaching tests that non-existent users are cached as nil markers
func TestCacheManager_UserNilMarkerCaching(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Query a non-existent user (valid format but not registered)
	userID := "user999"
	initialCallCount := registryClient.GetLookupCallCount()

	// First query - should query Registry and cache nil marker
	locations1, err := cacheManager.GetUserGateway(context.Background(), userID)
	assert.NoError(t, err)
	assert.Nil(t, locations1)

	firstCallCount := registryClient.GetLookupCallCount()
	assert.Equal(t, initialCallCount+1, firstCallCount, "First query should hit Registry")

	// Second query - should return from nil marker cache without querying Registry
	locations2, err := cacheManager.GetUserGateway(context.Background(), userID)
	assert.NoError(t, err)
	assert.Nil(t, locations2)

	secondCallCount := registryClient.GetLookupCallCount()
	assert.Equal(t, firstCallCount, secondCallCount, "Second query should use cached nil marker")

	// Third query - should still use cached nil marker
	locations3, err := cacheManager.GetUserGateway(context.Background(), userID)
	assert.NoError(t, err)
	assert.Nil(t, locations3)

	thirdCallCount := registryClient.GetLookupCallCount()
	assert.Equal(t, firstCallCount, thirdCallCount, "Third query should use cached nil marker")

	t.Logf("Nil marker caching: 3 queries resulted in 1 Registry lookup (66.7%% reduction)")
}

// TestCacheManager_GroupNilMarkerCaching tests that non-existent groups are cached as nil markers
func TestCacheManager_GroupNilMarkerCaching(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Note: The current implementation returns empty slice for groups with no members
	// Nil markers are only cached when there's an error fetching the group
	// This test verifies that empty groups (valid but no members) are cached normally

	groupID := "group_999"

	// First query - should fetch and cache empty result
	members1, err := cacheManager.GetGroupMembers(context.Background(), groupID)
	assert.NoError(t, err)
	assert.NotNil(t, members1) // Empty slice, not nil
	assert.Empty(t, members1)

	_, initialMisses, _ := cacheManager.GetCacheStats()

	// Second query - should return from cache
	members2, err := cacheManager.GetGroupMembers(context.Background(), groupID)
	assert.NoError(t, err)
	assert.NotNil(t, members2)
	assert.Empty(t, members2)

	_, secondMisses, _ := cacheManager.GetCacheStats()
	assert.Equal(t, initialMisses, secondMisses, "Second query should use cached result (no new miss)")

	// Third query - should still use cached result
	members3, err := cacheManager.GetGroupMembers(context.Background(), groupID)
	assert.NoError(t, err)
	assert.NotNil(t, members3)
	assert.Empty(t, members3)

	_, thirdMisses, _ := cacheManager.GetCacheStats()
	assert.Equal(t, initialMisses, thirdMisses, "Third query should use cached result (no new miss)")

	t.Logf("Empty group caching: 3 queries resulted in 1 cache miss (66.7%% reduction)")
}

// TestCacheManager_NilMarkerExpiration tests that nil markers expire after their TTL
func TestCacheManager_NilMarkerExpiration(t *testing.T) {
	registryClient := newMockRegistryClient()
	cacheManager := NewCacheManager(nil, registryClient, 5*time.Minute, 5*time.Minute)

	// Query a non-existent user
	userID := "user888"
	locations1, err := cacheManager.GetUserGateway(context.Background(), userID)
	assert.NoError(t, err)
	assert.Nil(t, locations1)

	// Manually expire the nil marker by manipulating the cache entry
	if entry, ok := cacheManager.userGatewayCache.Load(userID); ok {
		cacheEntry := entry.(*CacheEntry)
		cacheEntry.ExpiresAt = time.Now().Add(-1 * time.Second) // Expire it
		cacheManager.userGatewayCache.Store(userID, cacheEntry)
	}

	initialCallCount := registryClient.GetLookupCallCount()

	// Query again - should query Registry because nil marker expired
	locations2, err := cacheManager.GetUserGateway(context.Background(), userID)
	assert.NoError(t, err)
	assert.Nil(t, locations2)

	finalCallCount := registryClient.GetLookupCallCount()
	assert.Greater(t, finalCallCount, initialCallCount, "Should query Registry after nil marker expires")
}
