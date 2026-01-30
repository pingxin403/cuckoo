package com.pingxin403.cuckoo.flashsale.service.dto;

import java.util.List;

/**
 * 批量创建订单结果 Result of batch order creation operation.
 *
 * <p>This record contains the results of a batch order creation operation, including:
 *
 * <ul>
 *   <li>Total number of orders processed
 *   <li>Number of successfully created orders
 *   <li>Number of failed orders
 *   <li>List of failed order IDs for retry or investigation
 * </ul>
 *
 * @param totalCount total number of orders in the batch
 * @param successCount number of successfully created orders
 * @param failedCount number of failed orders
 * @param failedOrderIds list of order IDs that failed to create
 */
public record BatchCreateResult(
    int totalCount, int successCount, int failedCount, List<String> failedOrderIds) {

  /**
   * Creates a successful result where all orders were created.
   *
   * @param count the number of orders created
   * @return a BatchCreateResult indicating full success
   */
  public static BatchCreateResult success(int count) {
    return new BatchCreateResult(count, count, 0, List.of());
  }

  /**
   * Creates a partial success result.
   *
   * @param totalCount total orders attempted
   * @param successCount orders successfully created
   * @param failedOrderIds list of failed order IDs
   * @return a BatchCreateResult with partial success
   */
  public static BatchCreateResult partial(
      int totalCount, int successCount, List<String> failedOrderIds) {
    return new BatchCreateResult(totalCount, successCount, failedOrderIds.size(), failedOrderIds);
  }

  /**
   * Checks if all orders were successfully created.
   *
   * @return true if all orders succeeded
   */
  public boolean isFullSuccess() {
    return failedCount == 0 && successCount == totalCount;
  }

  /**
   * Checks if any orders were successfully created.
   *
   * @return true if at least one order succeeded
   */
  public boolean hasSuccess() {
    return successCount > 0;
  }
}
