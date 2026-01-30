package com.pingxin403.cuckoo.flashsale.service.impl;

import java.time.Duration;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.RedisConnectionFailureException;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.stereotype.Service;

import com.pingxin403.cuckoo.flashsale.service.AntiFraudService;
import com.pingxin403.cuckoo.flashsale.service.dto.DeviceFingerprint;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskLevel;
import com.pingxin403.cuckoo.flashsale.service.dto.SeckillRequest;

/**
 * 反作弊服务实现类 Implementation of AntiFraudService for flash sale risk assessment.
 *
 * <p>Implements multi-layer anti-fraud detection:
 *
 * <ul>
 *   <li>Device fingerprint risk scoring
 *   <li>Request frequency analysis
 *   <li>Behavior pattern detection
 * </ul>
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>device_risk:{deviceId} -> Hash {score, lastSeen, requestCount} -> TTL: 24小时
 *   <li>captcha:{userId} -> String (captcha code) -> TTL: 5分钟
 *   <li>request_count:{deviceId} -> Integer (request count in time window)
 * </ul>
 *
 * <p>Validates Requirements: 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 7.4
 */
@Service
public class AntiFraudServiceImpl implements AntiFraudService {

  private static final Logger logger = LoggerFactory.getLogger(AntiFraudServiceImpl.class);

  /** Redis key prefix for device risk data */
  private static final String DEVICE_RISK_KEY_PREFIX = "device_risk:";

  /** Redis key prefix for captcha codes */
  private static final String CAPTCHA_KEY_PREFIX = "captcha:";

  /** Redis key prefix for request count tracking */
  private static final String REQUEST_COUNT_KEY_PREFIX = "request_count:";

  /** Redis hash field for risk score */
  private static final String FIELD_SCORE = "score";

  /** Redis hash field for last seen timestamp */
  private static final String FIELD_LAST_SEEN = "lastSeen";

  /** Redis hash field for request count */
  private static final String FIELD_REQUEST_COUNT = "requestCount";

  /** TTL for device risk data: 24 hours */
  private static final Duration DEVICE_RISK_TTL = Duration.ofHours(24);

  /** TTL for captcha codes: 5 minutes */
  private static final Duration CAPTCHA_TTL = Duration.ofMinutes(5);

  /** TTL for request count window: 1 minute */
  private static final Duration REQUEST_COUNT_TTL = Duration.ofMinutes(1);

  /** Default threshold for HIGH risk (requests per minute) */
  private static final int DEFAULT_HIGH_RISK_THRESHOLD = 50;

  /** Default threshold for MEDIUM risk (requests per minute) */
  private static final int DEFAULT_MEDIUM_RISK_THRESHOLD = 20;

  /** Default threshold for device risk score to trigger HIGH risk */
  private static final int DEFAULT_HIGH_SCORE_THRESHOLD = 80;

  /** Default threshold for device risk score to trigger MEDIUM risk */
  private static final int DEFAULT_MEDIUM_SCORE_THRESHOLD = 50;

  private final StringRedisTemplate stringRedisTemplate;
  private final DefaultRedisScript<Long> tokenBucketScript;

  /** Dynamic rate limit thresholds (can be updated at runtime) */
  private final Map<String, Integer> rateLimitThresholds = new ConcurrentHashMap<>();

  /** Redis key prefix for token bucket rate limiting */
  private static final String TOKEN_BUCKET_KEY_PREFIX = "rate_limit:token_bucket:";

  /** Redis key prefix for token bucket last refill time */
  private static final String TOKEN_BUCKET_LAST_KEY_PREFIX = "rate_limit:token_bucket_last:";

  /** Default token bucket capacity for L2 rate limiting */
  private static final int DEFAULT_TOKEN_BUCKET_CAPACITY = 100;

  /** Default token refill rate (tokens per second) for L2 rate limiting */
  private static final int DEFAULT_TOKEN_REFILL_RATE = 10;

  /**
   * Constructor with dependency injection.
   *
   * @param stringRedisTemplate Redis template for string operations
   * @param tokenBucketScript Redis Lua script for token bucket algorithm
   */
  public AntiFraudServiceImpl(
      StringRedisTemplate stringRedisTemplate, DefaultRedisScript<Long> tokenBucketScript) {
    this.stringRedisTemplate = stringRedisTemplate;
    this.tokenBucketScript = tokenBucketScript;
    initializeDefaultThresholds();
  }

