package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/shortener-service/cache"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisClusterIntegration tests Redis Cluster functionality
// This test requires a real Redis Cluster to be running
// Skip this test if Redis Cluster is not available
//
// To run this test:
// 1. Start Redis Cluster (see deploy/docker/docker-compose.infra.yml)
// 2. Run: go test -v -run TestRedisClusterIntegration ./integration_test/
//
func TestRedisClusterIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Redis Cluster client
	// Note: This assumes a Redis Cluster is running on localhost:7000-7005
	clusterAddrs := []string{
		"localhost:7000",
		"localhost:7001",
		"localhost:7002",
		"localhost:7003",
		"localhost:7004",
		"localhost:7005",
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        clusterAddrs,
		MaxRedirects: 3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer client.Close()

	ctx := context.Background()

	// Test cluster connectivity
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Redis Cluster not available: %v", err)
	}

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})
	require.NoError(t, err)

	// Create ClusterClient wrapper
	clusterClient := cache.NewClusterClient(client, obs)

	t.Run("hash tag formatting", func(t *testing.T) {
		// Test hash tag formatting
		key1 := clusterClient.FormatKeyWithHashTag("user", "123")
		key2 := clusterClient.FormatKeyWithHashTag("user", "456")

		assert.Equal(t, "{user}:123", key1)
		assert.Equal(t, "{user}:456", key2)

		// Verify keys with same hash tag go to same slot
		slot1 := client.ClusterKeySlot(ctx, key1).Val()
		slot2 := client.ClusterKeySlot(ctx, key2).Val()
		assert.Equal(t, slot1, slot2, "Keys with same hash tag should be on same slot")
	})

	t.Run("batch operations with hash tags", func(t *testing.T) {
		prefix := "test"
		testData := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		// Batch set
		err := clusterClient.BatchSetWithHashTag(ctx, prefix, testData, 10*time.Minute)
		require.NoError(t, err)

		// Batch get
		keys := []string{"key1", "key2", "key3"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)

		assert.Equal(t, 3, len(results))
		assert.Equal(t, "value1", results["key1"])
		assert.Equal(t, "value2", results["key2"])
		assert.Equal(t, "value3", results["key3"])

		// Batch delete
		err = clusterClient.BatchDeleteWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)

		// Verify deletion
		results, err = clusterClient.BatchGetWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("automatic redirect handling", func(t *testing.T) {
		// Test that MOVED/ASK redirects are handled automatically
		// This is handled by the Redis client library

		prefix := "redirect"
		testData := map[string]string{
			"test1": "value1",
			"test2": "value2",
		}

		// Set data
		err := clusterClient.BatchSetWithHashTag(ctx, prefix, testData, 10*time.Minute)
		require.NoError(t, err)

		// Get data (may trigger redirects)
		keys := []string{"test1", "test2"}
		results, err := clusterClient.BatchGetWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)

		assert.Equal(t, 2, len(results))
		assert.Equal(t, "value1", results["test1"])
		assert.Equal(t, "value2", results["test2"])

		// Cleanup
		err = clusterClient.BatchDeleteWithHashTag(ctx, prefix, keys)
		require.NoError(t, err)
	})

	t.Run("cluster info metrics", func(t *testing.T) {
		// Test cluster info collection
		err := clusterClient.GetClusterInfo(ctx)
		require.NoError(t, err)

		// Verify cluster is healthy
		info, err := client.ClusterInfo(ctx).Result()
		require.NoError(t, err)
		assert.Contains(t, info, "cluster_state:ok")
	})

	t.Run("cross-slot operations", func(t *testing.T) {
		// Test that operations on keys with different hash tags fail
		// (This is expected behavior in Redis Cluster)

		key1 := clusterClient.FormatKeyWithHashTag("prefix1", "key1")
		key2 := clusterClient.FormatKeyWithHashTag("prefix2", "key2")

		// Set individual keys (should work)
		err := client.Set(ctx, key1, "value1", 0).Err()
		require.NoError(t, err)

		err = client.Set(ctx, key2, "value2", 0).Err()
		require.NoError(t, err)

		// Try to delete both keys in one command (should fail with CROSSSLOT)
		err = client.Del(ctx, key1, key2).Err()
		if err != nil {
			// Expected: CROSSSLOT error
			assert.Contains(t, err.Error(), "CROSSSLOT")
		}

		// Cleanup
		client.Del(ctx, key1)
		client.Del(ctx, key2)
	})
}

