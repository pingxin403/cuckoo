package dedup

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DedupService provides message deduplication using Redis SET operations
// Requirements: 8.1, 8.2, 8.3
type DedupService struct {
	client *redis.Client
	ttl    time.Duration
}

// Config holds configuration for DedupService
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	TTL           time.Duration // Default: 7 days
}

// NewDedupService creates a new deduplication service
// Requirements: 8.1
func NewDedupService(cfg Config) *DedupService {
	// Set default TTL if not specified
	if cfg.TTL == 0 {
		cfg.TTL = 7 * 24 * time.Hour // 7 days
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	return &DedupService{
		client: client,
		ttl:    cfg.TTL,
	}
}

// CheckDuplicate checks if a message ID has been processed before
// Returns true if the message is a duplicate (already exists in Redis)
// Requirements: 8.1, 8.3 - O(1) lookup using Redis SET
func (d *DedupService) CheckDuplicate(ctx context.Context, msgID string) (bool, error) {
	key := d.dedupKey(msgID)

	// Check if key exists in Redis
	exists, err := d.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate for msg_id %s: %w", msgID, err)
	}

	return exists > 0, nil
}

// MarkProcessed marks a message ID as processed in Redis
// Sets a 7-day TTL on the deduplication record
// Requirements: 8.2 - 7-day TTL on deduplication records
func (d *DedupService) MarkProcessed(ctx context.Context, msgID string) error {
	key := d.dedupKey(msgID)

	// Set key with TTL
	err := d.client.Set(ctx, key, "1", d.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to mark msg_id %s as processed: %w", msgID, err)
	}

	return nil
}

// CheckAndMark atomically checks for duplicate and marks as processed if not duplicate
// Returns true if the message is a duplicate, false if it's new and has been marked
// Requirements: 8.1, 8.2, 8.3
func (d *DedupService) CheckAndMark(ctx context.Context, msgID string) (isDuplicate bool, err error) {
	key := d.dedupKey(msgID)

	// Use SETNX (SET if Not eXists) for atomic check-and-set
	// Returns true if key was set (not duplicate), false if key already exists (duplicate)
	wasSet, err := d.client.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check and mark msg_id %s: %w", msgID, err)
	}

	// If wasSet is true, the key didn't exist (not a duplicate)
	// If wasSet is false, the key already existed (is a duplicate)
	return !wasSet, nil
}

// Close closes the Redis connection
func (d *DedupService) Close() error {
	return d.client.Close()
}

// Ping checks if Redis connection is alive
func (d *DedupService) Ping(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

// dedupKey generates the Redis key for a message ID
func (d *DedupService) dedupKey(msgID string) string {
	return fmt.Sprintf("dedup:msg:%s", msgID)
}
