package com.pingxin403.cuckoo.flashsale.kafka;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.locks.ReentrantLock;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.model.DlqMessage;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.dto.BatchCreateResult;

/**
 * 订单消息消费者 Kafka consumer for processing order messages in batches with retry and DLQ support.
 *
 * <p>This consumer implements batch processing of order messages with the following features:
 *
 * <ul>
 *   <li>Batch consumption: accumulates messages until batch size (100) is reached
 *   <li>Scheduled flush: flushes partial batches every 5 seconds
 *   <li>Manual acknowledgment: ensures reliable message processing
 *   <li>Thread-safe buffer management: uses locking for concurrent access
 *   <li>Retry mechanism: retries failed messages up to 3 times
 *   <li>Dead letter queue: sends messages to DLQ after 3 failed retries
 * </ul>
 *
 * <p>Requirements: 2.3, 2.4, 2.5
 */
@Component
public class OrderMessageConsumer {

  private static final Logger logger = LoggerFactory.getLogger(OrderMessageConsumer.class);

  /** Batch size for database writes - 100 messages per batch as per design. */
  private static final int BATCH_SIZE = 100;

  /** Maximum number of retry attempts before sending to DLQ. */
  private static final int MAX_RETRY_COUNT = 3;

  /** Buffer to accumulate messages before batch processing. */
  private final List<MessageWithMetadata> buffer = new ArrayList<>();

  /** Lock for thread-safe buffer access. */
  private final ReentrantLock bufferLock = new ReentrantLock();

  /** Tracks retry count for each message by order ID. */
  private final ConcurrentHashMap<String, Integer> retryCountMap = new ConcurrentHashMap<>();

  /** Order service for batch creating orders. */
  private final OrderService orderService;

  /** DLQ producer for sending failed messages to dead letter queue. */
  private final DlqMessageProducer dlqMessageProducer;

  /**
   * Constructs the consumer with required dependencies.
   *
   * @param orderService the order service for batch order creation
   * @param dlqMessageProducer the DLQ producer for sending failed messages
   */
  public OrderMessageConsumer(OrderService orderService, DlqMessageProducer dlqMessageProducer) {
    this.orderService = orderService;
    this.dlqMessageProducer = dlqMessageProducer;
  }

  /**
   * Consumes order messages from Kafka.
   *
   * <p>Messages are accumulated in a buffer until the batch size is reached, then flushed to the
   * database. Manual acknowledgment is used to ensure reliable processing.
   *
   * @param record the Kafka consumer record containing the order message
   * @param acknowledgment the acknowledgment for manual commit
   */
  @KafkaListener(
      topics = "${flash-sale.kafka.order-topic:seckill-orders}",
      groupId = "${spring.kafka.consumer.group-id:flash-sale-consumer}",
      containerFactory = "kafkaListenerContainerFactory")
  public void consume(ConsumerRecord<String, OrderMessage> record, Acknowledgment acknowledgment) {
    OrderMessage message = record.value();

    if (message == null) {
      logger.warn(
          "Received null message from partition {} offset {}", record.partition(), record.offset());
      acknowledgment.acknowledge();
      return;
    }

    logger.debug(
        "Received order message: orderId={}, userId={}, skuId={}",
        message.orderId(),
        message.userId(),
        message.skuId());

    bufferLock.lock();
    try {
      MessageWithMetadata messageWithMetadata =
          new MessageWithMetadata(
              message, acknowledgment, record.topic(), record.partition(), record.offset());
      buffer.add(messageWithMetadata);

      if (buffer.size() >= BATCH_SIZE) {
        flushBatch();
      }
    } catch (Exception e) {
      logger.error("Error processing message: orderId={}", message.orderId(), e);
      // Handle the error with retry logic
      handleMessageError(message, e, record, acknowledgment);
    } finally {
      bufferLock.unlock();
    }
  }

  /**
   * Scheduled task to flush partial batches.
   *
   * <p>This ensures that messages don't sit in the buffer indefinitely when traffic is low. Runs
   * every 5 seconds as per design specification.
   */
  @Scheduled(fixedRate = 5000)
  public void scheduledFlush() {
    bufferLock.lock();
    try {
      if (!buffer.isEmpty()) {
        logger.debug("Scheduled flush triggered with {} messages in buffer", buffer.size());
        flushBatch();
      }
    } finally {
      bufferLock.unlock();
    }
  }

