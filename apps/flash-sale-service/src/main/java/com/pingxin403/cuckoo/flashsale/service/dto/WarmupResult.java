package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 库存预热结果 Result record for stock warmup operations.
 *
 * <p>Contains the outcome of loading stock from database to Redis cache.
 *
 * @param success whether the warmup operation succeeded
 * @param skuId the SKU identifier
 * @param stock the stock quantity loaded to Redis
 * @param message additional message (e.g., error details)
 */
public record WarmupResult(boolean success, String skuId, int stock, String message) {

  /**
   * Factory method to create a successful warmup result.
   *
   * @param skuId the SKU identifier
   * @param stock the stock quantity loaded
   * @return successful WarmupResult
   */
  public static WarmupResult success(String skuId, int stock) {
    return new WarmupResult(true, skuId, stock, "库存预热成功");
  }

  /**
   * Factory method to create a failed warmup result.
   *
   * @param skuId the SKU identifier
   * @param message error message
   * @return failed WarmupResult
   */
  public static WarmupResult failure(String skuId, String message) {
    return new WarmupResult(false, skuId, 0, message);
  }
}
