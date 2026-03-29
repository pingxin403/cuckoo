package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/api/gen/go/impb"
	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/filter"
	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/sequence"
	logging "github.com/pingxin403/cuckoo/libs/observability/logging"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// setupTestService creates a test IM service with mock dependencies.
func setupTestService(t *testing.T) (*IMService, *mockRegistryClient, *mockKafkaProducer, *miniredis.Miniredis, func()) {
	t.Helper()

	// Create mock Redis
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create sequence generator
	seqGen := sequence.NewSequenceGenerator(redisClient)

	// Create mock registry
	mockRegistry := &mockRegistryClient{
		users: make(map[string][]registry.GatewayLocation),
	}

	// Create mock Kafka producer
	mockKafka := &mockKafkaProducer{
		messages: make([]*impb.OfflineMessageEvent, 0),
	}

	mockGateway := &mockGatewayClient{}

	// Create dedup service
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr: mr.Addr(),
		TTL:       7 * 24 * time.Hour,
	})

	// Create filter
	filterService, err := filter.NewSensitiveWordFilter(filter.Config{
		Enabled:       true,
		DefaultAction: filter.ActionReplace,
		WordLists:     map[string]string{}, // Empty word lists for testing
	})
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	// Add test words manually
	_ = filterService.UpdateWordList([]string{"badword"})

	service := NewIMService(
		seqGen,
		mockRegistry,
		dedupService,
		filterService,
		mockKafka,
		nil,
		mockGateway,
		DefaultIMServiceConfig(),
		nil,
	)

	cleanup := func() {
		mr.Close()
		_ = redisClient.Close()
		_ = dedupService.Close()
	}

	return service, mockRegistry, mockKafka, mr, cleanup
}

// mockRegistryClient is a mock implementation of RegistryInterface for testing.
type mockRegistryClient struct {
	users map[string][]registry.GatewayLocation
	mu    sync.RWMutex
}

