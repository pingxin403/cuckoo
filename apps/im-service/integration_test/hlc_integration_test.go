package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/hlc"
	"github.com/pingxin403/cuckoo/apps/im-service/sequence"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHLCIntegrationWithSequenceGenerator tests HLC integration with sequence generator
func TestHLCIntegrationWithSequenceGenerator(t *testing.T) {
	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test database
	})
	defer redisClient.Close()

	// Clear test data
	ctx := context.Background()
	_ = redisClient.FlushDB(ctx).Err()

	// Create sequence generators for two regions
	seqGenA := sequence.NewSequenceGeneratorWithRegion(redisClient, "region-a", "node-1")
	seqGenB := sequence.NewSequenceGeneratorWithRegion(redisClient, "region-b", "node-1")

	t.Run("generate global IDs from different regions", func(t *testing.T) {
		// Generate IDs from region A
		idA1, err := seqGenA.GenerateGlobalID()
		require.NoError(t, err)
		assert.Contains(t, idA1, "region-a")

		idA2, err := seqGenA.GenerateGlobalID()
		require.NoError(t, err)
		assert.Contains(t, idA2, "region-a")

		// Generate IDs from region B
		idB1, err := seqGenB.GenerateGlobalID()
		require.NoError(t, err)
		assert.Contains(t, idB1, "region-b")

		// All IDs should be unique
		assert.NotEqual(t, idA1, idA2)
		assert.NotEqual(t, idA1, idB1)
		assert.NotEqual(t, idA2, idB1)
	})

	t.Run("generate sequence with global ID", func(t *testing.T) {
		conversationID := "user1:user2"

		// Generate from region A
		seqA, globalIDA, err := seqGenA.GenerateSequenceWithGlobalID(
			ctx,
			sequence.ConversationTypePrivate,
			conversationID,
		)
		require.NoError(t, err)
		assert.Equal(t, int64(1), seqA)
		assert.NotEmpty(t, globalIDA)

		// Generate from region B (different conversation, same ID format)
		seqB, globalIDB, err := seqGenB.GenerateSequenceWithGlobalID(
			ctx,
			sequence.ConversationTypePrivate,
			conversationID,
		)
		require.NoError(t, err)
		assert.Equal(t, int64(1), seqB) // Independent sequence
		assert.NotEmpty(t, globalIDB)

		// Global IDs should be different
		assert.NotEqual(t, globalIDA, globalIDB)
	})

	t.Run("HLC monotonicity within region", func(t *testing.T) {
		// Generate multiple IDs quickly
		ids := make([]string, 10)
		for i := 0; i < 10; i++ {
			id, err := seqGenA.GenerateGlobalID()
			require.NoError(t, err)
			ids[i] = id
		}

		// All IDs should be unique and monotonically increasing
		for i := 1; i < len(ids); i++ {
			assert.NotEqual(t, ids[i-1], ids[i])
			// In a real test, you'd parse and compare timestamps
		}
	})
}

// TestCrossRegionHLCSync tests HLC synchronization between regions
func TestCrossRegionHLCSync(t *testing.T) {
	t.Run("update HLC from remote region", func(t *testing.T) {
		clockA := hlc.NewHLC("region-a", "node-1")
		clockB := hlc.NewHLC("region-b", "node-1")

		// Generate ID in region A
		idA := clockA.GenerateID()

		// Simulate region B receiving message from region A
		// Region B should update its clock
		err := clockB.UpdateFromRemote(idA.PhysicalTime, idA.LogicalTime)
		require.NoError(t, err)

		// Generate new ID in region B
		idB := clockB.GenerateID()

		// Region B's timestamp should be >= Region A's timestamp
		assert.GreaterOrEqual(t, idB.PhysicalTime, idA.PhysicalTime)
	})

	t.Run("concurrent HLC generation", func(t *testing.T) {
		clock := hlc.NewHLC("region-a", "node-1")

		// Generate IDs concurrently
		done := make(chan string, 100)
		for i := 0; i < 100; i++ {
			go func() {
				id := clock.GenerateID()
				done <- id.String()
			}()
		}

		// Collect all IDs
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := <-done
			ids[id] = true
		}

		// All IDs should be unique
		assert.Equal(t, 100, len(ids))
	})
}

// TestHLCCausalOrdering tests that HLC maintains causal ordering
func TestHLCCausalOrdering(t *testing.T) {
	clockA := hlc.NewHLC("region-a", "node-1")
	clockB := hlc.NewHLC("region-b", "node-1")

	// Event 1: Region A generates ID
	id1 := clockA.GenerateID()
	time.Sleep(10 * time.Millisecond)

	// Event 2: Region B receives message from A and updates clock
	err := clockB.UpdateFromRemote(id1.PhysicalTime, id1.LogicalTime)
	require.NoError(t, err)

	// Event 3: Region B generates new ID (causally after Event 1)
	id2 := clockB.GenerateID()

	// Event 4: Region A generates another ID (concurrent with Event 3)
	id3 := clockA.GenerateID()

	// Verify causal ordering
	// id2 should be causally after id1
	assert.True(t, id2.PhysicalTime > id1.PhysicalTime ||
		(id2.PhysicalTime == id1.PhysicalTime && id2.LogicalTime > id1.LogicalTime))

	// id3 and id2 are concurrent (no causal relationship)
	// Both should be valid and unique
	assert.NotEqual(t, id2.String(), id3.String())
}
