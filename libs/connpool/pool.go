package connpool

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

// PoolConfig holds configuration for connection pools
type PoolConfig struct {
	// Database pool configuration
	Database DatabasePoolConfig
	// Redis pool configuration
	Redis RedisPoolConfig
	// Kafka pool configuration
	Kafka KafkaPoolConfig
	// Health check configuration
	HealthCheck HealthCheckConfig
}

// DatabasePoolConfig holds database connection pool configuration
type DatabasePoolConfig struct {
	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a connection
	ConnMaxIdleTime time.Duration
	// PingTimeout is the timeout for ping operations
	PingTimeout time.Duration
}

// RedisPoolConfig holds Redis connection pool configuration
type RedisPoolConfig struct {
	// PoolSize is the maximum number of socket connections
	PoolSize int
	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int
	// MaxIdleConns is the maximum number of idle connections (deprecated in go-redis v9)
	MaxIdleConns int
	// ConnMaxLifetime is the maximum lifetime of a connection
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum idle time of a connection
	ConnMaxIdleTime time.Duration
	// PoolTimeout is the timeout for getting a connection from pool
	PoolTimeout time.Duration
	// ReadTimeout is the timeout for socket reads
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes
	WriteTimeout time.Duration
}

// KafkaPoolConfig holds Kafka producer/consumer configuration
type KafkaPoolConfig struct {
	// Producer configuration
	Producer KafkaProducerConfig
	// Consumer configuration
	Consumer KafkaConsumerConfig
}

// KafkaProducerConfig holds Kafka producer configuration
type KafkaProducerConfig struct {
	// MaxOpenRequests is the maximum number of in-flight requests
	MaxOpenRequests int
	// RequiredAcks is the level of acknowledgement reliability
	RequiredAcks sarama.RequiredAcks
	// Timeout is the timeout for produce requests
	Timeout time.Duration
	// Compression is the compression codec
	Compression sarama.CompressionCodec
	// MaxMessageBytes is the maximum message size
	MaxMessageBytes int
	// Idempotent enables idempotent producer
	Idempotent bool
	// RetryMax is the maximum number of retries
	RetryMax int
	// RetryBackoff is the backoff duration between retries
	RetryBackoff time.Duration
}

// KafkaConsumerConfig holds Kafka consumer configuration
type KafkaConsumerConfig struct {
	// SessionTimeout is the timeout for consumer session
	SessionTimeout time.Duration
	// HeartbeatInterval is the interval for heartbeat
	HeartbeatInterval time.Duration
	// RebalanceTimeout is the timeout for rebalance
	RebalanceTimeout time.Duration
	// MaxProcessingTime is the maximum time for processing a message
	MaxProcessingTime time.Duration
	// FetchMin is the minimum bytes to fetch
	FetchMin int32
	// FetchDefault is the default bytes to fetch
	FetchDefault int32
	// MaxWaitTime is the maximum wait time for fetch
	MaxWaitTime time.Duration
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	// Enabled indicates if health checks are enabled
	Enabled bool
	// Interval is the interval between health checks
	Interval time.Duration
	// Timeout is the timeout for health check operations
	Timeout time.Duration
	// FailureThreshold is the number of consecutive failures before marking unhealthy
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes before marking healthy
	SuccessThreshold int
}