// TestRedisClusterFailover tests cluster failover scenarios
// This test requires a real Redis Cluster with multiple nodes
func TestRedisClusterFailover(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test is manual and requires:
	// 1. A running Redis Cluster
	// 2. Manual node failure simulation
	// 3. Observation of automatic failover

	t.Skip("Manual test - requires Redis Cluster with failover simulation")

	// Test steps:
	// 1. Start Redis Cluster with 3 masters + 3 replicas
	// 2. Write data using hash tags
	// 3. Simulate master node failure (kill process)
	// 4. Verify automatic failover to replica
	// 5. Verify data is still accessible
	// 6. Verify MOVED redirects are handled
}

// TestRedisClusterScaling tests cluster scaling scenarios
// This test requires a real Redis Cluster
func TestRedisClusterScaling(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test is manual and requires:
	// 1. A running Redis Cluster
	// 2. Manual node addition/removal
	// 3. Observation of slot rebalancing

	t.Skip("Manual test - requires Redis Cluster with scaling operations")

	// Test steps:
	// 1. Start Redis Cluster with 3 masters
	// 2. Write data using hash tags
	// 3. Add new master node
	// 4. Trigger slot rebalancing
	// 5. Verify data is still accessible
	// 6. Verify ASK redirects during migration
	// 7. Remove node
	// 8. Verify data is still accessible
}

// BenchmarkClusterVsStandalone compares cluster vs standalone performance
func BenchmarkClusterVsStandalone(b *testing.B) {
	// Skip if not in integration test mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()

	// Create observability
	obs, _ := observability.New(observability.Config{
		ServiceName:    "shortener-service-test",
		ServiceVersion: "test",
		EnableMetrics:  false,
	})

	// Standalone client
	standaloneClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer standaloneClient.Close()

	// Test standalone connectivity
	if err := standaloneClient.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis standalone not available: %v", err)
	}

	standaloneCluster := cache.NewClusterClient(standaloneClient, obs)

	// Cluster client
	clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7000",
			"localhost:7001",
			"localhost:7002",
		},
	})
	defer clusterClient.Close()

	// Test cluster connectivity
	if err := clusterClient.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis Cluster not available: %v", err)
	}

	clusterWrapper := cache.NewClusterClient(clusterClient, obs)

	b.Run("standalone_batch_set", func(b *testing.B) {
		prefix := "bench"
		testData := make(map[string]string)
		for i := 0; i < 100; i++ {
			testData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			standaloneCluster.BatchSetWithHashTag(ctx, prefix, testData, 10*time.Minute)
		}
	})

	b.Run("cluster_batch_set", func(b *testing.B) {
		prefix := "bench"
		testData := make(map[string]string)
		for i := 0; i < 100; i++ {
			testData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			clusterWrapper.BatchSetWithHashTag(ctx, prefix, testData, 10*time.Minute)
		}
	})

	b.Run("standalone_batch_get", func(b *testing.B) {
		prefix := "bench"
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("key%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			standaloneCluster.BatchGetWithHashTag(ctx, prefix, keys)
		}
	})

	b.Run("cluster_batch_get", func(b *testing.B) {
		prefix := "bench"
		keys := make([]string, 100)
		for i := 0; i < 100; i++ {
			keys[i] = fmt.Sprintf("key%d", i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			clusterWrapper.BatchGetWithHashTag(ctx, prefix, keys)
		}
	})
}
