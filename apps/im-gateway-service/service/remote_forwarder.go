package service

import (
	"context"
	"fmt"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCRemoteForwarder struct {
	nodeAddrs map[string]string
}

func NewGRPCRemoteForwarder(nodeAddrs map[string]string) *GRPCRemoteForwarder {
	if nodeAddrs == nil {
		nodeAddrs = map[string]string{}
	}
	return &GRPCRemoteForwarder{nodeAddrs: nodeAddrs}
}

func (f *GRPCRemoteForwarder) ForwardMessage(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
	addr, ok := f.nodeAddrs[gatewayNode]
	if !ok || addr == "" {
		return nil, fmt.Errorf("gateway node address not configured: %s", gatewayNode)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	client := im_gatewaypb.NewUimUgatewayUserviceServiceClient(conn)
	resp, err := client.PushMessage(ctx, &im_gatewaypb.PushMessageRequest{
		MsgId:          req.MsgID,
		RecipientId:    req.RecipientID,
		DeviceId:       req.DeviceID,
		SenderId:       req.SenderID,
		Content:        req.Content,
		MessageType:    req.MessageType,
		SequenceNumber: req.SequenceNumber,
		Timestamp:      req.Timestamp,
	})
	if err != nil {
		return nil, err
	}

	return &PushMessageResponse{
		Success:        resp.GetSuccess(),
		DeliveredCount: resp.GetDeliveredCount(),
		FailedDevices:  resp.GetFailedDevices(),
		ErrorMessage:   resp.GetErrorMessage(),
	}, nil
}

func (f *GRPCRemoteForwarder) ForwardReadReceipt(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	addr, ok := f.nodeAddrs[gatewayNode]
	if !ok || addr == "" {
		return nil, fmt.Errorf("gateway node address not configured: %s", gatewayNode)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	client := im_gatewaypb.NewUimUgatewayUserviceServiceClient(conn)
	resp, err := client.PushReadReceipt(ctx, &im_gatewaypb.PushReadReceiptRequest{
		MsgId:          req.MsgID,
		SenderId:       req.SenderID,
		ReaderId:       req.ReaderID,
		ConversationId: req.ConversationID,
		ReadAt:         req.ReadAt,
	})
	if err != nil {
		return nil, err
	}

	return &PushMessageResponse{
		Success:        resp.GetSuccess(),
		DeliveredCount: resp.GetDeliveredCount(),
		FailedDevices:  resp.GetFailedDevices(),
		ErrorMessage:   resp.GetErrorMessage(),
	}, nil
}
