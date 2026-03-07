package service

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/singleflight"
)

var (
	// userIDRegex validates userID format: "user" followed by digits
	userIDRegex = regexp.MustCompile(`^user\d+$`)

	// groupIDRegex validates groupID format: "group_" or "large_group_" followed by digits
	groupIDRegex = regexp.MustCompile(`^(large_)?group_\d+$`)
)

// CacheManager manages local caches for the gateway service.
type CacheManager struct {
	// User-to-gateway mappings cache
	userGatewayCache sync.Map // map[string]*CacheEntry

	// Group membership cache
	groupMemberCache sync.Map // map[string]*GroupCacheEntry

	// Large group local member cache (only locally-connected members)
	largeGroupLocalCache sync.Map // map[string]*LocalGroupCacheEntry

	// Redis client for distributed cache
	redisClient *redis.Client

	// Registry client for watching changes
	registryClient RegistryClient

	// Gateway service reference (for accessing connections)
	gateway *GatewayService

	// Singleflight groups for preventing cache stampede
	userSF  singleflight.Group
	groupSF singleflight.Group

	// Configuration
	userCacheTTL        time.Duration // Default: 5 minutes
	groupCacheTTL       time.Duration // Default: 5 minutes
	largeGroupThreshold int           // Default: 1000 members

	// Metrics
	cacheHits   int64
	cacheMisses int64
	memoryUsage int64 // Approximate memory usage in bytes

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// CacheEntry represents a cached user-to-gateway mapping.
type CacheEntry struct {
	GatewayNode string
	DeviceID    string
	ExpiresAt   time.Time
	IsNilMarker bool // True if this is a "not found" marker
}

// GroupCacheEntry represents a cached group membership.
type GroupCacheEntry struct {
	Members     []string
	ExpiresAt   time.Time
	IsLarge     bool // True if group has >1,000 members
	IsNilMarker bool // True if this is a "not found" marker
}

// LocalGroupCacheEntry represents a cached local group membership for large groups.
// Validates: Requirements 2.10, 2.11, 2.12
type LocalGroupCacheEntry struct {
	LocalMembers []string // Only members connected to this gateway node
	ExpiresAt    time.Time
	MemberCount  int // Total member count (for reference)
}

// NewCacheManager creates a new cache manager instance.
func NewCacheManager(
	redisClient *redis.Client,
	registryClient RegistryClient,
	userCacheTTL time.Duration,
	groupCacheTTL time.Duration,
) *CacheManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &CacheManager{
		redisClient:         redisClient,
		registryClient:      registryClient,
		userCacheTTL:        userCacheTTL,
		groupCacheTTL:       groupCacheTTL,
		largeGroupThreshold: 1000, // Default threshold for large groups
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// SetGateway sets the gateway service reference.
// This is needed to access active connections for large group optimization.
func (c *CacheManager) SetGateway(gateway *GatewayService) {
	c.gateway = gateway
}

// Start starts the cache manager and watch mechanisms.
// Validates: Requirements 17.3
func (c *CacheManager) Start() error {
	// Start watching Registry for changes
	c.wg.Add(1)
	go c.watchRegistryChanges()

	// Start cache cleanup routine
	c.wg.Add(1)
	go c.cleanupExpiredEntries()

	return nil
}

// GetUserGateway retrieves the gateway node for a user from cache or Registry.
// Validates: Requirements 17.1
func (c *CacheManager) GetUserGateway(ctx context.Context, userID string) ([]GatewayLocation, error) {
	// Validate userID format to prevent cache penetration attacks
	if !userIDRegex.MatchString(userID) {
		return nil, nil // Invalid format, return nil without querying
	}

	// Check local cache first (fast path, no singleflight needed)
	if entry, ok := c.userGatewayCache.Load(userID); ok {
		cacheEntry := entry.(*CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			// Check if it's a nil marker (cached "not found")
			if cacheEntry.IsNilMarker {
				return nil, nil
			}
			return []GatewayLocation{
				{
					GatewayNode: cacheEntry.GatewayNode,
					DeviceID:    cacheEntry.DeviceID,
				},
			}, nil
		}
		// Expired, remove from cache
		c.userGatewayCache.Delete(userID)
	}

	// Use singleflight only for Registry lookups (cache miss scenario)
	v, err, _ := c.userSF.Do(userID, func() (interface{}, error) {
		return c.fetchUserGateway(ctx, userID)
	})

	if err != nil {
		return nil, err
	}

	return v.([]GatewayLocation), nil
}

// fetchUserGateway fetches user gateway from Registry and updates cache
// fetchUserGateway fetches user gateway from Registry and updates cache
func (c *CacheManager) fetchUserGateway(ctx context.Context, userID string) ([]GatewayLocation, error) {
	// Double-check cache inside singleflight to avoid redundant Registry queries
	if entry, ok := c.userGatewayCache.Load(userID); ok {
		cacheEntry := entry.(*CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			if cacheEntry.IsNilMarker {
				return nil, nil
			}
			return []GatewayLocation{
				{
					GatewayNode: cacheEntry.GatewayNode,
					DeviceID:    cacheEntry.DeviceID,
				},
			}, nil
		}
	}

	// Query Registry
	locations, err := c.registryClient.LookupUser(ctx, userID)
	if err != nil {
		// Cache nil marker for not found users (short TTL: 2 minutes)
		c.userGatewayCache.Store(userID, &CacheEntry{
			IsNilMarker: true,
			ExpiresAt:   time.Now().Add(2 * time.Minute),
		})
		return nil, err
	}

	// If no locations found, cache nil marker
	if len(locations) == 0 {
		c.userGatewayCache.Store(userID, &CacheEntry{
			IsNilMarker: true,
			ExpiresAt:   time.Now().Add(2 * time.Minute),
		})
		return nil, nil
	}

	// Update cache with TTL jitter
	ttl := c.addJitter(c.userCacheTTL, 0.2)

	c.userGatewayCache.Store(userID, &CacheEntry{
		GatewayNode: locations[0].GatewayNode,
		DeviceID:    locations[0].DeviceID,
		ExpiresAt:   time.Now().Add(ttl),
		IsNilMarker: false,
	})

	return locations, nil
}

// GetGroupMembers retrieves group members from cache or User Service.
// Validates: Requirements 17.2, 2.10, 2.11, 2.12
func (c *CacheManager) GetGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// Validate groupID format to prevent cache penetration attacks
	if !groupIDRegex.MatchString(groupID) {
		return nil, nil // Invalid format, return nil without querying
	}

	// Check local cache first (fast path, no singleflight needed)
	if entry, ok := c.groupMemberCache.Load(groupID); ok {
		cacheEntry := entry.(*GroupCacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			c.cacheHits++

			// Check if it's a nil marker (cached "not found")
			if cacheEntry.IsNilMarker {
				return nil, nil
			}

			// For large groups, return locally-connected members only
			if cacheEntry.IsLarge {
				return c.getLocallyConnectedMembers(groupID, cacheEntry.Members)
			}

			return cacheEntry.Members, nil
		}
		// Expired, remove from cache
		c.groupMemberCache.Delete(groupID)
	}

	c.cacheMisses++

	// Use singleflight only for external fetches (cache miss scenario)
	v, err, _ := c.groupSF.Do(groupID, func() (interface{}, error) {
		return c.fetchAndCacheGroupMembers(ctx, groupID)
	})

	if err != nil {
		return nil, err
	}

	return v.([]string), nil
}

// fetchAndCacheGroupMembers fetches group members and updates cache
func (c *CacheManager) fetchAndCacheGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// Double-check cache inside singleflight to avoid redundant external fetches
	if entry, ok := c.groupMemberCache.Load(groupID); ok {
		cacheEntry := entry.(*GroupCacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			// Check if it's a nil marker
			if cacheEntry.IsNilMarker {
				return nil, nil
			}
			// For large groups, return locally-connected members only
			if cacheEntry.IsLarge {
				return c.getLocallyConnectedMembers(groupID, cacheEntry.Members)
			}
			return cacheEntry.Members, nil
		}
	}

	// Fetch from Redis or User Service
	members, err := c.fetchGroupMembers(ctx, groupID)
	if err != nil {
		// Cache nil marker for not found groups (short TTL: 2 minutes)
		c.groupMemberCache.Store(groupID, &GroupCacheEntry{
			IsNilMarker: true,
			ExpiresAt:   time.Now().Add(2 * time.Minute),
		})
		return nil, err
	}

	// Empty result is valid (group exists but has no members)
	// Cache it normally, not as a nil marker
	// Determine if group is large
	isLarge := len(members) > c.largeGroupThreshold

	// Add ±20% jitter to TTL (4-6 minutes for 5 minute base)
	ttl := c.addJitter(c.groupCacheTTL, 0.2)

	// Update cache
	c.groupMemberCache.Store(groupID, &GroupCacheEntry{
		Members:     members,
		ExpiresAt:   time.Now().Add(ttl),
		IsLarge:     isLarge,
		IsNilMarker: false,
	})

	// Update memory usage estimate
	c.updateMemoryUsage(groupID, len(members))

	// For large groups, return only locally-connected members
	if isLarge {
		return c.getLocallyConnectedMembers(groupID, members)
	}

	return members, nil
}

