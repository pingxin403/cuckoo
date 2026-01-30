package com.pingxin403.cuckoo.flashsale.service.impl;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.pingxin403.cuckoo.flashsale.model.ReconciliationLog;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.repository.ReconciliationLogRepository;
import com.pingxin403.cuckoo.flashsale.repository.SeckillActivityRepository;
import com.pingxin403.cuckoo.flashsale.repository.SeckillOrderRepository;
import com.pingxin403.cuckoo.flashsale.service.ReconciliationService;
import com.pingxin403.cuckoo.flashsale.service.dto.Discrepancy;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

/**
 * 对账服务实现类 Implementation of ReconciliationService for data consistency verification.
 *
 * <p>Performs periodic reconciliation between Redis stock data and MySQL order data to detect and
 * fix data inconsistencies.
 *
 * <p>Validates Requirements: 6.4, 6.5, 6.6, 6.7
 */
@Service
public class ReconciliationServiceImpl implements ReconciliationService {

  private static final Logger logger = LoggerFactory.getLogger(ReconciliationServiceImpl.class);

  /** Redis key prefix for remaining stock */
  private static final String STOCK_KEY_PREFIX = "stock:sku_";

  /** Redis key prefix for sold count */
  private static final String SOLD_KEY_PREFIX = "sold:sku_";

  private final StringRedisTemplate stringRedisTemplate;
  private final SeckillActivityRepository activityRepository;
  private final SeckillOrderRepository orderRepository;
  private final ReconciliationLogRepository reconciliationLogRepository;
  private final ObjectMapper objectMapper;

  /**
   * Constructor with dependency injection.
   *
   * @param stringRedisTemplate Redis template for string operations
   * @param activityRepository Repository for activity data
   * @param orderRepository Repository for order data
   * @param reconciliationLogRepository Repository for reconciliation logs
   * @param objectMapper JSON object mapper
   */
  public ReconciliationServiceImpl(
      StringRedisTemplate stringRedisTemplate,
      SeckillActivityRepository activityRepository,
      SeckillOrderRepository orderRepository,
      ReconciliationLogRepository reconciliationLogRepository,
      ObjectMapper objectMapper) {
    this.stringRedisTemplate = stringRedisTemplate;
    this.activityRepository = activityRepository;
    this.orderRepository = orderRepository;
    this.reconciliationLogRepository = reconciliationLogRepository;
    this.objectMapper = objectMapper;
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirements 6.4, 6.5
   */
  @Override
  @Transactional(readOnly = true)
  public ReconciliationResult reconcile(String skuId) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("Reconcile failed: skuId is null or blank");
      return ReconciliationResult.failure(skuId, 0, 0, 0, List.of());
    }

    logger.info("Starting reconciliation for skuId={}", skuId);

