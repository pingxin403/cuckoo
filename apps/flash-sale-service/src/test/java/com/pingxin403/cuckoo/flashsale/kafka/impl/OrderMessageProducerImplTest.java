package com.pingxin403.cuckoo.flashsale.kafka.impl;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.concurrent.CompletableFuture;

import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.apache.kafka.common.TopicPartition;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.kafka.core.KafkaTemplate;

import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * Unit tests for OrderMessageProducerImpl.
 *
 * <p>Tests cover:
 *
 * <ul>
 *   <li>Successful message sending
 *   <li>Partition routing based on user_id hash
 *   <li>Error handling for various failure scenarios
 *   <li>Input validation
 * </ul>
 */
@ExtendWith(MockitoExtension.class)
class OrderMessageProducerImplTest {

  private static final String ORDER_TOPIC = "seckill-orders";
  private static final int NUM_PARTITIONS = 100;

  @Mock private KafkaTemplate<String, Object> kafkaTemplate;

  private OrderMessageProducerImpl producer;

  @BeforeEach
  void setUp() {
    producer = new OrderMessageProducerImpl(kafkaTemplate, ORDER_TOPIC, NUM_PARTITIONS);
  }

  @Nested
  @DisplayName("send() method tests")
  class SendTests {

    @Test
    @DisplayName("should successfully send message and return success result")
    void shouldSuccessfullySendMessage() {
      // Given
      OrderMessage message = createTestMessage("order-123", "user-456", "sku-789");
      int expectedPartition = producer.calculatePartition("user-456");

      org.springframework.kafka.support.SendResult<String, Object> kafkaSendResult =
          mockKafkaSendResult(ORDER_TOPIC, expectedPartition, 100L);
      when(kafkaTemplate.send(any(ProducerRecord.class)))
          .thenReturn(CompletableFuture.completedFuture(kafkaSendResult));

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isTrue();
      assertThat(result.orderId()).isEqualTo("order-123");
      assertThat(result.topic()).isEqualTo(ORDER_TOPIC);
      assertThat(result.partition()).isEqualTo(expectedPartition);
      assertThat(result.offset()).isEqualTo(100L);
      assertThat(result.errorMessage()).isNull();
    }

    @Test
    @DisplayName("should use correct partition based on user_id hash")
    void shouldUseCorrectPartitionBasedOnUserId() {
      // Given
      OrderMessage message = createTestMessage("order-123", "user-456", "sku-789");
      int expectedPartition = producer.calculatePartition("user-456");

      org.springframework.kafka.support.SendResult<String, Object> kafkaSendResult =
          mockKafkaSendResult(ORDER_TOPIC, expectedPartition, 100L);
      when(kafkaTemplate.send(any(ProducerRecord.class)))
          .thenReturn(CompletableFuture.completedFuture(kafkaSendResult));

      // When
      producer.send(message);

      // Then
      ArgumentCaptor<ProducerRecord<String, Object>> recordCaptor =
          ArgumentCaptor.forClass(ProducerRecord.class);
      verify(kafkaTemplate).send(recordCaptor.capture());

      ProducerRecord<String, Object> capturedRecord = recordCaptor.getValue();
      assertThat(capturedRecord.partition()).isEqualTo(expectedPartition);
      assertThat(capturedRecord.topic()).isEqualTo(ORDER_TOPIC);
      assertThat(capturedRecord.key()).isEqualTo("order-123");
    }

    @Test
    @DisplayName("should return failure result when message is null")
    void shouldReturnFailureWhenMessageIsNull() {
      // When
      SendResult result = producer.send(null);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.errorMessage()).isEqualTo("Message cannot be null");
    }

