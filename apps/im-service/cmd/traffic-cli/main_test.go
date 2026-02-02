package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRegionWeights(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expected    map[string]int
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid 90:10 split",
			args: []string{"region-a:90", "region-b:10"},
			expected: map[string]int{
				"region-a": 90,
				"region-b": 10,
			},
			expectError: false,
		},
		{
			name: "valid 50:50 split",
			args: []string{"region-a:50", "region-b:50"},
			expected: map[string]int{
				"region-a": 50,
				"region-b": 50,
			},
			expectError: false,
		},
		{
			name: "valid 100:0 split",
			args: []string{"region-a:100", "region-b:0"},
			expected: map[string]int{
				"region-a": 100,
				"region-b": 0,
			},
			expectError: false,
		},
		{
			name:        "invalid format - missing colon",
			args:        []string{"region-a90", "region-b:10"},
			expectError: true,
			errorMsg:    "invalid format",
		},
		{
			name:        "invalid format - multiple colons",
			args:        []string{"region-a:90:extra", "region-b:10"},
			expectError: true,
			errorMsg:    "invalid format",
		},
		{
			name:        "invalid weight - not a number",
			args:        []string{"region-a:abc", "region-b:10"},
			expectError: true,
			errorMsg:    "invalid weight",
		},
		{
			name:        "invalid weight - negative",
			args:        []string{"region-a:-10", "region-b:110"},
			expectError: true,
			errorMsg:    "must be between 0 and 100",
		},
		{
			name:        "invalid weight - over 100",
			args:        []string{"region-a:150", "region-b:10"},
			expectError: true,
			errorMsg:    "must be between 0 and 100",
		},
		{
			name:        "invalid total - doesn't sum to 100",
			args:        []string{"region-a:60", "region-b:30"},
			expectError: true,
			errorMsg:    "total weight must equal 100",
		},
		{
			name:        "invalid total - exceeds 100",
			args:        []string{"region-a:60", "region-b:50"},
			expectError: true,
			errorMsg:    "total weight must equal 100",
		},
		{
			name: "valid with whitespace",
			args: []string{" region-a : 70 ", " region-b : 30 "},
			expected: map[string]int{
				"region-a": 70,
				"region-b": 30,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRegionWeights(tt.args)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseRegionWeights_EdgeCases(t *testing.T) {
	t.Run("empty args", func(t *testing.T) {
		result, err := parseRegionWeights([]string{})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "total weight must equal 100")
	})

	t.Run("single region with 100%", func(t *testing.T) {
		// This is actually valid - a single region can have 100%
		result, err := parseRegionWeights([]string{"region-a:100"})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{
			"region-a": 100,
		}, result)
	})

	t.Run("three regions", func(t *testing.T) {
		result, err := parseRegionWeights([]string{
			"region-a:50",
			"region-b:30",
			"region-c:20",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]int{
			"region-a": 50,
			"region-b": 30,
			"region-c": 20,
		}, result)
	})
}

func TestGetDefaultOperator(t *testing.T) {
	// Save original env vars
	originalUser := os.Getenv("USER")
	originalUsername := os.Getenv("USERNAME")
	defer func() {
		_ = os.Setenv("USER", originalUser)
		_ = os.Setenv("USERNAME", originalUsername)
	}()

	t.Run("USER env var set", func(t *testing.T) {
		_ = os.Setenv("USER", "testuser")
		_ = os.Unsetenv("USERNAME")

		operator := getDefaultOperator()
		assert.Equal(t, "testuser", operator)
	})

	t.Run("USERNAME env var set", func(t *testing.T) {
		_ = os.Unsetenv("USER")
		_ = os.Setenv("USERNAME", "testuser2")

		operator := getDefaultOperator()
		assert.Equal(t, "testuser2", operator)
	})

	t.Run("no env vars set", func(t *testing.T) {
		_ = os.Unsetenv("USER")
		_ = os.Unsetenv("USERNAME")

		operator := getDefaultOperator()
		assert.Equal(t, "unknown", operator)
	})

	t.Run("USER takes precedence over USERNAME", func(t *testing.T) {
		_ = os.Setenv("USER", "user1")
		_ = os.Setenv("USERNAME", "user2")

		operator := getDefaultOperator()
		assert.Equal(t, "user1", operator)
	})
}

// Benchmark tests
func BenchmarkParseRegionWeights(b *testing.B) {
	args := []string{"region-a:90", "region-b:10"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRegionWeights(args)
	}
}

func BenchmarkParseRegionWeights_Complex(b *testing.B) {
	args := []string{
		"region-a:40",
		"region-b:30",
		"region-c:20",
		"region-d:10",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseRegionWeights(args)
	}
}
