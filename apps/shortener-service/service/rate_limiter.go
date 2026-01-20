package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter implements per-IP token bucket rate limiting
// Requirements: 6.1, 6.2, 6.5
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*TokenBucket
	limit   int           // tokens per minute
	refill  time.Duration // refill interval
}

// TokenBucket represents a token bucket for a single IP
type TokenBucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// Requirements: 6.1, 6.2
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		limit:   requestsPerMinute,
		refill:  time.Minute,
	}
}

// Allow checks if a request from the given IP should be allowed
// Requirements: 6.1, 6.2
func (rl *RateLimiter) Allow(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &TokenBucket{
			tokens:     rl.limit,
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= rl.refill {
		bucket.tokens = rl.limit
		bucket.lastRefill = now
	}

	// Check if tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true, 0
	}

	// Calculate retry-after duration
	retryAfter := rl.refill - elapsed
	return false, retryAfter
}

// UnaryServerInterceptor returns a gRPC unary interceptor for rate limiting
// Requirements: 6.1, 6.2, 6.5
func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract IP from context
		ip := extractIPFromGRPCContext(ctx)

		// Check rate limit
		allowed, retryAfter := rl.Allow(ip)
		if !allowed {
			// Requirements: 6.5 - Return 429 with Retry-After
			return nil, status.Errorf(
				codes.ResourceExhausted,
				"Rate limit exceeded. Retry after %v seconds",
				int(retryAfter.Seconds()),
			)
		}

		// Continue with the request
		return handler(ctx, req)
	}
}

// HTTPMiddleware returns an HTTP middleware for rate limiting
// Requirements: 6.1, 6.2, 6.5
func (rl *RateLimiter) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP from request
		ip := extractIPFromHTTPRequest(r)

		// Check rate limit
		allowed, retryAfter := rl.Allow(ip)
		if !allowed {
			// Requirements: 6.5 - Return HTTP 429 with Retry-After header
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Continue with the request
		next.ServeHTTP(w, r)
	})
}

// extractIPFromGRPCContext extracts the client IP from gRPC context
func extractIPFromGRPCContext(ctx context.Context) string {
	// Try to get IP from peer info
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	// Try to get IP from metadata (X-Forwarded-For header)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			return xff[0]
		}
	}

	return "unknown"
}

// extractIPFromHTTPRequest extracts the client IP from HTTP request
func extractIPFromHTTPRequest(r *http.Request) string {
	// Try X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Cleanup removes old buckets to prevent memory leaks
// Should be called periodically in a background goroutine
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, bucket := range rl.buckets {
		bucket.mu.Lock()
		// Remove buckets that haven't been used in the last 10 minutes
		if now.Sub(bucket.lastRefill) > 10*time.Minute {
			delete(rl.buckets, ip)
		}
		bucket.mu.Unlock()
	}
}

// StartCleanup starts a background goroutine to periodically clean up old buckets
func (rl *RateLimiter) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rl.Cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
}
