package service

import (
"context"
"errors"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestPushService_PushMessage_MultiDevice(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
{GatewayNode: "gateway-node-1", DeviceID: "device1", ConnectedAt: time.Now().Unix()},
		{GatewayNode: "gateway-node-1", DeviceID: "device2", ConnectedAt: time.Now().Unix()},
	})

	req := &PushMessageRequest{
		MsgID:          "msg-001",
		RecipientID:    "user123",
		SenderID:       "user789",
		Content:        "Hello!",
		SequenceNumber: 12345,
		Timestamp:      time.Now().Unix(),
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
}

func TestPushService_PushMessage_RegistryFailure(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetLookupError(errors.New("registry unavailable"))

	req := &PushMessageRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		Content:     "Test",
	}

	resp, err := gateway.pushService.PushMessage(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "failed to lookup user devices")
}
