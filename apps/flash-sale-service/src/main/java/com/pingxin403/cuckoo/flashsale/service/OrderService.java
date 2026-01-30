package com.pingxin403.cuckoo.flashsale.service;

import java.util.List;
import java.util.Optional;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.service.dto.BatchCreateResult;

/**
 * 订单服务接口 Interface for order management operations.
 *
 * <p>This interface defines the contract for managing seckill orders, including:
 *
 * <ul>
 *   <li>Creating individual orders
 *   <li>Batch creating orders from Kafka messages
 *   <li>Updating order status
 *   <li>Querying orders
 *   <li>Handling timeout orders
 * </ul>
 *
 * <p>Requirements: 5.1, 5.2, 5.3, 5.4
 */
public interface OrderService {

  /**
   * Creates a single order from an order message.
   *
   * <p>This method creates a new order with status PENDING_PAYMENT.
   *
   * @param message the order message containing order details
   * @return the created SeckillOrder
   */
  SeckillOrder createOrder(OrderMessage message);

  /**
   * Batch creates orders from a list of order messages.
   *
   * <p>This method is called by the Kafka consumer to batch insert orders into the database. It
   * should:
   *
   * <ul>
   *   <li>Process all messages in a single transaction when possible
   *   <li>Handle duplicates gracefully (idempotent)
   *   <li>Return detailed results about success/failure
   * </ul>
   *
   * <p>Requirements: 2.3
   *
   * @param messages list of order messages to process
   * @return BatchCreateResult containing success/failure counts
   */
  BatchCreateResult batchCreateOrders(List<OrderMessage> messages);

  /**
   * Updates the status of an order.
   *
   * <p>This method should be idempotent - calling it multiple times with the same status should
   * produce the same result.
   *
   * <p>Requirements: 5.2, 5.4
   *
   * @param orderId the order ID to update
   * @param newStatus the new status to set
   * @return true if the update was successful, false otherwise
   */
  boolean updateStatus(String orderId, OrderStatus newStatus);

  /**
   * Retrieves an order by its ID.
   *
   * @param orderId the order ID to look up
   * @return Optional containing the order if found, empty otherwise
   */
  Optional<SeckillOrder> getOrder(String orderId);

  /**
   * Handles orders that have timed out (not paid within the timeout period).
   *
   * <p>This method should:
   *
   * <ul>
   *   <li>Find all orders with status PENDING_PAYMENT older than timeoutMinutes
   *   <li>Update their status to TIMEOUT
   *   <li>Trigger stock rollback for each timed out order
   * </ul>
   *
   * <p>Requirements: 5.3
   *
   * @param timeoutMinutes the timeout threshold in minutes
   * @return the number of orders that were timed out
   */
  int handleTimeoutOrders(int timeoutMinutes);
}
