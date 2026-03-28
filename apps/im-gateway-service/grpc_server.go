package main

import (
	"context"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"github.com/pingxin403/cuckoo/apps/im-gateway-service/service"
)

type gatewayPushAPI interface {
	PushMessage(ctx context.Context, req *service.PushMessageRequest) (*service.PushMessageResponse, error)
	PushReadReceipt(ctx context.Context, req *service.PushReadReceiptRequest) (*service.PushMessageResponse, error)
}

type gatewayRPCServer struct {
	im_gatewaypb.UnimplementedUimUgatewayUserviceServiceServer
	gateway gatewayPushAPI
}

func newGatewayRPCServer(gateway gatewayPushAPI) *gatewayRPCServer {
	return &gatewayRPCServer{gateway: gateway}
}

func (s *gatewayRPCServer) HealthCheck(ctx context.Context, req *im_gatewaypb.HealthCheckRequest) (*im_gatewaypb.HealthCheckResponse, error) {
	_ = ctx
	_ = req
	return &im_gatewaypb.HealthCheckResponse{Status: "OK"}, nil
}

func (s *gatewayRPCServer) PushMessage(ctx context.Context, req *im_gatewaypb.PushMessageRequest) (*im_gatewaypb.PushMessageResponse, error) {
	resp, err := s.gateway.PushMessage(ctx, &service.PushMessageRequest{
		MsgID:          req.GetMsgId(),
		RecipientID:    req.GetRecipientId(),
		DeviceID:       req.GetDeviceId(),
		SenderID:       req.GetSenderId(),
		Content:        req.GetContent(),
		MessageType:    req.GetMessageType(),
		SequenceNumber: req.GetSequenceNumber(),
		Timestamp:      req.GetTimestamp(),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return &im_gatewaypb.PushMessageResponse{}, nil
	}
	return &im_gatewaypb.PushMessageResponse{
		Success:        resp.Success,
		DeliveredCount: resp.DeliveredCount,
		FailedDevices:  resp.FailedDevices,
		ErrorMessage:   resp.ErrorMessage,
	}, nil
}

func (s *gatewayRPCServer) PushReadReceipt(ctx context.Context, req *im_gatewaypb.PushReadReceiptRequest) (*im_gatewaypb.PushMessageResponse, error) {
	resp, err := s.gateway.PushReadReceipt(ctx, &service.PushReadReceiptRequest{
		MsgID:          req.GetMsgId(),
		SenderID:       req.GetSenderId(),
		ReaderID:       req.GetReaderId(),
		ConversationID: req.GetConversationId(),
		ReadAt:         req.GetReadAt(),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return &im_gatewaypb.PushMessageResponse{}, nil
	}
	return &im_gatewaypb.PushMessageResponse{
		Success:        resp.Success,
		DeliveredCount: resp.DeliveredCount,
		FailedDevices:  resp.FailedDevices,
		ErrorMessage:   resp.ErrorMessage,
	}, nil
}
