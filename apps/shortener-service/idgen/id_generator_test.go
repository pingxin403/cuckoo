package idgen

import (
	"context"
	"errors"
	"testing"
)

// MockStorageWithCollisions simulates storage that returns collisions
type MockStorageWithCollisions struct {
	collisionCount int
	currentAttempt int
}

func NewMockStorageWithCollisions(collisionCount int) *MockStorageWithCollisions {
	return &MockStorageWithCollisions{
		collisionCount: collisionCount,
		currentAttempt: 0,
	}
}

func (m *MockStorageWithCollisions) Exists(ctx context.Context, shortCode string) (bool, error) {
	m.currentAttempt++
	// Return true (collision) for the first N attempts, then false
	return m.currentAttempt <= m.collisionCount, nil
}

func (m *MockStorageWithCollisions) Reset() {
	m.currentAttempt = 0
}

// TestGenerate_RetryOnCollision tests that the generator retries when collisions occur
// Requirements: 1.3
func TestGenerate_RetryOnCollision(t *testing.T) {
	tests := []struct {
		name           string
		collisionCount int
		maxRetries     int
		expectError    bool
	}{
		{
			name:           "no collision - succeeds on first attempt",
			collisionCount: 0,
			maxRetries:     3,
			expectError:    false,
		},
		{
			name:           "one collision - succeeds on second attempt",
			collisionCount: 1,
			maxRetries:     3,
			expectError:    false,
		},
		{
			name:           "two collisions - succeeds on third attempt",
			collisionCount: 2,
			maxRetries:     3,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := NewMockStorageWithCollisions(tt.collisionCount)
			generator := &RandomIDGenerator{
				length:     7,
				maxRetries: tt.maxRetries,
				charset:    base62Charset,
				storage:    storage,
			}
			ctx := context.Background()

			// Act
			code, err := generator.Generate(ctx)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(code) != 7 {
					t.Errorf("Expected code length 7, got %d", len(code))
				}
			}

			// Verify the number of attempts made
			expectedAttempts := tt.collisionCount + 1
			if storage.currentAttempt != expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", expectedAttempts, storage.currentAttempt)
			}
		})
	}
}

// TestGenerate_MaxRetriesExceeded tests that the generator returns error after max retries
// Requirements: 1.3
func TestGenerate_MaxRetriesExceeded(t *testing.T) {
	tests := []struct {
		name           string
		collisionCount int
		maxRetries     int
	}{
		{
			name:           "exceeds max retries with 3 collisions and 3 max retries",
			collisionCount: 3,
			maxRetries:     3,
		},
		{
			name:           "exceeds max retries with 5 collisions and 3 max retries",
			collisionCount: 5,
			maxRetries:     3,
		},
		{
			name:           "exceeds max retries with 10 collisions and 5 max retries",
			collisionCount: 10,
			maxRetries:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := NewMockStorageWithCollisions(tt.collisionCount)
			generator := &RandomIDGenerator{
				length:     7,
				maxRetries: tt.maxRetries,
				charset:    base62Charset,
				storage:    storage,
			}
			ctx := context.Background()

			// Act
			code, err := generator.Generate(ctx)

			// Assert
			if err == nil {
				t.Errorf("Expected error but got none")
			}
			if !errors.Is(err, ErrMaxRetriesExceeded) {
				t.Errorf("Expected ErrMaxRetriesExceeded, got: %v", err)
			}
			if code != "" {
				t.Errorf("Expected empty code on error, got: %s", code)
			}

			// Verify exactly maxRetries attempts were made
			if storage.currentAttempt != tt.maxRetries {
				t.Errorf("Expected %d attempts (max retries), got %d", tt.maxRetries, storage.currentAttempt)
			}
		})
	}
}

// TestGenerate_StorageError tests that storage errors are propagated
// Requirements: 1.3
func TestGenerate_StorageError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("storage connection failed")
	storage := &MockStorageWithError{err: expectedErr}
	generator := &RandomIDGenerator{
		length:     7,
		maxRetries: 3,
		charset:    base62Charset,
		storage:    storage,
	}
	ctx := context.Background()

	// Act
	code, err := generator.Generate(ctx)

	// Assert
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	if code != "" {
		t.Errorf("Expected empty code on error, got: %s", code)
	}
	// Verify the error is wrapped and contains the original error
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to wrap storage error, got: %v", err)
	}
}

// MockStorageWithError simulates storage that always returns an error
type MockStorageWithError struct {
	err error
}

func (m *MockStorageWithError) Exists(ctx context.Context, shortCode string) (bool, error) {
	return false, m.err
}

// MockStorageEmpty simulates empty storage (no codes exist)
type MockStorageEmpty struct{}

func (m *MockStorageEmpty) Exists(ctx context.Context, shortCode string) (bool, error) {
	return false, nil
}

