package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config, obs)

	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, uint32(0), cb.GetFailureCount())
	assert.Equal(t, uint32(5), cb.failureThreshold)
	assert.Equal(t, 30*time.Second, cb.resetTimeout)
}

func TestNewCircuitBreaker_CustomConfig(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     10 * time.Second,
	}
	cb := NewCircuitBreaker(config, obs)

	assert.Equal(t, uint32(3), cb.failureThreshold)
	assert.Equal(t, 10*time.Second, cb.resetTimeout)
}

func TestNewCircuitBreaker_DefaultsOnZero(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{} // Zero values
	cb := NewCircuitBreaker(config, obs)

	assert.Equal(t, uint32(5), cb.failureThreshold)
	assert.Equal(t, 30*time.Second, cb.resetTimeout)
}

// TestCircuitBreaker_ClosedToOpen tests the transition from Closed to Open state
func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config, obs)

	// Circuit should start closed
	assert.Equal(t, StateClosed, cb.GetState())

	// Simulate 4 failures - should stay closed
	for i := 0; i < 4; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateClosed, cb.GetState())
	}
	assert.Equal(t, uint32(4), cb.GetFailureCount())

	// 5th failure should open the circuit
	err := cb.Execute(context.Background(), func() error {
		return errors.New("redis error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())
	assert.Equal(t, uint32(5), cb.GetFailureCount())
}

// TestCircuitBreaker_OpenRejectsRequests tests that open circuit rejects requests
func TestCircuitBreaker_OpenRejectsRequests(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     1 * time.Second,
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit with 2 failures
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Subsequent requests should be rejected immediately
	callCount := 0
	err := cb.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")
	assert.Equal(t, 0, callCount, "operation should not be executed when circuit is open")
}

// TestCircuitBreaker_OpenToHalfOpen tests the transition from Open to Half-Open
func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond, // Short timeout for testing
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Next request should transition to half-open
	callCount := 0
	err := cb.Execute(context.Background(), func() error {
		callCount++
		return errors.New("still failing")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount, "operation should be executed in half-open state")
	// After failure in half-open, should go back to open
	assert.Equal(t, StateOpen, cb.GetState())
}

// TestCircuitBreaker_HalfOpenToClose tests successful recovery
func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Successful request in half-open should close the circuit
	callCount := 0
	err := cb.Execute(context.Background(), func() error {
		callCount++
		return nil // Success!
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, uint32(0), cb.GetFailureCount(), "failure count should reset on success")
}

// TestCircuitBreaker_HalfOpenToOpen tests failed recovery
func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Failed request in half-open should reopen the circuit
	err := cb.Execute(context.Background(), func() error {
		return errors.New("still failing")
	})

	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())
}

// TestCircuitBreaker_SuccessResetsFailureCount tests that success resets failure count
func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config, obs)

	// Simulate 3 failures
	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, uint32(3), cb.GetFailureCount())
	assert.Equal(t, StateClosed, cb.GetState())

	// One success should reset failure count
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), cb.GetFailureCount())
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreaker_ConcurrentRequests tests thread safety
func TestCircuitBreaker_ConcurrentRequests(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 10,
		ResetTimeout:     1 * time.Second,
	}
	cb := NewCircuitBreaker(config, obs)

	var wg sync.WaitGroup
	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	// Run 50 concurrent requests (mix of success and failure)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			err := cb.Execute(context.Background(), func() error {
				if idx%3 == 0 {
					return errors.New("redis error")
				}
				return nil
			})

			mu.Lock()
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Should have both successes and errors
	assert.Greater(t, successCount, 0)
	assert.Greater(t, errorCount, 0)
	// Circuit should still be closed (not enough consecutive failures)
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreaker_Reset tests manual reset functionality
func TestCircuitBreaker_Reset(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     1 * time.Hour, // Long timeout
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	assert.Equal(t, StateOpen, cb.GetState())
	assert.Equal(t, uint32(2), cb.GetFailureCount())

	// Manual reset
	cb.Reset()

	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, uint32(0), cb.GetFailureCount())

	// Should accept requests now
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
}

// TestCircuitBreaker_StateString tests state string representation
func TestCircuitBreaker_StateString(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", CircuitState(99).String())
}

// TestCircuitBreaker_MultipleRecoveryAttempts tests multiple recovery cycles
func TestCircuitBreaker_MultipleRecoveryAttempts(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config, obs)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errors.New("redis error")
		})
	}
	require.Equal(t, StateOpen, cb.GetState())

	// First recovery attempt - fails
	time.Sleep(150 * time.Millisecond)
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("still failing")
	})
	assert.Equal(t, StateOpen, cb.GetState())

	// Second recovery attempt - succeeds
	time.Sleep(150 * time.Millisecond)
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

// TestCircuitBreaker_ContextPropagation tests that context is properly used
func TestCircuitBreaker_ContextPropagation(t *testing.T) {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	config := DefaultCircuitBreakerConfig()
	cb := NewCircuitBreaker(config, obs)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")

	executed := false
	err := cb.Execute(ctx, func() error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
}
