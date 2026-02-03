package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/libs/observability"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestPipeline creates a test PipelineHelper with miniredis
func setupTestPipeline(t *testing.T) (*PipelineHelper, *miniredis.Miniredis, func()) {
	t.Helper()

	// Start miniredis
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create observability
	obs, err := observability.New(observability.Config{
		ServiceName:    "test-pipeline",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	require.NoError(t, err)

	// Create PipelineHelper
	pipeline := NewPipelineHelper(client, obs)

	cleanup := func() {
		client.Close()
		mr.Close()
		obs.Shutdown(context.Background())
	}

	return pipeline, mr, cleanup
}

// TestPipelineHelper_BatchSet tests batch set operations
func TestPipelineHelper_BatchSet(t *testing.T) {
	pipeline, _, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("BatchSet with small batch", func(t *testing.T) {
		entries := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		err := pipeline.BatchSet(ctx, entries, 1*time.Hour)
		assert.NoError(t, err)

		// Verify all keys were set
		for key, expectedValue := range entries {
			val, err := pipeline.client.Get(ctx, key).Result()
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, val)
		}
	})

	t.Run("BatchSet with empty entries", func(t *testing.T) {
		entries := map[string]string{}

		err := pipeline.BatchSet(ctx, entries, 1*time.Hour)
		assert.NoError(t, err)
	})

	t.Run("BatchSet with TTL", func(t *testing.T) {
		entries := map[string]string{
			"ttl_key1": "ttl_value1",
		}

		ttl := 10 * time.Second
		err := pipeline.BatchSet(ctx, entries, ttl)
		assert.NoError(t, err)

		// Verify TTL is set
		actualTTL, err := pipeline.client.TTL(ctx, "ttl_key1").Result()
		assert.NoError(t, err)
		assert.Greater(t, actualTTL, 0*time.Second)
		assert.LessOrEqual(t, actualTTL, ttl)
	})
}

// TestPipelineHelper_BatchGet tests batch get operations
func TestPipelineHelper_BatchGet(t *testing.T) {
	pipeline, _, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("BatchGet with existing keys", func(t *testing.T) {
		// Set up test data
		testData := map[string]string{
			"get_key1": "get_value1",
			"get_key2": "get_value2",
			"get_key3": "get_value3",
		}

		for key, value := range testData {
			err := pipeline.client.Set(ctx, key, value, 1*time.Hour).Err()
			require.NoError(t, err)
		}

		// Batch get
		keys := []string{"get_key1", "get_key2", "get_key3"}
		results, err := pipeline.BatchGet(ctx, keys)
		assert.NoError(t, err)
		assert.Equal(t, len(testData), len(results))

		for key, expectedValue := range testData {
			assert.Equal(t, expectedValue, results[key])
		}
	})

	t.Run("BatchGet with non-existent keys", func(t *testing.T) {
		keys := []string{"nonexistent1", "nonexistent2"}
		results, err := pipeline.BatchGet(ctx, keys)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})

	t.Run("BatchGet with mixed keys", func(t *testing.T) {
		// Set up one existing key
		err := pipeline.client.Set(ctx, "mixed_key1", "mixed_value1", 1*time.Hour).Err()
		require.NoError(t, err)

		// Get mixed keys (one exists, one doesn't)
		keys := []string{"mixed_key1", "nonexistent_key"}
		results, err := pipeline.BatchGet(ctx, keys)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "mixed_value1", results["mixed_key1"])
	})

	t.Run("BatchGet with empty keys", func(t *testing.T) {
		keys := []string{}
		results, err := pipeline.BatchGet(ctx, keys)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(results))
	})
}

// TestPipelineHelper_BatchSplitting tests batch splitting logic
func TestPipelineHelper_BatchSplitting(t *testing.T) {
	pipeline, _, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Large batch is split correctly", func(t *testing.T) {
		// Create a batch larger than maxBatchSize
		largeEntries := make(map[string]string)
		for i := 0; i < 2500; i++ {
			largeEntries[fmt.Sprintf("large_key_%d", i)] = fmt.Sprintf("large_value_%d", i)
		}

		err := pipeline.BatchSet(ctx, largeEntries, 1*time.Hour)
		assert.NoError(t, err)

		// Verify all keys were set
		count := 0
		for key := range largeEntries {
			exists, err := pipeline.client.Exists(ctx, key).Result()
			assert.NoError(t, err)
			if exists > 0 {
				count++
			}
		}
		assert.Equal(t, len(largeEntries), count)
	})

	t.Run("splitIntoBatches with small batch", func(t *testing.T) {
		entries := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		batches := pipeline.splitIntoBatches(entries)
		assert.Equal(t, 1, len(batches))
		assert.Equal(t, 2, len(batches[0]))
	})

	t.Run("splitIntoBatches with exact maxBatchSize", func(t *testing.T) {
		entries := make(map[string]string)
		for i := 0; i < 1000; i++ {
			entries[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		batches := pipeline.splitIntoBatches(entries)
		assert.Equal(t, 1, len(batches))
		assert.Equal(t, 1000, len(batches[0]))
	})

	t.Run("splitIntoBatches with multiple batches", func(t *testing.T) {
		entries := make(map[string]string)
		for i := 0; i < 2500; i++ {
			entries[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		batches := pipeline.splitIntoBatches(entries)
		assert.Equal(t, 3, len(batches))
		assert.Equal(t, 1000, len(batches[0]))
		assert.Equal(t, 1000, len(batches[1]))
		assert.Equal(t, 500, len(batches[2]))
	})
}

// TestPipelineHelper_ErrorHandling tests error handling
func TestPipelineHelper_ErrorHandling(t *testing.T) {
	pipeline, mr, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("BatchSet with Redis error", func(t *testing.T) {
		// Close miniredis to simulate error
		mr.Close()

		entries := map[string]string{
			"error_key": "error_value",
		}

		err := pipeline.BatchSet(ctx, entries, 1*time.Hour)
		assert.Error(t, err)
	})

	t.Run("BatchGet with Redis error", func(t *testing.T) {
		// Redis is already closed from previous test
		keys := []string{"error_key"}

		_, err := pipeline.BatchGet(ctx, keys)
		assert.Error(t, err)
	})
}

// TestPipelineHelper_Concurrency tests concurrent operations
func TestPipelineHelper_Concurrency(t *testing.T) {
	pipeline, _, cleanup := setupTestPipeline(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Concurrent BatchSet operations", func(t *testing.T) {
		const numGoroutines = 10
		const entriesPerGoroutine = 100

		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				entries := make(map[string]string)
				for j := 0; j < entriesPerGoroutine; j++ {
					key := fmt.Sprintf("concurrent_key_%d_%d", id, j)
					value := fmt.Sprintf("concurrent_value_%d_%d", id, j)
					entries[key] = value
				}

				err := pipeline.BatchSet(ctx, entries, 1*time.Hour)
				errChan <- err
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			err := <-errChan
			assert.NoError(t, err)
		}

		// Verify total count
		totalKeys := numGoroutines * entriesPerGoroutine
		count := 0
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < entriesPerGoroutine; j++ {
				key := fmt.Sprintf("concurrent_key_%d_%d", i, j)
				exists, err := pipeline.client.Exists(ctx, key).Result()
				assert.NoError(t, err)
				if exists > 0 {
					count++
				}
			}
		}
		assert.Equal(t, totalKeys, count)
	})
}
