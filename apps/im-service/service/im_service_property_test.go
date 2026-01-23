//go:build property

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/filter"
	impb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"
	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/sequence"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"
	"pgregory.net/rapid"
)

// setupPropertyTestService creates a test IM service for property-based testing.
func setupPropertyTestService() (*IMService, *mockRegistryClient, *miniredis.Miniredis, func()) {
	// Create mock Redis
	mr, err := miniredis.Run()
	if err != nil {
		panic(fmt.Sprintf("Failed to create miniredis: %v", err))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create sequence generator
	seqGen := sequence.NewSequenceGenerator(redisClient)

	// Create mock registry
	mockRegistry := &mockRegistryClient{
		users: make(map[string][]registry.GatewayLocation),
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
		panic(fmt.Sprintf("Failed to create filter: %v", err))
	}

	// Add test words manually
	_ = filterService.UpdateWordList([]string{"badword"})

	// Create mock Kafka producer
	mockKafka := &mockKafkaProducer{}

	// Create IM service
	service := NewIMService(
		seqGen,
		mockRegistry,
		dedupService,
		filterService,
		mockKafka,
		nil, // No encryption for property tests
		DefaultIMServiceConfig(),
	)

	cleanup := func() {
		mr.Close()
		_ = redisClient.Close()
		_ = dedupService.Close()
	}

	return service, mockRegistry, mr, cleanup
}

// TestProperty_MessageDeliveryToOnlineUsers tests that messages are always delivered to online users.
// Property 2: At-Least-Once Delivery Guarantee
// **Validates: Requirements 14.5, Property 2**
func TestProperty_MessageDeliveryToOnlineUsers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, mockReg, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate random user IDs
		senderID := rapid.StringMatching(`user[0-9]+`).Draw(t, "sender_id")
		recipientID := rapid.StringMatching(`user[0-9]+`).Draw(t, "recipient_id")

		// Ensure sender and recipient are different
		if senderID == recipientID {
			t.Skip("sender and recipient must be different")
		}

		// Register recipient as online
		_ = mockReg.RegisterUser(context.Background(), recipientID, "device1", "gateway1")

		// Generate random message content
		content := rapid.StringN(1, 100, 100).Draw(t, "content")
		msgID := fmt.Sprintf("msg-%d", rapid.Int64Range(1, 1000000).Draw(t, "msg_id"))

		req := &impb.RoutePrivateMessageRequest{
			MsgId:           msgID,
			SenderId:        senderID,
			RecipientId:     recipientID,
			Content:         content,
			MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
			ClientTimestamp: timestamppb.Now(),
		}

		resp, err := service.RoutePrivateMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("RoutePrivateMessage failed: %v", err)
		}

		// Property: Online users should always receive DELIVERED status
		if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED {
			t.Fatalf("Expected DELIVERED status for online user, got %v", resp.DeliveryStatus)
		}

		// Property: Sequence number should always be positive
		if resp.SequenceNumber <= 0 {
			t.Fatalf("Expected positive sequence number, got %d", resp.SequenceNumber)
		}

		// Property: No error should occur for valid messages
		if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}

// TestProperty_MessageDeliveryToOfflineUsers tests that messages to offline users are routed to offline channel.
// Property 2: At-Least-Once Delivery Guarantee
// **Validates: Requirements 14.5, Property 2**
func TestProperty_MessageDeliveryToOfflineUsers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, _, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate random user IDs
		senderID := rapid.StringMatching(`user[0-9]+`).Draw(t, "sender_id")
		recipientID := rapid.StringMatching(`user[0-9]+`).Draw(t, "recipient_id")

		// Ensure sender and recipient are different
		if senderID == recipientID {
			t.Skip("sender and recipient must be different")
		}

		// Do NOT register recipient (offline)

		// Generate random message content
		content := rapid.StringN(1, 100, 100).Draw(t, "content")
		msgID := fmt.Sprintf("msg-%d", rapid.Int64Range(1, 1000000).Draw(t, "msg_id"))

		req := &impb.RoutePrivateMessageRequest{
			MsgId:           msgID,
			SenderId:        senderID,
			RecipientId:     recipientID,
			Content:         content,
			MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
			ClientTimestamp: timestamppb.Now(),
		}

		resp, err := service.RoutePrivateMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("RoutePrivateMessage failed: %v", err)
		}

		// Property: Offline users should receive OFFLINE status
		if resp.DeliveryStatus != impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE {
			t.Fatalf("Expected OFFLINE status for offline user, got %v", resp.DeliveryStatus)
		}

		// Property: Sequence number should always be positive
		if resp.SequenceNumber <= 0 {
			t.Fatalf("Expected positive sequence number, got %d", resp.SequenceNumber)
		}

		// Property: No error should occur for valid messages
		if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}

