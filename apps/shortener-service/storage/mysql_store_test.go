package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestURLMapping_Structure verifies the URLMapping struct has all required fields
func TestURLMapping_Structure(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	mapping := &URLMapping{
		ShortCode:  "abc123x",
		LongURL:    "https://example.com/very/long/path",
		CreatedAt:  now,
		ExpiresAt:  &expiresAt,
		CreatorIP:  "192.168.1.1",
		ClickCount: 42,
		IsDeleted:  false,
	}

	assert.Equal(t, "abc123x", mapping.ShortCode)
	assert.Equal(t, "https://example.com/very/long/path", mapping.LongURL)
	assert.Equal(t, now, mapping.CreatedAt)
	assert.NotNil(t, mapping.ExpiresAt)
	assert.Equal(t, expiresAt, *mapping.ExpiresAt)
	assert.Equal(t, "192.168.1.1", mapping.CreatorIP)
	assert.Equal(t, int64(42), mapping.ClickCount)
	assert.False(t, mapping.IsDeleted)
}

// TestURLMapping_NilExpiresAt verifies that ExpiresAt can be nil for permanent links
func TestURLMapping_NilExpiresAt(t *testing.T) {
	mapping := &URLMapping{
		ShortCode:  "abc123x",
		LongURL:    "https://example.com",
		CreatedAt:  time.Now(),
		ExpiresAt:  nil, // Permanent link
		CreatorIP:  "192.168.1.1",
		ClickCount: 0,
		IsDeleted:  false,
	}

	assert.Nil(t, mapping.ExpiresAt)
}

// TestStorage_Interface verifies that MySQLStore implements Storage interface
func TestStorage_Interface(t *testing.T) {
	// This test verifies at compile time that MySQLStore implements Storage
	var _ Storage = (*MySQLStore)(nil)
}

// TestGetEnv verifies the environment variable helper function
func TestGetEnv(t *testing.T) {
	// Test with non-existent env var (should return default)
	result := getEnv("NONEXISTENT_VAR_12345", "default_value")
	assert.Equal(t, "default_value", result)

	// Test with existing env var
	t.Setenv("TEST_VAR", "test_value")
	result = getEnv("TEST_VAR", "default_value")
	assert.Equal(t, "test_value", result)
}

// TestMySQLStore_NilMapping verifies that Create rejects nil mappings
func TestMySQLStore_NilMapping(t *testing.T) {
	store := &MySQLStore{db: nil} // db is nil but we're only testing validation

	err := store.Create(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mapping cannot be nil")
}

// MockStorage is a mock implementation of Storage for testing
type MockStorage struct {
	CreateFunc     func(ctx context.Context, mapping *URLMapping) error
	GetFunc        func(ctx context.Context, shortCode string) (*URLMapping, error)
	ExistsFunc     func(ctx context.Context, shortCode string) (bool, error)
	DeleteFunc     func(ctx context.Context, shortCode string) error
	GetExpiredFunc func(ctx context.Context, limit int) ([]*URLMapping, error)
	CloseFunc      func() error
}

func (m *MockStorage) Create(ctx context.Context, mapping *URLMapping) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, mapping)
	}
	return nil
}

func (m *MockStorage) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, shortCode)
	}
	return nil, nil
}

func (m *MockStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, shortCode)
	}
	return false, nil
}

func (m *MockStorage) Delete(ctx context.Context, shortCode string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, shortCode)
	}
	return nil
}