    try {
      // Get Redis stock data
      int redisStock = getRedisStock(skuId);
      int redisSold = getRedisSold(skuId);

      // Get MySQL order count (only count PENDING_PAYMENT and PAID orders)
      List<OrderStatus> validStatuses = List.of(OrderStatus.PENDING_PAYMENT, OrderStatus.PAID);
      long mysqlOrderCount = orderRepository.countBySkuIdAndStatusIn(skuId, validStatuses);

      // Get total stock from activity (if exists)
      Integer totalStock = getTotalStockFromActivity(skuId);

      // Detect discrepancies
      List<Discrepancy> discrepancies = new ArrayList<>();

      // Check 1: Redis sold count should match MySQL order count
      if (redisSold != mysqlOrderCount) {
        discrepancies.add(Discrepancy.orderCountMismatch(skuId, redisSold, (int) mysqlOrderCount));
        logger.warn(
            "Order count mismatch detected: skuId={}, redisSold={}, mysqlOrderCount={}",
            skuId,
            redisSold,
            mysqlOrderCount);
      }

      // Check 2: Total stock consistency (if activity exists)
      if (totalStock != null) {
        int calculatedTotal = redisStock + redisSold;
        if (calculatedTotal != totalStock) {
          discrepancies.add(
              Discrepancy.totalStockMismatch(skuId, totalStock, redisStock, redisSold));
          logger.warn(
              "Total stock mismatch detected: skuId={}, totalStock={}, redisStock={}, redisSold={},"
                  + " calculated={}",
              skuId,
              totalStock,
              redisStock,
              redisSold,
              calculatedTotal);
        }
      }

      // Create reconciliation result
      ReconciliationResult result;
      if (discrepancies.isEmpty()) {
        result = ReconciliationResult.success(skuId, redisStock, redisSold, (int) mysqlOrderCount);
        logger.info("Reconciliation passed for skuId={}", skuId);
      } else {
        result =
            ReconciliationResult.failure(
                skuId, redisStock, redisSold, (int) mysqlOrderCount, discrepancies);
        logger.warn(
            "Reconciliation failed for skuId={}, discrepancies={}", skuId, discrepancies.size());
      }

      // Save reconciliation log
      saveReconciliationLog(result);

      return result;

    } catch (Exception e) {
      logger.error("Error during reconciliation for skuId={}", skuId, e);
      return ReconciliationResult.failure(skuId, 0, 0, 0, List.of());
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirements 6.4, 6.5
   */
  @Override
  @Transactional(readOnly = true)
  public ReconciliationReport fullReconcile() {
    logger.info("Starting full reconciliation");

    try {
      // Get all active and in-progress activities
      List<SeckillActivity> activities = new ArrayList<>();
      activities.addAll(activityRepository.findByStatus(ActivityStatus.IN_PROGRESS));

      // Also include recently ended activities (for completeness)
      activities.addAll(activityRepository.findByStatus(ActivityStatus.ENDED));

      logger.info("Found {} activities to reconcile", activities.size());

      // Reconcile each SKU
      List<ReconciliationResult> results = new ArrayList<>();
      for (SeckillActivity activity : activities) {
        ReconciliationResult result = reconcile(activity.getSkuId());
        results.add(result);
      }

      // Generate report
      ReconciliationReport report = ReconciliationReport.from(results);

      logger.info(
          "Full reconciliation completed: total={}, passed={}, failed={}",
          report.totalSkus(),
          report.passedSkus(),
          report.failedSkus());

      // Log warning if there are failures
      if (!report.allPassed()) {
        logger.warn(
            "Reconciliation found {} discrepancies across {} SKUs",
            report.getTotalDiscrepancies(),
            report.failedSkus());
      }

      return report;

    } catch (Exception e) {
      logger.error("Error during full reconciliation", e);
      return ReconciliationReport.from(List.of());
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates: Requirement 6.7
   */
  @Override
  @Transactional
  public boolean fixDiscrepancy(Discrepancy discrepancy) {
    if (discrepancy == null) {
      logger.warn("FixDiscrepancy failed: discrepancy is null");
      return false;
    }

    logger.info("Attempting to fix discrepancy: type={}", discrepancy.type());

    try {
      Map<String, Object> details = discrepancy.details();

      switch (discrepancy.type()) {
        case "ORDER_COUNT_MISMATCH":
          return fixOrderCountMismatch(details);

        case "TOTAL_STOCK_MISMATCH":
          return fixTotalStockMismatch(details);

        case "STOCK_MISMATCH":
          return fixStockMismatch(details);

        default:
          logger.warn("Unknown discrepancy type: {}", discrepancy.type());
          return false;
      }

    } catch (Exception e) {
      logger.error("Error fixing discrepancy: type={}", discrepancy.type(), e);
      return false;
    }
  }

  /**
   * Get Redis stock count for a SKU.
   *
   * @param skuId the SKU identifier
   * @return stock count, or 0 if not found
   */
  private int getRedisStock(String skuId) {
    try {
      String stockKey = STOCK_KEY_PREFIX + skuId;
      String value = stringRedisTemplate.opsForValue().get(stockKey);
      return value != null ? Integer.parseInt(value) : 0;
    } catch (Exception e) {
      logger.error("Error getting Redis stock for skuId={}", skuId, e);
      return 0;
    }
  }

  /**
   * Get Redis sold count for a SKU.
   *
   * @param skuId the SKU identifier
   * @return sold count, or 0 if not found
   */
  private int getRedisSold(String skuId) {
    try {
      String soldKey = SOLD_KEY_PREFIX + skuId;
      String value = stringRedisTemplate.opsForValue().get(soldKey);
      return value != null ? Integer.parseInt(value) : 0;
    } catch (Exception e) {
      logger.error("Error getting Redis sold count for skuId={}", skuId, e);
      return 0;
    }
  }

  /**
   * Get total stock from activity.
   *
   * @param skuId the SKU identifier
   * @return total stock, or null if activity not found
   */
  private Integer getTotalStockFromActivity(String skuId) {
    try {
      return activityRepository
          .findActiveActivityBySkuId(skuId)
          .map(SeckillActivity::getTotalStock)
          .orElse(null);
    } catch (Exception e) {
      logger.error("Error getting total stock from activity for skuId={}", skuId, e);
      return null;
    }
  }

  /**
   * Save reconciliation log to database.
   *
   * @param result the reconciliation result
   */
  private void saveReconciliationLog(ReconciliationResult result) {
    try {
      ReconciliationLog log;

      if (result.passed()) {
        log =
            ReconciliationLog.createNormal(
                result.skuId(),
                result.redisStock(),
                result.redisSoldCount(),
                result.mysqlOrderCount());
      } else {
        String detailsJson = serializeDiscrepancies(result.discrepancies());
        log =
            ReconciliationLog.createDiscrepancy(
                result.skuId(),
                result.redisStock(),
                result.redisSoldCount(),
                result.mysqlOrderCount(),
                result.getDiscrepancyCount(),
                detailsJson);
      }

      reconciliationLogRepository.save(log);
      logger.debug("Reconciliation log saved for skuId={}", result.skuId());

    } catch (Exception e) {
      logger.error("Failed to save reconciliation log for skuId={}", result.skuId(), e);
    }
  }

  /**
   * Serialize discrepancies to JSON string.
   *
   * @param discrepancies list of discrepancies
   * @return JSON string
   */
  private String serializeDiscrepancies(List<Discrepancy> discrepancies) {
    try {
      return objectMapper.writeValueAsString(discrepancies);
    } catch (JsonProcessingException e) {
      logger.error("Failed to serialize discrepancies", e);
      return "[]";
    }
  }

  /**
   * Fix order count mismatch by adjusting Redis sold count to match MySQL.
   *
   * <p>MySQL is the source of truth for order data. This method updates Redis sold count to match
   * the actual order count in MySQL.
   *
   * @param details discrepancy details containing redisSold, mysqlCount, and skuId
   * @return true if fixed successfully
   */
  private boolean fixOrderCountMismatch(Map<String, Object> details) {
    try {
      int redisSold = (Integer) details.get("redisSold");
      int mysqlCount = (Integer) details.get("mysqlCount");

      // Extract skuId from details if available
      String skuId = extractSkuIdFromDetails(details);
      if (skuId == null) {
        logger.error("Cannot fix order count mismatch: skuId not found in details");
        return false;
      }

      logger.info(
          "Fixing order count mismatch for skuId={}: redisSold={}, mysqlCount={}",
          skuId,
          redisSold,
          mysqlCount);

      // Update Redis sold count to match MySQL
      String soldKey = SOLD_KEY_PREFIX + skuId;
      stringRedisTemplate.opsForValue().set(soldKey, String.valueOf(mysqlCount));

      logger.info(
          "Successfully fixed order count mismatch for skuId={}: updated Redis sold count to {}",
          skuId,
          mysqlCount);

      // Mark the reconciliation log as fixed
      markReconciliationAsFixed(skuId);

      return true;
    } catch (Exception e) {
      logger.error("Error fixing order count mismatch", e);
      return false;
    }
  }

  /**
   * Fix total stock mismatch by adjusting Redis stock values.
   *
   * <p>Recalculates Redis stock based on total stock and MySQL order count (source of truth).
   * Formula: redisStock = totalStock - mysqlOrderCount
   *
   * @param details discrepancy details containing totalStock, redisStock, redisSold, and skuId
   * @return true if fixed successfully
   */
  private boolean fixTotalStockMismatch(Map<String, Object> details) {
    try {
      int totalStock = (Integer) details.get("totalStock");
      int redisStock = (Integer) details.get("redisStock");
      int redisSold = (Integer) details.get("redisSold");

      // Extract skuId from details if available
      String skuId = extractSkuIdFromDetails(details);
      if (skuId == null) {
        logger.error("Cannot fix total stock mismatch: skuId not found in details");
        return false;
      }

      logger.info(
          "Fixing total stock mismatch for skuId={}: totalStock={}, redisStock={}, redisSold={}",
          skuId,
          totalStock,
          redisStock,
          redisSold);

      // Get actual MySQL order count (source of truth)
      List<OrderStatus> validStatuses = List.of(OrderStatus.PENDING_PAYMENT, OrderStatus.PAID);
      long mysqlOrderCount = orderRepository.countBySkuIdAndStatusIn(skuId, validStatuses);

      // Recalculate correct values
      int correctSold = (int) mysqlOrderCount;
      int correctStock = totalStock - correctSold;

      if (correctStock < 0) {
        logger.error(
            "Cannot fix total stock mismatch: calculated stock is negative (totalStock={}, sold={})",
            totalStock,
            correctSold);
        return false;
      }

      // Update Redis with correct values
      String stockKey = STOCK_KEY_PREFIX + skuId;
      String soldKey = SOLD_KEY_PREFIX + skuId;

      stringRedisTemplate.opsForValue().set(stockKey, String.valueOf(correctStock));
      stringRedisTemplate.opsForValue().set(soldKey, String.valueOf(correctSold));

      logger.info(
          "Successfully fixed total stock mismatch for skuId={}: stock={}, sold={}",
          skuId,
          correctStock,
          correctSold);

      // Mark the reconciliation log as fixed
      markReconciliationAsFixed(skuId);

      return true;
    } catch (Exception e) {
      logger.error("Error fixing total stock mismatch", e);
      return false;
    }
  }

  /**
   * Fix stock mismatch by adjusting Redis stock value.
   *
   * <p>Updates Redis stock to match the expected value based on activity configuration.
   *
   * @param details discrepancy details containing expected, actual, and skuId
   * @return true if fixed successfully
   */
  private boolean fixStockMismatch(Map<String, Object> details) {
    try {
      int expected = (Integer) details.get("expected");
      int actual = (Integer) details.get("actual");

      // Extract skuId from details if available
      String skuId = extractSkuIdFromDetails(details);
      if (skuId == null) {
        logger.error("Cannot fix stock mismatch: skuId not found in details");
        return false;
      }

      logger.info(
          "Fixing stock mismatch for skuId={}: expected={}, actual={}", skuId, expected, actual);

      // Update Redis stock to expected value
      String stockKey = STOCK_KEY_PREFIX + skuId;
      stringRedisTemplate.opsForValue().set(stockKey, String.valueOf(expected));

      logger.info(
          "Successfully fixed stock mismatch for skuId={}: updated Redis stock to {}",
          skuId,
          expected);

      // Mark the reconciliation log as fixed
      markReconciliationAsFixed(skuId);

      return true;
    } catch (Exception e) {
      logger.error("Error fixing stock mismatch", e);
      return false;
    }
  }

  /**
   * Extract skuId from discrepancy details.
   *
   * <p>Note: Current implementation requires skuId to be added to discrepancy details. This is a
   * limitation that should be addressed by updating the Discrepancy creation methods to include
   * skuId.
   *
   * @param details discrepancy details map
   * @return skuId if found, null otherwise
   */
  private String extractSkuIdFromDetails(Map<String, Object> details) {
    Object skuIdObj = details.get("skuId");
    if (skuIdObj instanceof String) {
      return (String) skuIdObj;
    }
    return null;
  }

  /**
   * Mark the most recent reconciliation log for a SKU as fixed.
   *
   * @param skuId the SKU identifier
   */
  private void markReconciliationAsFixed(String skuId) {
    try {
      reconciliationLogRepository
          .findFirstBySkuIdOrderByCreatedAtDesc(skuId)
          .ifPresent(
              log -> {
                if (log.getStatus()
                    == com.pingxin403.cuckoo.flashsale.model.enums.ReconciliationStatus
                        .DISCREPANCY) {
                  reconciliationLogRepository.markAsFixed(log.getId());
                  logger.info(
                      "Marked reconciliation log as fixed: id={}, skuId={}", log.getId(), skuId);
                }
              });
    } catch (Exception e) {
      logger.error("Error marking reconciliation as fixed for skuId={}", skuId, e);
    }
  }
}
