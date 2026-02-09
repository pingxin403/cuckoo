package connpool

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	// Verify database config
	assert.Equal(t, 50, config.Database.MaxOpenConns)
	assert.Equal(t, 10, config.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, config.Database.ConnMaxLifetime)

	// Verify Redis config
	assert.Equal(t, 100, config.Redis.PoolSize)
	assert.Equal(t, 20, config.Redis.MinIdleConns)
	assert.Equal(t, 5*time.Minute, config.Redis.ConnMaxLifetime)

	// Verify Kafka config
	assert.Equal(t, 5, config.Kafka.Producer.MaxOpenRequests)
	assert.Equal(t, sarama.WaitForLocal, config.Kafka.Producer.RequiredAcks)
	assert.Equal(t, sarama.CompressionSnappy, config.Kafka.Producer.Compression)
	assert.True(t, config.Kafka.Producer.Idempotent)

	// Verify health check config
	assert.True(t, config.HealthCheck.Enabled)
	assert.Equal(t, 30*time.Second, config.HealthCheck.Interval)
	assert.Equal(t, 3, config.HealthCheck.FailureThreshold)
}

func TestPoolManager_Lifecycle(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false // Disable for this test

	pm := NewPoolManager(config)
	require.NotNil(t, pm)

	// Start pool manager
	err := pm.Start()
	require.NoError(t, err)

	// Stop pool manager
	err = pm.Stop()
	require.NoError(t, err)
}

func TestPoolManager_GetDatabase_InvalidDSN(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false

	pm := NewPoolManager(config)
	defer pm.Stop()

	// Try to get database with invalid DSN
	_, err := pm.GetDatabase("test", "invalid-dsn")
	assert.Error(t, err)
}

func TestPoolManager_GetRedis_InvalidAddr(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false

	pm := NewPoolManager(config)
	defer pm.Stop()

	// Try to get Redis with invalid address
	opts := &redis.Options{
		Addr: "invalid:99999",
	}

	_, err := pm.GetRedis("test", opts)
	assert.Error(t, err)
}

func TestPoolManager_GetKafkaProducer_InvalidBrokers(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false

	pm := NewPoolManager(config)
	defer pm.Stop()

	// Try to get Kafka producer with invalid brokers
	_, err := pm.GetKafkaProducer("test", []string{"invalid:99999"})
	assert.Error(t, err)
}

func TestPoolManager_ConcurrentAccess(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false

	pm := NewPoolManager(config)
	defer pm.Stop()

	// Test concurrent access to pool manager
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			// Try to get database (will fail but shouldn't panic)
			_, _ = pm.GetDatabase("test", "invalid-dsn")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDatabasePoolConfig_Optimization(t *testing.T) {
	config := DatabasePoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PingTimeout:     3 * time.Second,
	}

	// Verify configuration values
	assert.Equal(t, 10, config.MaxOpenConns)
	assert.Equal(t, 2, config.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, config.ConnMaxLifetime)
}

func TestRedisPoolConfig_Optimization(t *testing.T) {
	config := RedisPoolConfig{
		PoolSize:        50,
		MinIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PoolTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	// Verify configuration values
	assert.Equal(t, 50, config.PoolSize)
	assert.Equal(t, 10, config.MinIdleConns)
	assert.Equal(t, 5*time.Minute, config.ConnMaxLifetime)
}

func TestKafkaProducerConfig_Optimization(t *testing.T) {
	config := KafkaProducerConfig{
		MaxOpenRequests: 5,
		RequiredAcks:    sarama.WaitForAll,
		Timeout:         10 * time.Second,
		Compression:     sarama.CompressionSnappy,
		MaxMessageBytes: 1000000,
		Idempotent:      true,
		RetryMax:        3,
		RetryBackoff:    100 * time.Millisecond,
	}

	// Verify configuration values
	assert.Equal(t, 5, config.MaxOpenRequests)
	assert.Equal(t, sarama.WaitForAll, config.RequiredAcks)
	assert.Equal(t, sarama.CompressionSnappy, config.Compression)
	assert.True(t, config.Idempotent)
}

func TestKafkaConsumerConfig_Optimization(t *testing.T) {
	config := KafkaConsumerConfig{
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
		RebalanceTimeout:  60 * time.Second,
		MaxProcessingTime: 30 * time.Second,
		FetchMin:          1,
		FetchDefault:      1024 * 1024,
		MaxWaitTime:       500 * time.Millisecond,
	}

	// Verify configuration values
	assert.Equal(t, 10*time.Second, config.SessionTimeout)
	assert.Equal(t, 3*time.Second, config.HeartbeatInterval)
	assert.Equal(t, int32(1), config.FetchMin)
}

func TestHealthCheckConfig(t *testing.T) {
	config := HealthCheckConfig{
		Enabled:          true,
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	// Verify configuration values
	assert.True(t, config.Enabled)
	assert.Equal(t, 30*time.Second, config.Interval)
	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.Equal(t, 3, config.FailureThreshold)
	assert.Equal(t, 2, config.SuccessThreshold)
}

func TestPoolManager_HealthStatus(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = true
	config.HealthCheck.Interval = 100 * time.Millisecond

	pm := NewPoolManager(config)
	defer pm.Stop()

	err := pm.Start()
	require.NoError(t, err)

	// Initially, no pools registered, so health status should be empty
	status := pm.GetHealthStatus()
	assert.Empty(t, status)
}

func TestPoolConfig_CrossRegionOptimization(t *testing.T) {
	config := DefaultPoolConfig()

	// Verify cross-region optimizations
	// Database: Higher connection limits for cross-region load
	assert.GreaterOrEqual(t, config.Database.MaxOpenConns, 50)
	assert.GreaterOrEqual(t, config.Database.MaxIdleConns, 10)

	// Redis: Higher pool size for high throughput
	assert.GreaterOrEqual(t, config.Redis.PoolSize, 100)
	assert.GreaterOrEqual(t, config.Redis.MinIdleConns, 20)

	// Kafka: Optimized for reliability and performance
	assert.Equal(t, sarama.WaitForLocal, config.Kafka.Producer.RequiredAcks)
	assert.True(t, config.Kafka.Producer.Idempotent)
	assert.Equal(t, sarama.CompressionSnappy, config.Kafka.Producer.Compression)

	// Health checks: Enabled by default
	assert.True(t, config.HealthCheck.Enabled)
	assert.Equal(t, 30*time.Second, config.HealthCheck.Interval)
}

func TestPoolManager_MultiplePoolsManagement(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = false

	pm := NewPoolManager(config)
	defer pm.Stop()

	// Verify internal maps are initialized
	assert.NotNil(t, pm.dbs)
	assert.NotNil(t, pm.redisClients)
	assert.NotNil(t, pm.kafkaProducers)
	assert.NotNil(t, pm.kafkaConsumers)

	// Verify all maps are empty initially
	assert.Empty(t, pm.dbs)
	assert.Empty(t, pm.redisClients)
	assert.Empty(t, pm.kafkaProducers)
	assert.Empty(t, pm.kafkaConsumers)
}

func TestPoolManager_ContextCancellation(t *testing.T) {
	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = true
	config.HealthCheck.Interval = 100 * time.Millisecond

	pm := NewPoolManager(config)

	err := pm.Start()
	require.NoError(t, err)

	// Stop should cancel context and wait for goroutines
	err = pm.Stop()
	require.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-pm.ctx.Done():
		// Context is cancelled as expected
	default:
		t.Error("Context should be cancelled after Stop()")
	}
}
