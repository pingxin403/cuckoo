package com.pingxin403.cuckoo.flashsale.service.impl;

import java.util.Arrays;
import java.util.UUID;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.RedisConnectionFailureException;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.stereotype.Service;

import com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics;
import com.pingxin403.cuckoo.flashsale.model.StockLog;
import com.pingxin403.cuckoo.flashsale.repository.StockLogRepository;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

/**
 * 库存服务实现类 Implementation of InventoryService using Redis for atomic stock operations.
 *
 * <p>Uses Redis Lua scripts to ensure atomic check-and-deduct operations, preventing overselling
 * under high concurrency scenarios.
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>stock:sku_{skuId} - Remaining stock count (Integer)
 *   <li>sold:sku_{skuId} - Sold count (Integer)
 * </ul>
 *
 * <p>Validates Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.7
 */
@Service
public class InventoryServiceImpl implements InventoryService {

  private static final Logger logger = LoggerFactory.getLogger(InventoryServiceImpl.class);

  /** Redis key prefix for remaining stock */
  private static final String STOCK_KEY_PREFIX = "stock:sku_";

  /** Redis key prefix for sold count */
  private static final String SOLD_KEY_PREFIX = "sold:sku_";

  private final StringRedisTemplate stringRedisTemplate;
  private final DefaultRedisScript<Long> stockDeductScript;
  private final DefaultRedisScript<Long> stockRollbackScript;
  private final StockLogRepository stockLogRepository;
  private final FlashSaleMetrics metrics;

  /**
   * Constructor with dependency injection.
   *
   * @param stringRedisTemplate Redis template for string operations
   * @param stockDeductScript Lua script for atomic stock deduction
   * @param stockRollbackScript Lua script for atomic stock rollback
   * @param stockLogRepository Repository for stock operation logs
   * @param metrics Metrics service for recording operations
   */
  public InventoryServiceImpl(
      StringRedisTemplate stringRedisTemplate,
      DefaultRedisScript<Long> stockDeductScript,
      DefaultRedisScript<Long> stockRollbackScript,
      StockLogRepository stockLogRepository,
      FlashSaleMetrics metrics) {
    this.stringRedisTemplate = stringRedisTemplate;
    this.stockDeductScript = stockDeductScript;
    this.stockRollbackScript = stockRollbackScript;
    this.stockLogRepository = stockLogRepository;
    this.metrics = metrics;
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirement 1.1 - Stock warmup to Redis
   */
  @Override
  public WarmupResult warmupStock(String skuId, int stock) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("Warmup failed: skuId is null or blank");
      return WarmupResult.failure(skuId, "SKU ID不能为空");
    }

    if (stock < 0) {
      logger.warn("Warmup failed: stock cannot be negative, skuId={}, stock={}", skuId, stock);
      return WarmupResult.failure(skuId, "库存数量不能为负数");
    }

    String stockKey = getStockKey(skuId);
    String soldKey = getSoldKey(skuId);

    try {
      // Set remaining stock
      stringRedisTemplate.opsForValue().set(stockKey, String.valueOf(stock));
      // Initialize sold count to 0
      stringRedisTemplate.opsForValue().set(soldKey, "0");

      // Register inventory gauge for this SKU
      metrics.registerInventoryGauge(skuId, () -> getCurrentStock(skuId));

      logger.info("Stock warmup successful: skuId={}, stock={}", skuId, stock);
      return WarmupResult.success(skuId, stock);

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during warmup: skuId={}", skuId, e);
      return WarmupResult.failure(skuId, "Redis连接失败: " + e.getMessage());
    } catch (Exception e) {
      logger.error("Unexpected error during warmup: skuId={}", skuId, e);
      return WarmupResult.failure(skuId, "系统错误: " + e.getMessage());
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirements 1.2, 1.3, 1.4, 1.7
   *
   * <ul>
   *   <li>1.2 - Atomic check and deduct using Lua script
   *   <li>1.3 - Returns SUCCESS with remaining stock when sufficient
   *   <li>1.4 - Returns OUT_OF_STOCK when insufficient, no deduction performed
   *   <li>1.7 - Returns SYSTEM_ERROR on Redis failure
   * </ul>
   */
  @Override
  public DeductResult deductStock(String skuId, String userId, int quantity) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("Deduct failed: skuId is null or blank");
      return DeductResult.systemError();
    }

    if (userId == null || userId.isBlank()) {
      logger.warn("Deduct failed: userId is null or blank, skuId={}", skuId);
      return DeductResult.systemError();
    }

    if (quantity <= 0) {
      logger.warn(
          "Deduct failed: quantity must be positive, skuId={}, quantity={}", skuId, quantity);
      return DeductResult.systemError();
    }

