package reconcile

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestMerkleTreePerformance_SmallDataset tests performance with small dataset
func TestMerkleTreePerformance_SmallDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 1000
	messages := createTestMessages(messageCount, "region-a")

	start := time.Now()
	tree := NewMerkleTree("region-a", messages)
	buildDuration := time.Since(start)

	t.Logf("Built Merkle tree with %d messages in %v", messageCount, buildDuration)

	if tree.GetMessageCount() != messageCount {
		t.Errorf("Expected %d messages, got %d", messageCount, tree.GetMessageCount())
	}

	// Performance assertion: should build in < 100ms for 1K messages
	if buildDuration > 100*time.Millisecond {
		t.Errorf("Tree build took too long: %v (expected < 100ms)", buildDuration)
	}
}

// TestMerkleTreePerformance_MediumDataset tests performance with medium dataset
func TestMerkleTreePerformance_MediumDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 10000
	messages := createTestMessages(messageCount, "region-a")

	start := time.Now()
	tree := NewMerkleTree("region-a", messages)
	buildDuration := time.Since(start)

	t.Logf("Built Merkle tree with %d messages in %v", messageCount, buildDuration)

	if tree.GetMessageCount() != messageCount {
		t.Errorf("Expected %d messages, got %d", messageCount, tree.GetMessageCount())
	}

	// Performance assertion: should build in < 1s for 10K messages
	if buildDuration > 1*time.Second {
		t.Errorf("Tree build took too long: %v (expected < 1s)", buildDuration)
	}
}

// TestMerkleTreePerformance_LargeDataset tests performance with large dataset
func TestMerkleTreePerformance_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 100000
	messages := createTestMessages(messageCount, "region-a")

	start := time.Now()
	tree := NewMerkleTree("region-a", messages)
	buildDuration := time.Since(start)

	t.Logf("Built Merkle tree with %d messages in %v", messageCount, buildDuration)

	if tree.GetMessageCount() != messageCount {
		t.Errorf("Expected %d messages, got %d", messageCount, tree.GetMessageCount())
	}

	// Performance assertion: should build in < 10s for 100K messages
	if buildDuration > 10*time.Second {
		t.Errorf("Tree build took too long: %v (expected < 10s)", buildDuration)
	}

	// Test memory efficiency
	stats := tree.GetTreeStats()
	t.Logf("Tree stats: %+v", stats)
}

// TestMerkleTreeDiffPerformance_IdenticalTrees tests diff performance on identical trees
func TestMerkleTreeDiffPerformance_IdenticalTrees(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 10000
	messages := createTestMessages(messageCount, "region-a")

	tree1 := NewMerkleTree("region-a", messages)
	tree2 := NewMerkleTree("region-b", messages)

	ctx := context.Background()
	start := time.Now()
	diff, err := tree1.FindDifferences(ctx, tree2)
	diffDuration := time.Since(start)

	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	t.Logf("Compared %d messages in %v (identical trees)", messageCount, diffDuration)

	if diff.DiffCount != 0 {
		t.Errorf("Expected 0 differences, got %d", diff.DiffCount)
	}

	// Performance assertion: should compare in < 100ms for identical trees
	if diffDuration > 100*time.Millisecond {
		t.Errorf("Diff took too long: %v (expected < 100ms)", diffDuration)
	}
}

// TestMerkleTreeDiffPerformance_SmallDifferences tests diff performance with small differences
func TestMerkleTreeDiffPerformance_SmallDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 10000
	diffCount := 100 // 1% difference

	messages1 := createTestMessages(messageCount, "region-a")
	messages2 := createTestMessages(messageCount, "region-a")

	// Modify some messages to create differences
	for i := 0; i < diffCount; i++ {
		messages2[i].Content = "Modified content " + fmt.Sprint(i)
		messages2[i].Hash = ComputeMessageHash(messages2[i])
	}

	tree1 := NewMerkleTree("region-a", messages1)
	tree2 := NewMerkleTree("region-b", messages2)

	ctx := context.Background()
	start := time.Now()
	diff, err := tree1.FindDifferences(ctx, tree2)
	diffDuration := time.Since(start)

	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	t.Logf("Compared %d messages with %d differences in %v", messageCount, diffCount, diffDuration)
	t.Logf("Found %d differences", diff.DiffCount)

	// Performance assertion: should complete in < 500ms
	if diffDuration > 500*time.Millisecond {
		t.Errorf("Diff took too long: %v (expected < 500ms)", diffDuration)
	}
}