  /** Initialize default rate limit thresholds. */
  private void initializeDefaultThresholds() {
    rateLimitThresholds.put("high_risk_request_count", DEFAULT_HIGH_RISK_THRESHOLD);
    rateLimitThresholds.put("medium_risk_request_count", DEFAULT_MEDIUM_RISK_THRESHOLD);
    rateLimitThresholds.put("high_risk_score", DEFAULT_HIGH_SCORE_THRESHOLD);
    rateLimitThresholds.put("medium_risk_score", DEFAULT_MEDIUM_SCORE_THRESHOLD);
    rateLimitThresholds.put("token_bucket_capacity", DEFAULT_TOKEN_BUCKET_CAPACITY);
    rateLimitThresholds.put("token_refill_rate", DEFAULT_TOKEN_REFILL_RATE);
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirements: 3.4, 3.5, 3.6
   *
   * <p>Risk assessment logic:
   *
   * <ol>
   *   <li>Check if request has valid device fingerprint
   *   <li>Try to acquire L2 rate limiting token (token bucket)
   *   <li>Get device risk score from Redis
   *   <li>Increment and check request count
   *   <li>Determine risk level based on score and frequency
   *   <li>Map risk level to action (LOW->PASS, MEDIUM->CAPTCHA, HIGH->BLOCK)
   * </ol>
   */
  @Override
  public RiskAssessment assess(SeckillRequest request) {
    if (request == null) {
      logger.warn("Risk assessment failed: request is null");
      return RiskAssessment.block("无效请求");
    }

    if (request.userId() == null || request.userId().isBlank()) {
      logger.warn("Risk assessment failed: userId is null or blank");
      return RiskAssessment.block("用户ID无效");
    }

    try {
      // Get device ID from fingerprint
      String deviceId = request.getDeviceId();

      // If no device fingerprint, treat as medium risk (require captcha)
      if (deviceId == null || deviceId.isBlank()) {
        logger.info("No device fingerprint for user {}, requiring captcha", request.userId());
        return RiskAssessment.captcha("缺少设备指纹信息");
      }

      // L2 Rate Limiting: Try to acquire token from token bucket (Requirement 3.2)
      String rateLimitKey = "device:" + deviceId;
      boolean tokenAcquired = tryAcquireL2Token(rateLimitKey);

      if (!tokenAcquired) {
        // Token bucket rate limit exceeded - trigger L2 rate limiting
        logger.info(
            "L2 rate limit exceeded for user {}, deviceId={}, requiring captcha",
            request.userId(),
            deviceId);
        return RiskAssessment.captcha("请求频率超过阈值，请完成验证");
      }

      // Get device risk score
      int deviceScore = getDeviceRiskScore(deviceId);

      // Increment and get request count
      int requestCount = incrementRequestCount(deviceId);

      // Log the assessment for monitoring (Requirement 3.7)
      logger.debug(
          "Risk assessment: userId={}, deviceId={}, score={}, requestCount={}",
          request.userId(),
          deviceId,
          deviceScore,
          requestCount);

      // Determine risk level based on score and frequency
      RiskLevel riskLevel = calculateRiskLevel(deviceScore, requestCount);

      // Generate reason based on risk factors
      String reason = generateReason(riskLevel, deviceScore, requestCount);

      // Create assessment with mapped action
      RiskAssessment assessment = RiskAssessment.fromLevel(riskLevel, reason);

      // Log blocked/captcha requests for analysis (Requirement 3.7)
      if (assessment.shouldBlock() || assessment.requiresCaptcha()) {
        logger.info(
            "Risk assessment result: userId={}, deviceId={}, level={}, action={}, reason={}",
            request.userId(),
            deviceId,
            riskLevel,
            assessment.action(),
            reason);
      }

      return assessment;

    } catch (RedisConnectionFailureException e) {
      logger.error(
          "Redis connection failure during risk assessment: userId={}", request.userId(), e);
      // On Redis failure, allow request but require captcha as precaution
      return RiskAssessment.captcha("系统繁忙，请完成验证");
    } catch (Exception e) {
      logger.error("Unexpected error during risk assessment: userId={}", request.userId(), e);
      return RiskAssessment.captcha("系统错误，请完成验证");
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 3.2
   */
  @Override
  public boolean verifyCaptcha(String userId, String captchaCode) {
    if (userId == null || userId.isBlank()) {
      logger.warn("Captcha verification failed: userId is null or blank");
      return false;
    }

    if (captchaCode == null || captchaCode.isBlank()) {
      logger.warn("Captcha verification failed: captchaCode is null or blank, userId={}", userId);
      return false;
    }

    String captchaKey = getCaptchaKey(userId);

    try {
      String storedCode = stringRedisTemplate.opsForValue().get(captchaKey);

      if (storedCode == null) {
        logger.info("Captcha verification failed: no captcha found for userId={}", userId);
        return false;
      }

      // Case-insensitive comparison
      boolean isValid = storedCode.equalsIgnoreCase(captchaCode.trim());

      if (isValid) {
        // Delete captcha after successful verification (one-time use)
        stringRedisTemplate.delete(captchaKey);
        logger.info("Captcha verification successful: userId={}", userId);
      } else {
        logger.info("Captcha verification failed: invalid code for userId={}", userId);
      }

      return isValid;

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during captcha verification: userId={}", userId, e);
      return false;
    } catch (Exception e) {
      logger.error("Unexpected error during captcha verification: userId={}", userId, e);
      return false;
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 3.3
   */
  @Override
  public void recordFingerprint(String userId, DeviceFingerprint fingerprint) {
    if (userId == null || userId.isBlank()) {
      logger.warn("Record fingerprint failed: userId is null or blank");
      return;
    }

    if (fingerprint == null || !fingerprint.hasValidDeviceId()) {
      logger.warn("Record fingerprint failed: invalid fingerprint for userId={}", userId);
      return;
    }

    String deviceId = fingerprint.deviceId();
    String deviceRiskKey = getDeviceRiskKey(deviceId);

    try {
      // Get current data or initialize
      Map<Object, Object> currentData = stringRedisTemplate.opsForHash().entries(deviceRiskKey);

      int currentScore = 0;
      int currentRequestCount = 0;

      if (!currentData.isEmpty()) {
        currentScore = parseIntOrDefault(currentData.get(FIELD_SCORE), 0);
        currentRequestCount = parseIntOrDefault(currentData.get(FIELD_REQUEST_COUNT), 0);
      }

      // Calculate new risk score based on fingerprint characteristics
      int newScore = calculateFingerprintScore(fingerprint, currentScore);

      // Update device risk data
      stringRedisTemplate.opsForHash().put(deviceRiskKey, FIELD_SCORE, String.valueOf(newScore));
      stringRedisTemplate
          .opsForHash()
          .put(deviceRiskKey, FIELD_LAST_SEEN, String.valueOf(System.currentTimeMillis()));
      stringRedisTemplate
          .opsForHash()
          .put(deviceRiskKey, FIELD_REQUEST_COUNT, String.valueOf(currentRequestCount + 1));

      // Set TTL (24 hours)
      stringRedisTemplate.expire(deviceRiskKey, DEVICE_RISK_TTL);

      logger.debug(
          "Device fingerprint recorded: userId={}, deviceId={}, score={}",
          userId,
          deviceId,
          newScore);

    } catch (RedisConnectionFailureException e) {
      logger.error(
          "Redis connection failure during fingerprint recording: userId={}, deviceId={}",
          userId,
          deviceId,
          e);
    } catch (Exception e) {
      logger.error(
          "Unexpected error during fingerprint recording: userId={}, deviceId={}",
          userId,
          deviceId,
          e);
    }
  }

  /**
   * {@inheritDoc}
   *
   * <p>Validates Requirement: 7.4
   *
   * <p>Supported threshold keys:
   *
   * <ul>
   *   <li>high_risk_request_count - Request count threshold for HIGH risk
   *   <li>medium_risk_request_count - Request count threshold for MEDIUM risk
   *   <li>high_risk_score - Device score threshold for HIGH risk
   *   <li>medium_risk_score - Device score threshold for MEDIUM risk
   *   <li>token_bucket_capacity - Maximum tokens in the bucket for L2 rate limiting
   *   <li>token_refill_rate - Token refill rate (tokens per second) for L2 rate limiting
   * </ul>
   */
  @Override
  public void updateRateLimitThreshold(String key, int newThreshold) {
    if (key == null || key.isBlank()) {
      logger.warn("Update rate limit threshold failed: key is null or blank");
      return;
    }

    if (newThreshold < 0) {
      logger.warn(
          "Update rate limit threshold failed: threshold cannot be negative, key={}, threshold={}",
          key,
          newThreshold);
      return;
    }

    int oldThreshold = rateLimitThresholds.getOrDefault(key, -1);
    rateLimitThresholds.put(key, newThreshold);

    logger.info(
        "Rate limit threshold updated: key={}, oldThreshold={}, newThreshold={}",
        key,
        oldThreshold,
        newThreshold);
  }

  /**
   * Try to acquire a token from the L2 rate limiting token bucket.
   *
   * <p>Uses Redis token bucket algorithm to control request rate at L2 level. This provides smooth
   * rate limiting with burst capacity.
   *
   * <p>Validates Requirements: 3.2, 7.4
   *
   * @param key the rate limit key (e.g., "device:{deviceId}" or "user:{userId}")
   * @return true if token acquired successfully, false if rate limited
   */
  private boolean tryAcquireL2Token(String key) {
    if (key == null || key.isBlank()) {
      logger.warn("L2 token acquisition failed: key is null or blank");
      return false;
    }

    String bucketKey = TOKEN_BUCKET_KEY_PREFIX + key;
    String lastKey = TOKEN_BUCKET_LAST_KEY_PREFIX + key;

    try {
      // Get current thresholds (can be dynamically updated)
      int capacity =
          rateLimitThresholds.getOrDefault("token_bucket_capacity", DEFAULT_TOKEN_BUCKET_CAPACITY);
      int refillRate =
          rateLimitThresholds.getOrDefault("token_refill_rate", DEFAULT_TOKEN_REFILL_RATE);
      long currentTime = System.currentTimeMillis();

      // Execute token bucket Lua script
      Long result =
          stringRedisTemplate.execute(
              tokenBucketScript,
              java.util.Arrays.asList(bucketKey, lastKey),
              String.valueOf(capacity),
              String.valueOf(refillRate),
              String.valueOf(currentTime));

      boolean acquired = result != null && result == 1L;

      if (!acquired) {
        logger.debug("L2 rate limit triggered: key={}", key);
      }

      return acquired;

    } catch (RedisConnectionFailureException e) {
      logger.error("Redis connection failure during L2 token acquisition: key={}", key, e);
      // On Redis failure, allow request (fail open to avoid blocking all traffic)
      return true;
    } catch (Exception e) {
      logger.error("Unexpected error during L2 token acquisition: key={}", key, e);
      // On unexpected error, allow request but log for investigation
      return true;
    }
  }

  /**
   * Get device risk score from Redis.
   *
   * @param deviceId the device identifier
   * @return the risk score (0-100), or 0 if not found
   */
  private int getDeviceRiskScore(String deviceId) {
    String deviceRiskKey = getDeviceRiskKey(deviceId);

    try {
      Object scoreValue = stringRedisTemplate.opsForHash().get(deviceRiskKey, FIELD_SCORE);
      return parseIntOrDefault(scoreValue, 0);
    } catch (Exception e) {
      logger.warn("Failed to get device risk score: deviceId={}", deviceId, e);
      return 0;
    }
  }

  /**
   * Increment and get request count for a device.
   *
   * @param deviceId the device identifier
   * @return the current request count in the time window
   */
  private int incrementRequestCount(String deviceId) {
    String requestCountKey = getRequestCountKey(deviceId);

    try {
      Long count = stringRedisTemplate.opsForValue().increment(requestCountKey);

      // Set TTL on first increment
      if (count != null && count == 1) {
        stringRedisTemplate.expire(requestCountKey, REQUEST_COUNT_TTL);
      }

      return count != null ? count.intValue() : 1;
    } catch (Exception e) {
      logger.warn("Failed to increment request count: deviceId={}", deviceId, e);
      return 1;
    }
  }

  /**
   * Calculate risk level based on device score and request frequency.
   *
   * <p>Risk level determination:
   *
   * <ul>
   *   <li>HIGH: score >= high_score_threshold OR requestCount >= high_request_threshold
   *   <li>MEDIUM: score >= medium_score_threshold OR requestCount >= medium_request_threshold
   *   <li>LOW: otherwise
   * </ul>
   *
   * @param deviceScore the device risk score (0-100)
   * @param requestCount the request count in the time window
   * @return the calculated risk level
   */
  private RiskLevel calculateRiskLevel(int deviceScore, int requestCount) {
    int highScoreThreshold =
        rateLimitThresholds.getOrDefault("high_risk_score", DEFAULT_HIGH_SCORE_THRESHOLD);
    int mediumScoreThreshold =
        rateLimitThresholds.getOrDefault("medium_risk_score", DEFAULT_MEDIUM_SCORE_THRESHOLD);
    int highRequestThreshold =
        rateLimitThresholds.getOrDefault("high_risk_request_count", DEFAULT_HIGH_RISK_THRESHOLD);
    int mediumRequestThreshold =
        rateLimitThresholds.getOrDefault(
            "medium_risk_request_count", DEFAULT_MEDIUM_RISK_THRESHOLD);

    // Check for HIGH risk
    if (deviceScore >= highScoreThreshold || requestCount >= highRequestThreshold) {
      return RiskLevel.HIGH;
    }

    // Check for MEDIUM risk
    if (deviceScore >= mediumScoreThreshold || requestCount >= mediumRequestThreshold) {
      return RiskLevel.MEDIUM;
    }

    // Default to LOW risk
    return RiskLevel.LOW;
  }

  /**
   * Generate reason string based on risk factors.
   *
   * @param level the risk level
   * @param deviceScore the device risk score
   * @param requestCount the request count
   * @return the reason string
   */
  private String generateReason(RiskLevel level, int deviceScore, int requestCount) {
    return switch (level) {
      case HIGH -> {
        if (requestCount
            >= rateLimitThresholds.getOrDefault(
                "high_risk_request_count", DEFAULT_HIGH_RISK_THRESHOLD)) {
          yield "请求频率过高";
        } else {
          yield "设备风险评分过高";
        }
      }
      case MEDIUM -> {
        if (requestCount
            >= rateLimitThresholds.getOrDefault(
                "medium_risk_request_count", DEFAULT_MEDIUM_RISK_THRESHOLD)) {
          yield "请求频率较高，需要验证";
        } else {
          yield "设备风险评分较高，需要验证";
        }
      }
      case LOW -> "正常用户";
    };
  }

  /**
   * Calculate fingerprint risk score based on characteristics.
   *
   * <p>Scoring factors:
   *
   * <ul>
   *   <li>Missing fingerprint fields increase score
   *   <li>Suspicious patterns increase score
   *   <li>Score is capped at 100
   * </ul>
   *
   * @param fingerprint the device fingerprint
   * @param currentScore the current score (for incremental updates)
   * @return the calculated score (0-100)
   */
  private int calculateFingerprintScore(DeviceFingerprint fingerprint, int currentScore) {
    int score = currentScore;

    // Missing platform info is suspicious
    if (fingerprint.platform() == null || fingerprint.platform().isBlank()) {
      score += 10;
    }

    // Missing browser info is suspicious
    if (fingerprint.browserName() == null || fingerprint.browserName().isBlank()) {
      score += 10;
    }

    // Missing canvas fingerprint might indicate automation
    if (fingerprint.canvasFingerprint() == null || fingerprint.canvasFingerprint().isBlank()) {
      score += 15;
    }

    // Missing WebGL fingerprint might indicate automation
    if (fingerprint.webglFingerprint() == null || fingerprint.webglFingerprint().isBlank()) {
      score += 15;
    }

    // Cap score at 100
    return Math.min(score, 100);
  }

  /**
   * Generate Redis key for device risk data.
   *
   * @param deviceId the device identifier
   * @return the Redis key
   */
  private String getDeviceRiskKey(String deviceId) {
    return DEVICE_RISK_KEY_PREFIX + deviceId;
  }

  /**
   * Generate Redis key for captcha code.
   *
   * @param userId the user identifier
   * @return the Redis key
   */
  private String getCaptchaKey(String userId) {
    return CAPTCHA_KEY_PREFIX + userId;
  }

  /**
   * Generate Redis key for request count.
   *
   * @param deviceId the device identifier
   * @return the Redis key
   */
  private String getRequestCountKey(String deviceId) {
    return REQUEST_COUNT_KEY_PREFIX + deviceId;
  }

  /**
   * Parse integer from object with default value.
   *
   * @param value the value to parse
   * @param defaultValue the default value if parsing fails
   * @return the parsed integer or default value
   */
  private int parseIntOrDefault(Object value, int defaultValue) {
    if (value == null) {
      return defaultValue;
    }
    try {
      return Integer.parseInt(value.toString());
    } catch (NumberFormatException e) {
      return defaultValue;
    }
  }
}
