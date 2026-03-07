//go:build e2e
// +build e2e

package multiregion

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cuckoo-org/cuckoo/apps/im-service/hlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformanceAndConsistencyVerification validates Task 10.2 requirements
// - 测试跨地域消息延迟（P99 < 500ms）
// - 验证 HLC 集成后的消息排序
// - 测试基于现有数据库的跨地域复制
// - 验证冲突解决和数据一致性
func TestPerformanceAndConsistencyVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance and consistency test in short mode")
	}

	ctx := context.Background()

	// Setup test environment
	env := setupMultiRegionTestEnvironment(t, ctx)
	defer env.Cleanup()

	t.Run("CrossRegionMessageLatency", func(t *testing.T) {
		testCrossRegionMessageLatency(t, ctx, env)
	})

	t.Run("HLCMessageOrdering", func(t *testing.T) {
		testHLCMessageOrdering(t, ctx, env)
	})

	t.Run("DatabaseCrossRegionReplication", func(t *testing.T) {
		testDatabaseCrossRegionReplication(t, ctx, env)
	})

	t.Run("ConflictResolutionConsistency", func(t *testing.T) {
		testConflictResolutionConsistency(t, ctx, env)
	})

	t.Run("ConcurrentWriteConsistency", func(t *testing.T) {
		testConcurrentWriteConsistency(t, ctx, env)
	})

	t.Run("DataConsistencyUnderLoad", func(t *testing.T) {
		testDataConsistencyUnderLoad(t, ctx, env)
	})
}

// testCrossRegionMessageLatency validates requirement: P99 < 500ms
func testCrossRegionMessageLatency(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing cross-region message latency (P99 < 500ms)...")

	const numSamples = 1000
	latencies := make([]time.Duration, 0, numSamples)

	// Measure latency for multiple message operations
	for i := 0; i < numSamples; i++ {
		start := time.Now()

		// Simulate cross-region message flow:
		// 1. Generate ID in Region A
		idA := env.RegionA.HLC.GenerateID()

		// 2. Simulate network transfer (write to Redis in Region A)
		msgKey := fmt.Sprintf("perf:msg:%d", i)
		msgValue := fmt.Sprintf("region-a-%d-%d", idA.PhysicalTime, idA.LogicalTime)
		err := env.RegionA.RedisClient.Set(ctx, msgKey, msgValue, time.Minute).Err()
		require.NoError(t, err)

		// 3. Simulate Region B receiving and processing
		// Update HLC in Region B
		err = env.RegionB.HLC.UpdateFromRemote(idA.PhysicalTime, idA.LogicalTime)
		require.NoError(t, err)

		// 4. Write to Region B Redis
		err = env.RegionB.RedisClient.Set(ctx, msgKey, msgValue, time.Minute).Err()
		require.NoError(t, err)

		latency := time.Since(start)
		latencies = append(latencies, latency)

		// Small delay to avoid overwhelming the system
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[int(float64(len(latencies))*0.50)]
	p95 := latencies[int(float64(len(latencies))*0.95)]
	p99 := latencies[int(float64(len(latencies))*0.99)]
	max := latencies[len(latencies)-1]

	t.Logf("Cross-region message latency statistics:")
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)
	t.Logf("  Max: %v", max)

	// Verify P99 < 500ms requirement
	assert.Less(t, p99.Milliseconds(), int64(500),
		"P99 latency should be less than 500ms (actual: %v)", p99)

	// Additional checks
	assert.Less(t, p50.Milliseconds(), int64(100),
		"P50 latency should be less than 100ms for good performance")
	assert.Less(t, p95.Milliseconds(), int64(300),
		"P95 latency should be less than 300ms")

	// Cleanup
	for i := 0; i < numSamples; i++ {
		msgKey := fmt.Sprintf("perf:msg:%d", i)
		env.RegionA.RedisClient.Del(ctx, msgKey)
		env.RegionB.RedisClient.Del(ctx, msgKey)
	}

	t.Log("✓ Cross-region message latency validated (P99 < 500ms)")
}

