package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are rejected
	StateOpen
	// StateHalfOpen means the circuit is testing if the service has recovered
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for Redis operations
// It protects against cascading failures by failing fast when Redis is unavailable
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	failureThreshold uint32        // Number of failures before opening circuit (default: 5)
	resetTimeout     time.Duration // Time to wait before attempting recovery (default: 30s)

	// State tracking
	state         CircuitState
	failureCount  uint32
	lastFailTime  time.Time
	lastStateTime time.Time

	// Observability
	obs observability.Observability
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold uint32        // Number of failures before opening (default: 5)
	ResetTimeout     time.Duration // Time before attempting recovery (default: 30s)
}

// DefaultCircuitBreakerConfig returns the default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig, obs observability.Observability) *CircuitBreaker {
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout == 0 {
		config.ResetTimeout = 30 * time.Second
	}

	cb := &CircuitBreaker{
		failureThreshold: config.FailureThreshold,
		resetTimeout:     config.ResetTimeout,
		state:            StateClosed,
		lastStateTime:    time.Now(),
		obs:              obs,
	}

	// Initialize state gauge
	cb.updateStateMetric()

	return cb
}

// Execute wraps a Redis operation with circuit breaker protection
// Returns an error if the circuit is open, otherwise executes the operation
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Check if we should allow the request
	if !cb.allowRequest() {
		cb.obs.Metrics().IncrementCounter("redis_circuit_breaker_rejected_total", nil)
		return fmt.Errorf("circuit breaker is open")
	}

	// Execute the operation
	err := operation()

	// Record the result
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// allowRequest determines if a request should be allowed based on circuit state
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Circuit is closed, allow request
		return true

	case StateOpen:
		// Check if reset timeout has elapsed
		if time.Since(cb.lastFailTime) >= cb.resetTimeout {
			cb.setState(StateHalfOpen)
			cb.obs.Logger().Info(context.Background(), "circuit breaker transitioning to half-open",
				"reset_timeout", cb.resetTimeout)
			return true
		}
		// Circuit is still open, reject request
		return false

	case StateHalfOpen:
		// In half-open state, allow one request to test recovery
		return true

	default:
		return false
	}
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Reset failure count on success
	cb.failureCount = 0

	// If we were in half-open state, transition to closed
	if cb.state == StateHalfOpen {
		cb.setState(StateClosed)
		cb.obs.Logger().Info(context.Background(), "circuit breaker closed after successful recovery")
	}
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	// Track failure in metrics
	cb.obs.Metrics().IncrementCounter("redis_circuit_breaker_failures_total", nil)

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.setState(StateOpen)
			cb.obs.Logger().Warn(context.Background(), "circuit breaker opened due to failures",
				"failure_count", cb.failureCount,
				"threshold", cb.failureThreshold)
		}

	case StateHalfOpen:
		// If test request fails in half-open state, go back to open
		cb.setState(StateOpen)
		cb.obs.Logger().Warn(context.Background(), "circuit breaker reopened after failed recovery attempt")
	}
}

// setState transitions the circuit breaker to a new state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateTime = time.Now()

	// Update state metric
	cb.updateStateMetric()

	// Track state transition
	cb.obs.Metrics().IncrementCounter("redis_circuit_breaker_state_changes_total", map[string]string{
		"from": oldState.String(),
		"to":   newState.String(),
	})
}

// updateStateMetric updates the state gauge metric
func (cb *CircuitBreaker) updateStateMetric() {
	var stateValue float64
	switch cb.state {
	case StateClosed:
		stateValue = 0
	case StateOpen:
		stateValue = 1
	case StateHalfOpen:
		stateValue = 2
	}

	cb.obs.Metrics().SetGauge("redis_circuit_breaker_state", stateValue, map[string]string{
		"state": cb.state.String(),
	})
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() uint32 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.setState(StateClosed)
	cb.obs.Logger().Info(context.Background(), "circuit breaker manually reset")
}
