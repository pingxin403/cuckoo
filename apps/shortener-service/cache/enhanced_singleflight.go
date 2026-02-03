package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// call represents a single in-flight or completed call
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// EnhancedSingleflight provides a duplicate function call suppression mechanism
// with context-based timeout support. It ensures that only one execution of a
// function is in-flight for a given key at a time, while other callers wait
// for the result or timeout.
type EnhancedSingleflight struct {
	mu      sync.Mutex
	calls   map[string]*call
	obs     observability.Observability
	timeout time.Duration // Default timeout for operations
}

// NewEnhancedSingleflight creates a new EnhancedSingleflight with default timeout
func NewEnhancedSingleflight(obs observability.Observability) *EnhancedSingleflight {
	return &EnhancedSingleflight{
		calls:   make(map[string]*call),
		obs:     obs,
		timeout: 5 * time.Second,
	}
}

// SetTimeout configures the default timeout for operations
func (sf *EnhancedSingleflight) SetTimeout(timeout time.Duration) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.timeout = timeout
}

// Do executes and returns the results of the given function, making sure that
// only one execution is in-flight for a given key at a time. If a duplicate
// comes in, the duplicate caller waits for the original to complete and receives
// the same results.
//
// The function respects the context timeout and will return an error if the
// context is cancelled or times out before the function completes.
//
func (sf *EnhancedSingleflight) Do(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error) {
	// Apply default timeout if context doesn't have one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sf.timeout)
		defer cancel()
	}

	sf.mu.Lock()
	if c, ok := sf.calls[key]; ok {
		// Another goroutine is already executing this call
		sf.mu.Unlock()

		sf.obs.Metrics().IncrementCounter("singleflight_wait_total", map[string]string{"key": key})

		sf.obs.Logger().Debug(ctx, "singleflight: coalescing request", "key", key)

		// Wait for the call to complete or context to timeout
		waitStart := time.Now()
		done := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Call completed successfully
			waitDuration := time.Since(waitStart).Seconds()
			sf.obs.Metrics().RecordHistogram("singleflight_wait_duration_seconds", waitDuration, map[string]string{"key": key})
			return c.val, c.err
		case <-ctx.Done():
			sf.obs.Metrics().IncrementCounter("singleflight_timeout_total", map[string]string{"key": key})
			return nil, fmt.Errorf("singleflight: timeout waiting for result: %w", ctx.Err())
		}
	}

	// No call in flight - create a new one
	c := &call{}
	c.wg.Add(1)
	sf.calls[key] = c
	sf.mu.Unlock()

	// Execute the function
	sf.obs.Metrics().IncrementCounter("singleflight_execute_total", map[string]string{"key": key})
	executeStart := time.Now()

	// Execute in a goroutine to respect context cancellation
	resultChan := make(chan struct{})
	go func() {
		c.val, c.err = fn()
		close(resultChan)
	}()

	// Wait for execution or context timeout
	select {
	case <-resultChan:
		// Function completed
		executeDuration := time.Since(executeStart).Seconds()
		sf.obs.Metrics().RecordHistogram("singleflight_execute_duration_seconds", executeDuration, map[string]string{"key": key})

		if c.err != nil {
			sf.obs.Metrics().IncrementCounter("singleflight_error_total", map[string]string{"key": key})
		}
	case <-ctx.Done():
		// Context cancelled/timeout during execution
		c.err = fmt.Errorf("singleflight: execution timeout: %w", ctx.Err())
		sf.obs.Metrics().IncrementCounter("singleflight_timeout_total", map[string]string{"key": key})
	}

	// Clean up and notify waiters
	c.wg.Done()
	sf.mu.Lock()
	delete(sf.calls, key)
	sf.mu.Unlock()

	return c.val, c.err
}

// Forget tells the singleflight to forget about a key. Future calls to Do for
// this key will call the function rather than waiting for an earlier call to complete.
func (sf *EnhancedSingleflight) Forget(key string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	delete(sf.calls, key)
}
