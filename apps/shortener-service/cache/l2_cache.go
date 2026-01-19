package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

// L2Cache represents the Redis cache layer
// Requirements: 4.2, 4.6
type L2Cache struct {
	client redis.UniversalClient
}

// L2CacheConfig holds configuration for Redis cache
type L2CacheConfig struct {
	// Addrs is a list of Redis server addresses
	// For standalone: ["localhost:6379"]
	// For cluster: ["node1:6379", "node2:6379", "node3:6379"]
	Addrs []string

	// Password for Redis authentication (optional)
	Password string

	// DB is the Redis database number (0-15, only for standalone)
	DB int

	// PoolSize is the maximum number of socket connections
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int
}

// NewL2Cache creates a new L2 Redis cache instance
// Supports both standalone and cluster configurations
// Requirements: 4.2
func NewL2Cache(config L2CacheConfig) (*L2Cache, error) {
	if len(config.Addrs) == 0 {
		return nil, fmt.Errorf("at least one Redis address is required")
	}

	// Set defaults
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 5
	}

	// Create universal client (supports both standalone and cluster)
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        config.Addrs,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &L2Cache{client: client}, nil
}

// Get retrieves a URL mapping from Redis
// Returns nil if the key is not found
// Requirements: 4.2
func (c *L2Cache) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	// Use Redis Hash to store URL mapping
	key := fmt.Sprintf("url:%s", shortCode)

	result, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // Not found
	}

	// Parse the mapping
	mapping := &URLMapping{
		ShortCode: result["short_code"],
		LongURL:   result["long_url"],
	}

	// Parse created_at timestamp
	if createdAtStr, ok := result["created_at"]; ok {
		createdAt, err := time.Parse(time.RFC3339, createdAtStr)
		if err == nil {
			mapping.CreatedAt = createdAt
		}
	}

	return mapping, nil
}

// Set stores a URL mapping in Redis with TTL jitter
// TTL: 7 days ±1 day (6-8 days) to prevent cache expiration stampede
// Requirements: 4.2
func (c *L2Cache) Set(ctx context.Context, shortCode string, longURL string, createdAt time.Time) error {
	key := fmt.Sprintf("url:%s", shortCode)

	// Prepare hash fields
	fields := map[string]interface{}{
		"short_code": shortCode,
		"long_url":   longURL,
		"created_at": createdAt.Format(time.RFC3339),
	}

	// Set the hash
	if err := c.client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	// Calculate TTL with ±1 day jitter to prevent thundering herd
	// Base TTL: 7 days (604800 seconds)
	// Jitter range: 6-8 days (518400-691200 seconds)
	baseTTL := 7 * 24 * 3600 // 7 days in seconds
	jitterRange := 24 * 3600 // 1 day in seconds

	// Generate random jitter: -1 day to +1 day
	jitter := rand.Intn(2*jitterRange+1) - jitterRange // #nosec G404 - weak random is acceptable for cache TTL jitter
	ttlSeconds := baseTTL + jitter
	ttl := time.Duration(ttlSeconds) * time.Second

	// Set TTL
	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL in Redis: %w", err)
	}

	return nil
}

// Delete removes a URL mapping from Redis
// Requirements: 4.6
func (c *L2Cache) Delete(ctx context.Context, shortCode string) error {
	key := fmt.Sprintf("url:%s", shortCode)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

// BatchGet retrieves multiple URL mappings from Redis
// Returns a map of shortCode -> URLMapping
// Requirements: 4.2
func (c *L2Cache) BatchGet(ctx context.Context, shortCodes []string) (map[string]*URLMapping, error) {
	if len(shortCodes) == 0 {
		return make(map[string]*URLMapping), nil
	}

	// Use pipeline for batch operations
	pipe := c.client.Pipeline()

	// Queue all HGETALL commands
	cmds := make(map[string]*redis.MapStringStringCmd)
	for _, shortCode := range shortCodes {
		key := fmt.Sprintf("url:%s", shortCode)
		cmds[shortCode] = pipe.HGetAll(ctx, key)
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to batch get from Redis: %w", err)
	}

	// Parse results
	results := make(map[string]*URLMapping)
	for shortCode, cmd := range cmds {
		result, err := cmd.Result()
		if err != nil || len(result) == 0 {
			continue
		}

		mapping := &URLMapping{
			ShortCode: result["short_code"],
			LongURL:   result["long_url"],
		}

		if createdAtStr, ok := result["created_at"]; ok {
			createdAt, err := time.Parse(time.RFC3339, createdAtStr)
			if err == nil {
				mapping.CreatedAt = createdAt
			}
		}

		results[shortCode] = mapping
	}

	return results, nil
}

// BatchDelete removes multiple URL mappings from Redis
// Requirements: 4.6
func (c *L2Cache) BatchDelete(ctx context.Context, shortCodes []string) error {
	if len(shortCodes) == 0 {
		return nil
	}

	// Prepare keys
	keys := make([]string, len(shortCodes))
	for i, shortCode := range shortCodes {
		keys[i] = fmt.Sprintf("url:%s", shortCode)
	}

	// Delete all keys
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to batch delete from Redis: %w", err)
	}

	return nil
}

// Ping checks if Redis is reachable
// Requirements: 4.2
func (c *L2Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *L2Cache) Close() error {
	return c.client.Close()
}

// Stats returns Redis statistics
func (c *L2Cache) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// MarshalBinary implements encoding.BinaryMarshaler for URLMapping
func (m *URLMapping) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler for URLMapping
func (m *URLMapping) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}
