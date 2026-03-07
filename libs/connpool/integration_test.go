package connpool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabasePool_Integration tests database pool with real MySQL connection
func TestDatabasePool_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to connect to MySQL
	dsn := "root:password@tcp(localhost:3306)/im_chat?parseTime=true"
	config := DatabasePoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PingTimeout:     3 * time.Second,
	}

	pool, err := NewDatabasePool("test-db", dsn, config)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to MySQL: %v", err)
		return
	}
	defer pool.Close()

	// Test ping
	ctx := context.Background()
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Test stats
	stats := pool.Stats()
	assert.GreaterOrEqual(t, stats.OpenConnections, 0)

	// Test metrics
	metrics := pool.GetMetrics()
	assert.Equal(t, "test-db", metrics.Name)
	assert.GreaterOrEqual(t, metrics.TotalConnections, int64(0))

	// Test health check
	healthy := pool.IsHealthy(ctx)
	assert.True(t, healthy)
}

// TestRedisPool_Integration tests Redis pool with real Redis connection
func TestRedisPool_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to connect to Redis
	opts := &redis.Options{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PoolTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	pool, err := NewRedisPool("test-redis", opts)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to Redis: %v", err)
		return
	}
	defer pool.Close()

	// Test ping
	ctx := context.Background()
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Test stats
	stats := pool.Stats()
	assert.NotNil(t, stats)

	// Test metrics
	metrics := pool.GetMetrics()
	assert.Equal(t, "test-redis", metrics.Name)

	// Test health check
	healthy := pool.IsHealthy(ctx)
	assert.True(t, healthy)

	// Test basic operations
	err = pool.Client.Set(ctx, "test-key", "test-value", 10*time.Second).Err()
	require.NoError(t, err)

	val, err := pool.Client.Get(ctx, "test-key").Result()
	require.NoError(t, err)
	assert.Equal(t, "test-value", val)
}

// TestPoolManager_Integration tests pool manager with real connections
func TestPoolManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultPoolConfig()
	config.HealthCheck.Enabled = true
	config.HealthCheck.Interval = 5 * time.Second

	pm := NewPoolManager(config)
	defer pm.Stop()

	err := pm.Start()
	require.NoError(t, err)

	// Test database pool
	mysqlDSN := "root:password@tcp(localhost:3306)/im_chat?parseTime=true"
	db, err := pm.GetDatabase("mysql-primary", mysqlDSN)
	if err != nil {
		t.Logf("Cannot connect to MySQL: %v", err)
	} else {
		assert.NotNil(t, db)

		// Test query
		var version string
		err = db.QueryRow("SELECT VERSION()").Scan(&version)
		require.NoError(t, err)
		t.Logf("MySQL version: %s", version)
	}

	// Test Redis pool
	redisOpts := &redis.Options{
		Addr: "localhost:6379",
	}
	redisClient, err := pm.GetRedis("redis-primary", redisOpts)
	if err != nil {
		t.Logf("Cannot connect to Redis: %v", err)
	} else {
		assert.NotNil(t, redisClient)

		// Test ping
		ctx := context.Background()
		err = redisClient.Ping(ctx).Err()
		require.NoError(t, err)
	}

	// Wait for health checks to run
	time.Sleep(6 * time.Second)

	// Check health status
	status := pm.GetHealthStatus()
	t.Logf("Health status: %+v", status)
}

// TestDatabasePool_ConnectionReuse tests connection reuse
func TestDatabasePool_ConnectionReuse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "root:password@tcp(localhost:3306)/im_chat?parseTime=true"
	config := DatabasePoolConfig{
		MaxOpenConns:    5,
		MaxIdleConns:    3,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PingTimeout:     3 * time.Second,
	}

	pool, err := NewDatabasePool("test-reuse", dsn, config)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to MySQL: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	// Execute multiple queries to test connection reuse
	for i := 0; i < 10; i++ {
		var result int
		err := pool.DB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	}

	// Check stats
	stats := pool.Stats()
	t.Logf("Stats after 10 queries: OpenConnections=%d, InUse=%d, Idle=%d, WaitCount=%d",
		stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)

	// Verify connection reuse (should not create 10 connections)
	assert.LessOrEqual(t, stats.OpenConnections, 5)
}

