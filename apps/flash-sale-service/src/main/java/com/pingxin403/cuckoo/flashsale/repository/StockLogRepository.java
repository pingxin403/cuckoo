package com.pingxin403.cuckoo.flashsale.repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.pingxin403.cuckoo.flashsale.model.StockLog;
import com.pingxin403.cuckoo.flashsale.model.enums.StockOperation;

/**
 * 库存流水数据访问层 Repository interface for StockLog entity operations.
 *
 * <p>Provides CRUD operations and custom queries for stock operation log management.
 */
@Repository
public interface StockLogRepository extends JpaRepository<StockLog, Long> {

  /**
   * Find all stock logs for a specific SKU.
   *
   * @param skuId the SKU identifier
   * @return list of stock logs for the SKU
   */
  List<StockLog> findBySkuId(String skuId);

  /**
   * Find all stock logs for a specific order.
   *
   * @param orderId the order identifier
   * @return list of stock logs for the order
   */
  List<StockLog> findByOrderId(String orderId);

  /**
   * Find stock logs by SKU and operation type.
   *
   * @param skuId the SKU identifier
   * @param operation the operation type
   * @return list of matching stock logs
   */
  List<StockLog> findBySkuIdAndOperation(String skuId, StockOperation operation);

  /**
   * Find the latest stock log for a SKU.
   *
   * @param skuId the SKU identifier
   * @return Optional containing the latest stock log if found
   */
  Optional<StockLog> findFirstBySkuIdOrderByCreatedAtDesc(String skuId);

  /**
   * Find the latest stock log for an order.
   *
   * @param orderId the order identifier
   * @return Optional containing the latest stock log if found
   */
  Optional<StockLog> findFirstByOrderIdOrderByCreatedAtDesc(String orderId);

  /**
   * Find stock logs within a time range for a SKU.
   *
   * @param skuId the SKU identifier
   * @param startTime the start time
   * @param endTime the end time
   * @return list of stock logs within the time range
   */
  @Query(
      "SELECT s FROM StockLog s WHERE s.skuId = :skuId AND s.createdAt >= :startTime AND"
          + " s.createdAt <= :endTime ORDER BY s.createdAt ASC")
  List<StockLog> findBySkuIdAndTimeRange(
      @Param("skuId") String skuId,
      @Param("startTime") LocalDateTime startTime,
      @Param("endTime") LocalDateTime endTime);

  /**
   * Count deductions for a SKU.
   *
   * @param skuId the SKU identifier
   * @return count of deduction operations
   */
  @Query("SELECT COUNT(s) FROM StockLog s WHERE s.skuId = :skuId AND s.operation = 0")
  long countDeductionsBySkuId(@Param("skuId") String skuId);

  /**
   * Count rollbacks for a SKU.
   *
   * @param skuId the SKU identifier
   * @return count of rollback operations
   */
  @Query("SELECT COUNT(s) FROM StockLog s WHERE s.skuId = :skuId AND s.operation = 1")
  long countRollbacksBySkuId(@Param("skuId") String skuId);

  /**
   * Sum total quantity deducted for a SKU.
   *
   * @param skuId the SKU identifier
   * @return total quantity deducted
   */
  @Query(
      "SELECT COALESCE(SUM(s.quantity), 0) FROM StockLog s WHERE s.skuId = :skuId AND s.operation ="
          + " 0")
  long sumDeductedQuantityBySkuId(@Param("skuId") String skuId);

  /**
   * Sum total quantity rolled back for a SKU.
   *
   * @param skuId the SKU identifier
   * @return total quantity rolled back
   */
  @Query(
      "SELECT COALESCE(SUM(s.quantity), 0) FROM StockLog s WHERE s.skuId = :skuId AND s.operation ="
          + " 1")
  long sumRolledBackQuantityBySkuId(@Param("skuId") String skuId);

  /**
   * Check if a deduction log exists for an order.
   *
   * @param orderId the order identifier
   * @return true if deduction log exists
   */
  @Query("SELECT COUNT(s) > 0 FROM StockLog s WHERE s.orderId = :orderId AND s.operation = 0")
  boolean existsDeductionByOrderId(@Param("orderId") String orderId);

  /**
   * Check if a rollback log exists for an order.
   *
   * @param orderId the order identifier
   * @return true if rollback log exists
   */
  @Query("SELECT COUNT(s) > 0 FROM StockLog s WHERE s.orderId = :orderId AND s.operation = 1")
  boolean existsRollbackByOrderId(@Param("orderId") String orderId);
}
