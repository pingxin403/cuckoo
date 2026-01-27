package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// HTTPHandler handles HTTP requests for offline message retrieval
type HTTPHandler struct {
	store *OfflineStore
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(store *OfflineStore) *HTTPHandler {
	return &HTTPHandler{store: store}
}

// OfflineMessagesResponse represents the response for offline messages
type OfflineMessagesResponse struct {
	Messages   []OfflineMessage `json:"messages"`
	NextCursor int64            `json:"next_cursor"`
	HasMore    bool             `json:"has_more"`
}

// MessageCountResponse represents the response for message count
type MessageCountResponse struct {
	Count int64 `json:"count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetOfflineMessages handles GET /api/v1/offline?cursor={last_id}&limit={page_size}
func (h *HTTPHandler) GetOfflineMessages(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context (set by authentication middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "User ID not found in context")
		return
	}

	// Parse query parameters
	cursor, err := h.parseCursor(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_cursor", err.Error())
		return
	}

	limit, err := h.parseLimit(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_limit", err.Error())
		return
	}

	// Retrieve messages from storage
	messages, err := h.store.GetMessages(r.Context(), userID, cursor, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "storage_error", "Failed to retrieve messages")
		return
	}

	// Prepare response
	response := OfflineMessagesResponse{
		Messages: messages,
		HasMore:  len(messages) == limit,
	}

	// Set next cursor if there are more messages
	if response.HasMore && len(messages) > 0 {
		response.NextCursor = messages[len(messages)-1].ID
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetMessageCount handles GET /api/v1/offline/count
func (h *HTTPHandler) GetMessageCount(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context (set by authentication middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		h.writeError(w, http.StatusUnauthorized, "unauthorized", "User ID not found in context")
		return
	}

	// Get message count from storage
	count, err := h.store.GetMessageCount(r.Context(), userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "storage_error", "Failed to count messages")
		return
	}

	response := MessageCountResponse{
		Count: count,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// parseCursor parses the cursor query parameter
func (h *HTTPHandler) parseCursor(r *http.Request) (int64, error) {
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr == "" {
		return 0, nil // Default to 0 for first page
	}

	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cursor must be a valid integer")
	}

	if cursor < 0 {
		return 0, fmt.Errorf("cursor must be non-negative")
	}

	return cursor, nil
}

// parseLimit parses the limit query parameter
func (h *HTTPHandler) parseLimit(r *http.Request) (int, error) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return 100, nil // Default to 100
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, fmt.Errorf("limit must be a valid integer")
	}

	if limit <= 0 || limit > 100 {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}

	return limit, nil
}

// writeJSON writes a JSON response
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (h *HTTPHandler) writeError(w http.ResponseWriter, status int, errorCode string, message string) {
	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
	}
	h.writeJSON(w, status, response)
}

// AuthMiddleware is a simple JWT authentication middleware
// In production, this should validate JWT tokens and extract user_id
type AuthMiddleware struct {
	// In production, add JWT secret, token validator, etc.
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// Authenticate validates the JWT token and adds user_id to context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// In production, validate JWT token here
		// For now, we'll extract user_id from a simple token format
		// Example: "user-123" -> user_id = "user-123"
		userID := m.extractUserID(token)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user_id to context
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractUserID extracts user_id from token
// In production, this should validate JWT and extract claims
func (m *AuthMiddleware) extractUserID(token string) string {
	// Simplified implementation for testing
	// In production, use proper JWT validation
	if token == "" {
		return ""
	}
	return token // For testing, token is the user_id
}

// RegisterRoutes registers HTTP routes for offline message API
func RegisterRoutes(mux *http.ServeMux, store *OfflineStore) {
	handler := NewHTTPHandler(store)
	auth := NewAuthMiddleware()

	// Register routes with authentication middleware
	mux.Handle("/api/v1/offline", auth.Authenticate(http.HandlerFunc(handler.GetOfflineMessages)))
	mux.Handle("/api/v1/offline/count", auth.Authenticate(http.HandlerFunc(handler.GetMessageCount)))
}
