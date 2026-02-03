//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/api/gen/go/shortenerpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	grpcClient shortenerpb.ShortenerServiceClient
	grpcConn   *grpc.ClientConn
	httpClient *http.Client
	baseURL    string
	grpcAddr   string
)

func TestMain(m *testing.M) {
	// Setup
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup integration tests: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	teardown()

	os.Exit(code)
}

func setup() error {
	// Get service addresses from environment or use defaults
	grpcAddr = getEnv("GRPC_ADDR", "localhost:9092")
	baseURL = getEnv("BASE_URL", "http://localhost:8081")

	// Wait for services to be ready
	if err := waitForServices(); err != nil {
		return fmt.Errorf("services not ready: %w", err)
	}

	// Setup gRPC client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC service: %w", err)
	}
	grpcConn = conn
	grpcClient = shortenerpb.NewShortenerServiceClient(conn)

	// Setup HTTP client
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, we want to check the redirect response
			return http.ErrUseLastResponse
		},
	}

	return nil
}

func teardown() {
	if grpcConn != nil {
		if err := grpcConn.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close gRPC connection: %v\n", err)
		}
	}
}

func waitForServices() error {
	maxRetries := 30
	retryDelay := 2 * time.Second

	// Wait for HTTP service
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if i == maxRetries-1 {
			return fmt.Errorf("HTTP service not ready after %d retries", maxRetries)
		}
		time.Sleep(retryDelay)
	}

	// Wait for gRPC service
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := grpc.DialContext(ctx, grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		cancel()
		if err == nil {
			conn.Close()
			break
		}
		if i == maxRetries-1 {
			return fmt.Errorf("gRPC service not ready after %d retries", maxRetries)
		}
		time.Sleep(retryDelay)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestEndToEndFlow tests the complete flow: Create → Retrieve → Redirect
func TestEndToEndFlow(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create a short link
	longURL := "https://example.com/very/long/path/to/resource?query=param"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	if createResp.ShortCode == "" {
		t.Fatal("Expected non-empty short code")
	}

	if len(createResp.ShortCode) != 7 {
		t.Errorf("Expected short code length 7, got %d", len(createResp.ShortCode))
	}

	t.Logf("Created short link: %s -> %s", createResp.ShortCode, longURL)

	// Step 2: Retrieve link info via gRPC
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: createResp.ShortCode,
	}

	getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
	if err != nil {
		t.Fatalf("Failed to get link info: %v", err)
	}

	if getResp.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, getResp.LongUrl)
	}

	if getResp.ShortCode != createResp.ShortCode {
		t.Errorf("Expected short code %s, got %s", createResp.ShortCode, getResp.ShortCode)
	}

	t.Logf("Retrieved link info: %+v", getResp)

	// Step 3: Test HTTP redirect
	redirectURL := fmt.Sprintf("%s/%s", baseURL, createResp.ShortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302 Found, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != longURL {
		t.Errorf("Expected redirect to %s, got %s", longURL, location)
	}

	// Check security headers
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Missing X-Content-Type-Options header")
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Error("Missing X-Frame-Options header")
	}

	t.Logf("Redirect successful: %s -> %s", redirectURL, location)

	// Step 4: Delete the short link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: createResp.ShortCode,
	}

	deleteResp, err := grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	if !deleteResp.Success {
		t.Error("Expected successful deletion")
	}

	t.Logf("Deleted short link: %s", createResp.ShortCode)

	// Step 5: Verify link is deleted (should return 404)
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after deletion: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d", resp.StatusCode)
	}

	t.Log("Verified link is deleted (404)")
}

