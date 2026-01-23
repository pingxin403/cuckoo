package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// GatewayService manages WebSocket connections and message routing.
type GatewayService struct {
	// Connection management
	connections sync.Map // map[string]*Connection (userID -> Connection)
	upgrader    websocket.Upgrader

	// External services
	authClient     AuthServiceClient
	registryClient RegistryClient
	imClient       IMServiceClient
	redisClient    *redis.Client

	// Internal services
	pushService   *PushService
	cacheManager  *CacheManager
	kafkaConsumer *KafkaConsumer

	// Configuration
	config GatewayConfig

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// GatewayConfig contains configuration for the gateway service.
type GatewayConfig struct {
	// Connection settings
	HeartbeatInterval time.Duration // Default: 30s
	HeartbeatTimeout  time.Duration // Default: 90s
	ReadBufferSize    int           // Default: 4096
	WriteBufferSize   int           // Default: 4096

	// Message settings
	MaxMessageSize int64         // Default: 10KB
	WriteWait      time.Duration // Default: 10s
	PongWait       time.Duration // Default: 60s
	PingPeriod     time.Duration // Default: 54s (must be less than PongWait)

	// Registry settings
	RegistryTTL           time.Duration // Default: 90s
	RegistryRenewInterval time.Duration // Default: 30s

	// Rate limiting
	MaxMessagesPerSecond int // Default: 100
}

// DefaultGatewayConfig returns default configuration.
func DefaultGatewayConfig() GatewayConfig {
	return GatewayConfig{
		HeartbeatInterval:     30 * time.Second,
		HeartbeatTimeout:      90 * time.Second,
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		MaxMessageSize:        10 * 1024, // 10KB
		WriteWait:             10 * time.Second,
		PongWait:              60 * time.Second,
		PingPeriod:            54 * time.Second,
		RegistryTTL:           90 * time.Second,
		RegistryRenewInterval: 30 * time.Second,
		MaxMessagesPerSecond:  100,
	}
}

// Connection represents a WebSocket connection.
type Connection struct {
	UserID   string
	DeviceID string
	Conn     *websocket.Conn
	Send     chan []byte
	Gateway  *GatewayService

	// Rate limiting
	lastMessageTime time.Time
	messageCount    int

	// Lifecycle
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// AuthServiceClient defines the interface for authentication service.
type AuthServiceClient interface {
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
}

// RegistryClient defines the interface for registry operations.
type RegistryClient interface {
	RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) error
	UnregisterUser(ctx context.Context, userID, deviceID string) error
	RenewLease(ctx context.Context, userID, deviceID string) error
	LookupUser(ctx context.Context, userID string) ([]GatewayLocation, error)
	Watch(ctx context.Context, prefix string, callback func(clientv3.WatchResponse)) error
	Close() error
}

// IMServiceClient defines the interface for IM service.
type IMServiceClient interface {
	RoutePrivateMessage(ctx context.Context, req *RoutePrivateMessageRequest) (*RoutePrivateMessageResponse, error)
	RouteGroupMessage(ctx context.Context, req *RouteGroupMessageRequest) (*RouteGroupMessageResponse, error)
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID    string
	DeviceID  string
	ExpiresAt int64
}

// GatewayLocation represents a user's location in the system.
type GatewayLocation struct {
	GatewayNode string
	DeviceID    string
	ConnectedAt int64
}

// RoutePrivateMessageRequest represents a private message routing request.
type RoutePrivateMessageRequest struct {
	MsgID       string
	SenderID    string
	RecipientID string
	Content     string
	MessageType string
	Timestamp   int64
}

// RoutePrivateMessageResponse represents a private message routing response.
type RoutePrivateMessageResponse struct {
	SequenceNumber  int64
	ServerTimestamp int64
	DeliveryStatus  string
	ErrorCode       string
	ErrorMessage    string
}

// RouteGroupMessageRequest represents a group message routing request.
type RouteGroupMessageRequest struct {
	MsgID       string
	SenderID    string
	GroupID     string
	Content     string
	MessageType string
	Timestamp   int64
}

// RouteGroupMessageResponse represents a group message routing response.
type RouteGroupMessageResponse struct {
	SequenceNumber     int64
	ServerTimestamp    int64
	OnlineMemberCount  int32
	OfflineMemberCount int32
	ErrorCode          string
	ErrorMessage       string
}

// ClientMessage represents a message from the client.
type ClientMessage struct {
	Type      string          `json:"type"` // "send_msg", "ack", "heartbeat"
	MsgID     string          `json:"msg_id"`
	Recipient string          `json:"recipient"` // user_id or group_id
	Content   string          `json:"content"`
	Timestamp int64           `json:"timestamp"`
	Extra     json.RawMessage `json:"extra,omitempty"`
}

// ServerMessage represents a message to the client.
type ServerMessage struct {
	Type           string `json:"type"` // "message", "ack", "ping", "error"
	MsgID          string `json:"msg_id"`
	Sender         string `json:"sender"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
	SequenceNumber int64  `json:"sequence_number"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// NewGatewayService creates a new gateway service instance.
func NewGatewayService(
	authClient AuthServiceClient,
	registryClient RegistryClient,
	imClient IMServiceClient,
	redisClient *redis.Client,
	config GatewayConfig,
) *GatewayService {
	ctx, cancel := context.WithCancel(context.Background())

	gateway := &GatewayService{
		authClient:     authClient,
		registryClient: registryClient,
		imClient:       imClient,
		redisClient:    redisClient,
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking
				return true
			},
		},
	}

	// Initialize internal services
	gateway.pushService = NewPushService(gateway)
	gateway.cacheManager = NewCacheManager(
		redisClient,
		registryClient,
		5*time.Minute, // user cache TTL
		5*time.Minute, // group cache TTL
	)

	return gateway
}

