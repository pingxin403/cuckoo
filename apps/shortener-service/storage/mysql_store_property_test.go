//go:build property
// +build property

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// InMemoryStorage is a simple in-memory implementation for property testing
type InMemoryStorage struct {
	mappings map[string]*URLMapping
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		mappings: make(map[string]*URLMapping),
	}
}

func (s *InMemoryStorage) Create(ctx context.Context, mapping *URLMapping) error {
	if mapping == nil {
		return assert.AnError
	}
	if _, exists := s.mappings[mapping.ShortCode]; exists {
		return assert.AnError // Duplicate
	}
	// Deep copy to avoid mutation issues
	copied := *mapping
	if mapping.ExpiresAt != nil {
		expiresAt := *mapping.ExpiresAt
		copied.ExpiresAt = &expiresAt
	}
	s.mappings[mapping.ShortCode] = &copied
	return nil
}

func (s *InMemoryStorage) Get(ctx context.Context, shortCode string) (*URLMapping, error) {
	mapping, exists := s.mappings[shortCode]
	if !exists || mapping.IsDeleted {
		return nil, assert.AnError
	}
	// Deep copy to avoid mutation issues
	copied := *mapping
	if mapping.ExpiresAt != nil {
		expiresAt := *mapping.ExpiresAt
		copied.ExpiresAt = &expiresAt
	}
	return &copied, nil
}

func (s *InMemoryStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	_, exists := s.mappings[shortCode]
	return exists, nil
}

func (s *InMemoryStorage) Delete(ctx context.Context, shortCode string) error {
	mapping, exists := s.mappings[shortCode]
	if !exists {
		return assert.AnError
	}
	mapping.IsDeleted = true
	return nil
}

func (s *InMemoryStorage) GetExpired(ctx context.Context, limit int) ([]*URLMapping, error) {
	var expired []*URLMapping
	now := time.Now()
	for _, mapping := range s.mappings {
		if mapping.ExpiresAt != nil && mapping.ExpiresAt.Before(now) && !mapping.IsDeleted {
			expired = append(expired, mapping)
			if len(expired) >= limit {
				break
			}
		}
	}
	return expired, nil
}

func (s *InMemoryStorage) Close() error {
	return nil
}

// TestProperty_CreateRetrieveConsistency tests Property 3: Create-Then-Retrieve Consistency
// Feature: url-shortener-service, Property 3: Create-Then-Retrieve Consistency
// For any successfully created URL mapping, immediately retrieving it by short code
// SHALL return the same long URL
// Requirements: 2.1, 13.2
func TestProperty_CreateRetrieveConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		storage := NewInMemoryStorage()
		ctx := context.Background()

		// Generate random URL mapping
		shortCode := rapid.StringMatching(`^[0-9a-zA-Z]{7}$`).Draw(t, "shortCode")
		longURL := "https://" + rapid.StringN(5, 50, -1).Draw(t, "domain") + ".com/" + rapid.StringN(0, 100, -1).Draw(t, "path")
		creatorIP := rapid.StringMatching(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(t, "ip")

		mapping := &URLMapping{
			ShortCode:  shortCode,
			LongURL:    longURL,
			CreatedAt:  time.Now(),
			ExpiresAt:  nil, // No expiration for this test
			CreatorIP:  creatorIP,
			ClickCount: 0,
			IsDeleted:  false,
		}

		// Create the mapping
		err := storage.Create(ctx, mapping)
		require.NoError(t, err, "Create should succeed")

		// Immediately retrieve it
		retrieved, err := storage.Get(ctx, shortCode)
		require.NoError(t, err, "Get should succeed")
		require.NotNil(t, retrieved, "Retrieved mapping should not be nil")

		// Verify consistency
		assert.Equal(t, mapping.ShortCode, retrieved.ShortCode, "ShortCode should match")
		assert.Equal(t, mapping.LongURL, retrieved.LongURL, "LongURL should match")
		assert.Equal(t, mapping.CreatorIP, retrieved.CreatorIP, "CreatorIP should match")
		assert.Equal(t, mapping.IsDeleted, retrieved.IsDeleted, "IsDeleted should match")

		// Verify timestamps are preserved (within 1 second tolerance for any rounding)
		assert.WithinDuration(t, mapping.CreatedAt, retrieved.CreatedAt, time.Second, "CreatedAt should be preserved")
	})
}

