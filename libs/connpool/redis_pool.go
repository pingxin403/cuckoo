package connpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPool manages a Redis connection pool
type RedisPool struct {
	Name   string
	Client *redis.Client
	config RedisPoolConfig

	// Metrics
	mu            sync.RWMutex
	totalHits     int64
	totalMisses   int64
	totalTimeouts int64
	totalConns    int64
	idleConns     int64
	staleConns    int64
}

// NewRedisPool creates a new Redis connection pool
func NewRedisPool(name string, opts *redis.Options) (*RedisPool, error) {
	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	pool := &RedisPool{
		Name:   name,
		Client: client,
		config: RedisPoolConfig{
			PoolSize:        opts.PoolSize,
			MinIdleConns:    opts.MinIdleConns,
			ConnMaxLifetime: opts.ConnMaxLifetime,
			ConnMaxIdleTime: opts.ConnMaxIdleTime,
			PoolTimeout:     opts.PoolTimeout,
			ReadTimeout:     opts.ReadTimeout,
			WriteTimeout:    opts.WriteTimeout,
		},
	}

	return pool, nil
}

// Close closes the Redis connection pool
func (p *RedisPool) Close() error {
	return p.Client.Close()
}

// Ping checks if the Redis connection is alive
func (p *RedisPool) Ping(ctx context.Context) error {
	return p.Client.Ping(ctx).Err()
}

// Stats returns Redis pool statistics
func (p *RedisPool) Stats() *redis.PoolStats {
	stats := p.Client.PoolStats()

	// Update internal metrics
	p.mu.Lock()
	p.totalHits = int64(stats.Hits)
	p.totalMisses = int64(stats.Misses)
	p.totalTimeouts = int64(stats.Timeouts)
	p.totalConns = int64(stats.TotalConns)
	p.idleConns = int64(stats.IdleConns)
	p.staleConns = int64(stats.StaleConns)
	p.mu.Unlock()

	return stats
}

// GetMetrics returns pool metrics
func (p *RedisPool) GetMetrics() RedisPoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	hitRate := 0.0
	totalRequests := p.totalHits + p.totalMisses
	if totalRequests > 0 {
		hitRate = float64(p.totalHits) / float64(totalRequests)
	}

	return RedisPoolMetrics{
		Name:          p.Name,
		TotalHits:     p.totalHits,
		TotalMisses:   p.totalMisses,
		TotalTimeouts: p.totalTimeouts,
		TotalConns:    p.totalConns,
		IdleConns:     p.idleConns,
		StaleConns:    p.staleConns,
		HitRate:       hitRate,
	}
}

// RedisPoolMetrics holds Redis pool metrics
type RedisPoolMetrics struct {
	Name          string
	TotalHits     int64
	TotalMisses   int64
	TotalTimeouts int64
	TotalConns    int64
	IdleConns     int64
	StaleConns    int64
	HitRate       float64
}

// IsHealthy checks if the pool is healthy
func (p *RedisPool) IsHealthy(ctx context.Context) bool {
	// Check if we can ping Redis
	if err := p.Ping(ctx); err != nil {
		return false
	}

	// Check pool statistics
	stats := p.Stats()

	// Pool is unhealthy if we have too many timeouts
	if stats.Timeouts > 100 {
		return false
	}

	// Pool is unhealthy if we have too many stale connections
	if stats.StaleConns > uint32(p.config.PoolSize/2) {
		return false
	}

	return true
}

// Optimize performs runtime optimization of the pool
func (p *RedisPool) Optimize() {
	stats := p.Stats()

	// If we have high miss rate, consider increasing pool size
	hitRate := 0.0
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		hitRate = float64(stats.Hits) / float64(totalRequests)
	}

	if hitRate < 0.8 && p.config.PoolSize < 200 {
		// Increase pool size by 20%
		newPoolSize := int(float64(p.config.PoolSize) * 1.2)
		if newPoolSize > 200 {
			newPoolSize = 200
		}

		// Note: go-redis doesn't support runtime pool size changes
		// This would require recreating the client
		p.config.PoolSize = newPoolSize
	}

	// If we have many timeouts, increase pool timeout
	if stats.Timeouts > 10 && p.config.PoolTimeout < 10*time.Second {
		p.config.PoolTimeout += time.Second
		// Note: go-redis doesn't support runtime timeout changes
	}
}

// GetClient returns the underlying Redis client
func (p *RedisPool) GetClient() *redis.Client {
	return p.Client
}
