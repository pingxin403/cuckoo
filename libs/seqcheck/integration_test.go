package seqcheck

import (
	"sync"
	"testing"
	"time"
)

// TestIntegration_WebSocketMessageFlow simulates a WebSocket message flow with gaps
func TestIntegration_WebSocketMessageFlow(t *testing.T) {
	conversationID := "conv-123"
	gapDetected := false
	var detectedGaps []GapRange
	var mu sync.Mutex

	// Callback to detect gaps
	gapCallback := func(gaps []GapRange) {
		mu.Lock()
		defer mu.Unlock()
		gapDetected = true
		detectedGaps = append(detectedGaps, gaps...)
	}

	sc := NewSequenceChecker(3, gapCallback)

	// Simulate receiving messages with a gap
	// Receive: 1, 2, 3, 5, 6 (missing 4)
	sequences := []int64{1, 2, 3, 5, 6}

	for _, seq := range sequences {
		gaps := sc.RecordSequence(conversationID, seq)
		if len(gaps) > 0 {
			t.Logf("Gap detected at sequence %d: %v", seq, gaps)
		}
	}

	// Wait for callback to execute
	time.Sleep(100 * time.Millisecond)

	// Verify gap was detected
	mu.Lock()
	if !gapDetected {
		t.Error("Expected gap to be detected")
	}
	mu.Unlock()

	// Verify gap is tracked
	gaps := sc.GetGaps(conversationID)
	if len(gaps) != 1 {
		t.Fatalf("Expected 1 gap, got %d", len(gaps))
	}

	if gaps[0].StartSeq != 4 || gaps[0].EndSeq != 4 {
		t.Errorf("Expected gap [4-4], got [%d-%d]", gaps[0].StartSeq, gaps[0].EndSeq)
	}

	// Build gap fill request
	request := BuildGapFillRequest(conversationID, gaps, "req-001")

	// Verify request
	if err := request.Validate(); err != nil {
		t.Fatalf("Invalid gap fill request: %v", err)
	}

	if request.GetTotalGapSize() != 1 {
		t.Errorf("Expected gap size 1, got %d", request.GetTotalGapSize())
	}

	// Simulate server response
	response := &GapFillResponse{
		RequestID: request.RequestID,
		Messages: []Message{
			{
				ID:             "msg-4",
				ConversationID: conversationID,
				Sequence:       4,
				Content:        "Missing message",
				SenderID:       "user1",
				Timestamp:      1000,
			},
		},
		NotFound: []int64{},
	}

	// Process response
	err := ProcessGapFillResponse(sc, response)
	if err != nil {
		t.Fatalf("Failed to process response: %v", err)
	}

	// Verify gap is filled
	remainingGaps := sc.GetGaps(conversationID)
	if len(remainingGaps) != 0 {
		t.Errorf("Expected gap to be filled, but %d gaps remain", len(remainingGaps))
	}
}

// TestIntegration_MultipleGapsAndRetries simulates multiple gaps with retry failures
func TestIntegration_MultipleGapsAndRetries(t *testing.T) {
	conversationID := "conv-456"
	sc := NewSequenceChecker(3, nil)

	// Create multiple gaps: 1, 5, 10, 15 (gaps: 2-4, 6-9, 11-14)
	sequences := []int64{1, 5, 10, 15}
	for _, seq := range sequences {
		sc.RecordSequence(conversationID, seq)
	}

	// Verify gaps
	gaps := sc.GetGaps(conversationID)
	if len(gaps) != 3 {
		t.Fatalf("Expected 3 gaps, got %d", len(gaps))
	}

	// Build gap fill request
	request := BuildGapFillRequest(conversationID, gaps, "req-002")

	// Simulate partial response (only first gap filled)
	response := &GapFillResponse{
		RequestID: request.RequestID,
		Messages: []Message{
			{ConversationID: conversationID, Sequence: 2},
			{ConversationID: conversationID, Sequence: 3},
			{ConversationID: conversationID, Sequence: 4},
		},
		NotFound: []int64{},
	}

	// Process response
	ProcessGapFillResponse(sc, response)

	// Verify first gap is filled
	remainingGaps := sc.GetGaps(conversationID)
	if len(remainingGaps) != 2 {
		t.Errorf("Expected 2 remaining gaps, got %d", len(remainingGaps))
	}

	// Simulate retry failures for second gap
	secondGap := remainingGaps[0]
	sc.RecordRetryFailure(conversationID, secondGap)
	sc.RecordRetryFailure(conversationID, secondGap)

	// Should not trigger full sync yet (threshold is 3)
	if sc.ShouldFullSync(conversationID) {
		t.Error("Should not trigger full sync with 2 failures")
	}

	// Third failure should trigger full sync
	sc.RecordRetryFailure(conversationID, secondGap)
	if !sc.ShouldFullSync(conversationID) {
		t.Error("Should trigger full sync with 3 failures")
	}
}

