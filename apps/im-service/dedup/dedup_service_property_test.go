//go:build property
// +build property

package dedup

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"pgregory.net/rapid"
)

// Property 3: Exactly-Once Display (Deduplication)
// **Validates: Requirements 8.1, 8.2, 8.3**
func TestProperty_ExactlyOnceDeduplication(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       7 * 24 * time.Hour,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		ctx := context.Background()

		// Generate random message ID
		msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id")

		// Property: First CheckAndMark should return false (not duplicate)
		isDup1, err := service.CheckAndMark(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check and mark: %v", err)
		}

		if isDup1 {
			t.Fatalf("First CheckAndMark should return false (not duplicate), got true")
		}

		// Property: Subsequent CheckAndMark calls should return true (duplicate)
		numChecks := rapid.IntRange(1, 10).Draw(t, "num_checks")
		for i := 0; i < numChecks; i++ {
			isDup, err := service.CheckAndMark(ctx, msgID)
			if err != nil {
				t.Fatalf("Failed to check and mark on iteration %d: %v", i, err)
			}

			if !isDup {
				t.Fatalf("CheckAndMark iteration %d should return true (duplicate), got false", i)
			}
		}

		// Property: CheckDuplicate should also return true
		isDup, err := service.CheckDuplicate(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check duplicate: %v", err)
		}

		if !isDup {
			t.Fatal("CheckDuplicate should return true after marking")
		}
	})
}

// Property: TTL expiration behavior
// **Validates: Requirements 8.2**
func TestProperty_TTLExpiration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		// Use short TTL for testing
		ttl := rapid.IntRange(1, 5).Draw(t, "ttl_seconds")
		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       time.Duration(ttl) * time.Second,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		ctx := context.Background()

		// Generate random message ID
		msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id")

		// Mark as processed
		err := service.MarkProcessed(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to mark processed: %v", err)
		}

		// Property: Should be duplicate before TTL expiration
		isDup, err := service.CheckDuplicate(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check duplicate: %v", err)
		}

		if !isDup {
			t.Fatal("Should be duplicate before TTL expiration")
		}

		// Fast-forward time past TTL
		mr.FastForward(time.Duration(ttl+1) * time.Second)

		// Property: Should NOT be duplicate after TTL expiration
		isDup, err = service.CheckDuplicate(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check duplicate after TTL: %v", err)
		}

		if isDup {
			t.Fatal("Should NOT be duplicate after TTL expiration")
		}
	})
}

// Property: Concurrent duplicate checks are consistent
// **Validates: Requirements 8.3**
func TestProperty_ConcurrentDuplicateChecks(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       7 * 24 * time.Hour,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		ctx := context.Background()

		// Generate random message ID
		msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id")

		// Number of concurrent goroutines
		numGoroutines := rapid.IntRange(5, 20).Draw(t, "num_goroutines")

		// Property: Exactly one goroutine should succeed in marking as new
		var wg sync.WaitGroup
		results := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				isDup, err := service.CheckAndMark(ctx, msgID)
				if err != nil {
					t.Errorf("Failed to check and mark: %v", err)
					return
				}
				results <- isDup
			}()
		}

		wg.Wait()
		close(results)

		// Count how many succeeded (not duplicate)
		newCount := 0
		dupCount := 0
		for isDup := range results {
			if isDup {
				dupCount++
			} else {
				newCount++
			}
		}

		// Property: Exactly one should succeed
		if newCount != 1 {
			t.Fatalf("Expected exactly 1 goroutine to mark as new, got %d", newCount)
		}

		if dupCount != numGoroutines-1 {
			t.Fatalf("Expected %d goroutines to see duplicate, got %d", numGoroutines-1, dupCount)
		}
	})
}

