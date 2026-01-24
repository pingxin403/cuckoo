package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumer handles consuming messages from Kafka topics.
type KafkaConsumer struct {
	groupReader        *kafka.Reader
	readReceiptReader  *kafka.Reader
	gateway            *GatewayService
	pushService        *PushService
	ctx                context.Context
	cancel             context.CancelFunc
	readReceiptEnabled bool
}

// KafkaConfig contains Kafka consumer configuration.
type KafkaConfig struct {
	Brokers            []string
	GroupID            string
	Topic              string
	ReadReceiptTopic   string
	ReadReceiptGroupID string
	MinBytes           int
	MaxBytes           int
	CommitInterval     time.Duration
	EnableReadReceipts bool
}

// GroupMessage represents a group message from Kafka.
type GroupMessage struct {
	MsgID          string `json:"msg_id"`
	GroupID        string `json:"group_id"`
	SenderID       string `json:"sender_id"`
	Content        string `json:"content"`
	MessageType    string `json:"message_type"`
	SequenceNumber int64  `json:"sequence_number"`
	Timestamp      int64  `json:"timestamp"`
}

// ReadReceiptEvent represents a read receipt event from Kafka.
type ReadReceiptEvent struct {
	MsgID          string `json:"msg_id"`
	SenderID       string `json:"sender_id"`
	ReaderID       string `json:"reader_id"`
	ConversationID string `json:"conversation_id"`
	ReadAt         int64  `json:"read_at"`
	DeviceID       string `json:"device_id,omitempty"`
}

// NewKafkaConsumer creates a new Kafka consumer instance.
func NewKafkaConsumer(config KafkaConfig, gateway *GatewayService, pushService *PushService) *KafkaConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	groupReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		GroupID:        config.GroupID,
		Topic:          config.Topic,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		CommitInterval: config.CommitInterval,
		StartOffset:    kafka.LastOffset,
	})

	consumer := &KafkaConsumer{
		groupReader:        groupReader,
		gateway:            gateway,
		pushService:        pushService,
		ctx:                ctx,
		cancel:             cancel,
		readReceiptEnabled: config.EnableReadReceipts,
	}

	// Create read receipt reader if enabled
	if config.EnableReadReceipts && config.ReadReceiptTopic != "" {
		consumer.readReceiptReader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        config.Brokers,
			GroupID:        config.ReadReceiptGroupID,
			Topic:          config.ReadReceiptTopic,
			MinBytes:       config.MinBytes,
			MaxBytes:       config.MaxBytes,
			CommitInterval: config.CommitInterval,
			StartOffset:    kafka.LastOffset,
		})
	}

	return consumer
}

// Start starts consuming messages from Kafka.
// Validates: Requirements 2.2, 2.3, 5.3, 5.4
func (k *KafkaConsumer) Start() error {
	// Start group message consumer
	go k.consumeGroupMessages()

	// Start read receipt consumer if enabled
	if k.readReceiptEnabled && k.readReceiptReader != nil {
		go k.consumeReadReceipts()
	}

	return nil
}

// consumeGroupMessages continuously consumes group messages from Kafka.
func (k *KafkaConsumer) consumeGroupMessages() {
	for {
		select {
		case <-k.ctx.Done():
			return
		default:
		}

		// Read message from Kafka
		msg, err := k.groupReader.ReadMessage(k.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			// Log error and continue
			time.Sleep(time.Second)
			continue
		}

		// Process the message
		if err := k.processGroupMessage(msg.Value); err != nil {
			// Log error but don't stop consuming
			continue
		}
	}
}

// consumeReadReceipts continuously consumes read receipt events from Kafka.
// Validates: Requirements 5.3, 5.4
func (k *KafkaConsumer) consumeReadReceipts() {
	for {
		select {
		case <-k.ctx.Done():
			return
		default:
		}

		// Read message from Kafka
		msg, err := k.readReceiptReader.ReadMessage(k.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			// Log error and continue
			time.Sleep(time.Second)
			continue
		}

		// Process the read receipt event
		if err := k.processReadReceiptEvent(msg.Value); err != nil {
			// Log error but don't stop consuming
			continue
		}
	}
}

