package com.pingxin403.cuckoo.flashsale.service.dto;

import java.util.List;

/**
 * 对账结果 Reconciliation result for a single SKU.
 *
 * <p>Contains the reconciliation status and any discrepancies found between Redis and MySQL data.
 *
 * @param skuId the SKU identifier
 * @param redisStock current stock in Redis
 * @param redisSoldCount sold count in Redis
 * @param mysqlOrderCount order count in MySQL
 * @param discrepancies list of discrepancies found
 * @param passed true if reconciliation passed (no discrepancies)
 */
public record ReconciliationResult(
    String skuId,
    int redisStock,
    int redisSoldCount,
    int mysqlOrderCount,
    List<Discrepancy> discrepancies,
    boolean passed) {

  /**
   * Create a successful reconciliation result (no discrepancies).
   *
   * @param skuId the SKU identifier
   * @param redisStock current stock in Redis
   * @param redisSoldCount sold count in Redis
   * @param mysqlOrderCount order count in MySQL
   * @return ReconciliationResult with passed=true
   */
  public static ReconciliationResult success(
      String skuId, int redisStock, int redisSoldCount, int mysqlOrderCount) {
    return new ReconciliationResult(
        skuId, redisStock, redisSoldCount, mysqlOrderCount, List.of(), true);
  }

  /**
   * Create a failed reconciliation result (with discrepancies).
   *
   * @param skuId the SKU identifier
   * @param redisStock current stock in Redis
   * @param redisSoldCount sold count in Redis
   * @param mysqlOrderCount order count in MySQL
   * @param discrepancies list of discrepancies found
   * @return ReconciliationResult with passed=false
   */
  public static ReconciliationResult failure(
      String skuId,
      int redisStock,
      int redisSoldCount,
      int mysqlOrderCount,
      List<Discrepancy> discrepancies) {
    return new ReconciliationResult(
        skuId, redisStock, redisSoldCount, mysqlOrderCount, discrepancies, false);
  }

  /**
   * Get the number of discrepancies found.
   *
   * @return count of discrepancies
   */
  public int getDiscrepancyCount() {
    return discrepancies.size();
  }
}
