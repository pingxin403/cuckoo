package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pingxin403/cuckoo/api/gen/authpb"
	"github.com/pingxin403/cuckoo/libs/observability"
)

const testSecret = "test-secret-key-for-testing-only"

// Helper function to create a test observability instance
func createTestObservability() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:   "auth-service-test",
		EnableMetrics: false,
		LogLevel:      "error",
	})
	return obs
}

// Helper function to generate a valid token
func generateTestToken(userID, deviceID string, expiry time.Duration) string {
	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	return tokenString
}

// Helper function to generate a token with invalid signature
func generateInvalidSignatureToken(userID, deviceID string) string {
	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("wrong-secret"))
	return tokenString
}

func TestValidateToken_ValidToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	userID := "user123"
	deviceID := "device456"
	token := generateTestToken(userID, deviceID, 15*time.Minute)

	req := &authpb.ValidateTokenRequest{
		AccessToken: token,
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !resp.Valid {
		t.Errorf("Expected token to be valid, got invalid with error: %s", resp.ErrorMessage)
	}
	if resp.UserId != userID {
		t.Errorf("Expected user_id %s, got %s", userID, resp.UserId)
	}
	if resp.DeviceId != deviceID {
		t.Errorf("Expected device_id %s, got %s", deviceID, resp.DeviceId)
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	// Generate an expired token (expired 1 hour ago)
	token := generateTestToken("user123", "device456", -1*time.Hour)

	req := &authpb.ValidateTokenRequest{
		AccessToken: token,
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (expired)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_TOKEN_EXPIRED {
		t.Errorf("Expected error code TOKEN_EXPIRED, got %v", resp.ErrorCode)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	token := generateInvalidSignatureToken("user123", "device456")

	req := &authpb.ValidateTokenRequest{
		AccessToken: token,
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (bad signature)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_SIGNATURE {
		t.Errorf("Expected error code INVALID_SIGNATURE, got %v", resp.ErrorCode)
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	req := &authpb.ValidateTokenRequest{
		AccessToken: "",
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (empty)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_TOKEN {
		t.Errorf("Expected error code INVALID_TOKEN, got %v", resp.ErrorCode)
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	req := &authpb.ValidateTokenRequest{
		AccessToken: "not.a.valid.jwt.token",
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (malformed)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_MALFORMED_TOKEN {
		t.Errorf("Expected error code MALFORMED_TOKEN, got %v", resp.ErrorCode)
	}
}

func TestValidateToken_MissingUserID(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	// Generate token without user_id
	token := generateTestToken("", "device456", 15*time.Minute)

	req := &authpb.ValidateTokenRequest{
		AccessToken: token,
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (missing user_id)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_MISSING_CLAIMS {
		t.Errorf("Expected error code MISSING_CLAIMS, got %v", resp.ErrorCode)
	}
}

func TestValidateToken_MissingDeviceID(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	// Generate token without device_id
	token := generateTestToken("user123", "", 15*time.Minute)

	req := &authpb.ValidateTokenRequest{
		AccessToken: token,
	}

	// Act
	resp, err := service.ValidateToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.Valid {
		t.Error("Expected token to be invalid (missing device_id)")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_MISSING_CLAIMS {
		t.Errorf("Expected error code MISSING_CLAIMS, got %v", resp.ErrorCode)
	}
}

func TestRefreshToken_ValidRefreshToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	userID := "user123"
	deviceID := "device456"
	refreshToken := generateTestToken(userID, deviceID, 7*24*time.Hour)

	req := &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// Act
	resp, err := service.RefreshToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("Expected access token to be generated")
	}
	if resp.RefreshToken == "" {
		t.Error("Expected new refresh token to be generated")
	}

	// Verify the new access token is valid
	validateReq := &authpb.ValidateTokenRequest{
		AccessToken: resp.AccessToken,
	}
	validateResp, err := service.ValidateToken(ctx, validateReq)
	if err != nil {
		t.Fatalf("Expected no error validating new token, got: %v", err)
	}
	if !validateResp.Valid {
		t.Error("Expected new access token to be valid")
	}
	if validateResp.UserId != userID {
		t.Errorf("Expected user_id %s in new token, got %s", userID, validateResp.UserId)
	}
	if validateResp.DeviceId != deviceID {
		t.Errorf("Expected device_id %s in new token, got %s", deviceID, validateResp.DeviceId)
	}
}

func TestRefreshToken_ExpiredRefreshToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	// Generate an expired refresh token
	refreshToken := generateTestToken("user123", "device456", -1*time.Hour)

	req := &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// Act
	resp, err := service.RefreshToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.AccessToken != "" {
		t.Error("Expected no access token for expired refresh token")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_TOKEN_EXPIRED {
		t.Errorf("Expected error code TOKEN_EXPIRED, got %v", resp.ErrorCode)
	}
}

func TestRefreshToken_InvalidRefreshToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	req := &authpb.RefreshTokenRequest{
		RefreshToken: "invalid.token.here",
	}

	// Act
	resp, err := service.RefreshToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.AccessToken != "" {
		t.Error("Expected no access token for invalid refresh token")
	}
	if resp.ErrorCode != authpb.AuthErrorCode_AUTH_ERROR_CODE_INVALID_REFRESH_TOKEN {
		t.Errorf("Expected error code INVALID_REFRESH_TOKEN, got %v", resp.ErrorCode)
	}
}

func TestRefreshToken_EmptyRefreshToken(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	req := &authpb.RefreshTokenRequest{
		RefreshToken: "",
	}

	// Act
	_, err := service.RefreshToken(ctx, req)

	// Assert
	if err == nil {
		t.Error("Expected error for empty refresh token")
	}
}

func TestRefreshToken_PreservesUserID(t *testing.T) {
	// Arrange
	obs := createTestObservability()
	service := NewAuthServiceServer(testSecret, obs)
	ctx := context.Background()

	userID := "user123"
	deviceID := "device456"
	refreshToken := generateTestToken(userID, deviceID, 7*24*time.Hour)

	req := &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	}

	// Act
	resp, err := service.RefreshToken(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Parse the new access token to verify user_id is preserved
	token, _ := jwt.ParseWithClaims(resp.AccessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(testSecret), nil
	})

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("Failed to parse claims from new access token")
	}

	if claims.UserID != userID {
		t.Errorf("Expected user_id %s to be preserved, got %s", userID, claims.UserID)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("Expected device_id %s to be preserved, got %s", deviceID, claims.DeviceID)
	}
}
