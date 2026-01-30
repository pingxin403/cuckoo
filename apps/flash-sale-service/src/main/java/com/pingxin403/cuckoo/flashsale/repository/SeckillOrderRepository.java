package com.pingxin403.cuckoo.flashsale.repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;

/**
 * 秒杀订单数据访问层 Repository interface for SeckillOrder entity operations.
 *
 * <p>Provides CRUD operations and custom queries for flash sale order management.
 */
@Repository
public interface SeckillOrderRepository extends JpaRepository<SeckillOrder, Long> {

  /**
   * Find order by unique order ID.
   *
   * @param orderId the unique order identifier
   * @return Optional containing the order if found
   */
  Optional<SeckillOrder> findByOrderId(String orderId);

  /**
   * Find all orders for a specific user.
   *
   * @param userId the user identifier
   * @return list of orders for the user
   */
  List<SeckillOrder> findByUserId(String userId);

  /**
   * Find all orders for a specific SKU.
   *
   * @param skuId the SKU identifier
   * @return list of orders for the SKU
   */
  List<SeckillOrder> findBySkuId(String skuId);

  /**
   * Find all orders for a specific activity.
   *
   * @param activityId the activity identifier
   * @return list of orders for the activity
   */
  List<SeckillOrder> findByActivityId(String activityId);

  /**
   * Find orders by status.
   *
   * @param status the order status
   * @return list of orders with the specified status
   */
  List<SeckillOrder> findByStatus(OrderStatus status);

  /**
   * Find orders by user and SKU (for purchase limit checking).
   *
   * @param userId the user identifier
   * @param skuId the SKU identifier
   * @return list of orders for the user and SKU
   */
  List<SeckillOrder> findByUserIdAndSkuId(String userId, String skuId);

  /**
   * Count orders by user and SKU with specific statuses (for purchase limit checking).
   *
   * @param userId the user identifier
   * @param skuId the SKU identifier
   * @param statuses the list of statuses to include
   * @return count of matching orders
   */
  @Query(
      "SELECT COUNT(o) FROM SeckillOrder o WHERE o.userId = :userId AND o.skuId = :skuId AND"
          + " o.status IN :statuses")
  long countByUserIdAndSkuIdAndStatusIn(
      @Param("userId") String userId,
      @Param("skuId") String skuId,
      @Param("statuses") List<OrderStatus> statuses);

  /**
   * Find timeout orders (pending payment and created before threshold).
   *
   * @param status the status to check (PENDING_PAYMENT)
   * @param threshold the time threshold
   * @return list of timeout orders
   */
  @Query(
      "SELECT o FROM SeckillOrder o WHERE o.status = :status AND o.createdAt < :threshold ORDER BY"
          + " o.createdAt ASC")
  List<SeckillOrder> findTimeoutOrders(
      @Param("status") OrderStatus status, @Param("threshold") LocalDateTime threshold);

  /**
   * Find orders by status and created before a specific time.
   *
   * @param status the order status
   * @param createdAt the time threshold
   * @return list of matching orders
   */
  List<SeckillOrder> findByStatusAndCreatedAtBefore(OrderStatus status, LocalDateTime createdAt);

  /**
   * Count orders by activity ID (for reconciliation).
   *
   * @param activityId the activity identifier
   * @return count of orders for the activity
   */
  long countByActivityId(String activityId);

  /**
   * Count orders by SKU ID (for reconciliation).
   *
   * @param skuId the SKU identifier
   * @return count of orders for the SKU
   */
  long countBySkuId(String skuId);

  /**
   * Count orders by SKU ID and status (for reconciliation).
   *
   * @param skuId the SKU identifier
   * @param statuses the list of statuses to include
   * @return count of matching orders
   */
  @Query("SELECT COUNT(o) FROM SeckillOrder o WHERE o.skuId = :skuId AND o.status IN :statuses")
  long countBySkuIdAndStatusIn(
      @Param("skuId") String skuId, @Param("statuses") List<OrderStatus> statuses);

  /**
   * Update order status.
   *
   * @param orderId the order identifier
   * @param newStatus the new status
   * @return number of rows updated
   */
  @Modifying
  @Query("UPDATE SeckillOrder o SET o.status = :newStatus WHERE o.orderId = :orderId")
  int updateStatus(@Param("orderId") String orderId, @Param("newStatus") OrderStatus newStatus);

  /**
   * Update order status to PAID with payment time.
   *
   * @param orderId the order identifier
   * @param paidAt the payment time
   * @return number of rows updated
   */
  @Modifying
  @Query(
      "UPDATE SeckillOrder o SET o.status = 1, o.paidAt = :paidAt WHERE o.orderId = :orderId AND"
          + " o.status = 0")
  int markAsPaid(@Param("orderId") String orderId, @Param("paidAt") LocalDateTime paidAt);

  /**
   * Update order status to CANCELLED with cancellation time.
   *
   * @param orderId the order identifier
   * @param cancelledAt the cancellation time
   * @return number of rows updated
   */
  @Modifying
  @Query(
      "UPDATE SeckillOrder o SET o.status = 2, o.cancelledAt = :cancelledAt WHERE o.orderId ="
          + " :orderId AND o.status = 0")
  int markAsCancelled(
      @Param("orderId") String orderId, @Param("cancelledAt") LocalDateTime cancelledAt);

  /**
   * Update order status to TIMEOUT with cancellation time.
   *
   * @param orderId the order identifier
   * @param cancelledAt the cancellation time
   * @return number of rows updated
   */
  @Modifying
  @Query(
      "UPDATE SeckillOrder o SET o.status = 3, o.cancelledAt = :cancelledAt WHERE o.orderId ="
          + " :orderId AND o.status = 0")
  int markAsTimeout(
      @Param("orderId") String orderId, @Param("cancelledAt") LocalDateTime cancelledAt);

  /**
   * Check if order exists by order ID.
   *
   * @param orderId the order identifier
   * @return true if order exists
   */
  boolean existsByOrderId(String orderId);
}
