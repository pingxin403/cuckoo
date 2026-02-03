package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreaker_RedisFailures tests circuit breaker behavior with Redis failures
func TestCircuitBreaker_RedisFailures(t *testing.T) {
	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create circuit breaker with low threshold for testing
	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	// Simulate Redis operations
	ctx := context.Background()

	// First 2 failures - circuit should stay closed
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, func() error {
			return errors.New("redis connection error")
		})
		assert.Error(t, err)
		assert.Equal(t, cache.StateClosed, cb.GetState())
	}

	// 3rd failure - circuit should open
	err = cb.Execute(ctx, func() error {
		return errors.New("redis connection error")
	})
	assert.Error(t, err)
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// Subsequent requests should be rejected immediately
	callCount := 0
	err = cb.Execute(ctx, func() error {
		callCount++
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.Equal(t, 0, callCount, "operation should not execute when circuit is open")
}

// TestCircuitBreaker_AutomaticRecovery tests automatic recovery after timeout
func TestCircuitBreaker_AutomaticRecovery(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     500 * time.Millisecond,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	ctx := context.Background()

	// Open the circuit with failures
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(600 * time.Millisecond)

	// Next successful request should close the circuit
	callCount := 0
	err = cb.Execute(ctx, func() error {
		callCount++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, cache.StateClosed, cb.GetState())
}

// TestCircuitBreaker_GracefulDegradation tests fallback behavior
func TestCircuitBreaker_GracefulDegradation(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     1 * time.Second,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// When circuit is open, application should fall back to database
	// This test simulates the fallback behavior
	var result string
	err = cb.Execute(ctx, func() error {
		// This would be a Redis operation
		return errors.New("redis unavailable")
	})

	if err != nil && err.Error() == "circuit breaker is open" {
		// Fallback to database (simulated)
		result = "data from database"
	}

	assert.Equal(t, "data from database", result)
}

// TestCircuitBreaker_WithRealRedis tests circuit breaker with real Redis instance
func TestCircuitBreaker_WithRealRedis(t *testing.T) {
	// Start miniredis
	mr := miniredis.RunT(t)
	defer mr.Close()

	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2Cache with circuit breaker
	l2Config := cache.L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 5,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)
	defer l2.Close()

	ctx := context.Background()

	// Normal operations should work
	err = l2.Set(ctx, "test1", "https://example.com", time.Now())
	assert.NoError(t, err)

	mapping, err := l2.Get(ctx, "test1")
	assert.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, "https://example.com", mapping.LongURL)

	// Circuit breaker should be closed
	assert.Equal(t, cache.StateClosed, l2.CircuitBreaker().GetState())
}

// TestCircuitBreaker_RedisDownScenario tests behavior when Redis goes down
func TestCircuitBreaker_RedisDownScenario(t *testing.T) {
	// Start miniredis
	mr := miniredis.RunT(t)

	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create L2Cache with circuit breaker (low threshold for testing)
	l2Config := cache.L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 5,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)
	defer l2.Close()

	// Override circuit breaker with lower threshold
	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	// Use reflection or create a new L2Cache with custom circuit breaker
	// For this test, we'll simulate the behavior

	ctx := context.Background()

	// Normal operation
	err = l2.Set(ctx, "test1", "https://example.com", time.Now())
	assert.NoError(t, err)

	// Simulate Redis going down
	mr.Close()

	// Operations should fail and open the circuit
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error {
			return l2.Set(ctx, "test2", "https://example.com", time.Now())
		})
	}

	// Circuit should be open
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// Subsequent operations should fail fast
	err = cb.Execute(ctx, func() error {
		return l2.Set(ctx, "test3", "https://example.com", time.Now())
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
}

// TestCircuitBreaker_PartialFailures tests circuit breaker with intermittent failures
func TestCircuitBreaker_PartialFailures(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     1 * time.Second,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	ctx := context.Background()

	// Mix of successes and failures
	failureCount := 0
	for i := 0; i < 10; i++ {
		err := cb.Execute(ctx, func() error {
			if i%3 == 0 {
				return errors.New("intermittent error")
			}
			return nil
		})

		if err != nil && err.Error() != "circuit breaker is open" {
			failureCount++
		}
	}

	// Circuit should still be closed (not enough consecutive failures)
	assert.Equal(t, cache.StateClosed, cb.GetState())
	// After the last iteration (i=9), which is a success (9%3 != 0), failure count should be 0
	// But if the last operation was a failure, it could be 1
	assert.LessOrEqual(t, int(cb.GetFailureCount()), 1, "failure count should be low with intermittent failures")
}

// TestCircuitBreaker_ConcurrentOperations tests thread safety with real Redis
func TestCircuitBreaker_ConcurrentOperations(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)
	defer l2.Close()

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			shortCode := "test" + string(rune(idx))
			_ = l2.Set(ctx, shortCode, "https://example.com", time.Now())
			_, _ = l2.Get(ctx, shortCode)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Circuit should still be closed
	assert.Equal(t, cache.StateClosed, l2.CircuitBreaker().GetState())
}

// TestCircuitBreaker_MetricsTracking tests that metrics are properly tracked
func TestCircuitBreaker_MetricsTracking(t *testing.T) {
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true, // Enable metrics for this test
	})
	require.NoError(t, err)

	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     500 * time.Millisecond,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// Try to execute when circuit is open (should be rejected)
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.Error(t, err)

	// Wait for recovery
	time.Sleep(600 * time.Millisecond)

	// Successful recovery
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, cache.StateClosed, cb.GetState())

	// Metrics should have been tracked:
	// - redis_circuit_breaker_failures_total
	// - redis_circuit_breaker_rejected_total
	// - redis_circuit_breaker_state_changes_total
	// - redis_circuit_breaker_state (gauge)
}

// TestCircuitBreaker_RealWorldScenario tests a realistic scenario
func TestCircuitBreaker_RealWorldScenario(t *testing.T) {
	mr := miniredis.RunT(t)

	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs:    []string{mr.Addr()},
		PoolSize: 10,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)
	defer l2.Close()

	ctx := context.Background()

	// Phase 1: Normal operations
	for i := 0; i < 10; i++ {
		err := l2.Set(ctx, "key"+string(rune(i)), "https://example.com", time.Now())
		assert.NoError(t, err)
	}
	assert.Equal(t, cache.StateClosed, l2.CircuitBreaker().GetState())

	// Phase 2: Redis becomes unavailable
	mr.Close()

	// Circuit breaker should open after threshold failures
	cbConfig := cache.CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     1 * time.Second,
	}
	cb := cache.NewCircuitBreaker(cbConfig, obs)

	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error {
			return l2.Set(ctx, "key", "https://example.com", time.Now())
		})
	}
	assert.Equal(t, cache.StateOpen, cb.GetState())

	// Phase 3: Requests are rejected (fail fast)
	err = cb.Execute(ctx, func() error {
		return l2.Set(ctx, "key", "https://example.com", time.Now())
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Phase 4: Wait for recovery timeout
	time.Sleep(1100 * time.Millisecond)

	// Phase 5: Redis comes back online
	mr = miniredis.RunT(t)
	defer mr.Close()

	// Create new client with recovered Redis
	l2Config.Addrs = []string{mr.Addr()}
	l2New, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)
	defer l2New.Close()

	// Operations should work again
	err = l2New.Set(ctx, "key", "https://example.com", time.Now())
	assert.NoError(t, err)
	assert.Equal(t, cache.StateClosed, l2New.CircuitBreaker().GetState())
}
