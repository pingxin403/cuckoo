package cache

import (
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
)

// RedisConfig holds comprehensive configuration for Redis connection pool optimization
// This configuration supports both standalone and cluster modes with optimized pool settings
type RedisConfig struct {
	// Connection settings
	// Addrs is a list of Redis server addresses
	// For standalone: ["localhost:6379"]
	// For cluster: ["node1:6379", "node2:6379", "node3:6379"]
	Addrs []string

	// Password for Redis authentication (optional)
	Password string

	// DB is the Redis database number (0-15, only for standalone mode)
	DB int

	// ClusterMode enables Redis Cluster support
	// When true, uses NewClusterClient; when false, uses NewClient
	ClusterMode bool

	// Connection pool settings
	// PoolSize is the maximum number of socket connections
	// Recommended: QPS/1000 (minimum 10, maximum 50)
	// Default: 20
	PoolSize int

	// MinIdleConns is the minimum number of idle connections to maintain
	// Recommended: 30% of PoolSize to avoid cold start delays
	// Default: 6 (30% of default PoolSize)
	MinIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection
	// Recommended: 30-60 minutes to prevent connection staleness
	// Default: 30 minutes
	ConnMaxLifetime time.Duration

	// Timeout settings
	// DialTimeout is the timeout for establishing new connections
	// Default: 5 seconds
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads
	// Default: 3 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes
	// Default: 3 seconds
	WriteTimeout time.Duration

	// Cluster settings (only applicable when ClusterMode is true)
	// MaxRedirects is the maximum number of retries for MOVED/ASK redirects
	// Default: 3
	MaxRedirects int
}

// DefaultRedisConfig returns a RedisConfig with recommended default values
// optimized for production workloads (20K QPS baseline)
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addrs:           []string{"localhost:6379"},
		Password:        "",
		DB:              0,
		ClusterMode:     false,
		PoolSize:        20, // Suitable for ~20K QPS
		MinIdleConns:    6,  // 30% of PoolSize
		ConnMaxLifetime: 30 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MaxRedirects:    3,
	}
}

// Validate checks if the RedisConfig has valid values
// Returns an error if any configuration value is invalid
func (c *RedisConfig) Validate() error {
	// Validate addresses
	if len(c.Addrs) == 0 {
		return fmt.Errorf("at least one Redis address is required")
	}

	// Validate pool size
	if c.PoolSize < 1 {
		return fmt.Errorf("PoolSize must be at least 1, got %d", c.PoolSize)
	}
	if c.PoolSize > 100 {
		return fmt.Errorf("PoolSize should not exceed 100 for optimal performance, got %d", c.PoolSize)
	}

	// Validate MinIdleConns
	if c.MinIdleConns < 0 {
		return fmt.Errorf("MinIdleConns cannot be negative, got %d", c.MinIdleConns)
	}
	if c.MinIdleConns > c.PoolSize {
		return fmt.Errorf("MinIdleConns (%d) cannot exceed PoolSize (%d)", c.MinIdleConns, c.PoolSize)
	}

	// Validate ConnMaxLifetime
	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("ConnMaxLifetime cannot be negative, got %v", c.ConnMaxLifetime)
	}
	if c.ConnMaxLifetime > 0 && c.ConnMaxLifetime < 1*time.Minute {
		return fmt.Errorf("ConnMaxLifetime should be at least 1 minute if set, got %v", c.ConnMaxLifetime)
	}

	// Validate timeouts
	if c.DialTimeout <= 0 {
		return fmt.Errorf("DialTimeout must be positive, got %v", c.DialTimeout)
	}
	if c.DialTimeout > 30*time.Second {
		return fmt.Errorf("DialTimeout should not exceed 30 seconds, got %v", c.DialTimeout)
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("ReadTimeout must be positive, got %v", c.ReadTimeout)
	}
	if c.ReadTimeout > 10*time.Second {
		return fmt.Errorf("ReadTimeout should not exceed 10 seconds, got %v", c.ReadTimeout)
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("WriteTimeout must be positive, got %v", c.WriteTimeout)
	}
	if c.WriteTimeout > 10*time.Second {
		return fmt.Errorf("WriteTimeout should not exceed 10 seconds, got %v", c.WriteTimeout)
	}

	// Validate cluster settings
	if c.ClusterMode {
		// Cluster mode requires at least 3 nodes for proper operation
		if len(c.Addrs) < 3 {
			return fmt.Errorf("cluster mode requires at least 3 addresses, got %d", len(c.Addrs))
		}

		// Validate MaxRedirects
		if c.MaxRedirects < 1 {
			return fmt.Errorf("MaxRedirects must be at least 1 for cluster mode, got %d", c.MaxRedirects)
		}
		if c.MaxRedirects > 10 {
			return fmt.Errorf("MaxRedirects should not exceed 10, got %d", c.MaxRedirects)
		}
	}

	// Validate DB number (only for standalone mode)
	if !c.ClusterMode {
		if c.DB < 0 || c.DB > 15 {
			return fmt.Errorf("DB must be between 0 and 15 for standalone mode, got %d", c.DB)
		}
	}

	return nil
}