// TestMerkleTreeDiffPerformance_LargeDifferences tests diff performance with many differences
func TestMerkleTreeDiffPerformance_LargeDifferences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 10000
	diffCount := 5000 // 50% difference

	messages1 := createTestMessages(messageCount, "region-a")
	messages2 := createTestMessages(messageCount, "region-a")

	// Modify many messages to create differences
	for i := 0; i < diffCount; i++ {
		messages2[i].Content = "Modified content " + fmt.Sprint(i)
		messages2[i].Hash = ComputeMessageHash(messages2[i])
	}

	tree1 := NewMerkleTree("region-a", messages1)
	tree2 := NewMerkleTree("region-b", messages2)

	ctx := context.Background()
	start := time.Now()
	diff, err := tree1.FindDifferences(ctx, tree2)
	diffDuration := time.Since(start)

	if err != nil {
		t.Fatalf("FindDifferences failed: %v", err)
	}

	t.Logf("Compared %d messages with %d differences in %v", messageCount, diffCount, diffDuration)
	t.Logf("Found %d differences", diff.DiffCount)

	// Performance assertion: should complete in < 2s even with many differences
	if diffDuration > 2*time.Second {
		t.Errorf("Diff took too long: %v (expected < 2s)", diffDuration)
	}
}

// TestReconciliationPerformance_SmallDataset tests full reconciliation performance
func TestReconciliationPerformance_SmallDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 24 * time.Hour
	config.EnableAutoRepair = true
	config.DryRun = false
	config.MaxConcurrentFixes = 10

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 1000 messages with 10 differences
	messageCount := 1000
	diffCount := 10

	allMessages := createTestMessages(messageCount, "region-a")

	// Add all to local
	for _, msg := range allMessages {
		localStore.AddMessage(msg)
	}

	// Add all but diffCount to remote
	for i := diffCount; i < messageCount; i++ {
		remoteStore.AddMessage(allMessages[i])
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	start := time.Now()
	stats, err := reconciler.RunReconciliation(ctx)
	reconcileDuration := time.Since(start)

	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	t.Logf("Reconciled %d messages with %d differences in %v", messageCount, diffCount, reconcileDuration)
	t.Logf("Stats: Checked=%d, Differences=%d, Repaired=%d, Failed=%d",
		stats.MessagesChecked, stats.Differences, stats.Repaired, stats.Failed)

	// Performance assertion: should complete in < 1s for 1K messages
	if reconcileDuration > 1*time.Second {
		t.Errorf("Reconciliation took too long: %v (expected < 1s)", reconcileDuration)
	}
}

// TestReconciliationPerformance_MediumDataset tests reconciliation with medium dataset
func TestReconciliationPerformance_MediumDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 24 * time.Hour
	config.EnableAutoRepair = true
	config.DryRun = false
	config.MaxConcurrentFixes = 20

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 10000 messages with 100 differences
	messageCount := 10000
	diffCount := 100

	allMessages := createTestMessages(messageCount, "region-a")

	// Add all to local
	for _, msg := range allMessages {
		localStore.AddMessage(msg)
	}

	// Add all but diffCount to remote
	for i := diffCount; i < messageCount; i++ {
		remoteStore.AddMessage(allMessages[i])
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	start := time.Now()
	stats, err := reconciler.RunReconciliation(ctx)
	reconcileDuration := time.Since(start)

	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	t.Logf("Reconciled %d messages with %d differences in %v", messageCount, diffCount, reconcileDuration)
	t.Logf("Stats: Checked=%d, Differences=%d, Repaired=%d, Failed=%d",
		stats.MessagesChecked, stats.Differences, stats.Repaired, stats.Failed)

	// Performance assertion: should complete in < 5s for 10K messages
	if reconcileDuration > 5*time.Second {
		t.Errorf("Reconciliation took too long: %v (expected < 5s)", reconcileDuration)
	}
}