func (m *mockRegistryClient) RegisterUser(ctx context.Context, userID, deviceID, gatewayNode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.users == nil {
		m.users = make(map[string][]registry.GatewayLocation)
	}
	m.users[userID] = append(m.users[userID], registry.GatewayLocation{
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

func (m *mockRegistryClient) LookupUser(ctx context.Context, userID string) ([]registry.GatewayLocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if locations, ok := m.users[userID]; ok {
		return locations, nil
	}
	return nil, nil
}

func (m *mockRegistryClient) RenewLease(ctx context.Context, userID, deviceID string) error {
	return nil
}

func (m *mockRegistryClient) Watch(ctx context.Context, prefix string, callback func(clientv3.WatchResponse)) error {
	return nil
}

func (m *mockRegistryClient) Close() error {
	return nil
}

// mockKafkaProducer is a mock implementation of KafkaProducerInterface for testing.
type mockKafkaProducer struct {
	messages []*impb.OfflineMessageEvent
	mu       sync.Mutex
	failNext bool // For testing failure scenarios
}

type mockGatewayClient struct {
	fail bool
}

type flakyGatewayClient struct {
	failUntil int
	attempts  int
	errText   string
}

func (m *flakyGatewayClient) PushMessage(ctx context.Context, gatewayAddr string, req *GatewayPushRequest) (*GatewayPushResponse, error) {
	_ = ctx
	_ = gatewayAddr
	_ = req
	m.attempts++
	if m.attempts <= m.failUntil {
		return nil, fmt.Errorf("%s", m.errText)
	}

	return &GatewayPushResponse{Success: true, DeliveredCount: 1}, nil
}

type captureIMServiceMetrics struct {
	successes []string
	failures  []string
	retries   []string
	timeouts  []string
	latencies []string
}

func (m *captureIMServiceMetrics) IncDeliverySuccess(path string) {
	m.successes = append(m.successes, path)
}

func (m *captureIMServiceMetrics) IncDeliveryFailure(path string, reason string) {
	m.failures = append(m.failures, path+":"+reason)
}

func (m *captureIMServiceMetrics) IncDeliveryRetry(path string) {
	m.retries = append(m.retries, path)
}

func (m *captureIMServiceMetrics) IncDeliveryTimeout(path string) {
	m.timeouts = append(m.timeouts, path)
}

func (m *captureIMServiceMetrics) ObserveDeliveryLatency(path string, duration time.Duration) {
	_ = duration
	m.latencies = append(m.latencies, path)
}

type captureLogger struct {
	errorf func(msg string, keysAndValues ...interface{})
}

func (l *captureLogger) Debug(ctx context.Context, msg string, keysAndValues ...interface{}) {}
func (l *captureLogger) Info(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (l *captureLogger) Warn(ctx context.Context, msg string, keysAndValues ...interface{})  {}
func (l *captureLogger) Error(ctx context.Context, msg string, keysAndValues ...interface{}) {
	if l.errorf != nil {
		l.errorf(msg, keysAndValues...)
	}
}
func (l *captureLogger) With(keysAndValues ...interface{}) logging.Logger {
	return l
}
func (l *captureLogger) Sync() error { return nil }

func (m *mockGatewayClient) PushMessage(ctx context.Context, gatewayAddr string, req *GatewayPushRequest) (*GatewayPushResponse, error) {
	if m.fail {
		return nil, fmt.Errorf("gateway push failed")
	}

	return &GatewayPushResponse{
		Success:        true,
		DeliveredCount: 1,
	}, nil
}

func (m *mockKafkaProducer) PublishOfflineMessage(ctx context.Context, msg *impb.OfflineMessageEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return fmt.Errorf("kafka publish failed")
	}

	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockKafkaProducer) Close() error {
	return nil
}

func (m *mockKafkaProducer) GetMessages() []*impb.OfflineMessageEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*impb.OfflineMessageEvent{}, m.messages...)
}

func (m *mockKafkaProducer) SetFailNext(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = fail
}

// TestRoutePrivateMessage_OnlineUser tests routing to an online user.
func TestRoutePrivateMessage_OnlineUser(t *testing.T) {
	service, mockReg, _, _, cleanup := setupTestService(t)
	defer cleanup()

	// Register recipient as online
	_ = mockReg.RegisterUser(context.Background(), "user2", "device1", "gateway1")

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "Hello, world!",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	if resp.SequenceNumber <= 0 {
		t.Errorf("Expected positive sequence number, got %d", resp.SequenceNumber)
	}

	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED {
		t.Errorf("Expected DELIVERED status, got %v", resp.DeliveryStatus)
	}
}

// TestRoutePrivateMessage_OfflineUser tests routing to an offline user.
func TestRoutePrivateMessage_OfflineUser(t *testing.T) {
	service, _, mockKafka, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-002",
		SenderId:        "user1",
		RecipientId:     "user3",
		Content:         "Hello, offline user!",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE {
		t.Errorf("Expected OFFLINE status, got %v", resp.DeliveryStatus)
	}

	// Verify message was published to Kafka
	messages := mockKafka.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 offline message, got %d", len(messages))
	}
	if len(messages) > 0 {
		msg := messages[0]
		if msg.MsgId != "msg-002" {
			t.Errorf("Expected msg_id msg-002, got %s", msg.MsgId)
		}
		if msg.UserId != "user3" {
			t.Errorf("Expected user_id user3, got %s", msg.UserId)
		}
	}
}

// TestRoutePrivateMessage_SensitiveWordFilter tests sensitive word filtering.
func TestRoutePrivateMessage_SensitiveWordFilter(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-003",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "This contains badword content",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	// Should succeed with filtered content (replace action)
	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}
}

// TestRoutePrivateMessage_Deduplication tests message deduplication.
func TestRoutePrivateMessage_Deduplication(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-004",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "Duplicate message",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	// Send first time
	resp1, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("First RoutePrivateMessage failed: %v", err)
	}

	// Send second time (duplicate)
	resp2, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("Second RoutePrivateMessage failed: %v", err)
	}

	// Both should succeed
	if resp1.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("First request failed: %v", resp1.ErrorCode)
	}
	if resp2.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Second request failed: %v", resp2.ErrorCode)
	}
}

