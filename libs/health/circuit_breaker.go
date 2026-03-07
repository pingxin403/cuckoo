package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NewCircuitBreaker creates a new circuit breaker with the given name and default configuration.
// The circuit breaker starts in the Closed state (normal operation).
//
// Default configuration:
// - MaxFailures: 3 consecutive failures before opening
// - Timeout: 30 seconds before attempting half-open
// - HalfOpenTimeout: 10 seconds for testing recovery
//
// Example:
//
//	cb := health.NewCircuitBreaker("database-check")
//	err := cb.Execute(func() error {
//	    return db.Ping()
//	})
func NewCircuitBreaker(name string) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:            name,
		maxFailures:     3,
		timeout:         30 * time.Second,
		halfOpenTimeout: 10 * time.Second,
	}
	
	// Initialize state to Closed
	cb.state.Store(StateClosed)
	
	// Initialize failures to 0
	cb.failures.Store(0)
	
	// Initialize lastFailureTime to zero time
	cb.lastFailureTime.Store(time.Time{})
	
	return cb
}

// NewCircuitBreakerWithConfig creates a circuit breaker with custom configuration
//
// Example:
//
//	cb := health.NewCircuitBreakerWithConfig("api-check", health.CircuitBreakerConfig{
//	    MaxFailures:     5,
//	    Timeout:         60 * time.Second,
//	    HalfOpenTimeout: 15 * time.Second,
//	})
func NewCircuitBreakerWithConfig(name string, config CircuitBreakerConfig) *CircuitBreaker {
	// Apply defaults if not set
	if config.MaxFailures == 0 {
		config.MaxFailures = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenTimeout == 0 {
		config.HalfOpenTimeout = 10 * time.Second
	}
	
	cb := &CircuitBreaker{
		name:            name,
		maxFailures:     config.MaxFailures,
		timeout:         config.Timeout,
		halfOpenTimeout: config.HalfOpenTimeout,
	}
	
	// Initialize state to Closed
	cb.state.Store(StateClosed)
	
	// Initialize failures to 0
	cb.failures.Store(0)
	
	// Initialize lastFailureTime to zero time
	cb.lastFailureTime.Store(time.Time{})
	
	return cb
}

// Execute runs the given function through the circuit breaker.
// The circuit breaker will:
// - In Closed state: Execute the function normally, track failures
// - In Open state: Reject execution immediately, check if timeout elapsed for half-open
// - In HalfOpen state: Allow one test execution to check if service recovered
//
// Returns an error if:
// - The circuit is open (service is failing)
// - The function execution fails
//
// Example:
//
//	err := cb.Execute(func() error {
//	    return performHealthCheck()
//	})
//	if err != nil {
//	    log.Printf("Circuit breaker: %v", err)
//	}
func (cb *CircuitBreaker) Execute(fn func() error) error {
	state := cb.getState()
	
	switch state {
	case StateOpen:
		// Check if we should try half-open
		if cb.shouldAttemptHalfOpen() {
			cb.setState(StateHalfOpen)
			return cb.tryHalfOpen(fn)
		}
		// Circuit is open, reject immediately
		return fmt.Errorf("circuit breaker open for %s", cb.name)
		
	case StateHalfOpen:
		// Testing recovery
		return cb.tryHalfOpen(fn)
		
	case StateClosed:
		// Normal operation
		return cb.tryExecute(fn)
	}
	
	return nil
}

// tryExecute executes the function in Closed state and tracks failures
func (cb *CircuitBreaker) tryExecute(fn func() error) error {
	err := fn()
	
	if err != nil {
		// Record failure
		failures := cb.failures.Add(1)
		cb.lastFailureTime.Store(time.Now())
		
		// Check if we should open the circuit
		if failures >= int32(cb.maxFailures) {
			cb.setState(StateOpen)
		}
		
		return err
	}
	
	// Success - reset failure count
	cb.failures.Store(0)
	return nil
}

// tryHalfOpen executes the function in HalfOpen state to test recovery
func (cb *CircuitBreaker) tryHalfOpen(fn func() error) error {
	// Use mutex to ensure only one test execution in half-open state
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// Double-check state (another goroutine might have changed it)
	if cb.getState() != StateHalfOpen {
		// State changed, retry with new state
		return cb.Execute(fn)
	}
	
	err := fn()
	
	if err != nil {
		// Test failed - reopen circuit
		cb.setState(StateOpen)
		cb.lastFailureTime.Store(time.Now())
		return err
	}
	
	// Test succeeded - close circuit and reset failures
	cb.setState(StateClosed)
	cb.failures.Store(0)
	return nil
}

// shouldAttemptHalfOpen checks if enough time has elapsed to try half-open state
func (cb *CircuitBreaker) shouldAttemptHalfOpen() bool {
	lastFailure := cb.lastFailureTime.Load().(time.Time)
	
	// If lastFailureTime is zero, we haven't failed yet
	if lastFailure.IsZero() {
		return false
	}
	
	// Check if timeout has elapsed
	return time.Since(lastFailure) > cb.timeout
}

// getState returns the current circuit breaker state
func (cb *CircuitBreaker) getState() State {
	return cb.state.Load().(State)
}

// setState updates the circuit breaker state
func (cb *CircuitBreaker) setState(newState State) {
	cb.state.Store(newState)
}

// GetState returns the current state of the circuit breaker.
// This is useful for monitoring and debugging.
//
// Example:
//
//	state := cb.GetState()
//	switch state {
//	case health.StateClosed:
//	    fmt.Println("Circuit is closed (normal)")
//	case health.StateOpen:
//	    fmt.Println("Circuit is open (failing)")
//	case health.StateHalfOpen:
//	    fmt.Println("Circuit is half-open (testing)")
//	}
func (cb *CircuitBreaker) GetState() State {
	return cb.getState()
}

// GetFailureCount returns the current consecutive failure count.
// This is useful for monitoring and debugging.
func (cb *CircuitBreaker) GetFailureCount() int {
	return int(cb.failures.Load())
}

// GetName returns the name of this circuit breaker
func (cb *CircuitBreaker) GetName() string {
	return cb.name
}

// Reset manually resets the circuit breaker to Closed state with zero failures.
// This should be used carefully, typically only for testing or manual intervention.
//
// Example:
//
//	cb.Reset() // Manually reset after fixing the underlying issue
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.setState(StateClosed)
	cb.failures.Store(0)
	cb.lastFailureTime.Store(time.Time{})
}