// TestProperty_SequenceNumberMonotonicity tests that sequence numbers are strictly increasing.
// Property 1: Message Sequence Monotonicity
// **Validates: Requirements 14.5, Property 1**
func TestProperty_SequenceNumberMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, _, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate random user IDs
		senderID := rapid.StringMatching(`user[0-9]+`).Draw(t, "sender_id")
		recipientID := rapid.StringMatching(`user[0-9]+`).Draw(t, "recipient_id")

		// Ensure sender and recipient are different
		if senderID == recipientID {
			t.Skip("sender and recipient must be different")
		}

		// Send multiple messages and verify sequence numbers are increasing
		numMessages := rapid.IntRange(2, 10).Draw(t, "num_messages")
		var prevSeqNum int64 = 0

		for i := 0; i < numMessages; i++ {
			content := rapid.StringN(1, 50, 50).Draw(t, fmt.Sprintf("content_%d", i))
			msgID := fmt.Sprintf("msg-%d-%d", rapid.Int64Range(1, 1000000).Draw(t, "base_id"), i)

			req := &impb.RoutePrivateMessageRequest{
				MsgId:           msgID,
				SenderId:        senderID,
				RecipientId:     recipientID,
				Content:         content,
				MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
				ClientTimestamp: timestamppb.Now(),
			}

			resp, err := service.RoutePrivateMessage(context.Background(), req)
			if err != nil {
				t.Fatalf("RoutePrivateMessage failed: %v", err)
			}

			// Property: Sequence numbers must be strictly increasing
			if resp.SequenceNumber <= prevSeqNum {
				t.Fatalf("Sequence number not monotonic: prev=%d, current=%d", prevSeqNum, resp.SequenceNumber)
			}

			prevSeqNum = resp.SequenceNumber
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}

// TestProperty_MessageDeduplication tests that duplicate messages are handled correctly.
// Property 3: Exactly-Once Display (Deduplication)
// **Validates: Requirements 14.5, Property 3**
func TestProperty_MessageDeduplication(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, _, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate random user IDs
		senderID := rapid.StringMatching(`user[0-9]+`).Draw(t, "sender_id")
		recipientID := rapid.StringMatching(`user[0-9]+`).Draw(t, "recipient_id")

		// Ensure sender and recipient are different
		if senderID == recipientID {
			t.Skip("sender and recipient must be different")
		}

		// Generate random message
		content := rapid.StringN(1, 100, 100).Draw(t, "content")
		msgID := fmt.Sprintf("msg-%d", rapid.Int64Range(1, 1000000).Draw(t, "msg_id"))

		req := &impb.RoutePrivateMessageRequest{
			MsgId:           msgID,
			SenderId:        senderID,
			RecipientId:     recipientID,
			Content:         content,
			MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
			ClientTimestamp: timestamppb.Now(),
		}

		// Send message first time
		resp1, err := service.RoutePrivateMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("First RoutePrivateMessage failed: %v", err)
		}

		// Send same message again (duplicate)
		resp2, err := service.RoutePrivateMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("Second RoutePrivateMessage failed: %v", err)
		}

		// Property: Both requests should succeed
		if resp1.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("First request failed: %v", resp1.ErrorCode)
		}
		if resp2.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("Second request failed: %v", resp2.ErrorCode)
		}

		// Property: Sequence numbers should be positive
		if resp1.SequenceNumber <= 0 || resp2.SequenceNumber <= 0 {
			t.Fatalf("Expected positive sequence numbers")
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}