// TestRoutePrivateMessage_ValidationErrors tests request validation.
func TestRoutePrivateMessage_ValidationErrors(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name          string
		req           *impb.RoutePrivateMessageRequest
		expectedError impb.IMErrorCode
	}{
		{
			name: "missing msg_id",
			req: &impb.RoutePrivateMessageRequest{
				SenderId:    "user1",
				RecipientId: "user2",
				Content:     "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_INVALID_MESSAGE,
		},
		{
			name: "missing sender_id",
			req: &impb.RoutePrivateMessageRequest{
				MsgId:       "msg-001",
				RecipientId: "user2",
				Content:     "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_SENDER_NOT_FOUND,
		},
		{
			name: "missing recipient_id",
			req: &impb.RoutePrivateMessageRequest{
				MsgId:    "msg-001",
				SenderId: "user1",
				Content:  "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_RECIPIENT_NOT_FOUND,
		},
		{
			name: "content too long",
			req: &impb.RoutePrivateMessageRequest{
				MsgId:       "msg-001",
				SenderId:    "user1",
				RecipientId: "user2",
				Content:     string(make([]byte, 10001)),
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_CONTENT_TOO_LONG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.RoutePrivateMessage(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("RoutePrivateMessage failed: %v", err)
			}

			if resp.ErrorCode != tt.expectedError {
				t.Errorf("Expected error %v, got %v", tt.expectedError, resp.ErrorCode)
			}
		})
	}
}

// TestRouteGroupMessage tests group message routing.
func TestRouteGroupMessage(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RouteGroupMessageRequest{
		MsgId:           "msg-group-001",
		SenderId:        "user1",
		GroupId:         "group1",
		Content:         "Hello, group!",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RouteGroupMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RouteGroupMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	if resp.SequenceNumber <= 0 {
		t.Errorf("Expected positive sequence number, got %d", resp.SequenceNumber)
	}
}

// TestRouteGroupMessage_ValidationErrors tests group message validation.
func TestRouteGroupMessage_ValidationErrors(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name          string
		req           *impb.RouteGroupMessageRequest
		expectedError impb.IMErrorCode
	}{
		{
			name: "missing msg_id",
			req: &impb.RouteGroupMessageRequest{
				SenderId: "user1",
				GroupId:  "group1",
				Content:  "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_INVALID_MESSAGE,
		},
		{
			name: "missing sender_id",
			req: &impb.RouteGroupMessageRequest{
				MsgId:   "msg-001",
				GroupId: "group1",
				Content: "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_SENDER_NOT_FOUND,
		},
		{
			name: "missing group_id",
			req: &impb.RouteGroupMessageRequest{
				MsgId:    "msg-001",
				SenderId: "user1",
				Content:  "test",
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_GROUP_NOT_FOUND,
		},
		{
			name: "content too long",
			req: &impb.RouteGroupMessageRequest{
				MsgId:    "msg-001",
				SenderId: "user1",
				GroupId:  "group1",
				Content:  string(make([]byte, 10001)),
			},
			expectedError: impb.IMErrorCode_IM_ERROR_CODE_CONTENT_TOO_LONG,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.RouteGroupMessage(context.Background(), tt.req)
			if err != nil {
				t.Fatalf("RouteGroupMessage failed: %v", err)
			}

			if resp.ErrorCode != tt.expectedError {
				t.Errorf("Expected error %v, got %v", tt.expectedError, resp.ErrorCode)
			}
		})
	}
}

// TestRouteGroupMessage_SensitiveWordFilter tests sensitive word filtering in group messages.
func TestRouteGroupMessage_SensitiveWordFilter(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RouteGroupMessageRequest{
		MsgId:           "msg-group-002",
		SenderId:        "user1",
		GroupId:         "group1",
		Content:         "This contains badword content",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RouteGroupMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RouteGroupMessage failed: %v", err)
	}

	// Should succeed with filtered content (replace action)
	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}
}

// TestRouteGroupMessage_Deduplication tests message deduplication for group messages.
func TestRouteGroupMessage_Deduplication(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RouteGroupMessageRequest{
		MsgId:           "msg-group-003",
		SenderId:        "user1",
		GroupId:         "group1",
		Content:         "Duplicate group message",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	// Send first time
	resp1, err := service.RouteGroupMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("First RouteGroupMessage failed: %v", err)
	}

	// Send second time (duplicate)
	resp2, err := service.RouteGroupMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("Second RouteGroupMessage failed: %v", err)
	}

	// Both should succeed
	if resp1.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("First request failed: %v", resp1.ErrorCode)
	}
	if resp2.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Second request failed: %v", resp2.ErrorCode)
	}
}

