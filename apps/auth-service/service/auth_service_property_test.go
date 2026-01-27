//go:build property
// +build property

package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pingxin403/cuckoo/api/gen/go/authpb"
	"pgregory.net/rapid"
)

// Property 1: Valid tokens always validate successfully
// **Validates: Requirements 11.2, 11.3**
func TestProperty_ValidTokensAlwaysValidate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user_id and device_id
		userID := rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")

		// Generate a valid token with random expiry (1 minute to 24 hours in the future)
		expiryMinutes := rapid.IntRange(1, 1440).Draw(t, "expiry_minutes")
		expiry := time.Duration(expiryMinutes) * time.Minute

		// Create service and generate token
		service := NewAuthServiceServer(testSecret)
		token := generateTestToken(userID, deviceID, expiry)

		// Validate the token
		req := &authpb.ValidateTokenRequest{
			AccessToken: token,
		}
		resp, err := service.ValidateToken(context.Background(), req)

		// Property: Valid tokens must always validate successfully
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !resp.Valid {
			t.Fatalf("Expected token to be valid, got invalid with error: %s", resp.ErrorMessage)
		}
		if resp.UserId != userID {
			t.Fatalf("Expected user_id %s, got %s", userID, resp.UserId)
		}
		if resp.DeviceId != deviceID {
			t.Fatalf("Expected device_id %s, got %s", deviceID, resp.DeviceId)
		}
	})
}

// Property 2: Expired tokens always fail validation
// **Validates: Requirements 11.2, 11.3**
func TestProperty_ExpiredTokensAlwaysFail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user_id and device_id
		userID := rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")

		// Generate an expired token (1 second to 24 hours in the past)
		expiryMinutes := rapid.IntRange(-1440, -1).Draw(t, "expiry_minutes")
		expiry := time.Duration(expiryMinutes) * time.Minute

		// Create service and generate expired token
		service := NewAuthServiceServer(testSecret)
		token := generateTestToken(userID, deviceID, expiry)

		// Validate the token
		req := &authpb.ValidateTokenRequest{
			AccessToken: token,
		}
		resp, err := service.ValidateToken(context.Background(), req)

		// Property: Expired tokens must always fail validation
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp.Valid {
			t.Fatal("Expected token to be invalid (expired)")
		}
		if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_TOKEN_EXPIRED {
			t.Fatalf("Expected error code TOKEN_EXPIRED, got %v", resp.ErrorCode)
		}
	})
}

// Property 3: Token refresh preserves user_id and device_id
// **Validates: Requirements 11.2**
func TestProperty_TokenRefreshPreservesIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user_id and device_id
		userID := rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")

		// Create service and generate refresh token
		service := NewAuthServiceServer(testSecret)
		refreshToken := generateTestToken(userID, deviceID, 7*24*time.Hour)

		// Refresh the token
		req := &authpb.RefreshTokenRequest{
			RefreshToken: refreshToken,
		}
		resp, err := service.RefreshToken(context.Background(), req)

		// Property: Refresh must succeed and preserve identity
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp.AccessToken == "" {
			t.Fatal("Expected access token to be generated")
		}
		if resp.RefreshToken == "" {
			t.Fatal("Expected new refresh token to be generated")
		}

		// Parse the new access token to verify identity is preserved
		token, err := jwt.ParseWithClaims(resp.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(testSecret), nil
		})
		if err != nil {
			t.Fatalf("Failed to parse new access token: %v", err)
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			t.Fatal("Failed to extract claims from new access token")
		}

		// Property: user_id and device_id must be preserved
		if claims.UserID != userID {
			t.Fatalf("Expected user_id %s to be preserved, got %s", userID, claims.UserID)
		}
		if claims.DeviceID != deviceID {
			t.Fatalf("Expected device_id %s to be preserved, got %s", deviceID, claims.DeviceID)
		}
	})
}

// Property 4: Invalid signatures always fail validation
// **Validates: Requirements 11.2, 11.3**
func TestProperty_InvalidSignaturesAlwaysFail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user_id and device_id
		userID := rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")

		// Generate a token with wrong secret
		wrongSecret := rapid.StringMatching(`^[a-zA-Z0-9]{16,64}$`).Draw(t, "wrong_secret")
		if wrongSecret == testSecret {
			// Skip if we randomly generated the correct secret
			t.Skip("Generated correct secret by chance")
		}

		claims := &Claims{
			UserID:   userID,
			DeviceID: deviceID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(wrongSecret))

		// Create service and validate token
		service := NewAuthServiceServer(testSecret)
		req := &authpb.ValidateTokenRequest{
			AccessToken: tokenString,
		}
		resp, err := service.ValidateToken(context.Background(), req)

		// Property: Invalid signatures must always fail validation
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp.Valid {
			t.Fatal("Expected token to be invalid (bad signature)")
		}
		if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_SIGNATURE {
			t.Fatalf("Expected error code INVALID_SIGNATURE, got %v", resp.ErrorCode)
		}
	})
}