// TestIncrementalRepairPerformance tests incremental repair performance
func TestIncrementalRepairPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := RepairConfig{
		RegionID:       "region-a",
		Strategy:       RepairStrategyPull,
		BatchSize:      50,
		RetryAttempts:  3,
		RetryDelay:     10 * time.Millisecond,
		MaxQueueSize:   10000,
		EnablePriority: false,
	}

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 1000 messages to repair
	repairCount := 1000
	messages := createTestMessages(repairCount, "region-a")

	// Add to remote only
	for _, msg := range messages {
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	repairer := NewIncrementalRepairer(config, localStore, provider)

	// Create repair tasks
	tasks := make([]RepairTask, repairCount)
	for i, msg := range messages {
		tasks[i] = RepairTask{
			GlobalID:     msg.GlobalID,
			Operation:    "fetch",
			Priority:     1,
			TargetRegion: "region-b",
		}
	}

	// Queue all tasks
	err := repairer.QueueRepairs(tasks)
	if err != nil {
		t.Fatalf("Failed to queue repairs: %v", err)
	}

	ctx := context.Background()
	start := time.Now()

	// Process all batches
	totalSuccessful := 0
	totalFailed := 0
	for repairer.GetQueueSize() > 0 {
		result, err := repairer.ProcessBatch(ctx)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		totalSuccessful += result.Successful
		totalFailed += result.Failed
	}

	repairDuration := time.Since(start)

	t.Logf("Repaired %d messages in %v (success rate: %.2f%%)",
		repairCount, repairDuration, float64(totalSuccessful)/float64(repairCount)*100)

	// Performance assertion: should repair 1000 messages in < 5s
	if repairDuration > 5*time.Second {
		t.Errorf("Repair took too long: %v (expected < 5s)", repairDuration)
	}

	// Verify all messages were repaired
	if totalSuccessful != repairCount {
		t.Errorf("Expected %d successful repairs, got %d", repairCount, totalSuccessful)
	}
}

// TestConcurrentReconciliationPerformance tests concurrent reconciliation operations
func TestConcurrentReconciliationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 1 * time.Hour
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Add messages
	messages := createTestMessages(1000, "region-a")
	for _, msg := range messages {
		localStore.AddMessage(msg)
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	concurrency := 10

	start := time.Now()
	done := make(chan error, concurrency)

	// Run multiple reconciliations concurrently
	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := reconciler.RunReconciliation(ctx)
			done <- err
		}()
	}

	// Wait for all to complete
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent reconciliation failed: %v", err)
		}
	}

	duration := time.Since(start)
	t.Logf("Completed %d concurrent reconciliations in %v", concurrency, duration)

	// Performance assertion: concurrent operations should complete in reasonable time
	if duration > 5*time.Second {
		t.Errorf("Concurrent reconciliation took too long: %v (expected < 5s)", duration)
	}
}

// TestMemoryUsage tests memory efficiency of Merkle tree
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageCount := 10000
	messages := createTestMessages(messageCount, "region-a")

	// Build tree
	tree := NewMerkleTree("region-a", messages)

	// Get stats
	stats := tree.GetTreeStats()
	t.Logf("Tree stats for %d messages: %+v", messageCount, stats)

	// Verify tree is built correctly
	if tree.GetMessageCount() != messageCount {
		t.Errorf("Expected %d messages, got %d", messageCount, tree.GetMessageCount())
	}

	// Note: Actual memory usage measurement would require runtime.MemStats
	// This test primarily validates that the tree can handle large datasets
}

// BenchmarkMerkleTreeBuild benchmarks tree construction
func BenchmarkMerkleTreeBuild(b *testing.B) {
	messages := createTestMessages(1000, "region-a")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMerkleTree("region-a", messages)
	}
}

// BenchmarkMerkleTreeDiff benchmarks tree comparison
func BenchmarkMerkleTreeDiff(b *testing.B) {
	messages := createTestMessages(1000, "region-a")
	tree1 := NewMerkleTree("region-a", messages)
	tree2 := NewMerkleTree("region-b", messages)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tree1.FindDifferences(ctx, tree2)
	}
}

// BenchmarkIncrementalRepair benchmarks repair operations
func BenchmarkIncrementalRepair(b *testing.B) {
	config := RepairConfig{
		RegionID:       "region-a",
		Strategy:       RepairStrategyPull,
		BatchSize:      50,
		RetryAttempts:  1,
		RetryDelay:     1 * time.Millisecond,
		MaxQueueSize:   10000,
		EnablePriority: false,
	}

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	messages := createTestMessages(100, "region-a")
	for _, msg := range messages {
		remoteStore.AddMessage(msg)
	}

	tasks := make([]RepairTask, len(messages))
	for i, msg := range messages {
		tasks[i] = RepairTask{
			GlobalID:     msg.GlobalID,
			Operation:    "fetch",
			Priority:     1,
			TargetRegion: "region-b",
		}
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	repairer := NewIncrementalRepairer(config, localStore, provider)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repairer.QueueRepairs(tasks)
		for repairer.GetQueueSize() > 0 {
			_, _ = repairer.ProcessBatch(ctx)
		}
	}
}