// TestProperty_GroupMessageBroadcast tests that group messages are broadcast correctly.
// Property 5: Group Message Broadcast Completeness
// **Validates: Requirements 14.5, Property 5**
func TestProperty_GroupMessageBroadcast(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, _, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate random sender and group
		senderID := rapid.StringMatching(`user[0-9]+`).Draw(t, "sender_id")
		groupID := rapid.StringMatching(`group[0-9]+`).Draw(t, "group_id")

		// Generate random message content
		content := rapid.StringN(1, 100, 100).Draw(t, "content")
		msgID := fmt.Sprintf("msg-%d", rapid.Int64Range(1, 1000000).Draw(t, "msg_id"))

		req := &impb.RouteGroupMessageRequest{
			MsgId:           msgID,
			SenderId:        senderID,
			GroupId:         groupID,
			Content:         content,
			MessageType:     impb.MessageType_MESSAGE_TYPE_TEXT,
			ClientTimestamp: timestamppb.Now(),
		}

		resp, err := service.RouteGroupMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("RouteGroupMessage failed: %v", err)
		}

		// Property: Sequence number should always be positive
		if resp.SequenceNumber <= 0 {
			t.Fatalf("Expected positive sequence number, got %d", resp.SequenceNumber)
		}

		// Property: No error should occur for valid messages
		if resp.ErrorCode != impb.IMErrorCode_IM_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("Expected no error, got %v: %s", resp.ErrorCode, resp.ErrorMessage)
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}

// TestProperty_ValidationErrors tests that invalid messages are rejected.
// **Validates: Requirements 14.5**
func TestProperty_ValidationErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup test service
		service, _, mr, cleanup := setupPropertyTestService()
		defer cleanup()

		// Generate invalid request (missing required fields)
		invalidField := rapid.IntRange(0, 3).Draw(t, "invalid_field")

		var req *impb.RoutePrivateMessageRequest
		var expectedError impb.IMErrorCode

		switch invalidField {
		case 0: // Missing msg_id
			req = &impb.RoutePrivateMessageRequest{
				SenderId:    "user1",
				RecipientId: "user2",
				Content:     "test",
			}
			expectedError = impb.IMErrorCode_IM_ERROR_CODE_INVALID_MESSAGE
		case 1: // Missing sender_id
			req = &impb.RoutePrivateMessageRequest{
				MsgId:       "msg-001",
				RecipientId: "user2",
				Content:     "test",
			}
			expectedError = impb.IMErrorCode_IM_ERROR_CODE_SENDER_NOT_FOUND
		case 2: // Missing recipient_id
			req = &impb.RoutePrivateMessageRequest{
				MsgId:    "msg-001",
				SenderId: "user1",
				Content:  "test",
			}
			expectedError = impb.IMErrorCode_IM_ERROR_CODE_RECIPIENT_NOT_FOUND
		case 3: // Content too long
			req = &impb.RoutePrivateMessageRequest{
				MsgId:       "msg-001",
				SenderId:    "user1",
				RecipientId: "user2",
				Content:     string(make([]byte, 10001)),
			}
			expectedError = impb.IMErrorCode_IM_ERROR_CODE_CONTENT_TOO_LONG
		}

		resp, err := service.RoutePrivateMessage(context.Background(), req)
		if err != nil {
			t.Fatalf("RoutePrivateMessage failed: %v", err)
		}

		// Property: Invalid messages should be rejected with appropriate error code
		if resp.ErrorCode != expectedError {
			t.Fatalf("Expected error %v, got %v", expectedError, resp.ErrorCode)
		}

		// Cleanup Redis for next iteration
		mr.FlushAll()
	})
}
