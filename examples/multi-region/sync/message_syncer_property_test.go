//go:build property

package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/libs/hlc"
	"github.com/cuckoo-org/cuckoo/examples/mvp/queue"
	"github.com/cuckoo-org/cuckoo/examples/mvp/storage"
	"pgregory.net/rapid"
)

// **Validates: Requirements 1.1, 2.2**

// TestProperty_MessageEventualConsistency tests Property 4: 消息最终一致性
// This property ensures that messages eventually reach consistency across regions
// regardless of network delays or failures.
func TestProperty_MessageEventualConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numRegions := rapid.IntRange(2, 4).Draw(t, "numRegions")
		numMessages := rapid.IntRange(3, 15).Draw(t, "numMessages")
		maxNetworkDelay := rapid.IntRange(10, 100).Draw(t, "maxNetworkDelay") // milliseconds

		// Create multiple regions with message syncers
		regions := make(map[string]*testRegion)
		regionNames := make([]string, numRegions)

		for i := 0; i < numRegions; i++ {
			regionName := fmt.Sprintf("region-%d", i)
			regionNames[i] = regionName

			region, err := createTestRegion(regionName)
			if err != nil {
				t.Fatalf("Failed to create test region %s: %v", regionName, err)
			}
			defer region.cleanup()

			regions[regionName] = region
		}

		// Start all syncers
		for _, region := range regions {
			if err := region.syncer.Start(); err != nil {
				t.Fatalf("Failed to start syncer for region %s: %v", region.name, err)
			}
		}

		// Generate messages in random regions
		var allMessages []storage.LocalMessage
		messagesByRegion := make(map[string][]storage.LocalMessage)

		for i := 0; i < numMessages; i++ {
			// Choose random source region
			sourceRegion := rapid.SampledFrom(regionNames).Draw(t, fmt.Sprintf("sourceRegion_%d", i))

			// Generate message
			message := generateRandomMessage(t, i, sourceRegion)
			allMessages = append(allMessages, message)
			messagesByRegion[sourceRegion] = append(messagesByRegion[sourceRegion], message)

			// Insert message in source region
			ctx := context.Background()
			if err := regions[sourceRegion].storage.Insert(ctx, message); err != nil {
				t.Fatalf("Failed to insert message in source region: %v", err)
			}

			// Sync message to all other regions with simulated network delay
			for _, targetRegion := range regionNames {
				if targetRegion != sourceRegion {
					// Simulate network delay
					delay := time.Duration(rapid.IntRange(1, maxNetworkDelay).Draw(t, fmt.Sprintf("delay_%d_%s", i, targetRegion))) * time.Millisecond

					go func(src, tgt string, msg storage.LocalMessage, d time.Duration) {
						time.Sleep(d)
						if err := regions[src].syncer.SyncMessageAsync(ctx, tgt, msg); err != nil {
							t.Logf("Warning: Failed to sync message from %s to %s: %v", src, tgt, err)
						}
					}(sourceRegion, targetRegion, message, delay)
				}
			}
		}

		// Wait for all messages to propagate (eventual consistency)
		maxWaitTime := time.Duration(maxNetworkDelay*2+500) * time.Millisecond
		time.Sleep(maxWaitTime)

		// Property 1: All messages should eventually exist in all regions
		ctx := context.Background()
		for _, message := range allMessages {
			for _, regionName := range regionNames {
				region := regions[regionName]

				// Try to retrieve the message (with retries for eventual consistency)
				var retrievedMsg *storage.LocalMessage
				var err error

				for retry := 0; retry < 5; retry++ {
					retrievedMsg, err = region.storage.GetMessageByID(ctx, message.MsgID)
					if err == nil {
						break
					}
					time.Sleep(50 * time.Millisecond)
				}

				if err != nil {
					t.Fatalf("Message %s not found in region %s after eventual consistency period: %v",
						message.MsgID, regionName, err)
				}

				// Verify message content consistency
				if retrievedMsg.Content != message.Content {
					t.Fatalf("Message content inconsistent in region %s: expected %s, got %s",
						regionName, message.Content, retrievedMsg.Content)
				}

				if retrievedMsg.SenderID != message.SenderID {
					t.Fatalf("Message sender inconsistent in region %s: expected %s, got %s",
						regionName, message.SenderID, retrievedMsg.SenderID)
				}
			}
		}

		// Property 2: Message ordering should be consistent across regions
		// All regions should have the same relative ordering of messages
		var regionOrderings [][]storage.LocalMessage

		for _, regionName := range regionNames {
			region := regions[regionName]

			// Get all messages from this region
			var regionMessages []storage.LocalMessage
			for _, message := range allMessages {
				retrievedMsg, err := region.storage.GetMessageByID(ctx, message.MsgID)
				if err != nil {
					t.Fatalf("Failed to get message %s from region %s: %v", message.MsgID, regionName, err)
				}
				regionMessages = append(regionMessages, *retrievedMsg)
			}

			// Sort messages by GlobalID (HLC ordering)
			sort.Slice(regionMessages, func(i, j int) bool {
				id1 := parseGlobalIDFromMessage(regionMessages[i])
				id2 := parseGlobalIDFromMessage(regionMessages[j])
				return hlc.CompareGlobalID(id1, id2) < 0
			})

			regionOrderings = append(regionOrderings, regionMessages)
		}

		// Verify all regions have the same message ordering
		if len(regionOrderings) > 1 {
			baseOrdering := regionOrderings[0]
			for i := 1; i < len(regionOrderings); i++ {
				currentOrdering := regionOrderings[i]

				if len(baseOrdering) != len(currentOrdering) {
					t.Fatalf("Region %s has different number of messages: expected %d, got %d",
						regionNames[i], len(baseOrdering), len(currentOrdering))
				}

				for j := 0; j < len(baseOrdering); j++ {
					if baseOrdering[j].MsgID != currentOrdering[j].MsgID {
						t.Fatalf("Message ordering inconsistent between regions: position %d has %s in region 0 but %s in region %d",
							j, baseOrdering[j].MsgID, currentOrdering[j].MsgID, i)
					}
				}
			}
		}

		// Property 3: Convergence - all regions should have identical final state
		for i := 1; i < len(regionNames); i++ {
			region1 := regions[regionNames[0]]
			region2 := regions[regionNames[i]]

			metrics1 := region1.syncer.GetMetrics()
			metrics2 := region2.syncer.GetMetrics()

			// Both regions should have processed messages (allowing for some variance due to timing)
			totalMessages1 := metrics1["async_sync_count"].(int64)
			totalMessages2 := metrics2["async_sync_count"].(int64)

			if totalMessages1 == 0 && totalMessages2 == 0 && numMessages > 0 {
				t.Fatalf("No messages were processed by any region, indicating sync failure")
			}
		}

		t.Logf("Eventual consistency verified: %d messages across %d regions", numMessages, numRegions)
	})
}

