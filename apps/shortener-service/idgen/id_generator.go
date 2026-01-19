package idgen

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrMaxRetriesExceeded is returned when collision detection exceeds max retries
	ErrMaxRetriesExceeded = errors.New("max retries exceeded for generating unique code")

	// ErrInvalidCustomCode is returned when a custom code doesn't meet requirements
	ErrInvalidCustomCode = errors.New("custom code is invalid")

	// ErrCustomCodeUnavailable is returned when a custom code is already taken
	ErrCustomCodeUnavailable = errors.New("custom code is unavailable")
)

// Storage defines the interface for checking code existence
type Storage interface {
	// Exists checks if a short code already exists
	Exists(ctx context.Context, shortCode string) (bool, error)
}

// IDGenerator defines the interface for generating short codes
type IDGenerator interface {
	// Generate creates a new unique short code
	// Returns error if max retries exceeded
	Generate(ctx context.Context) (string, error)

	// ValidateCustomCode checks if a custom code is available and valid
	ValidateCustomCode(ctx context.Context, code string) error
}

// RandomIDGenerator implements IDGenerator using cryptographic random generation
type RandomIDGenerator struct {
	length          int
	maxRetries      int
	charset         string
	storage         Storage
	reservedWords   map[string]bool
	customCodeRegex *regexp.Regexp
}

// Base62 charset: 0-9, a-z, A-Z (62 characters)
const base62Charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Reserved keywords that cannot be used as custom codes
var defaultReservedWords = []string{
	"admin", "api", "health", "metrics", "ready", "status",
	"login", "logout", "register", "signup", "signin",
	"dashboard", "settings", "profile", "account",
}

// NewRandomIDGenerator creates a new ID generator with default settings
func NewRandomIDGenerator(storage Storage) *RandomIDGenerator {
	reservedMap := make(map[string]bool)
	for _, word := range defaultReservedWords {
		reservedMap[strings.ToLower(word)] = true
	}

	return &RandomIDGenerator{
		length:          7,
		maxRetries:      3,
		charset:         base62Charset,
		storage:         storage,
		reservedWords:   reservedMap,
		customCodeRegex: regexp.MustCompile(`^[a-zA-Z0-9-]{4,20}$`),
	}
}

// Generate creates a new unique short code using cryptographic random generation
func (g *RandomIDGenerator) Generate(ctx context.Context) (string, error) {
	for i := 0; i < g.maxRetries; i++ {
		// Generate cryptographically secure random bytes
		bytes := make([]byte, g.length)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		// Convert to Base62
		code := g.toBase62(bytes)

		// Check for collision
		exists, err := g.storage.Exists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}

		if !exists {
			return code, nil
		}
	}

	return "", ErrMaxRetriesExceeded
}

// toBase62 converts random bytes to a Base62 encoded string
func (g *RandomIDGenerator) toBase62(bytes []byte) string {
	result := make([]byte, g.length)
	charsetLen := len(g.charset)

	for i := 0; i < g.length; i++ {
		// Use modulo to map byte value to charset index
		result[i] = g.charset[int(bytes[i])%charsetLen]
	}

	return string(result)
}

// ValidateCustomCode checks if a custom code is available and meets requirements
func (g *RandomIDGenerator) ValidateCustomCode(ctx context.Context, code string) error {
	// Check length (4-20 characters)
	if len(code) < 4 || len(code) > 20 {
		return fmt.Errorf("%w: length must be between 4 and 20 characters", ErrInvalidCustomCode)
	}

	// Check character set (alphanumeric + hyphen)
	if !g.customCodeRegex.MatchString(code) {
		return fmt.Errorf("%w: only alphanumeric characters and hyphens allowed", ErrInvalidCustomCode)
	}

	// Check reserved keywords
	if g.reservedWords[strings.ToLower(code)] {
		return fmt.Errorf("%w: code is a reserved keyword", ErrInvalidCustomCode)
	}

	// Check availability
	exists, err := g.storage.Exists(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to check code availability: %w", err)
	}

	if exists {
		return ErrCustomCodeUnavailable
	}

	return nil
}