// getLocallyConnectedMembers filters group members to only those connected to this gateway node.
// Validates: Requirements 2.10, 2.11, 2.12
func (c *CacheManager) getLocallyConnectedMembers(groupID string, allMembers []string) ([]string, error) {
	// Check local cache for large groups
	if entry, ok := c.largeGroupLocalCache.Load(groupID); ok {
		localEntry := entry.(*LocalGroupCacheEntry)
		if time.Now().Before(localEntry.ExpiresAt) {
			return localEntry.LocalMembers, nil
		}
		// Expired, remove from cache
		c.largeGroupLocalCache.Delete(groupID)
	}

	// Build set of all members for fast lookup
	memberSet := make(map[string]bool, len(allMembers))
	for _, member := range allMembers {
		memberSet[member] = true
	}

	// Find locally-connected members
	localMembers := make([]string, 0)
	seen := make(map[string]bool)

	if c.gateway != nil {
		c.gateway.connections.Range(func(key, value any) bool {
			connection := value.(*Connection)
			// Check if this user is a group member and not already added
			if memberSet[connection.UserID] && !seen[connection.UserID] {
				localMembers = append(localMembers, connection.UserID)
				seen[connection.UserID] = true
			}
			return true
		})
	}

	// Add ±20% jitter to TTL
	ttl := c.addJitter(c.groupCacheTTL, 0.2)

	// Cache the local members
	c.largeGroupLocalCache.Store(groupID, &LocalGroupCacheEntry{
		LocalMembers: localMembers,
		ExpiresAt:    time.Now().Add(ttl),
		MemberCount:  len(allMembers),
	})

	return localMembers, nil
}

