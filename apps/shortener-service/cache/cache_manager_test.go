package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// createTestObservability creates a test observability instance
func createTestObservability() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "shortener-service-test",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	return obs
}

// MockStorage implements Storage interface for testing
type MockStorage struct {
	data      map[string]*StorageMapping
	mu        sync.RWMutex
	callCount int32
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]*StorageMapping),
	}
}

func (m *MockStorage) Get(ctx context.Context, shortCode string) (*StorageMapping, error) {
	atomic.AddInt32(&m.callCount, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping, ok := m.data[shortCode]
	if !ok {
		return nil, fmt.Errorf("mapping not found")
	}
	return mapping, nil
}

func (m *MockStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data[shortCode]
	return ok, nil
}

func (m *MockStorage) Set(shortCode string, mapping *StorageMapping) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[shortCode] = mapping
}

func (m *MockStorage) GetCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func (m *MockStorage) ResetCallCount() {
	atomic.StoreInt32(&m.callCount, 0)
}

// TestCacheManagerSingleflight verifies singleflight request coalescing
// Requirements: 12.1, 12.2, 12.5
func TestCacheManagerSingleflight(t *testing.T) {
	// Create cache manager with mock storage
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()

	// Add test data to storage
	testMapping := &StorageMapping{
		ShortCode: "test123",
		LongURL:   "https://example.com",
		CreatedAt: time.Now(),
	}
	storage.Set("test123", testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())

	// Launch 100 concurrent requests for the same key
	const numRequests = 100
	var wg sync.WaitGroup
	wg.Add(numRequests)

	ctx := context.Background()
	results := make([]*URLMapping, numRequests)
	errors := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()
			mapping, err := cm.Get(ctx, "test123")
			results[idx] = mapping
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded
	for i, err := range errors {
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// Verify all results are consistent
	for i, mapping := range results {
		if mapping == nil {
			t.Errorf("Request %d returned nil mapping", i)
			continue
		}
		if mapping.ShortCode != "test123" {
			t.Errorf("Request %d returned wrong short code: %s", i, mapping.ShortCode)
		}
		if mapping.LongURL != "https://example.com" {
			t.Errorf("Request %d returned wrong long URL: %s", i, mapping.LongURL)
		}
	}

	// Verify singleflight coalesced requests
	// Due to timing sensitivity, we allow 1-3 DB queries instead of exactly 1
	// The key property is that 100 concurrent requests result in significantly fewer DB queries
	callCount := storage.GetCallCount()
	t.Logf("Singleflight coalesced %d concurrent requests to %d DB queries (%.1f%% reduction)",
		numRequests, callCount, float64(numRequests-callCount)/float64(numRequests)*100)

	if callCount < 1 {
		t.Errorf("Expected at least 1 DB query, got %d", callCount)
	}
	if callCount > 3 {
		t.Errorf("Expected at most 3 DB queries due to timing, got %d (singleflight may not be working)", callCount)
	}
}

// TestCacheManagerFallback verifies multi-tier cache fallback
// Requirements: 3.3, 4.4
func TestCacheManagerFallback(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	testMapping := &StorageMapping{
		ShortCode: "test456",
		LongURL:   "https://example.com/test",
		CreatedAt: time.Now(),
	}
	storage.Set("test456", testMapping)

	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// First request: L1 miss → DB query → backfill L1
	storage.ResetCallCount()
	mapping, err := cm.Get(ctx, "test456")
	if err != nil {
		t.Fatalf("First Get failed: %v", err)
	}
	if mapping.ShortCode != "test456" {
		t.Errorf("Expected short code test456, got %s", mapping.ShortCode)
	}
	if storage.GetCallCount() != 1 {
		t.Errorf("Expected 1 DB query on L1 miss, got %d", storage.GetCallCount())
	}

	// Wait for Ristretto to process the backfill
	time.Sleep(50 * time.Millisecond)

	// Second request: L1 hit → no DB query
	storage.ResetCallCount()
	mapping, err = cm.Get(ctx, "test456")
	if err != nil {
		t.Fatalf("Second Get failed: %v", err)
	}
	if mapping.ShortCode != "test456" {
		t.Errorf("Expected short code test456, got %s", mapping.ShortCode)
	}
	if storage.GetCallCount() != 0 {
		t.Errorf("Expected 0 DB queries on L1 hit, got %d", storage.GetCallCount())
	}
}

// TestCacheManagerDelete verifies cache invalidation
// Requirements: 4.6
func TestCacheManagerDelete(t *testing.T) {
	l1, err := NewL1Cache()
	if err != nil {
		t.Fatalf("Failed to create L1 cache: %v", err)
	}
	defer l1.Close()

	storage := NewMockStorage()
	cm := NewCacheManager(l1, nil, storage, createTestObservability())
	ctx := context.Background()

	// Set a value in L1
	l1.Set("test789", "https://example.com", time.Now())

	// Wait for Ristretto to process the set
	time.Sleep(50 * time.Millisecond)

	// Verify it's in cache
	if mapping := l1.Get("test789"); mapping == nil {
		t.Fatal("Expected mapping in L1 cache")
	}

	// Delete from cache
	err = cm.Delete(ctx, "test789")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's removed from L1
	if mapping := l1.Get("test789"); mapping != nil {
		t.Error("Expected mapping to be removed from L1 cache")
	}
}
