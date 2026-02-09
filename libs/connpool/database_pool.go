package connpool

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DatabasePool manages a database connection pool
type DatabasePool struct {
	Name   string
	DB     *sql.DB
	config DatabasePoolConfig

	// Metrics
	mu                sync.RWMutex
	totalConnections  int64
	activeConnections int64
	idleConnections   int64
	waitCount         int64
	waitDuration      time.Duration
	maxIdleClosed     int64
	maxLifetimeClosed int64
}

// NewDatabasePool creates a new database connection pool
func NewDatabasePool(name, dsn string, config DatabasePoolConfig) (*DatabasePool, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply pool configuration
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.PingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &DatabasePool{
		Name:   name,
		DB:     db,
		config: config,
	}

	return pool, nil
}

// Close closes the database connection pool
func (p *DatabasePool) Close() error {
	return p.DB.Close()
}

// Ping checks if the database connection is alive
func (p *DatabasePool) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, p.config.PingTimeout)
	defer cancel()

	return p.DB.PingContext(pingCtx)
}

// Stats returns database pool statistics
func (p *DatabasePool) Stats() sql.DBStats {
	stats := p.DB.Stats()

	// Update internal metrics
	p.mu.Lock()
	p.totalConnections = int64(stats.OpenConnections)
	p.activeConnections = int64(stats.InUse)
	p.idleConnections = int64(stats.Idle)
	p.waitCount = stats.WaitCount
	p.waitDuration = stats.WaitDuration
	p.maxIdleClosed = stats.MaxIdleClosed
	p.maxLifetimeClosed = stats.MaxLifetimeClosed
	p.mu.Unlock()

	return stats
}

// GetMetrics returns pool metrics
func (p *DatabasePool) GetMetrics() DatabasePoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgWaitTime := time.Duration(0)
	if p.waitCount > 0 {
		avgWaitTime = p.waitDuration / time.Duration(p.waitCount)
	}

	return DatabasePoolMetrics{
		Name:              p.Name,
		TotalConnections:  p.totalConnections,
		ActiveConnections: p.activeConnections,
		IdleConnections:   p.idleConnections,
		WaitCount:         p.waitCount,
		WaitDuration:      p.waitDuration,
		AvgWaitTime:       avgWaitTime,
		MaxIdleClosed:     p.maxIdleClosed,
		MaxLifetimeClosed: p.maxLifetimeClosed,
	}
}

// DatabasePoolMetrics holds database pool metrics
type DatabasePoolMetrics struct {
	Name              string
	TotalConnections  int64
	ActiveConnections int64
	IdleConnections   int64
	WaitCount         int64
	WaitDuration      time.Duration
	AvgWaitTime       time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
}

// IsHealthy checks if the pool is healthy
func (p *DatabasePool) IsHealthy(ctx context.Context) bool {
	// Check if we can ping the database
	if err := p.Ping(ctx); err != nil {
		return false
	}

	// Check pool statistics
	stats := p.Stats()

	// Pool is unhealthy if all connections are in use and we're waiting
	if stats.InUse >= stats.MaxOpenConnections && stats.WaitCount > 0 {
		return false
	}

	return true
}

// Optimize performs runtime optimization of the pool
func (p *DatabasePool) Optimize() {
	stats := p.Stats()

	// If we have too many idle connections being closed, increase MaxIdleConns
	if stats.MaxIdleClosed > 100 && p.config.MaxIdleConns < p.config.MaxOpenConns/2 {
		newMaxIdle := p.config.MaxIdleConns + 5
		if newMaxIdle > p.config.MaxOpenConns/2 {
			newMaxIdle = p.config.MaxOpenConns / 2
		}
		p.DB.SetMaxIdleConns(newMaxIdle)
		p.config.MaxIdleConns = newMaxIdle
	}

	// If we have high wait times, consider increasing MaxOpenConns
	avgWaitTime := time.Duration(0)
	if stats.WaitCount > 0 {
		avgWaitTime = stats.WaitDuration / time.Duration(stats.WaitCount)
	}

	if avgWaitTime > 100*time.Millisecond && p.config.MaxOpenConns < 100 {
		newMaxOpen := p.config.MaxOpenConns + 10
		if newMaxOpen > 100 {
			newMaxOpen = 100
		}
		p.DB.SetMaxOpenConns(newMaxOpen)
		p.config.MaxOpenConns = newMaxOpen
	}
}
