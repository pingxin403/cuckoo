package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 排队结果 Queue result for token acquisition attempts.
 *
 * <p>Represents the outcome of trying to acquire a token to enter the flash sale process.
 *
 * <p>Response Codes:
 *
 * <ul>
 *   <li>200 - Token acquired, proceed to stock deduction
 *   <li>202 - Queuing, user should wait and retry
 *   <li>410 - Sold out, activity ended
 * </ul>
 *
 * <p>Validates Requirements: 4.1, 4.2, 4.4, 4.6
 *
 * @param code the response code (200=acquired, 202=queuing, 410=sold out)
 * @param message the human-readable message
 * @param estimatedWait the estimated wait time in seconds (0 if token acquired)
 * @param queueToken the queue token for status tracking (null if sold out)
 */
public record QueueResult(int code, String message, int estimatedWait, String queueToken) {

  /**
   * Create a successful token acquisition result.
   *
   * @param queueToken the queue token for tracking
   * @return QueueResult with code 200
   */
  public static QueueResult acquired(String queueToken) {
    return new QueueResult(200, "获得令牌", 0, queueToken);
  }

  /**
   * Create a queuing result with estimated wait time.
   *
   * @param estimatedWait the estimated wait time in seconds
   * @param queueToken the queue token for tracking
   * @return QueueResult with code 202
   */
  public static QueueResult queuing(int estimatedWait, String queueToken) {
    return new QueueResult(202, "排队中", estimatedWait, queueToken);
  }

  /**
   * Create a sold out result.
   *
   * @return QueueResult with code 410
   */
  public static QueueResult soldOut() {
    return new QueueResult(410, "商品已售罄", 0, null);
  }
}
