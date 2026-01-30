package com.pingxin403.cuckoo.flashsale.repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.pingxin403.cuckoo.flashsale.model.ReconciliationLog;
import com.pingxin403.cuckoo.flashsale.model.enums.ReconciliationStatus;

/**
 * 对账记录数据访问层 Repository interface for ReconciliationLog entity operations.
 *
 * <p>Provides CRUD operations and custom queries for reconciliation log management.
 */
@Repository
public interface ReconciliationLogRepository extends JpaRepository<ReconciliationLog, Long> {

  /**
   * Find all reconciliation logs for a specific SKU.
   *
   * @param skuId the SKU identifier
   * @return list of reconciliation logs for the SKU
   */
  List<ReconciliationLog> findBySkuId(String skuId);

  /**
   * Find reconciliation logs by status.
   *
   * @param status the reconciliation status
   * @return list of reconciliation logs with the specified status
   */
  List<ReconciliationLog> findByStatus(ReconciliationStatus status);

  /**
   * Find the latest reconciliation log for a SKU.
   *
   * @param skuId the SKU identifier
   * @return Optional containing the latest reconciliation log if found
   */
  Optional<ReconciliationLog> findFirstBySkuIdOrderByCreatedAtDesc(String skuId);

  /**
   * Find reconciliation logs with discrepancies.
   *
   * @return list of reconciliation logs with discrepancies
   */
  @Query("SELECT r FROM ReconciliationLog r WHERE r.status = 1 ORDER BY r.createdAt DESC")
  List<ReconciliationLog> findAllWithDiscrepancies();

  /**
   * Find unfixed discrepancies.
   *
   * @return list of reconciliation logs with unfixed discrepancies
   */
  @Query("SELECT r FROM ReconciliationLog r WHERE r.status = 1 ORDER BY r.createdAt ASC")
  List<ReconciliationLog> findUnfixedDiscrepancies();

  /**
   * Find reconciliation logs within a time range.
   *
   * @param startTime the start time
   * @param endTime the end time
   * @return list of reconciliation logs within the time range
   */
  @Query(
      "SELECT r FROM ReconciliationLog r WHERE r.createdAt >= :startTime AND r.createdAt <="
          + " :endTime ORDER BY r.createdAt DESC")
  List<ReconciliationLog> findByTimeRange(
      @Param("startTime") LocalDateTime startTime, @Param("endTime") LocalDateTime endTime);

  /**
   * Find reconciliation logs for a SKU within a time range.
   *
   * @param skuId the SKU identifier
   * @param startTime the start time
   * @param endTime the end time
   * @return list of reconciliation logs within the time range
   */
  @Query(
      "SELECT r FROM ReconciliationLog r WHERE r.skuId = :skuId AND r.createdAt >= :startTime AND"
          + " r.createdAt <= :endTime ORDER BY r.createdAt DESC")
  List<ReconciliationLog> findBySkuIdAndTimeRange(
      @Param("skuId") String skuId,
      @Param("startTime") LocalDateTime startTime,
      @Param("endTime") LocalDateTime endTime);

  /**
   * Count discrepancies for a SKU.
   *
   * @param skuId the SKU identifier
   * @return count of discrepancies
   */
  @Query("SELECT COUNT(r) FROM ReconciliationLog r WHERE r.skuId = :skuId AND r.status = 1")
  long countDiscrepanciesBySkuId(@Param("skuId") String skuId);

  /**
   * Count total discrepancies.
   *
   * @return count of all discrepancies
   */
  @Query("SELECT COUNT(r) FROM ReconciliationLog r WHERE r.status = 1")
  long countAllDiscrepancies();

  /**
   * Update reconciliation status to FIXED.
   *
   * @param id the reconciliation log ID
   * @return number of rows updated
   */
  @Modifying
  @Query("UPDATE ReconciliationLog r SET r.status = 2 WHERE r.id = :id AND r.status = 1")
  int markAsFixed(@Param("id") Long id);

  /**
   * Find the most recent reconciliation for each SKU.
   *
   * @return list of the most recent reconciliation logs per SKU
   */
  @Query(
      "SELECT r FROM ReconciliationLog r WHERE r.createdAt = (SELECT MAX(r2.createdAt) FROM"
          + " ReconciliationLog r2 WHERE r2.skuId = r.skuId)")
  List<ReconciliationLog> findLatestForEachSku();
}
