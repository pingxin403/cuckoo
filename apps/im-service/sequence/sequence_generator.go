package sequence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
)

// ConversationType represents the type of conversation
type ConversationType string

const (
	// ConversationTypePrivate represents a private chat between two users
	ConversationTypePrivate ConversationType = "private"
	// ConversationTypeGroup represents a group chat
	ConversationTypeGroup ConversationType = "group"
)

// RedisClient defines the interface for Redis operations needed by SequenceGenerator
type RedisClient interface {
	Incr(ctx context.Context, key string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
}

// SequenceGenerator generates monotonically increasing sequence numbers for messages
type SequenceGenerator struct {
	redis RedisClient
}

// NewSequenceGenerator creates a new sequence generator with Redis backend
func NewSequenceGenerator(redisClient RedisClient) *SequenceGenerator {
	return &SequenceGenerator{
		redis: redisClient,
	}
}

// GenerateSequence generates the next sequence number for a conversation
// For private chats, conversationID should be the sorted concatenation of user IDs
// For group chats, conversationID should be the group ID
func (sg *SequenceGenerator) GenerateSequence(ctx context.Context, conversationType ConversationType, conversationID string) (int64, error) {
	if conversationID == "" {
		return 0, fmt.Errorf("conversation ID cannot be empty")
	}

	// Build Redis key: seq:{conversation_type}:{conversation_id}
	key := fmt.Sprintf("seq:%s:%s", conversationType, conversationID)

	// Use Redis INCR for atomic increment
	result, err := sg.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to generate sequence number: %w", err)
	}

	return result, nil
}

// GeneratePrivateChatSequence generates a sequence number for a private chat
// It automatically sorts the user IDs to ensure consistent conversation ID
func (sg *SequenceGenerator) GeneratePrivateChatSequence(ctx context.Context, userID1, userID2 string) (int64, error) {
	if userID1 == "" || userID2 == "" {
		return 0, fmt.Errorf("user IDs cannot be empty")
	}

	// Sort user IDs to ensure consistent conversation ID
	conversationID := sortAndJoinUserIDs(userID1, userID2)

	return sg.GenerateSequence(ctx, ConversationTypePrivate, conversationID)
}

// GenerateGroupChatSequence generates a sequence number for a group chat
func (sg *SequenceGenerator) GenerateGroupChatSequence(ctx context.Context, groupID string) (int64, error) {
	if groupID == "" {
		return 0, fmt.Errorf("group ID cannot be empty")
	}

	return sg.GenerateSequence(ctx, ConversationTypeGroup, groupID)
}

// GetCurrentSequence retrieves the current sequence number without incrementing
func (sg *SequenceGenerator) GetCurrentSequence(ctx context.Context, conversationType ConversationType, conversationID string) (int64, error) {
	if conversationID == "" {
		return 0, fmt.Errorf("conversation ID cannot be empty")
	}

	key := fmt.Sprintf("seq:%s:%s", conversationType, conversationID)

	result, err := sg.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		// Key doesn't exist, return 0
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get current sequence: %w", err)
	}

	return result, nil
}

// sortAndJoinUserIDs sorts two user IDs and joins them with a separator
// This ensures consistent conversation IDs regardless of message direction
func sortAndJoinUserIDs(userID1, userID2 string) string {
	users := []string{userID1, userID2}
	sort.Strings(users)
	return strings.Join(users, ":")
}
