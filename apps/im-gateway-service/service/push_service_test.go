package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/pingxin403/cuckoo/libs/observability/tracing"
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

type captureCrossGatewayMetrics struct {
	mu             sync.Mutex
	successKinds   []string
	failureRecords []string
	latencyKinds   []string
}

type capturePushSpan struct {
	name       string
	attributes map[string]interface{}
	statusCode tracing.StatusCode
	errorCount int
}

func (s *capturePushSpan) End() {}

func (s *capturePushSpan) SetAttribute(key string, value interface{}) {
	s.attributes[key] = value
}

func (s *capturePushSpan) SetAttributes(attributes map[string]interface{}) {
	for k, v := range attributes {
		s.attributes[k] = v
	}
}

func (s *capturePushSpan) RecordError(err error) {
	if err != nil {
		s.errorCount++
	}
}

func (s *capturePushSpan) SetStatus(code tracing.StatusCode, description string) {
	_ = description
	s.statusCode = code
}

type capturePushTracer struct {
	mu    sync.Mutex
	spans []*capturePushSpan
}

func (t *capturePushTracer) StartSpan(ctx context.Context, name string, opts ...tracing.SpanOption) (context.Context, tracing.Span) {
	cfg := &tracing.SpanConfig{Attributes: make(map[string]interface{})}
	for _, opt := range opts {
		opt(cfg)
	}
	span := &capturePushSpan{name: name, attributes: cfg.Attributes}
	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()
	return ctx, span
}

func (t *capturePushTracer) Shutdown(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *captureCrossGatewayMetrics) IncForwardSuccess(kind string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successKinds = append(m.successKinds, kind)
}

func (m *captureCrossGatewayMetrics) IncForwardFailure(kind, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureRecords = append(m.failureRecords, kind+":"+reason)
}

func (m *captureCrossGatewayMetrics) ObserveForwardLatency(kind string, duration time.Duration) {
	_ = duration
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencyKinds = append(m.latencyKinds, kind)
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

func TestPushService_PushMessage_SpecificDeviceRemoteGatewayForwarding(t *testing.T) {
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
		MsgID:       "msg-remote-specific-1",
		RecipientID: "user123",
		DeviceID:    "device-remote-1",
		SenderID:    "user789",
		Content:     "Hello remote specific device",
		Timestamp:   time.Now().Unix(),
	})

	require.NoError(t, err)
	require.True(t, called)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_PushMessage_RemoteGatewayDeliveredCountPreserved(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
	})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: true, DeliveredCount: 2}, nil
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-remote-count-1",
		RecipientID: "user123",
		SenderID:    "user789",
		Content:     "Hello remote count",
		Timestamp:   time.Now().Unix(),
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
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

func TestPushService_RemoteForwardMetrics_MessageSuccessAndFailure(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)
	metrics := &captureCrossGatewayMetrics{}
	gateway.pushService.SetMetrics(metrics)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-metric-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "ok",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	require.True(t, resp.Success)

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return nil, errors.New("temporary unavailable")
		},
	})

	resp, err = gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-metric-2",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "fail",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	require.False(t, resp.Success)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	assert.Contains(t, metrics.successKinds, "message")
	assert.Contains(t, metrics.latencyKinds, "message")
	assert.NotEmpty(t, metrics.failureRecords)
	assert.Contains(t, metrics.failureRecords[0], "message:")
}

func TestPushService_RemoteForwardMetrics_ReadReceiptFailureReason(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)
	metrics := &captureCrossGatewayMetrics{}
	gateway.pushService.SetMetrics(metrics)

	mockRegistry.SetUserLocations("sender123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardReadReceiptFunc: func(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: false, DeliveredCount: 0}, nil
		},
	})

	resp, err := gateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-metric-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err)
	require.False(t, resp.Success)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	assert.Contains(t, metrics.latencyKinds, "read_receipt")
	assert.NotEmpty(t, metrics.failureRecords)
	assert.Contains(t, metrics.failureRecords[0], "read_receipt:")
}

func TestPushService_PushReadReceipt_RemoteGatewayDeliveredCountPreserved(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("sender123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardReadReceiptFunc: func(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: true, DeliveredCount: 2}, nil
		},
	})

	resp, err := gateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-count-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_PushReadReceipt_RemoteGatewaySameNode_ForwardOnce(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("sender123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-2", ConnectedAt: time.Now().Unix()},
	})

	callCount := 0
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardReadReceiptFunc: func(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
			callCount++
			return &PushMessageResponse{Success: true, DeliveredCount: 2}, nil
		},
	})

	resp, err := gateway.pushService.PushReadReceipt(context.Background(), &PushReadReceiptRequest{
		MsgID:          "msg-rr-same-node-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
	assert.Equal(t, 1, callCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_Tracing_RemoteForwardMessageFailureAndSuccess(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)
	tracer := &capturePushTracer{}
	gateway.pushService.SetTracer(tracer)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{{
		GatewayNode: "gateway-node-2",
		DeviceID:    "device-remote-1",
		ConnectedAt: time.Now().Unix(),
	}})

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return nil, errors.New("deadline exceeded")
		},
	})

	respFail, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-trace-fail-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "fail",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.False(t, respFail.Success)

	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	respSucc, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-trace-succ-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "ok",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, respSucc.Success)

	tracer.mu.Lock()
	defer tracer.mu.Unlock()
	require.Len(t, tracer.spans, 2)

	assert.Equal(t, "im-gateway.forward.message", tracer.spans[0].name)
	assert.Equal(t, "failure", tracer.spans[0].attributes["forward.result"])
	assert.Equal(t, "timeout", tracer.spans[0].attributes["forward.failure_reason"])
	assert.Equal(t, tracing.StatusCodeError, tracer.spans[0].statusCode)

	assert.Equal(t, "im-gateway.forward.message", tracer.spans[1].name)
	assert.Equal(t, "success", tracer.spans[1].attributes["forward.result"])
	assert.Equal(t, int32(1), tracer.spans[1].attributes["forward.delivered_count"])
	assert.Equal(t, tracing.StatusCodeOK, tracer.spans[1].statusCode)
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