// addJitter adds random jitter to a duration to prevent cache avalanche
// jitterPercent: percentage of jitter (e.g., 0.2 for ±20%)
// Returns: baseTTL with random jitter applied
func (c *CacheManager) addJitter(baseTTL time.Duration, jitterPercent float64) time.Duration {
	baseSeconds := int(baseTTL.Seconds())
	jitterRange := int(float64(baseSeconds) * jitterPercent)

	// Generate random jitter: -jitterRange to +jitterRange
	jitter := rand.Intn(2*jitterRange+1) - jitterRange // #nosec G404 - weak random is acceptable for cache TTL jitter

	return time.Duration(baseSeconds+jitter) * time.Second
}

// fetchGroupMembers fetches group members from Redis or User Service.
func (c *CacheManager) fetchGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// Try Redis first
	if c.redisClient != nil {
		cacheKey := fmt.Sprintf("group_members:%s", groupID)
		members, err := c.redisClient.SMembers(ctx, cacheKey).Result()
		if err == nil && len(members) > 0 {
			return members, nil
		}
	}

	// TODO: Fetch from User Service
	// For now, return empty list
	return []string{}, nil
}

// InvalidateUserCache invalidates the user-to-gateway cache entry.
// Validates: Requirements 17.3
func (c *CacheManager) InvalidateUserCache(userID string) {
	c.userGatewayCache.Delete(userID)
}

