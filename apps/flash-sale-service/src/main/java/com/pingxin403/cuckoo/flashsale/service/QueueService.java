package com.pingxin403.cuckoo.flashsale.service;

import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;

/**
 * 排队服务接口 Queue service interface for flash sale traffic control.
 *
 * <p>Provides token bucket based rate limiting to control the flow of users entering the flash sale
 * process. This prevents overwhelming the backend systems with excessive concurrent requests.
 *
 * <p>Token Bucket Algorithm:
 *
 * <ul>
 *   <li>Tokens are generated at a fixed rate (e.g., 1000 tokens/second)
 *   <li>Each request consumes one token
 *   <li>When tokens are available, requests proceed immediately
 *   <li>When no tokens are available, requests receive a queuing response
 * </ul>
 *
 * <p>Redis Key Patterns:
 *
 * <ul>
 *   <li>token_bucket:{skuId} -> Integer (available tokens)
 *   <li>token_bucket_rate:{skuId} -> Integer (token generation rate per second)
 *   <li>token_bucket_last:{skuId} -> Long (last token refill timestamp in milliseconds)
 *   <li>order_status:{orderId} -> String (order status) -> TTL: 24 hours
 *   <li>sold_out:{skuId} -> "1" (sold out flag) -> TTL: activity end time
 * </ul>
 *
 * <p>Validates Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 *
 * @see QueueResult
 * @see OrderStatusResult
 */
public interface QueueService {

  /**
   * 尝试获取令牌进入秒杀 Try to acquire a token to enter the flash sale process.
   *
   * <p>Uses token bucket algorithm to control traffic flow:
   *
   * <ol>
   *   <li>Check if SKU is sold out
   *   <li>Calculate tokens to add based on elapsed time since last refill
   *   <li>Refill tokens up to bucket capacity
   *   <li>Try to consume one token
   *   <li>Return result based on token availability
   * </ol>
   *
   * <p>Response Codes:
   *
   * <ul>
   *   <li>200 - Token acquired, proceed to stock deduction
   *   <li>202 - No tokens available, user should queue and retry
   *   <li>410 - SKU sold out, activity ended
   * </ul>
   *
   * <p>Validates Requirements: 4.1, 4.4
   *
   * @param userId the user identifier
   * @param skuId the SKU identifier
   * @return QueueResult indicating whether token was acquired, queuing, or sold out
   */
  QueueResult tryAcquireToken(String userId, String skuId);

  /**
   * 查询排队/订单状态 Query the status of a queued request or order.
   *
   * <p>Allows users to check the current status of their order after receiving a queue token or
   * order ID. The status is cached in Redis for fast access.
   *
   * <p>Validates Requirement: 4.3
   *
   * @param orderId the order identifier or queue token
   * @return OrderStatusResult containing the current status
   */
  OrderStatusResult queryStatus(String orderId);

  /**
   * 获取预估等待时间 Get estimated wait time for a SKU.
   *
   * <p>Calculates the estimated wait time based on:
   *
   * <ul>
   *   <li>Current queue length (number of waiting requests)
   *   <li>Token generation rate (tokens per second)
   *   <li>Average processing time per request
   * </ul>
   *
   * <p>Formula: estimatedWait = queueLength / tokenRate
   *
   * <p>Validates Requirement: 4.5 - Estimated wait time accuracy within 50% of actual
   *
   * @param skuId the SKU identifier
   * @return estimated wait time in seconds (0 if tokens available)
   */
  int getEstimatedWaitTime(String skuId);

  /**
   * 通知售罄 Notify that a SKU is sold out.
   *
   * <p>Sets a sold out flag in Redis to immediately reject all queued and new requests for this
   * SKU. This prevents users from waiting unnecessarily when the product is no longer available.
   *
   * <p>Actions:
   *
   * <ul>
   *   <li>Set sold_out:{skuId} flag in Redis
   *   <li>Clear token bucket for this SKU
   *   <li>All subsequent tryAcquireToken calls will return code 410
   * </ul>
   *
   * <p>Validates Requirement: 4.6
   *
   * @param skuId the SKU identifier that is sold out
   */
  void notifySoldOut(String skuId);
}