    String stockKey = getStockKey(skuId);
    String soldKey = getSoldKey(skuId);

    try {
      // Get current stock before deduction for logging
      int beforeStock = getCurrentStock(skuId);

      // Record deduction duration
      return metrics
          .getInventoryDeductionDuration()
          .record(
              () -> {
                try {
                  // Execute Lua script for atomic deduction
                  Long result =
                      stringRedisTemplate.execute(
                          stockDeductScript,
                          Arrays.asList(stockKey, soldKey),
                          String.valueOf(quantity));

                  if (result == null) {
                    logger.error("Lua script returned null result: skuId={}", skuId);
                    metrics.recordInventoryDeduction(skuId, false);
                    return DeductResult.systemError();
                  }

                  long resultValue = result;

                  // Handle Lua script return values:
                  // > 0 = remaining stock (success)
                  // 0 = out of stock
                  // -1 = invalid input (error)
                  if (resultValue == -1) {
                    logger.error(
                        "Lua script returned error (-1): skuId={}, quantity={}", skuId, quantity);
                    metrics.recordInventoryDeduction(skuId, false);
                    return DeductResult.systemError();
                  }

                  if (resultValue == 0) {
                    logger.info("Out of stock: skuId={}, requestedQuantity={}", skuId, quantity);
                    metrics.recordInventoryDeduction(skuId, false);
                    return DeductResult.outOfStock(0);
                  }

                  // Success - generate order ID and log the operation
                  String orderId = generateOrderId();
                  int remainingStock = (int) resultValue;

                  // Log the stock operation
                  logStockDeduction(skuId, orderId, quantity, beforeStock, remainingStock);

                  // Record successful deduction
                  metrics.recordInventoryDeduction(skuId, true);

                  logger.info(
                      "Stock deduction successful: skuId={}, userId={}, quantity={}, remainingStock={}, orderId={}",
                      skuId,
                      userId,
                      quantity,
                      remainingStock,
                      orderId);

                  return DeductResult.success(remainingStock, orderId);

                } catch (RedisConnectionFailureException e) {
                  logger.error("Redis connection failure during deduction: skuId={}", skuId, e);
                  metrics.recordInventoryDeduction(skuId, false);
                  return DeductResult.systemError();
                } catch (Exception e) {
                  logger.error("Unexpected error during deduction: skuId={}", skuId, e);
                  metrics.recordInventoryDeduction(skuId, false);
                  return DeductResult.systemError();
                }
              });

    } catch (Exception e) {
      logger.error("Unexpected error in deductStock wrapper: skuId={}", skuId, e);
      metrics.recordInventoryDeduction(skuId, false);
      return DeductResult.systemError();
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirement 1.5 - Stock rollback for unpaid orders
   */
  @Override
  public RollbackResult rollbackStock(String skuId, String orderId, int quantity) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("Rollback failed: skuId is null or blank");
      return RollbackResult.failure(skuId, orderId, "SKU ID不能为空");
    }

    if (orderId == null || orderId.isBlank()) {
      logger.warn("Rollback failed: orderId is null or blank, skuId={}", skuId);
      return RollbackResult.failure(skuId, orderId, "订单ID不能为空");
    }

    if (quantity <= 0) {
      logger.warn(
          "Rollback failed: quantity must be positive, skuId={}, quantity={}", skuId, quantity);
      return RollbackResult.failure(skuId, orderId, "回滚数量必须为正数");
    }

    String stockKey = getStockKey(skuId);
    String soldKey = getSoldKey(skuId);

