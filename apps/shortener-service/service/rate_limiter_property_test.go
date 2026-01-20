package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property 10: Rate Limiting Token Bucket
// Validates: Requirements 6.1, 6.2, 6.5
// For any IP making rapid requests, the rate limiter should:
// 1. Allow up to the limit number of requests
// 2. Reject requests after the limit is exceeded
// 3. Provide a Retry-After duration when rejecting
func TestProperty_RateLimitingTokenBucket(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		requestsPerMinute := rapid.IntRange(10, 100).Draw(t, "requestsPerMinute")
		numRequests := rapid.IntRange(requestsPerMinute+1, requestsPerMinute*2).Draw(t, "numRequests")
		ip := rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "ip")

		// Create rate limiter
		rl := NewRateLimiter(requestsPerMinute)

		// Make rapid requests
		allowedCount := 0
		rejectedCount := 0
		var firstRetryAfter time.Duration

		for i := 0; i < numRequests; i++ {
			allowed, retryAfter := rl.Allow(ip)
			if allowed {
				allowedCount++
			} else {
				rejectedCount++
				if firstRetryAfter == 0 {
					firstRetryAfter = retryAfter
				}
			}
		}

		// Property 1: Exactly 'requestsPerMinute' requests should be allowed
		if allowedCount != requestsPerMinute {
			t.Fatalf("Expected %d allowed requests, got %d", requestsPerMinute, allowedCount)
		}

		// Property 2: Remaining requests should be rejected
		expectedRejected := numRequests - requestsPerMinute
		if rejectedCount != expectedRejected {
			t.Fatalf("Expected %d rejected requests, got %d", expectedRejected, rejectedCount)
		}

		// Property 3: Retry-After should be provided and be positive
		if firstRetryAfter <= 0 {
			t.Fatalf("Expected positive Retry-After duration, got %v", firstRetryAfter)
		}

		// Property 4: Retry-After should be less than or equal to refill interval
		if firstRetryAfter > time.Minute {
			t.Fatalf("Retry-After %v exceeds refill interval of 1 minute", firstRetryAfter)
		}
	})
}

// Property: Token bucket refills after the refill interval
// Validates: Requirements 6.2
func TestProperty_TokenBucketRefill(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		requestsPerMinute := rapid.IntRange(10, 50).Draw(t, "requestsPerMinute")
		ip := rapid.StringMatching(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).Draw(t, "ip")

		// Create rate limiter with short refill interval for testing
		rl := &RateLimiter{
			buckets: make(map[string]*TokenBucket),
			limit:   requestsPerMinute,
			refill:  100 * time.Millisecond, // Short interval for testing
		}

		// Exhaust all tokens
		for i := 0; i < requestsPerMinute; i++ {
			allowed, _ := rl.Allow(ip)
			if !allowed {
				t.Fatalf("Request %d should be allowed", i)
			}
		}

		// Next request should be rejected
		allowed, _ := rl.Allow(ip)
		if allowed {
			t.Fatal("Request should be rejected after exhausting tokens")
		}

		// Wait for refill
		time.Sleep(150 * time.Millisecond)

		// Tokens should be refilled
		allowed, _ = rl.Allow(ip)
		if !allowed {
			t.Fatal("Request should be allowed after refill")
		}
	})
}

// Property: Different IPs have independent rate limits
// Validates: Requirements 6.1
func TestProperty_IndependentIPRateLimits(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		requestsPerMinute := rapid.IntRange(10, 50).Draw(t, "requestsPerMinute")
		numIPs := rapid.IntRange(2, 5).Draw(t, "numIPs")

		// Generate unique IPs
		ips := make([]string, numIPs)
		for i := 0; i < numIPs; i++ {
			ips[i] = fmt.Sprintf("192.168.1.%d", i+1)
		}

		// Create rate limiter
		rl := NewRateLimiter(requestsPerMinute)

		// Each IP should be able to make 'requestsPerMinute' requests
		for _, ip := range ips {
			allowedCount := 0
			for i := 0; i < requestsPerMinute; i++ {
				allowed, _ := rl.Allow(ip)
				if allowed {
					allowedCount++
				}
			}

			if allowedCount != requestsPerMinute {
				t.Fatalf("IP %s: expected %d allowed requests, got %d", ip, requestsPerMinute, allowedCount)
			}
		}
	})
}

// Property: Concurrent requests are handled correctly
// Validates: Requirements 6.2
func TestProperty_ConcurrentRateLimiting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		requestsPerMinute := rapid.IntRange(50, 100).Draw(t, "requestsPerMinute")
		numGoroutines := rapid.IntRange(5, 20).Draw(t, "numGoroutines")
		requestsPerGoroutine := (requestsPerMinute / numGoroutines) + 1
		ip := "192.168.1.100"

		// Create rate limiter
		rl := NewRateLimiter(requestsPerMinute)

		// Make concurrent requests
		var wg sync.WaitGroup
		allowedCount := int32(0)
		rejectedCount := int32(0)
		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					allowed, _ := rl.Allow(ip)
					mu.Lock()
					if allowed {
						allowedCount++
					} else {
						rejectedCount++
					}
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// Property: Total allowed should not exceed the limit
		if int(allowedCount) > requestsPerMinute {
			t.Fatalf("Allowed count %d exceeds limit %d", allowedCount, requestsPerMinute)
		}

		// Property: Total allowed + rejected should equal total requests
		totalRequests := numGoroutines * requestsPerGoroutine
		if int(allowedCount+rejectedCount) != totalRequests {
			t.Fatalf("Total requests mismatch: allowed=%d, rejected=%d, expected=%d",
				allowedCount, rejectedCount, totalRequests)
		}
	})
}

// Property: Cleanup removes old buckets
// Validates: Requirements 6.2 (memory management)
func TestProperty_CleanupRemovesOldBuckets(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numIPs := rapid.IntRange(5, 20).Draw(t, "numIPs")

		// Create rate limiter
		rl := NewRateLimiter(100)

		// Create buckets for multiple IPs
		for i := 0; i < numIPs; i++ {
			ip := fmt.Sprintf("192.168.1.%d", i+1)
			rl.Allow(ip)
		}

		// Verify buckets exist
		rl.mu.RLock()
		initialCount := len(rl.buckets)
		rl.mu.RUnlock()

		if initialCount != numIPs {
			t.Fatalf("Expected %d buckets, got %d", numIPs, initialCount)
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
	})
}
