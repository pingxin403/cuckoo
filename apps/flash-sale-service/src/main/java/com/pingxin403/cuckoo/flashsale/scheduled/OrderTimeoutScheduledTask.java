package com.pingxin403.cuckoo.flashsale.scheduled;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.service.OrderService;

/**
 * 订单超时处理定时任务 Scheduled task for handling timeout orders.
 *
 * <p>This task runs periodically to:
 *
 * <ul>
 *   <li>Find orders in PENDING_PAYMENT status that have exceeded the timeout threshold
 *   <li>Update their status to TIMEOUT
 *   <li>Trigger stock rollback for each timed out order
 * </ul>
 *
 * <p>Validates: Requirement 5.3
 */
@Component
public class OrderTimeoutScheduledTask {

  private static final Logger logger = LoggerFactory.getLogger(OrderTimeoutScheduledTask.class);

  private final OrderService orderService;

  @Value("${flash-sale.order.timeout-minutes:10}")
  private int timeoutMinutes;

  public OrderTimeoutScheduledTask(OrderService orderService) {
    this.orderService = orderService;
  }

  /**
   * Executes timeout order handling every minute.
   *
   * <p>Scans for orders that have been in PENDING_PAYMENT status for longer than the configured
   * timeout period (default: 10 minutes) and processes them.
   *
   * <p>Validates: Requirements 5.3
   */
  @Scheduled(cron = "${flash-sale.order.timeout-cron:0 * * * * ?}")
  public void handleTimeoutOrders() {
    logger.debug("Starting scheduled timeout order handling");

    try {
      int count = orderService.handleTimeoutOrders(timeoutMinutes);

      if (count > 0) {
        logger.info("Handled {} timeout orders", count);
      } else {
        logger.debug("No timeout orders found");
      }
    } catch (Exception e) {
      logger.error("Error handling timeout orders", e);
    }
  }
}