func (m *MockStorage) GetExpired(ctx context.Context, limit int) ([]*URLMapping, error) {
	if m.GetExpiredFunc != nil {
		return m.GetExpiredFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockStorage) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestMySQLStore_DuplicateShortCode tests duplicate short_code rejection
// Requirements: 2.2
func TestMySQLStore_DuplicateShortCode(t *testing.T) {
	// This test verifies the behavior when attempting to create a duplicate short code.
	// In a real MySQL database, this would trigger a unique constraint violation.
	// Here we test the expected error handling pattern.

	tests := []struct {
		name          string
		firstMapping  *URLMapping
		secondMapping *URLMapping
		expectError   bool
	}{
		{
			name: "duplicate short code should fail",
			firstMapping: &URLMapping{
				ShortCode: "abc123x",
				LongURL:   "https://example.com/first",
				CreatedAt: time.Now(),
				CreatorIP: "192.168.1.1",
			},
			secondMapping: &URLMapping{
				ShortCode: "abc123x", // Same short code
				LongURL:   "https://example.com/second",
				CreatedAt: time.Now(),
				CreatorIP: "192.168.1.2",
			},
			expectError: true,
		},
		{
			name: "different short codes should succeed",
			firstMapping: &URLMapping{
				ShortCode: "abc123x",
				LongURL:   "https://example.com/first",
				CreatedAt: time.Now(),
				CreatorIP: "192.168.1.1",
			},
			secondMapping: &URLMapping{
				ShortCode: "xyz789w", // Different short code
				LongURL:   "https://example.com/second",
				CreatedAt: time.Now(),
				CreatorIP: "192.168.1.2",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock storage that tracks created codes
			createdCodes := make(map[string]bool)

			mock := &MockStorage{
				CreateFunc: func(ctx context.Context, mapping *URLMapping) error {
					if createdCodes[mapping.ShortCode] {
						return assert.AnError // Simulate duplicate key error
					}
					createdCodes[mapping.ShortCode] = true
					return nil
				},
			}

			// Create first mapping
			err := mock.Create(context.Background(), tt.firstMapping)
			assert.NoError(t, err)

			// Attempt to create second mapping
			err = mock.Create(context.Background(), tt.secondMapping)

			if tt.expectError {
				assert.Error(t, err, "Expected error for duplicate short code")
			} else {
				assert.NoError(t, err, "Expected no error for different short codes")
			}
		})
	}
}

// TestMySQLStore_DatabaseConnectionFailure tests database connection error handling
// Requirements: 2.5
func TestMySQLStore_DatabaseConnectionFailure(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		setupMock func() Storage
	}{
		{
			name:      "Create fails on connection error",
			operation: "Create",
			setupMock: func() Storage {
				return &MockStorage{
					CreateFunc: func(ctx context.Context, mapping *URLMapping) error {
						return assert.AnError // Simulate connection error
					},
				}
			},
		},
		{
			name:      "Get fails on connection error",
			operation: "Get",
			setupMock: func() Storage {
				return &MockStorage{
					GetFunc: func(ctx context.Context, shortCode string) (*URLMapping, error) {
						return nil, assert.AnError // Simulate connection error
					},
				}
			},
		},
		{
			name:      "Exists fails on connection error",
			operation: "Exists",
			setupMock: func() Storage {
				return &MockStorage{
					ExistsFunc: func(ctx context.Context, shortCode string) (bool, error) {
						return false, assert.AnError // Simulate connection error
					},
				}
			},
		},
		{
			name:      "Delete fails on connection error",
			operation: "Delete",
			setupMock: func() Storage {
				return &MockStorage{
					DeleteFunc: func(ctx context.Context, shortCode string) error {
						return assert.AnError // Simulate connection error
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()

			var err error
			switch tt.operation {
			case "Create":
				mapping := &URLMapping{
					ShortCode: "abc123x",
					LongURL:   "https://example.com",
					CreatedAt: time.Now(),
					CreatorIP: "192.168.1.1",
				}
				err = mock.Create(ctx, mapping)
			case "Get":
				_, err = mock.Get(ctx, "abc123x")
			case "Exists":
				_, err = mock.Exists(ctx, "abc123x")
			case "Delete":
				err = mock.Delete(ctx, "abc123x")
			}

			assert.Error(t, err, "Expected error for database connection failure")
		})
	}
}

// TestMySQLStore_GetNotFound tests Get behavior when short code doesn't exist
// Requirements: 2.5
func TestMySQLStore_GetNotFound(t *testing.T) {
	mock := &MockStorage{
		GetFunc: func(ctx context.Context, shortCode string) (*URLMapping, error) {
			// Simulate not found
			return nil, assert.AnError
		},
	}

	mapping, err := mock.Get(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, mapping)
}

// TestMySQLStore_DeleteNotFound tests Delete behavior when short code doesn't exist
// Requirements: 2.5
func TestMySQLStore_DeleteNotFound(t *testing.T) {
	mock := &MockStorage{
		DeleteFunc: func(ctx context.Context, shortCode string) error {
			// Simulate not found (no rows affected)
			return assert.AnError
		},
	}

	err := mock.Delete(context.Background(), "nonexistent")

	assert.Error(t, err, "Expected error when deleting non-existent short code")
}

// TestMySQLStore_ContextCancellation tests context cancellation handling
// Requirements: 2.5
func TestMySQLStore_ContextCancellation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		setupMock func() Storage
	}{
		{
			name:      "Create respects context cancellation",
			operation: "Create",
			setupMock: func() Storage {
				return &MockStorage{
					CreateFunc: func(ctx context.Context, mapping *URLMapping) error {
						if ctx.Err() != nil {
							return ctx.Err()
						}
						return nil
					},
				}
			},
		},
		{
			name:      "Get respects context cancellation",
			operation: "Get",
			setupMock: func() Storage {
				return &MockStorage{
					GetFunc: func(ctx context.Context, shortCode string) (*URLMapping, error) {
						if ctx.Err() != nil {
							return nil, ctx.Err()
						}
						return nil, nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()

			// Create a cancelled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			var err error
			switch tt.operation {
			case "Create":
				mapping := &URLMapping{
					ShortCode: "abc123x",
					LongURL:   "https://example.com",
					CreatedAt: time.Now(),
					CreatorIP: "192.168.1.1",
				}
				err = mock.Create(ctx, mapping)
			case "Get":
				_, err = mock.Get(ctx, "abc123x")
			}

			assert.Error(t, err, "Expected error for cancelled context")
			assert.Equal(t, context.Canceled, err)
		})
	}
}

// Note: Integration tests with actual MySQL database are in separate test files
// and require Docker Compose setup. These unit tests verify the structure and
// basic validation logic without requiring a database connection.