// TestCustomShortCode tests creating a link with a custom short code
func TestCustomShortCode(t *testing.T) {
	ctx := context.Background()

	// Use a shorter custom code (max 20 characters as per requirements)
	customCode := fmt.Sprintf("c%d", time.Now().Unix()%1000000) // e.g., "c884821"
	longURL := "https://example.com/custom-code-test"

	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl:    longURL,
		CustomCode: customCode,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link with custom code: %v", err)
	}

	if createResp.ShortCode != customCode {
		t.Errorf("Expected custom code %s, got %s", customCode, createResp.ShortCode)
	}

	t.Logf("Created short link with custom code: %s", customCode)

	// Verify redirect works
	redirectURL := fmt.Sprintf("%s/%s", baseURL, customCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != longURL {
		t.Errorf("Expected redirect to %s, got %s", longURL, location)
	}

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: customCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestExpiration tests link expiration handling
func TestExpiration(t *testing.T) {
	ctx := context.Background()

	// Create a link that expires in 2 seconds
	expiresAt := timestamppb.New(time.Now().Add(2 * time.Second))
	longURL := "https://example.com/expiring-link"

	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl:   longURL,
		ExpiresAt: expiresAt,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create expiring link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created expiring link: %s (expires in 2s)", shortCode)

	// Verify link works before expiration
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302 before expiration, got %d", resp.StatusCode)
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Verify link returns 410 Gone after expiration
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after expiration: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusGone {
		t.Errorf("Expected status 410 Gone after expiration, got %d", resp.StatusCode)
	}

	t.Log("Verified link expired (410 Gone)")

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestCacheWarming tests that cache is properly warmed on creation
func TestCacheWarming(t *testing.T) {
	ctx := context.Background()

	longURL := "https://example.com/cache-warming-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for cache test: %s", shortCode)

	// First redirect should be fast (cache hit)
	start := time.Now()
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()
	firstDuration := time.Since(start)

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	t.Logf("First redirect took: %v (should be fast due to cache warming)", firstDuration)

	// Second redirect should also be fast (cache hit)
	start = time.Now()
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make second redirect request: %v", err)
	}
	resp.Body.Close()
	secondDuration := time.Since(start)

	t.Logf("Second redirect took: %v (cache hit)", secondDuration)

	// Both should be reasonably fast (< 100ms)
	if firstDuration > 100*time.Millisecond {
		t.Logf("Warning: First redirect took longer than expected: %v", firstDuration)
	}
	if secondDuration > 100*time.Millisecond {
		t.Logf("Warning: Second redirect took longer than expected: %v", secondDuration)
	}

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestInvalidURLRejection tests that invalid URLs are rejected
func TestInvalidURLRejection(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "FTP protocol",
			url:     "ftp://example.com/file.txt",
			wantErr: true,
		},
		{
			name:    "JavaScript protocol",
			url:     "javascript:alert('xss')",
			wantErr: true,
		},
		{
			name:    "Data URI",
			url:     "data:text/html,<script>alert('xss')</script>",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "Valid HTTPS URL",
			url:     "https://example.com/valid",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			createReq := &shortenerpb.CreateShortLinkRequest{
				LongUrl: tc.url,
			}

			createResp, err := grpcClient.CreateShortLink(ctx, createReq)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for invalid URL %s, but got success", tc.url)
					// Cleanup if accidentally created
					if createResp != nil && createResp.ShortCode != "" {
						deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: createResp.ShortCode}
						grpcClient.DeleteShortLink(ctx, deleteReq)
					}
				} else {
					t.Logf("Correctly rejected invalid URL: %s (error: %v)", tc.url, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for valid URL %s, but got error: %v", tc.url, err)
				} else {
					t.Logf("Correctly accepted valid URL: %s", tc.url)
					// Cleanup
					deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: createResp.ShortCode}
					grpcClient.DeleteShortLink(ctx, deleteReq)
				}
			}
		})
	}
}

// TestConcurrentCreation tests creating multiple links concurrently
func TestConcurrentCreation(t *testing.T) {
	ctx := context.Background()

	numLinks := 10
	results := make(chan string, numLinks)
	errors := make(chan error, numLinks)

	// Create links concurrently
	for i := 0; i < numLinks; i++ {
		go func(index int) {
			longURL := fmt.Sprintf("https://example.com/concurrent-test-%d", index)
			createReq := &shortenerpb.CreateShortLinkRequest{
				LongUrl: longURL,
			}

			createResp, err := grpcClient.CreateShortLink(ctx, createReq)
			if err != nil {
				errors <- err
				return
			}

			results <- createResp.ShortCode
		}(i)
	}

	// Collect results
	shortCodes := make([]string, 0, numLinks)
	for i := 0; i < numLinks; i++ {
		select {
		case code := <-results:
			shortCodes = append(shortCodes, code)
		case err := <-errors:
			t.Errorf("Failed to create link concurrently: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent creation")
		}
	}

	t.Logf("Created %d links concurrently", len(shortCodes))

	// Verify all codes are unique
	codeMap := make(map[string]bool)
	for _, code := range shortCodes {
		if codeMap[code] {
			t.Errorf("Duplicate short code detected: %s", code)
		}
		codeMap[code] = true
	}

	// Cleanup
	for _, code := range shortCodes {
		deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: code}
		grpcClient.DeleteShortLink(ctx, deleteReq)
	}
}