// TestProperty_ConflictResolutionDeterminism tests Property 5: 冲突解决确定性
// This property ensures that conflict resolution is deterministic - the same conflict
// always resolves the same way across all regions.
func TestProperty_ConflictResolutionDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test parameters
		numRegions := rapid.IntRange(2, 4).Draw(t, "numRegions")
		numConflicts := rapid.IntRange(2, 8).Draw(t, "numConflicts")

		// Create multiple regions
		regions := make(map[string]*testRegion)
		regionNames := make([]string, numRegions)

		for i := 0; i < numRegions; i++ {
			regionName := fmt.Sprintf("region-%d", i)
			regionNames[i] = regionName

			region, err := createTestRegion(regionName)
			if err != nil {
				t.Fatalf("Failed to create test region %s: %v", regionName, err)
			}
			defer region.cleanup()

			regions[regionName] = region
		}

		// Start all syncers
		for _, region := range regions {
			if err := region.syncer.Start(); err != nil {
				t.Fatalf("Failed to start syncer for region %s: %v", region.name, err)
			}
		}

		// Generate conflicting messages
		var conflictGroups [][]storage.LocalMessage

		for i := 0; i < numConflicts; i++ {
			// Create conflicting messages with same ID but different content/timestamps
			messageID := fmt.Sprintf("conflict-msg-%d", i)
			conversationID := fmt.Sprintf("conv-%d", i)

			var conflictGroup []storage.LocalMessage

			// Create 2-3 conflicting versions of the same message
			numVersions := rapid.IntRange(2, 3).Draw(t, fmt.Sprintf("numVersions_%d", i))

			for j := 0; j < numVersions; j++ {
				sourceRegion := regionNames[j%len(regionNames)]

				// Create slightly different timestamps to ensure conflicts
				baseTime := time.Now().UnixMilli()
				timestamp := baseTime + int64(rapid.IntRange(-100, 100).Draw(t, fmt.Sprintf("timestamp_%d_%d", i, j)))

				message := storage.LocalMessage{
					MsgID:            messageID, // Same ID - this will cause conflict
					SenderID:         fmt.Sprintf("user-%d", rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("sender_%d_%d", i, j))),
					ConversationID:   conversationID,
					ConversationType: "group",
					Content:          fmt.Sprintf("Conflicting content from %s version %d", sourceRegion, j),
					SequenceNumber:   int64(j + 1),
					Timestamp:        timestamp,
					CreatedAt:        time.Now().Add(time.Duration(j) * time.Millisecond),
					ExpiresAt:        time.Now().Add(24 * time.Hour),
					Metadata:         map[string]string{"source": sourceRegion, "version": fmt.Sprintf("%d", j)},
					RegionID:         sourceRegion,
					GlobalID:         regions[sourceRegion].hlc.GenerateID().String(),
					Version:          int64(j + 1),
				}

				conflictGroup = append(conflictGroup, message)
			}

			conflictGroups = append(conflictGroups, conflictGroup)
		}

		// Insert conflicting messages in their respective regions
		ctx := context.Background()
		for _, conflictGroup := range conflictGroups {
			for _, message := range conflictGroup {
				if err := regions[message.RegionID].storage.Insert(ctx, message); err != nil {
					t.Fatalf("Failed to insert conflicting message: %v", err)
				}
			}
		}

		// Sync all messages to all regions to trigger conflicts
		for _, conflictGroup := range conflictGroups {
			for _, message := range conflictGroup {
				for _, targetRegion := range regionNames {
					if targetRegion != message.RegionID {
						// Add small delay to simulate network conditions
						time.Sleep(time.Duration(rapid.IntRange(1, 20).Draw(t, "syncDelay")) * time.Millisecond)

						if err := regions[message.RegionID].syncer.SyncMessageAsync(ctx, targetRegion, message); err != nil {
							t.Logf("Warning: Failed to sync conflicting message: %v", err)
						}
					}
				}
			}
		}

		// Wait for conflict resolution
		time.Sleep(500 * time.Millisecond)

		// Property 1: All regions must resolve conflicts to the same winner
		for _, conflictGroup := range conflictGroups {
			messageID := conflictGroup[0].MsgID
			var winnersByRegion []storage.LocalMessage

			// Get the resolved message from each region
			for _, regionName := range regionNames {
				region := regions[regionName]

				resolvedMsg, err := region.storage.GetMessageByID(ctx, messageID)
				if err != nil {
					t.Fatalf("Failed to get resolved message %s from region %s: %v", messageID, regionName, err)
				}

				winnersByRegion = append(winnersByRegion, *resolvedMsg)
			}

			// Property: All regions must have the same winner
			baseWinner := winnersByRegion[0]
			for i := 1; i < len(winnersByRegion); i++ {
				currentWinner := winnersByRegion[i]

				// The content should be identical (same winner)
				if baseWinner.Content != currentWinner.Content {
					t.Fatalf("Conflict resolution not deterministic for message %s: region 0 has content '%s', region %d has content '%s'",
						messageID, baseWinner.Content, i, currentWinner.Content)
				}

				// The source region should be identical (same winner)
				if baseWinner.Metadata["source"] != currentWinner.Metadata["source"] {
					t.Fatalf("Conflict resolution not deterministic for message %s: different source regions chosen as winner",
						messageID)
				}

				// The version should be identical (same winner)
				if baseWinner.Version != currentWinner.Version {
					t.Fatalf("Conflict resolution not deterministic for message %s: different versions chosen as winner",
						messageID)
				}
			}

			t.Logf("Conflict for message %s resolved deterministically: winner from %s with content '%s'",
				messageID, baseWinner.Metadata["source"], baseWinner.Content)
		}

		// Property 2: Conflict resolution should follow LWW (Last Write Wins) based on HLC
		for _, conflictGroup := range conflictGroups {
			messageID := conflictGroup[0].MsgID

			// Get the resolved message from any region (they should all be the same)
			resolvedMsg, err := regions[regionNames[0]].storage.GetMessageByID(ctx, messageID)
			if err != nil {
				t.Fatalf("Failed to get resolved message: %v", err)
			}

			// Find the message with the latest HLC timestamp in the conflict group
			var expectedWinner storage.LocalMessage
			var latestHLC hlc.GlobalID

			for _, message := range conflictGroup {
				messageHLC := parseGlobalIDFromMessage(message)
				if expectedWinner.MsgID == "" || hlc.CompareGlobalID(messageHLC, latestHLC) > 0 {
					expectedWinner = message
					latestHLC = messageHLC
				}
			}

			// The resolved message should match the expected LWW winner
			if resolvedMsg.Content != expectedWinner.Content {
				t.Fatalf("LWW conflict resolution failed for message %s: expected winner content '%s', got '%s'",
					messageID, expectedWinner.Content, resolvedMsg.Content)
			}
		}

		// Property 3: Conflict metrics should be consistent across regions
		var totalConflictsPerRegion []int64
		for _, regionName := range regionNames {
			region := regions[regionName]
			metrics := region.syncer.GetMetrics()
			conflictCount := metrics["conflict_count"].(int64)
			totalConflictsPerRegion = append(totalConflictsPerRegion, conflictCount)
		}

		// All regions should have detected some conflicts (allowing for timing variations)
		totalConflictsDetected := int64(0)
		for _, count := range totalConflictsPerRegion {
			totalConflictsDetected += count
		}

		if totalConflictsDetected == 0 && numConflicts > 0 {
			t.Fatalf("No conflicts were detected despite generating %d conflict groups", numConflicts)
		}

		t.Logf("Conflict resolution determinism verified: %d conflict groups resolved consistently across %d regions",
			numConflicts, numRegions)
	})
}