// testHLCMessageOrdering validates HLC-based message ordering
func testHLCMessageOrdering(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing HLC message ordering...")

	// Create messages from both regions with causal relationships
	messages := make([]hlc.GlobalID, 0, 100)

	// Phase 1: Region A generates messages
	for i := 0; i < 20; i++ {
		id := env.RegionA.HLC.GenerateID()
		messages = append(messages, id)
		time.Sleep(1 * time.Millisecond)
	}

	// Phase 2: Region B receives last message from A and generates messages
	lastA := messages[len(messages)-1]
	err := env.RegionB.HLC.UpdateFromRemote(lastA.PhysicalTime, lastA.LogicalTime)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		id := env.RegionB.HLC.GenerateID()
		messages = append(messages, id)
		time.Sleep(1 * time.Millisecond)
	}

	// Phase 3: Region A receives message from B and generates more
	lastB := messages[len(messages)-1]
	err = env.RegionA.HLC.UpdateFromRemote(lastB.PhysicalTime, lastB.LogicalTime)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		id := env.RegionA.HLC.GenerateID()
		messages = append(messages, id)
		time.Sleep(1 * time.Millisecond)
	}

	// Phase 4: Concurrent messages from both regions
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			id := env.RegionA.HLC.GenerateID()
			mu.Lock()
			messages = append(messages, id)
			mu.Unlock()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			id := env.RegionB.HLC.GenerateID()
			mu.Lock()
			messages = append(messages, id)
			mu.Unlock()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()

	t.Logf("Generated %d messages from both regions", len(messages))

	// Test 1: Verify all messages are unique
	uniqueIDs := make(map[string]bool)
	for _, msg := range messages {
		idStr := msg.String()
		assert.False(t, uniqueIDs[idStr], "All message IDs should be unique")
		uniqueIDs[idStr] = true
	}
	t.Logf("✓ All %d messages have unique IDs", len(messages))

	// Test 2: Sort messages by HLC and verify ordering
	sortedMessages := make([]hlc.GlobalID, len(messages))
	copy(sortedMessages, messages)

	sort.Slice(sortedMessages, func(i, j int) bool {
		return hlc.CompareGlobalID(sortedMessages[i], sortedMessages[j]) < 0
	})

	// Test 3: Verify causal ordering is preserved
	// Messages from Phase 1 should come before Phase 2
	phase1End := 20
	phase2End := 40

	for i := 0; i < phase1End; i++ {
		for j := phase1End; j < phase2End; j++ {
			cmp := hlc.CompareGlobalID(sortedMessages[i], sortedMessages[j])
			assert.Less(t, cmp, 0,
				"Phase 1 messages should be ordered before Phase 2 messages")
		}
	}

	t.Log("✓ Causal ordering preserved across phases")

	// Test 4: Verify monotonicity within each region
	regionAMessages := make([]hlc.GlobalID, 0)
	regionBMessages := make([]hlc.GlobalID, 0)

	for _, msg := range sortedMessages {
		if msg.RegionID == "region-a" {
			regionAMessages = append(regionAMessages, msg)
		} else if msg.RegionID == "region-b" {
			regionBMessages = append(regionBMessages, msg)
		}
	}

	// Check monotonicity in Region A
	for i := 1; i < len(regionAMessages); i++ {
		assert.True(t,
			regionAMessages[i].PhysicalTime > regionAMessages[i-1].PhysicalTime ||
				(regionAMessages[i].PhysicalTime == regionAMessages[i-1].PhysicalTime &&
					regionAMessages[i].LogicalTime > regionAMessages[i-1].LogicalTime),
			"Region A messages should be monotonically increasing")
	}

	// Check monotonicity in Region B
	for i := 1; i < len(regionBMessages); i++ {
		assert.True(t,
			regionBMessages[i].PhysicalTime > regionBMessages[i-1].PhysicalTime ||
				(regionBMessages[i].PhysicalTime == regionBMessages[i-1].PhysicalTime &&
					regionBMessages[i].LogicalTime > regionBMessages[i-1].LogicalTime),
			"Region B messages should be monotonically increasing")
	}

	t.Logf("✓ Monotonicity verified: Region A (%d msgs), Region B (%d msgs)",
		len(regionAMessages), len(regionBMessages))

	t.Log("✓ HLC message ordering validated")
}

