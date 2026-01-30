package com.pingxin403.cuckoo.flashsale.service.dto;

import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;

/**
 * 订单状态查询结果 Order status query result.
 *
 * <p>Represents the current status of an order in the flash sale system.
 *
 * <p>Validates Requirement: 4.3
 *
 * @param orderId the order identifier
 * @param status the current order status
 * @param message the human-readable status message
 */
public record OrderStatusResult(String orderId, OrderStatus status, String message) {

  /**
   * Create a status result for a pending payment order.
   *
   * @param orderId the order identifier
   * @return OrderStatusResult with PENDING_PAYMENT status
   */
  public static OrderStatusResult pendingPayment(String orderId) {
    return new OrderStatusResult(orderId, OrderStatus.PENDING_PAYMENT, "待支付");
  }

  /**
   * Create a status result for a paid order.
   *
   * @param orderId the order identifier
   * @return OrderStatusResult with PAID status
   */
  public static OrderStatusResult paid(String orderId) {
    return new OrderStatusResult(orderId, OrderStatus.PAID, "已支付");
  }

  /**
   * Create a status result for a cancelled order.
   *
   * @param orderId the order identifier
   * @return OrderStatusResult with CANCELLED status
   */
  public static OrderStatusResult cancelled(String orderId) {
    return new OrderStatusResult(orderId, OrderStatus.CANCELLED, "已取消");
  }

  /**
   * Create a status result for a timeout order.
   *
   * @param orderId the order identifier
   * @return OrderStatusResult with TIMEOUT status
   */
  public static OrderStatusResult timeout(String orderId) {
    return new OrderStatusResult(orderId, OrderStatus.TIMEOUT, "超时取消");
  }

  /**
   * Create a status result for an order not found.
   *
   * @param orderId the order identifier
   * @return OrderStatusResult with null status
   */
  public static OrderStatusResult notFound(String orderId) {
    return new OrderStatusResult(orderId, null, "订单不存在");
  }
}
