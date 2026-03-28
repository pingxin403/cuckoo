//go:build integration
// +build integration

package service

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type p0GatewayRPCBridge struct {
	im_gatewaypb.UnimplementedUimUgatewayUserviceServiceServer
	pushMessageFunc     func(ctx context.Context, in *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error)
	pushReadReceiptFunc func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error)
}

func (s *p0GatewayRPCBridge) HealthCheck(ctx context.Context, in *im_gatewaypb.HealthCheckRequest) (*im_gatewaypb.HealthCheckResponse, error) {
	_ = ctx
	_ = in
	return &im_gatewaypb.HealthCheckResponse{Status: "ok"}, nil
}

func (s *p0GatewayRPCBridge) PushMessage(ctx context.Context, in *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error) {
	if s.pushMessageFunc != nil {
		return s.pushMessageFunc(ctx, in)
	}
	return &im_gatewaypb.PushMessageResponse{Success: false, DeliveredCount: 0}, nil
}

func (s *p0GatewayRPCBridge) PushReadReceipt(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
	if s.pushReadReceiptFunc != nil {
		return s.pushReadReceiptFunc(ctx, in)
	}
	return &im_gatewaypb.PushMessageResponse{Success: false, DeliveredCount: 0}, nil
}

func startGatewayBridgeServer(t *testing.T, remotePushService *PushService) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	rpcServer := grpc.NewServer()
	im_gatewaypb.RegisterUimUgatewayUserviceServiceServer(rpcServer, &p0GatewayRPCBridge{
		pushMessageFunc: func(ctx context.Context, in *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error) {
			resp, err := remotePushService.PushMessage(ctx, &PushMessageRequest{
				MsgID:          in.GetMsgId(),
				RecipientID:    in.GetRecipientId(),
				DeviceID:       in.GetDeviceId(),
				SenderID:       in.GetSenderId(),
				Content:        in.GetContent(),
				MessageType:    in.GetMessageType(),
				SequenceNumber: in.GetSequenceNumber(),
				Timestamp:      in.GetTimestamp(),
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

	return lis.Addr().String()
}

func TestP0_CrossGatewayMessageDelivery_Integration(t *testing.T) {
	remoteGateway, _, remoteRegistry, _ := setupTestGateway(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	remoteConn := &Connection{
		UserID:   "recipient-1",
		DeviceID: "device-remote-1",
		Send:     make(chan []byte, 8),
		Gateway:  remoteGateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	remoteGateway.connections.Store("recipient-1_device-remote-1", remoteConn)
	remoteRegistry.SetUserLocations("recipient-1", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	bridgeAddr := startGatewayBridgeServer(t, NewPushService(remoteGateway))

	localGateway, _, localRegistry, _ := setupTestGateway(t)
	localRegistry.SetUserLocations("recipient-1", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})
	localGateway.SetRemoteForwarder(NewGRPCRemoteForwarder(map[string]string{
		"gateway-node-2": bridgeAddr,
	}))

	resp, err := localGateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:          "msg-cross-1",
		RecipientID:    "recipient-1",
		SenderID:       "sender-1",
		Content:        "hello cross gateway",
		MessageType:    "text",
		SequenceNumber: 1001,
		Timestamp:      time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)

	select {
	case raw := <-remoteConn.Send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		assert.Equal(t, "message", msg.Type)
		assert.Equal(t, "msg-cross-1", msg.MsgID)
		assert.Equal(t, "sender-1", msg.Sender)
		assert.Equal(t, "hello cross gateway", msg.Content)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remote message delivery")
	}
}

func TestP0_CrossGatewayReadReceiptDelivery_Integration(t *testing.T) {
	remoteGateway, _, remoteRegistry, _ := setupTestGateway(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	remoteConn := &Connection{
		UserID:   "sender-remote-1",
		DeviceID: "device-remote-1",
		Send:     make(chan []byte, 8),
		Gateway:  remoteGateway,
		ctx:      ctx,
		cancel:   cancel,
	}
	remoteGateway.connections.Store("sender-remote-1_device-remote-1", remoteConn)
	remoteRegistry.SetUserLocations("sender-remote-1", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	bridgeAddr := startGatewayBridgeServer(t, NewPushService(remoteGateway))

	localGateway, _, localRegistry, _ := setupTestGateway(t)
	localRegistry.SetUserLocations("sender-remote-1", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})
	localGateway.SetRemoteForwarder(NewGRPCRemoteForwarder(map[string]string{
		"gateway-node-2": bridgeAddr,
	}))

	resp, err := localGateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-cross-1",
		SenderID:       "sender-remote-1",
		ReaderID:       "reader-1",
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
		assert.Equal(t, "msg-rr-cross-1", msg.MsgID)
		assert.Equal(t, "reader-1", msg.ReaderID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for remote read receipt delivery")
	}
}

func TestP0_AckSuccessAndTimeoutPaths_Integration(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AckTimeout = 60 * time.Millisecond

	gateway.registerPendingAck("msg-ack-success", "user-1", "device-1")
	assert.Equal(t, "pending", gateway.getAckStatus("msg-ack-success", "user-1", "device-1"))

	resolved := gateway.resolveAck("msg-ack-success", "user-1", "device-1")
	require.True(t, resolved)
	assert.Equal(t, "delivered", gateway.getAckStatus("msg-ack-success", "user-1", "device-1"))

	gateway.registerPendingAck("msg-ack-timeout", "user-2", "device-2")
	require.Eventually(t, func() bool {
		return gateway.getAckStatus("msg-ack-timeout", "user-2", "device-2") == "timeout"
	}, time.Second, 10*time.Millisecond)
}

func TestP0_OfflinePathCompatibility_Integration(t *testing.T) {
	gateway, _, registry, _ := setupTestGateway(t)

	registry.SetUserLocations("offline-user-1", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-offline-1",
		ConnectedAt: time.Now().Unix(),
	}})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: false, DeliveredCount: 0}, nil
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-offline-compat-1",
		RecipientID: "offline-user-1",
		SenderID:    "sender-1",
		Content:     "offline path check",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, int32(0), resp.DeliveredCount)
	assert.Contains(t, resp.FailedDevices, "device-offline-1")
}
