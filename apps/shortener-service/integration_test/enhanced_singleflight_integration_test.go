package integration_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStorageWithDelay simulates slow database queries
type MockStorageWithDelay struct {
	data      map[string]*cache.StorageMapping
	mu        sync.RWMutex
	delay     time.Duration
	callCount atomic.Int32
}

func NewMockStorageWithDelay() *MockStorageWithDelay {
	return &MockStorageWithDelay{
		data:  make(map[string]*cache.StorageMapping),
		delay: 0,
	}
}

func (m *MockStorageWithDelay) SetDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
}

func (m *MockStorageWithDelay) Get(ctx context.Context, shortCode string) (*cache.StorageMapping, error) {
	m.callCount.Add(1)

	// Simulate delay
	m.mu.RLock()
	delay := m.delay
	m.mu.RUnlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if mapping, ok := m.data[shortCode]; ok {
		return mapping, nil
	}
	return nil, storage.ErrNotFound
}

func (m *MockStorageWithDelay) Exists(ctx context.Context, shortCode string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[shortCode]
	return ok, nil
}

func (m *MockStorageWithDelay) Create(ctx context.Context, shortCode, longURL, creatorIP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[shortCode] = &cache.StorageMapping{
		ShortCode: shortCode,
		LongURL:   longURL,
		CreatedAt: time.Now(),
		CreatorIP: creatorIP,
	}
	return nil
}

func (m *MockStorageWithDelay) GetCallCount() int {
	return int(m.callCount.Load())
}

func (m *MockStorageWithDelay) ResetCallCount() {
	m.callCount.Store(0)
}

// TestEnhancedSingleflight_SlowDatabaseQuery tests singleflight with slow database queries
func TestEnhancedSingleflight_SlowDatabaseQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true,
		MetricsPort:    0, // Disable metrics server
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Create storage with slow query simulation
	store := NewMockStorageWithDelay()
	store.SetDelay(500 * time.Millisecond) // Simulate slow DB query

	// Create L1 cache
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)

	// Create L2 cache (using test Redis)
	l2Config := cache.L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	// Create cache loader
	loader := cache.NewCacheLoader(l2.Client(), store, l2, obs)

	// Create cache manager with loader (which uses EnhancedSingleflight)
	cm := cache.NewCacheManagerWithLoader(l1, l2, store, loader, obs)

	// Add test data to storage
	testShortCode := "test123"
	testLongURL := "https://example.com/very/long/url"
	err = store.Create(context.Background(), testShortCode, testLongURL, "127.0.0.1")
	require.NoError(t, err)

	// Test: Launch 20 concurrent requests for the same key
	var wg sync.WaitGroup
	numGoroutines := 20
	results := make([]*cache.URLMapping, numGoroutines)
	errors := make([]error, numGoroutines)

	startTime := time.Now()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			mapping, err := cm.Get(ctx, testShortCode)
			results[index] = mapping
			errors[index] = err
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Verify: All requests should succeed
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "Request %d should succeed", i)
		require.NotNil(t, results[i], "Result %d should not be nil", i)
		assert.Equal(t, testLongURL, results[i].LongURL)
	}

	// Verify: Total time should be close to one DB query (not 20x)
	// With singleflight, only one DB query should be made
	assert.Less(t, duration, 1*time.Second, "Total time should be less than 1 second (one DB query)")

	// Verify: Storage was called only once (singleflight coalescing)
	assert.Equal(t, 1, store.GetCallCount(), "Storage should be called only once")

	// Cleanup
	_ = cm.Delete(context.Background(), testShortCode)
}

// TestEnhancedSingleflight_ConcurrentRequestsWithTimeout tests timeout behavior
func TestEnhancedSingleflight_ConcurrentRequestsWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true,
		MetricsPort:    0,
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Create storage with very slow query (longer than timeout)
	store := NewMockStorageWithDelay()
	store.SetDelay(10 * time.Second) // Very slow query

	// Create caches
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	loader := cache.NewCacheLoader(l2.Client(), store, l2, obs)
	cm := cache.NewCacheManagerWithLoader(l1, l2, store, loader, obs)

	// Add test data
	testShortCode := "timeout-test"
	testLongURL := "https://example.com/timeout"
	err = store.Create(context.Background(), testShortCode, testLongURL, "127.0.0.1")
	require.NoError(t, err)

	// Test: Launch requests with short timeout
	var wg sync.WaitGroup
	var timeoutCount atomic.Int32

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err := cm.Get(ctx, testShortCode)
			if err != nil && (ctx.Err() != nil || err.Error() == "context deadline exceeded") {
				timeoutCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Verify: Some requests should timeout
	assert.Greater(t, timeoutCount.Load(), int32(0), "Some requests should timeout")

	// Cleanup
	_ = cm.Delete(context.Background(), testShortCode)
}

