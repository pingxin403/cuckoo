package com.pingxin403.cuckoo.flashsale.kafka;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.*;

import java.util.List;
import java.util.UUID;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.kafka.support.Acknowledgment;

import com.pingxin403.cuckoo.flashsale.model.DlqMessage;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.dto.BatchCreateResult;

/**
 * Unit tests for OrderMessageConsumer.
 *
 * <p>Tests batch consumption logic, scheduled flush, retry mechanism, and DLQ handling.
 *
 * <p>Validates: Requirements 2.3, 2.4, 2.5
 */
@ExtendWith(MockitoExtension.class)
class OrderMessageConsumerTest {

  @Mock private OrderService orderService;

  @Mock private DlqMessageProducer dlqMessageProducer;

  @Mock private Acknowledgment acknowledgment;

  private OrderMessageConsumer consumer;

  @BeforeEach
  void setUp() {
    consumer = new OrderMessageConsumer(orderService, dlqMessageProducer);
  }

  @Test
  @DisplayName("Should accumulate messages in buffer until batch size is reached")
  void shouldAccumulateMessagesInBuffer() {
    // Given
    OrderMessage message = createOrderMessage();
    ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);

    // When - add less than batch size
    for (int i = 0; i < 50; i++) {
      consumer.consume(record, acknowledgment);
    }

