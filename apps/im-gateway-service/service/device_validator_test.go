package service

import (
	"testing"
)

func TestValidateDeviceID(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  string
		wantError bool
	}{
		{
			name:      "valid UUID v4",
			deviceID:  "550e8400-e29b-41d4-a716-446655440000",
			wantError: false,
		},
		{
			name:      "valid UUID v4 uppercase",
			deviceID:  "550E8400-E29B-41D4-A716-446655440000",
			wantError: false,
		},
		{
			name:      "valid UUID v4 mixed case",
			deviceID:  "550e8400-E29b-41D4-a716-446655440000",
			wantError: false,
		},
		{
			name:      "empty device_id",
			deviceID:  "",
			wantError: true,
		},
		{
			name:      "invalid UUID v1",
			deviceID:  "550e8400-e29b-11d4-a716-446655440000",
			wantError: true,
		},
		{
			name:      "invalid UUID v3",
			deviceID:  "550e8400-e29b-31d4-a716-446655440000",
			wantError: true,
		},
		{
			name:      "invalid UUID v5",
			deviceID:  "550e8400-e29b-51d4-a716-446655440000",
			wantError: true,
		},
		{
			name:      "invalid format - no dashes",
			deviceID:  "550e8400e29b41d4a716446655440000",
			wantError: true,
		},
		{
			name:      "invalid format - wrong length",
			deviceID:  "550e8400-e29b-41d4-a716",
			wantError: true,
		},
		{
			name:      "invalid format - non-hex characters",
			deviceID:  "550e8400-e29b-41d4-a716-44665544000g",
			wantError: true,
		},
		{
			name:      "invalid format - wrong variant",
			deviceID:  "550e8400-e29b-41d4-c716-446655440000",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceID(tt.deviceID)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateDeviceID() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestIsValidUUIDv4(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid UUID v4",
			input: "550e8400-e29b-41d4-a716-446655440000",
			want:  true,
		},
		{
			name:  "valid UUID v4 uppercase",
			input: "550E8400-E29B-41D4-A716-446655440000",
			want:  true,
		},
		{
			name:  "invalid UUID v1",
			input: "550e8400-e29b-11d4-a716-446655440000",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "random string",
			input: "not-a-uuid",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUUIDv4(tt.input); got != tt.want {
				t.Errorf("IsValidUUIDv4() = %v, want %v", got, tt.want)
			}
		})
	}
}
