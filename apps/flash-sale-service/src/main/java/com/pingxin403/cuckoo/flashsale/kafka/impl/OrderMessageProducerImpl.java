package com.pingxin403.cuckoo.flashsale.kafka.impl;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import org.apache.kafka.clients.producer.ProducerRecord;
import org.apache.kafka.clients.producer.RecordMetadata;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.kafka.OrderMessageProducer;
import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * 订单消息生产者实现 Implementation of OrderMessageProducer that sends order messages to Kafka.
 *
 * <p>This implementation:
 *
 * <ul>
 *   <li>Uses KafkaTemplate for sending messages
 *   <li>Routes messages to partitions based on user_id hash
 *   <li>Handles failures gracefully with appropriate error reporting
 *   <li>Supports synchronous sending with configurable timeout
 * </ul>
 *
 * <p>Requirements: 2.1, 2.2
 */
@Component
public class OrderMessageProducerImpl implements OrderMessageProducer {

  private static final Logger logger = LoggerFactory.getLogger(OrderMessageProducerImpl.class);

  /** Default timeout for synchronous send operations in seconds. */
  private static final int DEFAULT_SEND_TIMEOUT_SECONDS = 10;

  private final KafkaTemplate<String, Object> kafkaTemplate;
  private final String orderTopic;
  private final int numPartitions;

  /**
   * Constructs an OrderMessageProducerImpl.
   *
   * @param kafkaTemplate the Kafka template for sending messages
   * @param orderTopic the topic to send order messages to
   * @param numPartitions the number of partitions in the topic (for hash calculation)
   */
  public OrderMessageProducerImpl(
      KafkaTemplate<String, Object> kafkaTemplate,
      @Value("${flash-sale.kafka.order-topic:seckill-orders}") String orderTopic,
      @Value("${flash-sale.kafka.partitions:100}") int numPartitions) {
    this.kafkaTemplate = kafkaTemplate;
    this.orderTopic = orderTopic;
    this.numPartitions = numPartitions;
  }

  /**
   * Sends an order message to Kafka synchronously.
   *
   * <p>The message is routed to a partition based on the user_id hash, ensuring that all messages
   * for the same user go to the same partition. This provides ordering guarantees for messages from
   * the same user.
   *
   * @param message the order message to send
   * @return SendResult containing success/failure status and metadata
   */
  @Override
  public SendResult send(OrderMessage message) {
    if (message == null) {
      logger.error("Cannot send null message");
      return SendResult.failure(null, "Message cannot be null");
    }

    if (message.orderId() == null || message.orderId().isBlank()) {
      logger.error("Cannot send message with null or blank orderId");
      return SendResult.failure(null, "Order ID cannot be null or blank");
    }

    if (message.userId() == null || message.userId().isBlank()) {
      logger.error(
          "Cannot send message with null or blank userId for order: {}", message.orderId());
      return SendResult.failure(message.orderId(), "User ID cannot be null or blank");
    }

    try {
      // Calculate partition based on user_id hash
      int partition = calculatePartition(message.userId());

      // Create producer record with explicit partition
      ProducerRecord<String, Object> record =
          new ProducerRecord<>(orderTopic, partition, message.orderId(), message);

      logger.debug(
          "Sending order message to Kafka: orderId={}, userId={}, partition={}",
          message.orderId(),
          message.userId(),
          partition);

      // Send synchronously and wait for result
      org.springframework.kafka.support.SendResult<String, Object> kafkaResult =
          kafkaTemplate.send(record).get(DEFAULT_SEND_TIMEOUT_SECONDS, TimeUnit.SECONDS);

      RecordMetadata metadata = kafkaResult.getRecordMetadata();

      logger.info(
          "Successfully sent order message: orderId={}, topic={}, partition={}, offset={}",
          message.orderId(),
          metadata.topic(),
          metadata.partition(),
          metadata.offset());

      return SendResult.success(
          message.orderId(), metadata.topic(), metadata.partition(), metadata.offset());

    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      logger.error("Interrupted while sending order message: orderId={}", message.orderId(), e);
      return SendResult.failure(message.orderId(), orderTopic, "Send operation was interrupted");

    } catch (ExecutionException e) {
      Throwable cause = e.getCause();
      String errorMessage =
          cause != null ? cause.getMessage() : "Unknown error during message send";
      logger.error(
          "Failed to send order message: orderId={}, error={}", message.orderId(), errorMessage, e);
      return SendResult.failure(message.orderId(), orderTopic, errorMessage);

    } catch (TimeoutException e) {
      logger.error(
          "Timeout while sending order message: orderId={}, timeout={}s",
          message.orderId(),
          DEFAULT_SEND_TIMEOUT_SECONDS,
          e);
      return SendResult.failure(
          message.orderId(),
          orderTopic,
          "Send operation timed out after " + DEFAULT_SEND_TIMEOUT_SECONDS + " seconds");

    } catch (Exception e) {
      logger.error(
          "Unexpected error while sending order message: orderId={}", message.orderId(), e);
      return SendResult.failure(
          message.orderId(), orderTopic, "Unexpected error: " + e.getMessage());
    }
  }

  /**
   * Calculates the partition for a given user ID using consistent hashing.
   *
   * <p>This ensures that all messages for the same user are routed to the same partition, providing
   * ordering guarantees. The hash function uses Math.abs to handle negative hash codes.
   *
   * @param userId the user ID to calculate partition for
   * @return the partition number (0 to numPartitions-1)
   */
  int calculatePartition(String userId) {
    // Use Math.abs with special handling for Integer.MIN_VALUE
    int hash = userId.hashCode();
    int absHash = (hash == Integer.MIN_VALUE) ? 0 : Math.abs(hash);
    return absHash % numPartitions;
  }
}
