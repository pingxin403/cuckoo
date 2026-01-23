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

// IMService implements the IM message routing service.
type IMService struct {
	impb.UnimplementedIMServiceServer
	seqGen   *sequence.SequenceGenerator
	registry *registry.RegistryClient
	dedup    *dedup.DedupService
	filter   *filter.SensitiveWordFilter
	config   IMServiceConfig
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
	registry *registry.RegistryClient,
	dedup *dedup.DedupService,
	filter *filter.SensitiveWordFilter,
	config IMServiceConfig,
) *IMService {
	return &IMService{
		seqGen:   seqGen,
		registry: registry,
		dedup:    dedup,
		filter:   filter,
		config:   config,
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
	seqNum, err := s.seqGen.GetNextSequence(ctx, "private", conversationID)
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
	locations, err := s.registry.LookupUser(ctx, req.RecipientId, "")
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
		deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_DELIVERED
		// TODO: Implement actual delivery to Gateway nodes
		// For now, mark as delivered
	} else {
		// Slow Path: Recipient is offline (Requirement 1.2, 4.1)
		deliveryStatus = impb.DeliveryStatus_DELIVERY_STATUS_OFFLINE
		// TODO: Publish to Kafka offline_msg topic
	}

	// Mark as processed in dedup set (Requirement 8.2)
	if err := s.dedup.MarkProcessed(ctx, req.MsgId, 7*24*time.Hour); err != nil {
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
	seqNum, err := s.seqGen.GetNextSequence(ctx, "group", req.GroupId)
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
	if err := s.dedup.MarkProcessed(ctx, req.MsgId, 7*24*time.Hour); err != nil {
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
	result := s.filter.Filter(content)
	if result.ContainsSensitiveWords {
		config := s.filter.GetConfig()
		switch config.Action {
		case filter.ActionBlock:
			return "", fmt.Errorf("message contains sensitive words")
		case filter.ActionReplace:
			return result.Filtered, nil
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
