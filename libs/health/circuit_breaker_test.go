package health

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	
	if cb.name != "test-breaker" {
		t.Errorf("Expected name 'test-breaker', got '%s'", cb.name)
	}
	
	if cb.maxFailures != 3 {
		t.Errorf("Expected maxFailures 3, got %d", cb.maxFailures)
	}
	
	if cb.timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", cb.timeout)
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.GetState())
	}
	
	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected initial failure count 0, got %d", cb.GetFailureCount())
	}
}

func TestNewCircuitBreakerWithConfig(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:     5,
		Timeout:         60 * time.Second,
		HalfOpenTimeout: 15 * time.Second,
	}
	
	cb := NewCircuitBreakerWithConfig("custom-breaker", config)
	
	if cb.maxFailures != 5 {
		t.Errorf("Expected maxFailures 5, got %d", cb.maxFailures)
	}
	
	if cb.timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", cb.timeout)
	}
	
	if cb.halfOpenTimeout != 15*time.Second {
		t.Errorf("Expected halfOpenTimeout 15s, got %v", cb.halfOpenTimeout)
	}
}

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	// Execute successful function
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain Closed, got %v", cb.GetState())
	}
	
	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count 0, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_FailureTracking(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	testErr := errors.New("test error")
	
	// First failure
	err := cb.Execute(func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
	
	if cb.GetFailureCount() != 1 {
		t.Errorf("Expected failure count 1, got %d", cb.GetFailureCount())
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after 1 failure, got %v", cb.GetState())
	}
	
	// Second failure
	cb.Execute(func() error {
		return testErr
	})
	
	if cb.GetFailureCount() != 2 {
		t.Errorf("Expected failure count 2, got %d", cb.GetFailureCount())
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after 2 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	testErr := errors.New("test error")
	
	// Trigger 3 failures to open circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state Open after 3 failures, got %v", cb.GetState())
	}
	
	// Try to execute - should fail immediately without calling function
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil
	})
	
	if err == nil {
		t.Error("Expected error when circuit is open")
	}
	
	if callCount != 0 {
		t.Errorf("Expected function not to be called when circuit is open, got %d calls", callCount)
	}
	
	if err.Error() != "circuit breaker open for test" {
		t.Errorf("Expected circuit breaker error, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreakerWithConfig("test", CircuitBreakerConfig{
		MaxFailures:     3,
		Timeout:         100 * time.Millisecond, // Short timeout for testing
		HalfOpenTimeout: 50 * time.Millisecond,
	})
	
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}
	
	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.GetState())
	}
	
	// Wait for timeout to allow half-open
	time.Sleep(150 * time.Millisecond)
	
	// Next execution should try half-open
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil // Success
	})
	
	if err != nil {
		t.Errorf("Expected no error in half-open test, got %v", err)
	}
	
	if callCount != 1 {
		t.Errorf("Expected function to be called once in half-open, got %d", callCount)
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after successful half-open test, got %v", cb.GetState())
	}
	
	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count reset to 0, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreakerWithConfig("test", CircuitBreakerConfig{
		MaxFailures:     3,
		Timeout:         100 * time.Millisecond,
		HalfOpenTimeout: 50 * time.Millisecond,
	})
	
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}
	
	// Wait for timeout
	time.Sleep(150 * time.Millisecond)
	
	// Half-open test fails
	err := cb.Execute(func() error {
		return testErr
	})
	
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
	
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state Open after failed half-open test, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	testErr := errors.New("test error")
	
	// Two failures
	cb.Execute(func() error {
		return testErr
	})
	cb.Execute(func() error {
		return testErr
	})
	
	if cb.GetFailureCount() != 2 {
		t.Fatalf("Expected failure count 2, got %d", cb.GetFailureCount())
	}
	
	// Success should reset
	cb.Execute(func() error {
		return nil
	})
	
	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count reset to 0 after success, got %d", cb.GetFailureCount())
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}
	
	if cb.GetState() != StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.GetState())
	}
	
	// Manual reset
	cb.Reset()
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state Closed after reset, got %v", cb.GetState())
	}
	
	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count 0 after reset, got %d", cb.GetFailureCount())
	}
	
	// Should be able to execute normally
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
	
	if callCount != 1 {
		t.Errorf("Expected function to be called after reset, got %d calls", callCount)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("test")
	
	var wg sync.WaitGroup
	successCount := atomic.Int32{}
	errorCount := atomic.Int32{}
	
	// Execute many concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			err := cb.Execute(func() error {
				// Simulate some work
				time.Sleep(time.Millisecond)
				
				// Fail every 10th call
				if index%10 == 0 {
					return errors.New("test error")
				}
				return nil
			})
			
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}
	
	wg.Wait()
	
	total := successCount.Load() + errorCount.Load()
	if total != 100 {
		t.Errorf("Expected 100 total operations, got %d", total)
	}
	
	// Circuit breaker should be in some valid state
	state := cb.GetState()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("Invalid circuit breaker state: %v", state)
	}
}