// TestRedisPool_ConnectionReuse tests Redis connection reuse
func TestRedisPool_ConnectionReuse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	opts := &redis.Options{
		Addr:         "localhost:6379",
		PoolSize:     5,
		MinIdleConns: 3,
	}

	pool, err := NewRedisPool("test-reuse", opts)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to Redis: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	// Execute multiple commands to test connection reuse
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		err := pool.Client.Set(ctx, key, i, 10*time.Second).Err()
		require.NoError(t, err)

		val, err := pool.Client.Get(ctx, key).Int()
		require.NoError(t, err)
		assert.Equal(t, i, val)
	}

	// Check stats
	stats := pool.Stats()
	t.Logf("Stats after 20 operations: TotalConns=%d, IdleConns=%d, Hits=%d, Misses=%d",
		stats.TotalConns, stats.IdleConns, stats.Hits, stats.Misses)

	// Verify connection reuse
	assert.LessOrEqual(t, int(stats.TotalConns), 5)
}

// TestHealthChecker_Integration tests health checker with real connections
func TestHealthChecker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := HealthCheckConfig{
		Enabled:          true,
		Interval:         2 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	hc := NewHealthChecker(config)

	// Register database pool
	mysqlDSN := "root:password@tcp(localhost:3306)/im_chat?parseTime=true"
	dbConfig := DatabasePoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PingTimeout:     3 * time.Second,
	}

	dbPool, err := NewDatabasePool("test-db", mysqlDSN, dbConfig)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to MySQL: %v", err)
		return
	}
	defer dbPool.Close()

	hc.RegisterDatabase("test-db", dbPool)

	// Register Redis pool
	redisOpts := &redis.Options{
		Addr: "localhost:6379",
	}

	redisPool, err := NewRedisPool("test-redis", redisOpts)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to Redis: %v", err)
		return
	}
	defer redisPool.Close()

	hc.RegisterRedis("test-redis", redisPool)

	// Start health checker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hc.Start(ctx)

	// Wait for health checks to run
	time.Sleep(3 * time.Second)

	// Check health status
	summary := hc.GetHealthSummary()
	t.Logf("Health summary: %s", summary.String())

	assert.Equal(t, 2, summary.TotalPools)
	assert.True(t, summary.OverallHealthy)

	// Get individual status
	dbStatus := hc.GetStatus("test-db")
	require.NotNil(t, dbStatus)
	assert.True(t, dbStatus.Healthy)
	assert.Equal(t, "database", dbStatus.Type)

	redisStatus := hc.GetStatus("test-redis")
	require.NotNil(t, redisStatus)
	assert.True(t, redisStatus.Healthy)
	assert.Equal(t, "redis", redisStatus.Type)
}

// TestDatabasePool_ConcurrentAccess tests concurrent access to database pool
func TestDatabasePool_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "root:password@tcp(localhost:3306)/im_chat?parseTime=true"
	config := DatabasePoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		PingTimeout:     3 * time.Second,
	}

	pool, err := NewDatabasePool("test-concurrent", dsn, config)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to MySQL: %v", err)
		return
	}
	defer pool.Close()

	ctx := context.Background()

	// Run concurrent queries
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			var result int
			err := pool.DB.QueryRowContext(ctx, "SELECT ?", id).Scan(&result)
			assert.NoError(t, err)
			assert.Equal(t, id, result)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Check stats
	stats := pool.Stats()
	t.Logf("Stats after concurrent access: OpenConnections=%d, WaitCount=%d, WaitDuration=%v",
		stats.OpenConnections, stats.WaitCount, stats.WaitDuration)
}
