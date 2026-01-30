package com.pingxin403.cuckoo.flashsale.service.impl;

import java.util.UUID;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.RedisConnectionFailureException;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;

/**
 * 排队服务实现类 Implementation of QueueService using Redis token bucket algorithm.
 *
 * <p>Uses Redis to implement a distributed token bucket for rate limiting:
 *
 * <ul>
 *   <li>Tokens are stored in Redis and refilled at a configured rate
 *   <li>Each successful token acquisition allows one request to proceed
 *   <li>When no tokens are available, requests receive a queuing response
 * </ul>
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>token_bucket:{skuId} -> Integer (available tokens)
 *   <li>token_bucket_rate:{skuId} -> Integer (token generation rate per second)
 *   <li>token_bucket_last:{skuId} -> Long (last refill timestamp in milliseconds)
 *   <li>token_bucket_capacity:{skuId} -> Integer (maximum bucket capacity)
 *   <li>sold_out:{skuId} -> "1" (sold out flag)
 *   <li>order_status:{orderId} -> String (order status)
 * </ul>
 *
 * <p>Validates Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 */
@Service
public class QueueServiceImpl implements QueueService {

  private static final Logger logger = LoggerFactory.getLogger(QueueServiceImpl.class);

  /** Redis key prefix for token bucket available tokens */
  private static final String TOKEN_BUCKET_PREFIX = "token_bucket:";

  /** Redis key prefix for token generation rate */
  private static final String TOKEN_RATE_PREFIX = "token_bucket_rate:";

  /** Redis key prefix for last refill timestamp */
  private static final String TOKEN_LAST_PREFIX = "token_bucket_last:";

  /** Redis key prefix for bucket capacity */
  private static final String TOKEN_CAPACITY_PREFIX = "token_bucket_capacity:";

  /** Redis key prefix for sold out flag */
  private static final String SOLD_OUT_PREFIX = "sold_out:";

  /** Redis key prefix for order status cache */
  private static final String ORDER_STATUS_PREFIX = "order_status:";

  /** Default token generation rate (tokens per second) */
  private static final int DEFAULT_TOKEN_RATE = 1000;

  /** Default bucket capacity (maximum tokens) */
  private static final int DEFAULT_BUCKET_CAPACITY = 5000;

  /** Order status cache TTL in seconds (24 hours) */
  private static final long ORDER_STATUS_TTL = 24 * 60 * 60;

  private final StringRedisTemplate stringRedisTemplate;
  private final FlashSaleMetrics metrics;

