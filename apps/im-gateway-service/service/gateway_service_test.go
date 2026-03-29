package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pingxin403/cuckoo/libs/observability/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Mock implementations for testing

type mockAuthClient struct {
	validateFunc func(ctx context.Context, token string) (*TokenClaims, error)
}

func (m *mockAuthClient) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return &TokenClaims{
		UserID:    "user123",
		DeviceID:  "550e8400-e29b-41d4-a716-446655440000", // Valid UUID v4
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}, nil
}

type mockRegistryClient struct {
	users           map[string][]GatewayLocation
	mu              sync.RWMutex
	lookupError     error // For testing error cases
	lookupCallCount int   // Track number of LookupUser calls
}

func newMockRegistryClient() *mockRegistryClient {
	return &mockRegistryClient{
		users: make(map[string][]GatewayLocation),
	}
}

// SetLookupError sets an error to be returned by LookupUser
func (m *mockRegistryClient) SetLookupError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lookupError = err
}

// SetUserLocations sets the locations for a user (for testing)
func (m *mockRegistryClient) SetUserLocations(userID string, locations []GatewayLocation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[userID] = locations
}

func (m *mockRegistryClient) RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[userID] = append(m.users[userID], GatewayLocation{
		GatewayNode: gatewayNode,
		DeviceID:    deviceID,
		ConnectedAt: time.Now().Unix(),
	})
	return nil
}

func (m *mockRegistryClient) UnregisterUser(ctx context.Context, userID, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.users, userID)
	return nil
}

func (m *mockRegistryClient) RenewLease(ctx context.Context, userID, deviceID string) error {
	return nil
}

func (m *mockRegistryClient) LookupUser(ctx context.Context, userID string) ([]GatewayLocation, error) {
	m.mu.Lock()
	m.lookupCallCount++
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lookupError != nil {
		return nil, m.lookupError
	}

	if locations, ok := m.users[userID]; ok {
		return locations, nil
	}
	return nil, nil
}

// GetLookupCallCount returns the number of times LookupUser was called
func (m *mockRegistryClient) GetLookupCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lookupCallCount
}

func (m *mockRegistryClient) Watch(ctx context.Context, prefix string, callback func(clientv3.WatchResponse)) error {
	return nil
}

func (m *mockRegistryClient) Close() error {
	return nil
}

type mockIMClient struct {
	routePrivateFunc func(ctx context.Context, req *RoutePrivateMessageRequest) (*RoutePrivateMessageResponse, error)
	routeGroupFunc   func(ctx context.Context, req *RouteGroupMessageRequest) (*RouteGroupMessageResponse, error)
}

type captureAckLifecycleMetrics struct {
	mu           sync.Mutex
	pendingCount int
	successCount int
	timeoutCount int
	lateCount    int
}

type captureAckSpan struct {
	name       string
	attributes map[string]interface{}
}

func (s *captureAckSpan) End() {}

func (s *captureAckSpan) SetAttribute(key string, value interface{}) {
	s.attributes[key] = value
}

func (s *captureAckSpan) SetAttributes(attributes map[string]interface{}) {
	for k, v := range attributes {
		s.attributes[k] = v
	}
}

func (s *captureAckSpan) RecordError(err error) {
	_ = err
}

func (s *captureAckSpan) SetStatus(code tracing.StatusCode, description string) {
	_ = code
	_ = description
}

type captureAckTracer struct {
	mu    sync.Mutex
	spans []*captureAckSpan
}

func (t *captureAckTracer) StartSpan(ctx context.Context, name string, opts ...tracing.SpanOption) (context.Context, tracing.Span) {
	cfg := &tracing.SpanConfig{Attributes: make(map[string]interface{})}
	for _, opt := range opts {
		opt(cfg)
	}
	span := &captureAckSpan{name: name, attributes: cfg.Attributes}
	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()
	return ctx, span
}

func (t *captureAckTracer) Shutdown(ctx context.Context) error {
	_ = ctx
	return nil
}

func (m *captureAckLifecycleMetrics) IncrementAckPending() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingCount++
}

func (m *captureAckLifecycleMetrics) IncrementAckSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successCount++
}

func (m *captureAckLifecycleMetrics) IncrementAckTimeouts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeoutCount++
}

func (m *captureAckLifecycleMetrics) IncrementAckLate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lateCount++
}

func (m *mockIMClient) RoutePrivateMessage(ctx context.Context, req *RoutePrivateMessageRequest) (*RoutePrivateMessageResponse, error) {
	if m.routePrivateFunc != nil {
		return m.routePrivateFunc(ctx, req)
	}
	return &RoutePrivateMessageResponse{
		SequenceNumber:  12345,
		ServerTimestamp: time.Now().Unix(),
		DeliveryStatus:  "delivered",
	}, nil
}

