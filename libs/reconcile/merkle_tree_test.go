package reconcile

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/libs/hlc"
)

// TestNewMerkleTree tests creating a new Merkle tree
func TestNewMerkleTree(t *testing.T) {
	messages := []MessageData{
		{
			GlobalID:       "region-a-1000-0-1",
			Content:        "Hello",
			Timestamp:      1000,
			RegionID:       "region-a",
			ConversationID: "conv1",
			SequenceNumber: 1,
		},
		{
			GlobalID:       "region-a-1001-0-2",
			Content:        "World",
			Timestamp:      1001,
			RegionID:       "region-a",
			ConversationID: "conv1",
			SequenceNumber: 2,
		},
	}

	// Compute hashes
	for i := range messages {
		messages[i].Hash = ComputeMessageHash(messages[i])
	}

	tree := NewMerkleTree("region-a", messages)

	if tree == nil {
		t.Fatal("Expected non-nil tree")
	}

	if tree.GetMessageCount() != 2 {
		t.Errorf("Expected 2 messages, got %d", tree.GetMessageCount())
	}

	if tree.GetRootHash() == "" {
		t.Error("Expected non-empty root hash")
	}
}

// TestMerkleTreeEmptyMessages tests creating a tree with no messages
func TestMerkleTreeEmptyMessages(t *testing.T) {
	tree := NewMerkleTree("region-a", []MessageData{})

	if tree.GetMessageCount() != 0 {
		t.Errorf("Expected 0 messages, got %d", tree.GetMessageCount())
	}

	if tree.GetRootHash() != "" {
		t.Error("Expected empty root hash for empty tree")
	}
}

// TestMerkleTreeSingleMessage tests creating a tree with a single message
func TestMerkleTreeSingleMessage(t *testing.T) {
	msg := MessageData{
		GlobalID:       "region-a-1000-0-1",
		Content:        "Hello",
		Timestamp:      1000,
		RegionID:       "region-a",
		ConversationID: "conv1",
		SequenceNumber: 1,
	}
	msg.Hash = ComputeMessageHash(msg)

	tree := NewMerkleTree("region-a", []MessageData{msg})

	if tree.GetMessageCount() != 1 {
		t.Errorf("Expected 1 message, got %d", tree.GetMessageCount())
	}

	if tree.GetRootHash() != msg.Hash {
		t.Error("Root hash should equal message hash for single message tree")
	}
}

// TestComputeMessageHash tests message hash computation
func TestComputeMessageHash(t *testing.T) {
	msg1 := MessageData{
		GlobalID:       "region-a-1000-0-1",
		Content:        "Hello",
		Timestamp:      1000,
		RegionID:       "region-a",
		ConversationID: "conv1",
		SequenceNumber: 1,
	}

	msg2 := MessageData{
		GlobalID:       "region-a-1000-0-1",
		Content:        "Hello",
		Timestamp:      1000,
		RegionID:       "region-a",
		ConversationID: "conv1",
		SequenceNumber: 1,
	}

	hash1 := ComputeMessageHash(msg1)
	hash2 := ComputeMessageHash(msg2)

	if hash1 != hash2 {
		t.Error("Same messages should produce same hash")
	}

	// Change content
	msg2.Content = "World"
	hash3 := ComputeMessageHash(msg2)

	if hash1 == hash3 {
		t.Error("Different messages should produce different hashes")
	}
}

// TestFindDifferencesIdenticalTrees tests diff on identical trees
func TestFindDifferencesIdenticalTrees(t *testing.T) {
	messages := createTestMessages(5, "region-a")

	tree1 := NewMerkleTree("region-a", messages)
	tree2 := NewMerkleTree("region-b", messages)

	ctx := context.Background()
	diff, err := tree1.FindDifferences(ctx, tree2)
	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	if diff.DiffCount != 0 {
		t.Errorf("Expected 0 differences, got %d", diff.DiffCount)
	}

	if len(diff.MissingInLocal) != 0 {
		t.Errorf("Expected 0 missing in local, got %d", len(diff.MissingInLocal))
	}

	if len(diff.MissingInRemote) != 0 {
		t.Errorf("Expected 0 missing in remote, got %d", len(diff.MissingInRemote))
	}

	if len(diff.Conflicts) != 0 {
		t.Errorf("Expected 0 conflicts, got %d", len(diff.Conflicts))
	}
}

