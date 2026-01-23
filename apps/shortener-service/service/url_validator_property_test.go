//go:build property
// +build property

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// TestProperty_URLValidationRejectsInvalidInputs tests Property 2: URL Validation Rejects Invalid Inputs
// Feature: url-shortener-service, Property 2: URL Validation Rejects Invalid Inputs
// For any URL that does not use HTTP or HTTPS protocol, or exceeds 2048 characters,
// or contains malicious patterns, the service SHALL reject it with an appropriate error
// Requirements: 1.4, 14.1, 14.2, 14.3, 14.7
func TestProperty_URLValidationRejectsInvalidInputs(t *testing.T) {
	validator := NewURLValidator()

	t.Run("rejects invalid protocols", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate invalid protocols
			invalidProtocols := []string{"ftp", "file", "javascript", "data", "vbscript", "about", "mailto", "tel"}
			protocol := rapid.SampledFrom(invalidProtocols).Draw(t, "protocol")

			// Generate a domain
			domain := rapid.StringMatching(`^[a-z0-9-]+\.[a-z]{2,}$`).Draw(t, "domain")

			// Construct URL with invalid protocol
			url := protocol + "://" + domain

			// Validate
			err := validator.Validate(url)
			assert.Error(t, err, "Should reject invalid protocol: %s", protocol)
		})
	})

	t.Run("rejects URLs exceeding 2048 characters", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate URL length > 2048
			baseURL := "https://example.com/"
			baseLen := len(baseURL)

			// Generate padding that makes total length > 2048
			excessLength := rapid.IntRange(1, 1000).Draw(t, "excessLength")
			totalLength := 2048 + excessLength
			paddingLength := totalLength - baseLen

			padding := strings.Repeat("a", paddingLength)
			longURL := baseURL + padding

			// Verify length is over limit
			assert.Greater(t, len(longURL), 2048, "URL should exceed 2048 characters")

			// Validate
			err := validator.Validate(longURL)
			assert.Error(t, err, "Should reject URL exceeding 2048 characters")
		})
	})

	t.Run("rejects URLs with malicious patterns", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate malicious patterns
			maliciousPatterns := []string{
				"javascript:",
				"data:",
				"<script",
				"</script>",
				"onerror=",
				"onload=",
				"onclick=",
			}
			pattern := rapid.SampledFrom(maliciousPatterns).Draw(t, "pattern")

			// Generate base URL
			domain := rapid.StringMatching(`^[a-z0-9-]+\.[a-z]{2,}$`).Draw(t, "domain")
			baseURL := "http://" + domain

			// Inject malicious pattern in different locations
			locations := []string{
				baseURL + "?param=" + pattern,
				baseURL + "/" + pattern,
				baseURL + "#" + pattern,
			}
			url := rapid.SampledFrom(locations).Draw(t, "location")

			// Validate
			err := validator.Validate(url)
			assert.Error(t, err, "Should reject URL with malicious pattern: %s", pattern)
		})
	})

	t.Run("accepts valid HTTP/HTTPS URLs within length limit", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate valid protocol
			protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "protocol")

			// Generate valid domain
			domain := rapid.StringMatching(`^[a-z0-9-]+\.[a-z]{2,}$`).Draw(t, "domain")

			// Generate path (keep total length under 2048)
			// Use URL-safe characters only: alphanumeric, hyphen, underscore, forward slash
			pathLength := rapid.IntRange(0, 100).Draw(t, "pathLength")
			path := ""
			if pathLength > 0 {
				pathStr := rapid.StringMatching(`^[a-zA-Z0-9/_-]*$`).Filter(func(s string) bool {
					// Filter out malicious patterns and ensure reasonable length
					if len(s) == 0 || len(s) > pathLength {
						return false
					}
					lower := strings.ToLower(s)
					for _, pattern := range maliciousPatterns {
						if strings.Contains(lower, strings.ToLower(pattern)) {
							return false
						}
					}
					return true
				}).Draw(t, "pathStr")
				path = "/" + pathStr
			}

			url := protocol + "://" + domain + path

			// Ensure total length is within limit
			if len(url) <= 2048 {
				// Validate
				err := validator.Validate(url)
				assert.NoError(t, err, "Should accept valid URL: %s", url)
			}
		})
	})
}