// TestProperty_RequiredFieldsCompleteness tests Property 4: Required Fields Completeness
// Feature: url-shortener-service, Property 4: Required Fields Completeness
// For any created URL mapping, it SHALL have all required fields populated:
// short_code, long_url, created_at, and creator_ip
// Requirements: 2.3
func TestProperty_RequiredFieldsCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		storage := NewInMemoryStorage()
		ctx := context.Background()

		// Generate random URL mapping with all required fields
		shortCode := rapid.StringMatching(`^[0-9a-zA-Z]{7}$`).Draw(t, "shortCode")
		longURL := "https://" + rapid.StringN(5, 50, -1).Draw(t, "domain") + ".com"
		creatorIP := rapid.StringMatching(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(t, "ip")
		createdAt := time.Now()

		mapping := &URLMapping{
			ShortCode:  shortCode,
			LongURL:    longURL,
			CreatedAt:  createdAt,
			ExpiresAt:  nil,
			CreatorIP:  creatorIP,
			ClickCount: 0,
			IsDeleted:  false,
		}

		// Create the mapping
		err := storage.Create(ctx, mapping)
		require.NoError(t, err, "Create should succeed")

		// Retrieve and verify all required fields are present and non-empty
		retrieved, err := storage.Get(ctx, shortCode)
		require.NoError(t, err, "Get should succeed")
		require.NotNil(t, retrieved, "Retrieved mapping should not be nil")

		// Verify required fields are populated
		assert.NotEmpty(t, retrieved.ShortCode, "ShortCode must be populated")
		assert.NotEmpty(t, retrieved.LongURL, "LongURL must be populated")
		assert.False(t, retrieved.CreatedAt.IsZero(), "CreatedAt must be populated")
		assert.NotEmpty(t, retrieved.CreatorIP, "CreatorIP must be populated")

		// Verify the values match what was created
		assert.Equal(t, shortCode, retrieved.ShortCode)
		assert.Equal(t, longURL, retrieved.LongURL)
		assert.Equal(t, creatorIP, retrieved.CreatorIP)
		assert.WithinDuration(t, createdAt, retrieved.CreatedAt, time.Second)
	})
}

// TestProperty_CreateRetrieveWithExpiration tests create-retrieve consistency with expiration
// This extends Property 3 to include mappings with expiration times
// Requirements: 2.1, 5.1
func TestProperty_CreateRetrieveWithExpiration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		storage := NewInMemoryStorage()
		ctx := context.Background()

		// Generate random URL mapping with expiration
		shortCode := rapid.StringMatching(`^[0-9a-zA-Z]{7}$`).Draw(t, "shortCode")
		longURL := "https://" + rapid.StringN(5, 50, -1).Draw(t, "domain") + ".com"
		creatorIP := rapid.StringMatching(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(t, "ip")

		// Generate expiration time in the future (1 hour to 10 years)
		hoursInFuture := rapid.Int64Range(1, 87600).Draw(t, "hoursInFuture") // 1 hour to 10 years
		expiresAt := time.Now().Add(time.Duration(hoursInFuture) * time.Hour)

		mapping := &URLMapping{
			ShortCode:  shortCode,
			LongURL:    longURL,
			CreatedAt:  time.Now(),
			ExpiresAt:  &expiresAt,
			CreatorIP:  creatorIP,
			ClickCount: 0,
			IsDeleted:  false,
		}

		// Create the mapping
		err := storage.Create(ctx, mapping)
		require.NoError(t, err, "Create should succeed")

		// Immediately retrieve it
		retrieved, err := storage.Get(ctx, shortCode)
		require.NoError(t, err, "Get should succeed")
		require.NotNil(t, retrieved, "Retrieved mapping should not be nil")

		// Verify expiration is preserved
		require.NotNil(t, retrieved.ExpiresAt, "ExpiresAt should not be nil")
		assert.WithinDuration(t, *mapping.ExpiresAt, *retrieved.ExpiresAt, time.Second, "ExpiresAt should be preserved")
	})
}

