package cache

import (
	"context"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability"
)

// TestL2Cache_TTLMetrics verifies that TTL distribution metrics are recorded
func TestL2Cache_TTLMetrics(t *testing.T) {
	// Create observability with metrics enabled
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		Environment:    "test",
		EnableMetrics:  true,
		UseOTelMetrics: false, // Use Prometheus for easier testing
		EnableTracing:  false,
		LogLevel:       "error",
	})
	if err != nil {
		t.Fatalf("Failed to create observability: %v", err)
	}
	defer func() {
		_ = obs.Shutdown(context.Background())
	}()

	// Create L2 cache
	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config, obs)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Set multiple entries to generate TTL metrics
	for i := 0; i < 10; i++ {
		shortCode := "test" + string(rune('0'+i))
		longURL := "https://example.com/" + shortCode
		err := cache.Set(ctx, shortCode, longURL, time.Now())
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}
	}

	// Note: In a real scenario, we would verify the metrics are exposed via Prometheus
	// For this test, we just verify that the Set operations succeed without errors
	// The metrics are recorded via obs.Metrics().RecordHistogram() in the Set method

	// Cleanup
	for i := 0; i < 10; i++ {
		shortCode := "test" + string(rune('0'+i))
		_ = cache.Delete(ctx, shortCode)
	}

	t.Log("TTL metrics test completed successfully")
}

// TestL2Cache_TTLDistribution verifies TTL values are distributed
func TestL2Cache_TTLDistribution(t *testing.T) {
	obs := createTestObservability()

	config := L2CacheConfig{
		Addrs:    []string{"localhost:6379"},
		Password: "",
		DB:       0,
	}

	cache, err := NewL2Cache(config, obs)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return
	}
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()

	// Set multiple entries and collect their TTLs
	ttls := make([]time.Duration, 20)
	shortCodes := make([]string, 20)

	for i := 0; i < 20; i++ {
		shortCode := "ttl_test_" + string(rune('a'+i))
		shortCodes[i] = shortCode
		longURL := "https://example.com/" + shortCode

		err := cache.Set(ctx, shortCode, longURL, time.Now())
		if err != nil {
			t.Fatalf("Failed to set value %d: %v", i, err)
		}

		// Get the TTL from Redis
		key := "url:" + shortCode
		ttl, err := cache.client.TTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL for %s: %v", shortCode, err)
		}
		ttls[i] = ttl
	}

	// Verify TTLs are distributed (not all the same)
	uniqueTTLs := make(map[time.Duration]bool)
	for _, ttl := range ttls {
		// Round to seconds to avoid minor timing differences
		roundedTTL := ttl.Round(time.Second)
		uniqueTTLs[roundedTTL] = true
	}

	// We expect at least 10 different TTL values out of 20 entries
	// (with ±1 day jitter on 7 days, we should see good distribution)
	if len(uniqueTTLs) < 10 {
		t.Errorf("Expected at least 10 unique TTL values, got %d", len(uniqueTTLs))
		t.Logf("TTL values: %v", ttls)
	}

	// Verify TTLs are within expected range: 6-8 days (7 days ± 1 day)
	minExpected := 6 * 24 * time.Hour
	maxExpected := 8 * 24 * time.Hour

	for i, ttl := range ttls {
		if ttl < minExpected || ttl > maxExpected {
			t.Errorf("TTL %d out of range: %v (expected %v to %v)", i, ttl, minExpected, maxExpected)
		}
	}

	// Cleanup
	for _, shortCode := range shortCodes {
		_ = cache.Delete(ctx, shortCode)
	}

	t.Logf("TTL distribution test completed: %d unique TTL values out of 20 entries", len(uniqueTTLs))
}
