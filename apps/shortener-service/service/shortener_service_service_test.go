package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingxin403/cuckoo/api/gen/go/shortenerpb"
	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/apps/shortener-service/idgen"
	"github.com/pingxin403/cuckoo/apps/shortener-service/storage"
)

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	mappings map[string]*storage.URLMapping
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		mappings: make(map[string]*storage.URLMapping),
	}
}

func (m *MockStorage) Create(ctx context.Context, mapping *storage.URLMapping) error {
	m.mappings[mapping.ShortCode] = mapping
	return nil
}

func (m *MockStorage) Get(ctx context.Context, shortCode string) (*storage.URLMapping, error) {
	if mapping, ok := m.mappings[shortCode]; ok {
		return mapping, nil
	}
	return nil, storage.ErrNotFound
}

func (m *MockStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	_, exists := m.mappings[shortCode]
	return exists, nil
}

func (m *MockStorage) Delete(ctx context.Context, shortCode string) error {
	if _, ok := m.mappings[shortCode]; !ok {
		return storage.ErrNotFound
	}
	delete(m.mappings, shortCode)
	return nil
}

func (m *MockStorage) GetExpired(ctx context.Context, limit int) ([]*storage.URLMapping, error) {
	return nil, nil
}

func (m *MockStorage) Close() error {
	return nil
}

// TestNewShortenerServiceImpl tests service creation
func TestNewShortenerServiceImpl(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	assert.NotNil(t, service)
	assert.NotNil(t, service.storage)
	assert.NotNil(t, service.idGen)
	assert.NotNil(t, service.validator)
	assert.NotNil(t, service.cacheManager)
}

// TestCreateShortLink_Success tests successful short link creation
func TestCreateShortLink_Success(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	req := &shortenerpb.CreateShortLinkRequest{
		LongUrl: "https://example.com/very/long/path",
	}

	resp, err := service.CreateShortLink(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.ShortCode)
	assert.Equal(t, 7, len(resp.ShortCode))
	assert.Contains(t, resp.ShortUrl, resp.ShortCode)
	assert.NotNil(t, resp.CreatedAt)
}

// TestCreateShortLink_InvalidURL tests rejection of invalid URLs
func TestCreateShortLink_InvalidURL(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	tests := []struct {
		name    string
		longURL string
	}{
		{"Invalid protocol", "ftp://example.com"},
		{"Malicious javascript", "javascript:alert('xss')"},
		{"Empty URL", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &shortenerpb.CreateShortLinkRequest{
				LongUrl: tt.longURL,
			}

			resp, err := service.CreateShortLink(context.Background(), req)

			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}

// TestCreateShortLink_CustomCode tests custom code creation
func TestCreateShortLink_CustomCode(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	req := &shortenerpb.CreateShortLinkRequest{
		LongUrl:    "https://example.com",
		CustomCode: "promo2024",
	}

	resp, err := service.CreateShortLink(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "promo2024", resp.ShortCode)
	assert.Contains(t, resp.ShortUrl, "promo2024")
}

// TestGetLinkInfo_Success tests successful link info retrieval
func TestGetLinkInfo_Success(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link first
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: "https://example.com",
	}
	createResp, err := service.CreateShortLink(context.Background(), createReq)
	require.NoError(t, err)

	// Get link info
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: createResp.ShortCode,
	}

	resp, err := service.GetLinkInfo(context.Background(), getReq)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, createResp.ShortCode, resp.ShortCode)
	assert.Equal(t, "https://example.com", resp.LongUrl)
	assert.False(t, resp.IsExpired)
}