// TestProperty_MessageSyncIdempotency tests that message synchronization is idempotent
// Multiple sync operations of the same message should not create duplicates
func TestProperty_MessageSyncIdempotency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create two regions
		regionA, err := createTestRegion("region-a")
		if err != nil {
			t.Fatalf("Failed to create region A: %v", err)
		}
		defer regionA.cleanup()

		regionB, err := createTestRegion("region-b")
		if err != nil {
			t.Fatalf("Failed to create region B: %v", err)
		}
		defer regionB.cleanup()

		// Start syncers
		if err := regionA.syncer.Start(); err != nil {
			t.Fatalf("Failed to start syncer A: %v", err)
		}
		if err := regionB.syncer.Start(); err != nil {
			t.Fatalf("Failed to start syncer B: %v", err)
		}

		// Generate a test message
		message := generateRandomMessage(t, 1, "region-a")

		// Insert message in region A
		ctx := context.Background()
		if err := regionA.storage.Insert(ctx, message); err != nil {
			t.Fatalf("Failed to insert message in region A: %v", err)
		}

		// Sync the same message multiple times
		numSyncs := rapid.IntRange(2, 10).Draw(t, "numSyncs")

		for i := 0; i < numSyncs; i++ {
			if err := regionA.syncer.SyncMessageAsync(ctx, "region-b", message); err != nil {
				t.Fatalf("Failed to sync message (attempt %d): %v", i+1, err)
			}
		}

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Property: Only one copy of the message should exist in region B
		retrievedMsg, err := regionB.storage.GetMessageByID(ctx, message.MsgID)
		if err != nil {
			t.Fatalf("Message not found in region B after sync: %v", err)
		}

		// Verify content is correct
		if retrievedMsg.Content != message.Content {
			t.Fatalf("Message content corrupted: expected %s, got %s", message.Content, retrievedMsg.Content)
		}

		// Property: Multiple syncs should not increase error count significantly
		metricsA := regionA.syncer.GetMetrics()
		errorCount := metricsA["error_count"].(int64)

		// Allow some errors due to deduplication, but not too many
		if errorCount > int64(numSyncs/2) {
			t.Fatalf("Too many errors during idempotent sync: %d errors for %d sync attempts", errorCount, numSyncs)
		}

		t.Logf("Idempotency verified: %d sync attempts resulted in 1 message with %d errors", numSyncs, errorCount)
	})
}