// MockStorageWithCode simulates storage that contains specific codes
type MockStorageWithCode struct {
	existingCodes map[string]bool
}

func NewMockStorageWithCode(codes ...string) *MockStorageWithCode {
	m := &MockStorageWithCode{
		existingCodes: make(map[string]bool),
	}
	for _, code := range codes {
		m.existingCodes[code] = true
	}
	return m
}

func (m *MockStorageWithCode) Exists(ctx context.Context, shortCode string) (bool, error) {
	return m.existingCodes[shortCode], nil
}

// TestValidateCustomCode_LengthLimits tests custom code length validation
// Requirements: 8.1, 8.3
func TestValidateCustomCode_LengthLimits(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "too short - 3 characters",
			code:        "abc",
			expectError: true,
			errorMsg:    "length must be between 4 and 20 characters",
		},
		{
			name:        "minimum valid length - 4 characters",
			code:        "abcd",
			expectError: false,
		},
		{
			name:        "medium length - 10 characters",
			code:        "abcdefghij",
			expectError: false,
		},
		{
			name:        "maximum valid length - 20 characters",
			code:        "abcdefghij1234567890",
			expectError: false,
		},
		{
			name:        "too long - 21 characters",
			code:        "abcdefghij12345678901",
			expectError: true,
			errorMsg:    "length must be between 4 and 20 characters",
		},
		{
			name:        "empty string",
			code:        "",
			expectError: true,
			errorMsg:    "length must be between 4 and 20 characters",
		},
		{
			name:        "single character",
			code:        "a",
			expectError: true,
			errorMsg:    "length must be between 4 and 20 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := &MockStorageEmpty{}
			generator := NewRandomIDGenerator(storage)
			ctx := context.Background()

			// Act
			err := generator.ValidateCustomCode(ctx, tt.code)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !errors.Is(err, ErrInvalidCustomCode) {
					t.Errorf("Expected ErrInvalidCustomCode, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestValidateCustomCode_CharacterSet tests custom code character validation
// Requirements: 8.2, 8.5
func TestValidateCustomCode_CharacterSet(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid - lowercase letters only",
			code:        "abcdef",
			expectError: false,
		},
		{
			name:        "valid - uppercase letters only",
			code:        "ABCDEF",
			expectError: false,
		},
		{
			name:        "valid - mixed case letters",
			code:        "AbCdEf",
			expectError: false,
		},
		{
			name:        "valid - numbers only",
			code:        "123456",
			expectError: false,
		},
		{
			name:        "valid - alphanumeric mix",
			code:        "abc123XYZ",
			expectError: false,
		},
		{
			name:        "valid - with hyphens",
			code:        "my-custom-code",
			expectError: false,
		},
		{
			name:        "valid - hyphen at start",
			code:        "-mycode",
			expectError: false,
		},
		{
			name:        "valid - hyphen at end",
			code:        "mycode-",
			expectError: false,
		},
		{
			name:        "valid - multiple hyphens",
			code:        "my-custom-code-123",
			expectError: false,
		},
		{
			name:        "invalid - contains underscore",
			code:        "my_code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains space",
			code:        "my code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains special character @",
			code:        "my@code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains special character #",
			code:        "code#123",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains dot",
			code:        "my.code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains slash",
			code:        "my/code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
		{
			name:        "invalid - contains plus",
			code:        "my+code",
			expectError: true,
			errorMsg:    "only alphanumeric characters and hyphens allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := &MockStorageEmpty{}
			generator := NewRandomIDGenerator(storage)
			ctx := context.Background()

			// Act
			err := generator.ValidateCustomCode(ctx, tt.code)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !errors.Is(err, ErrInvalidCustomCode) {
					t.Errorf("Expected ErrInvalidCustomCode, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestValidateCustomCode_ReservedKeywords tests reserved keyword rejection
// Requirements: 8.2, 8.3
func TestValidateCustomCode_ReservedKeywords(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
	}{
		{
			name:        "reserved - admin",
			code:        "admin",
			expectError: true,
		},
		{
			name:        "reserved - api",
			code:        "api",
			expectError: true,
		},
		{
			name:        "reserved - health",
			code:        "health",
			expectError: true,
		},
		{
			name:        "reserved - metrics",
			code:        "metrics",
			expectError: true,
		},
		{
			name:        "reserved - ready",
			code:        "ready",
			expectError: true,
		},
		{
			name:        "reserved - status",
			code:        "status",
			expectError: true,
		},
		{
			name:        "reserved - login",
			code:        "login",
			expectError: true,
		},
		{
			name:        "reserved - logout",
			code:        "logout",
			expectError: true,
		},
		{
			name:        "reserved - register",
			code:        "register",
			expectError: true,
		},
		{
			name:        "reserved - signup",
			code:        "signup",
			expectError: true,
		},
		{
			name:        "reserved - signin",
			code:        "signin",
			expectError: true,
		},
		{
			name:        "reserved - dashboard",
			code:        "dashboard",
			expectError: true,
		},
		{
			name:        "reserved - settings",
			code:        "settings",
			expectError: true,
		},
		{
			name:        "reserved - profile",
			code:        "profile",
			expectError: true,
		},
		{
			name:        "reserved - account",
			code:        "account",
			expectError: true,
		},
		{
			name:        "reserved - case insensitive ADMIN",
			code:        "ADMIN",
			expectError: true,
		},
		{
			name:        "reserved - case insensitive Admin",
			code:        "Admin",
			expectError: true,
		},
		{
			name:        "reserved - case insensitive API",
			code:        "API",
			expectError: true,
		},
		{
			name:        "not reserved - similar to admin",
			code:        "administrator",
			expectError: false,
		},
		{
			name:        "not reserved - contains admin",
			code:        "myadmin",
			expectError: false,
		},
		{
			name:        "not reserved - custom code",
			code:        "mycode",
			expectError: false,
		},
		{
			name:        "not reserved - brand name",
			code:        "iphone",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := &MockStorageEmpty{}
			generator := NewRandomIDGenerator(storage)
			ctx := context.Background()

			// Act
			err := generator.ValidateCustomCode(ctx, tt.code)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for reserved keyword '%s' but got none", tt.code)
				} else if !errors.Is(err, ErrInvalidCustomCode) {
					t.Errorf("Expected ErrInvalidCustomCode, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for non-reserved code '%s' but got: %v", tt.code, err)
				}
			}
		})
	}
}

// TestValidateCustomCode_Availability tests code availability checking
// Requirements: 8.4, 8.5
func TestValidateCustomCode_Availability(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		existingCodes []string
		expectError   bool
		expectedErr   error
	}{
		{
			name:          "available - no existing codes",
			code:          "mycode",
			existingCodes: []string{},
			expectError:   false,
		},
		{
			name:          "available - different existing codes",
			code:          "mycode",
			existingCodes: []string{"other1", "other2", "other3"},
			expectError:   false,
		},
		{
			name:          "unavailable - exact match",
			code:          "mycode",
			existingCodes: []string{"mycode"},
			expectError:   true,
			expectedErr:   ErrCustomCodeUnavailable,
		},
		{
			name:          "unavailable - among multiple codes",
			code:          "mycode",
			existingCodes: []string{"other1", "mycode", "other2"},
			expectError:   true,
			expectedErr:   ErrCustomCodeUnavailable,
		},
		{
			name:          "available - case sensitive (different case)",
			code:          "MyCode",
			existingCodes: []string{"mycode"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := NewMockStorageWithCode(tt.existingCodes...)
			generator := NewRandomIDGenerator(storage)
			ctx := context.Background()

			// Act
			err := generator.ValidateCustomCode(ctx, tt.code)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got: %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestValidateCustomCode_StorageError tests storage error handling
// Requirements: 8.5
func TestValidateCustomCode_StorageError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("database connection failed")
	storage := &MockStorageWithError{err: expectedErr}
	generator := NewRandomIDGenerator(storage)
	ctx := context.Background()

	// Act
	err := generator.ValidateCustomCode(ctx, "validcode")

	// Assert
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	// Verify the error is wrapped and contains the original error
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to wrap storage error, got: %v", err)
	}
}

// TestValidateCustomCode_CombinedValidation tests multiple validation rules together
// Requirements: 8.1, 8.2, 8.3, 8.5
func TestValidateCustomCode_CombinedValidation(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		existingCodes []string
		expectError   bool
		expectedErr   error
	}{
		{
			name:          "valid - all checks pass",
			code:          "my-code-123",
			existingCodes: []string{},
			expectError:   false,
		},
		{
			name:          "invalid - too short and special char",
			code:          "ab@",
			existingCodes: []string{},
			expectError:   true,
			expectedErr:   ErrInvalidCustomCode,
		},
		{
			name:          "invalid - reserved and too short",
			code:          "api",
			existingCodes: []string{},
			expectError:   true,
			expectedErr:   ErrInvalidCustomCode,
		},
		{
			name:          "invalid - valid format but unavailable",
			code:          "mycode",
			existingCodes: []string{"mycode"},
			expectError:   true,
			expectedErr:   ErrCustomCodeUnavailable,
		},
		{
			name:          "invalid - too long with special chars",
			code:          "this-is-way-too-long-code-name",
			existingCodes: []string{},
			expectError:   true,
			expectedErr:   ErrInvalidCustomCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			storage := NewMockStorageWithCode(tt.existingCodes...)
			generator := NewRandomIDGenerator(storage)
			ctx := context.Background()

			// Act
			err := generator.ValidateCustomCode(ctx, tt.code)

			// Assert
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("Expected error %v, got: %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