    @Test
    @DisplayName("should return failure result when orderId is null")
    void shouldReturnFailureWhenOrderIdIsNull() {
      // Given
      OrderMessage message = createTestMessage(null, "user-456", "sku-789");

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.errorMessage()).isEqualTo("Order ID cannot be null or blank");
    }

    @Test
    @DisplayName("should return failure result when orderId is blank")
    void shouldReturnFailureWhenOrderIdIsBlank() {
      // Given
      OrderMessage message = createTestMessage("  ", "user-456", "sku-789");

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.errorMessage()).isEqualTo("Order ID cannot be null or blank");
    }

    @Test
    @DisplayName("should return failure result when userId is null")
    void shouldReturnFailureWhenUserIdIsNull() {
      // Given
      OrderMessage message = createTestMessage("order-123", null, "sku-789");

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.orderId()).isEqualTo("order-123");
      assertThat(result.errorMessage()).isEqualTo("User ID cannot be null or blank");
    }

    @Test
    @DisplayName("should return failure result when userId is blank")
    void shouldReturnFailureWhenUserIdIsBlank() {
      // Given
      OrderMessage message = createTestMessage("order-123", "", "sku-789");

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.orderId()).isEqualTo("order-123");
      assertThat(result.errorMessage()).isEqualTo("User ID cannot be null or blank");
    }

    @Test
    @DisplayName("should return failure result when Kafka send fails")
    @SuppressWarnings("unchecked")
    void shouldReturnFailureWhenKafkaSendFails() {
      // Given
      OrderMessage message = createTestMessage("order-123", "user-456", "sku-789");
      CompletableFuture<org.springframework.kafka.support.SendResult<String, Object>> failedFuture =
          new CompletableFuture<>();
      failedFuture.completeExceptionally(new RuntimeException("Kafka broker unavailable"));
      when(kafkaTemplate.send(any(ProducerRecord.class))).thenReturn(failedFuture);

      // When
      SendResult result = producer.send(message);

      // Then
      assertThat(result.success()).isFalse();
      assertThat(result.orderId()).isEqualTo("order-123");
      assertThat(result.topic()).isEqualTo(ORDER_TOPIC);
      assertThat(result.errorMessage()).contains("Kafka broker unavailable");
    }
  }

  @Nested
  @DisplayName("calculatePartition() method tests")
  class CalculatePartitionTests {

    @Test
    @DisplayName("should return consistent partition for same userId")
    void shouldReturnConsistentPartitionForSameUserId() {
      // Given
      String userId = "user-123";

      // When
      int partition1 = producer.calculatePartition(userId);
      int partition2 = producer.calculatePartition(userId);
      int partition3 = producer.calculatePartition(userId);

      // Then
      assertThat(partition1).isEqualTo(partition2).isEqualTo(partition3);
    }

    @Test
    @DisplayName("should return partition within valid range")
    void shouldReturnPartitionWithinValidRange() {
      // Given
      String[] userIds = {"user-1", "user-2", "user-abc", "user-xyz", "12345"};

      // When & Then
      for (String userId : userIds) {
        int partition = producer.calculatePartition(userId);
        assertThat(partition)
            .as("Partition for userId '%s' should be in range [0, %d)", userId, NUM_PARTITIONS)
            .isGreaterThanOrEqualTo(0)
            .isLessThan(NUM_PARTITIONS);
      }
    }

    @Test
    @DisplayName("should handle edge case of Integer.MIN_VALUE hash")
    void shouldHandleIntegerMinValueHash() {
      // This test ensures we handle the edge case where hashCode() returns Integer.MIN_VALUE
      // Math.abs(Integer.MIN_VALUE) returns Integer.MIN_VALUE (overflow), so we need special
      // handling

      // When
      int partition = producer.calculatePartition("polygenelubricants");

      // Then
      assertThat(partition).isGreaterThanOrEqualTo(0).isLessThan(NUM_PARTITIONS);
    }

    @Test
    @DisplayName("should distribute different userIds across partitions")
    void shouldDistributeDifferentUserIdsAcrossPartitions() {
      // Given - generate many user IDs
      int[] partitionCounts = new int[NUM_PARTITIONS];
      int numUsers = 1000;

      // When
      for (int i = 0; i < numUsers; i++) {
        String userId = "user-" + i;
        int partition = producer.calculatePartition(userId);
        partitionCounts[partition]++;
      }

      // Then - verify distribution (at least some partitions should be used)
      int usedPartitions = 0;
      for (int count : partitionCounts) {
        if (count > 0) {
          usedPartitions++;
        }
      }
      // With 1000 users and 100 partitions, we expect reasonable distribution
      assertThat(usedPartitions).isGreaterThan(50);
    }
  }

  // Helper methods

  private OrderMessage createTestMessage(String orderId, String userId, String skuId) {
    return new OrderMessage(
        orderId, userId, skuId, 1, System.currentTimeMillis(), "WEB", "trace-123");
  }

  @SuppressWarnings("unchecked")
  private org.springframework.kafka.support.SendResult<String, Object> mockKafkaSendResult(
      String topic, int partition, long offset) {
    org.springframework.kafka.support.SendResult<String, Object> sendResult =
        mock(org.springframework.kafka.support.SendResult.class);
    RecordMetadata metadata =
        new RecordMetadata(
            new TopicPartition(topic, partition),
            0L,
            (int) offset,
            System.currentTimeMillis(),
            0,
            0);
    when(sendResult.getRecordMetadata()).thenReturn(metadata);
    return sendResult;
  }
}