  /**
   * Flushes the current buffer to the database with retry support.
   *
   * <p>This method:
   *
   * <ul>
   *   <li>Creates a copy of the buffer for processing
   *   <li>Calls orderService.batchCreateOrders() to persist orders
   *   <li>Handles failures with retry logic
   *   <li>Sends to DLQ after max retries exceeded
   *   <li>Acknowledges all messages after processing
   *   <li>Clears the buffer
   * </ul>
   *
   * <p>Must be called while holding the bufferLock.
   */
  private void flushBatch() {
    if (buffer.isEmpty()) {
      return;
    }

    int batchSize = buffer.size();
    List<MessageWithMetadata> batchToProcess = new ArrayList<>(buffer);

    logger.info("Flushing batch of {} order messages to database", batchSize);

    try {
      List<OrderMessage> messages =
          batchToProcess.stream().map(MessageWithMetadata::message).toList();

      BatchCreateResult result = orderService.batchCreateOrders(messages);

      if (result.isFullSuccess()) {
        logger.info("Successfully created {} orders in batch", result.successCount());
        // Clear retry counts for successful messages
        for (MessageWithMetadata m : batchToProcess) {
          retryCountMap.remove(m.message().orderId());
        }
      } else {
        logger.warn(
            "Batch creation completed with {} successes and {} failures. Failed orderIds: {}",
            result.successCount(),
            result.failedCount(),
            result.failedOrderIds());

        // Handle failed messages with retry logic
        handleFailedMessages(batchToProcess, result);
      }

      // Acknowledge all messages after processing
      for (MessageWithMetadata m : batchToProcess) {
        m.acknowledgment().acknowledge();
      }

      // Clear buffer after processing
      buffer.clear();

    } catch (Exception e) {
      logger.error("Error flushing batch of {} messages", batchSize, e);
      // Handle batch error with retry logic for all messages
      handleBatchError(batchToProcess, e);
      buffer.clear();
    }
  }

  /**
   * Handles failed messages from batch processing with retry logic.
   *
   * @param batchToProcess the batch of messages that were processed
   * @param result the batch create result containing failure information
   */
  private void handleFailedMessages(
      List<MessageWithMetadata> batchToProcess, BatchCreateResult result) {
    for (MessageWithMetadata m : batchToProcess) {
      String orderId = m.message().orderId();
      if (result.failedOrderIds().contains(orderId)) {
        int currentRetryCount = retryCountMap.getOrDefault(orderId, 0) + 1;
        retryCountMap.put(orderId, currentRetryCount);

        if (currentRetryCount >= MAX_RETRY_COUNT) {
          // Send to DLQ after max retries
          sendToDlq(
              m, "Batch creation failed after " + MAX_RETRY_COUNT + " retries", currentRetryCount);
          retryCountMap.remove(orderId);
        } else {
          logger.warn(
              "Message failed, will retry: orderId={}, retryCount={}/{}",
              orderId,
              currentRetryCount,
              MAX_RETRY_COUNT);
        }
      } else {
        // Clear retry count for successful messages
        retryCountMap.remove(orderId);
      }
    }
  }

  /**
   * Handles batch-level errors with retry logic for all messages.
   *
   * @param batchToProcess the batch of messages that failed
   * @param error the exception that caused the failure
   */
  private void handleBatchError(List<MessageWithMetadata> batchToProcess, Exception error) {
    String errorMessage = error.getMessage() != null ? error.getMessage() : "Unknown batch error";

    for (MessageWithMetadata m : batchToProcess) {
      String orderId = m.message().orderId();
      int currentRetryCount = retryCountMap.getOrDefault(orderId, 0) + 1;
      retryCountMap.put(orderId, currentRetryCount);

      if (currentRetryCount >= MAX_RETRY_COUNT) {
        // Send to DLQ after max retries
        sendToDlq(m, errorMessage, currentRetryCount);
        retryCountMap.remove(orderId);
      } else {
        logger.warn(
            "Batch error, message will retry: orderId={}, retryCount={}/{}, error={}",
            orderId,
            currentRetryCount,
            MAX_RETRY_COUNT,
            errorMessage);
      }

      // Acknowledge to avoid infinite reprocessing
      m.acknowledgment().acknowledge();
    }
  }