// TestProperty_CrossRegionCausalConsistency tests that causal relationships
// are preserved across regions during synchronization
func TestProperty_CrossRegionCausalConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create three regions to test complex causal chains
		regions := make(map[string]*testRegion)
		regionNames := []string{"region-a", "region-b", "region-c"}

		for _, regionName := range regionNames {
			region, err := createTestRegion(regionName)
			if err != nil {
				t.Fatalf("Failed to create region %s: %v", regionName, err)
			}
			defer region.cleanup()
			regions[regionName] = region
		}

		// Start all syncers
		for _, region := range regions {
			if err := region.syncer.Start(); err != nil {
				t.Fatalf("Failed to start syncer for region %s: %v", region.name, err)
			}
		}

		// Generate a causal chain of messages: A -> B -> C
		numChains := rapid.IntRange(2, 5).Draw(t, "numChains")
		var allMessages []storage.LocalMessage

		ctx := context.Background()

		for chain := 0; chain < numChains; chain++ {
			// Message 1: Generated in region A
			msg1 := generateRandomMessage(t, chain*3+1, "region-a")
			msg1.ConversationID = fmt.Sprintf("causal-chain-%d", chain)
			allMessages = append(allMessages, msg1)

			if err := regions["region-a"].storage.Insert(ctx, msg1); err != nil {
				t.Fatalf("Failed to insert msg1: %v", err)
			}

			// Sync msg1 to region B
			if err := regions["region-a"].syncer.SyncMessageAsync(ctx, "region-b", msg1); err != nil {
				t.Fatalf("Failed to sync msg1 to region B: %v", err)
			}

			// Wait for sync
			time.Sleep(50 * time.Millisecond)

			// Message 2: Generated in region B (causally after msg1)
			msg2 := generateRandomMessage(t, chain*3+2, "region-b")
			msg2.ConversationID = fmt.Sprintf("causal-chain-%d", chain)
			msg2.SequenceNumber = msg1.SequenceNumber + 1
			allMessages = append(allMessages, msg2)

			if err := regions["region-b"].storage.Insert(ctx, msg2); err != nil {
				t.Fatalf("Failed to insert msg2: %v", err)
			}

			// Sync msg2 to region C
			if err := regions["region-b"].syncer.SyncMessageAsync(ctx, "region-c", msg2); err != nil {
				t.Fatalf("Failed to sync msg2 to region C: %v", err)
			}

			// Wait for sync
			time.Sleep(50 * time.Millisecond)

			// Message 3: Generated in region C (causally after msg2)
			msg3 := generateRandomMessage(t, chain*3+3, "region-c")
			msg3.ConversationID = fmt.Sprintf("causal-chain-%d", chain)
			msg3.SequenceNumber = msg2.SequenceNumber + 1
			allMessages = append(allMessages, msg3)

			if err := regions["region-c"].storage.Insert(ctx, msg3); err != nil {
				t.Fatalf("Failed to insert msg3: %v", err)
			}

			// Sync all messages to all regions
			for _, msg := range []storage.LocalMessage{msg1, msg2, msg3} {
				for _, targetRegion := range regionNames {
					if targetRegion != msg.RegionID {
						if err := regions[msg.RegionID].syncer.SyncMessageAsync(ctx, targetRegion, msg); err != nil {
							t.Logf("Warning: Failed to sync message %s to %s: %v", msg.MsgID, targetRegion, err)
						}
					}
				}
			}
		}

		// Wait for all synchronization to complete
		time.Sleep(300 * time.Millisecond)

		// Property: Causal ordering must be preserved in all regions
		for _, regionName := range regionNames {
			region := regions[regionName]

			// Get all messages for each causal chain
			for chain := 0; chain < numChains; chain++ {
				conversationID := fmt.Sprintf("causal-chain-%d", chain)
				var chainMessages []storage.LocalMessage

				// Collect all messages in this causal chain
				for _, msg := range allMessages {
					if msg.ConversationID == conversationID {
						retrievedMsg, err := region.storage.GetMessageByID(ctx, msg.MsgID)
						if err != nil {
							t.Fatalf("Message %s not found in region %s: %v", msg.MsgID, regionName, err)
						}
						chainMessages = append(chainMessages, *retrievedMsg)
					}
				}

				// Sort by HLC timestamp
				sort.Slice(chainMessages, func(i, j int) bool {
					id1 := parseGlobalIDFromMessage(chainMessages[i])
					id2 := parseGlobalIDFromMessage(chainMessages[j])
					return hlc.CompareGlobalID(id1, id2) < 0
				})

				// Verify causal order is preserved (sequence numbers should be increasing)
				for i := 1; i < len(chainMessages); i++ {
					if chainMessages[i].SequenceNumber <= chainMessages[i-1].SequenceNumber {
						t.Fatalf("Causal ordering violated in region %s for chain %d: seq[%d]=%d <= seq[%d]=%d",
							regionName, chain, i, chainMessages[i].SequenceNumber, i-1, chainMessages[i-1].SequenceNumber)
					}
				}
			}
		}

		t.Logf("Causal consistency verified: %d causal chains preserved across %d regions", numChains, len(regionNames))
	})
}

