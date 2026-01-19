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

// Note: Integration tests with actual MySQL database are in separate test files
// and require Docker Compose setup. These unit tests verify the structure and
// basic validation logic without requiring a database connection.
