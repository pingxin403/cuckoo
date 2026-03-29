package service

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testGatewayServiceServer struct {
	gatewaypb.UnimplementedUimUgatewayUserviceServiceServer
	pushMessageFunc func(ctx context.Context, req *gatewaypb.PushMessageRequest) (*gatewaypb.PushMessageResponse, error)
}

func (s *testGatewayServiceServer) HealthCheck(ctx context.Context, req *gatewaypb.HealthCheckRequest) (*gatewaypb.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &gatewaypb.HealthCheckResponse{Status: "ok"}, nil
}

func (s *testGatewayServiceServer) PushMessage(ctx context.Context, req *gatewaypb.PushMessageRequest) (*gatewaypb.PushMessageResponse, error) {
	if s.pushMessageFunc != nil {
		return s.pushMessageFunc(ctx, req)
	}
	return &gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

func (s *testGatewayServiceServer) PushReadReceipt(ctx context.Context, req *gatewaypb.PushReadReceiptRequest) (*gatewaypb.PushMessageResponse, error) {
	_ = ctx
	_ = req
	return &gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

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

func TestGatewayClient_PushMessage_RetryOnUnavailable(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	var attempts int32
	srv := grpc.NewServer()
	gatewaypb.RegisterUimUgatewayUserviceServiceServer(srv, &testGatewayServiceServer{
		pushMessageFunc: func(ctx context.Context, req *gatewaypb.PushMessageRequest) (*gatewaypb.PushMessageResponse, error) {
			_ = ctx
			_ = req
			if atomic.AddInt32(&attempts, 1) == 1 {
				return nil, status.Error(codes.Unavailable, "temporary unavailable")
			}
			return &gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})
	t.Cleanup(func() { srv.Stop() })

	go func() {
		_ = srv.Serve(lis)
	}()

	client := NewGatewayClient(1*time.Second, 1*time.Second)
	client.retryBackoff = 5 * time.Millisecond
	defer func() { _ = client.Close() }()

	req := &GatewayPushRequest{
		MsgID:          "msg-retry-001",
		RecipientID:    "user123",
		DeviceID:       "device-abc",
		SenderID:       "user456",
		Content:        "Hello",
		MessageType:    "text",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	resp, err := client.PushMessage(context.Background(), lis.Addr().String(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
}

func TestGatewayClient_PushMessage_CircuitBreakerOpenAndRecover(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	var attempts int32
	var failing atomic.Bool
	failing.Store(true)

	srv := grpc.NewServer()
	gatewaypb.RegisterUimUgatewayUserviceServiceServer(srv, &testGatewayServiceServer{
		pushMessageFunc: func(ctx context.Context, req *gatewaypb.PushMessageRequest) (*gatewaypb.PushMessageResponse, error) {
			_ = ctx
			_ = req
			atomic.AddInt32(&attempts, 1)
			if failing.Load() {
				return nil, status.Error(codes.Unavailable, "dependency unavailable")
			}
			return &gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})
	t.Cleanup(func() { srv.Stop() })

	go func() {
		_ = srv.Serve(lis)
	}()

	client := NewGatewayClient(1*time.Second, 200*time.Millisecond)
	client.retryBackoff = 5 * time.Millisecond
	client.maxRetries = 0
	client.breakerFailureThreshold = 1
	client.breakerOpenTimeout = 80 * time.Millisecond
	defer func() { _ = client.Close() }()

	req := &GatewayPushRequest{
		MsgID:          "msg-breaker-001",
		RecipientID:    "user123",
		DeviceID:       "device-abc",
		SenderID:       "user456",
		Content:        "Hello",
		MessageType:    "text",
		SequenceNumber: 1,
		Timestamp:      time.Now().UnixMilli(),
	}

	_, err = client.PushMessage(context.Background(), lis.Addr().String(), req)
	require.Error(t, err)
	firstAttempts := atomic.LoadInt32(&attempts)
	require.Equal(t, int32(1), firstAttempts)

	_, err = client.PushMessage(context.Background(), lis.Addr().String(), req)
	require.Error(t, err)
	assert.Equal(t, firstAttempts, atomic.LoadInt32(&attempts))

	time.Sleep(100 * time.Millisecond)
	failing.Store(false)

	resp, err := client.PushMessage(context.Background(), lis.Addr().String(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
}