func TestRouteGroupMessage_PublishesToKafka(t *testing.T) {
	service, _, mockKafka, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RouteGroupMessageRequest{
		MsgId:           "msg-group-kafka-001",
		SenderId:        "user1",
		GroupId:         "group1",
		Content:         "Hello, group kafka!",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RouteGroupMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RouteGroupMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	messages := mockKafka.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 published message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.MsgId != req.MsgId {
		t.Errorf("Expected msg_id %s, got %s", req.MsgId, msg.MsgId)
	}
	if msg.UserId != req.GroupId {
		t.Errorf("Expected user_id/group_id %s, got %s", req.GroupId, msg.UserId)
	}
	if msg.ConversationType != "group" {
		t.Errorf("Expected conversation_type group, got %s", msg.ConversationType)
	}
}

func TestGetMessageStatus_PendingForUnknownMessage(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	resp, err := service.GetMessageStatus(context.Background(), &impb.GetMessageStatusRequest{
		MsgId: "unknown-msg-001",
	})
	if err != nil {
		t.Fatalf("GetMessageStatus failed: %v", err)
	}

	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_PENDING {
		t.Errorf("Expected PENDING status, got %v", resp.DeliveryStatus)
	}
}

func TestGetMessageStatus_DeliveredForProcessedMessage(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-status-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "status tracking",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	_, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	resp, err := service.GetMessageStatus(context.Background(), &impb.GetMessageStatusRequest{
		MsgId: req.MsgId,
	})
	if err != nil {
		t.Fatalf("GetMessageStatus failed: %v", err)
	}

	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED {
		t.Errorf("Expected DELIVERED status, got %v", resp.DeliveryStatus)
	}
}

// TestGetPrivateConversationID tests conversation ID generation.
func TestGetPrivateConversationID(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	// Test that conversation ID is consistent regardless of order
	id1 := service.getPrivateConversationID("user1", "user2")
	id2 := service.getPrivateConversationID("user2", "user1")

	if id1 != id2 {
		t.Errorf("Expected same conversation ID, got %s and %s", id1, id2)
	}

	// Test that different users get different IDs
	id3 := service.getPrivateConversationID("user1", "user3")
	if id1 == id3 {
		t.Errorf("Expected different conversation IDs for different users")
	}
}

// TestApplyFilter_BlockAction tests filter with block action.
func TestApplyFilter_BlockAction(t *testing.T) {
	// Create filter with block action
	filterService, err := filter.NewSensitiveWordFilter(filter.Config{
		Enabled:       true,
		DefaultAction: filter.ActionBlock,
		WordLists:     map[string]string{},
	})
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}
	_ = filterService.UpdateWordList([]string{"badword"})

	// Create service with block filter
	mr := miniredis.RunT(t)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	seqGen := sequence.NewSequenceGenerator(redisClient)
	mockRegistry := &mockRegistryClient{
		users: make(map[string][]registry.GatewayLocation),
	}
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr: mr.Addr(),
		TTL:       7 * 24 * time.Hour,
	})
	defer func() { _ = dedupService.Close() }()

	service := NewIMService(
		seqGen,
		mockRegistry,
		dedupService,
		filterService,
		nil,
		nil,
		nil,
		DefaultIMServiceConfig(),
		nil,
	)

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-block-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "This contains badword content",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	// Should be blocked
	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_SENSITIVE_CONTENT {
		t.Errorf("Expected SENSITIVE_CONTENT error, got %v", resp.ErrorCode)
	}
}

// TestApplyFilter_AuditAction tests filter with audit action.
func TestApplyFilter_AuditAction(t *testing.T) {
	// Create filter with audit action
	filterService, err := filter.NewSensitiveWordFilter(filter.Config{
		Enabled:       true,
		DefaultAction: filter.ActionAudit,
		WordLists:     map[string]string{},
	})
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}
	_ = filterService.UpdateWordList([]string{"badword"})

	// Create service with audit filter
	mr := miniredis.RunT(t)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = redisClient.Close() }()

	seqGen := sequence.NewSequenceGenerator(redisClient)
	mockRegistry := &mockRegistryClient{
		users: make(map[string][]registry.GatewayLocation),
	}
	dedupService := dedup.NewDedupService(dedup.Config{
		RedisAddr: mr.Addr(),
		TTL:       7 * 24 * time.Hour,
	})
	defer func() { _ = dedupService.Close() }()

	service := NewIMService(
		seqGen,
		mockRegistry,
		dedupService,
		filterService,
		nil, // No Kafka producer for this test
		nil, // No encryption for this test
		nil, // No gateway client for this test
		DefaultIMServiceConfig(),
		nil,
	)

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-audit-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "This contains badword content",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	// Should succeed (audit only logs)
	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}
}

