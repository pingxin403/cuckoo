package com.pingxin403.cuckoo.flashsale.kafka.impl;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeoutException;

import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.apache.kafka.common.TopicPartition;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.kafka.core.KafkaTemplate;

import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.DlqMessage;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * Unit tests for DlqMessageProducerImpl.
 *
 * <p>Tests DLQ message sending, error handling, and metadata preservation.
 *
 * <p>Validates: Requirements 2.5
 */
@ExtendWith(MockitoExtension.class)
class DlqMessageProducerImplTest {

  private static final String DLQ_TOPIC = "seckill-dlq";

  @Mock private KafkaTemplate<String, Object> kafkaTemplate;

  private DlqMessageProducerImpl producer;

  @BeforeEach
  void setUp() {
    producer = new DlqMessageProducerImpl(kafkaTemplate, DLQ_TOPIC);
  }

  @Test
  @DisplayName("Should send DLQ message successfully")
  void shouldSendDlqMessageSuccessfully() throws Exception {
    // Given
    DlqMessage dlqMessage = createDlqMessage("order-123", "Test error", 3);
    RecordMetadata metadata = new RecordMetadata(new TopicPartition(DLQ_TOPIC, 0), 0, 0, 0, 0, 0);
    org.springframework.kafka.support.SendResult<String, Object> kafkaResult =
        new org.springframework.kafka.support.SendResult<>(null, metadata);
    CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> future =
        CompletableFuture.completedFuture(kafkaResult);

    when(kafkaTemplate.send(any(ProducerRecord.class))).thenReturn(future);

    // When
    SendResult result = producer.send(dlqMessage);

    // Then
    assertThat(result.success()).isTrue();
    assertThat(result.orderId()).isEqualTo("order-123");
    assertThat(result.topic()).isEqualTo(DLQ_TOPIC);
    assertThat(result.partition()).isEqualTo(0);
  }

  @Test
  @DisplayName("Should use order ID as message key")
  void shouldUseOrderIdAsMessageKey() throws Exception {
    // Given
    DlqMessage dlqMessage = createDlqMessage("order-456", "Test error", 3);
    RecordMetadata metadata = new RecordMetadata(new TopicPartition(DLQ_TOPIC, 0), 0, 0, 0, 0, 0);
    org.springframework.kafka.support.SendResult<String, Object> kafkaResult =
        new org.springframework.kafka.support.SendResult<>(null, metadata);
    CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> future =
        CompletableFuture.completedFuture(kafkaResult);

    @SuppressWarnings("unchecked")
    ArgumentCaptor<ProducerRecord<String, Object>> captor =
        ArgumentCaptor.forClass(ProducerRecord.class);
    when(kafkaTemplate.send(captor.capture())).thenReturn(future);

    // When
    producer.send(dlqMessage);

    // Then
    ProducerRecord<String, Object> sentRecord = captor.getValue();
    assertThat(sentRecord.key()).isEqualTo("order-456");
    assertThat(sentRecord.topic()).isEqualTo(DLQ_TOPIC);
    assertThat(sentRecord.value()).isEqualTo(dlqMessage);
  }

  @Test
  @DisplayName("Should return failure for null message")
  void shouldReturnFailureForNullMessage() {
    // When
    SendResult result = producer.send(null);

    // Then
    assertThat(result.success()).isFalse();
    assertThat(result.errorMessage()).contains("null");
    verify(kafkaTemplate, never()).send(any(ProducerRecord.class));
  }

  @Test
  @DisplayName("Should return failure for null original message")
  void shouldReturnFailureForNullOriginalMessage() {
    // Given
    DlqMessage dlqMessage =
        new DlqMessage(null, "Test error", 3, System.currentTimeMillis(), "seckill-orders", 0, 0);

    // When
    SendResult result = producer.send(dlqMessage);

    // Then
    assertThat(result.success()).isFalse();
    assertThat(result.errorMessage()).contains("null");
    verify(kafkaTemplate, never()).send(any(ProducerRecord.class));
  }

