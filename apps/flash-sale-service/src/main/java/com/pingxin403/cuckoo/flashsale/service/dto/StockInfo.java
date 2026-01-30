package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 库存信息 Stock information record for inventory queries.
 *
 * <p>Contains the current stock status for a SKU from Redis cache.
 *
 * @param skuId the SKU identifier
 * @param totalStock the total stock (initial stock loaded during warmup)
 * @param soldCount the number of items sold
 * @param remainingStock the remaining available stock
 */
public record StockInfo(String skuId, int totalStock, int soldCount, int remainingStock) {

  /**
   * Factory method to create a StockInfo from Redis values.
   *
   * @param skuId the SKU identifier
   * @param remainingStock the remaining stock from Redis
   * @param soldCount the sold count from Redis
   * @return StockInfo with calculated total stock
   */
  public static StockInfo fromRedis(String skuId, int remainingStock, int soldCount) {
    int totalStock = remainingStock + soldCount;
    return new StockInfo(skuId, totalStock, soldCount, remainingStock);
  }

  /**
   * Factory method to create an empty StockInfo when SKU is not found.
   *
   * @param skuId the SKU identifier
   * @return StockInfo with zero values
   */
  public static StockInfo empty(String skuId) {
    return new StockInfo(skuId, 0, 0, 0);
  }

  /**
   * Check if stock is available.
   *
   * @return true if remaining stock is greater than 0
   */
  public boolean isAvailable() {
    return remainingStock > 0;
  }

  /**
   * Check if stock is sold out.
   *
   * @return true if remaining stock is 0 or less
   */
  public boolean isSoldOut() {
    return remainingStock <= 0;
  }
}