// PoolManager manages all connection pools
type PoolManager struct {
	config PoolConfig

	// Database connections
	dbMu sync.RWMutex
	dbs  map[string]*DatabasePool

	// Redis connections
	redisMu      sync.RWMutex
	redisClients map[string]*RedisPool

	// Kafka connections
	kafkaMu        sync.RWMutex
	kafkaProducers map[string]*KafkaProducerPool
	kafkaConsumers map[string]*KafkaConsumerPool

	// Health checker
	healthChecker *HealthChecker

	// Cache warmers
	cacheWarmerMu sync.RWMutex
	cacheWarmers  map[string]*CacheWarmer

	// Batch processors
	batchProcessorMu sync.RWMutex
	batchProcessors  map[string]*BatchProcessor

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewPoolManager creates a new connection pool manager
func NewPoolManager(config PoolConfig) *PoolManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PoolManager{
		config:          config,
		dbs:             make(map[string]*DatabasePool),
		redisClients:    make(map[string]*RedisPool),
		kafkaProducers:  make(map[string]*KafkaProducerPool),
		kafkaConsumers:  make(map[string]*KafkaConsumerPool),
		cacheWarmers:    make(map[string]*CacheWarmer),
		batchProcessors: make(map[string]*BatchProcessor),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize health checker if enabled
	if config.HealthCheck.Enabled {
		pm.healthChecker = NewHealthChecker(config.HealthCheck)
	}

	return pm
}

// GetDatabase gets or creates a database connection pool
func (pm *PoolManager) GetDatabase(name, dsn string) (*sql.DB, error) {
	pm.dbMu.RLock()
	if pool, exists := pm.dbs[name]; exists {
		pm.dbMu.RUnlock()
		return pool.DB, nil
	}
	pm.dbMu.RUnlock()

	pm.dbMu.Lock()
	defer pm.dbMu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := pm.dbs[name]; exists {
		return pool.DB, nil
	}

	// Create new database pool
	pool, err := NewDatabasePool(name, dsn, pm.config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool %s: %w", name, err)
	}

	pm.dbs[name] = pool

	// Register health check
	if pm.healthChecker != nil {
		pm.healthChecker.RegisterDatabase(name, pool)
	}

	return pool.DB, nil
}

// GetRedis gets or creates a Redis connection pool
func (pm *PoolManager) GetRedis(name string, opts *redis.Options) (*redis.Client, error) {
	pm.redisMu.RLock()
	if pool, exists := pm.redisClients[name]; exists {
		pm.redisMu.RUnlock()
		return pool.Client, nil
	}
	pm.redisMu.RUnlock()

	pm.redisMu.Lock()
	defer pm.redisMu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := pm.redisClients[name]; exists {
		return pool.Client, nil
	}

	// Apply pool configuration to options
	if opts.PoolSize == 0 {
		opts.PoolSize = pm.config.Redis.PoolSize
	}
	if opts.MinIdleConns == 0 {
		opts.MinIdleConns = pm.config.Redis.MinIdleConns
	}
	if opts.ConnMaxLifetime == 0 {
		opts.ConnMaxLifetime = pm.config.Redis.ConnMaxLifetime
	}
	if opts.ConnMaxIdleTime == 0 {
		opts.ConnMaxIdleTime = pm.config.Redis.ConnMaxIdleTime
	}
	if opts.PoolTimeout == 0 {
		opts.PoolTimeout = pm.config.Redis.PoolTimeout
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = pm.config.Redis.ReadTimeout
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = pm.config.Redis.WriteTimeout
	}

	// Create new Redis pool
	pool, err := NewRedisPool(name, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis pool %s: %w", name, err)
	}

	pm.redisClients[name] = pool

	// Register health check
	if pm.healthChecker != nil {
		pm.healthChecker.RegisterRedis(name, pool)
	}

	return pool.Client, nil
}

// GetKafkaProducer gets or creates a Kafka producer pool
func (pm *PoolManager) GetKafkaProducer(name string, brokers []string) (sarama.SyncProducer, error) {
	pm.kafkaMu.RLock()
	if pool, exists := pm.kafkaProducers[name]; exists {
		pm.kafkaMu.RUnlock()
		return pool.Producer, nil
	}
	pm.kafkaMu.RUnlock()

	pm.kafkaMu.Lock()
	defer pm.kafkaMu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := pm.kafkaProducers[name]; exists {
		return pool.Producer, nil
	}

	// Create new Kafka producer pool
	pool, err := NewKafkaProducerPool(name, brokers, pm.config.Kafka.Producer)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer pool %s: %w", name, err)
	}

	pm.kafkaProducers[name] = pool

	// Register health check
	if pm.healthChecker != nil {
		pm.healthChecker.RegisterKafkaProducer(name, pool)
	}

	return pool.Producer, nil
}

