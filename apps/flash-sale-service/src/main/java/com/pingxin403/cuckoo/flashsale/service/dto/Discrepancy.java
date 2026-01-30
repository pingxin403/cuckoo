package com.pingxin403.cuckoo.flashsale.service.dto;

import java.util.Map;

/**
 * 差异记录 Discrepancy record for reconciliation.
 *
 * <p>Represents a data inconsistency found during reconciliation between Redis and MySQL.
 *
 * @param type the type of discrepancy (e.g., STOCK_MISMATCH, ORDER_MISSING)
 * @param description human-readable description of the discrepancy
 * @param details additional details about the discrepancy
 */
public record Discrepancy(String type, String description, Map<String, Object> details) {

  /**
   * Create a stock mismatch discrepancy.
   *
   * @param skuId the SKU identifier
   * @param expected expected value
   * @param actual actual value
   * @return Discrepancy instance
   */
  public static Discrepancy stockMismatch(String skuId, int expected, int actual) {
    return new Discrepancy(
        "STOCK_MISMATCH",
        String.format("Stock mismatch: expected=%d, actual=%d", expected, actual),
        Map.of("skuId", skuId, "expected", expected, "actual", actual));
  }

  /**
   * Create an order count mismatch discrepancy.
   *
   * @param skuId the SKU identifier
   * @param redisSold sold count in Redis
   * @param mysqlCount order count in MySQL
   * @return Discrepancy instance
   */
  public static Discrepancy orderCountMismatch(String skuId, int redisSold, int mysqlCount) {
    return new Discrepancy(
        "ORDER_COUNT_MISMATCH",
        String.format("Order count mismatch: Redis sold=%d, MySQL count=%d", redisSold, mysqlCount),
        Map.of("skuId", skuId, "redisSold", redisSold, "mysqlCount", mysqlCount));
  }

  /**
   * Create a total stock mismatch discrepancy.
   *
   * @param skuId the SKU identifier
   * @param totalStock expected total stock
   * @param redisStock current Redis stock
   * @param redisSold current Redis sold count
   * @return Discrepancy instance
   */
  public static Discrepancy totalStockMismatch(
      String skuId, int totalStock, int redisStock, int redisSold) {
    int calculated = redisStock + redisSold;
    return new Discrepancy(
        "TOTAL_STOCK_MISMATCH",
        String.format(
            "Total stock mismatch: expected=%d, calculated=%d (stock=%d + sold=%d)",
            totalStock, calculated, redisStock, redisSold),
        Map.of(
            "skuId", skuId,
            "totalStock", totalStock,
            "redisStock", redisStock,
            "redisSold", redisSold,
            "calculated", calculated));
  }
}