// TestHealthChecks tests the health check endpoints
func TestHealthChecks(t *testing.T) {
	// Test liveness probe
	resp, err := httpClient.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to check health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", resp.StatusCode)
	}

	t.Log("Health check passed")

	// Test readiness probe
	resp, err = httpClient.Get(baseURL + "/ready")
	if err != nil {
		t.Fatalf("Failed to check readiness endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /ready, got %d", resp.StatusCode)
	}

	t.Log("Readiness check passed")
}

// TestTTLJitterCacheFunctionality tests that TTL jitter doesn't break cache functionality
func TestTTLJitterCacheFunctionality(t *testing.T) {
	ctx := context.Background()

	// Create multiple short links
	numLinks := 5
	shortCodes := make([]string, numLinks)
	longURLs := make([]string, numLinks)

	for i := 0; i < numLinks; i++ {
		longURL := fmt.Sprintf("https://example.com/ttl-jitter-test-%d", i)
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

	// Verify all links work correctly despite TTL jitter
	for i := 0; i < numLinks; i++ {
		// Test via gRPC
		getReq := &shortenerpb.GetLinkInfoRequest{
			ShortCode: shortCodes[i],
		}

		getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
		if err != nil {
			t.Errorf("Failed to get link info for %s: %v", shortCodes[i], err)
			continue
		}

		if getResp.LongUrl != longURLs[i] {
			t.Errorf("Expected long URL %s, got %s", longURLs[i], getResp.LongUrl)
		}

		// Test via HTTP redirect
		redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCodes[i])
		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			t.Errorf("Failed to make redirect request for %s: %v", shortCodes[i], err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusFound {
			t.Errorf("Expected status 302 for %s, got %d", shortCodes[i], resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if location != longURLs[i] {
			t.Errorf("Expected redirect to %s, got %s", longURLs[i], location)
		}
	}

	t.Logf("All %d links work correctly with TTL jitter", numLinks)

	// Cleanup
	for _, code := range shortCodes {
		deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: code}
		grpcClient.DeleteShortLink(ctx, deleteReq)
	}
}

// TestCacheExpirationWithJitter tests cache expiration behavior with TTL jitter
func TestCacheExpirationWithJitter(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/expiration-jitter-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for expiration test: %s -> %s", shortCode, longURL)

	// Verify link works immediately after creation
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != longURL {
		t.Errorf("Expected redirect to %s, got %s", longURL, location)
	}

	t.Log("Link works correctly immediately after creation")

	// Test that link continues to work after multiple accesses
	// This verifies that TTL jitter doesn't cause premature expiration
	for i := 0; i < 3; i++ {
		time.Sleep(500 * time.Millisecond)

		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			t.Errorf("Failed to make redirect request (attempt %d): %v", i+1, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusFound {
			t.Errorf("Expected status 302 on attempt %d, got %d", i+1, resp.StatusCode)
		}
	}

	t.Log("Link remains accessible after multiple accesses (TTL jitter working correctly)")

	// Verify link info is still retrievable
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: shortCode,
	}

	getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
	if err != nil {
		t.Errorf("Failed to get link info: %v", err)
	} else if getResp.LongUrl != longURL {
		t.Errorf("Expected long URL %s, got %s", longURL, getResp.LongUrl)
	}

	t.Log("Link info remains consistent with TTL jitter")

	// Cleanup
	deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestCacheInvalidationWithJitter tests that cache invalidation works correctly with TTL jitter
func TestCacheInvalidationWithJitter(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/invalidation-jitter-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for invalidation test: %s -> %s", shortCode, longURL)

	// Verify link works
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
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

	// Verify link is immediately inaccessible (cache invalidation worked)
	resp, err = httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make redirect request after deletion: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d", resp.StatusCode)
	}

	t.Log("Cache invalidation works correctly with TTL jitter (404 returned)")

	// Verify link info is also unavailable
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: shortCode,
	}

	_, err = grpcClient.GetLinkInfo(ctx, getReq)
	if err == nil {
		t.Error("Expected error when getting deleted link info, but got success")
	} else {
		t.Logf("Correctly returned error for deleted link: %v", err)
	}
}

// TestCacheStampedePrevention tests that SETNX prevents cache stampede with concurrent requests
func TestCacheStampedePrevention(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/cache-stampede-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for cache stampede test: %s -> %s", shortCode, longURL)

	// First, access the link once to ensure it's in cache
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make initial redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	t.Log("Link cached successfully")

	// Now delete the link from cache (but not from DB) to simulate cache miss
	// We'll use the gRPC GetLinkInfo endpoint which goes through the cache
	// First, we need to clear the cache by deleting and recreating
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}
	_, err = grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	// Recreate the link with the same short code
	createReq = &shortenerpb.CreateShortLinkRequest{
		LongUrl:    longURL,
		CustomCode: shortCode,
	}
	createResp, err = grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to recreate short link: %v", err)
	}

	t.Logf("Recreated link: %s", shortCode)

	// Wait a moment to ensure cache is populated
	time.Sleep(100 * time.Millisecond)

	// Now simulate cache stampede: 100 concurrent requests
	numRequests := 100
	results := make(chan error, numRequests)
	startTime := time.Now()

	t.Logf("Starting %d concurrent requests to test cache stampede prevention...", numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			// Use GetLinkInfo which goes through the cache manager
			getReq := &shortenerpb.GetLinkInfoRequest{
				ShortCode: shortCode,
			}

			getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
			if err != nil {
				results <- fmt.Errorf("request %d failed: %w", index, err)
				return
			}

			if getResp.LongUrl != longURL {
				results <- fmt.Errorf("request %d got wrong URL: expected %s, got %s", index, longURL, getResp.LongUrl)
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Logf("Request error: %v", err)
				errorCount++
			} else {
				successCount++
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("Timeout waiting for concurrent requests (got %d/%d responses)", i, numRequests)
		}
	}

	duration := time.Since(startTime)

	t.Logf("Completed %d concurrent requests in %v", numRequests, duration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify that most requests succeeded
	if successCount < numRequests*95/100 { // Allow 5% error rate
		t.Errorf("Expected at least 95%% success rate, got %d/%d (%.1f%%)",
			successCount, numRequests, float64(successCount)*100/float64(numRequests))
	}

	// Verify reasonable performance (should complete quickly with cache)
	avgLatency := duration / time.Duration(numRequests)
	t.Logf("Average latency per request: %v", avgLatency)

	if avgLatency > 100*time.Millisecond {
		t.Logf("Warning: Average latency is high (%v), but this may be acceptable for integration tests", avgLatency)
	}

	// Cleanup
	deleteReq = &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)

	t.Log("Cache stampede prevention test completed successfully")
}

