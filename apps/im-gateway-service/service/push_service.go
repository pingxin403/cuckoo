package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PushService implements the gRPC service for pushing messages to clients.
type PushService struct {
	gateway *GatewayService
}

// NewPushService creates a new push service instance.
func NewPushService(gateway *GatewayService) *PushService {
	return &PushService{
		gateway: gateway,
	}
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
// Validates: Requirements 1.1, 3.2
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
		key := req.RecipientID + "_" + req.DeviceID
		if conn, ok := p.gateway.connections.Load(key); ok {
			connection := conn.(*Connection)
			if p.pushToConnection(connection, data, req.MsgID) {
				deliveredCount++
			} else {
				failedDevices = append(failedDevices, req.DeviceID)
			}
		} else {
			failedDevices = append(failedDevices, req.DeviceID)
		}
	} else {
		// Push to all devices for this user (multi-device support)
		// Validates: Requirements 15.1, 15.2, 15.3
		p.gateway.connections.Range(func(key, value interface{}) bool {
			keyStr := key.(string)
			// Check if this connection belongs to the recipient
			if len(keyStr) > len(req.RecipientID) && keyStr[:len(req.RecipientID)] == req.RecipientID {
				connection := value.(*Connection)
				if connection.UserID == req.RecipientID {
					if p.pushToConnection(connection, data, req.MsgID) {
						deliveredCount++
					} else {
						failedDevices = append(failedDevices, connection.DeviceID)
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
func (p *PushService) pushToConnection(conn *Connection, data []byte, msgID string) bool {
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
		p.gateway.connections.Range(func(key, value interface{}) bool {
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
	// Check cache first
	cacheKey := fmt.Sprintf("group_members:%s", groupID)

	// Try to get from Redis cache
	if g.redisClient != nil {
		cached, err := g.redisClient.SMembers(ctx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	// TODO: Fetch from User Service if not in cache
	// For now, return empty list
	return []string{}, nil
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