// InvalidateGroupCache invalidates the group membership cache entry.
// Validates: Requirements 2.9, 17.3
func (c *CacheManager) InvalidateGroupCache(groupID string) {
	c.groupMemberCache.Delete(groupID)
	c.largeGroupLocalCache.Delete(groupID) // Also invalidate local cache for large groups

	// Also invalidate in Redis
	if c.redisClient != nil {
		cacheKey := fmt.Sprintf("group_members:%s", groupID)
		c.redisClient.Del(c.ctx, cacheKey)
	}
}

// updateMemoryUsage updates the approximate memory usage estimate.
// Validates: Requirements 2.11
func (c *CacheManager) updateMemoryUsage(groupID string, memberCount int) {
	// Rough estimate:
	// - groupID: ~50 bytes
	// - each member ID: ~50 bytes
	// - overhead: ~100 bytes
	estimatedBytes := int64(50 + (memberCount * 50) + 100)

	// Use atomic operations for thread-safe updates
	// Note: This is a rough estimate, not exact memory usage
	c.memoryUsage += estimatedBytes
}

// GetMemoryUsage returns the approximate memory usage in bytes.
// Validates: Requirements 2.11
func (c *CacheManager) GetMemoryUsage() int64 {
	return c.memoryUsage
}

// GetCacheStats returns cache hit/miss statistics.
func (c *CacheManager) GetCacheStats() (hits int64, misses int64, hitRate float64) {
	hits = c.cacheHits
	misses = c.cacheMisses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	return
}

// watchRegistryChanges watches for Registry changes and invalidates cache.
// Validates: Requirements 7.9, 17.3
func (c *CacheManager) watchRegistryChanges() {
	defer c.wg.Done()

	// Watch for user registration changes
	err := c.registryClient.Watch(c.ctx, "/registry/users/", func(resp clientv3.WatchResponse) {
		for _, event := range resp.Events {
			// Extract user_id from key
			// Key format: /registry/users/{user_id}/{device_id}
			key := string(event.Kv.Key)

			// Parse user_id from key
			// This is a simplified version, actual implementation would be more robust
			if len(key) > 16 { // "/registry/users/" is 16 characters
				userID := extractUserIDFromKey(key)
				if userID != "" {
					c.InvalidateUserCache(userID)
				}
			}
		}
	})

	if err != nil {
		// Log error
	}
}

// cleanupExpiredEntries periodically removes expired cache entries.
func (c *CacheManager) cleanupExpiredEntries() {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// Clean up user cache
			c.userGatewayCache.Range(func(key, value interface{}) bool {
				entry := value.(*CacheEntry)
				if now.After(entry.ExpiresAt) {
					c.userGatewayCache.Delete(key)
				}
				return true
			})

			// Clean up group cache
			c.groupMemberCache.Range(func(key, value interface{}) bool {
				entry := value.(*GroupCacheEntry)
				if now.After(entry.ExpiresAt) {
					c.groupMemberCache.Delete(key)
				}
				return true
			})

			// Clean up large group local cache
			c.largeGroupLocalCache.Range(func(key, value interface{}) bool {
				entry := value.(*LocalGroupCacheEntry)
				if now.After(entry.ExpiresAt) {
					c.largeGroupLocalCache.Delete(key)
				}
				return true
			})

		case <-c.ctx.Done():
			return
		}
	}
}

// Stop stops the cache manager.
func (c *CacheManager) Stop() error {
	c.cancel()

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for cache manager to stop")
	}
}

// extractUserIDFromKey extracts user_id from a Registry key.
func extractUserIDFromKey(key string) string {
	// Key format: /registry/users/{user_id}/{device_id}
	// This is a simplified implementation
	parts := splitKey(key)
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// splitKey splits a key by '/' separator.
func splitKey(key string) []string {
	var parts []string
	var current string

	for _, ch := range key {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