func (m *mockIMClient) RouteGroupMessage(ctx context.Context, req *RouteGroupMessageRequest) (*RouteGroupMessageResponse, error) {
	if m.routeGroupFunc != nil {
		return m.routeGroupFunc(ctx, req)
	}
	return &RouteGroupMessageResponse{
		SequenceNumber:     12345,
		ServerTimestamp:    time.Now().Unix(),
		OnlineMemberCount:  5,
		OfflineMemberCount: 2,
	}, nil
}

// Helper function to create a test gateway service
func setupTestGateway(t *testing.T) (*GatewayService, *mockAuthClient, *mockRegistryClient, *mockIMClient) {
	t.Helper()

	authClient := &mockAuthClient{}
	registryClient := newMockRegistryClient()
	imClient := &mockIMClient{}

	config := DefaultGatewayConfig()
	config.PongWait = 1 * time.Second
	config.PingPeriod = 500 * time.Millisecond

	gateway := NewGatewayService(
		authClient,
		registryClient,
		imClient,
		nil, // Redis client not needed for basic tests
		config,
	)

	return gateway, authClient, registryClient, imClient
}

// TestNewGatewayService tests gateway service creation
func TestNewGatewayService(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	assert.NotNil(t, gateway)
	assert.NotNil(t, gateway.authClient)
	assert.NotNil(t, gateway.registryClient)
	assert.NotNil(t, gateway.imClient)
	assert.NotNil(t, gateway.pushService)
	assert.NotNil(t, gateway.cacheManager)
}

