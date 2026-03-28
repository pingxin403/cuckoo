package main

import (
	"context"
	"testing"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGatewayPushAPI struct {
	pushMessageFunc     func(ctx context.Context, req *service.PushMessageRequest) (*service.PushMessageResponse, error)
	pushReadReceiptFunc func(ctx context.Context, req *service.PushReadReceiptRequest) (*service.PushMessageResponse, error)
}

func (m *mockGatewayPushAPI) PushMessage(ctx context.Context, req *service.PushMessageRequest) (*service.PushMessageResponse, error) {
	if m.pushMessageFunc != nil {
		return m.pushMessageFunc(ctx, req)
	}
	return &service.PushMessageResponse{}, nil
}

func (m *mockGatewayPushAPI) PushReadReceipt(ctx context.Context, req *service.PushReadReceiptRequest) (*service.PushMessageResponse, error) {
	if m.pushReadReceiptFunc != nil {
		return m.pushReadReceiptFunc(ctx, req)
	}
	return &service.PushMessageResponse{}, nil
}

func TestGatewayRPCServer_PushReadReceipt(t *testing.T) {
	s := &gatewayRPCServer{gateway: &mockGatewayPushAPI{pushReadReceiptFunc: func(ctx context.Context, req *service.PushReadReceiptRequest) (*service.PushMessageResponse, error) {
		require.Equal(t, "msg-1", req.MsgID)
		require.Equal(t, "sender-1", req.SenderID)
		require.Equal(t, "reader-1", req.ReaderID)
		require.Equal(t, "conv-1", req.ConversationID)
		require.Equal(t, int64(123), req.ReadAt)
		return &service.PushMessageResponse{Success: true, DeliveredCount: 1}, nil
	}}}

	resp, err := s.PushReadReceipt(context.Background(), &im_gatewaypb.PushReadReceiptRequest{
		MsgId:          "msg-1",
		SenderId:       "sender-1",
		ReaderId:       "reader-1",
		ConversationId: "conv-1",
		ReadAt:         123,
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(1), resp.DeliveredCount)
}