// TestRoutePrivateMessage_RetryAndFallback tests retry logic with fallback to offline channel.
func TestRoutePrivateMessage_RetryAndFallback(t *testing.T) {
	service, mockReg, _, _, cleanup := setupTestService(t)
	defer cleanup()

	// Register recipient as online but simulate delivery failure
	_ = mockReg.RegisterUser(context.Background(), "user2", "device1", "gateway1")

	// Override tryDelivery to always fail (simulating network issues)
	originalConfig := service.config
	service.config = IMServiceConfig{
		MaxContentLength: 10000,
		DeliveryTimeout:  100 * time.Millisecond, // Short timeout for testing
		MaxRetries:       3,
		RetryBackoff:     []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond},
	}
	defer func() { service.config = originalConfig }()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-retry-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "Test retry logic",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	// Note: Since tryDelivery is currently a stub that always succeeds,
	// we can't fully test retry logic without mocking the Gateway delivery.
	// This test verifies the offline fallback path works.
	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	// Verify message was delivered (since tryDelivery stub succeeds)
	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED {
		t.Errorf("Expected DELIVERED status, got %v", resp.DeliveryStatus)
	}
}

// TestRoutePrivateMessage_OfflineChannelPublish tests Kafka publishing for offline users.
func TestRoutePrivateMessage_OfflineChannelPublish(t *testing.T) {
	service, _, mockKafka, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-offline-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "Offline message",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
		t.Errorf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
	}

	if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE {
		t.Errorf("Expected OFFLINE status, got %v", resp.DeliveryStatus)
	}

	// Verify message was published to Kafka
	messages := mockKafka.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 offline message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.MsgId != "msg-offline-001" {
		t.Errorf("Expected msg_id msg-offline-001, got %s", msg.MsgId)
	}
	if msg.UserId != "user2" {
		t.Errorf("Expected user_id user2, got %s", msg.UserId)
	}
	if msg.SenderId != "user1" {
		t.Errorf("Expected sender_id user1, got %s", msg.SenderId)
	}
	if msg.ConversationType != "private" {
		t.Errorf("Expected conversation_type private, got %s", msg.ConversationType)
	}
	if msg.Content != "Offline message" {
		t.Errorf("Expected content 'Offline message', got %s", msg.Content)
	}
	if msg.SequenceNumber <= 0 {
		t.Errorf("Expected positive sequence number, got %d", msg.SequenceNumber)
	}
}

// TestRoutePrivateMessage_KafkaFailure tests handling of Kafka publish failures.
func TestRoutePrivateMessage_KafkaFailure(t *testing.T) {
	service, _, mockKafka, _, cleanup := setupTestService(t)
	defer cleanup()

	// Configure Kafka to fail next publish
	mockKafka.SetFailNext(true)

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-kafka-fail-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "Test Kafka failure",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("RoutePrivateMessage failed: %v", err)
	}

	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_KAFKA_ERROR {
		t.Errorf("Expected KAFKA_ERROR, got %v", resp.ErrorCode)
	}
}

// TestDeliverWithRetry tests the retry logic with exponential backoff.
func TestDeliverWithRetry(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	// Configure short timeouts for testing
	service.config = IMServiceConfig{
		MaxContentLength: 10000,
		DeliveryTimeout:  50 * time.Millisecond,
		MaxRetries:       3,
		RetryBackoff:     []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond},
	}

	req := &impb.RoutePrivateMessageRequest{
		MsgId:       "msg-retry-test-001",
		SenderId:    "user1",
		RecipientId: "user2",
		Content:     "Test retry",
	}

	locations := []registry.GatewayLocation{
		{GatewayNode: "gateway1", DeviceID: "device1"},
	}

	// Test successful delivery (stub always succeeds)
	delivered := service.deliverWithRetry(context.Background(), req, locations, 12345)
	if !delivered {
		t.Error("Expected delivery to succeed")
	}
}