// ApplyDefaults fills in missing configuration values with recommended defaults
// This ensures the configuration is complete and ready to use
func (c *RedisConfig) ApplyDefaults() {
	if len(c.Addrs) == 0 {
		c.Addrs = []string{"localhost:6379"}
	}

	if c.PoolSize == 0 {
		c.PoolSize = 20
	}

	if c.MinIdleConns == 0 {
		// Set to 30% of PoolSize
		c.MinIdleConns = c.PoolSize * 3 / 10
		if c.MinIdleConns < 1 {
			c.MinIdleConns = 1
		}
	}

	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = 5 * time.Second
	}

	if c.ReadTimeout == 0 {
		c.ReadTimeout = 3 * time.Second
	}

	if c.WriteTimeout == 0 {
		c.WriteTimeout = 3 * time.Second
	}

	if c.MaxRedirects == 0 {
		c.MaxRedirects = 3
	}
}

// OptimizeForQPS adjusts pool settings based on expected queries per second
// This provides automatic tuning based on workload characteristics
// Recommended formula: PoolSize = QPS / 1000 (minimum 10, maximum 50)
func (c *RedisConfig) OptimizeForQPS(qps int) {
	if qps <= 0 {
		return
	}

	// Calculate optimal pool size: QPS / 1000
	optimalPoolSize := qps / 1000

	// Apply minimum and maximum bounds
	if optimalPoolSize < 10 {
		optimalPoolSize = 10
	}
	if optimalPoolSize > 50 {
		optimalPoolSize = 50
	}

	c.PoolSize = optimalPoolSize

	// Set MinIdleConns to 30% of PoolSize
	c.MinIdleConns = c.PoolSize * 3 / 10
	if c.MinIdleConns < 1 {
		c.MinIdleConns = 1
	}
}

// String returns a human-readable representation of the configuration
// Useful for logging and debugging (passwords are redacted)
func (c *RedisConfig) String() string {
	password := "<empty>"
	if c.Password != "" {
		password = "<redacted>"
	}

	mode := "standalone"
	if c.ClusterMode {
		mode = "cluster"
	}

	return fmt.Sprintf(
		"RedisConfig{Mode=%s, Addrs=%v, DB=%d, PoolSize=%d, MinIdleConns=%d, "+
			"ConnMaxLifetime=%v, DialTimeout=%v, ReadTimeout=%v, WriteTimeout=%v, "+
			"MaxRedirects=%d, Password=%s}",
		mode, c.Addrs, c.DB, c.PoolSize, c.MinIdleConns,
		c.ConnMaxLifetime, c.DialTimeout, c.ReadTimeout, c.WriteTimeout,
		c.MaxRedirects, password,
	)
}

