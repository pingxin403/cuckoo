package seqcheck

import (
	"testing"

	"pgregory.net/rapid"
)

// Property 6: 补洞请求覆盖完整性
// For any set of gaps, BuildGapFillRequest should generate a request that:
// 1. Covers all missing sequence numbers
// 2. Does not include any already-received sequence numbers
// Validates: Requirements 8.1.2, 8.1.3
func TestProperty_GapFillRequestCoverage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		// Generate a random sequence with gaps
		maxSeq := rapid.Int64Range(20, 100).Draw(t, "maxSeq")

		// Randomly select which sequences to include (some will be missing)
		presentSeqs := make(map[int64]bool)
		for i := int64(1); i <= maxSeq; i++ {
			if rapid.Bool().Draw(t, "include") {
				presentSeqs[i] = true
			}
		}

		// Ensure at least first and last are present to create gaps
		presentSeqs[1] = true
		presentSeqs[maxSeq] = true

		// Record present sequences IN ORDER (important for SequenceChecker)
		for seq := int64(1); seq <= maxSeq; seq++ {
			if presentSeqs[seq] {
				sc.RecordSequence(conversationID, seq)
			}
		}

		// Get gaps from sequence checker
		gaps := sc.GetGaps(conversationID)

		// If no gaps, skip this test iteration
		if len(gaps) == 0 {
			t.Skip("No gaps generated, skipping test")
		}

		// Build gap fill request
		request := BuildGapFillRequest(conversationID, gaps, "test-request")

		// Verify request is valid
		if err := request.Validate(); err != nil {
			t.Fatalf("Generated invalid request: %v", err)
		}

		// Extract all sequences covered by the request
		requestedSeqs := make(map[int64]bool)
		for _, gap := range request.Gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				requestedSeqs[seq] = true
			}
		}

		// Calculate expected missing sequences (from gaps returned by GetGaps)
		expectedMissing := make(map[int64]bool)
		for _, gap := range gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				expectedMissing[seq] = true
			}
		}

		// Verify: Request covers all missing sequences from gaps
		for seq := range expectedMissing {
			if !requestedSeqs[seq] {
				t.Fatalf("Request missing sequence %d that should be requested", seq)
			}
		}

		// Verify: Request does not include already-received sequences
		for seq := range requestedSeqs {
			if presentSeqs[seq] {
				t.Fatalf("Request includes sequence %d that was already received", seq)
			}
		}

		// Verify: Request exactly matches expected missing sequences from gaps
		if len(requestedSeqs) != len(expectedMissing) {
			t.Fatalf("Request coverage mismatch: expected %d sequences, got %d sequences",
				len(expectedMissing), len(requestedSeqs))
		}
	})
}

// Property 7: Gap fill response processing completeness
// After processing a complete gap fill response, all gaps should be filled
func TestProperty_GapFillResponseProcessing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"
		sc := NewSequenceChecker(3, nil)

		// Create gaps
		maxSeq := rapid.Int64Range(20, 50).Draw(t, "maxSeq")
		presentSeqs := make(map[int64]bool)

		for i := int64(1); i <= maxSeq; i++ {
			if rapid.Bool().Draw(t, "include") {
				presentSeqs[i] = true
			}
		}

		// Ensure at least first and last are present
		presentSeqs[1] = true
		presentSeqs[maxSeq] = true

		// Record present sequences
		for seq := range presentSeqs {
			sc.RecordSequence(conversationID, seq)
		}

		// Get gaps
		gaps := sc.GetGaps(conversationID)
		if len(gaps) == 0 {
			t.Skip("No gaps generated, skipping test")
		}

		// Build request
		request := BuildGapFillRequest(conversationID, gaps, "test-request")

		// Simulate complete response (all missing messages found)
		response := &GapFillResponse{
			RequestID: request.RequestID,
			Messages:  make([]Message, 0),
			NotFound:  make([]int64, 0),
		}

		// Add all missing sequences to response
		for _, gap := range gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				response.Messages = append(response.Messages, Message{
					ID:             "msg-" + string(rune(seq)),
					ConversationID: conversationID,
					Sequence:       seq,
					Content:        "test message",
					SenderID:       "sender1",
					Timestamp:      1000 + seq,
				})
			}
		}

		// Process response
		err := ProcessGapFillResponse(sc, response)
		if err != nil {
			t.Fatalf("Failed to process response: %v", err)
		}

		// Verify all gaps are filled
		remainingGaps := sc.GetGaps(conversationID)
		if len(remainingGaps) != 0 {
			t.Fatalf("Expected all gaps to be filled, but %d gaps remain", len(remainingGaps))
		}
	})
}

