package sequence

import (
	"context"
	"sync"
	"testing"

	"pgregory.net/rapid"
)

// Property 1: Message Sequence Monotonicity
// **Validates: Requirements 16.1**
func TestProperty_SequenceMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		// Generate random conversation type and ID
		conversationType := rapid.SampledFrom([]ConversationType{
			ConversationTypePrivate,
			ConversationTypeGroup,
		}).Draw(t, "conversation_type")

		conversationID := rapid.StringMatching(`^[a-z0-9]{8}$`).Draw(t, "conversation_id")

		// Generate random number of sequences (1-100)
		numSequences := rapid.IntRange(1, 100).Draw(t, "num_sequences")

		// Property: All generated sequences must be strictly increasing
		var prevSeq int64
		for i := 0; i < numSequences; i++ {
			seq, err := sg.GenerateSequence(ctx, conversationType, conversationID)
			if err != nil {
				t.Fatalf("Failed to generate sequence: %v", err)
			}

			if seq <= prevSeq {
				t.Fatalf("Sequence not monotonic: prev=%d, current=%d", prevSeq, seq)
			}

			prevSeq = seq
		}

		// Verify final sequence equals number of sequences generated
		if prevSeq != int64(numSequences) {
			t.Fatalf("Expected final sequence %d, got %d", numSequences, prevSeq)
		}
	})
}

// Property 2: Private chat user ID sorting consistency
// **Validates: Requirements 16.2**
func TestProperty_PrivateChatUserIDSorting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		// Generate two random user IDs
		userID1 := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id_1")
		userID2 := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "user_id_2")

		// Skip if user IDs are the same
		if userID1 == userID2 {
			t.Skip("User IDs are the same")
		}

		// Generate sequence with userID1 -> userID2
		seq1, err := sg.GeneratePrivateChatSequence(ctx, userID1, userID2)
		if err != nil {
			t.Fatalf("Failed to generate sequence: %v", err)
		}

		// Generate sequence with userID2 -> userID1 (reversed)
		seq2, err := sg.GeneratePrivateChatSequence(ctx, userID2, userID1)
		if err != nil {
			t.Fatalf("Failed to generate sequence: %v", err)
		}

		// Property: Both directions should use the same conversation and increment the same sequence
		if seq2 != seq1+1 {
			t.Fatalf("Expected seq2=%d to be seq1+1=%d", seq2, seq1+1)
		}
	})
}

// Property 3: Multiple conversations are independent
// **Validates: Requirements 16.2, 16.3**
func TestProperty_ConversationIndependence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		// Generate multiple random conversations
		numConversations := rapid.IntRange(2, 10).Draw(t, "num_conversations")
		conversations := make([]string, numConversations)
		for i := 0; i < numConversations; i++ {
			conversations[i] = rapid.StringMatching(`^conv[0-9]{4}$`).Draw(t, "conversation_"+string(rune(i)))
		}

		// Generate sequences for each conversation
		sequences := make(map[string][]int64)
		for _, convID := range conversations {
			numSeqs := rapid.IntRange(1, 10).Draw(t, "num_seqs_"+convID)
			for i := 0; i < numSeqs; i++ {
				seq, err := sg.GenerateSequence(ctx, ConversationTypeGroup, convID)
				if err != nil {
					t.Fatalf("Failed to generate sequence: %v", err)
				}
				sequences[convID] = append(sequences[convID], seq)
			}
		}

		// Property: Each conversation should have independent sequences starting from 1
		for convID, seqs := range sequences {
			if len(seqs) == 0 {
				continue
			}

			// First sequence should be 1
			if seqs[0] != 1 {
				t.Fatalf("Conversation %s: first sequence should be 1, got %d", convID, seqs[0])
			}

			// All sequences should be monotonic
			for i := 1; i < len(seqs); i++ {
				if seqs[i] != seqs[i-1]+1 {
					t.Fatalf("Conversation %s: sequence not monotonic at index %d: %d -> %d",
						convID, i, seqs[i-1], seqs[i])
				}
			}
		}
	})
}