  @Test
  @DisplayName("Should handle execution exception")
  void shouldHandleExecutionException() throws Exception {
    // Given
    DlqMessage dlqMessage = createDlqMessage("order-789", "Test error", 3);
    CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> future =
        new CompletableFuture<>();
    future.completeExceptionally(new ExecutionException("Kafka broker unavailable", null));

    when(kafkaTemplate.send(any(ProducerRecord.class))).thenReturn(future);

    // When
    SendResult result = producer.send(dlqMessage);

    // Then
    assertThat(result.success()).isFalse();
    assertThat(result.orderId()).isEqualTo("order-789");
    assertThat(result.topic()).isEqualTo(DLQ_TOPIC);
  }

  @Test
  @DisplayName("Should handle timeout exception")
  void shouldHandleTimeoutException() throws Exception {
    // Given
    DlqMessage dlqMessage = createDlqMessage("order-timeout", "Test error", 3);
    CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> future =
        new CompletableFuture<>();
    // Use ExecutionException wrapping TimeoutException since that's how CompletableFuture works
    future.completeExceptionally(new ExecutionException(new TimeoutException("Send timed out")));

    when(kafkaTemplate.send(any(ProducerRecord.class))).thenReturn(future);

    // When
    SendResult result = producer.send(dlqMessage);

    // Then
    assertThat(result.success()).isFalse();
    assertThat(result.orderId()).isEqualTo("order-timeout");
    // The error message will contain the timeout message from the cause
    assertThat(result.errorMessage()).containsIgnoringCase("timed out");
  }

  @Test
  @DisplayName("Should preserve DLQ message metadata")
  void shouldPreserveDlqMessageMetadata() throws Exception {
    // Given
    OrderMessage originalMessage =
        OrderMessage.builder()
            .orderId("order-meta")
            .userId("user-meta")
            .skuId("sku-001")
            .quantity(2)
            .timestamp(1234567890L)
            .source("APP")
            .traceId("trace-meta")
            .build();

    DlqMessage dlqMessage =
        DlqMessage.builder()
            .originalMessage(originalMessage)
            .errorMessage("Database connection failed")
            .retryCount(3)
            .timestamp(System.currentTimeMillis())
            .topic("seckill-orders")
            .partition(5)
            .offset(100)
            .build();

    RecordMetadata metadata = new RecordMetadata(new TopicPartition(DLQ_TOPIC, 0), 0, 0, 0, 0, 0);
    org.springframework.kafka.support.SendResult<String, Object> kafkaResult =
        new org.springframework.kafka.support.SendResult<>(null, metadata);
    CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> future =
        CompletableFuture.completedFuture(kafkaResult);

    @SuppressWarnings("unchecked")
    ArgumentCaptor<ProducerRecord<String, Object>> captor =
        ArgumentCaptor.forClass(ProducerRecord.class);
    when(kafkaTemplate.send(captor.capture())).thenReturn(future);

    // When
    SendResult result = producer.send(dlqMessage);

    // Then
    assertThat(result.success()).isTrue();
    ProducerRecord<String, Object> sentRecord = captor.getValue();
    DlqMessage sentMessage = (DlqMessage) sentRecord.value();

    assertThat(sentMessage.originalMessage().orderId()).isEqualTo("order-meta");
    assertThat(sentMessage.originalMessage().userId()).isEqualTo("user-meta");
    assertThat(sentMessage.errorMessage()).isEqualTo("Database connection failed");
    assertThat(sentMessage.retryCount()).isEqualTo(3);
    assertThat(sentMessage.topic()).isEqualTo("seckill-orders");
    assertThat(sentMessage.partition()).isEqualTo(5);
    assertThat(sentMessage.offset()).isEqualTo(100);
  }

  private DlqMessage createDlqMessage(String orderId, String errorMessage, int retryCount) {
    OrderMessage originalMessage =
        OrderMessage.builder()
            .orderId(orderId)
            .userId("user-" + orderId)
            .skuId("sku-001")
            .quantity(1)
            .timestamp(System.currentTimeMillis())
            .source("WEB")
            .traceId("trace-" + orderId)
            .build();

    return DlqMessage.create(originalMessage, errorMessage, retryCount, "seckill-orders", 0, 0);
  }
}
