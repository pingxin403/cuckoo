//go:build property
// +build property

package idgen

import (
	"context"
	"regexp"
	"sync"
	"testing"

	"pgregory.net/rapid"
)

// MockStorage is a simple in-memory storage for testing
type MockStorage struct {
	mu    sync.RWMutex
	codes map[string]bool
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		codes: make(map[string]bool),
	}
}

func (m *MockStorage) Exists(ctx context.Context, shortCode string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.codes[shortCode], nil
}

func (m *MockStorage) Add(shortCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[shortCode] = true
}

// TestProperty_ShortCodeFormatAndUniqueness tests Property 1:
// Short Code Format and Uniqueness
//
// Feature: url-shortener-service, Property 1: Short Code Format and Uniqueness
// Validates: Requirements 1.1, 1.2
//
// This property verifies that:
// 1. All generated short codes are exactly 7 characters long
// 2. All codes contain only Base62 characters (0-9, a-z, A-Z)
// 3. All generated codes are unique (no duplicates)
func TestProperty_ShortCodeFormatAndUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		storage := NewMockStorage()
		generator := NewRandomIDGenerator(storage)
		ctx := context.Background()

		// Generate multiple codes (1000+ as specified in requirements)
		numCodes := rapid.IntRange(1000, 1500).Draw(t, "numCodes")
		codes := make(map[string]bool)
		base62Pattern := regexp.MustCompile(`^[0-9a-zA-Z]{7}$`)

		// Act - Generate codes
		for i := 0; i < numCodes; i++ {
			code, err := generator.Generate(ctx)
			if err != nil {
				t.Fatalf("Generate failed on iteration %d: %v", i, err)
			}

			// Assert 1: Code is exactly 7 characters long
			if len(code) != 7 {
				t.Fatalf("Code length is %d, expected 7: %s", len(code), code)
			}

			// Assert 2: Code contains only Base62 characters
			if !base62Pattern.MatchString(code) {
				t.Fatalf("Code contains invalid characters: %s", code)
			}

			// Assert 3: Code is unique
			if codes[code] {
				t.Fatalf("Duplicate code generated: %s", code)
			}
			codes[code] = true

			// Add to storage to simulate persistence
			storage.Add(code)
		}

		// Final verification: All codes are unique
		if len(codes) != numCodes {
			t.Fatalf("Expected %d unique codes, got %d", numCodes, len(codes))
		}
	})
}

// TestProperty_ShortCodeUnpredictability tests that generated codes
// are not sequential or predictable
func TestProperty_ShortCodeUnpredictability(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		storage := NewMockStorage()
		generator := NewRandomIDGenerator(storage)
		ctx := context.Background()

		// Generate a small batch of codes
		numCodes := rapid.IntRange(10, 50).Draw(t, "numCodes")
		codes := make([]string, numCodes)

		// Act - Generate codes
		for i := 0; i < numCodes; i++ {
			code, err := generator.Generate(ctx)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}
			codes[i] = code
			storage.Add(code)
		}

		// Assert - Codes should not be sequential
		// Check that consecutive codes differ in multiple positions
		for i := 0; i < len(codes)-1; i++ {
			diffCount := 0
			for j := 0; j < 7; j++ {
				if codes[i][j] != codes[i+1][j] {
					diffCount++
				}
			}

			// At least 3 positions should differ (not sequential)
			if diffCount < 3 {
				t.Fatalf("Codes appear sequential: %s and %s (only %d positions differ)",
					codes[i], codes[i+1], diffCount)
			}
		}
	})
}

// TestProperty_CollisionRetry tests that the generator retries on collision
func TestProperty_CollisionRetry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		storage := NewMockStorage()
		generator := NewRandomIDGenerator(storage)
		ctx := context.Background()

		// Pre-populate storage with some codes
		numExisting := rapid.IntRange(5, 20).Draw(t, "numExisting")
		for i := 0; i < numExisting; i++ {
			code, err := generator.Generate(ctx)
			if err != nil {
				t.Fatalf("Failed to generate initial code: %v", err)
			}
			storage.Add(code)
		}

		// Act - Generate new codes (should avoid collisions)
		numNew := rapid.IntRange(10, 30).Draw(t, "numNew")
		newCodes := make(map[string]bool)

		for i := 0; i < numNew; i++ {
			code, err := generator.Generate(ctx)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Assert - New code should not exist in storage
			exists, _ := storage.Exists(ctx, code)
			if exists {
				t.Fatalf("Generated code already exists: %s", code)
			}

			newCodes[code] = true
			storage.Add(code)
		}

		// Verify all new codes are unique
		if len(newCodes) != numNew {
			t.Fatalf("Expected %d unique new codes, got %d", numNew, len(newCodes))
		}
	})
}
