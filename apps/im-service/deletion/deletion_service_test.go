package deletion

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestDeleteUserData_Success(t *testing.T) {
	// Create mock MySQL
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create mock etcd client
	mockEtcd := &mockEtcdClient{
		data: make(map[string]string),
	}

	// Create miniredis for testing
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	// Setup test data in Redis
	ctx := context.Background()
	redisClient.Set(ctx, "dedup:msg1:user123:device1", "1", time.Hour)
	redisClient.Set(ctx, "dedup:msg2:user123:device2", "1", time.Hour)

	// Setup test data in mock etcd
	mockEtcd.data["/registry/users/user123/device1"] = "gateway1"
	mockEtcd.data["/registry/users/user123/device2"] = "gateway2"

	// Setup MySQL expectations
	mock.ExpectExec("DELETE FROM offline_messages").
		WithArgs("user123", "user123").
		WillReturnResult(sqlmock.NewResult(0, 10))

	service := NewDeletionService(db, redisClient, mockEtcd)

	// Execute deletion
	result, err := service.DeleteUserData(ctx, "user123", "DELETE_ALL_DATA")

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(10), result.OfflineMessages)
	assert.Equal(t, int64(2), result.RedisKeys)
	assert.Equal(t, int64(2), result.RegistryKeys)
	assert.WithinDuration(t, time.Now(), result.Timestamp, time.Second)

	// Verify all expectations met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUserData_InvalidConfirmation(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewDeletionService(db, nil, nil)

	ctx := context.Background()
	result, err := service.DeleteUserData(ctx, "user123", "WRONG_CONFIRMATION")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid confirmation")
}

func TestDeleteUserData_EmptyUserID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	service := NewDeletionService(db, nil, nil)

	ctx := context.Background()
	result, err := service.DeleteUserData(ctx, "", "DELETE_ALL_DATA")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user_id is required")
}

func TestDeleteOfflineMessages_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM offline_messages").
		WithArgs("user123", "user123").
		WillReturnResult(sqlmock.NewResult(0, 5))

	service := NewDeletionService(db, nil, nil)

	ctx := context.Background()
	count, err := service.deleteOfflineMessages(ctx, "user123")

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOfflineMessages_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectExec("DELETE FROM offline_messages").
		WithArgs("user123", "user123").
		WillReturnError(fmt.Errorf("database error"))

	service := NewDeletionService(db, nil, nil)

	ctx := context.Background()
	count, err := service.deleteOfflineMessages(ctx, "user123")

	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.Contains(t, err.Error(), "database error")
}

func TestDeleteRedisKeys_Success(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	ctx := context.Background()
	redisClient.Set(ctx, "dedup:msg1:user123:device1", "1", time.Hour)
	redisClient.Set(ctx, "dedup:msg2:user123:device2", "1", time.Hour)
	redisClient.Set(ctx, "dedup:msg3:user456:device1", "1", time.Hour) // Different user

	service := NewDeletionService(nil, redisClient, nil)

	count, err := service.deleteRedisKeys(ctx, "user123")

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify keys are deleted
	keys, _ := redisClient.Keys(ctx, "dedup:*:user123:*").Result()
	assert.Empty(t, keys)

	// Verify other user's keys are not deleted
	keys, _ = redisClient.Keys(ctx, "dedup:*:user456:*").Result()
	assert.Len(t, keys, 1)
}

func TestDeleteRedisKeys_NoKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	service := NewDeletionService(nil, redisClient, nil)

	ctx := context.Background()
	count, err := service.deleteRedisKeys(ctx, "user123")

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestDeleteRegistryEntries_Success(t *testing.T) {
	mockEtcd := &mockEtcdClient{
		data: make(map[string]string),
	}

	ctx := context.Background()
	mockEtcd.data["/registry/users/user123/device1"] = "gateway1"
	mockEtcd.data["/registry/users/user123/device2"] = "gateway2"
	mockEtcd.data["/registry/users/user456/device1"] = "gateway3" // Different user

	service := NewDeletionService(nil, nil, mockEtcd)

	count, err := service.deleteRegistryEntries(ctx, "user123")

	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify keys are deleted
	assert.NotContains(t, mockEtcd.data, "/registry/users/user123/device1")
	assert.NotContains(t, mockEtcd.data, "/registry/users/user123/device2")

	// Verify other user's keys are not deleted
	assert.Contains(t, mockEtcd.data, "/registry/users/user456/device1")
}

func TestDeleteRegistryEntries_NoKeys(t *testing.T) {
	mockEtcd := &mockEtcdClient{
		data: make(map[string]string),
	}

	service := NewDeletionService(nil, nil, mockEtcd)

	ctx := context.Background()
	count, err := service.deleteRegistryEntries(ctx, "user123")

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// mockEtcdClient is a mock implementation of etcd client for testing
type mockEtcdClient struct {
	data map[string]string
	mu   sync.Mutex
}

func (m *mockEtcdClient) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if it's a prefix delete by checking if WithPrefix option is present
	isPrefix := false
	for range opts {
		// If any option is provided, assume it's WithPrefix for simplicity
		isPrefix = true
		break
	}

	deleted := int64(0)
	if isPrefix {
		// Delete all keys with prefix
		keysToDelete := []string{}
		for k := range m.data {
			if len(k) >= len(key) && k[:len(key)] == key {
				keysToDelete = append(keysToDelete, k)
			}
		}
		for _, k := range keysToDelete {
			delete(m.data, k)
			deleted++
		}
	} else {
		// Delete single key
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			deleted = 1
		}
	}

	return &clientv3.DeleteResponse{
		Deleted: deleted,
	}, nil
}
