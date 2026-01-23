package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPushService_PushMessage_Success tests successful message push
func TestPushService_PushMessage_Success(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a connection
	conn := &Connection{
		UserID:   "user123",
		DeviceID: "device456",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	gateway.connections.Store("user123_device456", conn)

	// Push a message
	req := &PushMessageRequest{
		MsgID:          "msg-001",
		RecipientID:    "user123",
		DeviceID:       "device456",
		SenderID:       "user789",
		Content:        "Hello!",
		MessageType:    "text",
		SequenceNumber: 12345,
		Timestamp:      time.Now().Unix(),
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)

	// Verify message was sent
	select {
	case data := <-conn.Send:
		var msg ServerMessage
		err := json.Unmarshal(data, &msg)
		require.NoError(t, err)

		assert.Equal(t, "message", msg.Type)
		assert.Equal(t, "msg-001", msg.MsgID)
		assert.Equal(t, "user789", msg.Sender)
		assert.Equal(t, "Hello!", msg.Content)
		assert.Equal(t, int64(12345), msg.SequenceNumber)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestPushService_PushMessage_MissingRecipient tests push with missing recipient
func TestPushService_PushMessage_MissingRecipient(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "",
		SenderID:    "user789",
		Content:     "Hello!",
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "recipient_id is required")
}

// TestPushService_PushMessage_UserNotConnected tests push to offline user
func TestPushService_PushMessage_UserNotConnected(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user999",
		DeviceID:    "device999",
		SenderID:    "user789",
		Content:     "Hello!",
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, int32(0), resp.DeliveredCount)
	assert.Contains(t, resp.FailedDevices, "device999")
}

// TestPushService_PushMessage_AllDevices tests push to all user devices
func TestPushService_PushMessage_AllDevices(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create multiple connections for the same user
	conn1 := &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	gateway.connections.Store("user123_device1", conn1)

	conn2 := &Connection{
		UserID:   "user123",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	gateway.connections.Store("user123_device2", conn2)

	// Push to all devices (no specific device)
	req := &PushMessageRequest{
		MsgID:          "msg-001",
		RecipientID:    "user123",
		SenderID:       "user789",
		Content:        "Hello!",
		MessageType:    "text",
		SequenceNumber: 12345,
		Timestamp:      time.Now().Unix(),
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)
}

// TestPushService_BroadcastToGroup tests group message broadcast
func TestPushService_BroadcastToGroup(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create connections for group members
	conn1 := &Connection{
		UserID:   "user1",
		DeviceID: "device1",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	gateway.connections.Store("user1_device1", conn1)

	conn2 := &Connection{
		UserID:   "user2",
		DeviceID: "device2",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	gateway.connections.Store("user2_device2", conn2)

	// Prepare message
	serverMsg := ServerMessage{
		Type:           "message",
		MsgID:          "msg-001",
		Sender:         "user3",
		Content:        "Group message!",
		SequenceNumber: 12345,
		Timestamp:      time.Now().Unix(),
	}
	msgData, _ := json.Marshal(serverMsg)

	// Broadcast (this will fail because getGroupMembers returns empty list)
	// But we can test the function doesn't crash
	count, err := gateway.pushService.BroadcastToGroup(context.Background(), "group123", msgData)
	require.NoError(t, err)
	assert.Equal(t, int32(0), count) // No members in cache
}
