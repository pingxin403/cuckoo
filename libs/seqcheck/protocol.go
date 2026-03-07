package seqcheck

import (
	"encoding/json"
	"fmt"
)

// Message represents a message in the IM system
type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Sequence       int64  `json:"sequence"`
	Content        string `json:"content"`
	SenderID       string `json:"sender_id"`
	Timestamp      int64  `json:"timestamp"`
}

// GapFillRequest represents a request to fill gaps in message sequence
// Sent from client to server
type GapFillRequest struct {
	ConversationID string     `json:"conversation_id"`
	Gaps           []GapRange `json:"gaps"`
	RequestID      string     `json:"request_id"`
}

// GapFillResponse represents the server's response to a gap fill request
// Sent from server to client
type GapFillResponse struct {
	RequestID string    `json:"request_id"`
	Messages  []Message `json:"messages"`
	NotFound  []int64   `json:"not_found"` // Sequences that couldn't be found
}

// BuildGapFillRequest creates a gap fill request from a list of gaps
func BuildGapFillRequest(conversationID string, gaps []GapRange, requestID string) *GapFillRequest {
	if requestID == "" {
		requestID = generateRequestID()
	}

	return &GapFillRequest{
		ConversationID: conversationID,
		Gaps:           gaps,
		RequestID:      requestID,
	}
}

// ProcessGapFillResponse processes a gap fill response and fills the gaps in the sequence checker
func ProcessGapFillResponse(sc *SequenceChecker, response *GapFillResponse) error {
	if response == nil {
		return fmt.Errorf("response is nil")
	}

	if len(response.Messages) == 0 && len(response.NotFound) == 0 {
		return fmt.Errorf("empty response")
	}

	// Extract conversation ID from first message (if any)
	var conversationID string
	if len(response.Messages) > 0 {
		conversationID = response.Messages[0].ConversationID
	}

	// Fill gaps with received messages
	for _, msg := range response.Messages {
		if msg.ConversationID != "" {
			conversationID = msg.ConversationID
		}
		sc.FillGap(conversationID, msg.Sequence)
	}

	return nil
}

// ToJSON serializes the request to JSON
func (r *GapFillRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON deserializes the request from JSON
func (r *GapFillRequest) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// ToJSON serializes the response to JSON
func (r *GapFillResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON deserializes the response from JSON
func (r *GapFillResponse) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}

// GetTotalGapSize returns the total number of missing sequences in the request
func (r *GapFillRequest) GetTotalGapSize() int64 {
	total := int64(0)
	for _, gap := range r.Gaps {
		total += gap.EndSeq - gap.StartSeq + 1
	}
	return total
}

// Validate validates the gap fill request
func (r *GapFillRequest) Validate() error {
	if r.ConversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}

	if r.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	if len(r.Gaps) == 0 {
		return fmt.Errorf("gaps list is empty")
	}

	// Validate each gap
	for i, gap := range r.Gaps {
		if gap.StartSeq <= 0 {
			return fmt.Errorf("gap %d: start_seq must be positive", i)
		}
		if gap.EndSeq < gap.StartSeq {
			return fmt.Errorf("gap %d: end_seq must be >= start_seq", i)
		}
		if gap.ConversationID != "" && gap.ConversationID != r.ConversationID {
			return fmt.Errorf("gap %d: conversation_id mismatch", i)
		}
	}

	return nil
}

// Validate validates the gap fill response
func (r *GapFillResponse) Validate() error {
	if r.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}

	// Validate messages
	seenSeqs := make(map[int64]bool)
	for i, msg := range r.Messages {
		if msg.Sequence <= 0 {
			return fmt.Errorf("message %d: sequence must be positive", i)
		}
		if seenSeqs[msg.Sequence] {
			return fmt.Errorf("message %d: duplicate sequence %d", i, msg.Sequence)
		}
		seenSeqs[msg.Sequence] = true
	}

	return nil
}

// generateRequestID generates a unique request ID
// In production, this should use UUID or similar
func generateRequestID() string {
	// Simple implementation for now
	// In production, use: uuid.New().String()
	return fmt.Sprintf("req-%d", 0) // Placeholder
}

// CoversAllGaps checks if the response covers all requested gaps
func CoversAllGaps(request *GapFillRequest, response *GapFillResponse) bool {
	// Build set of all requested sequences
	requestedSeqs := make(map[int64]bool)
	for _, gap := range request.Gaps {
		for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
			requestedSeqs[seq] = true
		}
	}

	// Mark sequences as covered
	for _, msg := range response.Messages {
		delete(requestedSeqs, msg.Sequence)
	}

	for _, seq := range response.NotFound {
		delete(requestedSeqs, seq)
	}

	// All sequences should be accounted for
	return len(requestedSeqs) == 0
}

// GetMissingSequences returns the list of sequences that are still missing
// after processing the response
func GetMissingSequences(request *GapFillRequest, response *GapFillResponse) []int64 {
	// Build set of all requested sequences
	requestedSeqs := make(map[int64]bool)
	for _, gap := range request.Gaps {
		for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
			requestedSeqs[seq] = true
		}
	}

	// Remove sequences that were found
	for _, msg := range response.Messages {
		delete(requestedSeqs, msg.Sequence)
	}

	// Convert to sorted list
	missing := make([]int64, 0, len(requestedSeqs))
	for seq := range requestedSeqs {
		missing = append(missing, seq)
	}

	// Sort for consistent output
	for i := 0; i < len(missing)-1; i++ {
		for j := i + 1; j < len(missing); j++ {
			if missing[i] > missing[j] {
				missing[i], missing[j] = missing[j], missing[i]
			}
		}
	}

	return missing
}
