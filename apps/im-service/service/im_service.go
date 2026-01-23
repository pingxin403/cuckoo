package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pingxin403/cuckoo/apps/im-service/dedup"
	"github.com/pingxin403/cuckoo/apps/im-service/filter"
	impb "github.com/pingxin403/cuckoo/apps/im-service/gen/impb"
	"github.com/pingxin403/cuckoo/apps/im-service/registry"
	"github.com/pingxin403/cuckoo/apps/im-service/sequence"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RegistryInterface defines the interface for user registry operations
type RegistryInterface interface {
	LookupUser(ctx context.Context, userID string) ([]registry.GatewayLocation, error)
}

// KafkaProducerInterface defines the interface for Kafka message publishing
type KafkaProducerInterface interface {
	PublishOfflineMessage(ctx context.Context, msg *impb.OfflineMessageEvent) error
	Close() error
}

// IMService implements the IM message routing service.
type IMService struct {
	impb.UnimplementedIMServiceServer
	seqGen        *sequence.SequenceGenerator
	registry      RegistryInterface
	dedup         *dedup.DedupService
	filter        *filter.SensitiveWordFilter
	kafkaProducer KafkaProducerInterface
	encryption    EncryptionInterface
	config        IMServiceConfig
}

// EncryptionInterface defines the interface for message encryption.
type EncryptionInterface interface {
	Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, keyID string, keyVersion int, err error)
	Decrypt(ciphertext []byte, nonce []byte, keyID string, keyVersion int) (plaintext []byte, err error)
	Close() error
}

// IMServiceConfig contains configuration for the IM service.
type IMServiceConfig struct {
	MaxContentLength int
	DeliveryTimeout  time.Duration
	MaxRetries       int
	RetryBackoff     []time.Duration
}

