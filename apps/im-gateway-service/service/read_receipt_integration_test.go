package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestReadReceiptMultiDeviceSync tests end-to-end read receipt sync across devices
// Validates: Requirements 15.4 (read receipt sync across devices)
func TestReadReceiptMultiDeviceSync(t *testing.T) {
	// Create mock registry
	mockRegistry := newMockRegistryClient()

	// Create gateway service
	gateway := &GatewayService{
		registryClient: mockRegistry,
	}
	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      context.Background(),
	})
	gateway.connections.Store("user123_device2", &Connection{
		UserID:   "user123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		ctx:      context.Background(),
	})

	// Set up mock registry to return both devices
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
	})

	// Create push service
	pushService := NewPushService(gateway)

	// Simulate read receipt event
	req := &PushReadReceiptRequest{
		MsgID:          "msg-123",
		SenderID:       "user123", // Original sender (receives the receipt)
		ReaderID:       "user456", // User who read the message
		ConversationID: "conv-789",
		ReadAt:         time.Now().Unix(),
	}

	// Push read receipt
	resp, err := pushService.PushReadReceipt(context.Background(), req)
	if err != nil {
		t.Fatalf("PushReadReceipt failed: %v", err)
	}

	// Verify success
	if !resp.Success {
		t.Errorf("Expected success=true, got false. Error: %s", resp.ErrorMessage)
	}

	// Verify both devices received the receipt
	if resp.DeliveredCount != 2 {
		t.Errorf("Expected DeliveredCount=2, got %d", resp.DeliveredCount)
	}

	// Verify no failed devices
	if len(resp.FailedDevices) != 0 {
		t.Errorf("Expected no failed devices, got %v", resp.FailedDevices)
	}

	// Verify message content on device 1
	conn1, _ := gateway.connections.Load("user123_device1")
	connection1 := conn1.(*Connection)
	select {
	case msg := <-connection1.Send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		// Verify message type
		if serverMsg.Type != "read_receipt" {
			t.Errorf("Expected type=read_receipt, got %s", serverMsg.Type)
		}

		// Verify message ID
		if serverMsg.MsgID != "msg-123" {
			t.Errorf("Expected msg_id=msg-123, got %s", serverMsg.MsgID)
		}

		// Verify reader ID
		if serverMsg.ReaderID != "user456" {
			t.Errorf("Expected reader_id=user456, got %s", serverMsg.ReaderID)
		}

		// Verify conversation ID
		if serverMsg.ConversationID != "conv-789" {
			t.Errorf("Expected conversation_id=conv-789, got %s", serverMsg.ConversationID)
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message on device 1")
	}

	// Verify message content on device 2
	conn2, _ := gateway.connections.Load("user123_device2")
	connection2 := conn2.(*Connection)
	select {
	case msg := <-connection2.Send:
		var serverMsg ServerMessage
		if err := json.Unmarshal(msg, &serverMsg); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		// Verify message type
		if serverMsg.Type != "read_receipt" {
			t.Errorf("Expected type=read_receipt, got %s", serverMsg.Type)
		}

	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for message on device 2")
	}
}

// TestReadReceiptOfflineDevice tests read receipt delivery when sender is offline
// Validates: Requirements 15.4 (handle device offline scenarios)
func TestReadReceiptOfflineDevice(t *testing.T) {
	// Create mock registry
	mockRegistry := newMockRegistryClient()

	// Create gateway service with no connections
	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Set up mock registry to return devices (but they're not connected)
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()}, // Remote gateway
	})

	// Create push service
	pushService := NewPushService(gateway)

	// Simulate read receipt event
	req := &PushReadReceiptRequest{
		MsgID:          "msg-123",
		SenderID:       "user123",
		ReaderID:       "user456",
		ConversationID: "conv-789",
		ReadAt:         time.Now().Unix(),
	}

	// Push read receipt
	resp, err := pushService.PushReadReceipt(context.Background(), req)
	if err != nil {
		t.Fatalf("PushReadReceipt failed: %v", err)
	}

	// Verify failure (no devices online locally)
	if resp.Success {
		t.Error("Expected success=false for offline devices")
	}

	// Verify zero delivered count
	if resp.DeliveredCount != 0 {
		t.Errorf("Expected DeliveredCount=0, got %d", resp.DeliveredCount)
	}

	// Verify failed device listed
	if len(resp.FailedDevices) != 1 {
		t.Errorf("Expected 1 failed device, got %d", len(resp.FailedDevices))
	}
}

// TestReadReceiptPartialDeviceOnline tests read receipt when some devices are online
// Validates: Requirements 15.3 (handle partial delivery failures)
func TestReadReceiptPartialDeviceOnline(t *testing.T) {
	// Create mock registry
	mockRegistry := newMockRegistryClient()

	// Create gateway service with one connection
	gateway := &GatewayService{
		registryClient: mockRegistry,
	}
	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      context.Background(),
	})

	// Set up mock registry to return two devices (one local, one remote)
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()}, // Local
		{DeviceID: "device2", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()}, // Remote
	})

	// Create push service
	pushService := NewPushService(gateway)

	// Simulate read receipt event
	req := &PushReadReceiptRequest{
		MsgID:          "msg-123",
		SenderID:       "user123",
		ReaderID:       "user456",
		ConversationID: "conv-789",
		ReadAt:         time.Now().Unix(),
	}

	// Push read receipt
	resp, err := pushService.PushReadReceipt(context.Background(), req)
	if err != nil {
		t.Fatalf("PushReadReceipt failed: %v", err)
	}

	// Verify partial success
	if !resp.Success {
		t.Error("Expected success=true for partial delivery")
	}

	// Verify one device delivered
	if resp.DeliveredCount != 1 {
		t.Errorf("Expected DeliveredCount=1, got %d", resp.DeliveredCount)
	}

	// Verify one failed device
	if len(resp.FailedDevices) != 1 {
		t.Errorf("Expected 1 failed device, got %d", len(resp.FailedDevices))
	}

	if resp.FailedDevices[0] != "device2" {
		t.Errorf("Expected failed device=device2, got %s", resp.FailedDevices[0])
	}
}