// CircuitBreakerCheck wraps an existing health check with circuit breaker protection.
// This prevents cascading failures by opening the circuit after repeated failures.
//
// The wrapped check will:
// - Execute normally when circuit is closed
// - Fail fast when circuit is open
// - Test recovery when circuit is half-open
//
// Example:
//
//	httpCheck := health.NewHTTPCheck("api", "http://api:8080/health", true)
//	wrappedCheck := health.NewCircuitBreakerCheck(httpCheck, health.CircuitBreakerConfig{
//	    MaxFailures: 3,
//	    Timeout:     30 * time.Second,
//	})
//	hc.RegisterCheck(wrappedCheck)
type CircuitBreakerCheck struct {
	check          Check
	circuitBreaker *CircuitBreaker
}

// NewCircuitBreakerCheck creates a new health check wrapped with circuit breaker protection
func NewCircuitBreakerCheck(check Check, config CircuitBreakerConfig) *CircuitBreakerCheck {
	return &CircuitBreakerCheck{
		check:          check,
		circuitBreaker: NewCircuitBreakerWithConfig(check.Name(), config),
	}
}

// Name returns the name of the wrapped check
func (cbc *CircuitBreakerCheck) Name() string {
	return cbc.check.Name()
}

// Check executes the wrapped check through the circuit breaker
func (cbc *CircuitBreakerCheck) Check(ctx context.Context) error {
	return cbc.circuitBreaker.Execute(func() error {
		return cbc.check.Check(ctx)
	})
}

// Timeout returns the timeout of the wrapped check
func (cbc *CircuitBreakerCheck) Timeout() time.Duration {
	return cbc.check.Timeout()
}

// Interval returns the interval of the wrapped check
func (cbc *CircuitBreakerCheck) Interval() time.Duration {
	return cbc.check.Interval()
}

// Critical returns whether the wrapped check is critical
func (cbc *CircuitBreakerCheck) Critical() bool {
	return cbc.check.Critical()
}

// GetCircuitBreaker returns the underlying circuit breaker for monitoring
func (cbc *CircuitBreakerCheck) GetCircuitBreaker() *CircuitBreaker {
	return cbc.circuitBreaker
}

// CircuitBreakerManager manages multiple circuit breakers for different checks.
// It provides centralized monitoring and control of all circuit breakers.
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Register registers a circuit breaker with the manager
func (cbm *CircuitBreakerManager) Register(cb *CircuitBreaker) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()
	
	cbm.breakers[cb.name] = cb
}

// Get retrieves a circuit breaker by name
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAll returns all registered circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]*CircuitBreaker, len(cbm.breakers))
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	
	return result
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() map[string]CircuitBreakerStats {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	stats := make(map[string]CircuitBreakerStats, len(cbm.breakers))
	for name, cb := range cbm.breakers {
		stats[name] = CircuitBreakerStats{
			Name:         name,
			State:        cb.GetState(),
			FailureCount: cb.GetFailureCount(),
		}
	}
	
	return stats
}

// ResetAll resets all circuit breakers to closed state
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	
	for _, cb := range cbm.breakers {
		cb.Reset()
	}
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	Name         string
	State        State
	FailureCount int
}

// String returns a string representation of the state
func (s State) String() string {
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
