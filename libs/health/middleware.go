package health

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// ReadinessMiddleware returns HTTP middleware that rejects requests when the service is not ready.
// It performs a lock-free atomic check on the readiness status and returns 503 Service Unavailable
// when the service is not ready to handle traffic.
//
// This middleware should be applied to all application endpoints (but not health check endpoints).
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api/users", usersHandler)
//	
//	// Wrap with readiness middleware
//	handler := health.ReadinessMiddleware(hc)(mux)
//	http.ListenAndServe(":8080", handler)
func ReadinessMiddleware(hc *HealthChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Lock-free readiness check using atomic operation
			if !hc.IsReady() {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("Service not ready"))
				
				// Log rejected request
				if hc.obs != nil {
					hc.obs.Logger().Debug(r.Context(), "Request rejected - service not ready",
						"path", r.URL.Path,
						"method", r.Method,
						"remote_addr", r.RemoteAddr,
					)
				}
				return
			}
			
			// Service is ready, proceed with request
			next.ServeHTTP(w, r)
		})
	}
}

// GracefulShutdown manages graceful shutdown of the service by tracking in-flight requests
// and ensuring they complete before shutdown. It integrates with the health checker to
// mark the service as not ready during shutdown.
//
// Example:
//
//	gs := health.NewGracefulShutdown(hc, 30*time.Second)
//	
//	// Wrap handlers with middleware
//	handler := gs.Middleware(mux)
//	
//	// Start server
//	server := &http.Server{Addr: ":8080", Handler: handler}
//	go server.ListenAndServe()
//	
//	// On shutdown signal
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	gs.Shutdown(ctx)
//	server.Shutdown(ctx)
type GracefulShutdown struct {
	hc              *HealthChecker
	inFlightReqs    atomic.Int32
	shutdownTimeout time.Duration
	isShuttingDown  atomic.Int32 // 1 = shutting down, 0 = normal operation
}

// NewGracefulShutdown creates a new graceful shutdown manager.
//
// Parameters:
//   - hc: The health checker to integrate with
//   - shutdownTimeout: Maximum time to wait for in-flight requests to complete
//
// Example:
//
//	gs := health.NewGracefulShutdown(hc, 30*time.Second)
func NewGracefulShutdown(hc *HealthChecker, shutdownTimeout time.Duration) *GracefulShutdown {
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	
	return &GracefulShutdown{
		hc:              hc,
		shutdownTimeout: shutdownTimeout,
	}
}

// Middleware returns HTTP middleware that tracks in-flight requests and rejects
// new requests during shutdown.
//
// Example:
//
//	gs := health.NewGracefulShutdown(hc, 30*time.Second)
//	handler := gs.Middleware(mux)
func (gs *GracefulShutdown) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if shutting down
		if gs.isShuttingDown.Load() == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service is shutting down"))
			return
		}
		
		// Increment in-flight counter
		gs.inFlightReqs.Add(1)
		defer gs.inFlightReqs.Add(-1)
		
		// Process request
		next.ServeHTTP(w, r)
	})
}

// Shutdown initiates graceful shutdown by:
// 1. Marking the service as not ready (stops accepting new requests via readiness probe)
// 2. Waiting for in-flight requests to complete (up to shutdownTimeout)
// 3. Returning when all requests are complete or timeout is reached
//
// This method should be called before shutting down the HTTP server.
//
// Example:
//
//	// On shutdown signal (SIGTERM, SIGINT)
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	
//	if err := gs.Shutdown(ctx); err != nil {
//	    log.Printf("Graceful shutdown error: %v", err)
//	}
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	// Mark as shutting down
	gs.isShuttingDown.Store(1)
	
	// Mark service as not ready (stops Kubernetes from sending new traffic)
	gs.hc.readinessProbe.isReady.Store(0)
	
	if gs.hc.obs != nil {
		gs.hc.obs.Logger().Info(ctx, "Initiating graceful shutdown",
			"service", gs.hc.config.ServiceName,
			"in_flight_requests", gs.inFlightReqs.Load(),
			"timeout", gs.shutdownTimeout,
		)
	}
	
	// Wait for in-flight requests to complete
	deadline := time.Now().Add(gs.shutdownTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		inFlight := gs.inFlightReqs.Load()
		
		// All requests completed
		if inFlight == 0 {
			if gs.hc.obs != nil {
				gs.hc.obs.Logger().Info(ctx, "Graceful shutdown completed - all requests finished",
					"service", gs.hc.config.ServiceName,
				)
			}
			return nil
		}
		
		// Check timeout
		if time.Now().After(deadline) {
			err := fmt.Errorf("shutdown timeout: %d requests still in flight after %v",
				inFlight, gs.shutdownTimeout)
			
			if gs.hc.obs != nil {
				gs.hc.obs.Logger().Warn(ctx, "Graceful shutdown timeout",
					"service", gs.hc.config.ServiceName,
					"in_flight_requests", inFlight,
					"timeout", gs.shutdownTimeout,
				)
			}
			return err
		}
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			err := fmt.Errorf("shutdown cancelled: %d requests still in flight: %w",
				inFlight, ctx.Err())
			
			if gs.hc.obs != nil {
				gs.hc.obs.Logger().Warn(ctx, "Graceful shutdown cancelled",
					"service", gs.hc.config.ServiceName,
					"in_flight_requests", inFlight,
				)
			}
			return err
		case <-ticker.C:
			// Continue waiting
			if gs.hc.obs != nil {
				gs.hc.obs.Logger().Debug(ctx, "Waiting for in-flight requests",
					"service", gs.hc.config.ServiceName,
					"in_flight_requests", inFlight,
				)
			}
		}
	}
}

// InFlightRequests returns the current number of in-flight requests.
// This can be useful for monitoring and debugging.
//
// Example:
//
//	count := gs.InFlightRequests()
//	fmt.Printf("In-flight requests: %d\n", count)
func (gs *GracefulShutdown) InFlightRequests() int32 {
	return gs.inFlightReqs.Load()
}

// IsShuttingDown returns true if graceful shutdown has been initiated.
//
// Example:
//
//	if gs.IsShuttingDown() {
//	    // Shutdown in progress
//	}
func (gs *GracefulShutdown) IsShuttingDown() bool {
	return gs.isShuttingDown.Load() == 1
}