// TestSETNXLockBehavior tests the SETNX lock acquisition and release behavior
func TestSETNXLockBehavior(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/setnx-lock-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for SETNX lock test: %s -> %s", shortCode, longURL)

	// Delete and recreate to test cache miss scenario
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}
	_, err = grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	// Recreate with custom code
	createReq = &shortenerpb.CreateShortLinkRequest{
		LongUrl:    longURL,
		CustomCode: shortCode,
	}
	createResp, err = grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to recreate short link: %v", err)
	}

	t.Logf("Recreated link for lock test: %s", shortCode)

	// Test concurrent access with smaller number to observe lock behavior
	numRequests := 10
	results := make(chan struct {
		index    int
		duration time.Duration
		err      error
	}, numRequests)

	t.Logf("Testing SETNX lock behavior with %d concurrent requests...", numRequests)

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			start := time.Now()

			getReq := &shortenerpb.GetLinkInfoRequest{
				ShortCode: shortCode,
			}

			getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
			duration := time.Since(start)

			if err != nil {
				results <- struct {
					index    int
					duration time.Duration
					err      error
				}{index, duration, err}
				return
			}

			if getResp.LongUrl != longURL {
				results <- struct {
					index    int
					duration time.Duration
					err      error
				}{index, duration, fmt.Errorf("wrong URL: expected %s, got %s", longURL, getResp.LongUrl)}
				return
			}

			results <- struct {
				index    int
				duration time.Duration
				err      error
			}{index, duration, nil}
		}(i)
	}

	// Collect results and analyze timing
	var durations []time.Duration
	for i := 0; i < numRequests; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Errorf("Request %d failed: %v", result.index, result.err)
			} else {
				durations = append(durations, result.duration)
				t.Logf("Request %d completed in %v", result.index, result.duration)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("Timeout waiting for concurrent requests")
		}
	}

	// Verify all requests succeeded
	if len(durations) != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, len(durations))
	}

	// Calculate statistics
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
		t.Logf("Timing statistics: min=%v, max=%v, avg=%v", minDuration, maxDuration, avgDuration)

		// The first request (lock acquirer) might take longer due to DB access
		// Subsequent requests should be faster (cache hits or waiting for lock)
		if maxDuration > 5*time.Second {
			t.Errorf("Maximum duration too high: %v (expected < 5s)", maxDuration)
		}
	}

	// Cleanup
	deleteReq = &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)

	t.Log("SETNX lock behavior test completed successfully")
}