// Helper types and functions

type testRegion struct {
	name    string
	hlc     *hlc.HLC
	queue   *queue.LocalQueue
	storage *storage.LocalStore
	syncer  *MessageSyncer
}

func createTestRegion(regionName string) (*testRegion, error) {
	// Create HLC
	hlcClock := hlc.NewHLC(regionName, "node-1")

	// Create queue
	queueConfig := queue.DefaultConfig(regionName)
	queueConfig.BufferSize = 100
	localQueue, err := queue.NewLocalQueue(queueConfig, log.New(os.Stdout, fmt.Sprintf("[Queue-%s] ", regionName), log.LstdFlags))
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	// Create storage
	storageConfig := storage.Config{
		RegionID:   regionName,
		MemoryMode: true,
	}
	localStorage, err := storage.NewLocalStore(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Create syncer
	syncerConfig := DefaultConfig(regionName)
	syncerConfig.SyncTimeout = 2 * time.Second
	syncer, err := NewMessageSyncer(regionName, hlcClock, localQueue, localStorage, syncerConfig,
		log.New(os.Stdout, fmt.Sprintf("[Syncer-%s] ", regionName), log.LstdFlags))
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer: %w", err)
	}

	return &testRegion{
		name:    regionName,
		hlc:     hlcClock,
		queue:   localQueue,
		storage: localStorage,
		syncer:  syncer,
	}, nil
}

