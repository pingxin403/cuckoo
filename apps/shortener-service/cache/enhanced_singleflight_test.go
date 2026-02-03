package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnhancedSingleflight(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	assert.NotNil(t, sf)
	assert.NotNil(t, sf.calls)
	assert.Equal(t, 5*time.Second, sf.timeout)
}

func TestEnhancedSingleflight_SetTimeout(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	sf.SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, sf.timeout)
}

func TestEnhancedSingleflight_Do_Success(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
		return "test-value", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "test-value", result)
}

func TestEnhancedSingleflight_Do_Error(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	expectedErr := errors.New("test error")
	result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
		return nil, expectedErr
	})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
}

func TestEnhancedSingleflight_Do_Coalescing(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	var callCount atomic.Int32
	var wg sync.WaitGroup

	// Launch 10 concurrent requests for the same key
	numGoroutines := 10
	results := make([]interface{}, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
				callCount.Add(1)
				time.Sleep(100 * time.Millisecond) // Simulate slow operation
				return "shared-result", nil
			})
			results[index] = result
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify only one function call was made
	assert.Equal(t, int32(1), callCount.Load(), "function should be called only once")

	// Verify all goroutines got the same result
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i])
		assert.Equal(t, "shared-result", results[i])
	}
}

func TestEnhancedSingleflight_Do_ErrorPropagation(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	var wg sync.WaitGroup

	// Launch 5 concurrent requests
	numGoroutines := 5
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return nil, expectedErr
			})
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all goroutines received the same error
	for i := 0; i < numGoroutines; i++ {
		assert.Equal(t, expectedErr, errors[i])
	}
}

func TestEnhancedSingleflight_Do_ContextTimeout(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return "should-not-return", nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Nil(t, result)
}

func TestEnhancedSingleflight_Do_ContextCancellation(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "should-not-return", nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Nil(t, result)
}

func TestEnhancedSingleflight_Do_WaitTimeout(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	var wg sync.WaitGroup
	wg.Add(2)

	// First goroutine: slow execution
	go func() {
		defer wg.Done()
		ctx := context.Background()
		_, _ = sf.Do(ctx, "test-key", func() (interface{}, error) {
			time.Sleep(300 * time.Millisecond)
			return "slow-result", nil
		})
	}()

	// Wait a bit to ensure first goroutine starts
	time.Sleep(50 * time.Millisecond)

	// Second goroutine: short timeout while waiting
	var secondErr error
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, secondErr = sf.Do(ctx, "test-key", func() (interface{}, error) {
			return "should-not-execute", nil
		})
	}()

	wg.Wait()

	// Second goroutine should timeout while waiting
	assert.Error(t, secondErr)
	assert.Contains(t, secondErr.Error(), "timeout")
}

func TestEnhancedSingleflight_Do_DefaultTimeout(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	sf.SetTimeout(100 * time.Millisecond)

	// Context without deadline - should use default timeout
	ctx := context.Background()

	result, err := sf.Do(ctx, "test-key", func() (interface{}, error) {
		time.Sleep(200 * time.Millisecond) // Longer than default timeout
		return "should-not-return", nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Nil(t, result)
}

func TestEnhancedSingleflight_Do_MultipleKeys(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	var callCount1, callCount2 atomic.Int32
	var wg sync.WaitGroup

	// Launch concurrent requests for two different keys
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = sf.Do(ctx, "key1", func() (interface{}, error) {
				callCount1.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "result1", nil
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = sf.Do(ctx, "key2", func() (interface{}, error) {
				callCount2.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "result2", nil
			})
		}()
	}

	wg.Wait()

	// Each key should have exactly one function call
	assert.Equal(t, int32(1), callCount1.Load())
	assert.Equal(t, int32(1), callCount2.Load())
}

func TestEnhancedSingleflight_Forget(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	// First call
	result1, err1 := sf.Do(ctx, "test-key", func() (interface{}, error) {
		return "result1", nil
	})
	require.NoError(t, err1)
	assert.Equal(t, "result1", result1)

	// Forget the key
	sf.Forget("test-key")

	// Second call should execute the function again
	var callCount atomic.Int32
	result2, err2 := sf.Do(ctx, "test-key", func() (interface{}, error) {
		callCount.Add(1)
		return "result2", nil
	})
	require.NoError(t, err2)
	assert.Equal(t, "result2", result2)
	assert.Equal(t, int32(1), callCount.Load())
}

func TestEnhancedSingleflight_Do_ConcurrentDifferentKeys(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)
	ctx := context.Background()

	var wg sync.WaitGroup
	numKeys := 100
	results := make([]interface{}, numKeys)
	errors := make([]error, numKeys)

	// Launch concurrent requests for different keys
	for i := 0; i < numKeys; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			key := string(rune('a'+index%26)) + string(rune('0'+index/26))
			result, err := sf.Do(ctx, key, func() (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return index, nil
			})
			results[index] = result
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests completed successfully
	for i := 0; i < numKeys; i++ {
		assert.NoError(t, errors[i])
		assert.Equal(t, i, results[i])
	}
}

func TestEnhancedSingleflight_Do_RetryAfterTimeout(t *testing.T) {
	obs := createTestObservability()
	sf := NewEnhancedSingleflight(obs)

	// First call with timeout
	ctx1, cancel1 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel1()

	_, err1 := sf.Do(ctx1, "test-key", func() (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return "result", nil
	})
	assert.Error(t, err1)
	assert.Contains(t, err1.Error(), "timeout")

	// Wait for first call to complete
	time.Sleep(250 * time.Millisecond)

	// Second call should succeed (retry after timeout)
	ctx2 := context.Background()
	result2, err2 := sf.Do(ctx2, "test-key", func() (interface{}, error) {
		return "retry-result", nil
	})
	assert.NoError(t, err2)
	assert.Equal(t, "retry-result", result2)
}
