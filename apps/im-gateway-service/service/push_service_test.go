package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type mockRemoteForwarder struct {
	forwardMessageFunc     func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error)
	forwardReadReceiptFunc func(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error)
}

type mockGatewayGroupMemberProvider struct {
	getGroupMembersFunc func(ctx context.Context, groupID string) ([]string, error)
}

func (m *mockGatewayGroupMemberProvider) GetGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	if m.getGroupMembersFunc != nil {
		return m.getGroupMembersFunc(ctx, groupID)
	}
	return []string{}, nil
}

func (m *mockRemoteForwarder) ForwardMessage(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
	if m.forwardMessageFunc != nil {
		return m.forwardMessageFunc(ctx, gatewayNode, req)
	}
	return &PushMessageResponse{Success: false}, nil
}

func (m *mockRemoteForwarder) ForwardReadReceipt(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	if m.forwardReadReceiptFunc != nil {
		return m.forwardReadReceiptFunc(ctx, gatewayNode, req)
	}
	return &PushMessageResponse{Success: false}, nil
}

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

func TestPushService_PushMessage_RemoteGatewayForwarding(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
	})

	called := false
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			called = true
			assert.Equal(t, "gateway-node-2", gatewayNode)
			assert.Equal(t, "device-remote-1", req.DeviceID)
			assert.Equal(t, "user123", req.RecipientID)
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-remote-1",
		RecipientID: "user123",
		SenderID:    "user789",
		Content:     "Hello remote",
		Timestamp:   time.Now().Unix(),
	})

	require.NoError(t, err)
	require.True(t, called)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_PushReadReceipt_RemoteGatewayForwarding(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("sender123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
	})

	called := false
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardReadReceiptFunc: func(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
			called = true
			assert.Equal(t, "gateway-node-2", gatewayNode)
			assert.Equal(t, "sender123", req.SenderID)
			assert.Equal(t, "reader456", req.ReaderID)
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	resp, err := gateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})

	require.NoError(t, err)
	require.True(t, called)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_PushReadReceipt_RemoteGatewayForwarding_EndToEnd(t *testing.T) {
	remoteRegistry := newMockRegistryClient()
	remoteGateway := NewGatewayService(nil, remoteRegistry, nil, nil, DefaultGatewayConfig())

	remoteConn := &Connection{
		UserID:   "sender123",
		DeviceID: "device-remote-1",
		Send:     make(chan []byte, 8),
		Gateway:  remoteGateway,
		ctx:      context.Background(),
		cancel:   func() {},
	}
	remoteGateway.connections.Store("sender123_device-remote-1", remoteConn)
	remoteRegistry.SetUserLocations("sender123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	remotePushService := NewPushService(remoteGateway)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	rpcServer := grpc.NewServer()
	im_gatewaypb.RegisterUimUgatewayUserviceServiceServer(rpcServer, &testGatewayRPCServer{
		pushReadReceiptFunc: func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
			resp, err := remotePushService.PushReadReceipt(ctx, &PushReadReceiptRequest{
				MsgID:          in.GetMsgId(),
				SenderID:       in.GetSenderId(),
				ReaderID:       in.GetReaderId(),
				ConversationID: in.GetConversationId(),
				ReadAt:         in.GetReadAt(),
			})
			if err != nil {
				return nil, err
			}
			return &im_gatewaypb.PushMessageResponse{
				Success:        resp.Success,
				DeliveredCount: resp.DeliveredCount,
				FailedDevices:  resp.FailedDevices,
				ErrorMessage:   resp.ErrorMessage,
			}, nil
		},
	})
	t.Cleanup(func() { rpcServer.Stop() })

	go func() {
		_ = rpcServer.Serve(lis)
	}()

	localGateway, _, localRegistry, _ := setupTestGateway(t)
	localRegistry.SetUserLocations("sender123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})
	localGateway.SetRemoteForwarder(NewGRPCRemoteForwarder(map[string]string{
		"gateway-node-2": lis.Addr().String(),
	}))

	resp, err := localGateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-e2e-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)

	select {
	case raw := <-remoteConn.Send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		assert.Equal(t, "read_receipt", msg.Type)
		assert.Equal(t, "msg-rr-e2e-1", msg.MsgID)
		assert.Equal(t, "reader456", msg.ReaderID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remote read receipt delivery")
	}
}

func TestGatewayService_GetGroupMembers_FallbackProviderOnCacheMiss(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.cacheManager = nil

	called := false
	gateway.SetGroupMemberProvider(&mockGatewayGroupMemberProvider{
		getGroupMembersFunc: func(ctx context.Context, groupID string) ([]string, error) {
			called = true
			assert.Equal(t, "group_123", groupID)
			return []string{"user123", "user456"}, nil
		},
	})

	members, err := gateway.getGroupMembers(context.Background(), "group_123")
	require.NoError(t, err)
	require.True(t, called)
	assert.Equal(t, []string{"user123", "user456"}, members)
}

func TestGatewayService_GetGroupMembers_FallbackProviderError(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.cacheManager = nil

	gateway.SetGroupMemberProvider(&mockGatewayGroupMemberProvider{
		getGroupMembersFunc: func(ctx context.Context, groupID string) ([]string, error) {
			return nil, errors.New("provider unavailable")
		},
	})

	members, err := gateway.getGroupMembers(context.Background(), "group_123")
	require.Error(t, err)
	assert.Nil(t, members)
	assert.Contains(t, err.Error(), "provider unavailable")
}
