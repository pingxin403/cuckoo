package com.pingxin403.cuckoo.flashsale.kafka;

import java.util.concurrent.atomic.AtomicLong;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.model.DlqMessage;

/**
 * 死信队列消息消费者 Kafka consumer for monitoring and alerting on dead letter queue messages.
 *
 * <p>This consumer processes messages from the dead letter queue (seckill-dlq) for:
 *
 * <ul>
 *   <li>Monitoring: tracks DLQ message count and logs details
 *   <li>Alerting: logs warnings for manual investigation
 *   <li>Metrics: exposes DLQ message count for monitoring systems
 * </ul>
 *
 * <p>Note: This is a placeholder implementation. In production, this should integrate with:
 *
 * <ul>
 *   <li>Alerting systems (PagerDuty, OpsGenie, etc.)
 *   <li>Monitoring dashboards (Grafana, etc.)
 *   <li>Manual reprocessing workflows
 * </ul>
 *
 * <p>Requirements: 2.5
 */
@Component
public class DlqMessageConsumer {

  private static final Logger logger = LoggerFactory.getLogger(DlqMessageConsumer.class);

  /** Counter for total DLQ messages received. */
  private final AtomicLong dlqMessageCount = new AtomicLong(0);

  /** Counter for DLQ messages received in the last hour (for alerting). */
  private final AtomicLong hourlyDlqMessageCount = new AtomicLong(0);

  /**
   * Consumes messages from the dead letter queue.
   *
   * <p>This method logs the failed message details for manual investigation and updates monitoring
   * counters. In a production system, this would also trigger alerts.
   *
   * @param record the Kafka consumer record containing the DLQ message
   * @param acknowledgment the acknowledgment for manual commit
   */
  @KafkaListener(
      topics = "${flash-sale.kafka.dlq-topic:seckill-dlq}",
      groupId = "${spring.kafka.consumer.group-id:flash-sale-dlq-consumer}",
      containerFactory = "dlqKafkaListenerContainerFactory")
  public void consume(ConsumerRecord<String, DlqMessage> record, Acknowledgment acknowledgment) {
    DlqMessage dlqMessage = record.value();

    if (dlqMessage == null) {
      logger.warn(
          "Received null DLQ message from partition {} offset {}",
          record.partition(),
          record.offset());
      acknowledgment.acknowledge();
      return;
    }

    // Increment counters
    long totalCount = dlqMessageCount.incrementAndGet();
    long hourlyCount = hourlyDlqMessageCount.incrementAndGet();

    // Log the failed message details for investigation
    logger.warn(
        "DLQ Message received - orderId={}, userId={}, skuId={}, retryCount={}, error={}, "
            + "originalTopic={}, originalPartition={}, originalOffset={}, timestamp={}",
        dlqMessage.originalMessage() != null ? dlqMessage.originalMessage().orderId() : "null",
        dlqMessage.originalMessage() != null ? dlqMessage.originalMessage().userId() : "null",
        dlqMessage.originalMessage() != null ? dlqMessage.originalMessage().skuId() : "null",
        dlqMessage.retryCount(),
        dlqMessage.errorMessage(),
        dlqMessage.topic(),
        dlqMessage.partition(),
        dlqMessage.offset(),
        dlqMessage.timestamp());

    // Log alert if hourly count exceeds threshold
    if (hourlyCount > 100) {
      logger.error(
          "ALERT: High DLQ message rate detected! {} messages in the last hour. "
              + "Total DLQ messages: {}. Manual investigation required.",
          hourlyCount,
          totalCount);
    }

    // TODO: In production, integrate with alerting systems:
    // - Send alert to PagerDuty/OpsGenie
    // - Update Prometheus metrics
    // - Store in database for manual reprocessing UI

    acknowledgment.acknowledge();
  }

  /**
   * Returns the total count of DLQ messages received.
   *
   * @return the total DLQ message count
   */
  public long getDlqMessageCount() {
    return dlqMessageCount.get();
  }

  /**
   * Returns the hourly count of DLQ messages received.
   *
   * @return the hourly DLQ message count
   */
  public long getHourlyDlqMessageCount() {
    return hourlyDlqMessageCount.get();
  }

  /** Resets the hourly counter. Should be called by a scheduled task every hour. */
  public void resetHourlyCounter() {
    long previousCount = hourlyDlqMessageCount.getAndSet(0);
    if (previousCount > 0) {
      logger.info("Hourly DLQ counter reset. Previous hour count: {}", previousCount);
    }
  }
}