// TestGetLinkInfo_NotFound tests link not found error
func TestGetLinkInfo_NotFound(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	req := &shortenerpb.GetLinkInfoRequest{
		ShortCode: "notfound",
	}

	resp, err := service.GetLinkInfo(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// TestGetLinkInfo_ExpiredLink tests expired link handling
func TestGetLinkInfo_ExpiredLink(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link with expiration in the past
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

	// Get link info
	req := &shortenerpb.GetLinkInfoRequest{
		ShortCode: "expired",
	}

	resp, err := service.GetLinkInfo(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "expired", resp.ShortCode)
	assert.Equal(t, "https://example.com/expired", resp.LongUrl)
	assert.True(t, resp.IsExpired, "Link should be marked as expired")
	assert.NotNil(t, resp.ExpiresAt)
}

// TestGetLinkInfo_NotExpiredLink tests non-expired link handling
func TestGetLinkInfo_NotExpiredLink(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link with expiration in the future
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

	// Get link info
	req := &shortenerpb.GetLinkInfoRequest{
		ShortCode: "active",
	}

	resp, err := service.GetLinkInfo(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "active", resp.ShortCode)
	assert.Equal(t, "https://example.com/active", resp.LongUrl)
	assert.False(t, resp.IsExpired, "Link should not be marked as expired")
	assert.NotNil(t, resp.ExpiresAt)
}

// TestGetLinkInfo_EmptyShortCode tests empty short code validation
func TestGetLinkInfo_EmptyShortCode(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	req := &shortenerpb.GetLinkInfoRequest{
		ShortCode: "",
	}

	resp, err := service.GetLinkInfo(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Short code cannot be empty")
}

// TestGetLinkInfo_WithClickCount tests link info with click count
func TestGetLinkInfo_WithClickCount(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link with click count
	mapping := &storage.URLMapping{
		ShortCode:  "popular",
		LongURL:    "https://example.com/popular",
		CreatedAt:  time.Now(),
		CreatorIP:  "127.0.0.1",
		ClickCount: 42,
	}
	err = store.Create(context.Background(), mapping)
	require.NoError(t, err)

	// Get link info
	req := &shortenerpb.GetLinkInfoRequest{
		ShortCode: "popular",
	}

	resp, err := service.GetLinkInfo(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "popular", resp.ShortCode)
	assert.Equal(t, int64(42), resp.ClickCount)
}

// TestDeleteShortLink_Success tests successful link deletion
func TestDeleteShortLink_Success(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link first
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: "https://example.com",
	}
	createResp, err := service.CreateShortLink(context.Background(), createReq)
	require.NoError(t, err)

	// Delete the link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: createResp.ShortCode,
	}

	resp, err := service.DeleteShortLink(context.Background(), deleteReq)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)

	// Verify it's deleted
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: createResp.ShortCode,
	}
	_, err = service.GetLinkInfo(context.Background(), getReq)
	assert.Error(t, err)
}

// TestDeleteShortLink_NotFound tests deletion of non-existent link
func TestDeleteShortLink_NotFound(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Try to delete non-existent link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: "notfound",
	}

	resp, err := service.DeleteShortLink(context.Background(), deleteReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")
}

// TestDeleteShortLink_EmptyShortCode tests deletion with empty short code
func TestDeleteShortLink_EmptyShortCode(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Try to delete with empty short code
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: "",
	}

	resp, err := service.DeleteShortLink(context.Background(), deleteReq)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Short code cannot be empty")
}