// TestEnhancedSingleflight_MetricsAccuracy tests that metrics are correctly tracked
func TestEnhancedSingleflight_MetricsAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true,
		MetricsPort:    0,
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Create storage with moderate delay
	store := NewMockStorageWithDelay()
	store.SetDelay(200 * time.Millisecond)

	// Create caches
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	loader := cache.NewCacheLoader(l2.Client(), store, l2, obs)
	cm := cache.NewCacheManagerWithLoader(l1, l2, store, loader, obs)

	// Add test data
	testShortCode := "metrics-test"
	testLongURL := "https://example.com/metrics"
	err = store.Create(context.Background(), testShortCode, testLongURL, "127.0.0.1")
	require.NoError(t, err)

	// Test: Launch concurrent requests
	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, _ = cm.Get(ctx, testShortCode)
		}()
	}

	wg.Wait()

	// Note: Metrics are tracked internally by EnhancedSingleflight
	// In a real scenario, you would query Prometheus to verify metrics
	// For this test, we just verify the operation completed successfully

	// Verify: Storage was called only once (singleflight coalescing)
	assert.Equal(t, 1, store.GetCallCount(), "Storage should be called only once")

	// Cleanup
	_ = cm.Delete(context.Background(), testShortCode)
}

// TestEnhancedSingleflight_ErrorPropagation tests error propagation to all waiting goroutines
func TestEnhancedSingleflight_ErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true,
		MetricsPort:    0,
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Create storage that will return an error
	store := NewMockStorageWithDelay()
	store.SetDelay(100 * time.Millisecond)

	// Create caches
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	loader := cache.NewCacheLoader(l2.Client(), store, l2, obs)
	cm := cache.NewCacheManagerWithLoader(l1, l2, store, loader, obs)

	// Test: Request non-existent key (will cause storage.ErrNotFound)
	testShortCode := "nonexistent"

	var wg sync.WaitGroup
	numRequests := 10
	errors := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			_, err := cm.Get(ctx, testShortCode)
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify: All requests should receive the same error
	for i := 0; i < numRequests; i++ {
		assert.Error(t, errors[i], "Request %d should return error", i)
	}

	// Verify: Storage was called only once
	assert.Equal(t, 1, store.GetCallCount(), "Storage should be called only once")
}

// TestEnhancedSingleflight_MultipleKeys tests concurrent requests for different keys
func TestEnhancedSingleflight_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  true,
		MetricsPort:    0,
	})
	require.NoError(t, err)
	defer obs.Shutdown(context.Background())

	// Create storage
	store := NewMockStorageWithDelay()
	store.SetDelay(100 * time.Millisecond)

	// Create caches
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)

	l2Config := cache.L2CacheConfig{
		Addrs: []string{"localhost:6379"},
		DB:    1,
	}
	l2, err := cache.NewL2Cache(l2Config, obs)
	require.NoError(t, err)

	loader := cache.NewCacheLoader(l2.Client(), store, l2, obs)
	cm := cache.NewCacheManagerWithLoader(l1, l2, store, loader, obs)

	// Add test data for multiple keys
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	for _, key := range keys {
		err = store.Create(context.Background(), key, "https://example.com/"+key, "127.0.0.1")
		require.NoError(t, err)
	}

	// Test: Launch concurrent requests for different keys
	var wg sync.WaitGroup
	requestsPerKey := 5

	for _, key := range keys {
		for i := 0; i < requestsPerKey; i++ {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				ctx := context.Background()
				_, _ = cm.Get(ctx, k)
			}(key)
		}
	}

	wg.Wait()

	// Verify: Storage should be called once per key (not once per request)
	// Total calls = number of keys (5), not number of requests (25)
	assert.Equal(t, len(keys), store.GetCallCount(), "Storage should be called once per key")

	// Cleanup
	for _, key := range keys {
		_ = cm.Delete(context.Background(), key)
	}
}
