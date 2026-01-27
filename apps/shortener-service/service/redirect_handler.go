package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pingxin403/cuckoo/apps/shortener-service/analytics"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
	"github.com/pingxin403/cuckoo/libs/observability"
)

// RedirectHandler handles HTTP redirect requests
// Requirements: 3.1, 3.5, 5.2, 14.4
type RedirectHandler struct {
	cacheManager    *cache.CacheManager
	storage         storage.Storage
	analyticsWriter *analytics.AnalyticsWriter
	obs             observability.Observability
}

// NewRedirectHandler creates a new RedirectHandler
func NewRedirectHandler(cacheManager *cache.CacheManager, storage storage.Storage, analyticsWriter *analytics.AnalyticsWriter, obs observability.Observability) *RedirectHandler {
	return &RedirectHandler{
		cacheManager:    cacheManager,
		storage:         storage,
		analyticsWriter: analyticsWriter,
		obs:             obs,
	}
}

// SetupRouter sets up the HTTP router with redirect handler
// Requirements: 3.1, 3.5, 5.2, 14.4
func (h *RedirectHandler) SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Health check endpoints
	r.Get("/health", h.HealthCheck)
	r.Get("/ready", h.ReadinessCheck)

	// Redirect handler - catch-all route for short codes
	r.Get("/{code}", h.HandleRedirect)

	return r
}

// HandleRedirect handles the redirect request for a short code
// Requirements: 3.1, 3.5, 5.2, 14.4
func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "code")

	// Validate short code
	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	// Get mapping from cache/storage
	ctx := r.Context()
	mapping, err := h.storage.Get(ctx, shortCode)
	if err != nil {
		if err == storage.ErrNotFound {
			// Requirements: 3.5 - Return 404 for non-existent codes
			h.obs.Metrics().IncrementCounter("shortener_errors_total", map[string]string{"type": "redirect_not_found"})
			http.Error(w, "Short code not found", http.StatusNotFound)
			return
		}
		// Internal error
		h.obs.Metrics().IncrementCounter("shortener_errors_total", map[string]string{"type": "redirect_error"})
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if expired
	// Requirements: 5.2 - Return 410 for expired codes
	isExpired := mapping.ExpiresAt != nil && time.Now().After(*mapping.ExpiresAt)
	if isExpired {
		h.obs.Metrics().IncrementCounter("shortener_errors_total", map[string]string{"type": "redirect_expired"})
		http.Error(w, "Short link has expired", http.StatusGone)
		return
	}

	// Log click event asynchronously
	// Requirements: 7.1, 7.2 - Async click logging
	if h.analyticsWriter != nil {
		go func() {
			event := analytics.ClickEvent{
				ShortCode: shortCode,
				Timestamp: time.Now(),
				SourceIP:  extractIPFromRequest(r),
				UserAgent: r.UserAgent(),
				Referer:   r.Referer(),
			}
			h.analyticsWriter.LogClick(event)
		}()
	}

	// Set security headers
	// Requirements: 14.4 - Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "no-referrer")

	// Perform redirect
	// Requirements: 3.1 - Return HTTP 302 redirect
	h.obs.Metrics().IncrementCounter("shortener_redirects_total", nil)
	http.Redirect(w, r, mapping.LongURL, http.StatusFound)
}

// extractIPFromRequest extracts the client IP from HTTP request
func extractIPFromRequest(r *http.Request) string {
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

// HealthCheck handles liveness probe
func (h *RedirectHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}

// ReadinessCheck handles readiness probe
func (h *RedirectHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check storage connectivity
	_, err := h.storage.Get(ctx, "health-check")
	if err != nil && err != storage.ErrNotFound {
		http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "Ready")
}
