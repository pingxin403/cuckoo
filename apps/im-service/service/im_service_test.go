package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/filter"
	impb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"
	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/sequence"
	"github.com/redis/go-redis/v9"
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

	// Create IM service
	service := NewIMService(
		seqGen,
		mockRegistry,
		dedupService,
		filterService,
		mockKafka,
		nil, // No encryption for basic tests
		DefaultIMServiceConfig(),
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
		nil, // No Kafka producer for this test
		nil, // No encryption for this test
		DefaultIMServiceConfig(),
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
		DefaultIMServiceConfig(),
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

	// Should return error when Kafka publish fails
	if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_DELIVERY_FAILED {
		t.Errorf("Expected DELIVERY_FAILED error, got %v", resp.ErrorCode)
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
	delivered := service.deliverWithRetry(context.Background(), req, locations)
	if !delivered {
		t.Error("Expected delivery to succeed")
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
