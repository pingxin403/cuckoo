package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pingxin403/cuckoo/api/gen/authpb"
	"github.com/pingxin403/cuckoo/libs/observability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

// AuthServiceServer implements the AuthService gRPC service
type AuthServiceServer struct {
	authpb.UnimplementedAuthServiceServer
	jwtSecret []byte
	obs       observability.Observability
}

// NewAuthServiceServer creates a new AuthServiceServer
func NewAuthServiceServer(jwtSecret string, obs observability.Observability) *AuthServiceServer {
	return &AuthServiceServer{
		jwtSecret: []byte(jwtSecret),
		obs:       obs,
	}
}

// ValidateToken validates a JWT authentication token
func (s *AuthServiceServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("auth_grpc_request_duration_seconds", duration, map[string]string{
			"method": "ValidateToken",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("auth_grpc_requests_total", map[string]string{
		"method": "ValidateToken",
	})

	if req.AccessToken == "" {
		s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
			"status": "failure",
			"reason": "empty_token",
		})
		return &authpb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_TOKEN,
			ErrorMessage: "Token is required",
		}, nil
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(req.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		// Check for specific error types using errors.Is
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
				"status": "failure",
				"reason": "expired",
			})
			return &authpb.ValidateTokenResponse{
				Valid:        false,
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_TOKEN_EXPIRED,
				ErrorMessage: "Token has expired",
			}, nil
		case errors.Is(err, jwt.ErrTokenMalformed):
			s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
				"status": "failure",
				"reason": "malformed",
			})
			return &authpb.ValidateTokenResponse{
				Valid:        false,
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_MALFORMED_TOKEN,
				ErrorMessage: "Token is malformed",
			}, nil
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
				"status": "failure",
				"reason": "invalid_signature",
			})
			return &authpb.ValidateTokenResponse{
				Valid:        false,
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_SIGNATURE,
				ErrorMessage: "Token signature is invalid",
			}, nil
		default:
			// Generic invalid token error
			s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
				"status": "failure",
				"reason": "invalid",
			})
			return &authpb.ValidateTokenResponse{
				Valid:        false,
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_TOKEN,
				ErrorMessage: fmt.Sprintf("Invalid token: %v", err),
			}, nil
		}
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
			"status": "failure",
			"reason": "invalid_claims",
		})
		return &authpb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_MALFORMED_TOKEN,
			ErrorMessage: "Invalid token claims",
		}, nil
	}

	// Validate required fields
	if claims.UserID == "" {
		s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
			"status": "failure",
			"reason": "missing_user_id",
		})
		return &authpb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_MISSING_CLAIMS,
			ErrorMessage: "Missing user_id in token",
		}, nil
	}

	if claims.DeviceID == "" {
		s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
			"status": "failure",
			"reason": "missing_device_id",
		})
		return &authpb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_MISSING_CLAIMS,
			ErrorMessage: "Missing device_id in token",
		}, nil
	}

	// Token is valid
	s.obs.Metrics().IncrementCounter("auth_token_validations_total", map[string]string{
		"status": "success",
	})
	return &authpb.ValidateTokenResponse{
		Valid:    true,
		UserId:   claims.UserID,
		DeviceId: claims.DeviceID,
	}, nil
}

// RefreshToken generates a new access token using a refresh token
func (s *AuthServiceServer) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.obs.Metrics().RecordDuration("auth_grpc_request_duration_seconds", duration, map[string]string{
			"method": "RefreshToken",
		})
	}()

	// Record request count
	s.obs.Metrics().IncrementCounter("auth_grpc_requests_total", map[string]string{
		"method": "RefreshToken",
	})

	if req.RefreshToken == "" {
		s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
			"status": "failure",
			"reason": "empty_refresh_token",
		})
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	// Parse and validate the refresh token
	token, err := jwt.ParseWithClaims(req.RefreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		// Check for specific error types using errors.Is
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
				"status": "failure",
				"reason": "expired",
			})
			return &authpb.RefreshTokenResponse{
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_TOKEN_EXPIRED,
				ErrorMessage: "Refresh token has expired",
			}, nil
		case errors.Is(err, jwt.ErrTokenMalformed):
			s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
				"status": "failure",
				"reason": "malformed",
			})
			return &authpb.RefreshTokenResponse{
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN,
				ErrorMessage: "Refresh token is malformed",
			}, nil
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
				"status": "failure",
				"reason": "invalid_signature",
			})
			return &authpb.RefreshTokenResponse{
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN,
				ErrorMessage: "Refresh token signature is invalid",
			}, nil
		default:
			s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
				"status": "failure",
				"reason": "invalid",
			})
			return &authpb.RefreshTokenResponse{
				ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN,
				ErrorMessage: fmt.Sprintf("Invalid refresh token: %v", err),
			}, nil
		}
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
			"status": "failure",
			"reason": "invalid_claims",
		})
		return &authpb.RefreshTokenResponse{
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN,
			ErrorMessage: "Invalid refresh token claims",
		}, nil
	}

	// Generate new access token (15 minutes expiration)
	accessTokenExpiry := time.Now().Add(15 * time.Minute)
	accessClaims := &Claims{
		UserID:   claims.UserID,
		DeviceID: claims.DeviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
			"status": "failure",
			"reason": "signing_error",
		})
		return &authpb.RefreshTokenResponse{
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to generate access token: %v", err),
		}, nil
	}

	// Generate new refresh token (7 days expiration)
	refreshTokenExpiry := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := &Claims{
		UserID:   claims.UserID,
		DeviceID: claims.DeviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshTokenExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
			"status": "failure",
			"reason": "signing_error",
		})
		return &authpb.RefreshTokenResponse{
			ErrorCode:    authpb.AuthErrorCode_AUTH_ERROR_CODE_INTERNAL_ERROR,
			ErrorMessage: fmt.Sprintf("Failed to generate refresh token: %v", err),
		}, nil
	}

	s.obs.Metrics().IncrementCounter("auth_token_generation_total", map[string]string{
		"status": "success",
	})
	return &authpb.RefreshTokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}
