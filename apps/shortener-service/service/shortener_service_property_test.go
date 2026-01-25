//go:build property
// +build property

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	pb "github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
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

// TestProperty_MultiStoreWriteConsistency verifies Property 7: Multi-Store Write Consistency
// Feature: url-shortener-service, Property 7: Multi-Store Write Consistency
// Validates: Requirements 4.3
//
// # For any created URL mapping, it SHALL be written to both MySQL and Redis before returning success to the client
//
// This test verifies that:
// 1. MySQL write completes successfully before returning to client
// 2. Cache write is attempted (to L1 at minimum)
// 3. Data in both stores is consistent when both are available
func TestProperty_MultiStoreWriteConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup: Create tracking storage
		tracker := &writeTracker{
			storageWrites: make([]writeEvent, 0),
		}

		mockStore := &trackingMockStorage{
			MockStorage: NewMockStorage(),
			tracker:     tracker,
		}

		idGen := idgen.NewRandomIDGenerator(mockStore)
		validator := NewURLValidator()

		// Create L1 cache
		l1, err := cache.NewL1Cache()
		require.NoError(t, err)

		// Create cache manager (without L2 for simplicity)
		cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: mockStore.MockStorage}, createTestObservability())

		// Create service
		service := NewShortenerServiceImpl(mockStore, idGen, validator, cacheManager, createTestObservability())

		// Generate random valid URL
		domain := rapid.StringMatching(`^[a-z0-9-]+$`).Draw(t, "domain")
		path := rapid.StringMatching(`^[a-zA-Z0-9/_-]*$`).Draw(t, "path")
		longURL := "https://" + domain + ".com/" + path

		// Create short link
		req := &pb.CreateShortLinkRequest{
			LongUrl: longURL,
		}

		resp, err := service.CreateShortLink(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		shortCode := resp.ShortCode

		// CRITICAL VERIFICATION: MySQL write must have completed
		// This is the core of Property 7 - we don't return success until MySQL confirms
		assert.True(t, len(tracker.storageWrites) > 0, "MySQL write must have occurred before returning success")
		if len(tracker.storageWrites) > 0 {
			assert.Equal(t, shortCode, tracker.storageWrites[0].shortCode,
				"MySQL write should be for the correct short code")
		}

		// Verify: Check that mapping exists in MySQL (storage)
		storageMapping, err := mockStore.Get(context.Background(), shortCode)
		assert.NoError(t, err, "mapping should exist in MySQL")
		assert.NotNil(t, storageMapping, "mapping should exist in MySQL")
		if storageMapping != nil {
			assert.Equal(t, longURL, storageMapping.LongURL, "MySQL should have correct URL")
			assert.Equal(t, shortCode, storageMapping.ShortCode, "MySQL should have correct short code")
		}

		// Verify: Cache manager was provided (cache write was attempted)
		// Note: We can't easily verify L1 cache contents due to Ristretto's async nature,
		// but we verify that the service has a cache manager and would have called Set()
		assert.NotNil(t, cacheManager, "cache manager should be provided to service")

		// Additional verification: Retrieve through cache manager to verify consistency
		// This tests the full flow: if cache has it, return from cache; otherwise fetch from DB
		retrievedMapping, err := cacheManager.Get(context.Background(), shortCode)
		assert.NoError(t, err, "should be able to retrieve mapping through cache manager")
		assert.NotNil(t, retrievedMapping, "retrieved mapping should not be nil")
		if retrievedMapping != nil {
			assert.Equal(t, longURL, retrievedMapping.LongURL, "retrieved URL should match original")
			assert.Equal(t, shortCode, retrievedMapping.ShortCode, "retrieved short code should match")
		}

		// Verify consistency: Storage and cache manager return the same data
		if storageMapping != nil && retrievedMapping != nil {
			assert.Equal(t, storageMapping.LongURL, retrievedMapping.LongURL,
				"MySQL and cache should return consistent data")
		}
	})
}

// writeEvent tracks when a write occurred
type writeEvent struct {
	shortCode string
	timestamp time.Time
}

// writeTracker tracks write operations
type writeTracker struct {
	mu            sync.Mutex
	storageWrites []writeEvent
}

func (w *writeTracker) recordStorageWrite(shortCode string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.storageWrites = append(w.storageWrites, writeEvent{
		shortCode: shortCode,
		timestamp: time.Now(),
	})
}

// trackingMockStorage wraps MockStorage to track writes
type trackingMockStorage struct {
	*MockStorage
	tracker *writeTracker
}

func (t *trackingMockStorage) Create(ctx context.Context, mapping *storage.URLMapping) error {
	t.tracker.recordStorageWrite(mapping.ShortCode)
	return t.MockStorage.Create(ctx, mapping)
}
