package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pingxin403/cuckoo/libs/observability/tracing"
)

// PushService implements the gRPC service for pushing messages to clients.
type PushService struct {
	gateway         *GatewayService
	remoteForwarder RemoteForwarder
	metrics         CrossGatewayMetrics
	tracer          tracing.Tracer
}

type CrossGatewayMetrics interface {
	IncForwardSuccess(kind string)
	IncForwardFailure(kind, reason string)
	ObserveForwardLatency(kind string, duration time.Duration)
}

type RemoteForwarder interface {
	ForwardMessage(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error)
	ForwardReadReceipt(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error)
}

type noOpRemoteForwarder struct{}

func (f *noOpRemoteForwarder) ForwardMessage(ctx context.Context, gatewayNode string, req *PushMessageRequest) (*PushMessageResponse, error) {
	return &PushMessageResponse{Success: false}, nil
}

func (f *noOpRemoteForwarder) ForwardReadReceipt(ctx context.Context, gatewayNode string, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	return &PushMessageResponse{Success: false}, nil
}

// NewPushService creates a new push service instance.
func NewPushService(gateway *GatewayService) *PushService {
	return &PushService{
		gateway:         gateway,
		remoteForwarder: &noOpRemoteForwarder{},
		tracer:          tracing.NewNoOpTracer(),
	}
}

func (p *PushService) SetRemoteForwarder(forwarder RemoteForwarder) {
	if forwarder == nil {
		p.remoteForwarder = &noOpRemoteForwarder{}
		return
	}
	p.remoteForwarder = forwarder
}

func (p *PushService) SetMetrics(metrics CrossGatewayMetrics) {
	p.metrics = metrics
}

func (p *PushService) SetTracer(tracer tracing.Tracer) {
	if tracer == nil {
		p.tracer = tracing.NewNoOpTracer()
		return
	}
	p.tracer = tracer
}

// PushMessageRequest represents a message push request from IM Service.
type PushMessageRequest struct {
	MsgID          string
	RecipientID    string
	DeviceID       string // Optional: specific device, empty for all devices
	SenderID       string
	Content        string
	MessageType    string
	SequenceNumber int64
	Timestamp      int64
}

// PushMessageResponse represents the response to a push request.
type PushMessageResponse struct {
	Success        bool
	DeliveredCount int32
	FailedDevices  []string
	ErrorMessage   string
}