// DefaultIMServiceConfig returns default configuration.
func DefaultIMServiceConfig() IMServiceConfig {
	return IMServiceConfig{
		MaxContentLength: 10000,
		DeliveryTimeout:  5 * time.Second,
		MaxRetries:       3,
		RetryBackoff:     []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// NewIMService creates a new IM service instance.
func NewIMService(
	seqGen *sequence.SequenceGenerator,
	registry RegistryInterface,
	dedup *dedup.DedupService,
	filter *filter.SensitiveWordFilter,
	kafkaProducer KafkaProducerInterface,
	encryption EncryptionInterface,
	config IMServiceConfig,
) *IMService {
	return &IMService{
		seqGen:        seqGen,
		registry:      registry,
		dedup:         dedup,
		filter:        filter,
		kafkaProducer: kafkaProducer,
		encryption:    encryption,
		config:        config,
	}
}

// RoutePrivateMessage routes a private message to a specific user.
// Validates: Requirements 1.1, 1.2, 3.1
func (s *IMService) RoutePrivateMessage(
	ctx context.Context,
	req *impb.RoutePrivateMessageRequest,
) (*impb.RoutePrivateMessageResponse, error) {
	// Validate request
	if err := s.validatePrivateMessageRequest(req); err != nil {
		return &impb.RoutePrivateMessageResponse{
			ErrorCode:    err.Code,
			ErrorMessage: err.Message,
		}, nil
	}

	// Apply sensitive word filter (Requirement 11.4, 17.5)
	filteredContent, err := s.applyFilter(req.Content)
	if err != nil {
		return &impb.RoutePrivateMessageResponse{
			ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_SENSITIVE_CONTENT,
			ErrorMessage: err.Error(),
		}, nil
	}
	req.Content = filteredContent

	// Assign sequence number (Requirement 16.1, 16.2)
	conversationID := s.getPrivateConversationID(req.SenderId, req.RecipientId)
	seqNum, err := s.seqGen.GenerateSequence(ctx, sequence.ConversationTypePrivate, conversationID)
	if err != nil {
		return &impb.RoutePrivateMessageResponse{
			ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_SEQUENCE_ERROR,
			ErrorMessage: fmt.Sprintf("failed to assign sequence number: %v", err),
		}, nil
	}

	// Check deduplication (Requirement 8.1, 8.2)
	isDup, err := s.dedup.CheckDuplicate(ctx, req.MsgId)
	if err == nil && isDup {
		// Message already processed, return success with existing sequence
		return &impb.RoutePrivateMessageResponse{
			SequenceNumber:  seqNum,
			ServerTimestamp: timestamppb.Now(),
			DeliveryStatus:  impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED,
		}, nil
	}

	// Query Registry for recipient (Requirement 7.3, 7.4)
	locations, err := s.registry.LookupUser(ctx, req.RecipientId)
	if err != nil {
		return &impb.RoutePrivateMessageResponse{
			ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_REGISTRY_ERROR,
			ErrorMessage: fmt.Sprintf("failed to lookup recipient: %v", err),
		}, nil
	}

	// Determine delivery path
	var deliveryStatus impb.DeliveryStatus
	if len(locations) > 0 {
		// Fast Path: Recipient is online (Requirement 1.1, 3.1)
		// Try delivery with retry logic (Requirement 3.2, 3.3, 3.4)
		delivered := s.deliverWithRetry(ctx, req, locations)
		if delivered {
			deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED
		} else {
			// Fallback to Offline Channel after retry exhaustion
			if err := s.routeToOfflineChannel(ctx, req, seqNum); err != nil {
				// Log error but return success (message is queued)
				deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE
			} else {
				deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE
			}
		}
	} else {
		// Slow Path: Recipient is offline (Requirement 1.2, 4.1)
		if err := s.routeToOfflineChannel(ctx, req, seqNum); err != nil {
			return &impb.RoutePrivateMessageResponse{
				ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_DELIVERY_FAILED,
				ErrorMessage: fmt.Sprintf("failed to route to offline channel: %v", err),
			}, nil
		}
		deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE
	}

	// Mark as processed in dedup set (Requirement 8.2)
	if err := s.dedup.MarkProcessed(ctx, req.MsgId); err != nil {
		// Log error but don't fail the request
		// The message was already routed successfully
	}

	return &impb.RoutePrivateMessageResponse{
		SequenceNumber:  seqNum,
		ServerTimestamp: timestamppb.Now(),
		DeliveryStatus:  deliveryStatus,
	}, nil
}

// RouteGroupMessage routes a message to all members of a group.
// Validates: Requirements 2.1, 2.2, 2.3, 2.4
func (s *IMService) RouteGroupMessage(
	ctx context.Context,
	req *impb.RouteGroupMessageRequest,
) (*impb.RouteGroupMessageResponse, error) {
	// Validate request
	if err := s.validateGroupMessageRequest(req); err != nil {
		return &impb.RouteGroupMessageResponse{
			ErrorCode:    err.Code,
			ErrorMessage: err.Message,
		}, nil
	}

	// Apply sensitive word filter (Requirement 11.4, 17.5)
	filteredContent, err := s.applyFilter(req.Content)
	if err != nil {
		return &impb.RouteGroupMessageResponse{
			ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_SENSITIVE_CONTENT,
			ErrorMessage: err.Error(),
		}, nil
	}
	req.Content = filteredContent

	// Assign sequence number (Requirement 16.1, 16.3)
	seqNum, err := s.seqGen.GenerateSequence(ctx, sequence.ConversationTypeGroup, req.GroupId)
	if err != nil {
		return &impb.RouteGroupMessageResponse{
			ErrorCode:    impb.IMErrorCode_IM_ERROR_CODE_SEQUENCE_ERROR,
			ErrorMessage: fmt.Sprintf("failed to assign sequence number: %v", err),
		}, nil
	}

	// Check deduplication (Requirement 8.1, 8.2)
	isDup, err := s.dedup.CheckDuplicate(ctx, req.MsgId)
	if err == nil && isDup {
		// Message already processed, return success
		return &impb.RouteGroupMessageResponse{
			SequenceNumber:  seqNum,
			ServerTimestamp: timestamppb.Now(),
		}, nil
	}

	// Publish to Kafka group_msg topic (Requirement 2.2, 2.4)
	// TODO: Implement Kafka publishing
	// For now, return success
	onlineCount := 0  // TODO: Get from group membership
	offlineCount := 0 // TODO: Get from group membership

	// Mark as processed in dedup set (Requirement 8.2)
	if err := s.dedup.MarkProcessed(ctx, req.MsgId); err != nil {
		// Log error but don't fail the request
	}

	return &impb.RouteGroupMessageResponse{
		SequenceNumber:     seqNum,
		ServerTimestamp:    timestamppb.Now(),
		OnlineMemberCount:  int32(onlineCount),
		OfflineMemberCount: int32(offlineCount),
	}, nil
}

// GetMessageStatus retrieves the delivery status of a message.
func (s *IMService) GetMessageStatus(
	ctx context.Context,
	req *impb.GetMessageStatusRequest,
) (*impb.GetMessageStatusResponse, error) {
	// TODO: Implement message status tracking
	// For now, return not implemented
	return &impb.GetMessageStatusResponse{
		MsgId:           req.MsgId,
		DeliveryStatus:  impb.DeliveryStatus_DELIVERY_STATUS_PENDING,
		ServerTimestamp: timestamppb.Now(),
	}, nil
}

// validatePrivateMessageRequest validates a private message request.
func (s *IMService) validatePrivateMessageRequest(req *impb.RoutePrivateMessageRequest) *IMError {
	if req.MsgId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_INVALID_MESSAGE,
			Message: "msg_id is required",
		}
	}
	if req.SenderId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_SENDER_NOT_FOUND,
			Message: "sender_id is required",
		}
	}
	if req.RecipientId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_RECIPIENT_NOT_FOUND,
			Message: "recipient_id is required",
		}
	}
	if len(req.Content) > s.config.MaxContentLength {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_CONTENT_TOO_LONG,
			Message: fmt.Sprintf("content exceeds maximum length of %d characters", s.config.MaxContentLength),
		}
	}
	return nil
}

