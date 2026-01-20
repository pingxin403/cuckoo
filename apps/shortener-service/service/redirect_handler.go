package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/metrics"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// RedirectHandler handles HTTP redirect requests
// Requirements: 3.1, 3.5, 5.2, 14.4
type RedirectHandler struct {
	cacheManager *cache.CacheManager
	storage      storage.Storage
}

// NewRedirectHandler creates a new RedirectHandler
func NewRedirectHandler(cacheManager *cache.CacheManager, storage storage.Storage) *RedirectHandler {
	return &RedirectHandler{
		cacheManager: cacheManager,
		storage:      storage,
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
			metrics.ErrorsTotal.WithLabelValues("redirect_not_found").Inc()
			http.Error(w, "Short code not found", http.StatusNotFound)
			return
		}
		// Internal error
		metrics.ErrorsTotal.WithLabelValues("redirect_error").Inc()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if expired
	// Requirements: 5.2 - Return 410 for expired codes
	isExpired := mapping.ExpiresAt != nil && time.Now().After(*mapping.ExpiresAt)
	if isExpired {
		metrics.ErrorsTotal.WithLabelValues("redirect_expired").Inc()
		http.Error(w, "Short link has expired", http.StatusGone)
		return
	}

	// Set security headers
	// Requirements: 14.4 - Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "no-referrer")

	// Perform redirect
	// Requirements: 3.1 - Return HTTP 302 redirect
	metrics.RedirectsTotal.Inc()
	http.Redirect(w, r, mapping.LongURL, http.StatusFound)
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