// Start starts the pool manager and health checks
func (pm *PoolManager) Start() error {
	if pm.healthChecker != nil {
		pm.wg.Add(1)
		go func() {
			defer pm.wg.Done()
			pm.healthChecker.Start(pm.ctx)
		}()
	}

	// Start all cache warmers
	pm.cacheWarmerMu.RLock()
	for _, warmer := range pm.cacheWarmers {
		if err := warmer.Start(); err != nil {
			pm.cacheWarmerMu.RUnlock()
			return fmt.Errorf("failed to start cache warmer: %w", err)
		}
	}
	pm.cacheWarmerMu.RUnlock()

	// Start all batch processors
	pm.batchProcessorMu.RLock()
	for _, processor := range pm.batchProcessors {
		if err := processor.Start(); err != nil {
			pm.batchProcessorMu.RUnlock()
			return fmt.Errorf("failed to start batch processor: %w", err)
		}
	}
	pm.batchProcessorMu.RUnlock()

	return nil
}

// Stop stops the pool manager and closes all connections
func (pm *PoolManager) Stop() error {
	// Cancel context to stop health checks and cache warmers
	pm.cancel()

	// Stop all batch processors
	pm.batchProcessorMu.Lock()
	for name, processor := range pm.batchProcessors {
		if err := processor.Stop(); err != nil {
			fmt.Printf("Error stopping batch processor %s: %v\n", name, err)
		}
	}
	pm.batchProcessors = make(map[string]*BatchProcessor)
	pm.batchProcessorMu.Unlock()

	// Stop all cache warmers
	pm.cacheWarmerMu.Lock()
	for name, warmer := range pm.cacheWarmers {
		if err := warmer.Stop(); err != nil {
			fmt.Printf("Error stopping cache warmer %s: %v\n", name, err)
		}
	}
	pm.cacheWarmers = make(map[string]*CacheWarmer)
	pm.cacheWarmerMu.Unlock()

	// Wait for health checker to stop
	pm.wg.Wait()

	// Close all database connections
	pm.dbMu.Lock()
	for name, pool := range pm.dbs {
		if err := pool.Close(); err != nil {
			fmt.Printf("Error closing database pool %s: %v\n", name, err)
		}
	}
	pm.dbs = make(map[string]*DatabasePool)
	pm.dbMu.Unlock()

	// Close all Redis connections
	pm.redisMu.Lock()
	for name, pool := range pm.redisClients {
		if err := pool.Close(); err != nil {
			fmt.Printf("Error closing Redis pool %s: %v\n", name, err)
		}
	}
	pm.redisClients = make(map[string]*RedisPool)
	pm.redisMu.Unlock()

	// Close all Kafka producers
	pm.kafkaMu.Lock()
	for name, pool := range pm.kafkaProducers {
		if err := pool.Close(); err != nil {
			fmt.Printf("Error closing Kafka producer pool %s: %v\n", name, err)
		}
	}
	pm.kafkaProducers = make(map[string]*KafkaProducerPool)
	pm.kafkaMu.Unlock()

	return nil
}

// GetHealthStatus returns the health status of all pools
func (pm *PoolManager) GetHealthStatus() map[string]HealthStatus {
	if pm.healthChecker == nil {
		return nil
	}
	return pm.healthChecker.GetAllStatus()
}

// GetOrCreateCacheWarmer gets or creates a cache warmer for a Redis pool
func (pm *PoolManager) GetOrCreateCacheWarmer(name string, config CacheWarmerConfig) (*CacheWarmer, error) {
	pm.cacheWarmerMu.RLock()
	if warmer, exists := pm.cacheWarmers[name]; exists {
		pm.cacheWarmerMu.RUnlock()
		return warmer, nil
	}
	pm.cacheWarmerMu.RUnlock()

	pm.cacheWarmerMu.Lock()
	defer pm.cacheWarmerMu.Unlock()

	// Double-check after acquiring write lock
	if warmer, exists := pm.cacheWarmers[name]; exists {
		return warmer, nil
	}

	// Get Redis pool
	pm.redisMu.RLock()
	redisPool, exists := pm.redisClients[name]
	pm.redisMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("Redis pool %s not found", name)
	}

	// Create cache warmer
	warmer := NewCacheWarmer(redisPool.Client, config)
	pm.cacheWarmers[name] = warmer

	// Start warmer if pool manager is already started
	if pm.ctx.Err() == nil {
		if err := warmer.Start(); err != nil {
			delete(pm.cacheWarmers, name)
			return nil, fmt.Errorf("failed to start cache warmer: %w", err)
		}
	}

	return warmer, nil
}

