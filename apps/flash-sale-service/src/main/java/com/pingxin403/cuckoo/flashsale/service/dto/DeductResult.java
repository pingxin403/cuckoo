package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 库存扣减结果 Result record for stock deduction operations.
 *
 * <p>Contains the outcome of an atomic stock deduction attempt using Redis Lua script.
 *
 * @param success whether the deduction operation succeeded
 * @param code the result code indicating the outcome
 * @param remainingStock the remaining stock after deduction (or current stock if failed)
 * @param orderId the generated order ID (only set on success)
 */
public record DeductResult(
    boolean success, DeductResultCode code, int remainingStock, String orderId) {

  /**
   * Factory method to create a successful deduction result.
   *
   * @param remainingStock the remaining stock after deduction
   * @param orderId the generated order ID
   * @return successful DeductResult
   */
  public static DeductResult success(int remainingStock, String orderId) {
    return new DeductResult(true, DeductResultCode.SUCCESS, remainingStock, orderId);
  }

  /**
   * Factory method to create an out-of-stock result.
   *
   * @param currentStock the current stock (unchanged)
   * @return out-of-stock DeductResult
   */
  public static DeductResult outOfStock(int currentStock) {
    return new DeductResult(false, DeductResultCode.OUT_OF_STOCK, currentStock, null);
  }

  /**
   * Factory method to create a system error result.
   *
   * @return system error DeductResult
   */
  public static DeductResult systemError() {
    return new DeductResult(false, DeductResultCode.SYSTEM_ERROR, -1, null);
  }
}
