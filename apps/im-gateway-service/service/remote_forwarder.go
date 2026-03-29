package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	im_gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCRemoteForwarder struct {
	nodeAddrs      map[string]string
	connPool       map[string]*grpc.ClientConn
	connPoolMu     sync.RWMutex
	requestTimeout time.Duration
	maxRetries     int
	retryBackoff   time.Duration

	breakerMu               sync.Mutex
	breakerStates           map[string]circuitBreakerState
	breakerFailureThreshold int
	breakerOpenTimeout      time.Duration
}

type circuitBreakerState struct {
	failures  int
	openUntil time.Time
}

func NewGRPCRemoteForwarder(nodeAddrs map[string]string) *GRPCRemoteForwarder {
	if nodeAddrs == nil {
		nodeAddrs = map[string]string{}
	}
	return &GRPCRemoteForwarder{
		nodeAddrs:      nodeAddrs,
		connPool:       make(map[string]*grpc.ClientConn),
		requestTimeout: 2 * time.Second,
		maxRetries:     2,
		retryBackoff:   50 * time.Millisecond,

		breakerStates:           make(map[string]circuitBreakerState),
		breakerFailureThreshold: 3,
		breakerOpenTimeout:      2 * time.Second,
	}
}

func (f *GRPCRemoteForwarder) ForwardMessage(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
	if f.isCircuitOpen(gatewayNode) {
		return nil, fmt.Errorf("circuit breaker open for gateway: %s", gatewayNode)
	}

	conn, err := f.getConnection(ctx, gatewayNode)
	if err != nil {
		f.recordFailure(gatewayNode)
		return nil, err
	}

	client := im_gatewaypb.NewUimUgatewayUserviceServiceClient(conn)
	resp, err := f.callWithRetry(ctx, func(callCtx context.Context) (*im_gatewaypb.PushMessageResponse, error) {
		return client.PushMessage(callCtx, &im_gatewaypb.PushMessageRequest{
			MsgId:          req.MsgID,
			RecipientId:    req.RecipientID,
			DeviceId:       req.DeviceID,
			SenderId:       req.SenderID,
			Content:        req.Content,
			MessageType:    req.MessageType,
			SequenceNumber: req.SequenceNumber,
			Timestamp:      req.Timestamp,
		})
	})
	if err != nil {
		f.recordFailure(gatewayNode)
		return nil, err
	}

	f.recordSuccess(gatewayNode)

	return &PushMessageResponse{
		Success:        resp.GetSuccess(),
		DeliveredCount: resp.GetDeliveredCount(),
		FailedDevices:  resp.GetFailedDevices(),
		ErrorMessage:   resp.GetErrorMessage(),
	}, nil
}

func (f *GRPCRemoteForwarder) ForwardReadReceipt(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	if f.isCircuitOpen(gatewayNode) {
		return nil, fmt.Errorf("circuit breaker open for gateway: %s", gatewayNode)
	}

	conn, err := f.getConnection(ctx, gatewayNode)
	if err != nil {
		f.recordFailure(gatewayNode)
		return nil, err
	}

	client := im_gatewaypb.NewUimUgatewayUserviceServiceClient(conn)
	resp, err := f.callWithRetry(ctx, func(callCtx context.Context) (*im_gatewaypb.PushMessageResponse, error) {
		return client.PushReadReceipt(callCtx, &im_gatewaypb.PushReadReceiptRequest{
			MsgId:          req.MsgID,
			SenderId:       req.SenderID,
			ReaderId:       req.ReaderID,
			ConversationId: req.ConversationID,
			ReadAt:         req.ReadAt,
		})
	})
	if err != nil {
		f.recordFailure(gatewayNode)
		return nil, err
	}

	f.recordSuccess(gatewayNode)

	return &PushMessageResponse{
		Success:        resp.GetSuccess(),
		DeliveredCount: resp.GetDeliveredCount(),
		FailedDevices:  resp.GetFailedDevices(),
		ErrorMessage:   resp.GetErrorMessage(),
	}, nil
}