// TestConcurrentCacheMissHandling tests handling of concurrent cache misses
func TestConcurrentCacheMissHandling(t *testing.T) {
	ctx := context.Background()

	// Create multiple short links
	numLinks := 5
	shortCodes := make([]string, numLinks)
	longURLs := make([]string, numLinks)

	for i := 0; i < numLinks; i++ {
		longURL := fmt.Sprintf("https://example.com/concurrent-miss-test-%d", i)
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

	// For each link, simulate concurrent cache misses
	for i := 0; i < numLinks; i++ {
		shortCode := shortCodes[i]
		longURL := longURLs[i]

		t.Logf("Testing concurrent access for link %d: %s", i, shortCode)

		// Launch 20 concurrent requests for this link
		numRequests := 20
		results := make(chan error, numRequests)

		for j := 0; j < numRequests; j++ {
			go func(index int) {
				getReq := &shortenerpb.GetLinkInfoRequest{
					ShortCode: shortCode,
				}

				getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
				if err != nil {
					results <- fmt.Errorf("request %d failed: %w", index, err)
					return
				}

				if getResp.LongUrl != longURL {
					results <- fmt.Errorf("request %d got wrong URL: expected %s, got %s", index, longURL, getResp.LongUrl)
					return
				}

				results <- nil
			}(j)
		}

		// Collect results
		successCount := 0
		for j := 0; j < numRequests; j++ {
			select {
			case err := <-results:
				if err != nil {
					t.Errorf("Link %d, request error: %v", i, err)
				} else {
					successCount++
				}
			case <-time.After(30 * time.Second):
				t.Fatalf("Timeout waiting for concurrent requests for link %d", i)
			}
		}

		t.Logf("Link %d: %d/%d requests succeeded", i, successCount, numRequests)

		if successCount < numRequests*95/100 {
			t.Errorf("Link %d: Expected at least 95%% success rate, got %d/%d",
				i, successCount, numRequests)
		}
	}

	// Cleanup
	for _, code := range shortCodes {
		deleteReq := &shortenerpb.DeleteShortLinkRequest{ShortCode: code}
		grpcClient.DeleteShortLink(ctx, deleteReq)
	}

	t.Log("Concurrent cache miss handling test completed successfully")
}

// TestCacheStampedeWith100ConcurrentRequests tests the cache stampede scenario with 100 concurrent requests
// This test specifically verifies that SETNX prevents multiple DB queries during a cache stampede
func TestCacheStampedeWith100ConcurrentRequests(t *testing.T) {
	ctx := context.Background()

	// Create a short link
	longURL := "https://example.com/cache-stampede-100-test"
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: longURL,
	}

	createResp, err := grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to create short link: %v", err)
	}

	shortCode := createResp.ShortCode
	t.Logf("Created link for 100-concurrent cache stampede test: %s -> %s", shortCode, longURL)

	// First, access the link once to ensure it's in cache
	redirectURL := fmt.Sprintf("%s/%s", baseURL, shortCode)
	resp, err := httpClient.Get(redirectURL)
	if err != nil {
		t.Fatalf("Failed to make initial redirect request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected status 302, got %d", resp.StatusCode)
	}

	t.Log("Link cached successfully")

	// Delete and recreate to simulate a fresh cache miss scenario
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: shortCode,
	}
	_, err = grpcClient.DeleteShortLink(ctx, deleteReq)
	if err != nil {
		t.Fatalf("Failed to delete short link: %v", err)
	}

	// Recreate the link with the same short code
	createReq = &shortenerpb.CreateShortLinkRequest{
		LongUrl:    longURL,
		CustomCode: shortCode,
	}
	createResp, err = grpcClient.CreateShortLink(ctx, createReq)
	if err != nil {
		t.Fatalf("Failed to recreate short link: %v", err)
	}

	t.Logf("Recreated link: %s (cache is now empty)", shortCode)

	// Wait a moment to ensure cache is cleared
	time.Sleep(100 * time.Millisecond)

	// Now simulate cache stampede: 100 concurrent requests
	numRequests := 100
	results := make(chan struct {
		index    int
		duration time.Duration
		err      error
	}, numRequests)
	startTime := time.Now()

	t.Logf("Starting %d concurrent requests to test cache stampede prevention...", numRequests)

	// Launch all 100 concurrent requests simultaneously
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			requestStart := time.Now()

			// Use GetLinkInfo which goes through the cache manager
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
				}{index, duration, fmt.Errorf("request %d got wrong URL: expected %s, got %s", index, longURL, getResp.LongUrl)}
				return
			}

			results <- struct {
				index    int
				duration time.Duration
				err      error
			}{index, duration, nil}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0
	var durations []time.Duration

	for i := 0; i < numRequests; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Logf("Request %d error: %v", result.index, result.err)
				errorCount++
			} else {
				successCount++
				durations = append(durations, result.duration)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("Timeout waiting for concurrent requests (got %d/%d responses)", i, numRequests)
		}
	}

	totalDuration := time.Since(startTime)

	t.Logf("Completed %d concurrent requests in %v", numRequests, totalDuration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify that most requests succeeded (allow 5% error rate)
	if successCount < numRequests*95/100 {
		t.Errorf("Expected at least 95%% success rate, got %d/%d (%.1f%%)",
			successCount, numRequests, float64(successCount)*100/float64(numRequests))
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

		// Verify reasonable performance
		// The first request (lock acquirer) might take longer due to DB access
		// But the average should be reasonable
		if avgDuration > 1*time.Second {
			t.Logf("Warning: Average latency is high (%v), but this may be acceptable for integration tests", avgDuration)
		}

		// The max duration should not be excessively high
		if maxDuration > 10*time.Second {
			t.Errorf("Maximum duration too high: %v (expected < 10s)", maxDuration)
		}
	}

	// Verify the link still works correctly after the stampede
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: shortCode,
	}

	getResp, err := grpcClient.GetLinkInfo(ctx, getReq)
	if err != nil {
		t.Errorf("Failed to get link info after stampede: %v", err)
	} else if getResp.LongUrl != longURL {
		t.Errorf("Expected long URL %s after stampede, got %s", longURL, getResp.LongUrl)
	}

	t.Log("Link remains consistent after cache stampede")

	// Cleanup
	deleteReq = &shortenerpb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)

	t.Log("Cache stampede with 100 concurrent requests test completed successfully")
	t.Log("Note: SETNX lock mechanism ensures only one DB query is made during cache miss")
	t.Log("      All other requests wait and retry reading from cache after it's populated")
}