// Start starts the gateway service and all internal components.
func (g *GatewayService) Start(kafkaConfig KafkaConfig) error {
	// Start cache manager
	if err := g.cacheManager.Start(); err != nil {
		return fmt.Errorf("failed to start cache manager: %w", err)
	}

	// Start Kafka consumer for group messages
	g.kafkaConsumer = NewKafkaConsumer(kafkaConfig, g, g.pushService)
	if err := g.kafkaConsumer.Start(); err != nil {
		return fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	return nil
}

// HandleWebSocket handles WebSocket connection upgrade and lifecycle.
// Validates: Requirements 6.1, 6.2, 11.1
func (g *GatewayService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract JWT token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		http.Error(w, "Missing authentication token", http.StatusUnauthorized)
		return
	}

	// Validate JWT token via Auth Service (Requirement 11.2)
	claims, err := g.authClient.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upgrade connection: %v", err), http.StatusInternalServerError)
		return
	}

	// Create connection context
	ctx, cancel := context.WithCancel(g.ctx)

	// Create connection object
	connection := &Connection{
		UserID:   claims.UserID,
		DeviceID: claims.DeviceID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Gateway:  g,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Register user in Registry (Requirement 7.1, 7.2)
	if err := g.registryClient.RegisterUser(ctx, claims.UserID, claims.DeviceID, g.getGatewayNodeID()); err != nil {
		conn.Close()
		http.Error(w, fmt.Sprintf("Failed to register user: %v", err), http.StatusInternalServerError)
		return
	}

	// Store connection
	g.connections.Store(connection.UserID+"_"+connection.DeviceID, connection)

	// Start connection handlers
	g.wg.Add(2)
	go connection.readPump()
	go connection.writePump()

	// Start heartbeat and registry renewal
	g.wg.Add(1)
	go connection.heartbeatLoop()
}

// getGatewayNodeID returns the unique identifier for this gateway node.
func (g *GatewayService) getGatewayNodeID() string {
	// TODO: Implement proper node ID generation (e.g., hostname, pod name)
	return "gateway-node-1"
}

// Shutdown gracefully shuts down the gateway service.
// Validates: Requirement 6.5
func (g *GatewayService) Shutdown(ctx context.Context) error {
	// Cancel context to signal shutdown
	g.cancel()

	// Stop Kafka consumer
	if g.kafkaConsumer != nil {
		if err := g.kafkaConsumer.Stop(); err != nil {
			// Log error but continue shutdown
		}
	}

	// Stop cache manager
	if g.cacheManager != nil {
		if err := g.cacheManager.Stop(); err != nil {
			// Log error but continue shutdown
		}
	}

	// Send close notification to all connections
	closeMsg := ServerMessage{
		Type:         "close",
		ErrorCode:    "SERVER_SHUTDOWN",
		ErrorMessage: "Server is shutting down",
		Timestamp:    time.Now().Unix(),
	}
	closeData, _ := json.Marshal(closeMsg)

	// Close all connections with notification
	g.connections.Range(func(key, value interface{}) bool {
		conn := value.(*Connection)
		// Try to send close notification
		select {
		case conn.Send <- closeData:
		case <-time.After(1 * time.Second):
			// Timeout, force close
		}
		conn.Close()
		return true
	})

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("shutdown timeout after 30 seconds")
	}
}

