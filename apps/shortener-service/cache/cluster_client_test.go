package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestObservabilityForCluster creates a test observability instance
func createTestObservabilityForCluster() observability.Observability {
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	return obs
}

func TestNewClusterClient(t *testing.T) {
	// Create mock Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	obs := createTestObservabilityForCluster()

	// Create cluster client
	clusterClient := NewClusterClient(client, obs)

	assert.NotNil(t, clusterClient)
	assert.NotNil(t, clusterClient.client)
	assert.NotNil(t, clusterClient.obs)
}

func TestFormatKeyWithHashTag(t *testing.T) {
	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(nil, obs)

	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "user prefix",
			prefix:   "user",
			key:      "123",
			expected: "{user}:123",
		},
		{
			name:     "url prefix",
			prefix:   "url",
			key:      "abc",
			expected: "{url}:abc",
		},
		{
			name:     "empty key",
			prefix:   "test",
			key:      "",
			expected: "{test}:",
		},
		{
			name:     "complex key",
			prefix:   "session",
			key:      "user:123:token",
			expected: "{session}:user:123:token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clusterClient.FormatKeyWithHashTag(tt.prefix, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBatchGetWithHashTag(t *testing.T) {
	// Create mock Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(client, obs)

	ctx := context.Background()

	// Set up test data
	prefix := "user"
	testData := map[string]string{
		"123": "Alice",
		"456": "Bob",
		"789": "Charlie",
	}

	// Populate Redis with hash-tagged keys
	for key, value := range testData {
		formattedKey := clusterClient.FormatKeyWithHashTag(prefix, key)
		err := client.Set(ctx, formattedKey, value, 0).Err()
		require.NoError(t, err)
	}

	t.Run("get all keys", func(t *testing.T) {
		keys := []string{"123", "456", "789"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)

		require.NoError(t, err)
		assert.Equal(t, 3, len(results))
		assert.Equal(t, "Alice", results["123"])
		assert.Equal(t, "Bob", results["456"])
		assert.Equal(t, "Charlie", results["789"])
	})

	t.Run("get subset of keys", func(t *testing.T) {
		keys := []string{"123", "456"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)

		require.NoError(t, err)
		assert.Equal(t, 2, len(results))
		assert.Equal(t, "Alice", results["123"])
		assert.Equal(t, "Bob", results["456"])
	})

	t.Run("get non-existent keys", func(t *testing.T) {
		keys := []string{"999", "888"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)

		require.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("get mix of existent and non-existent keys", func(t *testing.T) {
		keys := []string{"123", "999", "456"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)

		require.NoError(t, err)
		assert.Equal(t, 2, len(results))
		assert.Equal(t, "Alice", results["123"])
		assert.Equal(t, "Bob", results["456"])
	})

	t.Run("empty keys", func(t *testing.T) {
		keys := []string{}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)

		require.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})
}

func TestBatchSetWithHashTag(t *testing.T) {
	// Create mock Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(client, obs)

	ctx := context.Background()

	t.Run("set multiple keys", func(t *testing.T) {
		prefix := "user"
		kvPairs := map[string]string{
			"123": "Alice",
			"456": "Bob",
			"789": "Charlie",
		}
		ttl := 10 * time.Minute

		err := clusterClient.BatchSetWithHashTag(ctx, prefix, kvPairs, ttl)
		require.NoError(t, err)

		// Verify keys were set
		for key, expectedValue := range kvPairs {
			formattedKey := clusterClient.FormatKeyWithHashTag(prefix, key)
			value, err := client.Get(ctx, formattedKey).Result()
			require.NoError(t, err)
			assert.Equal(t, expectedValue, value)

			// Verify TTL was set
			ttlResult, err := client.TTL(ctx, formattedKey).Result()
			require.NoError(t, err)
			assert.Greater(t, ttlResult, time.Duration(0))
			assert.LessOrEqual(t, ttlResult, ttl)
		}
	})

	t.Run("set with zero TTL", func(t *testing.T) {
		prefix := "session"
		kvPairs := map[string]string{
			"abc": "value1",
		}

		err := clusterClient.BatchSetWithHashTag(ctx, prefix, kvPairs, 0)
		require.NoError(t, err)

		// Verify key was set
		formattedKey := clusterClient.FormatKeyWithHashTag(prefix, "abc")
		value, err := client.Get(ctx, formattedKey).Result()
		require.NoError(t, err)
		assert.Equal(t, "value1", value)

		// Verify no TTL
		ttlResult, err := client.TTL(ctx, formattedKey).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-1), ttlResult) // -1 means no expiration
	})

	t.Run("empty kvPairs", func(t *testing.T) {
		prefix := "test"
		kvPairs := map[string]string{}

		err := clusterClient.BatchSetWithHashTag(ctx, prefix, kvPairs, 0)
		require.NoError(t, err)
	})
}

func TestBatchDeleteWithHashTag(t *testing.T) {
	// Create mock Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(client, obs)

	ctx := context.Background()

	t.Run("delete multiple keys", func(t *testing.T) {
		prefix := "user"
		keys := []string{"123", "456", "789"}

		// Set up test data
		for _, key := range keys {
			formattedKey := clusterClient.FormatKeyWithHashTag(prefix, key)
			err := client.Set(ctx, formattedKey, "value", 0).Err()
			require.NoError(t, err)
		}

		// Delete keys
		err := clusterClient.BatchDeleteWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)

		// Verify keys were deleted
		for _, key := range keys {
			formattedKey := clusterClient.FormatKeyWithHashTag(prefix, key)
			_, err := client.Get(ctx, formattedKey).Result()
			assert.Equal(t, redis.Nil, err)
		}
	})

	t.Run("delete non-existent keys", func(t *testing.T) {
		prefix := "test"
		keys := []string{"999", "888"}

		// Delete non-existent keys (should not error)
		err := clusterClient.BatchDeleteWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)
	})

	t.Run("empty keys", func(t *testing.T) {
		prefix := "test"
		keys := []string{}

		err := clusterClient.BatchDeleteWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)
	})
}

func TestClusterClient_Client(t *testing.T) {
	// Create mock Redis
	mr := miniredis.RunT(t)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(client, obs)

	// Verify Client() returns the underlying client
	underlyingClient := clusterClient.Client()
	assert.Equal(t, client, underlyingClient)
}

func TestHashTagConsistency(t *testing.T) {
	// Verify that keys with the same hash tag produce consistent formatting
	obs := createTestObservabilityForCluster()
	clusterClient := NewClusterClient(nil, obs)

	prefix := "user"
	keys := []string{"123", "456", "789"}

	// Format keys multiple times
	for i := 0; i < 3; i++ {
		for _, key := range keys {
			result := clusterClient.FormatKeyWithHashTag(prefix, key)
			expected := "{user}:" + key
			assert.Equal(t, expected, result, "Hash tag formatting should be consistent")
		}
	}
}