// GetCacheWarmer gets an existing cache warmer
func (pm *PoolManager) GetCacheWarmer(name string) (*CacheWarmer, bool) {
	pm.cacheWarmerMu.RLock()
	defer pm.cacheWarmerMu.RUnlock()
	warmer, exists := pm.cacheWarmers[name]
	return warmer, exists
}

// GetAllCacheWarmerMetrics returns metrics for all cache warmers
func (pm *PoolManager) GetAllCacheWarmerMetrics() map[string]CacheWarmerMetrics {
	pm.cacheWarmerMu.RLock()
	defer pm.cacheWarmerMu.RUnlock()

	metrics := make(map[string]CacheWarmerMetrics)
	for name, warmer := range pm.cacheWarmers {
		metrics[name] = warmer.GetMetrics()
	}
	return metrics
}

// DefaultPoolConfig returns a default pool configuration optimized for cross-region
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		Database: DatabasePoolConfig{
			MaxOpenConns:    50, // Increased for cross-region load
			MaxIdleConns:    10, // Keep more idle connections
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 2 * time.Minute,
			PingTimeout:     3 * time.Second,
		},
		Redis: RedisPoolConfig{
			PoolSize:        100, // Increased for high throughput
			MinIdleConns:    20,  // Keep more idle connections
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 2 * time.Minute,
			PoolTimeout:     5 * time.Second,
			ReadTimeout:     3 * time.Second,
			WriteTimeout:    3 * time.Second,
		},
		Kafka: KafkaPoolConfig{
			Producer: KafkaProducerConfig{
				MaxOpenRequests: 5,
				RequiredAcks:    sarama.WaitForLocal, // Balance between performance and reliability
				Timeout:         10 * time.Second,
				Compression:     sarama.CompressionSnappy,
				MaxMessageBytes: 1000000, // 1MB
				Idempotent:      true,
				RetryMax:        3,
				RetryBackoff:    100 * time.Millisecond,
			},
			Consumer: KafkaConsumerConfig{
				SessionTimeout:    10 * time.Second,
				HeartbeatInterval: 3 * time.Second,
				RebalanceTimeout:  60 * time.Second,
				MaxProcessingTime: 30 * time.Second,
				FetchMin:          1,
				FetchDefault:      1024 * 1024, // 1MB
				MaxWaitTime:       500 * time.Millisecond,
			},
		},
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 2,
		},
	}
}

// GetOrCreateBatchProcessor gets or creates a batch processor
func (pm *PoolManager) GetOrCreateBatchProcessor(name string, config BatchProcessorConfig) (*BatchProcessor, error) {
	pm.batchProcessorMu.RLock()
	if processor, exists := pm.batchProcessors[name]; exists {
		pm.batchProcessorMu.RUnlock()
		return processor, nil
	}
	pm.batchProcessorMu.RUnlock()

	pm.batchProcessorMu.Lock()
	defer pm.batchProcessorMu.Unlock()

	// Double-check after acquiring write lock
	if processor, exists := pm.batchProcessors[name]; exists {
		return processor, nil
	}

	// Create batch processor
	processor := NewBatchProcessor(config)
	pm.batchProcessors[name] = processor

	// Start processor if pool manager is already started
	if pm.ctx.Err() == nil {
		if err := processor.Start(); err != nil {
			delete(pm.batchProcessors, name)
			return nil, fmt.Errorf("failed to start batch processor: %w", err)
		}
	}

	return processor, nil
}

// GetBatchProcessor gets an existing batch processor
func (pm *PoolManager) GetBatchProcessor(name string) (*BatchProcessor, bool) {
	pm.batchProcessorMu.RLock()
	defer pm.batchProcessorMu.RUnlock()
	processor, exists := pm.batchProcessors[name]
	return processor, exists
}

// GetAllBatchProcessorMetrics returns metrics for all batch processors
func (pm *PoolManager) GetAllBatchProcessorMetrics() map[string]BatchProcessorMetrics {
	pm.batchProcessorMu.RLock()
	defer pm.batchProcessorMu.RUnlock()

	metrics := make(map[string]BatchProcessorMetrics)
	for name, processor := range pm.batchProcessors {
		metrics[name] = processor.GetMetrics()
	}
	return metrics
}
