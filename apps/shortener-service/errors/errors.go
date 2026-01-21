package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error codes for the shortener service
// Requirements: 9.5
const (
	// Client errors (4xx)
	CodeInvalidURL            = "INVALID_URL"
	CodeURLTooLong            = "URL_TOO_LONG"
	CodeInvalidProtocol       = "INVALID_PROTOCOL"
	CodeMaliciousPattern      = "MALICIOUS_PATTERN"
	CodeInvalidShortCode      = "INVALID_SHORT_CODE"
	CodeShortCodeNotFound     = "SHORT_CODE_NOT_FOUND"
	CodeShortCodeExists       = "SHORT_CODE_EXISTS"
	CodeShortCodeExpired      = "SHORT_CODE_EXPIRED"
	CodeInvalidCustomCode     = "INVALID_CUSTOM_CODE"
	CodeCustomCodeUnavailable = "CUSTOM_CODE_UNAVAILABLE"
	CodeRateLimitExceeded     = "RATE_LIMIT_EXCEEDED"

	// Server errors (5xx)
	CodeInternalError      = "INTERNAL_ERROR"
	CodeStorageUnavailable = "STORAGE_UNAVAILABLE"
	CodeCacheUnavailable   = "CACHE_UNAVAILABLE"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ServiceError represents a structured error response
// Requirements: 9.5
type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewServiceError creates a new ServiceError
func NewServiceError(code, message, details string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ToGRPCStatus converts a ServiceError to a gRPC status
// Requirements: 9.5
func (e *ServiceError) ToGRPCStatus() error {
	grpcCode := mapErrorCodeToGRPC(e.Code)
	return status.Errorf(grpcCode, "%s: %s", e.Code, e.Message)
}

// mapErrorCodeToGRPC maps service error codes to gRPC status codes
func mapErrorCodeToGRPC(code string) codes.Code {
	switch code {
	// Client errors
	case CodeInvalidURL, CodeURLTooLong, CodeInvalidProtocol, CodeMaliciousPattern:
		return codes.InvalidArgument
	case CodeInvalidShortCode, CodeInvalidCustomCode:
		return codes.InvalidArgument
	case CodeShortCodeNotFound:
		return codes.NotFound
	case CodeShortCodeExists, CodeCustomCodeUnavailable:
		return codes.AlreadyExists
	case CodeShortCodeExpired:
		return codes.FailedPrecondition
	case CodeRateLimitExceeded:
		return codes.ResourceExhausted

	// Server errors
	case CodeStorageUnavailable, CodeCacheUnavailable, CodeServiceUnavailable:
		return codes.Unavailable
	case CodeInternalError:
		return codes.Internal

	default:
		return codes.Unknown
	}
}

// Common error constructors
// Requirements: 9.5

// NewInvalidURLError creates an invalid URL error
func NewInvalidURLError(details string) *ServiceError {
	return NewServiceError(CodeInvalidURL, "Invalid URL provided", details)
}

// NewURLTooLongError creates a URL too long error
func NewURLTooLongError(length int) *ServiceError {
	return NewServiceError(CodeURLTooLong, "URL exceeds maximum length", fmt.Sprintf("length: %d", length))
}

// NewInvalidProtocolError creates an invalid protocol error
func NewInvalidProtocolError(protocol string) *ServiceError {
	return NewServiceError(CodeInvalidProtocol, "Invalid URL protocol", fmt.Sprintf("protocol: %s", protocol))
}

// NewMaliciousPatternError creates a malicious pattern error
func NewMaliciousPatternError(pattern string) *ServiceError {
	return NewServiceError(CodeMaliciousPattern, "Malicious pattern detected in URL", pattern)
}

// NewShortCodeNotFoundError creates a short code not found error
func NewShortCodeNotFoundError(shortCode string) *ServiceError {
	return NewServiceError(CodeShortCodeNotFound, "Short code not found", shortCode)
}

// NewShortCodeExistsError creates a short code already exists error
func NewShortCodeExistsError(shortCode string) *ServiceError {
	return NewServiceError(CodeShortCodeExists, "Short code already exists", shortCode)
}

// NewShortCodeExpiredError creates a short code expired error
func NewShortCodeExpiredError(shortCode string) *ServiceError {
	return NewServiceError(CodeShortCodeExpired, "Short code has expired", shortCode)
}

// NewInvalidCustomCodeError creates an invalid custom code error
func NewInvalidCustomCodeError(details string) *ServiceError {
	return NewServiceError(CodeInvalidCustomCode, "Invalid custom code", details)
}

// NewCustomCodeUnavailableError creates a custom code unavailable error
func NewCustomCodeUnavailableError(code string) *ServiceError {
	return NewServiceError(CodeCustomCodeUnavailable, "Custom code is not available", code)
}

// NewRateLimitExceededError creates a rate limit exceeded error
func NewRateLimitExceededError(retryAfter int) *ServiceError {
	return NewServiceError(CodeRateLimitExceeded, "Rate limit exceeded", fmt.Sprintf("retry_after: %d seconds", retryAfter))
}

// NewStorageUnavailableError creates a storage unavailable error
func NewStorageUnavailableError(details string) *ServiceError {
	return NewServiceError(CodeStorageUnavailable, "Storage service unavailable", details)
}

// NewInternalError creates an internal error
func NewInternalError(details string) *ServiceError {
	return NewServiceError(CodeInternalError, "Internal server error", details)
}
