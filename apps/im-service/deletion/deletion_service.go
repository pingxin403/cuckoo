package deletion

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdClient defines the interface for etcd operations
type EtcdClient interface {
	Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error)
}

// DeletionService handles GDPR-compliant data deletion
type DeletionService struct {
	mysqlClient *sql.DB
	redisClient *redis.Client
	etcdClient  EtcdClient
}

// DeletionResult contains the results of a deletion operation
type DeletionResult struct {
	OfflineMessages int64
	RedisKeys       int64
	RegistryKeys    int64
	Timestamp       time.Time
}

// NewDeletionService creates a new deletion service
func NewDeletionService(
	mysqlClient *sql.DB,
	redisClient *redis.Client,
	etcdClient EtcdClient,
) *DeletionService {
	return &DeletionService{
		mysqlClient: mysqlClient,
		redisClient: redisClient,
		etcdClient:  etcdClient,
	}
}

// DeleteUserData deletes all data for a user across all storage systems
// Validates: Requirements 13.1, 13.2
func (s *DeletionService) DeleteUserData(ctx context.Context, userID string, confirmation string) (*DeletionResult, error) {
	// Require explicit confirmation
	if confirmation != "DELETE_ALL_DATA" {
		return nil, fmt.Errorf("invalid confirmation: expected 'DELETE_ALL_DATA', got '%s'", confirmation)
	}

	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	result := &DeletionResult{
		Timestamp: time.Now(),
	}

	// 1. Delete offline messages from MySQL
	offlineCount, err := s.deleteOfflineMessages(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete offline messages: %w", err)
	}
	result.OfflineMessages = offlineCount

	// 2. Delete Redis dedup keys
	redisCount, err := s.deleteRedisKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete redis keys: %w", err)
	}
	result.RedisKeys = redisCount

	// 3. Delete registry entries
	registryCount, err := s.deleteRegistryEntries(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete registry entries: %w", err)
	}
	result.RegistryKeys = registryCount

	return result, nil
}

// deleteOfflineMessages deletes all offline messages for a user
func (s *DeletionService) deleteOfflineMessages(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM offline_messages WHERE user_id = ? OR sender_id = ?`
	result, err := s.mysqlClient.ExecContext(ctx, query, userID, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// deleteRedisKeys deletes all dedup keys for a user
func (s *DeletionService) deleteRedisKeys(ctx context.Context, userID string) (int64, error) {
	pattern := fmt.Sprintf("dedup:*:%s:*", userID)
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := s.redisClient.Del(ctx, keys...).Result()
	return deleted, err
}

// deleteRegistryEntries deletes all registry entries for a user
func (s *DeletionService) deleteRegistryEntries(ctx context.Context, userID string) (int64, error) {
	prefix := fmt.Sprintf("/registry/users/%s/", userID)
	resp, err := s.etcdClient.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return 0, err
	}
	return resp.Deleted, nil
}
