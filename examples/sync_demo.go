package main

import (
	"fmt"
	"time"
)

// SyncMessage represents a message to be synchronized (simplified for demo)
type SyncMessage struct {
	MessageID      string
	ConversationID string
	SenderID       string
	Content        string
	SequenceNumber int64
	Timestamp      int64
}

// MessageSyncer simplified for demo
type MessageSyncer struct {
	regionID string
	config   Config
}

// Config simplified for demo
type Config struct {
	EnableChecksum bool
}

func main() {
	fmt.Println("=== Multi-Region Message Synchronizer Demo ===")

	// Test basic functionality
	testChecksumCalculation()
	testSyncMessageCreation()
	testConfigDefaults()

	fmt.Println("\n=== Demo Complete ===")
}

func testChecksumCalculation() {
	fmt.Println("\n1. Testing checksum calculation...")

	syncMsg := &SyncMessage{
		MessageID:      "test-msg-001",
		ConversationID: "conv-123",
		SenderID:       "user-456",
		Content:        "Test message content",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	checksum1 := calculateChecksum(syncMsg)
	checksum2 := calculateChecksum(syncMsg)

	if checksum1 == checksum2 {
		fmt.Printf("✅ Checksum consistency: %s\n", checksum1)
	} else {
		fmt.Printf("❌ Checksum inconsistency: %s != %s\n", checksum1, checksum2)
	}

	// Test checksum changes with content
	syncMsg.Content = "Modified content"
	checksum3 := calculateChecksum(syncMsg)

	if checksum1 != checksum3 {
		fmt.Printf("✅ Checksum changes with content: %s -> %s\n", checksum1, checksum3)
	} else {
		fmt.Println("❌ Checksum should change with content")
	}
}

func testSyncMessageCreation() {
	fmt.Println("\n2. Testing SyncMessage creation...")

	syncMsg := &SyncMessage{
		MessageID:      "msg-001",
		ConversationID: "conv-123",
		SenderID:       "user-456",
		Content:        "Hello from region A!",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	fmt.Printf("✅ SyncMessage created: ID=%s, Sender=%s, Content=%s\n",
		syncMsg.MessageID, syncMsg.SenderID, syncMsg.Content)
}

func testConfigDefaults() {
	fmt.Println("\n3. Testing default configuration...")

	regionID := "test-region"
	config := defaultConfig(regionID)

	fmt.Printf("✅ Default config: Region=%s, Checksum=%t\n",
		config.RegionID, config.EnableChecksum)
}

// Simplified checksum calculation
func calculateChecksum(syncMsg *SyncMessage) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%d|%d",
		syncMsg.MessageID,
		syncMsg.ConversationID,
		syncMsg.SenderID,
		syncMsg.Content,
		syncMsg.SequenceNumber,
		syncMsg.Timestamp,
	)

	// Simple hash for demo (in real implementation, use SHA-256)
	hash := 0
	for _, c := range data {
		hash = hash*31 + int(c)
	}

	return fmt.Sprintf("hash-%d", hash)
}

// Simplified config for demo
type ConfigDemo struct {
	RegionID       string
	EnableChecksum bool
}

func defaultConfig(regionID string) ConfigDemo {
	return ConfigDemo{
		RegionID:       regionID,
		EnableChecksum: true,
	}
}