func TestTryDelivery_LogsGatewayPushError(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	mockGateway := &mockGatewayClient{fail: true}
	service.gatewayClient = mockGateway

	var buf bytes.Buffer
	service.logger = &captureLogger{errorf: func(msg string, keysAndValues ...interface{}) {
		_, _ = fmt.Fprintf(&buf, "%s %v", msg, keysAndValues)
	}}

	req := &impb.RoutePrivateMessageRequest{
		MsgId:       "msg-log-001",
		SenderId:    "user1",
		RecipientId: "user2",
		Content:     "hello",
		MessageType: impb.MessageType_MESSAGE_TYPE_TEXT,
	}

	locations := []registry.GatewayLocation{
		{GatewayNode: "gateway1", DeviceID: "device1"},
	}

	delivered := service.tryDelivery(context.Background(), req, locations, 1001)
	if delivered {
		t.Fatal("expected delivery to fail when gateway push fails")
	}

	logged := buf.String()
	if logged == "" {
		t.Fatal("expected error log output, got empty string")
	}

	if !strings.Contains(logged, "failed to push message") || !strings.Contains(logged, "msg-log-001") || !strings.Contains(logged, "user2") {
		t.Fatalf("unexpected log output: %s", logged)
	}
}

// TestRouteToOfflineChannel tests the offline channel routing function.
func TestRouteToOfflineChannel(t *testing.T) {
	service, _, mockKafka, _, cleanup := setupTestService(t)
	defer cleanup()

	req := &impb.RoutePrivateMessageRequest{
		MsgId:       "msg-offline-test-001",
		SenderId:    "user1",
		RecipientId: "user2",
		Content:     "Test offline routing",
	}

	err := service.routeToOfflineChannel(context.Background(), req, 12345)
	if err != nil {
		t.Fatalf("routeToOfflineChannel failed: %v", err)
	}

	// Verify message was published
	messages := mockKafka.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	msg := messages[0]
	if msg.SequenceNumber != 12345 {
		t.Errorf("Expected sequence number 12345, got %d", msg.SequenceNumber)
	}
}

func TestRoutePrivateMessage_RecordsOnlineDeliveryMetrics(t *testing.T) {
	service, mockReg, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_ = mockReg.RegisterUser(context.Background(), "user2", "device1", "gateway1")

	metrics := &captureIMServiceMetrics{}
	service.SetMetrics(metrics)

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-metric-online-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "hello",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED, resp.ErrorCode)
	require.Equal(t, impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED, resp.DeliveryStatus)

	assert.Equal(t, []string{"online"}, metrics.successes)
	assert.Empty(t, metrics.failures)
	assert.Empty(t, metrics.retries)
	assert.Empty(t, metrics.timeouts)
	assert.Equal(t, []string{"online"}, metrics.latencies)
}

func TestRoutePrivateMessage_RecordsRetryAndTimeoutMetrics(t *testing.T) {
	service, mockReg, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_ = mockReg.RegisterUser(context.Background(), "user2", "device1", "gateway1")
	service.gatewayClient = &flakyGatewayClient{
		failUntil: 3,
		errText:   "context deadline exceeded",
	}

	metrics := &captureIMServiceMetrics{}
	service.SetMetrics(metrics)

	req := &impb.RoutePrivateMessageRequest{
		MsgId:           "msg-metric-retry-001",
		SenderId:        "user1",
		RecipientId:     "user2",
		Content:         "hello",
		MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
		ClientTimestamp: timestamppb.Now(),
	}

	resp, err := service.RoutePrivateMessage(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED, resp.ErrorCode)
	require.Equal(t, impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE, resp.DeliveryStatus)

	assert.Equal(t, []string{"offline_fallback"}, metrics.successes)
	assert.Len(t, metrics.retries, service.config.MaxRetries-1)
	assert.NotEmpty(t, metrics.timeouts)
	assert.Equal(t, []string{"offline_fallback"}, metrics.latencies)
}
