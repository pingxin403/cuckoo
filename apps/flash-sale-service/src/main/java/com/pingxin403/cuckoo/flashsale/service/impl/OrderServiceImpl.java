package com.pingxin403.cuckoo.flashsale.service.impl;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.repository.SeckillOrderRepository;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.dto.BatchCreateResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;

/**
 * 订单服务实现 Implementation of OrderService for managing seckill orders.
 *
 * <p>This implementation provides:
 *
 * <ul>
 *   <li>Single and batch order creation
 *   <li>Order status management with idempotency
 *   <li>Timeout order handling
 *   <li>Redis cache for order status (TTL: 24 hours)
 * </ul>
 *
 * <p>Requirements: 4.3, 5.1, 5.2, 5.3, 5.4
 */
@Service
public class OrderServiceImpl implements OrderService {

  private static final Logger logger = LoggerFactory.getLogger(OrderServiceImpl.class);

  /** Redis key prefix for order status cache */
  private static final String ORDER_STATUS_PREFIX = "order_status:";

  /** Order status cache TTL in seconds (24 hours) */
  private static final long ORDER_STATUS_TTL = 24 * 60 * 60;

  private final SeckillOrderRepository orderRepository;
  private final StringRedisTemplate stringRedisTemplate;
  private final InventoryService inventoryService;

  public OrderServiceImpl(
      SeckillOrderRepository orderRepository,
      StringRedisTemplate stringRedisTemplate,
      InventoryService inventoryService) {
    this.orderRepository = orderRepository;
    this.stringRedisTemplate = stringRedisTemplate;
    this.inventoryService = inventoryService;
  }

  @Override
  @Transactional
  public SeckillOrder createOrder(OrderMessage message) {
    logger.debug("Creating order: orderId={}", message.orderId());

    // Check for duplicate order (idempotency)
    Optional<SeckillOrder> existing = orderRepository.findByOrderId(message.orderId());
    if (existing.isPresent()) {
      logger.info("Order already exists: orderId={}", message.orderId());
      return existing.get();
    }

    SeckillOrder order = new SeckillOrder();
    order.setOrderId(message.orderId());
    order.setUserId(message.userId());
    order.setSkuId(message.skuId());
    order.setQuantity(message.quantity());
    order.setStatus(OrderStatus.PENDING_PAYMENT);
    order.setCreatedAt(LocalDateTime.now());
    order.setSource(message.source());
    order.setTraceId(message.traceId());
    // ActivityId will be set based on SKU lookup in full implementation
    order.setActivityId("default");

    SeckillOrder savedOrder = orderRepository.save(order);

    // Cache order status to Redis for fast query (Requirement 4.3)
    cacheOrderStatus(savedOrder.getOrderId(), savedOrder.getStatus());

    return savedOrder;
  }

  @Override
  @Transactional
  public BatchCreateResult batchCreateOrders(List<OrderMessage> messages) {
    if (messages == null || messages.isEmpty()) {
      return BatchCreateResult.success(0);
    }

    logger.info("Batch creating {} orders", messages.size());

    int successCount = 0;
    List<String> failedOrderIds = new ArrayList<>();

    for (OrderMessage message : messages) {
      try {
        // Check for duplicate order (idempotency)
        if (orderRepository.findByOrderId(message.orderId()).isPresent()) {
          logger.debug("Order already exists, skipping: orderId={}", message.orderId());
          successCount++;
          continue;
        }

        SeckillOrder order = new SeckillOrder();
        order.setOrderId(message.orderId());
        order.setUserId(message.userId());
        order.setSkuId(message.skuId());
        order.setQuantity(message.quantity());
        order.setStatus(OrderStatus.PENDING_PAYMENT);
        order.setCreatedAt(LocalDateTime.now());
        order.setSource(message.source());
        order.setTraceId(message.traceId());
        order.setActivityId("default");

        orderRepository.save(order);

        // Cache order status to Redis for fast query (Requirement 4.3)
        cacheOrderStatus(order.getOrderId(), order.getStatus());

        successCount++;
      } catch (Exception e) {
        logger.error("Failed to create order: orderId={}", message.orderId(), e);
        failedOrderIds.add(message.orderId());
      }
    }

    logger.info(
        "Batch creation completed: total={}, success={}, failed={}",
        messages.size(),
        successCount,
        failedOrderIds.size());

    if (failedOrderIds.isEmpty()) {
      return BatchCreateResult.success(successCount);
    } else {
      return BatchCreateResult.partial(messages.size(), successCount, failedOrderIds);
    }
  }