// Property 4: Concurrent sequence generation maintains monotonicity
// **Validates: Requirements 16.1, 16.6**
func TestProperty_ConcurrentSequenceGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		conversationID := rapid.StringMatching(`^conv[0-9]{4}$`).Draw(t, "conversation_id")
		numGoroutines := rapid.IntRange(2, 10).Draw(t, "num_goroutines")
		seqsPerGoroutine := rapid.IntRange(5, 20).Draw(t, "seqs_per_goroutine")

		// Generate sequences concurrently
		var wg sync.WaitGroup
		allSequences := make([][]int64, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sequences := make([]int64, seqsPerGoroutine)
				for j := 0; j < seqsPerGoroutine; j++ {
					seq, err := sg.GenerateSequence(ctx, ConversationTypeGroup, conversationID)
					if err != nil {
						t.Errorf("Failed to generate sequence: %v", err)
						return
					}
					sequences[j] = seq
				}
				allSequences[idx] = sequences
			}(i)
		}
		wg.Wait()

		// Collect all sequences
		var allSeqs []int64
		for _, seqs := range allSequences {
			allSeqs = append(allSeqs, seqs...)
		}

		// Property: All sequences should be unique (no duplicates)
		seenSeqs := make(map[int64]bool)
		for _, seq := range allSeqs {
			if seenSeqs[seq] {
				t.Fatalf("Duplicate sequence detected: %d", seq)
			}
			seenSeqs[seq] = true
		}

		// Property: Total number of sequences should match expected
		expectedTotal := numGoroutines * seqsPerGoroutine
		if len(allSeqs) != expectedTotal {
			t.Fatalf("Expected %d sequences, got %d", expectedTotal, len(allSeqs))
		}
	})
}

// Property 5: GetCurrentSequence returns correct value without incrementing
// **Validates: Requirements 16.1**
func TestProperty_GetCurrentSequenceNoIncrement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		conversationID := rapid.StringMatching(`^conv[0-9]{4}$`).Draw(t, "conversation_id")
		numSequences := rapid.IntRange(1, 50).Draw(t, "num_sequences")

		// Generate some sequences
		var lastSeq int64
		for i := 0; i < numSequences; i++ {
			seq, err := sg.GenerateSequence(ctx, ConversationTypeGroup, conversationID)
			if err != nil {
				t.Fatalf("Failed to generate sequence: %v", err)
			}
			lastSeq = seq
		}

		// Property: GetCurrentSequence should return the last generated value
		current, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, conversationID)
		if err != nil {
			t.Fatalf("Failed to get current sequence: %v", err)
		}

		if current != lastSeq {
			t.Fatalf("Expected current sequence %d, got %d", lastSeq, current)
		}

		// Property: Multiple calls to GetCurrentSequence should return the same value
		for i := 0; i < 5; i++ {
			current2, err := sg.GetCurrentSequence(ctx, ConversationTypeGroup, conversationID)
			if err != nil {
				t.Fatalf("Failed to get current sequence: %v", err)
			}

			if current2 != current {
				t.Fatalf("GetCurrentSequence not idempotent: first=%d, call %d=%d", current, i+1, current2)
			}
		}
	})
}

// Property 6: Empty inputs are rejected
// **Validates: Requirements 16.2, 16.3**
func TestProperty_EmptyInputValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRedis := NewMockRedisClient()
		sg := NewSequenceGenerator(mockRedis)
		ctx := context.Background()

		// Test empty conversation ID
		_, err := sg.GenerateSequence(ctx, ConversationTypeGroup, "")
		if err == nil {
			t.Fatal("Expected error for empty conversation ID")
		}

		// Test empty user IDs for private chat
		validUserID := rapid.StringMatching(`^user[0-9]{3}$`).Draw(t, "valid_user_id")

		_, err = sg.GeneratePrivateChatSequence(ctx, "", validUserID)
		if err == nil {
			t.Fatal("Expected error for empty userID1")
		}

		_, err = sg.GeneratePrivateChatSequence(ctx, validUserID, "")
		if err == nil {
			t.Fatal("Expected error for empty userID2")
		}

		// Test empty group ID
		_, err = sg.GenerateGroupChatSequence(ctx, "")
		if err == nil {
			t.Fatal("Expected error for empty group ID")
		}
	})
}
