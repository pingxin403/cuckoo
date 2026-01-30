package com.pingxin403.cuckoo.flashsale.service;

import com.pingxin403.cuckoo.flashsale.service.dto.Discrepancy;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

/**
 * 对账服务接口 Reconciliation service interface for data consistency verification.
 *
 * <p>Provides periodic reconciliation between Redis stock data and MySQL order data to ensure data
 * consistency across the system.
 *
 * <p>Key responsibilities:
 *
 * <ul>
 *   <li>Compare Redis stock/sold counts with MySQL order counts
 *   <li>Detect and report discrepancies
 *   <li>Fix data inconsistencies when possible
 *   <li>Generate reconciliation reports for monitoring
 * </ul>
 *
 * <p>Validates Requirements: 6.4, 6.5, 6.6, 6.7
 *
 * @see ReconciliationResult
 * @see ReconciliationReport
 * @see Discrepancy
 */
public interface ReconciliationService {

  /**
   * 执行单SKU对账 Perform reconciliation for a single SKU.
   *
   * <p>Compares Redis stock data with MySQL order data for the specified SKU and detects any
   * discrepancies.
   *
   * <p>Checks performed:
   *
   * <ul>
   *   <li>Redis sold count matches MySQL order count
   *   <li>Total stock consistency (redisStock + redisSold = totalStock)
   * </ul>
   *
   * <p>Validates: Requirements 6.4, 6.5
   *
   * @param skuId the SKU identifier to reconcile
   * @return ReconciliationResult containing reconciliation status and any discrepancies
   */
  ReconciliationResult reconcile(String skuId);

  /**
   * 全量对账 Perform full reconciliation across all active SKUs.
   *
   * <p>Executes reconciliation for all SKUs that have active flash sale activities and generates a
   * comprehensive report.
   *
   * <p>This method should be called periodically (e.g., hourly) to ensure ongoing data consistency.
   *
   * <p>Validates: Requirements 6.4, 6.5
   *
   * @return ReconciliationReport containing results for all SKUs
   */
  ReconciliationReport fullReconcile();

  /**
   * 修复差异数据 Fix a detected discrepancy.
   *
   * <p>Attempts to automatically fix data inconsistencies by:
   *
   * <ul>
   *   <li>Adjusting Redis stock/sold counts based on MySQL order data (source of truth)
   *   <li>Logging the fix operation for audit purposes
   *   <li>Marking the reconciliation log as FIXED
   * </ul>
   *
   * <p>Validates: Requirement 6.7
   *
   * @param discrepancy the discrepancy to fix
   * @return true if the discrepancy was successfully fixed, false otherwise
   */
  boolean fixDiscrepancy(Discrepancy discrepancy);
}
