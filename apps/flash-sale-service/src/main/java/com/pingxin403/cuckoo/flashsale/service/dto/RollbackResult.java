package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 库存回滚结果 Result record for stock rollback operations.
 *
 * <p>Contains the outcome of rolling back previously deducted stock (e.g., when order times out or
 * is cancelled).
 *
 * @param success whether the rollback operation succeeded
 * @param skuId the SKU identifier
 * @param orderId the order ID associated with the rollback
 * @param quantity the quantity rolled back
 * @param newStock the new stock count after rollback
 * @param message additional message (e.g., error details)
 */
public record RollbackResult(
    boolean success, String skuId, String orderId, int quantity, int newStock, String message) {

  /**
   * Factory method to create a successful rollback result.
   *
   * @param skuId the SKU identifier
   * @param orderId the order ID
   * @param quantity the quantity rolled back
   * @param newStock the new stock count after rollback
   * @return successful RollbackResult
   */
  public static RollbackResult success(String skuId, String orderId, int quantity, int newStock) {
    return new RollbackResult(true, skuId, orderId, quantity, newStock, "库存回滚成功");
  }

  /**
   * Factory method to create a failed rollback result.
   *
   * @param skuId the SKU identifier
   * @param orderId the order ID
   * @param message error message
   * @return failed RollbackResult
   */
  public static RollbackResult failure(String skuId, String orderId, String message) {
    return new RollbackResult(false, skuId, orderId, 0, -1, message);
  }
}
