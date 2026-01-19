package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestURLValidator_ValidHTTPURLs tests validation of valid HTTP URLs
// Requirements: 1.4, 14.1
func TestURLValidator_ValidHTTPURLs(t *testing.T) {
	validator := NewURLValidator()

	validURLs := []string{
		"http://example.com",
		"https://example.com",
		"http://example.com/path",
		"https://example.com/path/to/resource",
		"http://example.com:8080",
		"https://example.com:443/path?query=value",
		"http://subdomain.example.com",
		"https://example.com/path?key1=value1&key2=value2",
		"http://example.com/path#fragment",
		"https://example.com/path?query=value#fragment",
		"http://192.168.1.1",
		"https://192.168.1.1:8080/path",
		"http://example.com/path/with-hyphens",
		"https://example.com/path_with_underscores",
		"http://example.com/path%20with%20spaces",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.Validate(url)
			assert.NoError(t, err, "Expected valid URL: %s", url)
		})
	}
}

// TestURLValidator_InvalidProtocol tests rejection of non-HTTP/HTTPS protocols
// Requirements: 14.1
func TestURLValidator_InvalidProtocol(t *testing.T) {
	validator := NewURLValidator()

	invalidURLs := []struct {
		url      string
		protocol string
	}{
		{"ftp://example.com", "ftp"},
		{"file:///etc/passwd", "file"},
		{"javascript:alert('xss')", "javascript"},
		{"data:text/html,<script>alert('xss')</script>", "data"},
		{"vbscript:msgbox('xss')", "vbscript"},
		{"about:blank", "about"},
		{"mailto:user@example.com", "mailto"},
		{"tel:+1234567890", "tel"},
	}

	for _, tc := range invalidURLs {
		t.Run(tc.url, func(t *testing.T) {
			err := validator.Validate(tc.url)
			assert.Error(t, err, "Expected error for protocol: %s", tc.protocol)
			assert.True(t, errors.Is(err, ErrInvalidProtocol), "Expected ErrInvalidProtocol")
		})
	}
}

// TestURLValidator_URLLength tests URL length validation
// Requirements: 14.2
func TestURLValidator_URLLength(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "exactly 2048 characters - valid",
			url:         "https://example.com/" + strings.Repeat("a", 2048-20), // 2048 total
			expectError: false,
		},
		{
			name:        "2049 characters - too long",
			url:         "https://example.com/" + strings.Repeat("a", 2049-20), // 2049 total
			expectError: true,
		},
		{
			name:        "very long URL - 3000 characters",
			url:         "https://example.com/" + strings.Repeat("a", 3000),
			expectError: true,
		},
		{
			name:        "short URL",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "medium URL - 1000 characters",
			url:         "https://example.com/" + strings.Repeat("a", 1000),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.url)
			if tt.expectError {
				assert.Error(t, err, "Expected error for URL length")
				assert.True(t, errors.Is(err, ErrURLTooLong), "Expected ErrURLTooLong")
			} else {
				assert.NoError(t, err, "Expected no error for valid length")
			}
		})
	}
}

// TestURLValidator_MaliciousPatterns tests detection of malicious patterns
// Requirements: 14.3, 14.7
func TestURLValidator_MaliciousPatterns(t *testing.T) {
	validator := NewURLValidator()

	maliciousURLs := []struct {
		url     string
		pattern string
	}{
		{"http://example.com?redirect=javascript:alert('xss')", "javascript:"},
		{"http://example.com?data=data:text/html,<script>", "data:"},
		{"http://example.com/<script>alert('xss')</script>", "<script"},
		{"http://example.com/</script>", "</script>"},
		{"http://example.com?param=<img onerror=alert('xss')>", "onerror="},
		{"http://example.com?param=<body onload=alert('xss')>", "onload="},
		{"http://example.com?param=<div onclick=alert('xss')>", "onclick="},
		{"http://example.com/vbscript:msgbox('xss')", "vbscript:"},
		{"http://example.com/file:///etc/passwd", "file:"},
		{"http://example.com/about:blank", "about:"},
	}

	for _, tc := range maliciousURLs {
		t.Run(tc.url, func(t *testing.T) {
			err := validator.Validate(tc.url)
			assert.Error(t, err, "Expected error for malicious pattern: %s", tc.pattern)
			assert.True(t, errors.Is(err, ErrMaliciousPattern), "Expected ErrMaliciousPattern")
		})
	}
}

// TestURLValidator_CaseInsensitiveMaliciousPatterns tests case-insensitive detection
// Requirements: 14.3
func TestURLValidator_CaseInsensitiveMaliciousPatterns(t *testing.T) {
	validator := NewURLValidator()

	maliciousURLs := []string{
		"http://example.com?redirect=JAVASCRIPT:alert('xss')",
		"http://example.com?redirect=JavaScript:alert('xss')",
		"http://example.com?data=DATA:text/html",
		"http://example.com/<SCRIPT>alert('xss')</SCRIPT>",
		"http://example.com?param=<img ONERROR=alert('xss')>",
	}

	for _, url := range maliciousURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.Validate(url)
			assert.Error(t, err, "Expected error for case-insensitive malicious pattern")
			assert.True(t, errors.Is(err, ErrMaliciousPattern), "Expected ErrMaliciousPattern")
		})
	}
}