    try {
      // Get current stock before rollback for logging
      int beforeStock = getCurrentStock(skuId);

      // Execute Lua script for atomic rollback
      Long result =
          stringRedisTemplate.execute(
              stockRollbackScript, Arrays.asList(stockKey, soldKey), String.valueOf(quantity));

      if (result == null) {
        logger.error(
            "Rollback Lua script returned null result: skuId={}, orderId={}", skuId, orderId);
        return RollbackResult.failure(skuId, orderId, "系统错误: Lua脚本返回空值");
      }

      long resultValue = result;

      // Handle Lua script return values:
      // > 0 = new stock count (success)
      // -1 = invalid input (error)
      if (resultValue == -1) {
        logger.error(
            "Rollback Lua script returned error (-1): skuId={}, orderId={}, quantity={}",
            skuId,
            orderId,
            quantity);
        return RollbackResult.failure(skuId, orderId, "系统错误: 无效的回滚参数");
      }

      int newStock = (int) resultValue;

      // Log the stock rollback operation
      logStockRollback(skuId, orderId, quantity, beforeStock, newStock);

      // Record rollback metric
      metrics.recordInventoryRollback(skuId);

      logger.info(
          "Stock rollback successful: skuId={}, orderId={}, quantity={}, newStock={}",
          skuId,
          orderId,
          quantity,
          newStock);

      return RollbackResult.success(skuId, orderId, quantity, newStock);

    } catch (RedisConnectionFailureException e) {
      logger.error(
          "Redis connection failure during rollback: skuId={}, orderId={}", skuId, orderId, e);
      return RollbackResult.failure(skuId, orderId, "Redis连接失败: " + e.getMessage());
    } catch (Exception e) {
      logger.error("Unexpected error during rollback: skuId={}, orderId={}", skuId, orderId, e);
      return RollbackResult.failure(skuId, orderId, "系统错误: " + e.getMessage());
    }
  }

  /** {@inheritDoc} */
  @Override
  public StockInfo getStock(String skuId) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("GetStock failed: skuId is null or blank");
      return StockInfo.empty(skuId);
    }

    String stockKey = getStockKey(skuId);
    String soldKey = getSoldKey(skuId);

    try {
      String stockValue = stringRedisTemplate.opsForValue().get(stockKey);
      String soldValue = stringRedisTemplate.opsForValue().get(soldKey);

      int remainingStock = stockValue != null ? Integer.parseInt(stockValue) : 0;
      int soldCount = soldValue != null ? Integer.parseInt(soldValue) : 0;

      return StockInfo.fromRedis(skuId, remainingStock, soldCount);

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during getStock: skuId={}", skuId, e);
      return StockInfo.empty(skuId);
    } catch (NumberFormatException e) {
      logger.error("Invalid stock value format in Redis: skuId={}", skuId, e);
      return StockInfo.empty(skuId);
    } catch (Exception e) {
      logger.error("Unexpected error during getStock: skuId={}", skuId, e);
      return StockInfo.empty(skuId);
    }
  }

  /**
   * Generate Redis key for remaining stock.
   *
   * @param skuId the SKU identifier
   * @return the Redis key
   */
  private String getStockKey(String skuId) {
    return STOCK_KEY_PREFIX + skuId;
  }

  /**
   * Generate Redis key for sold count.
   *
   * @param skuId the SKU identifier
   * @return the Redis key
   */
  private String getSoldKey(String skuId) {
    return SOLD_KEY_PREFIX + skuId;
  }

  /**
   * Generate a unique order ID.
   *
   * <p>Format: ORD-{timestamp}-{uuid-suffix}
   *
   * @return unique order ID
   */
  private String generateOrderId() {
    long timestamp = System.currentTimeMillis();
    String uuidSuffix = UUID.randomUUID().toString().substring(0, 8).toUpperCase();
    return String.format("ORD-%d-%s", timestamp, uuidSuffix);
  }

  /**
   * Get current stock from Redis.
   *
   * @param skuId the SKU identifier
   * @return current stock count, or 0 if not found
   */
  private int getCurrentStock(String skuId) {
    try {
      String stockValue = stringRedisTemplate.opsForValue().get(getStockKey(skuId));
      return stockValue != null ? Integer.parseInt(stockValue) : 0;
    } catch (Exception e) {
      logger.warn("Failed to get current stock for logging: skuId={}", skuId, e);
      return 0;
    }
  }

  /**
   * Log stock deduction operation to database.
   *
   * @param skuId the SKU identifier
   * @param orderId the order ID
   * @param quantity the quantity deducted
   * @param beforeStock stock before deduction
   * @param afterStock stock after deduction
   */
  private void logStockDeduction(
      String skuId, String orderId, int quantity, int beforeStock, int afterStock) {
    try {
      StockLog log = StockLog.createDeductLog(skuId, orderId, quantity, beforeStock, afterStock);
      stockLogRepository.save(log);
      logger.debug("Stock deduction logged: skuId={}, orderId={}", skuId, orderId);
    } catch (Exception e) {
      // Log failure should not affect the main operation
      logger.error("Failed to log stock deduction: skuId={}, orderId={}", skuId, orderId, e);
    }
  }

  /**
   * Log stock rollback operation to database.
   *
   * @param skuId the SKU identifier
   * @param orderId the order ID
   * @param quantity the quantity rolled back
   * @param beforeStock stock before rollback
   * @param afterStock stock after rollback
   */
  private void logStockRollback(
      String skuId, String orderId, int quantity, int beforeStock, int afterStock) {
    try {
      StockLog log = StockLog.createRollbackLog(skuId, orderId, quantity, beforeStock, afterStock);
      stockLogRepository.save(log);
      logger.debug("Stock rollback logged: skuId={}, orderId={}", skuId, orderId);
    } catch (Exception e) {
      // Log failure should not affect the main operation
      logger.error("Failed to log stock rollback: skuId={}, orderId={}", skuId, orderId, e);
    }
  }
}