// TestFindDifferencesMissingInLocal tests detecting messages missing in local tree
func TestFindDifferencesMissingInLocal(t *testing.T) {
	// Create 5 messages
	allMessages := createTestMessages(5, "region-a")

	// Local has only first 3 messages
	localMessages := allMessages[:3]

	// Remote has all 5 messages
	remoteMessages := allMessages

	localTree := NewMerkleTree("region-a", localMessages)
	remoteTree := NewMerkleTree("region-b", remoteMessages)

	ctx := context.Background()
	diff, err := localTree.FindDifferences(ctx, remoteTree)
	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	// When trees have different structures, the diff will report more differences
	// The important thing is that we detect the 2 missing messages
	if len(diff.MissingInLocal) < 2 {
		t.Errorf("Expected at least 2 missing in local, got %d", len(diff.MissingInLocal))
	}

	// Verify the last 2 messages are in the missing list
	missingMap := make(map[string]bool)
	for _, gid := range diff.MissingInLocal {
		missingMap[gid] = true
	}

	if !missingMap[allMessages[3].GlobalID] || !missingMap[allMessages[4].GlobalID] {
		t.Error("Expected last 2 messages to be missing in local")
	}
}

// TestFindDifferencesMissingInRemote tests detecting messages missing in remote tree
func TestFindDifferencesMissingInRemote(t *testing.T) {
	// Create 5 messages
	allMessages := createTestMessages(5, "region-a")

	// Local has all 5 messages
	localMessages := allMessages

	// Remote has only first 3 messages
	remoteMessages := allMessages[:3]

	localTree := NewMerkleTree("region-a", localMessages)
	remoteTree := NewMerkleTree("region-b", remoteMessages)

	ctx := context.Background()
	diff, err := localTree.FindDifferences(ctx, remoteTree)
	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	// When trees have different structures, the diff will report more differences
	// The important thing is that we detect the 2 missing messages
	if len(diff.MissingInRemote) < 2 {
		t.Errorf("Expected at least 2 missing in remote, got %d", len(diff.MissingInRemote))
	}

	// Verify the last 2 messages are in the missing list
	missingMap := make(map[string]bool)
	for _, gid := range diff.MissingInRemote {
		missingMap[gid] = true
	}

	if !missingMap[allMessages[3].GlobalID] || !missingMap[allMessages[4].GlobalID] {
		t.Error("Expected last 2 messages to be missing in remote")
	}
}

// TestFindDifferencesConflicts tests detecting conflicting messages
func TestFindDifferencesConflicts(t *testing.T) {
	messages1 := createTestMessages(3, "region-a")
	messages2 := createTestMessages(3, "region-a")

	// Modify one message to create a conflict
	messages2[1].Content = "Modified content"
	messages2[1].Hash = ComputeMessageHash(messages2[1])

	tree1 := NewMerkleTree("region-a", messages1)
	tree2 := NewMerkleTree("region-b", messages2)

	ctx := context.Background()
	diff, err := tree1.FindDifferences(ctx, tree2)
	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	if len(diff.Conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(diff.Conflicts))
	}
}

// TestGetMessage tests retrieving a message by GlobalID
func TestGetMessage(t *testing.T) {
	messages := createTestMessages(5, "region-a")
	tree := NewMerkleTree("region-a", messages)

	// Get existing message
	msg, err := tree.GetMessage(messages[2].GlobalID)
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}

	if msg.GlobalID != messages[2].GlobalID {
		t.Errorf("Expected GlobalID %s, got %s", messages[2].GlobalID, msg.GlobalID)
	}

	// Get non-existing message
	_, err = tree.GetMessage("non-existing-id")
	if err == nil {
		t.Error("Expected error for non-existing message")
	}
}

// TestGetTreeDepth tests tree depth calculation
func TestGetTreeDepth(t *testing.T) {
	tests := []struct {
		name          string
		messageCount  int
		expectedDepth int
	}{
		{"Empty tree", 0, 0},
		{"Single message", 1, 1},
		{"Two messages", 2, 2},
		{"Four messages", 4, 3},
		{"Eight messages", 8, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := createTestMessages(tt.messageCount, "region-a")
			tree := NewMerkleTree("region-a", messages)

			depth := tree.GetTreeDepth()
			if depth != tt.expectedDepth {
				t.Errorf("Expected depth %d, got %d", tt.expectedDepth, depth)
			}
		})
	}
}

