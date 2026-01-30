package com.pingxin403.cuckoo.flashsale.kafka;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * 订单消息生产者接口 Interface for producing order messages to Kafka.
 *
 * <p>This interface defines the contract for sending order messages to Kafka after successful stock
 * deduction. Implementations should ensure:
 *
 * <ul>
 *   <li>Messages are sent to the configured order topic (seckill-orders)
 *   <li>Partition routing is based on user_id hash for ordering guarantees
 *   <li>Failures are handled gracefully with appropriate error reporting
 * </ul>
 *
 * <p>Requirements: 2.1, 2.2
 */
public interface OrderMessageProducer {

  /**
   * Sends an order message to Kafka.
   *
   * <p>The message will be routed to a partition based on the user_id hash, ensuring that all
   * messages for the same user go to the same partition for ordering guarantees.
   *
   * @param message the order message to send
   * @return SendResult containing success/failure status and metadata
   */
  SendResult send(OrderMessage message);
}
