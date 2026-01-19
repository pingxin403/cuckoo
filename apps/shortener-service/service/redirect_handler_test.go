package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// TestHandleRedirect_Success tests successful redirect
func TestHandleRedirect_Success(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create a mapping
	mapping := &storage.URLMapping{
		ShortCode: "test123",
		LongURL:   "https://example.com/destination",
		CreatedAt: time.Now(),
		CreatorIP: "127.0.0.1",
	}
	err = store.Create(context.Background(), mapping)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/test123", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify redirect
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/destination", w.Header().Get("Location"))

	// Verify security headers
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
}

// TestHandleRedirect_NotFound tests 404 for non-existent code
func TestHandleRedirect_NotFound(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create request for non-existent code
	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify 404
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

// TestHandleRedirect_Expired tests 410 for expired code
func TestHandleRedirect_Expired(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create an expired mapping
	expiredTime := time.Now().Add(-1 * time.Hour)
	mapping := &storage.URLMapping{
		ShortCode: "expired",
		LongURL:   "https://example.com/expired",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: &expiredTime,
		CreatorIP: "127.0.0.1",
	}
	err = store.Create(context.Background(), mapping)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/expired", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify 410 Gone
	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "expired")
}

// TestHandleRedirect_EmptyCode tests empty short code
func TestHandleRedirect_EmptyCode(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create request with empty code
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Should return 404 (no route matches)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestHandleRedirect_SecurityHeaders tests that all security headers are set
func TestHandleRedirect_SecurityHeaders(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create a mapping
	mapping := &storage.URLMapping{
		ShortCode: "secure",
		LongURL:   "https://example.com/secure",
		CreatedAt: time.Now(),
		CreatorIP: "127.0.0.1",
	}
	err = store.Create(context.Background(), mapping)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify all security headers are present
	assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, w.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, w.Header().Get("Referrer-Policy"))
}

// TestHealthCheck tests health check endpoint
func TestHealthCheck(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

// TestReadinessCheck tests readiness check endpoint
func TestReadinessCheck(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Ready", w.Body.String())
}

// TestHandleRedirect_NotExpired tests non-expired link
func TestHandleRedirect_NotExpired(t *testing.T) {
	store := NewMockStorage()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store})

	handler := NewRedirectHandler(cacheManager, store)

	// Create a mapping with future expiration
	futureTime := time.Now().Add(24 * time.Hour)
	mapping := &storage.URLMapping{
		ShortCode: "active",
		LongURL:   "https://example.com/active",
		CreatedAt: time.Now(),
		ExpiresAt: &futureTime,
		CreatorIP: "127.0.0.1",
	}
	err = store.Create(context.Background(), mapping)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/active", nil)
	w := httptest.NewRecorder()

	// Setup router and handle request
	router := handler.SetupRouter()
	router.ServeHTTP(w, req)

	// Verify redirect (not expired)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/active", w.Header().Get("Location"))
}
