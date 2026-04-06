package service

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func setupBenchmarkGateway() (*GatewayService, *mockRegistryClient) {
	authClient := &mockAuthClient{}
	registryClient := newMockRegistryClient()
	imClient := &mockIMClient{}

	gateway := NewGatewayService(
		authClient,
		registryClient,
		imClient,
		nil,
		DefaultGatewayConfig(),
	)

	return gateway, registryClient
}

func BenchmarkPushService_PushMessage_LocalSingleDevice(b *testing.B) {
	gateway, _ := setupBenchmarkGateway()
	ctx := context.Background()
	sendCh := make(chan []byte, 2048)
	stopDrain := make(chan struct{})
	defer close(stopDrain)
	go func() {
		for {
			select {
			case <-sendCh:
			case <-stopDrain:
				return
			}
		}
	}()

	gateway.connections.Store("user123_device1", &Connection{
		UserID:   "user123",
		DeviceID: "device1",
		Send:     sendCh,
		ctx:      ctx,
	})

	req := &PushMessageRequest{
		MsgID:       "bench-msg-local-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello",
		Timestamp:   time.Now().Unix(),
	}

	pushService := NewPushService(gateway)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = pushService.PushMessage(context.Background(), req)
	}
}

func BenchmarkPushService_PushMessage_RemoteForwardSingleGateway(b *testing.B) {
	gateway, mockRegistry := setupBenchmarkGateway()
	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
	})

	pushService := NewPushService(gateway)
	pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	req := &PushMessageRequest{
		MsgID:       "bench-msg-remote-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello",
		Timestamp:   time.Now().Unix(),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = pushService.PushMessage(context.Background(), req)
	}
}

func BenchmarkGatewayService_AckRegisterResolve(b *testing.B) {
	gateway, _ := setupBenchmarkGateway()
	gateway.config.AckTimeout = 30 * time.Second

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msgID := fmt.Sprintf("bench-ack-msg-%d", i)
		userID := "user123"
		deviceID := "device123"
		gateway.registerPendingAck(msgID, userID, deviceID)
		_ = gateway.resolveAck(msgID, userID, deviceID)
	}
}