// NewOptimizedRedisClient creates a new Redis client with optimized connection pool settings
// Supports both standalone and cluster modes based on configuration
func NewOptimizedRedisClient(cfg RedisConfig) redis.UniversalClient {
	// Apply defaults if not set
	cfg.ApplyDefaults()

	if cfg.ClusterMode {
		// Create Redis Cluster client
		return redis.NewClusterClient(&redis.ClusterOptions{
			// Connection settings
			Addrs:    cfg.Addrs,
			Password: cfg.Password,

			// Connection pool settings
			PoolSize: cfg.PoolSize,

			MinIdleConns: cfg.MinIdleConns,

			ConnMaxLifetime: cfg.ConnMaxLifetime,

			// Timeout settings
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,

			// Cluster-specific settings
			MaxRedirects: cfg.MaxRedirects,
		})
	}

	// Create standalone Redis client
	return redis.NewClient(&redis.Options{
		// Connection settings
		Addr:     cfg.Addrs[0], // Use first address for standalone
		Password: cfg.Password,
		DB:       cfg.DB,

		// Connection pool settings
		PoolSize: cfg.PoolSize,

		MinIdleConns: cfg.MinIdleConns,

		ConnMaxLifetime: cfg.ConnMaxLifetime,

		// Timeout settings
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
}

// RedisClientWithMetrics wraps a Redis client and provides connection pool metrics
// This struct enables background metric collection for monitoring pool health
type RedisClientWithMetrics struct {
	client   redis.UniversalClient
	obs      observability.Observability
	stopChan chan struct{}
}

// NewRedisClientWithMetrics creates a Redis client wrapper that exposes pool metrics
// The metrics are collected in a background goroutine every 10 seconds
func NewRedisClientWithMetrics(cfg RedisConfig, obs observability.Observability) *RedisClientWithMetrics {
	client := NewOptimizedRedisClient(cfg)

	wrapper := &RedisClientWithMetrics{
		client:   client,
		obs:      obs,
		stopChan: make(chan struct{}),
	}

	// Start background metric collection
	wrapper.ExposePoolMetrics()

	return wrapper
}

// Client returns the underlying Redis client
func (r *RedisClientWithMetrics) Client() redis.UniversalClient {
	return r.client
}

// Stop stops the background metric collection goroutine
// Should be called during graceful shutdown
func (r *RedisClientWithMetrics) Stop() {
	close(r.stopChan)
}

// ExposePoolMetrics starts a background goroutine that collects and exposes
// connection pool statistics every 10 seconds via Prometheus metrics
//
// Metrics exposed:
// - redis_pool_hits_total: Total number of times a free connection was found in the pool
// - redis_pool_misses_total: Total number of times a free connection was NOT found in the pool
// - redis_pool_timeouts_total: Total number of times a wait timeout occurred
// - redis_pool_connections{state="total"}: Total number of connections in the pool
// - redis_pool_connections{state="idle"}: Number of idle connections in the pool
// - redis_pool_connections{state="active"}: Number of active connections in use
func (r *RedisClientWithMetrics) ExposePoolMetrics() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Get pool statistics from the Redis client
				stats := r.client.PoolStats()

				// Expose pool statistics as Prometheus gauges

				// Hits: Number of times a free connection was found in the pool
				r.obs.Metrics().SetGauge("redis_pool_hits_total", float64(stats.Hits), nil)

				// Misses: Number of times a free connection was NOT found in the pool
				r.obs.Metrics().SetGauge("redis_pool_misses_total", float64(stats.Misses), nil)

				// Timeouts: Number of times a wait timeout occurred
				r.obs.Metrics().SetGauge("redis_pool_timeouts_total", float64(stats.Timeouts), nil)

				// Total connections in the pool
				r.obs.Metrics().SetGauge("redis_pool_connections", float64(stats.TotalConns), map[string]string{"state": "total"})

				// Idle connections in the pool
				r.obs.Metrics().SetGauge("redis_pool_connections", float64(stats.IdleConns), map[string]string{"state": "idle"})

				// Active connections (total - idle)
				activeConns := stats.TotalConns - stats.IdleConns
				r.obs.Metrics().SetGauge("redis_pool_connections", float64(activeConns), map[string]string{"state": "active"})

				// (active connections, idle connections, wait count, wait duration)
				// Note: StaleConns is also available but not exposed as it's less critical

			case <-r.stopChan:
				// Graceful shutdown
				return
			}
		}
	}()
}
