//go:build integration
// +build integration

package service

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type p1DependencyRPCServer struct {
	im_gatewaypb.UnimplementedUimUgatewayUserviceServiceServer
	pushReadReceiptFunc func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error)
}

func (s *p1DependencyRPCServer) HealthCheck(ctx context.Context, in *im_gatewaypb.HealthCheckRequest) (*im_gatewaypb.HealthCheckResponse, error) {
	_ = ctx
	_ = in
	return &im_gatewaypb.HealthCheckResponse{Status: "ok"}, nil
}

func (s *p1DependencyRPCServer) PushMessage(ctx context.Context, in *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error) {
	_ = ctx
	_ = in
	return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

func (s *p1DependencyRPCServer) PushReadReceipt(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
	if s.pushReadReceiptFunc != nil {
		return s.pushReadReceiptFunc(ctx, in)
	}
	return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

func startP1DependencyRPCServer(t *testing.T, server *p1DependencyRPCServer) string {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	grpcServer := grpc.NewServer()
	im_gatewaypb.RegisterUimUgatewayUserviceServiceServer(grpcServer, server)
	t.Cleanup(func() { grpcServer.Stop() })

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	return lis.Addr().String()
}

func TestP1_DependencyJitterRetry_Integration(t *testing.T) {
	var attempts int32
	addr := startP1DependencyRPCServer(t, &p1DependencyRPCServer{
		pushReadReceiptFunc: func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
			_ = ctx
			_ = in
			current := atomic.AddInt32(&attempts, 1)
			if current <= 2 {
				return nil, status.Error(codes.Unavailable, "transient jitter")
			}
			return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	forwarder := NewGRPCRemoteForwarder(map[string]string{"gateway-node-jitter": addr})
	forwarder.maxRetries = 3
	forwarder.retryBackoff = 5 * time.Millisecond
	forwarder.requestTimeout = 200 * time.Millisecond
	t.Cleanup(func() { _ = forwarder.Close() })

	resp, err := forwarder.ForwardReadReceipt(context.Background(), "gateway-node-jitter", &PushReadReceiptRequest{
		MsgID:          "msg-p1-jitter-1",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
}

func TestP1_DependencyTimeout_Integration(t *testing.T) {
	var attempts int32
	addr := startP1DependencyRPCServer(t, &p1DependencyRPCServer{
		pushReadReceiptFunc: func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
			_ = in
			atomic.AddInt32(&attempts, 1)
			select {
			case <-time.After(120 * time.Millisecond):
				return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	forwarder := NewGRPCRemoteForwarder(map[string]string{"gateway-node-timeout": addr})
	forwarder.maxRetries = 1
	forwarder.retryBackoff = 5 * time.Millisecond
	forwarder.requestTimeout = 30 * time.Millisecond
	t.Cleanup(func() { _ = forwarder.Close() })

	start := time.Now()
	resp, err := forwarder.ForwardReadReceipt(context.Background(), "gateway-node-timeout", &PushReadReceiptRequest{
		MsgID:          "msg-p1-timeout-1",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
	assert.GreaterOrEqual(t, elapsed, 60*time.Millisecond)
}

func TestP1_FaultInjectionAndRecovery_Integration(t *testing.T) {
	var attempts int32
	var failing atomic.Bool
	failing.Store(true)

	addr := startP1DependencyRPCServer(t, &p1DependencyRPCServer{
		pushReadReceiptFunc: func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
			_ = ctx
			_ = in
			atomic.AddInt32(&attempts, 1)
			if failing.Load() {
				return nil, status.Error(codes.Unavailable, "injected fault")
			}
			return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})

	forwarder := NewGRPCRemoteForwarder(map[string]string{"gateway-node-fault": addr})
	forwarder.maxRetries = 0
	forwarder.retryBackoff = 5 * time.Millisecond
	forwarder.breakerFailureThreshold = 1
	forwarder.breakerOpenTimeout = 80 * time.Millisecond
	t.Cleanup(func() { _ = forwarder.Close() })

	resp1, err1 := forwarder.ForwardReadReceipt(context.Background(), "gateway-node-fault", &PushReadReceiptRequest{
		MsgID:          "msg-p1-fault-1",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.Error(t, err1)
	assert.Nil(t, resp1)
	firstAttempts := atomic.LoadInt32(&attempts)
	assert.Equal(t, int32(1), firstAttempts)

	resp2, err2 := forwarder.ForwardReadReceipt(context.Background(), "gateway-node-fault", &PushReadReceiptRequest{
		MsgID:          "msg-p1-fault-2",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.Error(t, err2)
	assert.Nil(t, resp2)
	assert.Equal(t, firstAttempts, atomic.LoadInt32(&attempts))

	time.Sleep(100 * time.Millisecond)
	failing.Store(false)

	resp3, err3 := forwarder.ForwardReadReceipt(context.Background(), "gateway-node-fault", &PushReadReceiptRequest{
		MsgID:          "msg-p1-fault-3",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})
	require.NoError(t, err3)
	require.NotNil(t, resp3)
	assert.True(t, resp3.Success)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
}