  /**
   * Constructor with dependency injection.
   *
   * @param stringRedisTemplate Redis template for string operations
   * @param metrics Metrics service for recording operations
   */
  public QueueServiceImpl(StringRedisTemplate stringRedisTemplate, FlashSaleMetrics metrics) {
    this.stringRedisTemplate = stringRedisTemplate;
    this.metrics = metrics;
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirements: 4.1, 4.4
   *
   * <p>Implementation:
   *
   * <ol>
   *   <li>Check if SKU is sold out
   *   <li>Initialize token bucket if not exists
   *   <li>Refill tokens based on elapsed time
   *   <li>Try to consume one token
   *   <li>Return appropriate result
   * </ol>
   */
  @Override
  public QueueResult tryAcquireToken(String userId, String skuId) {
    if (userId == null || userId.isBlank()) {
      logger.warn("tryAcquireToken failed: userId is null or blank");
      return QueueResult.queuing(0, null);
    }

    if (skuId == null || skuId.isBlank()) {
      logger.warn("tryAcquireToken failed: skuId is null or blank");
      return QueueResult.queuing(0, null);
    }

    try {
      // Check if SKU is sold out
      if (isSoldOut(skuId)) {
        logger.info("SKU is sold out: skuId={}, userId={}", skuId, userId);
        return QueueResult.soldOut();
      }

      // Initialize token bucket if not exists
      initializeTokenBucketIfNeeded(skuId);

      // Refill tokens based on elapsed time
      refillTokens(skuId);

      // Try to consume one token
      boolean acquired = consumeToken(skuId);

      // Record metrics
      metrics.recordQueueTokenAcquisition(acquired);

      if (acquired) {
        String queueToken = generateQueueToken();
        logger.info(
            "Token acquired: skuId={}, userId={}, queueToken={}", skuId, userId, queueToken);
        return QueueResult.acquired(queueToken);
      } else {
        // No tokens available, calculate estimated wait time
        int estimatedWait = getEstimatedWaitTime(skuId);

        // Update queue length metric (estimated based on negative tokens)
        updateQueueLengthMetric(skuId);

        String queueToken = generateQueueToken();
        logger.info(
            "Token not available, queuing: skuId={}, userId={}, estimatedWait={}s",
            skuId,
            userId,
            estimatedWait);
        return QueueResult.queuing(estimatedWait, queueToken);
      }

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during tryAcquireToken: skuId={}", skuId, e);
      // On Redis failure, return queuing to avoid blocking users
      return QueueResult.queuing(5, null);
    } catch (Exception e) {
      logger.error("Unexpected error during tryAcquireToken: skuId={}", skuId, e);
      return QueueResult.queuing(5, null);
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 4.3
   */
  @Override
  public OrderStatusResult queryStatus(String orderId) {
    if (orderId == null || orderId.isBlank()) {
      logger.warn("queryStatus failed: orderId is null or blank");
      return OrderStatusResult.notFound(orderId);
    }

    try {
      String statusKey = getOrderStatusKey(orderId);
      String statusValue = stringRedisTemplate.opsForValue().get(statusKey);

      if (statusValue == null) {
        logger.info("Order status not found in cache: orderId={}", orderId);
        return OrderStatusResult.notFound(orderId);
      }

      // Parse status from Redis
      OrderStatus status = OrderStatus.valueOf(statusValue);

      return switch (status) {
        case PENDING_PAYMENT -> OrderStatusResult.pendingPayment(orderId);
        case PAID -> OrderStatusResult.paid(orderId);
        case CANCELLED -> OrderStatusResult.cancelled(orderId);
        case TIMEOUT -> OrderStatusResult.timeout(orderId);
      };

    } catch (IllegalArgumentException e) {
      logger.error("Invalid order status value in Redis: orderId={}", orderId, e);
      return OrderStatusResult.notFound(orderId);
    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during queryStatus: orderId={}", orderId, e);
      return OrderStatusResult.notFound(orderId);
    } catch (Exception e) {
      logger.error("Unexpected error during queryStatus: orderId={}", orderId, e);
      return OrderStatusResult.notFound(orderId);
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 4.5
   *
   * <p>Calculation: estimatedWait = max(0, -availableTokens / tokenRate)
   *
   * <p>If tokens are available (positive), wait time is 0. If tokens are negative (queue depth),
   * divide by rate to get wait time in seconds.
   */
  @Override
  public int getEstimatedWaitTime(String skuId) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("getEstimatedWaitTime failed: skuId is null or blank");
      return 0;
    }

    try {
      String tokenKey = getTokenBucketKey(skuId);
      String rateKey = getTokenRateKey(skuId);

      String tokenValue = stringRedisTemplate.opsForValue().get(tokenKey);
      String rateValue = stringRedisTemplate.opsForValue().get(rateKey);

      int availableTokens = tokenValue != null ? Integer.parseInt(tokenValue) : 0;
      int tokenRate = rateValue != null ? Integer.parseInt(rateValue) : DEFAULT_TOKEN_RATE;

      // If tokens are available, no wait time
      if (availableTokens > 0) {
        return 0;
      }

      // Calculate wait time based on queue depth (negative tokens) and rate
      // Use absolute value of negative tokens as queue depth
      int queueDepth = Math.abs(availableTokens);
      int estimatedWait = (int) Math.ceil((double) queueDepth / tokenRate);

      logger.debug(
          "Estimated wait time: skuId={}, queueDepth={}, tokenRate={}, estimatedWait={}s",
          skuId,
          queueDepth,
          tokenRate,
          estimatedWait);

      return estimatedWait;

    } catch (NumberFormatException e) {
      logger.error("Invalid token bucket value format: skuId={}", skuId, e);
      return 0;
    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during getEstimatedWaitTime: skuId={}", skuId, e);
      return 0;
    } catch (Exception e) {
      logger.error("Unexpected error during getEstimatedWaitTime: skuId={}", skuId, e);
      return 0;
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 4.6
   */
  @Override
  public void notifySoldOut(String skuId) {
    if (skuId == null || skuId.isBlank()) {
      logger.warn("notifySoldOut failed: skuId is null or blank");
      return;
    }

    try {
      String soldOutKey = getSoldOutKey(skuId);
      String tokenKey = getTokenBucketKey(skuId);

      // Set sold out flag (no expiration, will be cleared when activity ends)
      stringRedisTemplate.opsForValue().set(soldOutKey, "1");

      // Clear token bucket to prevent further token acquisition
      stringRedisTemplate.delete(tokenKey);

      logger.info("SKU marked as sold out: skuId={}", skuId);

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during notifySoldOut: skuId={}", skuId, e);
    } catch (Exception e) {
      logger.error("Unexpected error during notifySoldOut: skuId={}", skuId, e);
    }
  }

  /**
   * Check if a SKU is sold out.
   *
   * @param skuId the SKU identifier
   * @return true if sold out, false otherwise
   */
  private boolean isSoldOut(String skuId) {
    try {
      String soldOutKey = getSoldOutKey(skuId);
      String value = stringRedisTemplate.opsForValue().get(soldOutKey);
      return "1".equals(value);
    } catch (Exception e) {
      logger.warn("Failed to check sold out status: skuId={}", skuId, e);
      return false;
    }
  }

  /**
   * Initialize token bucket for a SKU if it doesn't exist.
   *
   * @param skuId the SKU identifier
   */
  private void initializeTokenBucketIfNeeded(String skuId) {
    try {
      String tokenKey = getTokenBucketKey(skuId);
      String rateKey = getTokenRateKey(skuId);
      String lastKey = getTokenLastKey(skuId);
      String capacityKey = getTokenCapacityKey(skuId);

      // Check if token bucket exists
      Boolean exists = stringRedisTemplate.hasKey(tokenKey);

      if (Boolean.FALSE.equals(exists)) {
        // Initialize token bucket with default values
        stringRedisTemplate.opsForValue().set(tokenKey, String.valueOf(DEFAULT_BUCKET_CAPACITY));
        stringRedisTemplate.opsForValue().set(rateKey, String.valueOf(DEFAULT_TOKEN_RATE));
        stringRedisTemplate.opsForValue().set(capacityKey, String.valueOf(DEFAULT_BUCKET_CAPACITY));
        stringRedisTemplate.opsForValue().set(lastKey, String.valueOf(System.currentTimeMillis()));

        logger.info(
            "Token bucket initialized: skuId={}, capacity={}, rate={}",
            skuId,
            DEFAULT_BUCKET_CAPACITY,
            DEFAULT_TOKEN_RATE);
      }

    } catch (Exception e) {
      logger.error("Failed to initialize token bucket: skuId={}", skuId, e);
    }
  }

  /**
   * Refill tokens based on elapsed time since last refill.
   *
   * <p>Formula: tokensToAdd = (currentTime - lastRefillTime) * tokenRate / 1000
   *
   * <p>Tokens are capped at bucket capacity.
   *
   * @param skuId the SKU identifier
   */
  private void refillTokens(String skuId) {
    try {
      String tokenKey = getTokenBucketKey(skuId);
      String rateKey = getTokenRateKey(skuId);
      String lastKey = getTokenLastKey(skuId);
      String capacityKey = getTokenCapacityKey(skuId);

      long currentTime = System.currentTimeMillis();

      String lastValue = stringRedisTemplate.opsForValue().get(lastKey);
      String rateValue = stringRedisTemplate.opsForValue().get(rateKey);
      String capacityValue = stringRedisTemplate.opsForValue().get(capacityKey);
      String tokenValue = stringRedisTemplate.opsForValue().get(tokenKey);

      if (lastValue == null || rateValue == null || capacityValue == null || tokenValue == null) {
        logger.warn("Token bucket not properly initialized: skuId={}", skuId);
        return;
      }

      long lastRefillTime = Long.parseLong(lastValue);
      int tokenRate = Integer.parseInt(rateValue);
      int capacity = Integer.parseInt(capacityValue);
      int currentTokens = Integer.parseInt(tokenValue);

      // Calculate elapsed time in seconds
      long elapsedMillis = currentTime - lastRefillTime;
      double elapsedSeconds = elapsedMillis / 1000.0;

      // Calculate tokens to add
      int tokensToAdd = (int) (elapsedSeconds * tokenRate);

      if (tokensToAdd > 0) {
        // Add tokens, capped at capacity
        int newTokens = Math.min(currentTokens + tokensToAdd, capacity);

        // Update Redis
        stringRedisTemplate.opsForValue().set(tokenKey, String.valueOf(newTokens));
        stringRedisTemplate.opsForValue().set(lastKey, String.valueOf(currentTime));

        logger.debug(
            "Tokens refilled: skuId={}, tokensAdded={}, newTokens={}, elapsedSeconds={}",
            skuId,
            tokensToAdd,
            newTokens,
            elapsedSeconds);
      }

    } catch (NumberFormatException e) {
      logger.error("Invalid token bucket value format during refill: skuId={}", skuId, e);
    } catch (Exception e) {
      logger.error("Failed to refill tokens: skuId={}", skuId, e);
    }
  }

  /**
   * Try to consume one token from the bucket.
   *
   * @param skuId the SKU identifier
   * @return true if token was consumed, false if no tokens available
   */
  private boolean consumeToken(String skuId) {
    try {
      String tokenKey = getTokenBucketKey(skuId);

      // Decrement token count atomically
      Long newValue = stringRedisTemplate.opsForValue().decrement(tokenKey);

      if (newValue == null) {
        logger.error("Failed to decrement token: skuId={}", skuId);
        return false;
      }

      // If result is non-negative, token was successfully consumed
      if (newValue >= 0) {
        logger.debug("Token consumed: skuId={}, remainingTokens={}", skuId, newValue);
        return true;
      } else {
        // No tokens available, increment back to restore state
        stringRedisTemplate.opsForValue().increment(tokenKey);
        logger.debug("No tokens available: skuId={}", skuId);
        return false;
      }

    } catch (Exception e) {
      logger.error("Failed to consume token: skuId={}", skuId, e);
      return false;
    }
  }

  /**
   * Generate a unique queue token for tracking.
   *
   * @return unique queue token
   */
  private String generateQueueToken() {
    return "QT-" + System.currentTimeMillis() + "-" + UUID.randomUUID().toString().substring(0, 8);
  }

  /**
   * Get Redis key for token bucket.
   *
   * @param skuId the SKU identifier
   * @return Redis key
   */
  private String getTokenBucketKey(String skuId) {
    return TOKEN_BUCKET_PREFIX + skuId;
  }

  /**
   * Get Redis key for token generation rate.
   *
   * @param skuId the SKU identifier
   * @return Redis key
   */
  private String getTokenRateKey(String skuId) {
    return TOKEN_RATE_PREFIX + skuId;
  }

  /**
   * Get Redis key for last refill timestamp.
   *
   * @param skuId the SKU identifier
   * @return Redis key
   */
  private String getTokenLastKey(String skuId) {
    return TOKEN_LAST_PREFIX + skuId;
  }

  /**
   * Get Redis key for bucket capacity.
   *
   * @param skuId the SKU identifier
   * @return Redis key
   */
  private String getTokenCapacityKey(String skuId) {
    return TOKEN_CAPACITY_PREFIX + skuId;
  }

  /**
   * Get Redis key for sold out flag.
   *
   * @param skuId the SKU identifier
   * @return Redis key
   */
  private String getSoldOutKey(String skuId) {
    return SOLD_OUT_PREFIX + skuId;
  }

  /**
   * Get Redis key for order status cache.
   *
   * @param orderId the order identifier
   * @return Redis key
   */
  private String getOrderStatusKey(String orderId) {
    return ORDER_STATUS_PREFIX + orderId;
  }

  /**
   * Update queue length metric based on current token bucket state.
   *
   * @param skuId the SKU identifier
   */
  private void updateQueueLengthMetric(String skuId) {
    try {
      String tokenKey = getTokenBucketKey(skuId);
      String tokenValue = stringRedisTemplate.opsForValue().get(tokenKey);

      if (tokenValue != null) {
        int availableTokens = Integer.parseInt(tokenValue);
        // If tokens are negative, that represents queue depth
        int queueLength = availableTokens < 0 ? Math.abs(availableTokens) : 0;
        metrics.updateQueueLength(queueLength);
      }
    } catch (Exception e) {
      logger.debug("Failed to update queue length metric: skuId={}", skuId, e);
    }
  }
}
