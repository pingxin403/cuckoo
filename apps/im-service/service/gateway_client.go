package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GatewayClient implements the GatewayClientInterface for communicating with gateway nodes.
type GatewayClient struct {
	connPool       map[string]*grpc.ClientConn
	connPoolMu     sync.RWMutex
	dialTimeout    time.Duration
	requestTimeout time.Duration
}

// NewGatewayClient creates a new gateway client instance.
func NewGatewayClient(dialTimeout, requestTimeout time.Duration) *GatewayClient {
	return &GatewayClient{
		connPool:       make(map[string]*grpc.ClientConn),
		dialTimeout:    dialTimeout,
		requestTimeout: requestTimeout,
	}
}

// PushMessage sends a message to a gateway node for delivery to the client.
func (gc *GatewayClient) PushMessage(ctx context.Context, gatewayAddr string, req *GatewayPushRequest) (*GatewayPushResponse, error) {
	// Get or create connection to gateway node
	conn, err := gc.getConnection(ctx, gatewayAddr)
	if err != nil {
		return &GatewayPushResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to connect to gateway: %v", err),
		}, err
	}

	// Create gateway client
	client := gatewaypb.NewUimUgatewayUserviceServiceClient(conn)

	// Create context with timeout
	pushCtx, cancel := context.WithTimeout(ctx, gc.requestTimeout)
	defer cancel()

	// Build gRPC request
	grpcReq := &gatewaypb.PushMessageRequest{
		MsgId:          req.MsgID,
		RecipientId:    req.RecipientID,
		DeviceId:       req.DeviceID,
		SenderId:       req.SenderID,
		Content:        req.Content,
		MessageType:    req.MessageType,
		SequenceNumber: req.SequenceNumber,
		Timestamp:      req.Timestamp,
	}

	// Call gateway PushMessage RPC
	grpcResp, err := client.PushMessage(pushCtx, grpcReq)
	if err != nil {
		return &GatewayPushResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("gateway push failed: %v", err),
		}, err
	}

	// Convert response
	return &GatewayPushResponse{
		Success:        grpcResp.Success,
		DeliveredCount: grpcResp.DeliveredCount,
		FailedDevices:  grpcResp.FailedDevices,
		ErrorMessage:   grpcResp.ErrorMessage,
	}, nil
}

// getConnection retrieves or creates a gRPC connection to the gateway node.
func (gc *GatewayClient) getConnection(ctx context.Context, gatewayAddr string) (*grpc.ClientConn, error) {
	// Check if connection already exists
	gc.connPoolMu.RLock()
	if conn, ok := gc.connPool[gatewayAddr]; ok {
		gc.connPoolMu.RUnlock()
		// Verify connection is still valid
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		// Connection is dead, remove it
		gc.closeConnection(gatewayAddr)
	} else {
		gc.connPoolMu.RUnlock()
	}

	// Create new connection
	gc.connPoolMu.Lock()
	defer gc.connPoolMu.Unlock()

	// Double-check after acquiring write lock
	if conn, ok := gc.connPool[gatewayAddr]; ok {
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		gc.closeConnectionUnsafe(gatewayAddr)
	}

	// Create dial context with timeout
	dialCtx, cancel := context.WithTimeout(ctx, gc.dialTimeout)
	defer cancel()

	// Dial gateway node
	conn, err := grpc.DialContext(
		dialCtx,
		gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gateway %s: %w", gatewayAddr, err)
	}

	// Store connection in pool
	gc.connPool[gatewayAddr] = conn

	return conn, nil
}

// closeConnection closes and removes a connection from the pool.
func (gc *GatewayClient) closeConnection(gatewayAddr string) {
	gc.connPoolMu.Lock()
	defer gc.connPoolMu.Unlock()
	gc.closeConnectionUnsafe(gatewayAddr)
}

// closeConnectionUnsafe closes a connection without locking (caller must hold lock).
func (gc *GatewayClient) closeConnectionUnsafe(gatewayAddr string) {
	if conn, ok := gc.connPool[gatewayAddr]; ok {
		_ = conn.Close()
		delete(gc.connPool, gatewayAddr)
	}
}

// Close closes all connections in the pool.
func (gc *GatewayClient) Close() error {
	gc.connPoolMu.Lock()
	defer gc.connPoolMu.Unlock()

	for addr, conn := range gc.connPool {
		_ = conn.Close()
		delete(gc.connPool, addr)
	}

	return nil
}