// TestDeleteShortLink_CacheInvalidation tests that cache is invalidated on deletion
func TestDeleteShortLink_CacheInvalidation(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Create a link
	createReq := &shortenerpb.CreateShortLinkRequest{
		LongUrl: "https://example.com/cached",
	}
	createResp, err := service.CreateShortLink(context.Background(), createReq)
	require.NoError(t, err)

	// Verify it's in cache by getting it (this will populate cache)
	getReq := &shortenerpb.GetLinkInfoRequest{
		ShortCode: createResp.ShortCode,
	}
	_, err = service.GetLinkInfo(context.Background(), getReq)
	require.NoError(t, err)

	// Delete the link
	deleteReq := &shortenerpb.DeleteShortLinkRequest{
		ShortCode: createResp.ShortCode,
	}
	resp, err := service.DeleteShortLink(context.Background(), deleteReq)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify it's deleted from storage
	_, err = service.GetLinkInfo(context.Background(), getReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// mockCacheStorage adapts MockStorage to cache.Storage interface
type mockCacheStorage struct {
	store *MockStorage
}

func (m *mockCacheStorage) Get(ctx context.Context, shortCode string) (*cache.StorageMapping, error) {
	mapping, err := m.store.Get(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	return &cache.StorageMapping{
		ShortCode: mapping.ShortCode,
		LongURL:   mapping.LongURL,
		CreatedAt: mapping.CreatedAt,
		ExpiresAt: mapping.ExpiresAt,
		CreatorIP: mapping.CreatorIP,
	}, nil
}

func (m *mockCacheStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	return m.store.Exists(ctx, shortCode)
}

// TestCreateShortLink_ValidationErrors tests various validation error scenarios
func TestCreateShortLink_ValidationErrors(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	tests := []struct {
		name        string
		longURL     string
		expectedErr string
	}{
		{
			name:        "FTP protocol",
			longURL:     "ftp://example.com/file.txt",
			expectedErr: "Invalid protocol",
		},
		{
			name:        "JavaScript protocol",
			longURL:     "javascript:alert('xss')",
			expectedErr: "Invalid protocol",
		},
		{
			name:        "Data URI",
			longURL:     "data:text/html,<script>alert('xss')</script>",
			expectedErr: "Invalid protocol",
		},
		{
			name:        "File protocol",
			longURL:     "file:///etc/passwd",
			expectedErr: "Invalid protocol",
		},
		{
			name:        "Empty URL",
			longURL:     "",
			expectedErr: "Invalid URL",
		},
		{
			name:        "Whitespace only",
			longURL:     "   ",
			expectedErr: "Invalid URL",
		},
		{
			name:        "URL too long",
			longURL:     "https://example.com/" + string(make([]byte, 2100)),
			expectedErr: "URL too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &shortenerpb.CreateShortLinkRequest{
				LongUrl: tt.longURL,
			}

			resp, err := service.CreateShortLink(context.Background(), req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestCreateShortLink_StorageErrors tests storage error scenarios
func TestCreateShortLink_StorageErrors(t *testing.T) {
	t.Run("Storage create error", func(t *testing.T) {
		// Create a mock storage that returns an error on Create
		errorStore := &errorMockStorage{
			MockStorage: NewMockStorage(),
			createError: fmt.Errorf("database connection failed"),
		}

		idGen := idgen.NewRandomIDGenerator(errorStore)
		validator := NewURLValidator()
		l1, err := cache.NewL1Cache()
		require.NoError(t, err)
		cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: errorStore.MockStorage}, createTestObservability())

		service := NewShortenerServiceImpl(errorStore, idGen, validator, cacheManager, createTestObservability())

		req := &shortenerpb.CreateShortLinkRequest{
			LongUrl: "https://example.com",
		}

		resp, err := service.CreateShortLink(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "Failed to create mapping")
	})
}

// errorMockStorage is a mock storage that returns errors
type errorMockStorage struct {
	*MockStorage
	createError error
}

func (e *errorMockStorage) Create(ctx context.Context, mapping *storage.URLMapping) error {
	if e.createError != nil {
		return e.createError
	}
	return e.MockStorage.Create(ctx, mapping)
}

// TestCreateShortLink_CustomCodeConflict tests custom code conflict scenario
func TestCreateShortLink_CustomCodeConflict(t *testing.T) {
	store := NewMockStorage()

	// Pre-populate with a mapping using custom code
	existingCode := "promo2024"
	_ = store.Create(context.Background(), &storage.URLMapping{
		ShortCode: existingCode,
		LongURL:   "https://existing.com",
		CreatedAt: time.Now(),
		CreatorIP: "127.0.0.1",
	})

	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	// Try to create with the same custom code
	req := &shortenerpb.CreateShortLinkRequest{
		LongUrl:    "https://newurl.com",
		CustomCode: existingCode,
	}

	resp, err := service.CreateShortLink(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "already in use")
}

// TestCreateShortLink_InvalidCustomCode tests invalid custom code scenarios
func TestCreateShortLink_InvalidCustomCode(t *testing.T) {
	store := NewMockStorage()
	idGen := idgen.NewRandomIDGenerator(store)
	validator := NewURLValidator()
	l1, err := cache.NewL1Cache()
	require.NoError(t, err)
	cacheManager := cache.NewCacheManager(l1, nil, &mockCacheStorage{store: store}, createTestObservability())

	service := NewShortenerServiceImpl(store, idGen, validator, cacheManager, createTestObservability())

	tests := []struct {
		name       string
		customCode string
	}{
		{"Too short", "abc"},
		{"Too long", "this-is-a-very-long-custom-code-that-exceeds-limit"},
		{"Invalid characters", "test@code"},
		{"Reserved keyword", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &shortenerpb.CreateShortLinkRequest{
				LongUrl:    "https://example.com",
				CustomCode: tt.customCode,
			}

			resp, err := service.CreateShortLink(context.Background(), req)

			assert.Error(t, err)
			assert.Nil(t, resp)
		})
	}
}

// deterministicIDGenerator is a test helper that always returns the same code
// nolint:unused // Reserved for future test use
type deterministicIDGenerator struct {
	code string
}

// nolint:unused // Reserved for future test use
func (d *deterministicIDGenerator) Generate(ctx context.Context) (string, error) {
	return d.code, nil
}

// nolint:unused // Reserved for future test use
func (d *deterministicIDGenerator) ValidateCustomCode(ctx context.Context, code string) error {
	// Simple validation for testing
	if len(code) < 4 || len(code) > 20 {
		return idgen.ErrInvalidCustomCode
	}
	// Check if exists
	return nil
}
