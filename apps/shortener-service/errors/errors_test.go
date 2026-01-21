package errors

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ServiceError
		want string
	}{
		{
			name: "error with details",
			err:  NewServiceError("TEST_CODE", "Test message", "test details"),
			want: "TEST_CODE: Test message (test details)",
		},
		{
			name: "error without details",
			err:  NewServiceError("TEST_CODE", "Test message", ""),
			want: "TEST_CODE: Test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ServiceError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceError_ToGRPCStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServiceError
		wantCode codes.Code
	}{
		{
			name:     "invalid URL maps to InvalidArgument",
			err:      NewInvalidURLError("test"),
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "not found maps to NotFound",
			err:      NewShortCodeNotFoundError("abc123"),
			wantCode: codes.NotFound,
		},
		{
			name:     "already exists maps to AlreadyExists",
			err:      NewShortCodeExistsError("abc123"),
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "rate limit maps to ResourceExhausted",
			err:      NewRateLimitExceededError(60),
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "storage unavailable maps to Unavailable",
			err:      NewStorageUnavailableError("MySQL down"),
			wantCode: codes.Unavailable,
		},
		{
			name:     "internal error maps to Internal",
			err:      NewInternalError("unexpected error"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := tt.err.ToGRPCStatus()
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Errorf("ToGRPCStatus() code = %v, want %v", st.Code(), tt.wantCode)
			}
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServiceError
		wantCode string
	}{
		{
			name:     "NewInvalidURLError",
			err:      NewInvalidURLError("invalid format"),
			wantCode: CodeInvalidURL,
		},
		{
			name:     "NewURLTooLongError",
			err:      NewURLTooLongError(3000),
			wantCode: CodeURLTooLong,
		},
		{
			name:     "NewInvalidProtocolError",
			err:      NewInvalidProtocolError("ftp"),
			wantCode: CodeInvalidProtocol,
		},
		{
			name:     "NewMaliciousPatternError",
			err:      NewMaliciousPatternError("javascript:"),
			wantCode: CodeMaliciousPattern,
		},
		{
			name:     "NewShortCodeNotFoundError",
			err:      NewShortCodeNotFoundError("abc123"),
			wantCode: CodeShortCodeNotFound,
		},
		{
			name:     "NewShortCodeExistsError",
			err:      NewShortCodeExistsError("abc123"),
			wantCode: CodeShortCodeExists,
		},
		{
			name:     "NewShortCodeExpiredError",
			err:      NewShortCodeExpiredError("abc123"),
			wantCode: CodeShortCodeExpired,
		},
		{
			name:     "NewInvalidCustomCodeError",
			err:      NewInvalidCustomCodeError("too short"),
			wantCode: CodeInvalidCustomCode,
		},
		{
			name:     "NewCustomCodeUnavailableError",
			err:      NewCustomCodeUnavailableError("mycode"),
			wantCode: CodeCustomCodeUnavailable,
		},
		{
			name:     "NewRateLimitExceededError",
			err:      NewRateLimitExceededError(60),
			wantCode: CodeRateLimitExceeded,
		},
		{
			name:     "NewStorageUnavailableError",
			err:      NewStorageUnavailableError("connection failed"),
			wantCode: CodeStorageUnavailable,
		},
		{
			name:     "NewInternalError",
			err:      NewInternalError("unexpected panic"),
			wantCode: CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("error code = %v, want %v", tt.err.Code, tt.wantCode)
			}
			if tt.err.Message == "" {
				t.Error("error message should not be empty")
			}
		})
	}
}

func TestMapErrorCodeToGRPC(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		wantCode codes.Code
	}{
		// Client errors
		{name: "INVALID_URL", code: CodeInvalidURL, wantCode: codes.InvalidArgument},
		{name: "URL_TOO_LONG", code: CodeURLTooLong, wantCode: codes.InvalidArgument},
		{name: "INVALID_PROTOCOL", code: CodeInvalidProtocol, wantCode: codes.InvalidArgument},
		{name: "MALICIOUS_PATTERN", code: CodeMaliciousPattern, wantCode: codes.InvalidArgument},
		{name: "INVALID_SHORT_CODE", code: CodeInvalidShortCode, wantCode: codes.InvalidArgument},
		{name: "INVALID_CUSTOM_CODE", code: CodeInvalidCustomCode, wantCode: codes.InvalidArgument},
		{name: "SHORT_CODE_NOT_FOUND", code: CodeShortCodeNotFound, wantCode: codes.NotFound},
		{name: "SHORT_CODE_EXISTS", code: CodeShortCodeExists, wantCode: codes.AlreadyExists},
		{name: "CUSTOM_CODE_UNAVAILABLE", code: CodeCustomCodeUnavailable, wantCode: codes.AlreadyExists},
		{name: "SHORT_CODE_EXPIRED", code: CodeShortCodeExpired, wantCode: codes.FailedPrecondition},
		{name: "RATE_LIMIT_EXCEEDED", code: CodeRateLimitExceeded, wantCode: codes.ResourceExhausted},

		// Server errors
		{name: "STORAGE_UNAVAILABLE", code: CodeStorageUnavailable, wantCode: codes.Unavailable},
		{name: "CACHE_UNAVAILABLE", code: CodeCacheUnavailable, wantCode: codes.Unavailable},
		{name: "SERVICE_UNAVAILABLE", code: CodeServiceUnavailable, wantCode: codes.Unavailable},
		{name: "INTERNAL_ERROR", code: CodeInternalError, wantCode: codes.Internal},

		// Unknown
		{name: "UNKNOWN_CODE", code: "UNKNOWN_CODE", wantCode: codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapErrorCodeToGRPC(tt.code); got != tt.wantCode {
				t.Errorf("mapErrorCodeToGRPC(%v) = %v, want %v", tt.code, got, tt.wantCode)
			}
		})
	}
}