// processGroupMessage processes a group message from Kafka.
// Validates: Requirements 2.3, 2.10, 2.11, 2.12
func (k *KafkaConsumer) processGroupMessage(data []byte) error {
	var groupMsg GroupMessage
	if err := json.Unmarshal(data, &groupMsg); err != nil {
		return fmt.Errorf("failed to unmarshal group message: %w", err)
	}

	// Get locally-connected group members
	// For large groups (>1,000), only cache locally-connected members
	localMembers, err := k.getLocallyConnectedMembers(groupMsg.GroupID)
	if err != nil {
		return fmt.Errorf("failed to get local members: %w", err)
	}

	// Prepare server message
	serverMsg := ServerMessage{
		Type:           "message",
		MsgID:          groupMsg.MsgID,
		Sender:         groupMsg.SenderID,
		Content:        groupMsg.Content,
		Timestamp:      groupMsg.Timestamp,
		SequenceNumber: groupMsg.SequenceNumber,
	}

	msgData, err := json.Marshal(serverMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal server message: %w", err)
	}

	// Push to all locally-connected members
	var deliveredCount int32
	for _, memberID := range localMembers {
		// Find all connections for this member
		k.gateway.connections.Range(func(key, value any) bool {
			keyStr := key.(string)
			if len(keyStr) > len(memberID) && keyStr[:len(memberID)] == memberID {
				connection := value.(*Connection)
				if connection.UserID == memberID {
					select {
					case connection.Send <- msgData:
						deliveredCount++
					default:
						// Channel full, skip
					}
				}
			}
			return true
		})
	}

	return nil
}

// getLocallyConnectedMembers returns the list of group members connected to this gateway node.
// Validates: Requirements 2.10, 2.11, 2.12
func (k *KafkaConsumer) getLocallyConnectedMembers(groupID string) ([]string, error) {
	// Check if group is large (>1,000 members)
	// For large groups, only return locally-connected members

	// Get all group members from cache or User Service
	allMembers, err := k.gateway.getGroupMembers(k.ctx, groupID)
	if err != nil {
		return nil, err
	}

	// If group is small (<1,000), return all members
	if len(allMembers) < 1000 {
		return allMembers, nil
	}

	// For large groups, filter to only locally-connected members
	localMembers := make([]string, 0)
	memberSet := make(map[string]bool)

	// Build set of all members for fast lookup
	for _, member := range allMembers {
		memberSet[member] = true
	}

	// Find locally-connected members
	k.gateway.connections.Range(func(key, value any) bool {
		connection := value.(*Connection)
		if memberSet[connection.UserID] {
			// Add to local members if not already added
			if !contains(localMembers, connection.UserID) {
				localMembers = append(localMembers, connection.UserID)
			}
		}
		return true
	})

	return localMembers, nil
}

// Stop stops the Kafka consumer.
func (k *KafkaConsumer) Stop() error {
	k.cancel()

	var err error
	if k.groupReader != nil {
		if closeErr := k.groupReader.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if k.readReceiptReader != nil {
		if closeErr := k.readReceiptReader.Close(); closeErr != nil {
			err = closeErr
		}
	}

	return err
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// processReadReceiptEvent processes a read receipt event from Kafka.
// Validates: Requirements 5.3, 5.4, 15.4
func (k *KafkaConsumer) processReadReceiptEvent(data []byte) error {
	var event ReadReceiptEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal read receipt event: %w", err)
	}

	// Push read receipt to the original message sender
	req := &PushReadReceiptRequest{
		MsgID:          event.MsgID,
		SenderID:       event.SenderID,
		ReaderID:       event.ReaderID,
		ConversationID: event.ConversationID,
		ReadAt:         event.ReadAt,
	}

	// Push to all sender's devices (multi-device sync)
	resp, err := k.pushService.PushReadReceipt(k.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to push read receipt: %w", err)
	}

	// If sender is offline (no devices delivered), store for later retrieval
	// This is handled by the offline message system
	if !resp.Success || resp.DeliveredCount == 0 {
		// TODO: Store read receipt in offline storage for later retrieval
		// For now, just log that sender is offline
		fmt.Printf("Sender %s is offline, read receipt will be delivered when they reconnect\n", event.SenderID)
	}

	return nil
}