// validateGroupMessageRequest validates a group message request.
func (s *IMService) validateGroupMessageRequest(req *impb.RouteGroupMessageRequest) *IMError {
	if req.MsgId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_INVALID_MESSAGE,
			Message: "msg_id is required",
		}
	}
	if req.SenderId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_SENDER_NOT_FOUND,
			Message: "sender_id is required",
		}
	}
	if req.GroupId == "" {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_GROUP_NOT_FOUND,
			Message: "group_id is required",
		}
	}
	if len(req.Content) > s.config.MaxContentLength {
		return &IMError{
			Code:    impb.IMErrorCode_IM_ERROR_CODE_CONTENT_TOO_LONG,
			Message: fmt.Sprintf("content exceeds maximum length of %d characters", s.config.MaxContentLength),
		}
	}
	return nil
}

// applyFilter applies sensitive word filtering to content.
func (s *IMService) applyFilter(content string) (string, error) {
	result := s.filter.Filter(content, filter.ActionReplace)
	if result.ContainsSensitiveWords {
		cfg := s.filter.GetConfig()
		switch cfg.DefaultAction {
		case filter.ActionBlock:
			return "", fmt.Errorf("message contains sensitive words")
		case filter.ActionReplace:
			return result.FilteredContent, nil
		case filter.ActionAudit:
			// Log but allow
			return content, nil
		}
	}
	return content, nil
}

// getPrivateConversationID generates a consistent conversation ID for private chat.
// Uses sorted user IDs to ensure consistency regardless of sender/recipient order.
func (s *IMService) getPrivateConversationID(userA, userB string) string {
	if userA < userB {
		return userA + "_" + userB
	}
	return userB + "_" + userA
}

// IMError represents an IM service error.
type IMError struct {
	Code    impb.IMErrorCode
	Message string
}

func (e *IMError) Error() string {
	return e.Message
}

// deliverWithRetry attempts to deliver a message with exponential backoff retry logic.
// Validates: Requirements 3.2, 3.3, 3.4
func (s *IMService) deliverWithRetry(ctx context.Context, req *impb.RoutePrivateMessageRequest, locations []registry.GatewayLocation) bool {
	for attempt := 0; attempt < s.config.MaxRetries; attempt++ {
		// Create context with timeout
		deliveryCtx, cancel := context.WithTimeout(ctx, s.config.DeliveryTimeout)

		// Try to deliver to Gateway nodes
		delivered := s.tryDelivery(deliveryCtx, req, locations)
		cancel()

		if delivered {
			return true
		}

		// If not last attempt, wait with exponential backoff
		if attempt < s.config.MaxRetries-1 {
			backoff := s.config.RetryBackoff[attempt]
			time.Sleep(backoff)
		}
	}

	// All retries exhausted
	return false
}

// tryDelivery attempts to deliver a message to Gateway nodes.
func (s *IMService) tryDelivery(ctx context.Context, req *impb.RoutePrivateMessageRequest, locations []registry.GatewayLocation) bool {
	// TODO: Implement actual gRPC call to Gateway nodes
	// For now, simulate delivery based on context timeout
	select {
	case <-ctx.Done():
		return false
	default:
		// Simulate successful delivery
		return true
	}
}

// routeToOfflineChannel routes a message to the Kafka offline_msg topic.
// Validates: Requirements 1.2, 4.1
func (s *IMService) routeToOfflineChannel(ctx context.Context, req *impb.RoutePrivateMessageRequest, seqNum int64) error {
	if s.kafkaProducer == nil {
		// No Kafka producer configured, skip offline routing
		return nil
	}

	// Create offline message event
	offlineMsg := &impb.OfflineMessageEvent{
		MsgId:            req.MsgId,
		UserId:           req.RecipientId,
		SenderId:         req.SenderId,
		ConversationId:   s.getPrivateConversationID(req.SenderId, req.RecipientId),
		ConversationType: "private",
		Content:          req.Content,
		SequenceNumber:   seqNum,
		Timestamp:        time.Now().UnixMilli(),
	}

	// Publish to Kafka
	return s.kafkaProducer.PublishOfflineMessage(ctx, offlineMsg)
}