// TestProperty_ValidURLsPassValidation tests that properly formatted URLs pass validation
// Requirements: 1.4, 14.1
func TestProperty_ValidURLsPassValidation(t *testing.T) {
	validator := NewURLValidator()

	rapid.Check(t, func(t *rapid.T) {
		// Generate valid protocol
		protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "protocol")

		// Generate valid domain components
		subdomain := rapid.StringMatching(`^[a-z0-9-]{1,20}$`).Draw(t, "subdomain")
		domain := rapid.StringMatching(`^[a-z0-9-]{1,20}$`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net", "io", "dev"}).Draw(t, "tld")

		// Construct URL
		url := protocol + "://" + subdomain + "." + domain + "." + tld

		// Ensure length is within limit
		if len(url) <= 2048 {
			err := validator.Validate(url)
			assert.NoError(t, err, "Valid URL should pass validation: %s", url)
		}
	})
}

// TestProperty_SanitizationPreservesValidURLs tests that sanitization doesn't break valid URLs
// Requirements: 14.3
func TestProperty_SanitizationPreservesValidURLs(t *testing.T) {
	validator := NewURLValidator()

	rapid.Check(t, func(t *rapid.T) {
		// Generate valid URL
		protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "protocol")
		domain := rapid.StringMatching(`^[a-z0-9-]+\.[a-z]{2,}$`).Draw(t, "domain")
		url := protocol + "://" + domain

		// Add random whitespace
		whitespacePrefix := rapid.StringMatching(`^[ \t\n]*$`).Draw(t, "prefix")
		whitespaceSuffix := rapid.StringMatching(`^[ \t\n]*$`).Draw(t, "suffix")
		urlWithWhitespace := whitespacePrefix + url + whitespaceSuffix

		// Sanitize
		sanitized := validator.Sanitize(urlWithWhitespace)

		// Verify sanitized URL is valid
		err := validator.Validate(sanitized)
		assert.NoError(t, err, "Sanitized URL should be valid")

		// Verify whitespace is removed
		assert.Equal(t, strings.TrimSpace(urlWithWhitespace), sanitized)
	})
}

// TestProperty_MaliciousPatternsCaseInsensitive tests case-insensitive malicious pattern detection
// Requirements: 14.3
func TestProperty_MaliciousPatternsCaseInsensitive(t *testing.T) {
	validator := NewURLValidator()

	rapid.Check(t, func(t *rapid.T) {
		// Generate malicious pattern
		patterns := []string{"javascript:", "data:", "<script", "onerror="}
		pattern := rapid.SampledFrom(patterns).Draw(t, "pattern")

		// Generate random case variation
		var caseVariation strings.Builder
		for _, char := range pattern {
			if rapid.Bool().Draw(t, "uppercase") {
				caseVariation.WriteString(strings.ToUpper(string(char)))
			} else {
				caseVariation.WriteRune(char)
			}
		}
		caseVariedPattern := caseVariation.String()

		// Create URL with case-varied malicious pattern
		domain := rapid.StringMatching(`^[a-z0-9-]+\.[a-z]{2,}$`).Draw(t, "domain")
		url := "http://" + domain + "?param=" + caseVariedPattern

		// Validate
		err := validator.Validate(url)
		assert.Error(t, err, "Should reject malicious pattern regardless of case: %s", caseVariedPattern)
	})
}

// TestProperty_LengthBoundary tests URLs at the 2048 character boundary
// Requirements: 14.2
func TestProperty_LengthBoundary(t *testing.T) {
	validator := NewURLValidator()

	rapid.Check(t, func(t *rapid.T) {
		baseURL := "https://example.com/"
		baseLen := len(baseURL)

		// Generate padding to reach exactly 2048 or slightly over
		targetLength := rapid.IntRange(2047, 2050).Draw(t, "targetLength")
		paddingLength := targetLength - baseLen

		if paddingLength > 0 {
			padding := strings.Repeat("a", paddingLength)
			url := baseURL + padding

			err := validator.Validate(url)

			if len(url) <= 2048 {
				assert.NoError(t, err, "URL with %d characters should be valid", len(url))
			} else {
				assert.Error(t, err, "URL with %d characters should be invalid", len(url))
			}
		}
	})
}

// TestProperty_EmptyAndWhitespaceURLsRejected tests rejection of empty/whitespace URLs
// Requirements: 1.4
func TestProperty_EmptyAndWhitespaceURLsRejected(t *testing.T) {
	validator := NewURLValidator()

	rapid.Check(t, func(t *rapid.T) {
		// Generate whitespace-only string
		whitespace := rapid.StringMatching(`^[ \t\n]+$`).Draw(t, "whitespace")

		err := validator.Validate(whitespace)
		assert.Error(t, err, "Should reject whitespace-only URL")
	})
}