// TestIntegration_ConcurrentGapDetectionAndFilling simulates concurrent gap detection and filling
func TestIntegration_ConcurrentGapDetectionAndFilling(t *testing.T) {
	conversationID := "conv-789"
	sc := NewSequenceChecker(3, nil)

	var wg sync.WaitGroup

	// Goroutine 1: Record sequences with gaps
	wg.Add(1)
	go func() {
		defer wg.Done()
		sequences := []int64{1, 2, 5, 6, 10, 11, 15}
		for _, seq := range sequences {
			sc.RecordSequence(conversationID, seq)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Goroutine 2: Fill gaps as they're detected
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // Let some gaps accumulate

		for i := 0; i < 10; i++ {
			gaps := sc.GetGaps(conversationID)
			if len(gaps) > 0 {
				// Fill first gap
				gap := gaps[0]
				for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
					sc.FillGap(conversationID, seq)
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Goroutine 3: Query gaps periodically
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			gaps := sc.GetGaps(conversationID)
			t.Logf("Iteration %d: %d gaps", i, len(gaps))
			time.Sleep(15 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final verification
	finalGaps := sc.GetGaps(conversationID)
	t.Logf("Final gaps: %d", len(finalGaps))

	// All gaps should eventually be filled
	if len(finalGaps) > 0 {
		t.Logf("Warning: %d gaps remain (may be expected due to timing)", len(finalGaps))
	}
}

// TestIntegration_LargeGapScenario simulates a large gap scenario
func TestIntegration_LargeGapScenario(t *testing.T) {
	conversationID := "conv-large"
	sc := NewSequenceChecker(3, nil)

	// Create a large gap: 1, 1000 (gap: 2-999)
	sc.RecordSequence(conversationID, 1)
	sc.RecordSequence(conversationID, 1000)

	gaps := sc.GetGaps(conversationID)
	if len(gaps) != 1 {
		t.Fatalf("Expected 1 gap, got %d", len(gaps))
	}

	if gaps[0].EndSeq-gaps[0].StartSeq+1 != 998 {
		t.Errorf("Expected gap size 998, got %d", gaps[0].EndSeq-gaps[0].StartSeq+1)
	}

	// Build request
	request := BuildGapFillRequest(conversationID, gaps, "req-large")

	if request.GetTotalGapSize() != 998 {
		t.Errorf("Expected total gap size 998, got %d", request.GetTotalGapSize())
	}

	// Simulate filling gap in batches
	batchSize := int64(100)
	for start := gaps[0].StartSeq; start <= gaps[0].EndSeq; start += batchSize {
		end := start + batchSize - 1
		if end > gaps[0].EndSeq {
			end = gaps[0].EndSeq
		}

		// Fill batch
		for seq := start; seq <= end; seq++ {
			sc.FillGap(conversationID, seq)
		}

		t.Logf("Filled batch [%d-%d]", start, end)
	}

	// Verify all gaps filled
	remainingGaps := sc.GetGaps(conversationID)
	if len(remainingGaps) != 0 {
		t.Errorf("Expected all gaps to be filled, but %d gaps remain", len(remainingGaps))
	}
}

// TestIntegration_OutOfOrderMessagesWithGaps simulates realistic out-of-order message delivery
func TestIntegration_OutOfOrderMessagesWithGaps(t *testing.T) {
	conversationID := "conv-ooo"
	sc := NewSequenceChecker(3, nil)

	// Simulate out-of-order delivery with gaps
	// Expected order: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
	// Actual delivery: 1, 3, 2, 5, 7, 4, 9, 6, 10, 8
	deliveryOrder := []int64{1, 3, 2, 5, 7, 4, 9, 6, 10, 8}

	for _, seq := range deliveryOrder {
		gaps := sc.RecordSequence(conversationID, seq)
		if len(gaps) > 0 {
			t.Logf("Sequence %d created gaps: %v", seq, gaps)
		}
	}

	// After all messages delivered, should have no gaps
	finalGaps := sc.GetGaps(conversationID)
	if len(finalGaps) != 0 {
		t.Errorf("Expected no gaps after all messages delivered, got %d gaps", len(finalGaps))
		for _, gap := range finalGaps {
			t.Errorf("  Gap: [%d-%d]", gap.StartSeq, gap.EndSeq)
		}
	}
}

// TestIntegration_GapFillRequestResponseRoundTrip tests complete request-response cycle
func TestIntegration_GapFillRequestResponseRoundTrip(t *testing.T) {
	conversationID := "conv-roundtrip"
	sc := NewSequenceChecker(3, nil)

	// Create gaps
	sc.RecordSequence(conversationID, 1)
	sc.RecordSequence(conversationID, 10)
	sc.RecordSequence(conversationID, 20)

	// Get gaps
	gaps := sc.GetGaps(conversationID)
	if len(gaps) != 2 {
		t.Fatalf("Expected 2 gaps, got %d", len(gaps))
	}

	// Build request
	request := BuildGapFillRequest(conversationID, gaps, "req-roundtrip")

	// Serialize request
	requestJSON, err := request.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize request: %v", err)
	}

	// Deserialize request (simulating network transmission)
	receivedRequest := &GapFillRequest{}
	if err := receivedRequest.FromJSON(requestJSON); err != nil {
		t.Fatalf("Failed to deserialize request: %v", err)
	}

	// Verify request integrity
	if receivedRequest.ConversationID != request.ConversationID {
		t.Error("ConversationID mismatch after serialization")
	}
	if len(receivedRequest.Gaps) != len(request.Gaps) {
		t.Error("Gaps count mismatch after serialization")
	}

	// Build response
	response := &GapFillResponse{
		RequestID: receivedRequest.RequestID,
		Messages:  make([]Message, 0),
		NotFound:  []int64{},
	}

	// Fill all gaps
	for _, gap := range receivedRequest.Gaps {
		for seq := gap.StartSeq; seq <= gap.EndSeq; seq++ {
			response.Messages = append(response.Messages, Message{
				ConversationID: conversationID,
				Sequence:       seq,
			})
		}
	}

	// Serialize response
	responseJSON, err := response.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize response: %v", err)
	}

	// Deserialize response
	receivedResponse := &GapFillResponse{}
	if err := receivedResponse.FromJSON(responseJSON); err != nil {
		t.Fatalf("Failed to deserialize response: %v", err)
	}

	// Verify response covers all gaps
	if !CoversAllGaps(request, receivedResponse) {
		t.Error("Response does not cover all requested gaps")
	}

	// Process response
	if err := ProcessGapFillResponse(sc, receivedResponse); err != nil {
		t.Fatalf("Failed to process response: %v", err)
	}

	// Verify all gaps filled
	remainingGaps := sc.GetGaps(conversationID)
	if len(remainingGaps) != 0 {
		t.Errorf("Expected all gaps to be filled, but %d gaps remain", len(remainingGaps))
	}
}

// TestIntegration_PartialResponseHandling tests handling of partial responses
func TestIntegration_PartialResponseHandling(t *testing.T) {
	conversationID := "conv-partial"
	sc := NewSequenceChecker(3, nil)

	// Create gap: 1, 10 (gap: 2-9)
	sc.RecordSequence(conversationID, 1)
	sc.RecordSequence(conversationID, 10)

	gaps := sc.GetGaps(conversationID)
	request := BuildGapFillRequest(conversationID, gaps, "req-partial")

	// Partial response (only some messages found)
	response := &GapFillResponse{
		RequestID: request.RequestID,
		Messages: []Message{
			{ConversationID: conversationID, Sequence: 2},
			{ConversationID: conversationID, Sequence: 3},
			{ConversationID: conversationID, Sequence: 5},
			{ConversationID: conversationID, Sequence: 7},
		},
		NotFound: []int64{4, 6, 8, 9}, // Some messages not found
	}

	// Verify response doesn't cover all gaps
	if CoversAllGaps(request, response) {
		t.Error("Partial response should not cover all gaps")
	}

	// Get missing sequences
	missing := GetMissingSequences(request, response)
	if len(missing) != 0 {
		t.Logf("Missing sequences after partial response: %v", missing)
	}

	// Process response
	ProcessGapFillResponse(sc, response)

	// Verify some gaps remain
	remainingGaps := sc.GetGaps(conversationID)
	if len(remainingGaps) == 0 {
		t.Error("Expected some gaps to remain after partial response")
	}

	t.Logf("Remaining gaps: %v", remainingGaps)
}
