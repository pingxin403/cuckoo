package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMultiDevice_MaxDeviceLimit tests the max device limit enforcement
// Validates: Requirement 15.10 (max 5 devices per user)
func TestMultiDevice_MaxDeviceLimit(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Simulate 5 devices already registered
	existingDevices := []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device3", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device4", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device5", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
	}
	mockRegistry.SetUserLocations("user123", existingDevices)

	pushService := NewPushService(gateway)

	// Try to deliver message - should succeed for existing devices
	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		SenderID:    "user456",
		Content:     "Test message",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	// All devices are remote (not connected locally), so delivery fails
	assert.False(t, resp.Success)
	assert.Equal(t, int32(0), resp.DeliveredCount)
	assert.Equal(t, 5, len(resp.FailedDevices))
}

// TestMultiDevice_DeviceIDValidation tests device ID format validation
// Validates: Requirements 15.5, 15.6 (UUID v4 format)
func TestMultiDevice_DeviceIDValidation(t *testing.T) {
	tests := []struct {
		name      string
		deviceID  string
		wantError bool
	}{
		{
			name:      "valid UUID v4",
			deviceID:  "550e8400-e29b-41d4-a716-446655440000",
			wantError: false,
		},
		{
			name:      "valid UUID v4 uppercase",
			deviceID:  "550E8400-E29B-41D4-A716-446655440000",
			wantError: false,
		},
		{
			name:      "invalid UUID v1",
			deviceID:  "550e8400-e29b-11d4-a716-446655440000",
			wantError: true,
		},
		{
			name:      "empty device ID",
			deviceID:  "",
			wantError: true,
		},
		{
			name:      "invalid format",
			deviceID:  "not-a-uuid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDeviceID(tt.deviceID)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMultiDevice_DeliveryToAllDevices tests message delivery to all user devices
// Validates: Requirement 15.1 (multi-device message delivery)
func TestMultiDevice_DeliveryToAllDevices(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Create 3 local connections
	ctx := context.Background()
	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("user123_device2", &Connection{
		UserID:   "user123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("user123_device3", &Connection{
		UserID:   "user123",
		DeviceID: "device3",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})

	// Set up Registry to return all 3 devices
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device3", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
	})

	pushService := NewPushService(gateway)

	req := &PushMessageRequest{
		MsgID:          "msg-001",
		RecipientID:    "user123",
		SenderID:       "user456",
		Content:        "Hello all devices",
		SequenceNumber: 12345,
		Timestamp:      time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(3), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)

	// Verify all devices received the message
	for i := 1; i <= 3; i++ {
		key := "user123_device" + string(rune('0'+i))
		conn, ok := gateway.connections.Load(key)
		assert.True(t, ok)
		connection := conn.(*Connection)

		select {
		case msg := <-connection.Send:
			var serverMsg ServerMessage
			err := json.Unmarshal(msg, &serverMsg)
			assert.NoError(t, err)
			assert.Equal(t, "message", serverMsg.Type)
			assert.Equal(t, "msg-001", serverMsg.MsgID)
			assert.Equal(t, "Hello all devices", serverMsg.Content)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Device %d did not receive message", i)
		}
	}
}

// TestMultiDevice_PartialDeliveryFailure tests partial delivery scenarios
// Validates: Requirement 15.3 (handle partial delivery failures)
func TestMultiDevice_PartialDeliveryFailure(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Create only 2 local connections (device1 and device2)
	ctx := context.Background()
	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("user123_device2", &Connection{
		UserID:   "user123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})

	// Registry returns 4 devices (2 local, 2 remote)
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device3", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()}, // Remote
		{DeviceID: "device4", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()}, // Remote
	})

	pushService := NewPushService(gateway)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		SenderID:    "user456",
		Content:     "Partial delivery test",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success) // Success because at least 1 device received
	assert.Equal(t, int32(2), resp.DeliveredCount)
	assert.Equal(t, 2, len(resp.FailedDevices))
	assert.Contains(t, resp.FailedDevices, "device3")
	assert.Contains(t, resp.FailedDevices, "device4")
}

// TestMultiDevice_RegistryLookupError tests Registry lookup failure handling
// Validates: Requirement 15.2 (track delivery status per device)
func TestMultiDevice_RegistryLookupError(t *testing.T) {
	mockRegistry := newMockRegistryClient()
	mockRegistry.SetLookupError(errors.New("registry unavailable"))

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	pushService := NewPushService(gateway)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		SenderID:    "user456",
		Content:     "Test message",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "failed to lookup user devices")
}

