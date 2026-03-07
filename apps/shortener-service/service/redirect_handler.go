package service

import (
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
func (h *RedirectHandler) SetupRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	// Health check endpoints (must be registered before catch-all)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ready"))
	})

	// Redirect handler - catch-all route for short codes
	r.Get("/{code}", h.HandleRedirect)

	return r
}

// HandleRedirect handles the redirect request for a short code
func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "code")

	// Validate short code
	if shortCode == "" {
		http.Error(w, "Short code is required", http.StatusBadRequest)
		return
	}

	// Get mapping from cache/storage with multi-tier fallback
	ctx := r.Context()

	// Try cache first (if available), then fallback to storage
	var mapping *storage.URLMapping
	var err error

	if h.cacheManager != nil {
		// Use cache manager for multi-tier cache lookup (L1 → L2 → DB with backfill)
		cacheMapping, cacheErr := h.cacheManager.Get(ctx, shortCode)
		if cacheErr != nil {
			// Cache error, fallback to direct storage query
			h.obs.Logger().Warn(ctx, "Cache lookup failed, falling back to storage",
				"short_code", shortCode,
				"error", cacheErr)
			mapping, err = h.storage.Get(ctx, shortCode)
		} else if cacheMapping == nil {
			// Not found in cache or storage
			err = storage.ErrNotFound
		} else {
			// Cache hit - convert cache mapping to storage mapping
			mapping = &storage.URLMapping{
				ShortCode: cacheMapping.ShortCode,
				LongURL:   cacheMapping.LongURL,
				CreatedAt: cacheMapping.CreatedAt,
				ExpiresAt: cacheMapping.ExpiresAt,
			}
		}
	} else {
		// No cache manager, query storage directly
		mapping, err = h.storage.Get(ctx, shortCode)
	}

	if err != nil {
		if err == storage.ErrNotFound {
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
	isExpired := mapping.ExpiresAt != nil && time.Now().After(*mapping.ExpiresAt)
	if isExpired {
		h.obs.Metrics().IncrementCounter("shortener_errors_total", map[string]string{"type": "redirect_expired"})
		http.Error(w, "Short link has expired", http.StatusGone)
		return
	}

	// Log click event asynchronously
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
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "no-referrer")

	// Perform redirect
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