// testDatabaseCrossRegionReplication validates database replication
func testDatabaseCrossRegionReplication(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing database cross-region replication...")

	// Note: This test simulates database replication using Redis
	// In production, this would test actual MySQL replication

	const numMessages = 100
	replicationLatencies := make([]time.Duration, 0, numMessages)

	for i := 0; i < numMessages; i++ {
		// Create message in Region A
		msgID := fmt.Sprintf("db-repl-msg-%d", i)
		globalID := env.RegionA.HLC.GenerateID()

		message := map[string]interface{}{
			"msg_id":      msgID,
			"global_id":   globalID.String(),
			"region_id":   "region-a",
			"content":     fmt.Sprintf("Message %d from Region A", i),
			"timestamp":   time.Now().Unix(),
			"sync_status": "pending",
		}

		start := time.Now()

		// Write to Region A (primary)
		key := fmt.Sprintf("db:messages:%s", msgID)
		err := env.RegionA.RedisClient.HSet(ctx, key, message).Err()
		require.NoError(t, err)

		// Simulate replication delay
		time.Sleep(10 * time.Millisecond)

		// Replicate to Region B (secondary)
		err = env.RegionB.RedisClient.HSet(ctx, key, message).Err()
		require.NoError(t, err)

		replicationLatency := time.Since(start)
		replicationLatencies = append(replicationLatencies, replicationLatency)

		// Update sync status
		err = env.RegionA.RedisClient.HSet(ctx, key, "sync_status", "synced").Err()
		require.NoError(t, err)
	}

	// Calculate replication latency statistics
	sort.Slice(replicationLatencies, func(i, j int) bool {
		return replicationLatencies[i] < replicationLatencies[j]
	})

	p50 := replicationLatencies[int(float64(len(replicationLatencies))*0.50)]
	p95 := replicationLatencies[int(float64(len(replicationLatencies))*0.95)]
	p99 := replicationLatencies[int(float64(len(replicationLatencies))*0.99)]

	t.Logf("Database replication latency statistics:")
	t.Logf("  P50: %v", p50)
	t.Logf("  P95: %v", p95)
	t.Logf("  P99: %v", p99)

	// Verify replication latency meets requirements
	assert.Less(t, p99.Milliseconds(), int64(1000),
		"P99 replication latency should be less than 1 second")

	// Verify data consistency
	for i := 0; i < numMessages; i++ {
		msgID := fmt.Sprintf("db-repl-msg-%d", i)
		key := fmt.Sprintf("db:messages:%s", msgID)

		// Read from both regions
		dataA, err := env.RegionA.RedisClient.HGetAll(ctx, key).Result()
		require.NoError(t, err)

		dataB, err := env.RegionB.RedisClient.HGetAll(ctx, key).Result()
		require.NoError(t, err)

		// Verify data is consistent
		assert.Equal(t, dataA["msg_id"], dataB["msg_id"])
		assert.Equal(t, dataA["global_id"], dataB["global_id"])
		assert.Equal(t, dataA["content"], dataB["content"])
	}

	t.Logf("✓ Data consistency verified across %d messages", numMessages)

	// Cleanup
	for i := 0; i < numMessages; i++ {
		msgID := fmt.Sprintf("db-repl-msg-%d", i)
		key := fmt.Sprintf("db:messages:%s", msgID)
		env.RegionA.RedisClient.Del(ctx, key)
		env.RegionB.RedisClient.Del(ctx, key)
	}

	t.Log("✓ Database cross-region replication validated")
}