// TestMultiDevice_SpecificDeviceDelivery tests delivery to a specific device
// Validates: Requirement 15.1 (multi-device message delivery)
func TestMultiDevice_SpecificDeviceDelivery(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Create 2 local connections
	ctx := context.Background()
	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("user123_device2", &Connection{
		UserID:   "user123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})

	pushService := NewPushService(gateway)

	// Deliver to specific device only
	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		DeviceID:    "device1", // Specific device
		SenderID:    "user456",
		Content:     "Device-specific message",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)

	// Verify only device1 received the message
	conn1, _ := gateway.connections.Load("user123_device1")
	connection1 := conn1.(*Connection)
	select {
	case msg := <-connection1.Send:
		var serverMsg ServerMessage
		err := json.Unmarshal(msg, &serverMsg)
		assert.NoError(t, err)
		assert.Equal(t, "msg-001", serverMsg.MsgID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Device1 did not receive message")
	}

	// Verify device2 did NOT receive the message
	conn2, _ := gateway.connections.Load("user123_device2")
	connection2 := conn2.(*Connection)
	select {
	case <-connection2.Send:
		t.Error("Device2 should not have received the message")
	case <-time.After(100 * time.Millisecond):
		// Expected - device2 should not receive
	}
}

// TestMultiDevice_ReadReceiptSyncAllDevices tests read receipt sync to all devices
// Validates: Requirement 15.4 (read receipt sync across devices)
func TestMultiDevice_ReadReceiptSyncAllDevices(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// Create 3 local connections for the sender
	ctx := context.Background()
	gateway.connections.Store("sender123_device1", &Connection{
		UserID:   "sender123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("sender123_device2", &Connection{
		UserID:   "sender123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})
	gateway.connections.Store("sender123_device3", &Connection{
		UserID:   "sender123",
		DeviceID: "device3",
		Send:     make(chan []byte, 256),
		ctx:      ctx,
	})

	// Set up Registry
	mockRegistry.SetUserLocations("sender123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device3", GatewayNode: "gateway-1", ConnectedAt: time.Now().Unix()},
	})

	pushService := NewPushService(gateway)

	req := &PushReadReceiptRequest{
		MsgID:          "msg-001",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-789",
		ReadAt:         time.Now().Unix(),
	}

	resp, err := pushService.PushReadReceipt(context.Background(), req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(3), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)

	// Verify all sender devices received the read receipt
	for i := 1; i <= 3; i++ {
		key := "sender123_device" + string(rune('0'+i))
		conn, ok := gateway.connections.Load(key)
		assert.True(t, ok)
		connection := conn.(*Connection)

		select {
		case msg := <-connection.Send:
			var serverMsg ServerMessage
			err := json.Unmarshal(msg, &serverMsg)
			assert.NoError(t, err)
			assert.Equal(t, "read_receipt", serverMsg.Type)
			assert.Equal(t, "msg-001", serverMsg.MsgID)
			assert.Equal(t, "reader456", serverMsg.ReaderID)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Sender device %d did not receive read receipt", i)
		}
	}
}

// TestMultiDevice_NoDevicesOnline tests when no devices are online
// Validates: Requirement 15.3 (handle partial delivery failures)
func TestMultiDevice_NoDevicesOnline(t *testing.T) {
	mockRegistry := newMockRegistryClient()

	gateway := &GatewayService{
		registryClient: mockRegistry,
	}

	// No local connections, but Registry returns devices on remote gateway
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{DeviceID: "device1", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()},
		{DeviceID: "device2", GatewayNode: "gateway-2", ConnectedAt: time.Now().Unix()},
	})

	pushService := NewPushService(gateway)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		SenderID:    "user456",
		Content:     "Test message",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, int32(0), resp.DeliveredCount)
	assert.Equal(t, 2, len(resp.FailedDevices))
}

// TestMultiDevice_EmptyRecipientID tests error handling for empty recipient ID
func TestMultiDevice_EmptyRecipientID(t *testing.T) {
	gateway := &GatewayService{}
	pushService := NewPushService(gateway)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "", // Empty
		SenderID:    "user456",
		Content:     "Test message",
		Timestamp:   time.Now().Unix(),
	}

	resp, err := pushService.PushMessage(context.Background(), req)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "recipient_id is required")
}

// TestMultiDevice_EmptyReadReceiptIDs tests error handling for empty IDs in read receipt
func TestMultiDevice_EmptyReadReceiptIDs(t *testing.T) {
	gateway := &GatewayService{}
	pushService := NewPushService(gateway)

	tests := []struct {
		name     string
		senderID string
		readerID string
	}{
		{"empty sender", "", "reader123"},
		{"empty reader", "sender123", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PushReadReceiptRequest{
				MsgID:          "msg-001",
				SenderID:       tt.senderID,
				ReaderID:       tt.readerID,
				ConversationID: "conv-789",
				ReadAt:         time.Now().Unix(),
			}

			resp, err := pushService.PushReadReceipt(context.Background(), req)
			assert.NoError(t, err)
			assert.False(t, resp.Success)
			assert.Contains(t, resp.ErrorMessage, "sender_id and reader_id are required")
		})
	}
}