func TestCircuitBreakerCheck(t *testing.T) {
	// Create a mock check
	mockCheck := &mockCheck{
		name:     "test-check",
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}
	
	config := CircuitBreakerConfig{
		MaxFailures:     3,
		Timeout:         30 * time.Second,
		HalfOpenTimeout: 10 * time.Second,
	}
	
	wrappedCheck := NewCircuitBreakerCheck(mockCheck, config)
	
	// Test basic properties
	if wrappedCheck.Name() != "test-check" {
		t.Errorf("Expected name 'test-check', got '%s'", wrappedCheck.Name())
	}
	
	if wrappedCheck.Timeout() != 100*time.Millisecond {
		t.Errorf("Expected timeout 100ms, got %v", wrappedCheck.Timeout())
	}
	
	if wrappedCheck.Interval() != 5*time.Second {
		t.Errorf("Expected interval 5s, got %v", wrappedCheck.Interval())
	}
	
	if !wrappedCheck.Critical() {
		t.Error("Expected critical to be true")
	}
	
	// Test successful check
	err := wrappedCheck.Check(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test circuit breaker integration
	cb := wrappedCheck.GetCircuitBreaker()
	if cb == nil {
		t.Fatal("Expected circuit breaker to be available")
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Expected circuit breaker state Closed, got %v", cb.GetState())
	}
}

func TestCircuitBreakerCheck_OpensOnFailures(t *testing.T) {
	callCount := 0
	mockCheck := &mockCheck{
		name:     "failing-check",
		timeout:  100 * time.Millisecond,
		interval: 5 * time.Second,
		critical: true,
		checkFn: func(ctx context.Context) error {
			callCount++
			return errors.New("check failed")
		},
	}
	
	config := CircuitBreakerConfig{
		MaxFailures:     3,
		Timeout:         30 * time.Second,
		HalfOpenTimeout: 10 * time.Second,
	}
	
	wrappedCheck := NewCircuitBreakerCheck(mockCheck, config)
	
	// Trigger 3 failures
	for i := 0; i < 3; i++ {
		wrappedCheck.Check(context.Background())
	}
	
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
	
	cb := wrappedCheck.GetCircuitBreaker()
	if cb.GetState() != StateOpen {
		t.Errorf("Expected circuit breaker to be Open, got %v", cb.GetState())
	}
	
	// Next check should fail fast without calling the function
	err := wrappedCheck.Check(context.Background())
	if err == nil {
		t.Error("Expected error when circuit is open")
	}
	
	if callCount != 3 {
		t.Errorf("Expected call count to remain 3, got %d", callCount)
	}
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewCircuitBreakerManager()
	
	// Register circuit breakers
	cb1 := NewCircuitBreaker("breaker-1")
	cb2 := NewCircuitBreaker("breaker-2")
	
	manager.Register(cb1)
	manager.Register(cb2)
	
	// Test Get
	retrieved, exists := manager.Get("breaker-1")
	if !exists {
		t.Error("Expected breaker-1 to exist")
	}
	if retrieved != cb1 {
		t.Error("Expected to retrieve the same circuit breaker instance")
	}
	
	// Test GetAll
	all := manager.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 circuit breakers, got %d", len(all))
	}
	
	// Test GetStats
	stats := manager.GetStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 stats entries, got %d", len(stats))
	}
	
	for name, stat := range stats {
		if stat.State != StateClosed {
			t.Errorf("Expected %s to be Closed, got %v", name, stat.State)
		}
		if stat.FailureCount != 0 {
			t.Errorf("Expected %s failure count 0, got %d", name, stat.FailureCount)
		}
	}
}

func TestCircuitBreakerManager_ResetAll(t *testing.T) {
	manager := NewCircuitBreakerManager()
	
	cb1 := NewCircuitBreaker("breaker-1")
	cb2 := NewCircuitBreaker("breaker-2")
	
	manager.Register(cb1)
	manager.Register(cb2)
	
	// Open both circuits
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb1.Execute(func() error { return testErr })
		cb2.Execute(func() error { return testErr })
	}
	
	if cb1.GetState() != StateOpen || cb2.GetState() != StateOpen {
		t.Fatal("Expected both circuits to be open")
	}
	
	// Reset all
	manager.ResetAll()
	
	if cb1.GetState() != StateClosed {
		t.Errorf("Expected breaker-1 to be Closed after reset, got %v", cb1.GetState())
	}
	if cb2.GetState() != StateClosed {
		t.Errorf("Expected breaker-2 to be Closed after reset, got %v", cb2.GetState())
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(999), "unknown"},
	}
	
	for _, tt := range tests {
		result := tt.state.String()
		if result != tt.expected {
			t.Errorf("Expected %s.String() = %s, got %s", tt.state, tt.expected, result)
		}
	}
}

// mockCheck is a test helper that implements the Check interface
type mockCheck struct {
	name     string
	timeout  time.Duration
	interval time.Duration
	critical bool
	checkFn  func(ctx context.Context) error
}

func (m *mockCheck) Name() string {
	return m.name
}

func (m *mockCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

func (m *mockCheck) Timeout() time.Duration {
	return m.timeout
}

func (m *mockCheck) Interval() time.Duration {
	return m.interval
}

func (m *mockCheck) Critical() bool {
	return m.critical
}
