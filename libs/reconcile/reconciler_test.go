package reconcile

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockMessageStore implements MessageStore for testing
type MockMessageStore struct {
	mu       sync.RWMutex
	messages map[string]*MessageData
}

func NewMockMessageStore() *MockMessageStore {
	return &MockMessageStore{
		messages: make(map[string]*MessageData),
	}
}

func (m *MockMessageStore) GetMessagesForReconciliation(ctx context.Context, startTime, endTime time.Time) ([]MessageData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MessageData, 0)
	for _, msg := range m.messages {
		// For testing, include all messages regardless of time window
		// In production, this would filter by timestamp
		result = append(result, *msg)
	}
	return result, nil
}

func (m *MockMessageStore) GetMessageByGlobalID(ctx context.Context, globalID string) (*MessageData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if msg, ok := m.messages[globalID]; ok {
		return msg, nil
	}
	return nil, fmt.Errorf("message not found: %s", globalID)
}

func (m *MockMessageStore) StoreMessage(ctx context.Context, msg *MessageData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[msg.GlobalID] = msg
	return nil
}

func (m *MockMessageStore) DeleteMessage(ctx context.Context, globalID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.messages, globalID)
	return nil
}

func (m *MockMessageStore) AddMessage(msg MessageData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages[msg.GlobalID] = &msg
}

func (m *MockMessageStore) GetMessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// MockRemoteTreeProvider implements RemoteTreeProvider for testing
type MockRemoteTreeProvider struct {
	mu          sync.RWMutex
	remoteTrees map[string]*MerkleTree
	remoteStore *MockMessageStore
}

func NewMockRemoteTreeProvider(remoteStore *MockMessageStore) *MockRemoteTreeProvider {
	return &MockRemoteTreeProvider{
		remoteTrees: make(map[string]*MerkleTree),
		remoteStore: remoteStore,
	}
}

func (m *MockRemoteTreeProvider) GetRemoteTree(ctx context.Context, regionID string, startTime, endTime time.Time) (*MerkleTree, error) {
	messages, err := m.remoteStore.GetMessagesForReconciliation(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Compute hashes
	for i := range messages {
		messages[i].Hash = ComputeMessageHash(messages[i])
	}

	return NewMerkleTree(regionID, messages), nil
}

func (m *MockRemoteTreeProvider) GetRemoteMessages(ctx context.Context, regionID string, globalIDs []string) ([]MessageData, error) {
	result := make([]MessageData, 0)
	for _, gid := range globalIDs {
		msg, err := m.remoteStore.GetMessageByGlobalID(ctx, gid)
		if err == nil {
			result = append(result, *msg)
		}
	}
	return result, nil
}

// TestNewReconciler tests creating a new reconciler
func TestNewReconciler(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	store := NewMockMessageStore()
	provider := NewMockRemoteTreeProvider(NewMockMessageStore())

	reconciler := NewReconciler(config, store, provider)

	if reconciler == nil {
		t.Fatal("Expected non-nil reconciler")
	}

	if reconciler.IsRunning() {
		t.Error("Reconciler should not be running initially")
	}
}

// TestReconcilerStartStop tests starting and stopping the reconciler
func TestReconcilerStartStop(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.CheckInterval = 100 * time.Millisecond
	store := NewMockMessageStore()
	provider := NewMockRemoteTreeProvider(NewMockMessageStore())

	reconciler := NewReconciler(config, store, provider)

	ctx := context.Background()

	// Start reconciler
	err := reconciler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start reconciler: %v", err)
	}

	if !reconciler.IsRunning() {
		t.Error("Reconciler should be running after start")
	}

	// Try to start again (should fail)
	err = reconciler.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running reconciler")
	}

	// Stop reconciler
	err = reconciler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop reconciler: %v", err)
	}

	if reconciler.IsRunning() {
		t.Error("Reconciler should not be running after stop")
	}

	// Try to stop again (should fail)
	err = reconciler.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped reconciler")
	}
}