// PushMessage pushes a message to the specified user's WebSocket connection(s).
// Validates: Requirements 1.1, 3.2, 15.1, 15.2, 15.3
func (p *PushService) PushMessage(ctx context.Context, req *PushMessageRequest) (*PushMessageResponse, error) {
	if req.RecipientID == "" {
		return &PushMessageResponse{
			Success:      false,
			ErrorMessage: "recipient_id is required",
		}, nil
	}

	// Prepare server message
	serverMsg := ServerMessage{
		Type:           "message",
		MsgID:          req.MsgID,
		Sender:         req.SenderID,
		Content:        req.Content,
		Timestamp:      req.Timestamp,
		SequenceNumber: req.SequenceNumber,
	}

	data, err := json.Marshal(serverMsg)
	if err != nil {
		return &PushMessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to marshal message: %v", err),
		}, nil
	}

	var deliveredCount int32
	var failedDevices []string

	// If specific device is specified, push to that device only
	if req.DeviceID != "" {
		// Try local connection first
		key := req.RecipientID + "_" + req.DeviceID
		if conn, ok := p.gateway.connections.Load(key); ok {
			connection := conn.(*Connection)
			if p.pushToConnection(connection, data, req.MsgID) {
				deliveredCount++
			} else {
				failedDevices = append(failedDevices, req.DeviceID)
			}
		} else {
			// Device not connected locally, check Registry for remote gateway
			// This handles the case where device is on another gateway node
			failedDevices = append(failedDevices, req.DeviceID)
		}
	} else {
		// Multi-device delivery: Query Registry for all user devices
		// Validates: Requirements 15.1, 15.2, 15.3
		locations, err := p.gateway.registryClient.LookupUser(ctx, req.RecipientID)
		if err != nil {
			return &PushMessageResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("failed to lookup user devices: %v", err),
			}, nil
		}

		// Track which devices we've attempted to deliver to
		attemptedDevices := make(map[string]bool)

		// Deliver to all devices found in Registry
		for _, location := range locations {
			attemptedDevices[location.DeviceID] = true

			// Check if device is connected to this gateway node
			key := req.RecipientID + "_" + location.DeviceID
			if conn, ok := p.gateway.connections.Load(key); ok {
				// Device is connected locally
				connection := conn.(*Connection)
				if p.pushToConnection(connection, data, req.MsgID) {
					deliveredCount++
				} else {
					failedDevices = append(failedDevices, location.DeviceID)
				}
			} else {
				forwardReq := *req
				forwardReq.DeviceID = location.DeviceID
				spanCtx, span := p.tracer.StartSpan(ctx, "im-gateway.forward.message", tracing.WithSpanKind(tracing.SpanKindClient), tracing.WithAttributes(map[string]interface{}{
					"msg.id":           req.MsgID,
					"recipient.id":     req.RecipientID,
					"target.gateway":   location.GatewayNode,
					"target.device_id": location.DeviceID,
				}))
				forwardStart := time.Now()
				forwardResp, forwardErr := p.remoteForwarder.ForwardMessage(spanCtx, location.GatewayNode, &forwardReq)
				if p.metrics != nil {
					p.metrics.ObserveForwardLatency("message", time.Since(forwardStart))
				}
				if forwardErr != nil || forwardResp == nil || !forwardResp.Success || forwardResp.DeliveredCount <= 0 {
					failureReason := classifyForwardFailureReason(forwardErr, forwardResp)
					span.SetAttribute("forward.result", "failure")
					span.SetAttribute("forward.failure_reason", failureReason)
					span.SetAttribute("forward.latency_ms", time.Since(forwardStart).Milliseconds())
					if forwardErr != nil {
						span.RecordError(forwardErr)
						span.SetStatus(tracing.StatusCodeError, forwardErr.Error())
					}
					span.End()
					if p.metrics != nil {
						p.metrics.IncForwardFailure("message", failureReason)
					}
					failedDevices = append(failedDevices, location.DeviceID)
					continue
				}
				span.SetAttribute("forward.result", "success")
				span.SetAttribute("forward.delivered_count", forwardResp.DeliveredCount)
				span.SetAttribute("forward.latency_ms", time.Since(forwardStart).Milliseconds())
				span.SetStatus(tracing.StatusCodeOK, "")
				span.End()
				if p.metrics != nil {
					p.metrics.IncForwardSuccess("message")
				}
				deliveredCount += forwardResp.DeliveredCount
			}
		}

		// Also check for any local connections not in Registry (edge case)
		// This handles race conditions where device just connected
		p.gateway.connections.Range(func(key, value any) bool {
			keyStr := key.(string)
			// Check if this connection belongs to the recipient
			if len(keyStr) > len(req.RecipientID) && keyStr[:len(req.RecipientID)] == req.RecipientID {
				connection := value.(*Connection)
				if connection.UserID == req.RecipientID {
					// Skip if we already attempted this device
					if !attemptedDevices[connection.DeviceID] {
						if p.pushToConnection(connection, data, req.MsgID) {
							deliveredCount++
						} else {
							failedDevices = append(failedDevices, connection.DeviceID)
						}
					}
				}
			}
			return true
		})
	}

	return &PushMessageResponse{
		Success:        deliveredCount > 0,
		DeliveredCount: deliveredCount,
		FailedDevices:  failedDevices,
	}, nil
}

// pushToConnection attempts to push a message to a specific connection.
// Returns true if successful, false otherwise.
func (p *PushService) pushToConnection(conn *Connection, data []byte, _ string) bool {
	// Try to send message
	select {
	case conn.Send <- data:
		// Message queued successfully
		return true
	case <-time.After(1 * time.Second):
		// Channel full or blocked
		return false
	case <-conn.ctx.Done():
		// Connection closed
		return false
	}
}