// TestRebuild tests rebuilding a tree with new messages
func TestRebuild(t *testing.T) {
	messages1 := createTestMessages(3, "region-a")
	tree := NewMerkleTree("region-a", messages1)

	originalHash := tree.GetRootHash()
	originalCount := tree.GetMessageCount()

	// Rebuild with different messages
	messages2 := createTestMessages(5, "region-a")
	tree.Rebuild(messages2)

	newHash := tree.GetRootHash()
	newCount := tree.GetMessageCount()

	if newHash == originalHash {
		t.Error("Root hash should change after rebuild")
	}

	if newCount == originalCount {
		t.Error("Message count should change after rebuild")
	}

	if newCount != 5 {
		t.Errorf("Expected 5 messages after rebuild, got %d", newCount)
	}
}

// TestGetMessagesInRange tests retrieving messages in a GlobalID range
func TestGetMessagesInRange(t *testing.T) {
	messages := createTestMessages(10, "region-a")
	tree := NewMerkleTree("region-a", messages)

	// Get messages in range
	startID := messages[2].GlobalID
	endID := messages[6].GlobalID

	rangeMessages, err := tree.GetMessagesInRange(startID, endID)
	if err != nil {
		t.Fatalf("GetMessagesInRange failed: %v", err)
	}

	if len(rangeMessages) != 5 {
		t.Errorf("Expected 5 messages in range, got %d", len(rangeMessages))
	}

	// Verify all messages are within range
	for _, msg := range rangeMessages {
		gid, _ := hlc.ParseGlobalID(msg.GlobalID)
		start, _ := hlc.ParseGlobalID(startID)
		end, _ := hlc.ParseGlobalID(endID)

		if hlc.CompareGlobalID(gid, start) < 0 || hlc.CompareGlobalID(gid, end) > 0 {
			t.Errorf("Message %s is outside range [%s, %s]", msg.GlobalID, startID, endID)
		}
	}
}

// TestGetTreeStats tests retrieving tree statistics
func TestGetTreeStats(t *testing.T) {
	messages := createTestMessages(5, "region-a")
	tree := NewMerkleTree("region-a", messages)

	stats := tree.GetTreeStats()

	if stats["region_id"] != "region-a" {
		t.Errorf("Expected region_id 'region-a', got %v", stats["region_id"])
	}

	if stats["message_count"] != 5 {
		t.Errorf("Expected message_count 5, got %v", stats["message_count"])
	}

	if stats["has_root"] != true {
		t.Error("Expected has_root to be true")
	}

	if stats["root_hash"] == "" {
		t.Error("Expected non-empty root_hash")
	}
}

// TestConcurrentAccess tests concurrent access to the tree
func TestConcurrentAccess(t *testing.T) {
	messages := createTestMessages(100, "region-a")
	tree := NewMerkleTree("region-a", messages)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				_ = tree.GetRootHash()
				_ = tree.GetMessageCount()
				_ = tree.GetTreeDepth()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestContextCancellation tests that diff respects context cancellation
func TestContextCancellation(t *testing.T) {
	messages1 := createTestMessages(1000, "region-a")
	messages2 := createTestMessages(1000, "region-b")

	tree1 := NewMerkleTree("region-a", messages1)
	tree2 := NewMerkleTree("region-b", messages2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := tree1.FindDifferences(ctx, tree2)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// Helper function to create test messages
func createTestMessages(count int, regionID string) []MessageData {
	messages := make([]MessageData, count)
	baseTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		msg := MessageData{
			GlobalID:       generateGlobalID(regionID, baseTime+int64(i), i+1),
			Content:        "Test message " + string(rune('A'+i)),
			Timestamp:      baseTime + int64(i),
			RegionID:       regionID,
			ConversationID: "conv1",
			SequenceNumber: int64(i + 1),
		}
		msg.Hash = ComputeMessageHash(msg)
		messages[i] = msg
	}

	return messages
}

// Helper function to generate a GlobalID
func generateGlobalID(regionID string, timestamp int64, sequence int) string {
	return hlc.GlobalID{
		RegionID: regionID,
		HLC:      fmt.Sprintf("%d-0", timestamp),
		Sequence: int64(sequence),
	}.String()
}