// TestRunReconciliationIdenticalData tests reconciliation with identical data
func TestRunReconciliationIdenticalData(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 1 * time.Hour
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Add identical messages to both stores
	messages := createTestMessages(5, "region-a")
	for _, msg := range messages {
		localStore.AddMessage(msg)
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	stats, err := reconciler.RunReconciliation(ctx)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	if stats.Differences != 0 {
		t.Errorf("Expected 0 differences, got %d", stats.Differences)
	}

	if stats.Repaired != 0 {
		t.Errorf("Expected 0 repaired, got %d", stats.Repaired)
	}
}

// TestRunReconciliationMissingInLocal tests reconciliation with missing local messages
func TestRunReconciliationMissingInLocal(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 24 * time.Hour // Use larger window
	config.EnableAutoRepair = true
	config.DryRun = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 5 messages
	allMessages := createTestMessages(5, "region-a")

	// Add only first 3 to local
	for i := 0; i < 3; i++ {
		localStore.AddMessage(allMessages[i])
	}

	// Add all 5 to remote
	for _, msg := range allMessages {
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	stats, err := reconciler.RunReconciliation(ctx)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	// When trees have different structures, more differences are detected
	// The important thing is that missing messages are repaired
	if stats.Differences == 0 {
		t.Error("Expected some differences to be detected")
	}

	if stats.Repaired < 2 {
		t.Errorf("Expected at least 2 repaired, got %d", stats.Repaired)
	}

	// Verify messages were added to local store
	if localStore.GetMessageCount() != 5 {
		t.Errorf("Expected 5 messages in local store, got %d", localStore.GetMessageCount())
	}
}

// TestRunReconciliationDryRun tests reconciliation in dry-run mode
func TestRunReconciliationDryRun(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 24 * time.Hour // Use larger window
	config.EnableAutoRepair = true
	config.DryRun = true

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 5 messages
	allMessages := createTestMessages(5, "region-a")

	// Add only first 3 to local
	for i := 0; i < 3; i++ {
		localStore.AddMessage(allMessages[i])
	}

	// Add all 5 to remote
	for _, msg := range allMessages {
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	stats, err := reconciler.RunReconciliation(ctx)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	// When trees have different structures, more differences are detected
	if stats.Differences == 0 {
		t.Error("Expected some differences to be detected")
	}

	if stats.Repaired != 0 {
		t.Errorf("Expected 0 repaired in dry-run mode, got %d", stats.Repaired)
	}

	// Verify messages were NOT added to local store
	if localStore.GetMessageCount() != 3 {
		t.Errorf("Expected 3 messages in local store (no changes in dry-run), got %d", localStore.GetMessageCount())
	}
}

// TestRunReconciliationConflicts tests reconciliation with conflicts
func TestRunReconciliationConflicts(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 24 * time.Hour // Use larger window
	config.EnableAutoRepair = true
	config.DryRun = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Create 3 messages
	messages := createTestMessages(3, "region-a")

	// Add all messages to local
	for _, msg := range messages {
		localStore.AddMessage(msg)
	}

	// Modify one message for remote (create conflict)
	conflictMsg := messages[1]
	conflictMsg.Content = "Modified content"
	conflictMsg.Hash = ComputeMessageHash(conflictMsg)

	// Add messages to remote with one modified
	for i, msg := range messages {
		if i == 1 {
			remoteStore.AddMessage(conflictMsg)
		} else {
			remoteStore.AddMessage(msg)
		}
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	stats, err := reconciler.RunReconciliation(ctx)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	if stats.Differences != 1 {
		t.Errorf("Expected 1 difference (conflict), got %d", stats.Differences)
	}

	if stats.Repaired != 1 {
		t.Errorf("Expected 1 repaired, got %d", stats.Repaired)
	}

	// Verify local message was updated with remote version
	localMsg, err := localStore.GetMessageByGlobalID(ctx, conflictMsg.GlobalID)
	if err != nil {
		t.Fatalf("Failed to get local message: %v", err)
	}

	if localMsg.Content != "Modified content" {
		t.Errorf("Expected content 'Modified content', got '%s'", localMsg.Content)
	}
}

// TestGetLastRunStats tests retrieving last run statistics
func TestGetLastRunStats(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 1 * time.Hour
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	// Initially should be nil
	if reconciler.GetLastRunStats() != nil {
		t.Error("Expected nil stats before first run")
	}

	// Run reconciliation
	ctx := context.Background()
	_, err := reconciler.RunReconciliation(ctx)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	// Should have stats now
	stats := reconciler.GetLastRunStats()
	if stats == nil {
		t.Fatal("Expected non-nil stats after run")
	}

	if stats.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

// TestGetTotalStats tests retrieving cumulative statistics
func TestGetTotalStats(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 1 * time.Hour
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()

	// Run reconciliation multiple times
	for i := 0; i < 3; i++ {
		_, err := reconciler.RunReconciliation(ctx)
		if err != nil {
			t.Fatalf("RunReconciliation failed: %v", err)
		}
	}

	totalStats := reconciler.GetTotalStats()
	if totalStats["total_runs"].(int64) != 3 {
		t.Errorf("Expected 3 total runs, got %v", totalStats["total_runs"])
	}
}

// TestRunOnDemandReconciliation tests on-demand reconciliation
func TestRunOnDemandReconciliation(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Add messages
	messages := createTestMessages(5, "region-a")
	for _, msg := range messages {
		localStore.AddMessage(msg)
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	stats, diff, err := reconciler.RunOnDemandReconciliation(ctx, startTime, endTime, "region-b")
	if err != nil {
		t.Fatalf("RunOnDemandReconciliation failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if diff == nil {
		t.Fatal("Expected non-nil diff")
	}

	if diff.DiffCount != 0 {
		t.Errorf("Expected 0 differences, got %d", diff.DiffCount)
	}
}

// TestConcurrentReconciliation tests concurrent reconciliation operations
func TestConcurrentReconciliation(t *testing.T) {
	config := DefaultReconcilerConfig("region-a")
	config.TimeWindow = 1 * time.Hour
	config.EnableAutoRepair = false

	localStore := NewMockMessageStore()
	remoteStore := NewMockMessageStore()

	// Add messages
	messages := createTestMessages(10, "region-a")
	for _, msg := range messages {
		localStore.AddMessage(msg)
		remoteStore.AddMessage(msg)
	}

	provider := NewMockRemoteTreeProvider(remoteStore)
	reconciler := NewReconciler(config, localStore, provider)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Run multiple reconciliations concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := reconciler.RunReconciliation(ctx)
			if err != nil {
				t.Errorf("RunReconciliation failed: %v", err)
			}
		}()
	}

	wg.Wait()
}
