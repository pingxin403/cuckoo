package readreceipt

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// HTTPHandler handles HTTP requests for read receipts
type HTTPHandler struct {
	service *ReadReceiptService
}

// NewHTTPHandler creates a new HTTP handler for read receipts
func NewHTTPHandler(service *ReadReceiptService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// MarkAsReadRequest represents the request body for marking a message as read
type MarkAsReadRequest struct {
	MsgID            string `json:"msg_id"`
	ReaderID         string `json:"reader_id"`
	SenderID         string `json:"sender_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationType string `json:"conversation_type"`
	DeviceID         string `json:"device_id,omitempty"`
}

// MarkAsReadResponse represents the response for marking a message as read
type MarkAsReadResponse struct {
	Success bool         `json:"success"`
	Receipt *ReadReceipt `json:"receipt,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// GetUnreadCountResponse represents the response for getting unread count
type GetUnreadCountResponse struct {
	UserID string `json:"user_id"`
	Count  int    `json:"count"`
}

// GetReadReceiptsResponse represents the response for getting read receipts
type GetReadReceiptsResponse struct {
	MsgID    string         `json:"msg_id"`
	Receipts []*ReadReceipt `json:"receipts"`
}

// HandleMarkAsRead handles POST /api/v1/messages/read
// This implements Requirement 5.1 and 5.2
func (h *HTTPHandler) HandleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MarkAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, MarkAsReadResponse{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.MsgID == "" || req.ReaderID == "" || req.SenderID == "" {
		respondJSON(w, http.StatusBadRequest, MarkAsReadResponse{
			Success: false,
			Error:   "msg_id, reader_id, and sender_id are required",
		})
		return
	}

	receipt, err := h.service.MarkAsRead(
		r.Context(),
		req.MsgID,
		req.ReaderID,
		req.SenderID,
		req.ConversationID,
		req.ConversationType,
		req.DeviceID,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, MarkAsReadResponse{
			Success: false,
			Error:   "Failed to mark message as read: " + err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, MarkAsReadResponse{
		Success: true,
		Receipt: receipt,
	})
}

// HandleGetUnreadCount handles GET /api/v1/messages/unread/count?user_id={user_id}
func (h *HTTPHandler) HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get unread count: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, GetUnreadCountResponse{
		UserID: userID,
		Count:  count,
	})
}

// HandleGetUnreadMessages handles GET /api/v1/messages/unread?user_id={user_id}&limit={limit}&offset={offset}
func (h *HTTPHandler) HandleGetUnreadMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	limit := 50 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	msgIDs, err := h.service.GetUnreadMessages(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get unread messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"msg_ids": msgIDs,
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleGetReadReceipts handles GET /api/v1/messages/{msg_id}/receipts
func (h *HTTPHandler) HandleGetReadReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	msgID := r.URL.Query().Get("msg_id")
	if msgID == "" {
		http.Error(w, "msg_id parameter is required", http.StatusBadRequest)
		return
	}

	receipts, err := h.service.GetReadReceipts(r.Context(), msgID)
	if err != nil {
		http.Error(w, "Failed to get read receipts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, GetReadReceiptsResponse{
		MsgID:    msgID,
		Receipts: receipts,
	})
}

// HandleMarkConversationAsRead handles POST /api/v1/conversations/read
func (h *HTTPHandler) HandleMarkConversationAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID         string `json:"user_id"`
		ConversationID string `json:"conversation_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.ConversationID == "" {
		http.Error(w, "user_id and conversation_id are required", http.StatusBadRequest)
		return
	}

	rowsAffected, err := h.service.MarkConversationAsRead(r.Context(), req.UserID, req.ConversationID)
	if err != nil {
		http.Error(w, "Failed to mark conversation as read: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"rows_affected": rowsAffected,
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