func TestPushService_BroadcastToGroup_DeduplicatesDuplicateMembers(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.cacheManager = nil
	gateway.SetGroupMemberProvider(&mockGatewayGroupMemberProvider{
		getGroupMembersFunc: func(ctx context.Context, groupID string) ([]string, error) {
			return []string{"user123", "user123"}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn1 := &Connection{UserID: "user123", DeviceID: "device1", Send: make(chan []byte, 8), Gateway: gateway, ctx: ctx, cancel: cancel}
	conn2 := &Connection{UserID: "user123", DeviceID: "device2", Send: make(chan []byte, 8), Gateway: gateway, ctx: ctx, cancel: cancel}
	connOther := &Connection{UserID: "user456", DeviceID: "device1", Send: make(chan []byte, 8), Gateway: gateway, ctx: ctx, cancel: cancel}

	gateway.connections.Store("user123_device1", conn1)
	gateway.connections.Store("user123_device2", conn2)
	gateway.connections.Store("user456_device1", connOther)

	payload := []byte("group-message")
	delivered, err := gateway.pushService.BroadcastToGroup(context.Background(), "group_123", payload)
	require.NoError(t, err)

	assert.Equal(t, int32(2), delivered)
	assert.Equal(t, 1, len(conn1.Send))
	assert.Equal(t, 1, len(conn2.Send))
	assert.Equal(t, 0, len(connOther.Send))
}

func TestPushService_BroadcastToGroup_SkipsBlockedConnectionAndDeliversOthers(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.cacheManager = nil
	gateway.SetGroupMemberProvider(&mockGatewayGroupMemberProvider{
		getGroupMembersFunc: func(ctx context.Context, groupID string) ([]string, error) {
			return []string{"user123"}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blockedConn := &Connection{UserID: "user123", DeviceID: "blocked", Send: make(chan []byte, 1), Gateway: gateway, ctx: ctx, cancel: cancel}
	fastConn := &Connection{UserID: "user123", DeviceID: "fast", Send: make(chan []byte, 1), Gateway: gateway, ctx: ctx, cancel: cancel}
	blockedConn.Send <- []byte("already-full")

	gateway.connections.Store("user123_blocked", blockedConn)
	gateway.connections.Store("user123_fast", fastConn)

	delivered, err := gateway.pushService.BroadcastToGroup(context.Background(), "group_123", []byte("group-message"))
	require.NoError(t, err)
	assert.Equal(t, int32(1), delivered)
	assert.Equal(t, 1, len(fastConn.Send))
	assert.Equal(t, 1, len(blockedConn.Send))
}

func TestPushService_BroadcastToGroup_DeduplicatesSameDeviceAcrossAliasKeys(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.cacheManager = nil
	gateway.SetGroupMemberProvider(&mockGatewayGroupMemberProvider{
		getGroupMembersFunc: func(ctx context.Context, groupID string) ([]string, error) {
			return []string{"user123"}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &Connection{UserID: "user123", DeviceID: "device1", Send: make(chan []byte, 8), Gateway: gateway, ctx: ctx, cancel: cancel}
	gateway.connections.Store("user123_device1", conn)
	gateway.connections.Store("user123_device1_alias", conn)

	delivered, err := gateway.pushService.BroadcastToGroup(context.Background(), "group_123", []byte("group-message"))
	require.NoError(t, err)
	assert.Equal(t, int32(1), delivered)
	assert.Equal(t, 1, len(conn.Send))
}

func TestPushService_PushMessage_RemoteGatewaySameNode_ForwardOnce(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-2", ConnectedAt: time.Now().Unix()},
	})

	callCount := 0
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			callCount++
			return &PushMessageResponse{Success: true, DeliveredCount: 2}, nil
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-forward-same-node-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(2), resp.DeliveredCount)
	assert.Equal(t, 1, callCount)
	assert.Empty(t, resp.FailedDevices)
}

func TestPushService_PushMessage_RemoteGatewaySameNode_FailureForwardOnce(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-2", ConnectedAt: time.Now().Unix()},
	})

	callCount := 0
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			callCount++
			return nil, errors.New("temporary unavailable")
		},
	})

	resp, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-forward-same-node-fail-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, int32(0), resp.DeliveredCount)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, []string{"device-remote-1"}, resp.FailedDevices)
}

func TestPushService_PushMessage_RemoteGatewayTransientRecovery(t *testing.T) {
	gateway, _, mockRegistry, _ := setupTestGateway(t)

	mockRegistry.SetUserLocations("user123", []GatewayLocation{
		{GatewayNode: "gateway-node-2", DeviceID: "device-remote-1", ConnectedAt: time.Now().Unix()},
	})

	callCount := 0
	gateway.pushService.SetRemoteForwarder(&mockRemoteForwarder{
		forwardMessageFunc: func(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
			callCount++
			if callCount == 1 {
				return nil, errors.New("temporary unavailable")
			}
			return &PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	first, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-transient-1",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.False(t, first.Success)
	assert.Equal(t, int32(0), first.DeliveredCount)

	second, err := gateway.pushService.PushMessage(context.Background(), &PushMessageRequest{
		MsgID:       "msg-transient-2",
		RecipientID: "user123",
		SenderID:    "sender1",
		Content:     "hello again",
		Timestamp:   time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.True(t, second.Success)
	assert.Equal(t, int32(1), second.DeliveredCount)
}
