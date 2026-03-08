package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatewayClient(t *testing.T) {
	client := NewGatewayClient(5*time.Second, 3*time.Second)

	assert.NotNil(t, client)
	assert.NotNil(t, client.connPool)
	assert.Equal(t, 5*time.Second, client.dialTimeout)
	assert.Equal(t, 3*time.Second, client.requestTimeout)
}

func TestGatewayClient_Close(t *testing.T) {
	client := NewGatewayClient(5*time.Second, 3*time.Second)

	err := client.Close()
	require.NoError(t, err)

	// Verify connection pool is empty
	assert.Equal(t, 0, len(client.connPool))
}

func TestGatewayClient_PushMessage_InvalidAddress(t *testing.T) {
	client := NewGatewayClient(1*time.Second, 1*time.Second)
	defer func() { _ = client.Close() }()

	req := &GatewayPushRequest{
		MsgID:       "msg-001",
		RecipientID: "user123",
		DeviceID:    "device-abc",
		SenderID:    "user456",
		Content:     "Hello",
		MessageType: "text",
		Timestamp:   time.Now().UnixMilli(),
	}

	ctx := context.Background()
	resp, err := client.PushMessage(ctx, "invalid-address:9999", req)

	// Should return error for invalid address
	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.ErrorMessage, "failed to connect to gateway")
}