// TestURLValidator_EmptyURL tests validation of empty URLs
// Requirements: 1.4
func TestURLValidator_EmptyURL(t *testing.T) {
	validator := NewURLValidator()

	emptyURLs := []string{
		"",
		"   ",
		"\t",
		"\n",
		"  \t\n  ",
	}

	for _, url := range emptyURLs {
		t.Run("empty_url", func(t *testing.T) {
			err := validator.Validate(url)
			assert.Error(t, err, "Expected error for empty URL")
			assert.True(t, errors.Is(err, ErrInvalidURL), "Expected ErrInvalidURL")
		})
	}
}

// TestURLValidator_MissingHost tests validation of URLs without host
// Requirements: 1.4
func TestURLValidator_MissingHost(t *testing.T) {
	validator := NewURLValidator()

	invalidURLs := []string{
		"http://",
		"https://",
		"http:///path",
		"https:///path",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			err := validator.Validate(url)
			assert.Error(t, err, "Expected error for missing host")
			assert.True(t, errors.Is(err, ErrInvalidURL), "Expected ErrInvalidURL")
		})
	}
}

// TestURLValidator_Sanitize tests URL sanitization
// Requirements: 14.3
func TestURLValidator_Sanitize(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim leading whitespace",
			input:    "  https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "trim trailing whitespace",
			input:    "https://example.com  ",
			expected: "https://example.com",
		},
		{
			name:     "trim both sides",
			input:    "  https://example.com  ",
			expected: "https://example.com",
		},
		{
			name:     "no whitespace",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "tabs and newlines",
			input:    "\t\nhttps://example.com\n\t",
			expected: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Sanitize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestURLValidator_ValidateAndSanitize tests combined validation and sanitization
// Requirements: 1.4, 14.3
func TestURLValidator_ValidateAndSanitize(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "valid URL with whitespace",
			input:       "  https://example.com  ",
			expected:    "https://example.com",
			expectError: false,
		},
		{
			name:        "invalid protocol with whitespace",
			input:       "  ftp://example.com  ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "malicious URL with whitespace",
			input:       "  http://example.com?param=<script>  ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "valid URL no whitespace",
			input:       "https://example.com/path",
			expected:    "https://example.com/path",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateAndSanitize(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestURLValidator_EdgeCases tests various edge cases
// Requirements: 14.2, 14.3
func TestURLValidator_EdgeCases(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name        string
		url         string
		expectError bool
		errorType   error
	}{
		{
			name:        "URL with unicode characters",
			url:         "https://example.com/path/文件",
			expectError: false,
		},
		{
			name:        "URL with encoded spaces",
			url:         "https://example.com/path%20with%20spaces",
			expectError: false,
		},
		{
			name:        "URL with query parameters",
			url:         "https://example.com?key1=value1&key2=value2&key3=value3",
			expectError: false,
		},
		{
			name:        "URL with fragment",
			url:         "https://example.com/path#section",
			expectError: false,
		},
		{
			name:        "URL with port",
			url:         "https://example.com:8443/path",
			expectError: false,
		},
		{
			name:        "URL with authentication (valid but unusual)",
			url:         "https://user:pass@example.com/path",
			expectError: false,
		},
		{
			name:        "exactly at 2048 character boundary",
			url:         "https://example.com/" + strings.Repeat("a", 2028), // Total 2048
			expectError: false,
		},
		{
			name:        "one character over 2048 limit",
			url:         "https://example.com/" + strings.Repeat("a", 2029), // Total 2049
			expectError: true,
			errorType:   ErrURLTooLong,
		},
		{
			name:        "malicious pattern in query parameter",
			url:         "https://example.com?redirect=javascript:void(0)",
			expectError: true,
			errorType:   ErrMaliciousPattern,
		},
		{
			name:        "malicious pattern in fragment",
			url:         "https://example.com#<script>alert(1)</script>",
			expectError: true,
			errorType:   ErrMaliciousPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.url)
			if tt.expectError {
				require.Error(t, err, "Expected error for: %s", tt.name)
				if tt.errorType != nil {
					assert.True(t, errors.Is(err, tt.errorType), "Expected error type %v, got %v", tt.errorType, err)
				}
			} else {
				assert.NoError(t, err, "Expected no error for: %s", tt.name)
			}
		})
	}
}

// TestURLValidator_BoundaryConditions tests boundary conditions
// Requirements: 14.2
func TestURLValidator_BoundaryConditions(t *testing.T) {
	validator := NewURLValidator()

	// Test exactly at the 2048 boundary
	baseURL := "https://example.com/"
	baseLen := len(baseURL)

	// Create URL with exactly 2048 characters
	padding := strings.Repeat("a", 2048-baseLen)
	exactURL := baseURL + padding
	assert.Equal(t, 2048, len(exactURL), "URL should be exactly 2048 characters")

	err := validator.Validate(exactURL)
	assert.NoError(t, err, "URL with exactly 2048 characters should be valid")

	// Create URL with 2049 characters (one over limit)
	overURL := exactURL + "a"
	assert.Equal(t, 2049, len(overURL), "URL should be exactly 2049 characters")

	err = validator.Validate(overURL)
	assert.Error(t, err, "URL with 2049 characters should be invalid")
	assert.True(t, errors.Is(err, ErrURLTooLong), "Expected ErrURLTooLong")
}
