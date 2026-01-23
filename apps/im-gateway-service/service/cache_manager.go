package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// CacheManager manages local caches for the gateway service.
type CacheManager struct {
	// User-to-gateway mappings cache
	userGatewayCache sync.Map // map[string]*CacheEntry

	// Group membership cache
	groupMemberCache sync.Map // map[string]*GroupCacheEntry

	// Redis client for distributed cache
	redisClient *redis.Client

	// Registry client for watching changes
	registryClient RegistryClient

	// Configuration
	userCacheTTL  time.Duration // Default: 5 minutes
	groupCacheTTL time.Duration // Default: 5 minutes

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
}

// GroupCacheEntry represents a cached group membership.
type GroupCacheEntry struct {
	Members   []string
	ExpiresAt time.Time
	IsLarge   bool // True if group has >1,000 members
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
		redisClient:    redisClient,
		registryClient: registryClient,
		userCacheTTL:   userCacheTTL,
		groupCacheTTL:  groupCacheTTL,
		ctx:            ctx,
		cancel:         cancel,
	}
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
	// Check local cache first
	if entry, ok := c.userGatewayCache.Load(userID); ok {
		cacheEntry := entry.(*CacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
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

	// Cache miss, query Registry
	locations, err := c.registryClient.LookupUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update cache
	if len(locations) > 0 {
		// Cache the first location (for simplicity)
		c.userGatewayCache.Store(userID, &CacheEntry{
			GatewayNode: locations[0].GatewayNode,
			DeviceID:    locations[0].DeviceID,
			ExpiresAt:   time.Now().Add(c.userCacheTTL),
		})
	}

	return locations, nil
}

// GetGroupMembers retrieves group members from cache or User Service.
// Validates: Requirements 17.2
func (c *CacheManager) GetGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// Check local cache first
	if entry, ok := c.groupMemberCache.Load(groupID); ok {
		cacheEntry := entry.(*GroupCacheEntry)
		if time.Now().Before(cacheEntry.ExpiresAt) {
			return cacheEntry.Members, nil
		}
		// Expired, remove from cache
		c.groupMemberCache.Delete(groupID)
	}

	// Cache miss, query Redis or User Service
	members, err := c.fetchGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Determine if group is large
	isLarge := len(members) > 1000

	// Update cache
	c.groupMemberCache.Store(groupID, &GroupCacheEntry{
		Members:   members,
		ExpiresAt: time.Now().Add(c.groupCacheTTL),
		IsLarge:   isLarge,
	})

	return members, nil
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

	// Also invalidate in Redis
	if c.redisClient != nil {
		cacheKey := fmt.Sprintf("group_members:%s", groupID)
		c.redisClient.Del(c.ctx, cacheKey)
	}
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