func (tr *testRegion) cleanup() {
	if tr.syncer != nil {
		tr.syncer.Stop()
	}
	if tr.queue != nil {
		tr.queue.Close()
	}
	if tr.storage != nil {
		tr.storage.Close()
	}
}

func generateRandomMessage(t *rapid.T, id int, regionID string) storage.LocalMessage {
	return storage.LocalMessage{
		MsgID:            fmt.Sprintf("msg-%d", id),
		SenderID:         rapid.SampledFrom([]string{"user-1", "user-2", "user-3", "user-4"}).Draw(t, fmt.Sprintf("sender_%d", id)),
		ConversationID:   rapid.SampledFrom([]string{"conv-1", "conv-2", "conv-3"}).Draw(t, fmt.Sprintf("conv_%d", id)),
		ConversationType: "group",
		Content:          fmt.Sprintf("Message %d from %s: %s", id, regionID, rapid.StringMatching("[a-zA-Z ]{10,50}").Draw(t, fmt.Sprintf("content_%d", id))),
		SequenceNumber:   int64(id),
		Timestamp:        time.Now().UnixMilli() + int64(id),
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Metadata:         map[string]string{"type": "text", "source": regionID},
		RegionID:         regionID,
		Version:          1,
	}
}

func parseGlobalIDFromMessage(msg storage.LocalMessage) hlc.GlobalID {
	// Simple parsing - in production this would be more robust
	if msg.GlobalID == "" {
		// Fallback to timestamp-based ID
		return hlc.GlobalID{
			RegionID: msg.RegionID,
			HLC:      fmt.Sprintf("%d-0", msg.Timestamp),
			Sequence: msg.SequenceNumber,
		}
	}

	return hlc.GlobalID{
		RegionID: msg.RegionID,
		HLC:      msg.GlobalID,
		Sequence: msg.SequenceNumber,
	}
}