// readPump reads messages from the WebSocket connection.
// Validates: Requirements 1.1, 2.1
func (c *Connection) readPump() {
	defer func() {
		c.Gateway.wg.Done()
		c.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(c.Gateway.config.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Gateway.config.PongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			return
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			// Send error response
			c.sendError("INVALID_MESSAGE", "Failed to parse message")
			continue
		}

		// Handle message based on type
		switch clientMsg.Type {
		case "send_msg":
			c.handleSendMessage(&clientMsg)
		case "ack":
			c.handleAck(&clientMsg)
		case "heartbeat":
			c.handleHeartbeat(&clientMsg)
		default:
			c.sendError("UNKNOWN_MESSAGE_TYPE", fmt.Sprintf("Unknown message type: %s", clientMsg.Type))
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.Gateway.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Gateway.wg.Done()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Gateway.config.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Gateway.config.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// heartbeatLoop maintains the registry lease.
// Validates: Requirements 6.3, 6.4, 7.2
func (c *Connection) heartbeatLoop() {
	ticker := time.NewTicker(c.Gateway.config.RegistryRenewInterval)
	defer func() {
		ticker.Stop()
		c.Gateway.wg.Done()
	}()

	for {
		select {
		case <-ticker.C:
			if err := c.Gateway.registryClient.RenewLease(c.ctx, c.UserID, c.DeviceID); err != nil {
				// Log error and close connection
				c.Close()
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// handleSendMessage handles a send message request from the client.
func (c *Connection) handleSendMessage(msg *ClientMessage) {
	// Rate limiting (Requirement 10.5)
	if !c.checkRateLimit() {
		c.sendError("RATE_LIMIT_EXCEEDED", "Too many messages")
		return
	}

	// TODO: Determine if this is a private or group message
	// For now, assume private message if recipient starts with "user_"
	// and group message if it starts with "group_"

	if len(msg.Recipient) > 5 && msg.Recipient[:5] == "user_" {
		// Private message
		req := &RoutePrivateMessageRequest{
			MsgID:       msg.MsgID,
			SenderID:    c.UserID,
			RecipientID: msg.Recipient,
			Content:     msg.Content,
			MessageType: "text",
			Timestamp:   msg.Timestamp,
		}

		resp, err := c.Gateway.imClient.RoutePrivateMessage(c.ctx, req)
		if err != nil {
			c.sendError("ROUTING_ERROR", fmt.Sprintf("Failed to route message: %v", err))
			return
		}

		if resp.ErrorCode != "" {
			c.sendError(resp.ErrorCode, resp.ErrorMessage)
			return
		}

		// Send ACK to sender
		c.sendAck(msg.MsgID, resp.SequenceNumber)
	} else {
		// Group message
		req := &RouteGroupMessageRequest{
			MsgID:       msg.MsgID,
			SenderID:    c.UserID,
			GroupID:     msg.Recipient,
			Content:     msg.Content,
			MessageType: "text",
			Timestamp:   msg.Timestamp,
		}

		resp, err := c.Gateway.imClient.RouteGroupMessage(c.ctx, req)
		if err != nil {
			c.sendError("ROUTING_ERROR", fmt.Sprintf("Failed to route message: %v", err))
			return
		}

		if resp.ErrorCode != "" {
			c.sendError(resp.ErrorCode, resp.ErrorMessage)
			return
		}

		// Send ACK to sender
		c.sendAck(msg.MsgID, resp.SequenceNumber)
	}
}

// handleAck handles an acknowledgment from the client.
func (c *Connection) handleAck(msg *ClientMessage) {
	// TODO: Implement ACK handling
	// This would update delivery status in the system
}

// handleHeartbeat handles a heartbeat message from the client.
func (c *Connection) handleHeartbeat(msg *ClientMessage) {
	// Send heartbeat response
	response := ServerMessage{
		Type:      "heartbeat",
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		// Channel full, skip
	}
}

// checkRateLimit checks if the connection is within rate limits.
func (c *Connection) checkRateLimit() bool {
	now := time.Now()
	if now.Sub(c.lastMessageTime) > time.Second {
		c.messageCount = 0
		c.lastMessageTime = now
	}

	c.messageCount++
	return c.messageCount <= c.Gateway.config.MaxMessagesPerSecond
}

// sendAck sends an acknowledgment to the client.
func (c *Connection) sendAck(msgID string, seqNum int64) {
	response := ServerMessage{
		Type:           "ack",
		MsgID:          msgID,
		SequenceNumber: seqNum,
		Timestamp:      time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		// Channel full, skip
	}
}

// sendError sends an error message to the client.
func (c *Connection) sendError(code, message string) {
	response := ServerMessage{
		Type:         "error",
		ErrorCode:    code,
		ErrorMessage: message,
		Timestamp:    time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		// Channel full, skip
	}
}

// Close closes the connection and cleans up resources.
func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		c.cancel()

		// Unregister from Registry
		c.Gateway.registryClient.UnregisterUser(context.Background(), c.UserID, c.DeviceID)

		// Remove from connections map
		c.Gateway.connections.Delete(c.UserID + "_" + c.DeviceID)

		// Close send channel
		close(c.Send)

		// Close WebSocket connection if it exists
		if c.Conn != nil {
			c.Conn.Close()
		}
	})
}