  @Override
  @Transactional
  public boolean updateStatus(String orderId, OrderStatus newStatus) {
    logger.debug("Updating order status: orderId={}, newStatus={}", orderId, newStatus);

    Optional<SeckillOrder> orderOpt = orderRepository.findByOrderId(orderId);
    if (orderOpt.isEmpty()) {
      logger.warn("Order not found: orderId={}", orderId);
      return false;
    }

    SeckillOrder order = orderOpt.get();

    // Idempotency check - if already in target status, return success
    if (order.getStatus() == newStatus) {
      logger.debug("Order already in status {}: orderId={}", newStatus, orderId);
      return true;
    }

    // Validate state transition
    if (!isValidTransition(order.getStatus(), newStatus)) {
      logger.warn(
          "Invalid status transition: orderId={}, from={}, to={}",
          orderId,
          order.getStatus(),
          newStatus);
      return false;
    }

    order.setStatus(newStatus);

    // Set timestamp based on new status
    switch (newStatus) {
      case PAID -> order.setPaidAt(LocalDateTime.now());
      case CANCELLED, TIMEOUT -> order.setCancelledAt(LocalDateTime.now());
      default -> {
        // No additional timestamp needed
      }
    }

    orderRepository.save(order);

    // Update order status cache in Redis (Requirement 4.3)
    cacheOrderStatus(orderId, newStatus);

    logger.info("Order status updated: orderId={}, newStatus={}", orderId, newStatus);
    return true;
  }

  @Override
  public Optional<SeckillOrder> getOrder(String orderId) {
    return orderRepository.findByOrderId(orderId);
  }

  @Override
  @Transactional
  public int handleTimeoutOrders(int timeoutMinutes) {
    LocalDateTime cutoffTime = LocalDateTime.now().minusMinutes(timeoutMinutes);
    logger.info("Handling timeout orders created before {}", cutoffTime);

    List<SeckillOrder> timeoutOrders =
        orderRepository.findByStatusAndCreatedAtBefore(OrderStatus.PENDING_PAYMENT, cutoffTime);

    int count = 0;
    for (SeckillOrder order : timeoutOrders) {
      try {
        order.setStatus(OrderStatus.TIMEOUT);
        order.setCancelledAt(LocalDateTime.now());
        orderRepository.save(order);

        // Update order status cache in Redis (Requirement 4.3)
        cacheOrderStatus(order.getOrderId(), OrderStatus.TIMEOUT);

        count++;

        // Trigger stock rollback via InventoryService (Requirement 5.3)
        try {
          RollbackResult rollbackResult =
              inventoryService.rollbackStock(
                  order.getSkuId(), order.getOrderId(), order.getQuantity());
          if (rollbackResult.success()) {
            logger.info(
                "Stock rollback successful: orderId={}, skuId={}, quantity={}",
                order.getOrderId(),
                order.getSkuId(),
                order.getQuantity());
          } else {
            logger.error(
                "Stock rollback failed: orderId={}, skuId={}, reason={}",
                order.getOrderId(),
                order.getSkuId(),
                rollbackResult.message());
          }
        } catch (Exception rollbackException) {
          logger.error(
              "Exception during stock rollback: orderId={}, skuId={}",
              order.getOrderId(),
              order.getSkuId(),
              rollbackException);
        }

        logger.info("Order timed out: orderId={}", order.getOrderId());
      } catch (Exception e) {
        logger.error("Failed to timeout order: orderId={}", order.getOrderId(), e);
      }
    }

    logger.info("Timed out {} orders", count);
    return count;
  }

  /**
   * Validates if a status transition is allowed.
   *
   * <p>Valid transitions:
   *
   * <ul>
   *   <li>PENDING_PAYMENT -> PAID
   *   <li>PENDING_PAYMENT -> CANCELLED
   *   <li>PENDING_PAYMENT -> TIMEOUT
   * </ul>
   *
   * @param from current status
   * @param to target status
   * @return true if transition is valid
   */
  private boolean isValidTransition(OrderStatus from, OrderStatus to) {
    if (from == OrderStatus.PENDING_PAYMENT) {
      return to == OrderStatus.PAID || to == OrderStatus.CANCELLED || to == OrderStatus.TIMEOUT;
    }
    return false;
  }

  /**
   * Cache order status to Redis for fast query.
   *
   * <p>Validates Requirement: 4.3 - Provide /status endpoint for frontend polling
   *
   * @param orderId the order identifier
   * @param status the order status
   */
  private void cacheOrderStatus(String orderId, OrderStatus status) {
    try {
      String key = ORDER_STATUS_PREFIX + orderId;
      stringRedisTemplate.opsForValue().set(key, status.name(), ORDER_STATUS_TTL, TimeUnit.SECONDS);
      logger.debug("Cached order status: orderId={}, status={}", orderId, status);
    } catch (Exception e) {
      // Log error but don't fail the transaction - cache is not critical
      logger.error("Failed to cache order status: orderId={}, status={}", orderId, status, e);
    }
  }
}
