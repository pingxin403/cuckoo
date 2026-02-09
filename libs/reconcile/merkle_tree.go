package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pingxin403/cuckoo/libs/hlc"
)

// MessageData represents the data structure for a message used in reconciliation
type MessageData struct {
	GlobalID       string `json:"global_id"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
	RegionID       string `json:"region_id"`
	ConversationID string `json:"conversation_id"`
	SequenceNumber int64  `json:"sequence_number"`
	Hash           string `json:"hash"` // SHA256 hash of the message data
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash   string        // Hash of this node
	Left   *MerkleNode   // Left child
	Right  *MerkleNode   // Right child
	Data   *MessageData  // Leaf node data (nil for internal nodes)
	IsLeaf bool          // Whether this is a leaf node
	Range  *MessageRange // Range of messages covered by this node
}

// MessageRange represents a range of messages in the tree
type MessageRange struct {
	StartGlobalID string // First message global ID in this range
	EndGlobalID   string // Last message global ID in this range
	Count         int    // Number of messages in this range
}

// MerkleTree represents a Merkle tree for message reconciliation
type MerkleTree struct {
	mu       sync.RWMutex
	root     *MerkleNode
	messages []MessageData // Sorted by GlobalID
	regionID string
}

// DiffResult represents the result of comparing two Merkle trees
type DiffResult struct {
	MissingInLocal  []string // GlobalIDs missing in local tree
	MissingInRemote []string // GlobalIDs missing in remote tree
	Conflicts       []string // GlobalIDs with different hashes
	TotalChecked    int      // Total number of messages checked
	DiffCount       int      // Total number of differences found
}

// ReconcileStats represents statistics about a reconciliation operation
type ReconcileStats struct {
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	MessagesChecked int
	Differences     int
	Repaired        int
	Failed          int
}

// NewMerkleTree creates a new Merkle tree for the given messages
func NewMerkleTree(regionID string, messages []MessageData) *MerkleTree {
	mt := &MerkleTree{
		regionID: regionID,
		messages: make([]MessageData, len(messages)),
	}
	copy(mt.messages, messages)

	// Sort messages by GlobalID for consistent tree structure
	sort.Slice(mt.messages, func(i, j int) bool {
		gid1, _ := hlc.ParseGlobalID(mt.messages[i].GlobalID)
		gid2, _ := hlc.ParseGlobalID(mt.messages[j].GlobalID)
		return hlc.CompareGlobalID(gid1, gid2) < 0
	})

	// Build the tree
	mt.root = mt.buildTree(mt.messages)

	return mt
}

// buildTree recursively builds a Merkle tree from a sorted list of messages
func (mt *MerkleTree) buildTree(messages []MessageData) *MerkleNode {
	if len(messages) == 0 {
		return nil
	}

	// Base case: single message (leaf node)
	if len(messages) == 1 {
		msg := messages[0]
		return &MerkleNode{
			Hash:   msg.Hash,
			Data:   &msg,
			IsLeaf: true,
			Range: &MessageRange{
				StartGlobalID: msg.GlobalID,
				EndGlobalID:   msg.GlobalID,
				Count:         1,
			},
		}
	}

	// Recursive case: split messages and create internal node
	mid := len(messages) / 2
	left := mt.buildTree(messages[:mid])
	right := mt.buildTree(messages[mid:])

	// Compute hash of internal node from children
	hash := computeInternalHash(left.Hash, right.Hash)

	return &MerkleNode{
		Hash:   hash,
		Left:   left,
		Right:  right,
		IsLeaf: false,
		Range: &MessageRange{
			StartGlobalID: messages[0].GlobalID,
			EndGlobalID:   messages[len(messages)-1].GlobalID,
			Count:         len(messages),
		},
	}
}

// GetRootHash returns the hash of the root node
func (mt *MerkleTree) GetRootHash() string {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if mt.root == nil {
		return ""
	}
	return mt.root.Hash
}

// GetMessageCount returns the number of messages in the tree
func (mt *MerkleTree) GetMessageCount() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.messages)
}

// FindDifferences compares this tree with a remote tree and returns differences
// This implements an efficient diff algorithm that only traverses subtrees with different hashes
func (mt *MerkleTree) FindDifferences(ctx context.Context, remoteTree *MerkleTree) (*DiffResult, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	result := &DiffResult{
		MissingInLocal:  make([]string, 0),
		MissingInRemote: make([]string, 0),
		Conflicts:       make([]string, 0),
	}

	// Quick check: if root hashes match, trees are identical
	if mt.root != nil && remoteTree.root != nil && mt.root.Hash == remoteTree.root.Hash {
		result.TotalChecked = len(mt.messages)
		return result, nil
	}

	// Perform recursive diff
	err := mt.diffNodes(ctx, mt.root, remoteTree.root, result)
	if err != nil {
		return nil, err
	}

	result.DiffCount = len(result.MissingInLocal) + len(result.MissingInRemote) + len(result.Conflicts)
	result.TotalChecked = len(mt.messages)

	return result, nil
}

// diffNodes recursively compares two nodes and accumulates differences
func (mt *MerkleTree) diffNodes(ctx context.Context, local, remote *MerkleNode, result *DiffResult) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Handle nil nodes
	if local == nil && remote == nil {
		return nil
	}

	if local == nil {
		// All messages in remote subtree are missing in local
		mt.collectMissingMessages(remote, &result.MissingInLocal)
		return nil
	}

	if remote == nil {
		// All messages in local subtree are missing in remote
		mt.collectMissingMessages(local, &result.MissingInRemote)
		return nil
	}

	// If hashes match, subtrees are identical
	if local.Hash == remote.Hash {
		return nil
	}

	// If both are leaf nodes, we have a conflict or different messages
	if local.IsLeaf && remote.IsLeaf {
		if local.Data.GlobalID == remote.Data.GlobalID {
			// Same message ID but different content - conflict
			result.Conflicts = append(result.Conflicts, local.Data.GlobalID)
		} else {
			// Different messages at same position
			result.MissingInRemote = append(result.MissingInRemote, local.Data.GlobalID)
			result.MissingInLocal = append(result.MissingInLocal, remote.Data.GlobalID)
		}
		return nil
	}

	// If one is leaf and other is internal, collect all messages from internal node
	if local.IsLeaf && !remote.IsLeaf {
		result.MissingInRemote = append(result.MissingInRemote, local.Data.GlobalID)
		mt.collectMissingMessages(remote, &result.MissingInLocal)
		return nil
	}

	if !local.IsLeaf && remote.IsLeaf {
		mt.collectMissingMessages(local, &result.MissingInRemote)
		result.MissingInLocal = append(result.MissingInLocal, remote.Data.GlobalID)
		return nil
	}

	// Both are internal nodes - recursively compare children
	if err := mt.diffNodes(ctx, local.Left, remote.Left, result); err != nil {
		return err
	}
	if err := mt.diffNodes(ctx, local.Right, remote.Right, result); err != nil {
		return err
	}

	return nil
}

// collectMissingMessages collects all message GlobalIDs from a subtree
func (mt *MerkleTree) collectMissingMessages(node *MerkleNode, result *[]string) {
	if node == nil {
		return
	}

	if node.IsLeaf {
		*result = append(*result, node.Data.GlobalID)
		return
	}

	mt.collectMissingMessages(node.Left, result)
	mt.collectMissingMessages(node.Right, result)
}

// GetMessage returns a message by its GlobalID
func (mt *MerkleTree) GetMessage(globalID string) (*MessageData, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Binary search since messages are sorted
	idx := sort.Search(len(mt.messages), func(i int) bool {
		gid1, _ := hlc.ParseGlobalID(mt.messages[i].GlobalID)
		gid2, _ := hlc.ParseGlobalID(globalID)
		return hlc.CompareGlobalID(gid1, gid2) >= 0
	})

	if idx < len(mt.messages) && mt.messages[idx].GlobalID == globalID {
		return &mt.messages[idx], nil
	}

	return nil, fmt.Errorf("message not found: %s", globalID)
}

// ComputeMessageHash computes the SHA256 hash of a message's data
func ComputeMessageHash(msg MessageData) string {
	// Create a deterministic string representation of the message
	data := fmt.Sprintf("%s|%s|%d|%s|%s|%d",
		msg.GlobalID,
		msg.Content,
		msg.Timestamp,
		msg.RegionID,
		msg.ConversationID,
		msg.SequenceNumber,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// computeInternalHash computes the hash of an internal node from its children
func computeInternalHash(leftHash, rightHash string) string {
	combined := leftHash + rightHash
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// GetTreeDepth returns the depth of the Merkle tree
func (mt *MerkleTree) GetTreeDepth() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.getNodeDepth(mt.root)
}

// getNodeDepth recursively computes the depth of a node
func (mt *MerkleTree) getNodeDepth(node *MerkleNode) int {
	if node == nil {
		return 0
	}
	if node.IsLeaf {
		return 1
	}

	leftDepth := mt.getNodeDepth(node.Left)
	rightDepth := mt.getNodeDepth(node.Right)

	if leftDepth > rightDepth {
		return leftDepth + 1
	}
	return rightDepth + 1
}

// GetTreeStats returns statistics about the Merkle tree
func (mt *MerkleTree) GetTreeStats() map[string]interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return map[string]interface{}{
		"region_id":     mt.regionID,
		"message_count": len(mt.messages),
		"tree_depth":    mt.getNodeDepth(mt.root),
		"root_hash":     mt.GetRootHash(),
		"has_root":      mt.root != nil,
	}
}

// Rebuild rebuilds the Merkle tree with updated messages
func (mt *MerkleTree) Rebuild(messages []MessageData) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.messages = make([]MessageData, len(messages))
	copy(mt.messages, messages)

	// Sort messages by GlobalID
	sort.Slice(mt.messages, func(i, j int) bool {
		gid1, _ := hlc.ParseGlobalID(mt.messages[i].GlobalID)
		gid2, _ := hlc.ParseGlobalID(mt.messages[j].GlobalID)
		return hlc.CompareGlobalID(gid1, gid2) < 0
	})

	// Rebuild the tree
	mt.root = mt.buildTree(mt.messages)
}

// GetMessagesInRange returns all messages within a specific GlobalID range
func (mt *MerkleTree) GetMessagesInRange(startGlobalID, endGlobalID string) ([]MessageData, error) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	result := make([]MessageData, 0)

	for _, msg := range mt.messages {
		gid, err := hlc.ParseGlobalID(msg.GlobalID)
		if err != nil {
			continue
		}

		start, err := hlc.ParseGlobalID(startGlobalID)
		if err != nil {
			return nil, fmt.Errorf("invalid start GlobalID: %w", err)
		}

		end, err := hlc.ParseGlobalID(endGlobalID)
		if err != nil {
			return nil, fmt.Errorf("invalid end GlobalID: %w", err)
		}

		if hlc.CompareGlobalID(gid, start) >= 0 && hlc.CompareGlobalID(gid, end) <= 0 {
			result = append(result, msg)
		}
	}

	return result, nil
}
