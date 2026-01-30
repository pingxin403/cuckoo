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

import com.pingxin403.cuckoo.flashsale.kafka.DlqMessageProducer;
import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.DlqMessage;

/**
 * 死信队列消息生产者实现 Implementation of DlqMessageProducer that sends failed messages to the dead letter
 * queue.
 *
 * <p>This implementation:
 *
 * <ul>
 *   <li>Uses KafkaTemplate for sending messages to the DLQ topic
 *   <li>Preserves original message metadata for debugging
 *   <li>Handles failures gracefully with appropriate error logging
 *   <li>Uses the original order ID as the message key for traceability
 * </ul>
 *
 * <p>Requirements: 2.5
 */
@Component
public class DlqMessageProducerImpl implements DlqMessageProducer {

  private static final Logger logger = LoggerFactory.getLogger(DlqMessageProducerImpl.class);

  /** Default timeout for synchronous send operations in seconds. */
  private static final int DEFAULT_SEND_TIMEOUT_SECONDS = 10;

  private final KafkaTemplate<String, Object> kafkaTemplate;
  private final String dlqTopic;

  /**
   * Constructs a DlqMessageProducerImpl.
   *
   * @param kafkaTemplate the Kafka template for sending messages
   * @param dlqTopic the dead letter queue topic name
   */
  public DlqMessageProducerImpl(
      KafkaTemplate<String, Object> kafkaTemplate,
      @Value("${flash-sale.kafka.dlq-topic:seckill-dlq}") String dlqTopic) {
    this.kafkaTemplate = kafkaTemplate;
    this.dlqTopic = dlqTopic;
  }

  /**
   * Sends a failed message to the dead letter queue.
   *
   * <p>The message is sent synchronously to ensure delivery. The original order ID is used as the
   * message key for easy lookup and correlation.
   *
   * @param message the DLQ message containing the original message and failure details
   * @return SendResult containing success/failure status and metadata
   */
  @Override
  public SendResult send(DlqMessage message) {
    if (message == null) {
      logger.error("Cannot send null DLQ message");
      return SendResult.failure(null, dlqTopic, "DLQ message cannot be null");
    }

    if (message.originalMessage() == null) {
      logger.error("Cannot send DLQ message with null original message");
      return SendResult.failure(null, dlqTopic, "Original message cannot be null");
    }

    String orderId = message.originalMessage().orderId();

    try {
      // Use order ID as the key for traceability
      ProducerRecord<String, Object> record = new ProducerRecord<>(dlqTopic, orderId, message);

      logger.info(
          "Sending message to DLQ: orderId={}, retryCount={}, error={}",
          orderId,
          message.retryCount(),
          message.errorMessage());

      // Send synchronously and wait for result
      org.springframework.kafka.support.SendResult<String, Object> kafkaResult =
          kafkaTemplate.send(record).get(DEFAULT_SEND_TIMEOUT_SECONDS, TimeUnit.SECONDS);

      RecordMetadata metadata = kafkaResult.getRecordMetadata();

      logger.info(
          "Successfully sent message to DLQ: orderId={}, topic={}, partition={}, offset={}",
          orderId,
          metadata.topic(),
          metadata.partition(),
          metadata.offset());

      return SendResult.success(orderId, metadata.topic(), metadata.partition(), metadata.offset());

    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      logger.error("Interrupted while sending DLQ message: orderId={}", orderId, e);
      return SendResult.failure(orderId, dlqTopic, "Send operation was interrupted");

    } catch (ExecutionException e) {
      Throwable cause = e.getCause();
      String errorMessage =
          cause != null ? cause.getMessage() : "Unknown error during DLQ message send";
      logger.error("Failed to send DLQ message: orderId={}, error={}", orderId, errorMessage, e);
      return SendResult.failure(orderId, dlqTopic, errorMessage);

    } catch (TimeoutException e) {
      logger.error(
          "Timeout while sending DLQ message: orderId={}, timeout={}s",
          orderId,
          DEFAULT_SEND_TIMEOUT_SECONDS,
          e);
      return SendResult.failure(
          orderId,
          dlqTopic,
          "Send operation timed out after " + DEFAULT_SEND_TIMEOUT_SECONDS + " seconds");

    } catch (Exception e) {
      logger.error("Unexpected error while sending DLQ message: orderId={}", orderId, e);
      return SendResult.failure(orderId, dlqTopic, "Unexpected error: " + e.getMessage());
    }
  }
}