// Property: Multiple different messages are independently tracked
// **Validates: Requirements 8.1, 8.3**
func TestProperty_IndependentMessageTracking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       7 * 24 * time.Hour,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		ctx := context.Background()

		// Generate multiple unique message IDs
		numMessages := rapid.IntRange(5, 20).Draw(t, "num_messages")
		messageIDs := make([]string, numMessages)
		seenIDs := make(map[string]bool)

		for i := 0; i < numMessages; i++ {
			// Generate unique message ID
			for {
				msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id_"+string(rune(i)))
				if !seenIDs[msgID] {
					messageIDs[i] = msgID
					seenIDs[msgID] = true
					break
				}
			}
		}

		// Property: Each message should be marked as new independently
		for _, msgID := range messageIDs {
			isDup, err := service.CheckAndMark(ctx, msgID)
			if err != nil {
				t.Fatalf("Failed to check and mark %s: %v", msgID, err)
			}

			if isDup {
				t.Fatalf("Message %s should not be duplicate on first check", msgID)
			}
		}

		// Property: All messages should now be duplicates
		for _, msgID := range messageIDs {
			isDup, err := service.CheckDuplicate(ctx, msgID)
			if err != nil {
				t.Fatalf("Failed to check duplicate %s: %v", msgID, err)
			}

			if !isDup {
				t.Fatalf("Message %s should be duplicate after marking", msgID)
			}
		}

		// Property: A new message should not be duplicate
		newMsgID := "msg-newtest"
		isDup, err := service.CheckDuplicate(ctx, newMsgID)
		if err != nil {
			t.Fatalf("Failed to check new message: %v", err)
		}

		if isDup {
			t.Fatal("New message should not be duplicate")
		}
	})
}

// Property: CheckDuplicate and MarkProcessed are consistent
// **Validates: Requirements 8.1, 8.2**
func TestProperty_CheckAndMarkConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       7 * 24 * time.Hour,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		ctx := context.Background()

		// Generate random message ID
		msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id")

		// Property: Before marking, should not be duplicate
		isDup, err := service.CheckDuplicate(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check duplicate: %v", err)
		}

		if isDup {
			t.Fatal("Should not be duplicate before marking")
		}

		// Mark as processed
		err = service.MarkProcessed(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to mark processed: %v", err)
		}

		// Property: After marking, should be duplicate
		isDup, err = service.CheckDuplicate(ctx, msgID)
		if err != nil {
			t.Fatalf("Failed to check duplicate after marking: %v", err)
		}

		if !isDup {
			t.Fatal("Should be duplicate after marking")
		}

		// Property: Multiple checks should return consistent results
		for i := 0; i < 5; i++ {
			isDup, err = service.CheckDuplicate(ctx, msgID)
			if err != nil {
				t.Fatalf("Failed to check duplicate on iteration %d: %v", i, err)
			}

			if !isDup {
				t.Fatalf("Should be duplicate on iteration %d", i)
			}
		}
	})
}

// Property: Dedup key format is consistent
// **Validates: Requirements 8.1**
func TestProperty_DedupKeyFormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := Config{
			RedisAddr: mr.Addr(),
			TTL:       7 * 24 * time.Hour,
		}
		service := NewDedupService(cfg)
		defer service.Close()

		// Generate random message ID
		msgID := rapid.StringMatching(`^msg-[a-z0-9]{8}$`).Draw(t, "msg_id")

		// Property: dedupKey should always produce the same key for the same msgID
		key1 := service.dedupKey(msgID)
		key2 := service.dedupKey(msgID)

		if key1 != key2 {
			t.Fatalf("dedupKey not consistent: %s != %s", key1, key2)
		}

		// Property: Key should have expected format
		expectedPrefix := "dedup:msg:"
		if len(key1) <= len(expectedPrefix) {
			t.Fatalf("Key too short: %s", key1)
		}

		if key1[:len(expectedPrefix)] != expectedPrefix {
			t.Fatalf("Key has wrong prefix: %s", key1)
		}

		// Property: Key should contain the message ID
		if key1 != expectedPrefix+msgID {
			t.Fatalf("Key format incorrect: expected %s, got %s", expectedPrefix+msgID, key1)
		}
	})
}
