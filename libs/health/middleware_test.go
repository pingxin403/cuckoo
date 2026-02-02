package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadinessMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		isReady        bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "service ready - request passes through",
			isReady:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "service not ready - request rejected",
			isReady:        false,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Service not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:    "test-service",
				CheckInterval:  1 * time.Second,
				DefaultTimeout: 100 * time.Millisecond,
			}, nil)

			// Set readiness state
			if tt.isReady {
				hc.readinessProbe.isReady.Store(1)
			} else {
				hc.readinessProbe.isReady.Store(0)
			}

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// Wrap with middleware
			middleware := ReadinessMiddleware(hc)
			wrappedHandler := middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestGracefulShutdown_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		isShuttingDown bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "normal operation - request passes through",
			isShuttingDown: false,
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "shutting down - request rejected",
			isShuttingDown: true,
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Service is shutting down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:    "test-service",
				CheckInterval:  1 * time.Second,
				DefaultTimeout: 100 * time.Millisecond,
			}, nil)

			// Create graceful shutdown
			gs := NewGracefulShutdown(hc, 5*time.Second)

			// Set shutdown state
			if tt.isShuttingDown {
				gs.isShuttingDown.Store(1)
			}

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// Wrap with middleware
			wrappedHandler := gs.Middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestGracefulShutdown_InFlightTracking(t *testing.T) {
	// Create health checker
	hc := NewHealthChecker(Config{
		ServiceName:    "test-service",
		CheckInterval:  1 * time.Second,
		DefaultTimeout: 100 * time.Millisecond,
	}, nil)

	// Create graceful shutdown
	gs := NewGracefulShutdown(hc, 5*time.Second)

	// Create slow handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrappedHandler := gs.Middleware(handler)

	// Start request in background
	done := make(chan bool)
	go func() {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
		done <- true
	}()

	// Wait for request to start
	time.Sleep(10 * time.Millisecond)

	// Verify in-flight count
	if gs.InFlightRequests() != 1 {
		t.Errorf("expected 1 in-flight request, got %d", gs.InFlightRequests())
	}

	// Wait for request to complete
	<-done

	// Verify in-flight count is back to 0
	if gs.InFlightRequests() != 0 {
		t.Errorf("expected 0 in-flight requests, got %d", gs.InFlightRequests())
	}
}

func TestGracefulShutdown_Shutdown(t *testing.T) {
	tests := []struct {
		name           string
		requestDelay   time.Duration
		shutdownTimeout time.Duration
		expectError    bool
	}{
		{
			name:           "requests complete before timeout",
			requestDelay:   50 * time.Millisecond,
			shutdownTimeout: 1 * time.Second,
			expectError:    false,
		},
		{
			name:           "requests timeout",
			requestDelay:   500 * time.Millisecond,
			shutdownTimeout: 100 * time.Millisecond,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create health checker
			hc := NewHealthChecker(Config{
				ServiceName:    "test-service",
				CheckInterval:  1 * time.Second,
				DefaultTimeout: 100 * time.Millisecond,
			}, nil)

			// Create graceful shutdown
			gs := NewGracefulShutdown(hc, tt.shutdownTimeout)

			// Create slow handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.requestDelay)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("success"))
			})

			// Wrap with middleware
			wrappedHandler := gs.Middleware(handler)

			// Start request in background
			go func() {
				req := httptest.NewRequest("GET", "/test", nil)
				rec := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(rec, req)
			}()

			// Wait for request to start
			time.Sleep(10 * time.Millisecond)

			// Initiate shutdown
			ctx := context.Background()
			err := gs.Shutdown(ctx)

			// Verify result
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			// Verify service is marked as not ready
			if hc.IsReady() {
				t.Error("expected service to be marked as not ready")
			}

			// Verify shutdown flag is set
			if !gs.IsShuttingDown() {
				t.Error("expected IsShuttingDown to be true")
			}
		})
	}
}

func TestGracefulShutdown_ShutdownWithContext(t *testing.T) {
	// Create health checker
	hc := NewHealthChecker(Config{
		ServiceName:    "test-service",
		CheckInterval:  1 * time.Second,
		DefaultTimeout: 100 * time.Millisecond,
	}, nil)

	// Create graceful shutdown with long timeout
	gs := NewGracefulShutdown(hc, 10*time.Second)

	// Create slow handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrappedHandler := gs.Middleware(handler)

	// Start request in background
	go func() {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}()

	// Wait for request to start
	time.Sleep(10 * time.Millisecond)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Initiate shutdown
	err := gs.Shutdown(ctx)

	// Should get context deadline exceeded error
	if err == nil {
		t.Error("expected error due to context cancellation, got nil")
	}
}

func TestNewGracefulShutdown(t *testing.T) {
	hc := NewHealthChecker(Config{
		ServiceName:    "test-service",
		CheckInterval:  1 * time.Second,
		DefaultTimeout: 100 * time.Millisecond,
	}, nil)

	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "custom timeout",
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "zero timeout uses default",
			timeout:         0,
			expectedTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGracefulShutdown(hc, tt.timeout)

			if gs.shutdownTimeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, gs.shutdownTimeout)
			}

			if gs.hc != hc {
				t.Error("health checker not set correctly")
			}

			if gs.InFlightRequests() != 0 {
				t.Errorf("expected 0 in-flight requests, got %d", gs.InFlightRequests())
			}

			if gs.IsShuttingDown() {
				t.Error("expected IsShuttingDown to be false")
			}
		})
	}
}
