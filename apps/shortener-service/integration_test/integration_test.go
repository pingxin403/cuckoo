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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/pingxin403/cuckoo/apps/shortener-service/gen/shortener_servicepb"
)

var (
	grpcClient pb.ShortenerServiceClient
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
	grpcClient = pb.NewShortenerServiceClient(conn)

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
		grpcConn.Close()
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
	createReq := &pb.CreateShortLinkRequest{
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
	getReq := &pb.GetLinkInfoRequest{
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
	deleteReq := &pb.DeleteShortLinkRequest{
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

	createReq := &pb.CreateShortLinkRequest{
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
	deleteReq := &pb.DeleteShortLinkRequest{ShortCode: customCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestExpiration tests link expiration handling
func TestExpiration(t *testing.T) {
	ctx := context.Background()

	// Create a link that expires in 2 seconds
	expiresAt := timestamppb.New(time.Now().Add(2 * time.Second))
	longURL := "https://example.com/expiring-link"

	createReq := &pb.CreateShortLinkRequest{
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
	deleteReq := &pb.DeleteShortLinkRequest{ShortCode: shortCode}
	grpcClient.DeleteShortLink(ctx, deleteReq)
}

// TestCacheWarming tests that cache is properly warmed on creation
func TestCacheWarming(t *testing.T) {
	ctx := context.Background()

	longURL := "https://example.com/cache-warming-test"
	createReq := &pb.CreateShortLinkRequest{
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
	deleteReq := &pb.DeleteShortLinkRequest{ShortCode: shortCode}
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
			createReq := &pb.CreateShortLinkRequest{
				LongUrl: tc.url,
			}

			createResp, err := grpcClient.CreateShortLink(ctx, createReq)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for invalid URL %s, but got success", tc.url)
					// Cleanup if accidentally created
					if createResp != nil && createResp.ShortCode != "" {
						deleteReq := &pb.DeleteShortLinkRequest{ShortCode: createResp.ShortCode}
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
					deleteReq := &pb.DeleteShortLinkRequest{ShortCode: createResp.ShortCode}
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
			createReq := &pb.CreateShortLinkRequest{
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
		deleteReq := &pb.DeleteShortLinkRequest{ShortCode: code}
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
