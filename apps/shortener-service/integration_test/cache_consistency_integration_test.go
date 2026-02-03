//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/shortenerpb"
)

// TestCacheConsistency_CreateOperation tests cache consistency during create operations
func TestCacheConsistency_CreateOperation(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/consistency-create-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link: %s -> %s", shortCode, longURL)

	// Immediately verify the link is accessible (cache should be populated)
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 302 {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != longURL {
		t.Errorf("Expected redirect to %s, got %s", longURL, location)
	}

	t.Log("Link accessible immediately after creation")

	// Wait for delayed delete to complete (1 second + buffer)
	time.Sleep(1500 * time.Millisecond)

	// Verify link is still accessible after delayed delete
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after delayed delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 302 {
		t.Errorf("Expected status 302 after delayed delete, got %d", resp.StatusCode)
	}

	t.Log("Link still accessible after delayed delete (consistency maintained)")

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestCacheConsistency_DeleteOperation tests cache consistency during delete operations
func TestCacheConsistency_DeleteOperation(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/consistency-delete-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link: %s -> %s", shortCode, longURL)

	// Verify link works
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 302 {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	t.Log("Link works before deletion")

	// Delete the link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}

	deleteResp, err := grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	if !deleteResp.Success {
		t.Error("Expected successful deletion")
	}

	t.Log("Link deleted successfully")

	// Immediately verify link is inaccessible (immediate delete worked)
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after deletion: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 immediately after deletion, got %d", resp.StatusCode)
	}

	t.Log("Link immediately inaccessible after deletion (immediate delete worked)")

	// Wait for delayed delete to complete
	time.Sleep(1500 * time.Millisecond)

	// Verify link is still inaccessible (delayed delete completed)
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after delayed delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 after delayed delete, got %d", resp.StatusCode)
	}

	t.Log("Link still inaccessible after delayed delete (consistency maintained)")
}

// TestCacheConsistency_ReplicationLagSimulation tests cache consistency with simulated replication lag
func TestCacheConsistency_ReplicationLagSimulation(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/replication-lag-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link: %s -> %s", shortCode, longURL)

	// Access the link multiple times to ensure it's in cache
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	for i := 0; i < 3; i++ {
		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			t.Fatalf("Failed to make redirect request %d: %v", i+1, err)
		}
		resp.Body.Close()

		if resp.StatusCode != 302 {
			t.Errorf("Expected status 302 on access %d, got %d", i+1, resp.StatusCode)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Link cached and accessed multiple times")

	// Delete the link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}

	deleteResp, err := grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	if !deleteResp.Success {
		t.Error("Expected successful deletion")
	}

	t.Log("Link deleted")

	// Verify immediate cache invalidation
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after deletion: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 immediately after deletion, got %d", resp.StatusCode)
	}

	t.Log("Immediate cache invalidation successful")

	// Wait for delayed delete (simulating replication lag window)
	// During this time, if there was replication lag, stale data might be read
	// The delayed delete ensures cache is cleared again after lag resolves
	time.Sleep(1500 * time.Millisecond)

	// Verify link is still inaccessible
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after delayed delete: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 after delayed delete, got %d", resp.StatusCode)
	}

	t.Log("Delayed delete completed, cache consistency maintained despite potential replication lag")
}

// TestCacheConsistency_ConcurrentOperations tests cache consistency with concurrent operations
func TestCacheConsistency_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()

	// Create multiple short links
	numLinks := 5
	shortCodes := make([]string, numLinks)
	longURLs := make([]string, numLinks)

	for i := 0; i < numLinks; i++ {
		longURL := fmt.Sprintf("https://example.com/concurrent-consistency-test-%d", i)
		longURLs[i] = longURL

		createReq := &shortenerpb.CreateShortLinkRequest{
			LongUrl: longURL,
		}

		createResp, err := grpcClient.CreateShortLink(ctx, createReq)
		if err != nil {
			t.Fatalf("Failed to create short link %d: %v", i, err)
		}

		shortCodes[i] = createResp.ShortCode
		t.Logf("Created link %d: %s -> %s", i, createResp.ShortCode, longURL)
	}

	// Access all links concurrently to populate cache
	results := make(chan error, numLinks)
	for i := 0; i < numLinks; i++ {
		go func(index int) {
			redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCodes[index])
			resp, err := httpClient.Get(redirectURL)
			if err != nil {
				results <- fmt.Errorf("failed to access link %d: %w", index, err)
				return
			}
			resp.Body.Close()

			if resp.StatusCode != 302 {
				results <- fmt.Errorf("link %d returned status %d, expected 302", index, resp.StatusCode)
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numLinks; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent access error: %v", err)
		}
	}

	t.Log("All links cached successfully")

	// Delete all links concurrently
	for i := 0; i < numLinks; i++ {
		go func(index int) {
			deleteReq := &shortenerpb.DeleteShortLinkRequest{
				ShortCode: shortCodes[index],
			}

			deleteResp, err := grpcClient.DeleteShortLink(ctx, deleteReq)
			if err != nil {
				results <- fmt.Errorf("failed to delete link %d: %w", index, err)
				return
			}

			if !deleteResp.Success {
				results <- fmt.Errorf("link %d deletion not successful", index)
				return
			}

			results <- nil
		}(i)
	}

	// Collect deletion results
	for i := 0; i < numLinks; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent deletion error: %v", err)
		}
	}

	t.Log("All links deleted concurrently")

	// Verify all links are immediately inaccessible
	for i := 0; i < numLinks; i++ {
		redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCodes[i])
		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			t.Errorf("Failed to check link %d after deletion: %v", i, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Errorf("Link %d returned status %d after deletion, expected 404", i, resp.StatusCode)
		}
	}

	t.Log("All links immediately inaccessible after concurrent deletion")

	// Wait for delayed deletes to complete
	time.Sleep(1500 * time.Millisecond)

	// Verify all links are still inaccessible
	for i := 0; i < numLinks; i++ {
		redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCodes[i])
		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			t.Errorf("Failed to check link %d after delayed delete: %v", i, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 404 {
			t.Errorf("Link %d returned status %d after delayed delete, expected 404", i, resp.StatusCode)
		}
	}

	t.Log("Cache consistency maintained for all links after concurrent operations")
}

// TestCacheConsistency_ErrorHandling tests cache consistency error handling
func TestCacheConsistency_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/error-handling-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link: %s -> %s", shortCode, longURL)

	// Verify link works
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 302 {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	t.Log("Link works correctly")

	// Delete the link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}

	deleteResp, err := grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	if !deleteResp.Success {
		t.Error("Expected successful deletion")
	}

	t.Log("Link deleted successfully")

	// Try to delete again (should handle gracefully)
	deleteResp, err = grpcClient.DeleteShortLink(ctx, deleteReq)
	if err == nil {
		t.Error("Expected error when deleting non-existent link, but got success")
	} else {
		t.Logf("Correctly returned error for duplicate deletion: %v", err)
	}

	// Verify link is still inaccessible
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	t.Log("Error handling works correctly, cache consistency maintained")
}