    // Then - buffer should have messages, no flush yet
    assertThat(consumer.getBufferSize()).isEqualTo(50);
    verify(orderService, never()).batchCreateOrders(anyList());
  }

  @Test
  @DisplayName("Should flush batch when batch size is reached")
  void shouldFlushBatchWhenBatchSizeReached() {
    // Given
    when(orderService.batchCreateOrders(anyList())).thenReturn(BatchCreateResult.success(100));

    // When - add exactly batch size messages
    for (int i = 0; i < 100; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    // Then - batch should be flushed
    assertThat(consumer.getBufferSize()).isEqualTo(0);
    verify(orderService, times(1)).batchCreateOrders(anyList());
    verify(acknowledgment, times(100)).acknowledge();
  }

  @Test
  @DisplayName("Should flush batch with correct number of messages")
  void shouldFlushBatchWithCorrectMessageCount() {
    // Given
    @SuppressWarnings("unchecked")
    ArgumentCaptor<List<OrderMessage>> captor = ArgumentCaptor.forClass(List.class);
    when(orderService.batchCreateOrders(anyList())).thenReturn(BatchCreateResult.success(100));

    // When - add exactly batch size messages
    for (int i = 0; i < 100; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    // Then - verify batch contains 100 messages
    verify(orderService).batchCreateOrders(captor.capture());
    assertThat(captor.getValue()).hasSize(100);
  }

  @Test
  @DisplayName("Should flush partial batch on scheduled flush")
  void shouldFlushPartialBatchOnScheduledFlush() {
    // Given
    when(orderService.batchCreateOrders(anyList())).thenReturn(BatchCreateResult.success(50));

    // Add 50 messages (less than batch size)
    for (int i = 0; i < 50; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    assertThat(consumer.getBufferSize()).isEqualTo(50);

    // When - trigger scheduled flush
    consumer.scheduledFlush();

    // Then - buffer should be empty
    assertThat(consumer.getBufferSize()).isEqualTo(0);
    verify(orderService, times(1)).batchCreateOrders(anyList());
    verify(acknowledgment, times(50)).acknowledge();
  }

  @Test
  @DisplayName("Should not flush on scheduled flush when buffer is empty")
  void shouldNotFlushWhenBufferIsEmpty() {
    // When - trigger scheduled flush with empty buffer
    consumer.scheduledFlush();

    // Then - no flush should occur
    verify(orderService, never()).batchCreateOrders(anyList());
  }

  @Test
  @DisplayName("Should handle null message gracefully")
  void shouldHandleNullMessageGracefully() {
    // Given
    ConsumerRecord<String, OrderMessage> record =
        new ConsumerRecord<>("seckill-orders", 0, 0, "key", null);

    // When
    consumer.consume(record, acknowledgment);

    // Then - should acknowledge and not add to buffer
    assertThat(consumer.getBufferSize()).isEqualTo(0);
    verify(acknowledgment, times(1)).acknowledge();
    verify(orderService, never()).batchCreateOrders(anyList());
  }

  @Test
  @DisplayName("Should acknowledge messages even when batch creation fails")
  void shouldAcknowledgeMessagesWhenBatchCreationFails() {
    // Given
    when(orderService.batchCreateOrders(anyList()))
        .thenThrow(new RuntimeException("Database error"));
    // DLQ producer will be called after max retries - need to stub it
    lenient()
        .when(dlqMessageProducer.send(any(DlqMessage.class)))
        .thenReturn(SendResult.success("order-id", "seckill-dlq", 0, 0));

    // When - add batch size messages
    for (int i = 0; i < 100; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    // Then - messages should still be acknowledged to avoid infinite retry
    assertThat(consumer.getBufferSize()).isEqualTo(0);
    // All messages acknowledged
    verify(acknowledgment, times(100)).acknowledge();
  }

  @Test
  @DisplayName("Should handle partial success in batch creation")
  void shouldHandlePartialSuccessInBatchCreation() {
    // Given
    BatchCreateResult partialResult =
        BatchCreateResult.partial(
            100, 95, List.of("order-1", "order-2", "order-3", "order-4", "order-5"));
    when(orderService.batchCreateOrders(anyList())).thenReturn(partialResult);

    // When - add batch size messages
    for (int i = 0; i < 100; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    // Then - all messages should be acknowledged
    assertThat(consumer.getBufferSize()).isEqualTo(0);
    verify(acknowledgment, times(100)).acknowledge();
  }

  @Test
  @DisplayName("Should return correct batch size constant")
  void shouldReturnCorrectBatchSize() {
    assertThat(consumer.getBatchSize()).isEqualTo(100);
  }

  @Test
  @DisplayName("Should return correct max retry count constant")
  void shouldReturnCorrectMaxRetryCount() {
    assertThat(consumer.getMaxRetryCount()).isEqualTo(3);
  }

  @Test
  @DisplayName("Should flush multiple batches when receiving more than batch size")
  void shouldFlushMultipleBatches() {
    // Given
    when(orderService.batchCreateOrders(anyList())).thenReturn(BatchCreateResult.success(100));

    // When - add 250 messages (2 full batches + 50 remaining)
    for (int i = 0; i < 250; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }

    // Then - 2 batches should be flushed, 50 remaining in buffer
    assertThat(consumer.getBufferSize()).isEqualTo(50);
    verify(orderService, times(2)).batchCreateOrders(anyList());
    verify(acknowledgment, times(200)).acknowledge();
  }

  @Test
  @DisplayName("Should send message to DLQ after max retries exceeded")
  void shouldSendToDlqAfterMaxRetries() {
    // Given
    String failedOrderId = "failed-order-123";
    OrderMessage failedMessage =
        OrderMessage.builder()
            .orderId(failedOrderId)
            .userId("user-123")
            .skuId("sku-001")
            .quantity(1)
            .timestamp(System.currentTimeMillis())
            .source("WEB")
            .traceId("trace-123")
            .build();

    // Simulate partial failure where this specific order always fails
    BatchCreateResult partialResult = BatchCreateResult.partial(100, 99, List.of(failedOrderId));
    when(orderService.batchCreateOrders(anyList())).thenReturn(partialResult);
    when(dlqMessageProducer.send(any(DlqMessage.class)))
        .thenReturn(SendResult.success(failedOrderId, "seckill-dlq", 0, 0));

    // When - process the same failed message 3 times (max retries)
    for (int retry = 0; retry < 3; retry++) {
      // Add 99 successful messages + 1 failing message
      for (int i = 0; i < 99; i++) {
        OrderMessage message = createOrderMessage();
        ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
        consumer.consume(record, acknowledgment);
      }
      ConsumerRecord<String, OrderMessage> failedRecord = createConsumerRecord(failedMessage);
      consumer.consume(failedRecord, acknowledgment);
    }

    // Then - DLQ producer should be called once for the failed message after 3 retries
    ArgumentCaptor<DlqMessage> dlqCaptor = ArgumentCaptor.forClass(DlqMessage.class);
    verify(dlqMessageProducer, times(1)).send(dlqCaptor.capture());

    DlqMessage sentDlqMessage = dlqCaptor.getValue();
    assertThat(sentDlqMessage.originalMessage().orderId()).isEqualTo(failedOrderId);
    assertThat(sentDlqMessage.retryCount()).isEqualTo(3);
  }

  @Test
  @DisplayName("Should track retry count for failed messages")
  void shouldTrackRetryCountForFailedMessages() {
    // Given
    String failedOrderId = "failed-order-456";
    OrderMessage failedMessage =
        OrderMessage.builder()
            .orderId(failedOrderId)
            .userId("user-456")
            .skuId("sku-001")
            .quantity(1)
            .timestamp(System.currentTimeMillis())
            .source("WEB")
            .traceId("trace-456")
            .build();

    BatchCreateResult partialResult = BatchCreateResult.partial(100, 99, List.of(failedOrderId));
    when(orderService.batchCreateOrders(anyList())).thenReturn(partialResult);

    // When - process the failed message once
    for (int i = 0; i < 99; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }
    ConsumerRecord<String, OrderMessage> failedRecord = createConsumerRecord(failedMessage);
    consumer.consume(failedRecord, acknowledgment);

    // Then - retry count should be 1
    assertThat(consumer.getRetryCount(failedOrderId)).isEqualTo(1);

    // Process again
    for (int i = 0; i < 99; i++) {
      OrderMessage message = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
      consumer.consume(record, acknowledgment);
    }
    consumer.consume(failedRecord, acknowledgment);

    // Then - retry count should be 2
    assertThat(consumer.getRetryCount(failedOrderId)).isEqualTo(2);
  }

  @Test
  @DisplayName("Should clear retry count after successful processing")
  void shouldClearRetryCountAfterSuccess() {
    // Given
    String orderId = "order-789";
    OrderMessage message =
        OrderMessage.builder()
            .orderId(orderId)
            .userId("user-789")
            .skuId("sku-001")
            .quantity(1)
            .timestamp(System.currentTimeMillis())
            .source("WEB")
            .traceId("trace-789")
            .build();

    // First batch fails
    BatchCreateResult failResult = BatchCreateResult.partial(100, 99, List.of(orderId));
    when(orderService.batchCreateOrders(anyList())).thenReturn(failResult);

    // Process once - should increment retry count
    for (int i = 0; i < 99; i++) {
      OrderMessage otherMessage = createOrderMessage();
      ConsumerRecord<String, OrderMessage> record = createConsumerRecord(otherMessage);
      consumer.consume(record, acknowledgment);
    }
    ConsumerRecord<String, OrderMessage> record = createConsumerRecord(message);
    consumer.consume(record, acknowledgment);

    assertThat(consumer.getRetryCount(orderId)).isEqualTo(1);

    // Second batch succeeds
    when(orderService.batchCreateOrders(anyList())).thenReturn(BatchCreateResult.success(100));

    // Process again - should clear retry count
    for (int i = 0; i < 99; i++) {
      OrderMessage otherMessage = createOrderMessage();
      ConsumerRecord<String, OrderMessage> otherRecord = createConsumerRecord(otherMessage);
      consumer.consume(otherRecord, acknowledgment);
    }
    consumer.consume(record, acknowledgment);

    // Then - retry count should be cleared
    assertThat(consumer.getRetryCount(orderId)).isEqualTo(0);
  }

  private OrderMessage createOrderMessage() {
    return OrderMessage.builder()
        .orderId(UUID.randomUUID().toString())
        .userId("user-" + UUID.randomUUID().toString().substring(0, 8))
        .skuId("sku-001")
        .quantity(1)
        .timestamp(System.currentTimeMillis())
        .source("WEB")
        .traceId(UUID.randomUUID().toString())
        .build();
  }

  private ConsumerRecord<String, OrderMessage> createConsumerRecord(OrderMessage message) {
    return new ConsumerRecord<>("seckill-orders", 0, 0, message.userId(), message);
  }
}
