package com.pingxin403.cuckoo.flashsale.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

/**
 * Rate limiting configuration for Flash Sale Service.
 *
 * <p>Configures multi-layer rate limiting:
 *
 * <ul>
 *   <li>L1: Gateway-level IP rate limiting (configured in Higress)
 *   <li>L2: Application-level user rate limiting
 *   <li>L3: Token bucket for queue management
 * </ul>
 */
@Configuration
public class RateLimitConfig {

  @Value("${flash-sale.token-bucket.default-rate:1000}")
  private int defaultTokenRate;

  @Value("${flash-sale.token-bucket.default-capacity:5000}")
  private int defaultTokenCapacity;

  @Value("${flash-sale.token-bucket.key-prefix:token_bucket:}")
  private String tokenBucketKeyPrefix;

  /**
   * Gets the default token generation rate (tokens per second).
   *
   * @return tokens per second
   */
  public int getDefaultTokenRate() {
    return defaultTokenRate;
  }

  /**
   * Gets the default token bucket capacity.
   *
   * @return maximum tokens in bucket
   */
  public int getDefaultTokenCapacity() {
    return defaultTokenCapacity;
  }

  /**
   * Gets the Redis key prefix for token buckets.
   *
   * @return key prefix
   */
  public String getTokenBucketKeyPrefix() {
    return tokenBucketKeyPrefix;
  }
}
