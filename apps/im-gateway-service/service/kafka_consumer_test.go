package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaConsumer_ProcessReadReceiptEvent_PersistWhenAllOffline(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	kafkaConsumer := &KafkaConsumer{
		gateway:     gateway,
		pushService: gateway.pushService,
		ctx:         context.Background(),
	}

	persisted := false
	kafkaConsumer.persistReadReceipt = func(ctx context.Context, event *ReadReceiptEvent) error {
		persisted = true
		require.Equal(t, "sender123", event.SenderID)
		require.Equal(t, "msg-1", event.MsgID)
		return nil
	}

	err := kafkaConsumer.processReadReceiptEvent([]byte(`{"msg_id":"msg-1","sender_id":"sender123","reader_id":"reader456","conversation_id":"conv-1","read_at":123}`))
	require.NoError(t, err)
	assert.True(t, persisted)
}

func TestKafkaConsumer_ProcessReadReceiptEvent_PersistFailureReturnsError(t *testing.T) {
	gateway, _, _, _ := setupTestGateway(t)

	kafkaConsumer := &KafkaConsumer{
		gateway:     gateway,
		pushService: gateway.pushService,
		ctx:         context.Background(),
	}

	kafkaConsumer.persistReadReceipt = func(ctx context.Context, event *ReadReceiptEvent) error {
		return errors.New("persist failed")
	}

	err := kafkaConsumer.processReadReceiptEvent([]byte(`{"msg_id":"msg-1","sender_id":"sender123","reader_id":"reader456","conversation_id":"conv-1","read_at":123}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to persist offline read receipt")
}

func TestKafkaConsumer_PersistReadReceiptOffline_NoRedisIsNoop(t *testing.T) {
	kafkaConsumer := &KafkaConsumer{
		gateway: &GatewayService{},
	}

	err := kafkaConsumer.persistReadReceiptOffline(context.Background(), &ReadReceiptEvent{
		MsgID:          "msg-1",
		SenderID:       "sender123",
		ReaderID:       "reader456",
		ConversationID: "conv-1",
		ReadAt:         time.Now().Unix(),
	})

	require.NoError(t, err)
}
