package service

import (
	"fmt"
	"regexp"
	"strings"
)

// UUID v4 regex pattern
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
// where x is any hexadecimal digit and y is one of 8, 9, A, or B
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// ValidateDeviceID validates that a device_id is a valid UUID v4
// Validates: Requirements 15.5, 15.9
func ValidateDeviceID(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device_id cannot be empty")
	}

	// Normalize to lowercase for validation
	deviceID = strings.ToLower(deviceID)

	// Check UUID v4 format
	if !uuidV4Pattern.MatchString(deviceID) {
		return fmt.Errorf("device_id must be a valid UUID v4 format")
	}

	return nil
}

// IsValidUUIDv4 checks if a string is a valid UUID v4
func IsValidUUIDv4(s string) bool {
	return uuidV4Pattern.MatchString(strings.ToLower(s))
}