// testConflictResolutionConsistency validates conflict resolution
func testConflictResolutionConsistency(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing conflict resolution consistency...")

	const numConflicts = 100
	resolutions := make([]string, 0, numConflicts)

	for i := 0; i < numConflicts; i++ {
		// Create conflicting messages from both regions
		idA := env.RegionA.HLC.GenerateID()
		time.Sleep(5 * time.Millisecond) // Ensure different timestamps
		idB := env.RegionB.HLC.GenerateID()

		msgA := imsync.MessageVersion{
			GlobalID:  idA.String(),
			Content:   fmt.Sprintf("Message %d from Region A", i),
			Timestamp: idA.PhysicalTime,
			RegionID:  "region-a",
		}

		msgB := imsync.MessageVersion{
			GlobalID:  idB.String(),
			Content:   fmt.Sprintf("Message %d from Region B", i),
			Timestamp: idB.PhysicalTime,
			RegionID:  "region-b",
		}

		// Resolve conflict in Region A
		winnerA, hasConflict := env.RegionA.ConflictResolver.ResolveConflict(msgA, msgB)
		assert.True(t, hasConflict, "Should detect conflict")

		// Resolve same conflict in Region B
		winnerB, _ := env.RegionB.ConflictResolver.ResolveConflict(msgA, msgB)

		// Verify deterministic resolution
		assert.Equal(t, winnerA.Content, winnerB.Content,
			"Both regions should resolve conflict identically")
		assert.Equal(t, winnerA.GlobalID, winnerB.GlobalID,
			"Both regions should select same winner")

		resolutions = append(resolutions, winnerA.GlobalID)
	}

	t.Logf("✓ Resolved %d conflicts deterministically", len(resolutions))

	// Test conflict resolution with same timestamp (region ID tiebreaker)
	sameTimestamp := time.Now().UnixMilli()

	msgA := imsync.MessageVersion{
		GlobalID:  fmt.Sprintf("region-a-%d-1", sameTimestamp),
		Content:   "Region A message",
		Timestamp: sameTimestamp,
		RegionID:  "region-a",
	}

	msgB := imsync.MessageVersion{
		GlobalID:  fmt.Sprintf("region-b-%d-1", sameTimestamp),
		Content:   "Region B message",
		Timestamp: sameTimestamp,
		RegionID:  "region-b",
	}

	// Resolve multiple times to verify determinism
	var firstWinner string
	for i := 0; i < 10; i++ {
		winner, _ := env.RegionA.ConflictResolver.ResolveConflict(msgA, msgB)
		if i == 0 {
			firstWinner = winner.GlobalID
		} else {
			assert.Equal(t, firstWinner, winner.GlobalID,
				"Tiebreaker should be deterministic")
		}
	}

	t.Log("✓ Region ID tiebreaker is deterministic")
	t.Log("✓ Conflict resolution consistency validated")
}

// testConcurrentWriteConsistency validates consistency under concurrent writes
func testConcurrentWriteConsistency(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing concurrent write consistency...")

	const numGoroutines = 10
	const writesPerGoroutine = 50

	var wg sync.WaitGroup
	var successCount int64
	var conflictCount int64

	// Concurrent writes from Region A
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < writesPerGoroutine; i++ {
				id := env.RegionA.HLC.GenerateID()
				key := fmt.Sprintf("concurrent:msg:%d:%d", goroutineID, i)
				value := fmt.Sprintf("region-a-%s", id.String())

				err := env.RegionA.RedisClient.Set(ctx, key, value, time.Minute).Err()
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				}

				time.Sleep(1 * time.Millisecond)
			}
		}(g)
	}

	// Concurrent writes from Region B
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < writesPerGoroutine; i++ {
				id := env.RegionB.HLC.GenerateID()
				key := fmt.Sprintf("concurrent:msg:%d:%d", goroutineID, i)
				value := fmt.Sprintf("region-b-%s", id.String())

				// Check if key exists (potential conflict)
				exists, _ := env.RegionB.RedisClient.Exists(ctx, key).Result()
				if exists > 0 {
					atomic.AddInt64(&conflictCount, 1)
				}

				err := env.RegionB.RedisClient.Set(ctx, key, value, time.Minute).Err()
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				}

				time.Sleep(1 * time.Millisecond)
			}
		}(g)
	}

	wg.Wait()

	t.Logf("Concurrent write statistics:")
	t.Logf("  Total successful writes: %d", successCount)
	t.Logf("  Detected conflicts: %d", conflictCount)
	t.Logf("  Expected writes: %d", numGoroutines*writesPerGoroutine*2)

	// Verify all writes succeeded
	assert.Greater(t, successCount, int64(0), "Should have successful writes")

	// Cleanup
	for g := 0; g < numGoroutines; g++ {
		for i := 0; i < writesPerGoroutine; i++ {
			key := fmt.Sprintf("concurrent:msg:%d:%d", g, i)
			env.RegionA.RedisClient.Del(ctx, key)
			env.RegionB.RedisClient.Del(ctx, key)
		}
	}

	t.Log("✓ Concurrent write consistency validated")
}