func (f *GRPCRemoteForwarder) getConnection(ctx context.Context, gatewayNode string) (*grpc.ClientConn, error) {
	addr, ok := f.nodeAddrs[gatewayNode]
	if !ok || addr == "" {
		return nil, fmt.Errorf("gateway node address not configured: %s", gatewayNode)
	}

	f.connPoolMu.RLock()
	if conn, exists := f.connPool[gatewayNode]; exists {
		f.connPoolMu.RUnlock()
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		f.removeConnection(gatewayNode)
	} else {
		f.connPoolMu.RUnlock()
	}

	f.connPoolMu.Lock()
	defer f.connPoolMu.Unlock()

	if conn, exists := f.connPool[gatewayNode]; exists {
		if conn.GetState().String() != "SHUTDOWN" {
			return conn, nil
		}
		_ = conn.Close()
		delete(f.connPool, gatewayNode)
	}

	dialCtx := ctx
	cancel := func() {}
	if f.requestTimeout > 0 {
		dialCtx, cancel = context.WithTimeout(ctx, f.requestTimeout)
	}
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	f.connPool[gatewayNode] = conn
	return conn, nil
}

func (f *GRPCRemoteForwarder) removeConnection(gatewayNode string) {
	f.connPoolMu.Lock()
	defer f.connPoolMu.Unlock()

	if conn, exists := f.connPool[gatewayNode]; exists {
		_ = conn.Close()
		delete(f.connPool, gatewayNode)
	}
}

func (f *GRPCRemoteForwarder) Close() error {
	f.connPoolMu.Lock()
	defer f.connPoolMu.Unlock()

	for node, conn := range f.connPool {
		_ = conn.Close()
		delete(f.connPool, node)
	}

	return nil
}

func (f *GRPCRemoteForwarder) isCircuitOpen(target string) bool {
	if f.breakerFailureThreshold <= 0 || f.breakerOpenTimeout <= 0 {
		return false
	}

	now := time.Now()
	f.breakerMu.Lock()
	defer f.breakerMu.Unlock()

	state, ok := f.breakerStates[target]
	if !ok {
		return false
	}

	if state.openUntil.After(now) {
		return true
	}

	if !state.openUntil.IsZero() {
		state.openUntil = time.Time{}
		state.failures = 0
		f.breakerStates[target] = state
	}

	return false
}

func (f *GRPCRemoteForwarder) recordFailure(target string) {
	if f.breakerFailureThreshold <= 0 || f.breakerOpenTimeout <= 0 {
		return
	}

	f.breakerMu.Lock()
	defer f.breakerMu.Unlock()

	state := f.breakerStates[target]
	state.failures++
	if state.failures >= f.breakerFailureThreshold {
		state.openUntil = time.Now().Add(f.breakerOpenTimeout)
	}
	f.breakerStates[target] = state
}

func (f *GRPCRemoteForwarder) recordSuccess(target string) {
	if f.breakerFailureThreshold <= 0 || f.breakerOpenTimeout <= 0 {
		return
	}

	f.breakerMu.Lock()
	defer f.breakerMu.Unlock()

	state := f.breakerStates[target]
	state.failures = 0
	state.openUntil = time.Time{}
	f.breakerStates[target] = state
}

func (f *GRPCRemoteForwarder) callWithRetry(ctx context.Context, call func(context.Context) (*im_gatewaypb.PushMessageResponse, error)) (*im_gatewaypb.PushMessageResponse, error) {
	attempts := f.maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		callCtx := ctx
		cancel := func() {}
		if f.requestTimeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, f.requestTimeout)
		}

		resp, err := call(callCtx)
		cancel()
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !isRetryableGRPCError(err) || i == attempts-1 {
			return nil, err
		}

		if f.retryBackoff > 0 {
			timer := time.NewTimer(f.retryBackoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return nil, lastErr
}

func isRetryableGRPCError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}
