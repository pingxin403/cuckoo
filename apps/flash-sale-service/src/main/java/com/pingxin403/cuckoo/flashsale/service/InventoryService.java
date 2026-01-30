package com.pingxin403.cuckoo.flashsale.service;

import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

/**
 * 库存服务接口 Inventory service interface for flash sale stock management.
 *
 * <p>Provides atomic stock operations using Redis Lua scripts to ensure:
 *
 * <ul>
 *   <li>Strong consistency - No overselling under high concurrency
 *   <li>High performance - Target ≥50K QPS per Redis instance
 *   <li>Fault tolerance - Graceful handling of Redis failures
 * </ul>
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>stock:sku_{skuId} - Remaining stock count
 *   <li>sold:sku_{skuId} - Sold count
 * </ul>
 *
 * @see DeductResult
 * @see StockInfo
 * @see WarmupResult
 * @see RollbackResult
 */
public interface InventoryService {

  /**
   * 库存预热 - 将数据库库存加载到Redis Warmup stock - Load stock from database to Redis cache.
   *
   * <p>This method should be called before a flash sale activity starts to ensure stock data is
   * available in Redis for fast access.
   *
   * <p>Sets:
   *
   * <ul>
   *   <li>stock:sku_{skuId} = stock (remaining stock)
   *   <li>sold:sku_{skuId} = 0 (sold count initialized to 0)
   * </ul>
   *
   * <p>Validates: Requirement 1.1
   *
   * @param skuId the SKU identifier
   * @param stock the initial stock quantity to load
   * @return WarmupResult indicating success or failure
   */
  WarmupResult warmupStock(String skuId, int stock);

  /**
   * 原子扣减库存 - 使用Lua脚本 Atomic stock deduction using Redis Lua script.
   *
   * <p>Performs atomic check-and-deduct operation to prevent overselling:
   *
   * <ol>
   *   <li>Check if remaining stock >= requested quantity
   *   <li>If sufficient, atomically decrement stock and increment sold count
   *   <li>Generate unique order ID on success
   * </ol>
   *
   * <p>Validates: Requirements 1.2, 1.3, 1.4
   *
   * @param skuId the SKU identifier
   * @param userId the user ID making the purchase
   * @param quantity the quantity to deduct
   * @return DeductResult with success status, result code, remaining stock, and order ID
   */
  DeductResult deductStock(String skuId, String userId, int quantity);

  /**
   * 回滚库存 - 订单超时未支付时调用 Rollback stock - Called when order times out or is cancelled.
   *
   * <p>Atomically restores previously deducted stock:
   *
   * <ol>
   *   <li>Increment remaining stock by quantity
   *   <li>Decrement sold count by quantity
   * </ol>
   *
   * <p>Validates: Requirement 1.5
   *
   * @param skuId the SKU identifier
   * @param orderId the order ID associated with the rollback
   * @param quantity the quantity to rollback
   * @return RollbackResult indicating success or failure
   */
  RollbackResult rollbackStock(String skuId, String orderId, int quantity);

  /**
   * 查询剩余库存 Query remaining stock information.
   *
   * <p>Retrieves current stock status from Redis cache.
   *
   * @param skuId the SKU identifier
   * @return StockInfo containing total stock, sold count, and remaining stock
   */
  StockInfo getStock(String skuId);
}
