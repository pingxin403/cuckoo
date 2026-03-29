package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	gatewaypb "github.com/pingxin403/cuckoo/api/gen/go/im-gatewaypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// GatewayClient implements the GatewayClientInterface for communicating with gateway nodes.
type GatewayClient struct {
	connPool       map[string]*grpc.ClientConn
	connPoolMu     sync.RWMutex
	dialTimeout    time.Duration
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

const (
	gatewayErrCodeCircuitOpen     = "CIRCUIT_OPEN"
	gatewayErrCodeDialFailure     = "DIAL_FAILURE"
	gatewayErrCodeTimeout         = "TIMEOUT"
	gatewayErrCodeUnavailable     = "UNAVAILABLE"
	gatewayErrCodeResourceLimited = "RESOURCE_EXHAUSTED"
	gatewayErrCodeUnknown         = "UNKNOWN"
)

// NewGatewayClient creates a new gateway client instance.
func NewGatewayClient(dialTimeout, requestTimeout time.Duration) *GatewayClient {
	return &GatewayClient{
		connPool:       make(map[string]*grpc.ClientConn),
		dialTimeout:    dialTimeout,
		requestTimeout: requestTimeout,
		maxRetries:     2,
		retryBackoff:   50 * time.Millisecond,

		breakerStates:           make(map[string]circuitBreakerState),
		breakerFailureThreshold: 3,
		breakerOpenTimeout:      2 * time.Second,
	}
}

// PushMessage sends a message to a gateway node for delivery to the client.
func (gc *GatewayClient) PushMessage(ctx context.Context, gatewayAddr string, req *GatewayPushRequest) (*GatewayPushResponse, error) {
	if gc.isCircuitOpen(gatewayAddr) {
		err := fmt.Errorf("%s: circuit breaker open for gateway: %s", gatewayErrCodeCircuitOpen, gatewayAddr)
		return &GatewayPushResponse{
			Success:      false,
			ErrorCode:    gatewayErrCodeCircuitOpen,
			ErrorMessage: err.Error(),
		}, err
	}

	// Get or create connection to gateway node
	conn, err := gc.getConnection(ctx, gatewayAddr)
	if err != nil {
		gc.recordFailure(gatewayAddr)
		errCode := classifyGatewayError(err)
		return &GatewayPushResponse{
			Success:      false,
			ErrorCode:    errCode,
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
	grpcResp, err := gc.callWithRetry(pushCtx, func(callCtx context.Context) (*gatewaypb.PushMessageResponse, error) {
		return client.PushMessage(callCtx, grpcReq)
	})
	if err != nil {
		gc.recordFailure(gatewayAddr)
		errCode := classifyGatewayError(err)
		return &GatewayPushResponse{
			Success:      false,
			ErrorCode:    errCode,
			ErrorMessage: fmt.Sprintf("gateway push failed: %v", err),
		}, err
	}

	gc.recordSuccess(gatewayAddr)

	// Convert response
	return &GatewayPushResponse{
		Success:        grpcResp.Success,
		DeliveredCount: grpcResp.DeliveredCount,
		FailedDevices:  grpcResp.FailedDevices,
		ErrorCode:      "",
		ErrorMessage:   grpcResp.ErrorMessage,
	}, nil
}

func (gc *GatewayClient) isCircuitOpen(target string) bool {
	if gc.breakerFailureThreshold <= 0 || gc.breakerOpenTimeout <= 0 {
		return false
	}

	now := time.Now()
	gc.breakerMu.Lock()
	defer gc.breakerMu.Unlock()

	state, ok := gc.breakerStates[target]
	if !ok {
		return false
	}

	if state.openUntil.After(now) {
		return true
	}

	if !state.openUntil.IsZero() {
		state.openUntil = time.Time{}
		state.failures = 0
		gc.breakerStates[target] = state
	}

	return false
}

func (gc *GatewayClient) recordFailure(target string) {
	if gc.breakerFailureThreshold <= 0 || gc.breakerOpenTimeout <= 0 {
		return
	}

	gc.breakerMu.Lock()
	defer gc.breakerMu.Unlock()

	state := gc.breakerStates[target]
	state.failures++
	if state.failures >= gc.breakerFailureThreshold {
		state.openUntil = time.Now().Add(gc.breakerOpenTimeout)
	}
	gc.breakerStates[target] = state
}

func (gc *GatewayClient) recordSuccess(target string) {
	if gc.breakerFailureThreshold <= 0 || gc.breakerOpenTimeout <= 0 {
		return
	}

	gc.breakerMu.Lock()
	defer gc.breakerMu.Unlock()

	state := gc.breakerStates[target]
	state.failures = 0
	state.openUntil = time.Time{}
	gc.breakerStates[target] = state
}

func (gc *GatewayClient) callWithRetry(ctx context.Context, call func(context.Context) (*gatewaypb.PushMessageResponse, error)) (*gatewaypb.PushMessageResponse, error) {
	attempts := gc.maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := call(ctx)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !isRetryableGatewayError(err) || i == attempts-1 {
			return nil, err
		}

		if gc.retryBackoff > 0 {
			timer := time.NewTimer(gc.retryBackoff)
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

func isRetryableGatewayError(err error) bool {
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

func classifyGatewayError(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return gatewayErrCodeTimeout
	}

	st, ok := status.FromError(err)
	if !ok {
		return gatewayErrCodeUnknown
	}

	switch st.Code() {
	case codes.DeadlineExceeded:
		return gatewayErrCodeTimeout
	case codes.Unavailable:
		return gatewayErrCodeUnavailable
	case codes.ResourceExhausted:
		return gatewayErrCodeResourceLimited
	default:
		return gatewayErrCodeUnknown
	}
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