// TestProperty_MultipleCreateRetrieve tests consistency across multiple mappings
// This verifies that creating multiple mappings doesn't interfere with each other
// Requirements: 2.1, 2.2
func TestProperty_MultipleCreateRetrieve(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		storage := NewInMemoryStorage()
		ctx := context.Background()

		// Generate multiple unique short codes
		numMappings := rapid.IntRange(2, 10).Draw(t, "numMappings")
		mappings := make([]*URLMapping, numMappings)
		usedCodes := make(map[string]bool)

		for i := 0; i < numMappings; i++ {
			// Generate unique short code
			var shortCode string
			for {
				shortCode = rapid.StringMatching(`^[0-9a-zA-Z]{7}$`).Draw(t, "shortCode")
				if !usedCodes[shortCode] {
					usedCodes[shortCode] = true
					break
				}
			}

			longURL := "https://example" + rapid.StringN(1, 10, -1).Draw(t, "suffix") + ".com"
			creatorIP := rapid.StringMatching(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(t, "ip")

			mapping := &URLMapping{
				ShortCode:  shortCode,
				LongURL:    longURL,
				CreatedAt:  time.Now(),
				ExpiresAt:  nil,
				CreatorIP:  creatorIP,
				ClickCount: 0,
				IsDeleted:  false,
			}

			mappings[i] = mapping

			// Create the mapping
			err := storage.Create(ctx, mapping)
			require.NoError(t, err, "Create should succeed for mapping %d", i)
		}

		// Retrieve all mappings and verify consistency
		for i, original := range mappings {
			retrieved, err := storage.Get(ctx, original.ShortCode)
			require.NoError(t, err, "Get should succeed for mapping %d", i)
			require.NotNil(t, retrieved, "Retrieved mapping %d should not be nil", i)

			assert.Equal(t, original.ShortCode, retrieved.ShortCode, "ShortCode should match for mapping %d", i)
			assert.Equal(t, original.LongURL, retrieved.LongURL, "LongURL should match for mapping %d", i)
			assert.Equal(t, original.CreatorIP, retrieved.CreatorIP, "CreatorIP should match for mapping %d", i)
		}
	})
}

// TestProperty_SoftDeletePreservesData tests that soft delete doesn't lose data
// Requirements: 4.6
func TestProperty_SoftDeletePreservesData(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		storage := NewInMemoryStorage()
		ctx := context.Background()

		// Generate random URL mapping
		shortCode := rapid.StringMatching(`^[0-9a-zA-Z]{7}$`).Draw(t, "shortCode")
		longURL := "https://" + rapid.StringN(5, 50, -1).Draw(t, "domain") + ".com"
		creatorIP := rapid.StringMatching(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`).Draw(t, "ip")

		mapping := &URLMapping{
			ShortCode:  shortCode,
			LongURL:    longURL,
			CreatedAt:  time.Now(),
			ExpiresAt:  nil,
			CreatorIP:  creatorIP,
			ClickCount: 0,
			IsDeleted:  false,
		}

		// Create the mapping
		err := storage.Create(ctx, mapping)
		require.NoError(t, err, "Create should succeed")

		// Soft delete it
		err = storage.Delete(ctx, shortCode)
		require.NoError(t, err, "Delete should succeed")

		// Verify it still exists in storage but is marked as deleted
		exists, err := storage.Exists(ctx, shortCode)
		require.NoError(t, err, "Exists check should succeed")
		assert.True(t, exists, "Mapping should still exist after soft delete")

		// Verify Get returns error (because it's deleted)
		retrieved, err := storage.Get(ctx, shortCode)
		assert.Error(t, err, "Get should fail for deleted mapping")
		assert.Nil(t, retrieved, "Retrieved mapping should be nil for deleted code")
	})
}