// BroadcastToGroup broadcasts a message to all locally-connected group members.
// Validates: Requirements 2.2, 2.3
func (p *PushService) BroadcastToGroup(ctx context.Context, groupID string, message []byte) (int32, error) {
	var deliveredCount int32

	// Get group members from cache
	members, err := p.gateway.getGroupMembers(ctx, groupID)
	if err != nil {
		return 0, fmt.Errorf("failed to get group members: %w", err)
	}

	// Push to all locally-connected members
	for _, memberID := range members {
		// Find all connections for this member
		p.gateway.connections.Range(func(key, value any) bool {
			keyStr := key.(string)
			if len(keyStr) > len(memberID) && keyStr[:len(memberID)] == memberID {
				connection := value.(*Connection)
				if connection.UserID == memberID {
					select {
					case connection.Send <- message:
						deliveredCount++
					default:
						// Channel full, skip
					}
				}
			}
			return true
		})
	}

	return deliveredCount, nil
}

// getGroupMembers retrieves group members from cache or User Service.
// Validates: Requirements 17.1, 17.2
func (g *GatewayService) getGroupMembers(ctx context.Context, groupID string) ([]string, error) {
	// Use cache manager if available
	if g.cacheManager != nil {
		return g.cacheManager.GetGroupMembers(ctx, groupID)
	}

	// Fallback: Check Redis cache
	cacheKey := fmt.Sprintf("group_members:%s", groupID)
	if g.redisClient != nil {
		cached, err := g.redisClient.SMembers(ctx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	if g.groupMemberProvider != nil {
		members, err := g.groupMemberProvider.GetGroupMembers(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if g.redisClient != nil && len(members) > 0 {
			values := make([]interface{}, 0, len(members))
			for _, member := range members {
				values = append(values, member)
			}
			if len(values) > 0 {
				_ = g.redisClient.SAdd(ctx, cacheKey, values...).Err()
				_ = g.redisClient.Expire(ctx, cacheKey, 5*time.Minute).Err()
			}
		}
		return members, nil
	}

	return []string{}, nil
}

func (g *GatewayService) SetGroupMemberProvider(provider GroupMemberProvider) {
	g.groupMemberProvider = provider
	if g.cacheManager != nil {
		g.cacheManager.SetGroupMemberProvider(provider)
	}
}

func (g *GatewayService) SetRemoteForwarder(forwarder RemoteForwarder) {
	if g.pushService != nil {
		g.pushService.SetRemoteForwarder(forwarder)
	}
}

func (g *GatewayService) PushMessage(ctx context.Context, req *PushMessageRequest) (*PushMessageResponse, error) {
	if g.pushService == nil {
		return nil, fmt.Errorf("push service is not initialized")
	}
	return g.pushService.PushMessage(ctx, req)
}

func (g *GatewayService) PushReadReceipt(ctx context.Context, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	if g.pushService == nil {
		return nil, fmt.Errorf("push service is not initialized")
	}
	return g.pushService.PushReadReceipt(ctx, req)
}

// InvalidateGroupCache invalidates the group membership cache.
// Validates: Requirements 2.9, 17.3
func (g *GatewayService) InvalidateGroupCache(ctx context.Context, groupID string) error {
	cacheKey := fmt.Sprintf("group_members:%s", groupID)

	if g.redisClient != nil {
		return g.redisClient.Del(ctx, cacheKey).Err()
	}

	return nil
}

// PushReadReceiptRequest represents a read receipt push request.
type PushReadReceiptRequest struct {
	MsgID          string
	SenderID       string // Original message sender (recipient of read receipt)
	ReaderID       string // User who read the message
	ConversationID string
	ReadAt         int64
}

// PushReadReceipt pushes a read receipt to the message sender.
// Validates: Requirements 5.3, 5.4, 15.4
func (p *PushService) PushReadReceipt(ctx context.Context, req *PushReadReceiptRequest) (*PushMessageResponse, error) {
	if req.SenderID == "" || req.ReaderID == "" {
		return &PushMessageResponse{
			Success:      false,
			ErrorMessage: "sender_id and reader_id are required",
		}, nil
	}

	// Prepare read receipt message
	readReceiptMsg := ServerMessage{
		Type:           "read_receipt",
		MsgID:          req.MsgID,
		ReaderID:       req.ReaderID,
		ReadAt:         req.ReadAt,
		ConversationID: req.ConversationID,
		Timestamp:      time.Now().Unix(),
	}

	data, err := json.Marshal(readReceiptMsg)
	if err != nil {
		return &PushMessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to marshal read receipt: %v", err),
		}, nil
	}

	var deliveredCount int32
	var failedDevices []string

	// Multi-device delivery: Query Registry for all sender devices
	// Validates: Requirements 15.4 (read receipt sync across devices)
	locations, err := p.gateway.registryClient.LookupUser(ctx, req.SenderID)
	if err != nil {
		return &PushMessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to lookup sender devices: %v", err),
		}, nil
	}

	// Track which devices we've attempted to deliver to
	attemptedDevices := make(map[string]bool)

	// Deliver to all devices found in Registry
	for _, location := range locations {
		attemptedDevices[location.DeviceID] = true

		// Check if device is connected to this gateway node
		key := req.SenderID + "_" + location.DeviceID
		if conn, ok := p.gateway.connections.Load(key); ok {
			// Device is connected locally
			connection := conn.(*Connection)
			if p.pushToConnection(connection, data, req.MsgID) {
				deliveredCount++
			} else {
				failedDevices = append(failedDevices, location.DeviceID)
			}
		} else {
			spanCtx, span := p.tracer.StartSpan(ctx, "im-gateway.forward.read_receipt", tracing.WithSpanKind(tracing.SpanKindClient), tracing.WithAttributes(map[string]interface{}{
				"msg.id":           req.MsgID,
				"sender.id":        req.SenderID,
				"reader.id":        req.ReaderID,
				"target.gateway":   location.GatewayNode,
				"target.device_id": location.DeviceID,
			}))
			forwardStart := time.Now()
			forwardResp, forwardErr := p.remoteForwarder.ForwardReadReceipt(spanCtx, location.GatewayNode, req)
			if p.metrics != nil {
				p.metrics.ObserveForwardLatency("read_receipt", time.Since(forwardStart))
			}
			if forwardErr != nil || forwardResp == nil || !forwardResp.Success || forwardResp.DeliveredCount <= 0 {
				failureReason := classifyForwardFailureReason(forwardErr, forwardResp)
				span.SetAttribute("forward.result", "failure")
				span.SetAttribute("forward.failure_reason", failureReason)
				span.SetAttribute("forward.latency_ms", time.Since(forwardStart).Milliseconds())
				if forwardErr != nil {
					span.RecordError(forwardErr)
					span.SetStatus(tracing.StatusCodeError, forwardErr.Error())
				}
				span.End()
				if p.metrics != nil {
					p.metrics.IncForwardFailure("read_receipt", failureReason)
				}
				failedDevices = append(failedDevices, location.DeviceID)
				continue
			}
			span.SetAttribute("forward.result", "success")
			span.SetAttribute("forward.delivered_count", forwardResp.DeliveredCount)
			span.SetAttribute("forward.latency_ms", time.Since(forwardStart).Milliseconds())
			span.SetStatus(tracing.StatusCodeOK, "")
			span.End()
			if p.metrics != nil {
				p.metrics.IncForwardSuccess("read_receipt")
			}
			deliveredCount += forwardResp.DeliveredCount
		}
	}

	// Also check for any local connections not in Registry (edge case)
	p.gateway.connections.Range(func(key, value any) bool {
		keyStr := key.(string)
		// Check if this connection belongs to the sender
		if len(keyStr) > len(req.SenderID) && keyStr[:len(req.SenderID)] == req.SenderID {
			connection := value.(*Connection)
			if connection.UserID == req.SenderID {
				// Skip if we already attempted this device
				if !attemptedDevices[connection.DeviceID] {
					if p.pushToConnection(connection, data, req.MsgID) {
						deliveredCount++
					} else {
						failedDevices = append(failedDevices, connection.DeviceID)
					}
				}
			}
		}
		return true
	})

	return &PushMessageResponse{
		Success:        deliveredCount > 0,
		DeliveredCount: deliveredCount,
		FailedDevices:  failedDevices,
	}, nil
}

func classifyForwardFailureReason(err error, resp *PushMessageResponse) string {
	if err != nil {
		errText := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errText, "deadline") || strings.Contains(errText, "timeout"):
			return "timeout"
		case strings.Contains(errText, "unavailable"):
			return "unavailable"
		default:
			return "transport_error"
		}
	}

	if resp == nil {
		return "empty_response"
	}

	if !resp.Success {
		if resp.ErrorMessage != "" {
			return "remote_error"
		}
		return "not_delivered"
	}

	if resp.DeliveredCount <= 0 {
		return "zero_delivered"
	}

	return "unknown"
}
