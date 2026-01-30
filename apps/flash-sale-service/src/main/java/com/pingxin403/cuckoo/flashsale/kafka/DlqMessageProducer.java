package com.pingxin403.cuckoo.flashsale.kafka;

import com.pingxin403.cuckoo.flashsale.model.DlqMessage;

/**
 * 死信队列消息生产者接口 Interface for producing messages to the dead letter queue.
 *
 * <p>This interface defines the contract for sending failed messages to the dead letter queue
 * (seckill-dlq) after retry attempts have been exhausted. Implementations should ensure:
 *
 * <ul>
 *   <li>Messages are sent to the configured DLQ topic (seckill-dlq)
 *   <li>Original message metadata is preserved for debugging
 *   <li>Failures are logged but don't block the consumer
 * </ul>
 *
 * <p>Requirements: 2.5
 */
public interface DlqMessageProducer {

  /**
   * Sends a failed message to the dead letter queue.
   *
   * <p>The message will be sent to the DLQ topic for manual investigation and potential
   * reprocessing.
   *
   * @param message the DLQ message containing the original message and failure details
   * @return SendResult containing success/failure status and metadata
   */
  SendResult send(DlqMessage message);
}
