package main

import (
	"context"
	"testing"
	"time"

	authpb "github.com/pingxin403/cuckoo/api/gen/go/authpb"
	impb "github.com/pingxin403/cuckoo/api/gen/go/impb"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/config"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLoadConfig_IncludesSecurityPolicy(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.NotNil(t, cfg.Security.AllowedOrigins)
	assert.True(t, cfg.Security.AllowEmptyOrigin)
}

type mockAuthPBClient struct {
	validateFunc func(ctx context.Context, in *authpb.ValidateTokenRequest, opts ...grpc.CallOption) (*authpb.ValidateTokenResponse, error)
}

func (m *mockAuthPBClient) ValidateToken(ctx context.Context, in *authpb.ValidateTokenRequest, opts ...grpc.CallOption) (*authpb.ValidateTokenResponse, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, in, opts...)
	}
	return &authpb.ValidateTokenResponse{Valid: false}, nil
}

func (m *mockAuthPBClient) RefreshToken(ctx context.Context, in *authpb.RefreshTokenRequest, opts ...grpc.CallOption) (*authpb.RefreshTokenResponse, error) {
	return &authpb.RefreshTokenResponse{}, nil
}

type mockIMPBClient struct {
	routePrivateFunc func(ctx context.Context, in *impb.RoutePrivateMessageRequest, opts ...grpc.CallOption) (*impb.RoutePrivateMessageResponse, error)
	routeGroupFunc   func(ctx context.Context, in *impb.RouteGroupMessageRequest, opts ...grpc.CallOption) (*impb.RouteGroupMessageResponse, error)
}

func (m *mockIMPBClient) RoutePrivateMessage(ctx context.Context, in *impb.RoutePrivateMessageRequest, opts ...grpc.CallOption) (*impb.RoutePrivateMessageResponse, error) {
	if m.routePrivateFunc != nil {
		return m.routePrivateFunc(ctx, in, opts...)
	}
	return &impb.RoutePrivateMessageResponse{}, nil
}

func (m *mockIMPBClient) RouteGroupMessage(ctx context.Context, in *impb.RouteGroupMessageRequest, opts ...grpc.CallOption) (*impb.RouteGroupMessageResponse, error) {
	if m.routeGroupFunc != nil {
		return m.routeGroupFunc(ctx, in, opts...)
	}
	return &impb.RouteGroupMessageResponse{}, nil
}

func (m *mockIMPBClient) GetMessageStatus(ctx context.Context, in *impb.GetMessageStatusRequest, opts ...grpc.CallOption) (*impb.GetMessageStatusResponse, error) {
	return &impb.GetMessageStatusResponse{}, nil
}

func TestAuthClientAdapter_ValidateToken(t *testing.T) {
	adapter := &authClientAdapter{client: &mockAuthPBClient{validateFunc: func(ctx context.Context, in *authpb.ValidateTokenRequest, opts ...grpc.CallOption) (*authpb.ValidateTokenResponse, error) {
		require.Equal(t, "token-123", in.AccessToken)
		return &authpb.ValidateTokenResponse{
			Valid:     true,
			UserId:    "user-1",
			DeviceId:  "device-1",
			ExpiresAt: timestamppb.New(time.Unix(12345, 0)),
		}, nil
	}}}

	claims, err := adapter.ValidateToken(context.Background(), "token-123")
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "device-1", claims.DeviceID)
	assert.Equal(t, int64(12345), claims.ExpiresAt)
}

func TestIMClientAdapter_RoutePrivateMessage(t *testing.T) {
	adapter := &imClientAdapter{client: &mockIMPBClient{routePrivateFunc: func(ctx context.Context, in *impb.RoutePrivateMessageRequest, opts ...grpc.CallOption) (*impb.RoutePrivateMessageResponse, error) {
		require.Equal(t, "msg-1", in.MsgId)
		require.Equal(t, "sender-1", in.SenderId)
		require.Equal(t, "recipient-1", in.RecipientId)
		require.Equal(t, "hello", in.Content)
		return &impb.RoutePrivateMessageResponse{
			SequenceNumber: 99,
			DeliveryStatus: impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED,
			ErrorCode:      impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED,
		}, nil
	}}}

	resp, err := adapter.RoutePrivateMessage(context.Background(), &service.RoutePrivateMessageRequest{
		MsgID:       "msg-1",
		SenderID:    "sender-1",
		RecipientID: "recipient-1",
		Content:     "hello",
		MessageType: "text",
		Timestamp:   time.Now().Unix(),
	})

	require.NoError(t, err)
	assert.Equal(t, int64(99), resp.SequenceNumber)
	assert.Equal(t, "delivered", resp.DeliveryStatus)
	assert.Equal(t, "", resp.ErrorCode)
}
