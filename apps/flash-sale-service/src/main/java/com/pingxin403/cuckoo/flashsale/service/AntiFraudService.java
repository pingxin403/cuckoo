package com.pingxin403.cuckoo.flashsale.service;

import com.pingxin403.cuckoo.flashsale.service.dto.DeviceFingerprint;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.SeckillRequest;

/**
 * 反作弊服务接口 Anti-fraud service interface for flash sale risk assessment.
 *
 * <p>Provides multi-layer anti-fraud capabilities:
 *
 * <ul>
 *   <li>L2 Rate Limiting - Request frequency based throttling
 *   <li>L3 Risk Control - Device fingerprint and behavior model based detection
 * </ul>
 *
 * <p>Risk Assessment Flow:
 *
 * <ol>
 *   <li>Check device fingerprint risk score
 *   <li>Analyze request frequency patterns
 *   <li>Apply behavior model scoring
 *   <li>Return risk level and recommended action
 * </ol>
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>device_risk:{deviceId} -> Hash {score, lastSeen, requestCount} -> TTL: 24小时
 *   <li>captcha:{userId} -> String (captcha code) -> TTL: 5分钟
 *   <li>rate_limit:{key} -> Integer (threshold) -> No TTL
 * </ul>
 *
 * <p>Validates Requirements: 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 7.4
 *
 * @see RiskAssessment
 * @see SeckillRequest
 * @see DeviceFingerprint
 */
public interface AntiFraudService {

  /**
   * 风险评估 Assess the risk level of a seckill request.
   *
   * <p>Evaluates the request based on:
   *
   * <ul>
   *   <li>Device fingerprint risk score
   *   <li>Request frequency (requests per time window)
   *   <li>Behavior patterns (timing, patterns)
   * </ul>
   *
   * <p>Returns a RiskAssessment with:
   *
   * <ul>
   *   <li>LOW level -> PASS action (normal user, proceed without verification)
   *   <li>MEDIUM level -> CAPTCHA action (suspicious user, require captcha)
   *   <li>HIGH level -> BLOCK action (high-risk user, reject request)
   * </ul>
   *
   * <p>Validates Requirements: 3.4, 3.5, 3.6
   *
   * @param request the seckill request to assess
   * @return RiskAssessment containing risk level, action, and reason
   */
  RiskAssessment assess(SeckillRequest request);

  /**
   * 验证码校验 Verify captcha code for a user.
   *
   * <p>Validates the captcha code submitted by a user who was flagged as suspicious (MEDIUM risk).
   * The captcha code is stored in Redis with a 5-minute TTL.
   *
   * <p>Validates Requirement: 3.2
   *
   * @param userId the user identifier
   * @param captchaCode the captcha code to verify
   * @return true if the captcha code is valid, false otherwise
   */
  boolean verifyCaptcha(String userId, String captchaCode);

  /**
   * 记录设备指纹 Record device fingerprint for a user.
   *
   * <p>Stores the device fingerprint information in Redis for future risk assessment. Updates the
   * device risk profile including:
   *
   * <ul>
   *   <li>Risk score based on fingerprint characteristics
   *   <li>Last seen timestamp
   *   <li>Request count
   * </ul>
   *
   * <p>Redis Key: device_risk:{deviceId} -> TTL: 24小时
   *
   * <p>Validates Requirement: 3.3
   *
   * @param userId the user identifier
   * @param fingerprint the device fingerprint to record
   */
  void recordFingerprint(String userId, DeviceFingerprint fingerprint);

  /**
   * 更新限流阈值 Update rate limit threshold dynamically.
   *
   * <p>Allows dynamic adjustment of rate limiting thresholds without service restart. The new
   * threshold takes effect immediately for subsequent risk assessments.
   *
   * <p>Validates Requirement: 7.4
   *
   * @param key the rate limit key (e.g., "device", "user", "ip")
   * @param newThreshold the new threshold value
   */
  void updateRateLimitThreshold(String key, int newThreshold);
}