  /**
   * Handles individual message errors with retry logic.
   *
   * @param message the order message that failed
   * @param error the exception that caused the failure
   * @param record the original Kafka record
   * @param acknowledgment the acknowledgment for the message
   */
  private void handleMessageError(
      OrderMessage message,
      Exception error,
      ConsumerRecord<String, OrderMessage> record,
      Acknowledgment acknowledgment) {
    String orderId = message.orderId();
    int currentRetryCount = retryCountMap.getOrDefault(orderId, 0) + 1;
    retryCountMap.put(orderId, currentRetryCount);

    String errorMessage = error.getMessage() != null ? error.getMessage() : "Unknown error";

    if (currentRetryCount >= MAX_RETRY_COUNT) {
      // Send to DLQ after max retries
      DlqMessage dlqMessage =
          DlqMessage.create(
              message,
              errorMessage,
              currentRetryCount,
              record.topic(),
              record.partition(),
              record.offset());

      SendResult dlqResult = dlqMessageProducer.send(dlqMessage);
      if (dlqResult.success()) {
        logger.info("Message sent to DLQ after {} retries: orderId={}", currentRetryCount, orderId);
      } else {
        logger.error(
            "Failed to send message to DLQ: orderId={}, error={}",
            orderId,
            dlqResult.errorMessage());
      }

      retryCountMap.remove(orderId);
    } else {
      logger.warn(
          "Message error, will retry: orderId={}, retryCount={}/{}, error={}",
          orderId,
          currentRetryCount,
          MAX_RETRY_COUNT,
          errorMessage);
    }

    // Acknowledge to avoid infinite reprocessing from Kafka
    acknowledgment.acknowledge();
  }

  /**
   * Sends a failed message to the dead letter queue.
   *
   * @param messageWithMetadata the message with its metadata
   * @param errorMessage the error message describing the failure
   * @param retryCount the number of retries attempted
   */
  private void sendToDlq(
      MessageWithMetadata messageWithMetadata, String errorMessage, int retryCount) {
    DlqMessage dlqMessage =
        DlqMessage.create(
            messageWithMetadata.message(),
            errorMessage,
            retryCount,
            messageWithMetadata.topic(),
            messageWithMetadata.partition(),
            messageWithMetadata.offset());

    SendResult dlqResult = dlqMessageProducer.send(dlqMessage);
    if (dlqResult.success()) {
      logger.info(
          "Message sent to DLQ after {} retries: orderId={}",
          retryCount,
          messageWithMetadata.message().orderId());
    } else {
      logger.error(
          "Failed to send message to DLQ: orderId={}, error={}",
          messageWithMetadata.message().orderId(),
          dlqResult.errorMessage());
    }
  }

  /**
   * Returns the current buffer size for monitoring purposes.
   *
   * @return the number of messages currently in the buffer
   */
  public int getBufferSize() {
    bufferLock.lock();
    try {
      return buffer.size();
    } finally {
      bufferLock.unlock();
    }
  }

  /**
   * Returns the configured batch size.
   *
   * @return the batch size (100)
   */
  public int getBatchSize() {
    return BATCH_SIZE;
  }

  /**
   * Returns the maximum retry count.
   *
   * @return the maximum retry count (3)
   */
  public int getMaxRetryCount() {
    return MAX_RETRY_COUNT;
  }

  /**
   * Returns the current retry count for a specific order ID.
   *
   * @param orderId the order ID to check
   * @return the current retry count, or 0 if not found
   */
  public int getRetryCount(String orderId) {
    return retryCountMap.getOrDefault(orderId, 0);
  }

  /**
   * Internal record to hold message with its metadata and acknowledgment.
   *
   * @param message the order message
   * @param acknowledgment the Kafka acknowledgment
   * @param topic the source topic
   * @param partition the source partition
   * @param offset the message offset
   */
  private record MessageWithMetadata(
      OrderMessage message,
      Acknowledgment acknowledgment,
      String topic,
      int partition,
      long offset) {}
}