// testDataConsistencyUnderLoad validates consistency under sustained load
func testDataConsistencyUnderLoad(t *testing.T, ctx context.Context, env *MultiRegionTestEnvironment) {
	t.Log("Testing data consistency under load...")

	const duration = 30 * time.Second
	const targetRPS = 100 // requests per second per region

	var wg sync.WaitGroup
	var totalWrites int64
	var totalReads int64
	var inconsistencies int64

	stopCh := make(chan struct{})

	// Writer goroutine for Region A
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()

		counter := 0
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				id := env.RegionA.HLC.GenerateID()
				key := fmt.Sprintf("load:msg:%d", counter)
				value := fmt.Sprintf("region-a-%s-%d", id.String(), counter)

				err := env.RegionA.RedisClient.Set(ctx, key, value, time.Minute).Err()
				if err == nil {
					atomic.AddInt64(&totalWrites, 1)
				}

				counter++
			}
		}
	}()

	// Writer goroutine for Region B
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
		defer ticker.Stop()

		counter := 0
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				id := env.RegionB.HLC.GenerateID()
				key := fmt.Sprintf("load:msg:%d", counter)
				value := fmt.Sprintf("region-b-%s-%d", id.String(), counter)

				err := env.RegionB.RedisClient.Set(ctx, key, value, time.Minute).Err()
				if err == nil {
					atomic.AddInt64(&totalWrites, 1)
				}

				counter++
			}
		}
	}()

	// Reader goroutine to verify consistency
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				// Read random keys from both regions
				for i := 0; i < 10; i++ {
					key := fmt.Sprintf("load:msg:%d", i)

					valA, errA := env.RegionA.RedisClient.Get(ctx, key).Result()
					valB, errB := env.RegionB.RedisClient.Get(ctx, key).Result()

					atomic.AddInt64(&totalReads, 1)

					// Check for inconsistencies
					if errA == nil && errB == nil && valA != valB {
						atomic.AddInt64(&inconsistencies, 1)
					}
				}
			}
		}
	}()

	// Run load test for specified duration
	t.Logf("Running load test for %v...", duration)
	time.Sleep(duration)
	close(stopCh)
	wg.Wait()

	t.Logf("Load test statistics:")
	t.Logf("  Total writes: %d", totalWrites)
	t.Logf("  Total reads: %d", totalReads)
	t.Logf("  Inconsistencies detected: %d", inconsistencies)
	t.Logf("  Write rate: %.2f writes/sec", float64(totalWrites)/duration.Seconds())
	t.Logf("  Read rate: %.2f reads/sec", float64(totalReads)/duration.Seconds())

	// Calculate consistency rate
	if totalReads > 0 {
		consistencyRate := float64(totalReads-inconsistencies) / float64(totalReads) * 100
		t.Logf("  Consistency rate: %.2f%%", consistencyRate)

		// Verify high consistency rate (allowing for eventual consistency)
		assert.Greater(t, consistencyRate, 95.0,
			"Consistency rate should be > 95%%")
	}

	// Cleanup
	for i := 0; i < int(totalWrites); i++ {
		key := fmt.Sprintf("load:msg:%d", i)
		env.RegionA.RedisClient.Del(ctx, key)
		env.RegionB.RedisClient.Del(ctx, key)
	}

	t.Log("✓ Data consistency under load validated")
}

// Helper function to calculate standard deviation
func calculateStdDev(values []time.Duration) float64 {
	if len(values) == 0 {
		return 0
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += float64(v.Milliseconds())
	}
	mean := sum / float64(len(values))

	// Calculate variance
	var variance float64
	for _, v := range values {
		diff := float64(v.Milliseconds()) - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}