// Property 5: Missing required claims always fail validation
// **Validates: Requirements 11.2, 11.3**
func TestProperty_MissingClaimsAlwaysFail(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Randomly choose which claim to omit
		omitUserID := rapid.Bool().Draw(t, "omit_user_id")

		var userID, deviceID string
		if omitUserID {
			// Omit user_id
			userID = ""
			deviceID = rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")
		} else {
			// Omit device_id
			userID = rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
			deviceID = ""
		}

		// Create service and generate token with missing claim
		service := NewAuthServiceServer(testSecret)
		token := generateTestToken(userID, deviceID, 15*time.Minute)

		// Validate the token
		req := &authpb.ValidateTokenRequest{
			AccessToken: token,
		}
		resp, err := service.ValidateToken(context.Background(), req)

		// Property: Missing required claims must always fail validation
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp.Valid {
			t.Fatal("Expected token to be invalid (missing claims)")
		}
		if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_MISSING_CLAIMS {
			t.Fatalf("Expected error code MISSING_CLAIMS, got %v", resp.ErrorCode)
		}
	})
}

// Property 6: Token expiration times are correctly set
// **Validates: Requirements 11.2**
func TestProperty_TokenExpirationTimesCorrect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random user_id and device_id
		userID := rapid.StringMatching(`^[a-zA-Z0-9]{8,32}$`).Draw(t, "user_id")
		deviceID := rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`).Draw(t, "device_id")

		// Create service and generate refresh token
		service := NewAuthServiceServer(testSecret)
		refreshToken := generateTestToken(userID, deviceID, 7*24*time.Hour)

		// Record time before refresh
		beforeRefresh := time.Now()

		// Refresh the token
		req := &authpb.RefreshTokenRequest{
			RefreshToken: refreshToken,
		}
		resp, err := service.RefreshToken(context.Background(), req)

		// Record time after refresh
		afterRefresh := time.Now()

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Parse the new access token
		accessToken, err := jwt.ParseWithClaims(resp.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(testSecret), nil
		})
		if err != nil {
			t.Fatalf("Failed to parse access token: %v", err)
		}

		accessClaims := accessToken.Claims.(*Claims)

		// Parse the new refresh token
		newRefreshToken, err := jwt.ParseWithClaims(resp.RefreshToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(testSecret), nil
		})
		if err != nil {
			t.Fatalf("Failed to parse refresh token: %v", err)
		}

		refreshClaims := newRefreshToken.Claims.(*Claims)

		// Property: Access token should expire in ~15 minutes
		accessExpiry := accessClaims.ExpiresAt.Time
		expectedAccessExpiry := beforeRefresh.Add(15 * time.Minute)
		accessExpiryDiff := accessExpiry.Sub(expectedAccessExpiry).Abs()
		if accessExpiryDiff > 5*time.Second {
			t.Fatalf("Access token expiry is off by %v (expected ~15 minutes from now)", accessExpiryDiff)
		}

		// Property: Refresh token should expire in ~7 days
		refreshExpiry := refreshClaims.ExpiresAt.Time
		expectedRefreshExpiry := beforeRefresh.Add(7 * 24 * time.Hour)
		refreshExpiryDiff := refreshExpiry.Sub(expectedRefreshExpiry).Abs()
		if refreshExpiryDiff > 5*time.Second {
			t.Fatalf("Refresh token expiry is off by %v (expected ~7 days from now)", refreshExpiryDiff)
		}

		// Property: Both tokens should be issued around the same time
		issuedAtDiff := accessClaims.IssuedAt.Time.Sub(refreshClaims.IssuedAt.Time).Abs()
		if issuedAtDiff > 1*time.Second {
			t.Fatalf("Token issued times differ by %v (should be nearly identical)", issuedAtDiff)
		}

		// Property: Issued time should be between beforeRefresh and afterRefresh (with 1 second tolerance)
		if accessClaims.IssuedAt.Time.Before(beforeRefresh.Add(-1*time.Second)) || accessClaims.IssuedAt.Time.After(afterRefresh.Add(1*time.Second)) {
			t.Fatalf("Token issued time %v is outside the expected range [%v, %v]",
				accessClaims.IssuedAt.Time, beforeRefresh, afterRefresh)
		}
	})
}