// TestHandleWebSocket_MissingToken tests WebSocket upgrade with missing token
func TestHandleWebSocket_MissingToken(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	gateway.HandleWebSocket(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing authentication token")
}

// TestHandleWebSocket_InvalidToken tests WebSocket upgrade with invalid token
func TestHandleWebSocket_InvalidToken(t *testing.T) {
	gateway, authClient, _, _ := setupTestGateway(t)

	authClient.validateFunc = func(ctx context.Context, token string) (*TokenClaims, error) {
		return nil, assert.AnError
	}

	req := httptest.NewRequest("GET", "/ws?token=invalid", nil)
	w := httptest.NewRecorder()

	gateway.HandleWebSocket(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

// TestHandleWebSocket_ValidToken tests successful WebSocket upgrade
func TestHandleWebSocket_ValidToken(t *testing.T) {
	gateway, _, registryClient, _ := setupTestGateway(t)
	gateway.config.AllowedOrigins = []string{"https://app.example.com"}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(gateway.HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=valid_token"

	// Connect as client
	headers := http.Header{}
	headers.Set("Origin", "https://app.example.com")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	require.NoError(t, err)
	defer func() { _ = ws.Close() }()

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Verify user was registered
	locations, err := registryClient.LookupUser(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", locations[0].DeviceID)
}

// TestConnection_RateLimit tests rate limiting
func TestConnection_RateLimit(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Gateway:  gateway,
	}

	// First 100 messages should pass
	for i := 0; i < 100; i++ {
		assert.True(t, connection.checkRateLimit())
	}

	// 101st message should be rate limited
	assert.False(t, connection.checkRateLimit())

	// After 1 second, rate limit should reset
	time.Sleep(1100 * time.Millisecond)
	assert.True(t, connection.checkRateLimit())
}

// TestConnection_SendAck tests sending ACK to client
func TestConnection_SendAck(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	connection.sendAck("msg-001", 12345)

	select {
	case data := <-connection.Send:
		var msg ServerMessage
		err := json.Unmarshal(data, &msg)
		require.NoError(t, err)

		assert.Equal(t, "ack", msg.Type)
		assert.Equal(t, "msg-001", msg.MsgID)
		assert.Equal(t, int64(12345), msg.SequenceNumber)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ACK")
	}
}

// TestConnection_SendError tests sending error to client
func TestConnection_SendError(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	connection.sendError("RATE_LIMIT_EXCEEDED", "Too many messages")

	select {
	case data := <-connection.Send:
		var msg ServerMessage
		err := json.Unmarshal(data, &msg)
		require.NoError(t, err)

		assert.Equal(t, "error", msg.Type)
		assert.Equal(t, "RATE_LIMIT_EXCEEDED", msg.ErrorCode)
		assert.Equal(t, "Too many messages", msg.ErrorMessage)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for error")
	}
}

// TestConnection_HandleHeartbeat tests heartbeat handling
func TestConnection_HandleHeartbeat(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	msg := &ClientMessage{
		Type: "heartbeat",
	}

	connection.handleHeartbeat(msg)

	select {
	case data := <-connection.Send:
		var serverMsg ServerMessage
		err := json.Unmarshal(data, &serverMsg)
		require.NoError(t, err)

		assert.Equal(t, "heartbeat", serverMsg.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for heartbeat response")
	}
}

// TestConnection_HandleSendMessage tests message sending
func TestConnection_HandleSendMessage(t *testing.T) {
	gateway, _, _, imClient := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Test private message
	msg := &ClientMessage{
		Type:      "send_msg",
		MsgID:     "msg-001",
		Recipient: "user_456",
		Content:   "Hello, world!",
		Timestamp: time.Now().Unix(),
	}

	var routedReq *RoutePrivateMessageRequest
	imClient.routePrivateFunc = func(ctx context.Context, req *RoutePrivateMessageRequest) (*RoutePrivateMessageResponse, error) {
		routedReq = req
		return &RoutePrivateMessageResponse{
			SequenceNumber:  12345,
			ServerTimestamp: time.Now().Unix(),
			DeliveryStatus:  "delivered",
		}, nil
	}

	connection.handleSendMessage(msg)

	// Verify message was routed
	require.NotNil(t, routedReq)
	assert.Equal(t, "msg-001", routedReq.MsgID)
	assert.Equal(t, "user123", routedReq.SenderID)
	assert.Equal(t, "user_456", routedReq.RecipientID)
	assert.Equal(t, "Hello, world!", routedReq.Content)

	// Verify ACK was sent
	select {
	case data := <-connection.Send:
		var serverMsg ServerMessage
		err := json.Unmarshal(data, &serverMsg)
		require.NoError(t, err)

		assert.Equal(t, "ack", serverMsg.Type)
		assert.Equal(t, "msg-001", serverMsg.MsgID)
		assert.Equal(t, int64(12345), serverMsg.SequenceNumber)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for ACK")
	}
}

func TestConnection_HandleSendMessage_GroupMessage(t *testing.T) {
	gateway, _, _, imClient := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	msg := &ClientMessage{
		Type:      "send_msg",
		MsgID:     "msg-group-001",
		Recipient: "group_789",
		Content:   "Hello group",
		Timestamp: time.Now().Unix(),
	}

	var routedReq *RouteGroupMessageRequest
	imClient.routeGroupFunc = func(ctx context.Context, req *RouteGroupMessageRequest) (*RouteGroupMessageResponse, error) {
		routedReq = req
		return &RouteGroupMessageResponse{
			SequenceNumber:     888,
			ServerTimestamp:    time.Now().Unix(),
			OnlineMemberCount:  2,
			OfflineMemberCount: 1,
		}, nil
	}

	connection.handleSendMessage(msg)

	require.NotNil(t, routedReq)
	assert.Equal(t, "msg-group-001", routedReq.MsgID)
	assert.Equal(t, "user123", routedReq.SenderID)
	assert.Equal(t, "group_789", routedReq.GroupID)

	select {
	case data := <-connection.Send:
		var serverMsg ServerMessage
		err := json.Unmarshal(data, &serverMsg)
		require.NoError(t, err)
		assert.Equal(t, "ack", serverMsg.Type)
		assert.Equal(t, "msg-group-001", serverMsg.MsgID)
		assert.Equal(t, int64(888), serverMsg.SequenceNumber)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for group ACK")
	}
}

func TestConnection_HandleSendMessage_InvalidRecipient(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	msg := &ClientMessage{
		Type:      "send_msg",
		MsgID:     "msg-invalid-001",
		Recipient: "invalid-recipient",
		Content:   "Hello",
		Timestamp: time.Now().Unix(),
	}

	connection.handleSendMessage(msg)

	select {
	case data := <-connection.Send:
		var serverMsg ServerMessage
		err := json.Unmarshal(data, &serverMsg)
		require.NoError(t, err)
		assert.Equal(t, "error", serverMsg.Type)
		assert.Equal(t, "INVALID_RECIPIENT", serverMsg.ErrorCode)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for error response")
	}
}

// TestConnection_Close tests connection cleanup
func TestConnection_Close(t *testing.T) {
	gateway, _, registryClient, _ := setupTestGateway(t)
	gateway.config.AllowedOrigins = []string{"https://app.example.com"}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(gateway.HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?token=valid_token"

	// Connect as client
	headers := http.Header{}
	headers.Set("Origin", "https://app.example.com")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	require.NoError(t, err)

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Verify user was registered
	locations, err := registryClient.LookupUser(context.Background(), "user123")
	require.NoError(t, err)
	assert.Len(t, locations, 1)

	// Close connection
	_ = ws.Close()

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify user was unregistered
	locations, _ = registryClient.LookupUser(context.Background(), "user123")
	assert.Len(t, locations, 0)

	// Verify connection was removed
	_, exists := gateway.connections.Load("user123_550e8400-e29b-41d4-a716-446655440000")
	assert.False(t, exists)
}

// TestGatewayService_Shutdown tests graceful shutdown
func TestGatewayService_Shutdown(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	// Add some connections
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	conn1 := &Connection{
		UserID:   "user1",
		DeviceID: "550e8400-e29b-41d4-a716-446655440001",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx1,
		cancel:   cancel1,
	}
	gateway.connections.Store("user1_550e8400-e29b-41d4-a716-446655440001", conn1)

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	conn2 := &Connection{
		UserID:   "user2",
		DeviceID: "550e8400-e29b-41d4-a716-446655440002",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx2,
		cancel:   cancel2,
	}
	gateway.connections.Store("user2_550e8400-e29b-41d4-a716-446655440002", conn2)

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	err := gateway.Shutdown(shutdownCtx)
	assert.NoError(t, err)

	// Verify all connections were closed
	count := 0
	gateway.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count)
}

// TestDefaultGatewayConfig tests default configuration
func TestDefaultGatewayConfig(t *testing.T) {
	config := DefaultGatewayConfig()

	assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 90*time.Second, config.HeartbeatTimeout)
	assert.Equal(t, 4096, config.ReadBufferSize)
	assert.Equal(t, 4096, config.WriteBufferSize)
	assert.Equal(t, int64(10*1024), config.MaxMessageSize)
	assert.Equal(t, 10*time.Second, config.WriteWait)
	assert.Equal(t, 60*time.Second, config.PongWait)
	assert.Equal(t, 54*time.Second, config.PingPeriod)
	assert.Equal(t, 90*time.Second, config.RegistryTTL)
	assert.Equal(t, 30*time.Second, config.RegistryRenewInterval)
	assert.Equal(t, 100, config.MaxMessagesPerSecond)
}

// TestGetGatewayNodeID tests gateway node ID generation
func TestGetGatewayNodeID(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	nodeID := gateway.getGatewayNodeID()
	assert.NotEmpty(t, nodeID)
}

func TestGetGatewayNodeID_UsesEnvOverride(t *testing.T) {
	t.Setenv("GATEWAY_NODE_ID", "gateway-node-test")

	gateway, _, _, _ := setupTestGateway(t)

	nodeID := gateway.getGatewayNodeID()
	assert.Equal(t, "gateway-node-test", nodeID)
}

func TestGetGatewayNodeID_FallbackUsesHostname(t *testing.T) {
	t.Setenv("GATEWAY_NODE_ID", "")

	gateway, _, _, _ := setupTestGateway(t)

	nodeID := gateway.getGatewayNodeID()
	hostname, err := os.Hostname()
	require.NoError(t, err)
	assert.Equal(t, "gateway-"+hostname, nodeID)
}

func TestGatewayService_CheckOrigin_Allowed(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AllowedOrigins = []string{"https://app.example.com"}

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://app.example.com")

	assert.True(t, gateway.upgrader.CheckOrigin(req))
}

func TestGatewayService_CheckOrigin_Disallowed(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AllowedOrigins = []string{"https://app.example.com"}

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")

	assert.False(t, gateway.upgrader.CheckOrigin(req))
}

func TestGatewayService_CheckOrigin_EmptyOriginPolicy(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AllowedOrigins = []string{"https://app.example.com"}

	req := httptest.NewRequest("GET", "/ws", nil)
	assert.False(t, gateway.upgrader.CheckOrigin(req))

	gateway.config.AllowEmptyOrigin = true
	assert.True(t, gateway.upgrader.CheckOrigin(req))
}

func TestGatewayService_AckLifecycle_Delivered(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AckTimeout = 50 * time.Millisecond

	gateway.registerPendingAck("msg-ack-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, "pending", gateway.getAckStatus("msg-ack-1", "user123", "550e8400-e29b-41d4-a716-446655440000"))

	resolved := gateway.resolveAck("msg-ack-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	assert.True(t, resolved)
	assert.Equal(t, "delivered", gateway.getAckStatus("msg-ack-1", "user123", "550e8400-e29b-41d4-a716-446655440000"))
}

func TestGatewayService_AckLifecycle_Timeout(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	gateway.config.AckTimeout = 20 * time.Millisecond

	gateway.registerPendingAck("msg-ack-2", "user123", "550e8400-e29b-41d4-a716-446655440000")
	time.Sleep(40 * time.Millisecond)

	assert.Equal(t, "timeout", gateway.getAckStatus("msg-ack-2", "user123", "550e8400-e29b-41d4-a716-446655440000"))
}

func TestConnection_HandleAck_NotFound(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	connection.handleAck(&ClientMessage{Type: "ack", MsgID: "unknown-msg"})

	select {
	case data := <-connection.Send:
		var msg ServerMessage
		err := json.Unmarshal(data, &msg)
		require.NoError(t, err)
		assert.Equal(t, "error", msg.Type)
		assert.Equal(t, "ACK_NOT_FOUND", msg.ErrorCode)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for ack not found error")
	}
}

func TestConnection_HandleAck_ResolvesPending(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection := &Connection{
		UserID:   "user123",
		DeviceID: "550e8400-e29b-41d4-a716-446655440000",
		Send:     make(chan []byte, 256),
		Gateway:  gateway,
		ctx:      ctx,
		cancel:   cancel,
	}

	gateway.registerPendingAck("msg-ack-3", connection.UserID, connection.DeviceID)
	connection.handleAck(&ClientMessage{Type: "ack", MsgID: "msg-ack-3"})

	assert.Equal(t, "delivered", gateway.getAckStatus("msg-ack-3", connection.UserID, connection.DeviceID))
}

func TestGatewayService_AckLifecycleMetrics_PendingTimeoutAndSuccess(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	metrics := &captureAckLifecycleMetrics{}
	gateway.SetAckMetrics(metrics)

	gateway.config.AckTimeout = 20 * time.Millisecond

	gateway.registerPendingAck("msg-ack-m-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	time.Sleep(40 * time.Millisecond)

	gateway.registerPendingAck("msg-ack-m-2", "user123", "550e8400-e29b-41d4-a716-446655440000")
	resolved := gateway.resolveAck("msg-ack-m-2", "user123", "550e8400-e29b-41d4-a716-446655440000")
	require.True(t, resolved)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	assert.Equal(t, 2, metrics.pendingCount)
	assert.Equal(t, 1, metrics.timeoutCount)
	assert.Equal(t, 1, metrics.successCount)
	assert.Equal(t, 0, metrics.lateCount)
}

func TestGatewayService_AckLifecycleMetrics_LateAck(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	metrics := &captureAckLifecycleMetrics{}
	gateway.SetAckMetrics(metrics)

	gateway.config.AckTimeout = 20 * time.Millisecond
	gateway.registerPendingAck("msg-ack-late-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	time.Sleep(40 * time.Millisecond)

	resolved := gateway.resolveAck("msg-ack-late-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	require.True(t, resolved)
	assert.Equal(t, "delivered", gateway.getAckStatus("msg-ack-late-1", "user123", "550e8400-e29b-41d4-a716-446655440000"))

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	assert.Equal(t, 1, metrics.pendingCount)
	assert.Equal(t, 1, metrics.timeoutCount)
	assert.Equal(t, 0, metrics.successCount)
	assert.Equal(t, 1, metrics.lateCount)
}

func TestGatewayService_AckTracing_Transitions(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)
	tracer := &captureAckTracer{}
	gateway.SetTracer(tracer)

	gateway.config.AckTimeout = 20 * time.Millisecond
	gateway.registerPendingAck("msg-ack-trace-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	time.Sleep(40 * time.Millisecond)

	resolved := gateway.resolveAck("msg-ack-trace-1", "user123", "550e8400-e29b-41d4-a716-446655440000")
	require.True(t, resolved)

	tracer.mu.Lock()
	defer tracer.mu.Unlock()
	require.Len(t, tracer.spans, 3)

	assert.Equal(t, "im-gateway.ack.register", tracer.spans[0].name)
	assert.Equal(t, "pending", tracer.spans[0].attributes["ack.transition"])
	assert.Equal(t, "msg-ack-trace-1", tracer.spans[0].attributes["msg.id"])

	assert.Equal(t, "im-gateway.ack.timeout", tracer.spans[1].name)
	assert.Equal(t, "timeout", tracer.spans[1].attributes["ack.transition"])

	assert.Equal(t, "im-gateway.ack.resolve", tracer.spans[2].name)
	assert.Equal(t, "late", tracer.spans[2].attributes["ack.transition"])
}
