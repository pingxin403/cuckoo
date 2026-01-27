package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Test token consumption and refill
// Requirements: 6.2
func TestTokenBucket_ConsumptionAndRefill(t *testing.T) {
	rl := NewRateLimiter(10) // 10 requests per minute
	ip := "192.168.1.1"

	// Should allow first 10 requests
	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow(ip)
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// 11th request should be rejected
	allowed, retryAfter := rl.Allow(ip)
	if allowed {
		t.Fatal("11th request should be rejected")
	}
	if retryAfter <= 0 {
		t.Fatal("Retry-After should be positive")
	}
}

// Test concurrent access to token bucket
// Requirements: 6.2
func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100)
	ip := "192.168.1.1"

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Launch 10 goroutines, each making 20 requests
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				allowed, _ := rl.Allow(ip)
				if allowed {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 100 requests
	if allowedCount != 100 {
		t.Fatalf("Expected 100 allowed requests, got %d", allowedCount)
	}
}

// Test token refill after time passes
// Requirements: 6.2
func TestTokenBucket_Refill(t *testing.T) {
	// Create rate limiter with short refill interval
	rl := &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		limit:   5,
		refill:  100 * time.Millisecond,
	}
	ip := "192.168.1.1"

	// Exhaust all tokens
	for i := 0; i < 5; i++ {
		allowed, _ := rl.Allow(ip)
		if !allowed {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// Next request should be rejected
	allowed, _ := rl.Allow(ip)
	if allowed {
		t.Fatal("Request should be rejected after exhausting tokens")
	}

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should allow requests again
	for i := 0; i < 5; i++ {
		allowed, _ := rl.Allow(ip)
		if !allowed {
			t.Fatalf("Request %d should be allowed after refill", i+1)
		}
	}
}

// Test independent rate limits for different IPs
// Requirements: 6.1
func TestRateLimiter_IndependentIPs(t *testing.T) {
	rl := NewRateLimiter(10)

	// IP1 exhausts its tokens
	ip1 := "192.168.1.1"
	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow(ip1)
		if !allowed {
			t.Fatalf("IP1 request %d should be allowed", i+1)
		}
	}

	// IP1's next request should be rejected
	allowed, _ := rl.Allow(ip1)
	if allowed {
		t.Fatal("IP1's 11th request should be rejected")
	}

	// IP2 should still have full tokens
	ip2 := "192.168.1.2"
	for i := 0; i < 10; i++ {
		allowed, _ := rl.Allow(ip2)
		if !allowed {
			t.Fatalf("IP2 request %d should be allowed", i+1)
		}
	}
}

// Test gRPC interceptor
// Requirements: 6.1, 6.2, 6.5
func TestRateLimiter_UnaryServerInterceptor(t *testing.T) {
	rl := NewRateLimiter(5)
	interceptor := rl.UnaryServerInterceptor()

	// Mock handler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// Create context with peer info
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &mockAddr{addr: "192.168.1.1:12345"},
	})

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
		if err != nil {
			t.Fatalf("Request %d should succeed: %v", i+1, err)
		}
	}

	// 6th request should be rate limited
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		t.Fatal("6th request should be rate limited")
	}

	// Check error code
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("Error should be a gRPC status error")
	}
	if st.Code() != codes.ResourceExhausted {
		t.Fatalf("Expected ResourceExhausted code, got %v", st.Code())
	}
}

// Test HTTP middleware
// Requirements: 6.1, 6.2, 6.5
func TestRateLimiter_HTTPMiddleware(t *testing.T) {
	rl := NewRateLimiter(5)

	// Mock handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Wrap with rate limiter middleware
	middleware := rl.HTTPMiddleware(handler)

	// First 5 requests should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Request %d should succeed, got status %d", i+1, w.Code)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected status 429, got %d", w.Code)
	}

	// Check Retry-After header
	retryAfter := w.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("Retry-After header should be set")
	}
}

// Test cleanup removes old buckets
// Requirements: 6.2
func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10)

	// Create buckets for multiple IPs
	for i := 1; i <= 10; i++ {
		ip := "192.168.1." + string(rune('0'+i))
		rl.Allow(ip)
	}

	// Verify buckets exist
	rl.mu.RLock()
	initialCount := len(rl.buckets)
	rl.mu.RUnlock()

	if initialCount != 10 {
		t.Fatalf("Expected 10 buckets, got %d", initialCount)
	}

	// Manually set lastRefill to old time
	rl.mu.Lock()
	for _, bucket := range rl.buckets {
		bucket.mu.Lock()
		bucket.lastRefill = time.Now().Add(-15 * time.Minute)
		bucket.mu.Unlock()
	}
	rl.mu.Unlock()

	// Run cleanup
	rl.Cleanup()

	// Verify buckets are removed
	rl.mu.RLock()
	finalCount := len(rl.buckets)
	rl.mu.RUnlock()

	if finalCount != 0 {
		t.Fatalf("Expected 0 buckets after cleanup, got %d", finalCount)
	}
}

// Test StartCleanup background goroutine
// Requirements: 6.2
func TestRateLimiter_StartCleanup(t *testing.T) {
	rl := NewRateLimiter(10)

	// Create a bucket
	rl.Allow("192.168.1.1")

	// Start cleanup with short interval
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Override cleanup interval for testing
	rl.mu.Lock()
	for _, bucket := range rl.buckets {
		bucket.mu.Lock()
		bucket.lastRefill = time.Now().Add(-15 * time.Minute)
		bucket.mu.Unlock()
	}
	rl.mu.Unlock()

	// Start cleanup
	rl.StartCleanup(ctx)

	// Wait a bit for cleanup to run
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop cleanup
	cancel()

	// Note: This test just verifies the cleanup goroutine starts and stops
	// without panicking. Full cleanup testing is done in TestRateLimiter_Cleanup
}

// Test extractIPFromHTTPRequest with X-Forwarded-For
// Requirements: 6.1
func TestExtractIPFromHTTPRequest_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := extractIPFromHTTPRequest(req)
	if ip != "203.0.113.1" {
		t.Fatalf("Expected IP from X-Forwarded-For, got %s", ip)
	}
}

// Test extractIPFromHTTPRequest with X-Real-IP
// Requirements: 6.1
func TestExtractIPFromHTTPRequest_XRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "203.0.113.2")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := extractIPFromHTTPRequest(req)
	if ip != "203.0.113.2" {
		t.Fatalf("Expected IP from X-Real-IP, got %s", ip)
	}
}

// Test extractIPFromHTTPRequest with RemoteAddr fallback
// Requirements: 6.1
func TestExtractIPFromHTTPRequest_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := extractIPFromHTTPRequest(req)
	if ip != "192.168.1.1:12345" {
		t.Fatalf("Expected IP from RemoteAddr, got %s", ip)
	}
}

// Mock address for testing
type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.addr
}
