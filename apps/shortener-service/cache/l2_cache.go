package cache

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// L2Cache represents the Redis cache layer
type L2Cache struct {
	client         redis.UniversalClient
	obs            observability.Observability
	circuitBreaker *CircuitBreaker
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
func NewL2Cache(config L2CacheConfig, obs observability.Observability) (*L2Cache, error) {
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

	// Create circuit breaker for Redis operations
	cbConfig := DefaultCircuitBreakerConfig()
	circuitBreaker := NewCircuitBreaker(cbConfig, obs)

	return &L2Cache{
		client:         client,
		obs:            obs,
		circuitBreaker: circuitBreaker,
	}, nil
}

// Get retrieves a URL mapping from Redis
// Returns nil if the key is not found
func (c *L2Cache) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	var mapping *URLMapping
	var getErr error

	// Wrap Redis operation with circuit breaker
	err := c.circuitBreaker.Execute(ctx, func() error {
		// Use Redis Hash to store URL mapping
		key := fmt.Sprintf("url:%s", shortCode)

		result, err := c.client.HGetAll(ctx, key).Result()
		if err != nil {
			return fmt.Errorf("failed to get from Redis: %w", err)
		}

		if len(result) == 0 {
			mapping = nil
			return nil // Not found
		}

		// Check for empty marker to prevent cache penetration
		longURL := result["long_url"]
		if longURL == "__EMPTY__" || longURL == "" {
			c.obs.Metrics().IncrementCounter("redis_empty_cache_hits_total", nil)
			c.obs.Logger().Debug(ctx, "Empty cache hit", "short_code", shortCode)
			// Return a null entry mapping instead of nil
			mapping = &URLMapping{
				ShortCode: shortCode,
				LongURL:   "", // Empty URL indicates null entry
				CreatedAt: time.Now(),
			}
			return nil
		}

		// Parse the mapping
		mapping = &URLMapping{
			ShortCode: result["short_code"],
			LongURL:   longURL,
		}

		// Parse created_at timestamp
		if createdAtStr, ok := result["created_at"]; ok {
			createdAt, err := time.Parse(time.RFC3339, createdAtStr)
			if err == nil {
				mapping.CreatedAt = createdAt
			}
		}

		// Parse expires_at timestamp
		if expiresAtStr, ok := result["expires_at"]; ok && expiresAtStr != "" {
			expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
			if err == nil {
				mapping.ExpiresAt = &expiresAt
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapping, getErr
}

// Set stores a URL mapping in Redis with TTL jitter
// TTL: 7 days ±1 day (6-8 days) to prevent cache expiration stampede
// Uses crypto/rand for secure random number generation
func (c *L2Cache) Set(ctx context.Context, shortCode string, longURL string, createdAt time.Time, expiresAt *time.Time) error {
	// Wrap Redis operation with circuit breaker
	return c.circuitBreaker.Execute(ctx, func() error {
		key := fmt.Sprintf("url:%s", shortCode)

		// Prepare hash fields
		fields := map[string]interface{}{
			"short_code": shortCode,
			"long_url":   longURL,
			"created_at": createdAt.Format(time.RFC3339),
		}

		// Add expires_at if present
		if expiresAt != nil {
			fields["expires_at"] = expiresAt.Format(time.RFC3339)
		}

		// Set the hash
		if err := c.client.HSet(ctx, key, fields).Err(); err != nil {
			return fmt.Errorf("failed to set in Redis: %w", err)
		}

		// Calculate TTL with ±1 day jitter to prevent thundering herd
		// Base TTL: 7 days (604800 seconds)
		// Jitter range: ±1 day (86400 seconds)
		baseTTL := 7 * 24 * 3600 // 7 days in seconds
		jitterRange := 24 * 3600 // 1 day in seconds

		// Generate cryptographically secure random jitter: -1 day to +1 day
		var randomBytes [8]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			return fmt.Errorf("failed to generate random jitter: %w", err)
		}

		// Convert random bytes to uint64, then to int in range [0, 2*jitterRange]
		randomUint64 := binary.BigEndian.Uint64(randomBytes[:])
		jitter := int(randomUint64%uint64(2*jitterRange+1)) - jitterRange

		ttlSeconds := baseTTL + jitter
		ttl := time.Duration(ttlSeconds) * time.Second

		// Track TTL distribution metrics
		c.obs.Metrics().RecordHistogram("redis_ttl_seconds", ttl.Seconds(), map[string]string{
			"layer": "L2",
		})

		// Set TTL
		if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
			return fmt.Errorf("failed to set TTL in Redis: %w", err)
		}

		return nil
	})
}

// Delete removes a URL mapping from Redis
func (c *L2Cache) Delete(ctx context.Context, shortCode string) error {
	// Wrap Redis operation with circuit breaker
	return c.circuitBreaker.Execute(ctx, func() error {
		key := fmt.Sprintf("url:%s", shortCode)

		if err := c.client.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete from Redis: %w", err)
		}

		return nil
	})
}

// SetWithTTL stores a URL mapping in Redis with custom TTL
// This is useful for caching null entries with short TTL to prevent cache penetration
func (c *L2Cache) SetWithTTL(ctx context.Context, shortCode string, longURL string, createdAt time.Time, ttl time.Duration) error {
	// Wrap Redis operation with circuit breaker
	return c.circuitBreaker.Execute(ctx, func() error {
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

		// Set custom TTL
		if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
			return fmt.Errorf("failed to set TTL in Redis: %w", err)
		}

		return nil
	})
}

// BatchGet retrieves multiple URL mappings from Redis
// Returns a map of shortCode -> URLMapping
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
func (c *L2Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *L2Cache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client for health checks
func (c *L2Cache) Client() redis.UniversalClient {
	return c.client
}

// CircuitBreaker returns the circuit breaker for monitoring
func (c *L2Cache) CircuitBreaker() *CircuitBreaker {
	return c.circuitBreaker
}

// Stats returns Redis statistics
func (c *L2Cache) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// SetEmpty caches an empty result to prevent cache penetration
func (c *L2Cache) SetEmpty(ctx context.Context, shortCode string) error {
	// Wrap Redis operation with circuit breaker
	return c.circuitBreaker.Execute(ctx, func() error {
		key := fmt.Sprintf("url:%s", shortCode)

		// Use special marker for empty values
		emptyMarker := map[string]interface{}{
			"short_code": shortCode,
			"long_url":   "__EMPTY__", // Special marker for empty cache
			"created_at": time.Now().Format(time.RFC3339),
		}

		// Short TTL for empty values (5 minutes instead of 7 days)
		ttl := 5 * time.Minute

		// Use pipeline for atomic operation
		pipe := c.client.Pipeline()
		pipe.HSet(ctx, key, emptyMarker)
		pipe.Expire(ctx, key, ttl)

		_, err := pipe.Exec(ctx)
		if err != nil {
			c.obs.Metrics().IncrementCounter("redis_empty_cache_set_errors_total", nil)
			return fmt.Errorf("failed to set empty cache: %w", err)
		}

		c.obs.Metrics().IncrementCounter("redis_empty_cache_set_total", nil)
		c.obs.Logger().Debug(ctx, "Cached empty value",
			"short_code", shortCode,
			"ttl_seconds", int(ttl.Seconds()))

		return nil
	})
}

// MarshalBinary implements encoding.BinaryMarshaler for URLMapping
func (m *URLMapping) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler for URLMapping
func (m *URLMapping) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}