// Property 8: Gap fill request idempotence
// Building a request from the same gaps multiple times should produce equivalent requests
func TestProperty_GapFillRequestIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"

		// Generate random gaps
		numGaps := rapid.IntRange(1, 10).Draw(t, "numGaps")
		gaps := make([]GapRange, numGaps)

		lastEnd := int64(0)
		for i := 0; i < numGaps; i++ {
			start := lastEnd + rapid.Int64Range(2, 10).Draw(t, "gapStart")
			end := start + rapid.Int64Range(0, 10).Draw(t, "gapSize")
			gaps[i] = GapRange{
				ConversationID: conversationID,
				StartSeq:       start,
				EndSeq:         end,
			}
			lastEnd = end
		}

		// Build request multiple times
		req1 := BuildGapFillRequest(conversationID, gaps, "test-req")
		req2 := BuildGapFillRequest(conversationID, gaps, "test-req")

		// Verify requests are equivalent
		if req1.ConversationID != req2.ConversationID {
			t.Fatalf("ConversationID mismatch")
		}

		if len(req1.Gaps) != len(req2.Gaps) {
			t.Fatalf("Gap count mismatch: %d vs %d", len(req1.Gaps), len(req2.Gaps))
		}

		for i := range req1.Gaps {
			if req1.Gaps[i].StartSeq != req2.Gaps[i].StartSeq ||
				req1.Gaps[i].EndSeq != req2.Gaps[i].EndSeq {
				t.Fatalf("Gap %d mismatch: [%d-%d] vs [%d-%d]",
					i,
					req1.Gaps[i].StartSeq, req1.Gaps[i].EndSeq,
					req2.Gaps[i].StartSeq, req2.Gaps[i].EndSeq)
			}
		}

		// Verify total gap size is consistent
		if req1.GetTotalGapSize() != req2.GetTotalGapSize() {
			t.Fatalf("Total gap size mismatch: %d vs %d",
				req1.GetTotalGapSize(), req2.GetTotalGapSize())
		}
	})
}

// Property 9: Response coverage validation
// CoversAllGaps should return true only when all requested sequences are accounted for
func TestProperty_ResponseCoverageValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := "test-conv"

		// Generate random gaps
		numGaps := rapid.IntRange(1, 5).Draw(t, "numGaps")
		gaps := make([]GapRange, numGaps)

		lastEnd := int64(0)
		for i := 0; i < numGaps; i++ {
			start := lastEnd + rapid.Int64Range(2, 5).Draw(t, "gapStart")
			end := start + rapid.Int64Range(1, 5).Draw(t, "gapSize")
			gaps[i] = GapRange{
				ConversationID: conversationID,
				StartSeq:       start,
				EndSeq:         end,
			}
			lastEnd = end
		}

		request := BuildGapFillRequest(conversationID, gaps, "test-req")

		// Build complete response
		response := &GapFillResponse{
			RequestID: request.RequestID,
			Messages:  make([]Message, 0),
			NotFound:  make([]int64, 0),
		}

		// Add all sequences to response
		for _, gap := range gaps {
			for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
				response.Messages = append(response.Messages, Message{
					ConversationID: conversationID,
					Sequence:       seq,
				})
			}
		}

		// Complete response should cover all gaps
		if !CoversAllGaps(request, response) {
			t.Fatalf("Complete response should cover all gaps")
		}

		// Incomplete response should not cover all gaps
		if len(response.Messages) > 0 {
			incompleteResponse := &GapFillResponse{
				RequestID: request.RequestID,
				Messages:  response.Messages[:len(response.Messages)-1], // Remove last message
				NotFound:  make([]int64, 0),
			}

			if CoversAllGaps(request, incompleteResponse) {
				t.Fatalf("Incomplete response should not cover all gaps")
			}
		}
	})
}

// Property 10: Request validation consistency
// Valid requests should always pass validation, invalid requests should always fail
func TestProperty_RequestValidationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conversationID := rapid.String().Draw(t, "conversationID")
		requestID := rapid.String().Draw(t, "requestID")

		// Generate valid gaps
		numGaps := rapid.IntRange(1, 10).Draw(t, "numGaps")
		gaps := make([]GapRange, numGaps)

		lastEnd := int64(0)
		for i := 0; i < numGaps; i++ {
			start := lastEnd + rapid.Int64Range(1, 10).Draw(t, "gapStart")
			end := start + rapid.Int64Range(0, 10).Draw(t, "gapSize")
			gaps[i] = GapRange{
				ConversationID: conversationID,
				StartSeq:       start,
				EndSeq:         end,
			}
			lastEnd = end
		}

		// Valid request should pass validation
		if conversationID != "" && requestID != "" && len(gaps) > 0 {
			validRequest := &GapFillRequest{
				ConversationID: conversationID,
				RequestID:      requestID,
				Gaps:           gaps,
			}

			if err := validRequest.Validate(); err != nil {
				t.Fatalf("Valid request failed validation: %v", err)
			}
		}

		// Invalid request (empty conversation ID) should fail
		invalidRequest1 := &GapFillRequest{
			ConversationID: "",
			RequestID:      requestID,
			Gaps:           gaps,
		}
		if err := invalidRequest1.Validate(); err == nil {
			t.Fatalf("Invalid request (empty conversation ID) should fail validation")
		}

		// Invalid request (empty request ID) should fail
		if conversationID != "" {
			invalidRequest2 := &GapFillRequest{
				ConversationID: conversationID,
				RequestID:      "",
				Gaps:           gaps,
			}
			if err := invalidRequest2.Validate(); err == nil {
				t.Fatalf("Invalid request (empty request ID) should fail validation")
			}
		}

		// Invalid request (empty gaps) should fail
		if conversationID != "" && requestID != "" {
			invalidRequest3 := &GapFillRequest{
				ConversationID: conversationID,
				RequestID:      requestID,
				Gaps:           []GapRange{},
			}
			if err := invalidRequest3.Validate(); err == nil {
				t.Fatalf("Invalid request (empty gaps) should fail validation")
			}
		}
	})
}
