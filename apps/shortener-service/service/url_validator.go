package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	// ErrInvalidURL is returned when a URL fails validation
	ErrInvalidURL = errors.New("invalid URL")

	// ErrURLTooLong is returned when a URL exceeds the maximum length
	ErrURLTooLong = errors.New("URL exceeds maximum length of 2048 characters")

	// ErrInvalidProtocol is returned when a URL uses a non-HTTP/HTTPS protocol
	ErrInvalidProtocol = errors.New("URL must use HTTP or HTTPS protocol")

	// ErrMaliciousPattern is returned when a URL contains malicious patterns
	ErrMaliciousPattern = errors.New("URL contains malicious patterns")
)

// Malicious patterns to check for
var maliciousPatterns = []string{
	"javascript:",
	"data:",
	"vbscript:",
	"file:",
	"about:",
	"<script",
	"</script>",
	"onerror=",
	"onload=",
	"onclick=",
}

// URLValidator validates URLs for the shortener service
type URLValidator struct {
	maxLength         int
	allowedProtocols  map[string]bool
	maliciousPatterns []string
}

// NewURLValidator creates a new URL validator with default settings
func NewURLValidator() *URLValidator {
	return &URLValidator{
		maxLength: 2048,
		allowedProtocols: map[string]bool{
			"http":  true,
			"https": true,
		},
		maliciousPatterns: maliciousPatterns,
	}
}

// Validate checks if a URL is valid and safe
// Requirements: 1.4, 14.1, 14.2, 14.3
func (v *URLValidator) Validate(rawURL string) error {
	// Check length first (before parsing)
	if len(rawURL) > v.maxLength {
		return fmt.Errorf("%w: got %d characters", ErrURLTooLong, len(rawURL))
	}

	// Check for empty URL
	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("%w: URL cannot be empty", ErrInvalidURL)
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// Validate protocol
	if !v.allowedProtocols[strings.ToLower(parsedURL.Scheme)] {
		return fmt.Errorf("%w: got %s", ErrInvalidProtocol, parsedURL.Scheme)
	}

	// Check for malicious patterns (case-insensitive)
	lowerURL := strings.ToLower(rawURL)
	for _, pattern := range v.maliciousPatterns {
		if strings.Contains(lowerURL, strings.ToLower(pattern)) {
			return fmt.Errorf("%w: contains %s", ErrMaliciousPattern, pattern)
		}
	}

	// Validate host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	return nil
}

// Sanitize performs basic sanitization on a URL
// This trims whitespace and normalizes the URL
func (v *URLValidator) Sanitize(rawURL string) string {
	// Trim whitespace
	sanitized := strings.TrimSpace(rawURL)

	// Parse and reconstruct to normalize
	if parsedURL, err := url.Parse(sanitized); err == nil {
		sanitized = parsedURL.String()
	}

	return sanitized
}

// ValidateAndSanitize validates and sanitizes a URL in one step
func (v *URLValidator) ValidateAndSanitize(rawURL string) (string, error) {
	sanitized := v.Sanitize(rawURL)
	if err := v.Validate(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}
