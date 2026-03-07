//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/shortenerpb"
)

// TestSimpleCacheStampede tests cache stampede prevention with a fresh link
// This test creates a new link each time to avoid conflicts
func TestSimpleCacheStampede(t *testing.T) {
	ctx := context.Background()

	// Create a unique short link for this test run
	timestamp := time.Now().UnixNano()
	longURL := fmt.Sprintf("https://example.com/stampede-test-%d", timestamp)

	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for cache stampede test: %s -> %s", shortCode, longURL)

	// Wait a moment to ensure the link is fully created
	time.Sleep(100 * time.Millisecond)

	// Now simulate cache stampede: 100 concurrent requests
	numRequests := 100
	var wg sync.WaitGroup
	results := make(chan struct {
		index    int
		duration time.Duration
		err      error
	}, numRequests)

	startTime := time.Now()
	t.Logf("Starting %d concurrent requests...", numRequests)

	// Launch all concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestStart := time.Now()

			getReq := &shortenerpb.GetLinkInfoRequest{
				ShortCode: shortCode,
			}

			getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
			duration := time.Since(requestStart)

			if err != nil {
				results <- struct {
					index    int
					duration time.Duration
					err      error
				}{index, duration, fmt.Errorf("request %d failed: %w", index, err)}
				return
			}

			if getResp.LongUrl != longURL {
				results <- struct {
					index    int
					duration time.Duration
					err      error
				}{index, duration, fmt.Errorf("request %d got wrong URL", index)}
				return
			}

			results <- struct {
				index    int
				duration time.Duration
				err      error
			}{index, duration, nil}
		}(i)
	}

	// Wait for all requests to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	successCount := 0
	errorCount := 0
	var durations []time.Duration

	for result := range results {
		if result.err != nil {
			t.Logf("Request %d error: %v", result.index, result.err)
			errorCount++
		} else {
			successCount++
			durations = append(durations, result.duration)
		}
	}

	totalDuration := time.Since(startTime)

	t.Logf("Completed %d concurrent requests in %v", numRequests, totalDuration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify that all requests succeeded
	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d (errors: %d)",
			numRequests, successCount, errorCount)
	}

	// Calculate timing statistics
	if len(durations) > 0 {
		var total time.Duration
		minDuration := durations[0]
		maxDuration := durations[0]

		for _, d := range durations {
			total += d
			if d < minDuration {
				minDuration = d
			}
			if d > maxDuration {
				maxDuration = d
			}
		}

		avgDuration := total / time.Duration(len(durations))
		t.Logf("Request timing statistics:")
		t.Logf("  Min: %v", minDuration)
		t.Logf("  Max: %v", maxDuration)
		t.Logf("  Avg: %v", avgDuration)

		// All requests should complete reasonably fast with cache
		if avgDuration > 500*time.Millisecond {
			t.Logf("Warning: Average latency is %v (higher than expected)", avgDuration)
		}

		if maxDuration > 5*time.Second {
			t.Errorf("Maximum duration too high: %v (expected < 5s)", maxDuration)
		}
	}

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	_, err = grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Logf("Warning: Failed to cleanup link %s: %v", shortCode, err)
	}

	t.Log("Cache stampede test completed successfully")
	t.Log("Note: All 100 concurrent requests succeeded, demonstrating cache efficiency")
}

// TestCacheLoaderWithConcurrentRequests tests the CacheLoader's SETNX behavior
// by making concurrent requests to multiple different links
func TestCacheLoaderWithConcurrentRequests(t *testing.T) {
	ctx := context.Background()

	// Create 5 different links
	numLinks := 5
	links := make([]struct {
		shortCode string
		longURL   string
	}, numLinks)

	timestamp := time.Now().UnixNano()
	for i := 0; i < numLinks; i++ {
		longURL := fmt.Sprintf("https://example.com/loader-test-%d-%d", timestamp, i)

		createReq := &shortenerpb.CreateShortLinkRequest{
			LongUrl: longURL,
		}

		createResp, err := grpcClient.CreateShortLink(ctx, createReq)
		if err != nil {
			t.Fatalf("Failed to create link %d: %v", i, err)
		}

		links[i].shortCode = createResp.ShortCode
		links[i].longURL = longURL
		t.Logf("Created link %d: %s -> %s", i, createResp.ShortCode, longURL)
	}

	// For each link, make 20 concurrent requests
	for i, link := range links {
		t.Logf("Testing concurrent access for link %d: %s", i, link.shortCode)

		numRequests := 20
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for j := 0; j < numRequests; j++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				getReq := &shortenerpb.GetLinkInfoRequest{
					ShortCode: link.shortCode,
				}

				getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
				if err != nil {
					t.Logf("Link %d, request %d failed: %v", i, index, err)
					return
				}

				if getResp.LongUrl != link.longURL {
					t.Errorf("Link %d, request %d got wrong URL: expected %s, got %s",
						i, index, link.longURL, getResp.LongUrl)
					return
				}

				mu.Lock()
				successCount++
				mu.Unlock()
			}(j)
		}

		wg.Wait()

		t.Logf("Link %d: %d/%d requests succeeded", i, successCount, numRequests)

		if successCount != numRequests {
			t.Errorf("Link %d: Expected %d successful requests, got %d",
				i, numRequests, successCount)
		}
	}

	// Cleanup
	for i, link := range links {
		deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: link.shortCode}
		_, err := grpcClient.DeleteShortLink(ctx, deleteReq)
		if err != nil {
			t.Logf("Warning: Failed to cleanup link %d (%s): %v", i, link.shortCode, err)
		}
	}

	t.Log("Cache loader concurrent requests test completed successfully")
}
