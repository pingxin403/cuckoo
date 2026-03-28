package service

import (
	"context"
	"net"
	"testing"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type testGatewayRPCServer struct {
	im_gatewaypb.UnimplementedUimUgatewayUserviceServiceServer
	pushReadReceiptFunc func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error)
}

func (s *testGatewayRPCServer) HealthCheck(ctx context.Context, in *im_gatewaypb.HealthCheckRequest) (*im_gatewaypb.HealthCheckResponse, error) {
	_ = ctx
	_ = in
	return &im_gatewaypb.HealthCheckResponse{Status: "ok"}, nil
}

func (s *testGatewayRPCServer) PushMessage(ctx context.Context, in *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error) {
	_ = ctx
	_ = in
	return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

func (s *testGatewayRPCServer) PushReadReceipt(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
	if s.pushReadReceiptFunc != nil {
		return s.pushReadReceiptFunc(ctx, in)
	}
	return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
}

func TestGRPCRemoteForwarder_ForwardMessage_NoGatewayAddress(t *testing.T) {
	f := NewGRPCRemoteForwarder(map[string]string{})

	resp, err := f.ForwardMessage(context.Background(), "gateway-remote", &PushMessageRequest{MsgID: "msg-1"})
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestGRPCRemoteForwarder_ForwardReadReceipt_RemoteRPC(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	srv := grpc.NewServer()
	im_gatewaypb.RegisterUimUgatewayUserviceServiceServer(srv, &testGatewayRPCServer{
		pushReadReceiptFunc: func(ctx context.Context, in *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
			require.Equal(t, "msg-rr-1", in.GetMsgId())
			require.Equal(t, "sender-1", in.GetSenderId())
			require.Equal(t, "reader-1", in.GetReaderId())
			return &im_gatewaypb.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
		},
	})
	t.Cleanup(func() { srv.Stop() })

	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	f := NewGRPCRemoteForwarder(map[string]string{"gateway-remote": lis.Addr().String()})

	resp, err := f.ForwardReadReceipt(context.Background(), "gateway-remote", &PushReadReceiptRequest{
		MsgID:          "msg-rr-1",
		SenderID:       "sender-1",
		ReaderID:       "reader-1",
		ConversationID: "conv-1",
		ReadAt:         123,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
}
