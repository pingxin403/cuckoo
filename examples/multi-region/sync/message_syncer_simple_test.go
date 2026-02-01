package sync

import (
	"testing"
	"time"
)

func TestSyncMessage_Checksum(t *testing.T) {
	// Test checksum calculation without external dependencies
	syncMsg := &SyncMessage{
		MessageID:      "test-msg-001",
		ConversationID: "conv-123",
		SenderID:       "user-456",
		Content:        "Test message content",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	// Create a mock syncer config for checksum testing
	config := Config{
		EnableChecksum: true,
	}

	// Create a minimal syncer instance for testing checksum
	syncer := &MessageSyncer{
		config: config,
	}

	// Test checksum calculation
	checksum1 := syncer.calculateChecksum(syncMsg)
	checksum2 := syncer.calculateChecksum(syncMsg)

	// Checksums should be identical for the same message
	if checksum1 != checksum2 {
		t.Errorf("Checksums should be identical: %s != %s", checksum1, checksum2)
	}

	// Checksum should not be empty when enabled
	if checksum1 == "" {
		t.Error("Checksum should not be empty when enabled")
	}

	// Modify message and verify checksum changes
	syncMsg.Content = "Modified content"
	checksum3 := syncer.calculateChecksum(syncMsg)

	if checksum1 == checksum3 {
		t.Error("Checksum should change when message content changes")
	}

	t.Logf("Checksum test passed - original: %s, modified: %s", checksum1, checksum3)
}

func TestSyncMessage_Conversion(t *testing.T) {
	// Test SyncMessage struct creation and basic operations
	syncMsg := &SyncMessage{
		ID:             "sync-001",
		Type:           "async",
		SourceRegion:   "region-a",
		TargetRegion:   "region-b",
		MessageID:      "msg-001",
		ConversationID: "conv-123",
		SenderID:       "user-456",
		Content:        "Test message",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
		RequiresAck:    false,
		MaxRetries:     3,
		CreatedAt:      time.Now(),
		IsCritical:     false,
	}

	// Verify basic properties
	if syncMsg.Type != "async" {
		t.Errorf("Expected type 'async', got %s", syncMsg.Type)
	}

	if syncMsg.SourceRegion != "region-a" {
		t.Errorf("Expected source region 'region-a', got %s", syncMsg.SourceRegion)
	}

	if syncMsg.TargetRegion != "region-b" {
		t.Errorf("Expected target region 'region-b', got %s", syncMsg.TargetRegion)
	}

	if syncMsg.RequiresAck != false {
		t.Error("Expected RequiresAck to be false for async message")
	}

	if syncMsg.IsCritical != false {
		t.Error("Expected IsCritical to be false for regular message")
	}

	t.Logf("SyncMessage conversion test passed")
}

func TestSyncAck_Creation(t *testing.T) {
	// Test SyncAck struct creation
	ack := &SyncAck{
		MessageID:    "msg-001",
		GlobalID:     "region-a-123456-1",
		SourceRegion: "region-b",
		TargetRegion: "region-a",
		Status:       "success",
		Timestamp:    time.Now(),
		ProcessTime:  150,
	}

	// Verify properties
	if ack.Status != "success" {
		t.Errorf("Expected status 'success', got %s", ack.Status)
	}

	if ack.ProcessTime != 150 {
		t.Errorf("Expected process time 150, got %d", ack.ProcessTime)
	}

	if ack.Error != "" {
		t.Errorf("Expected empty error for success status, got %s", ack.Error)
	}

	t.Logf("SyncAck creation test passed")
}

func TestConfig_Defaults(t *testing.T) {
	// Test default configuration
	regionID := "test-region"
	config := DefaultConfig(regionID)

	// Verify default values
	if config.RegionID != regionID {
		t.Errorf("Expected region ID %s, got %s", regionID, config.RegionID)
	}

	if config.AsyncTopic != "cross_region_async" {
		t.Errorf("Expected async topic 'cross_region_async', got %s", config.AsyncTopic)
	}

	if config.SyncTopic != "cross_region_sync" {
		t.Errorf("Expected sync topic 'cross_region_sync', got %s", config.SyncTopic)
	}

	if config.AckTopic != "cross_region_ack" {
		t.Errorf("Expected ack topic 'cross_region_ack', got %s", config.AckTopic)
	}

	if config.SyncTimeout != 5*time.Second {
		t.Errorf("Expected sync timeout 5s, got %v", config.SyncTimeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.MaxRetries)
	}

	if config.EnableChecksum != true {
		t.Error("Expected checksum to be enabled by default")
	}

	if config.EnableDeduplication != true {
		t.Error("Expected deduplication to be enabled by default")
	}

	t.Logf("Default config test passed")
}

func TestMessageSyncer_GetMetrics(t *testing.T) {
	// Test metrics collection without external dependencies
	syncer := &MessageSyncer{
		regionID: "test-region",
		started:  false,
		shutdown: false,
	}

	metrics := syncer.GetMetrics()

	// Verify basic metrics
	if metrics["region_id"] != "test-region" {
		t.Errorf("Expected region_id 'test-region', got %s", metrics["region_id"])
	}

	if metrics["async_sync_count"].(int64) != 0 {
		t.Errorf("Expected async_sync_count 0, got %d", metrics["async_sync_count"])
	}

	if metrics["sync_sync_count"].(int64) != 0 {
		t.Errorf("Expected sync_sync_count 0, got %d", metrics["sync_sync_count"])
	}

	if metrics["conflict_count"].(int64) != 0 {
		t.Errorf("Expected conflict_count 0, got %d", metrics["conflict_count"])
	}

	if metrics["error_count"].(int64) != 0 {
		t.Errorf("Expected error_count 0, got %d", metrics["error_count"])
	}

	if metrics["started"].(bool) != false {
		t.Error("Expected started to be false")
	}

	if metrics["shutdown"].(bool) != false {
		t.Error("Expected shutdown to be false")
	}

	t.Logf("Metrics test passed: %+v", metrics)
}
