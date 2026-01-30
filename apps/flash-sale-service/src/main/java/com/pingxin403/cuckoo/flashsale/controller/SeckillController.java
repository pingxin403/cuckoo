package com.pingxin403.cuckoo.flashsale.controller;

import java.util.List;
import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.pingxin403.cuckoo.flashsale.kafka.OrderMessageProducer;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.service.ActivityService;
import com.pingxin403.cuckoo.flashsale.service.AntiFraudService;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.DeviceFingerprint;
import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.SeckillRequest;

import jakarta.servlet.http.HttpServletRequest;

/**
 * 秒杀控制器 REST controller for flash sale (seckill) operations.
 *
 * <p>Provides endpoints for:
 *
 * <ul>
 *   <li>Seckill entry - POST /api/seckill/{skuId}
 *   <li>Status query - GET /api/seckill/status/{orderId}
 *   <li>Activity management - CRUD operations
 * </ul>
 *
 * <p>Integrates all service components following the three-layer funnel model:
 *
 * <ol>
 *   <li>Anti-fraud layer - Risk assessment and rate limiting
 *   <li>Queue layer - Token bucket based traffic control
 *   <li>Inventory layer - Atomic stock deduction
 *   <li>Async processing - Kafka message queue for order persistence
 * </ol>
 *
 * <p>Validates all requirements from the design document.
 */
@RestController
@RequestMapping("/api/seckill")
public class SeckillController {

  private static final Logger logger = LoggerFactory.getLogger(SeckillController.class);

  private final InventoryService inventoryService;
  private final QueueService queueService;
  private final OrderService orderService;
  private final AntiFraudService antiFraudService;
  private final ActivityService activityService;
  private final OrderMessageProducer orderMessageProducer;
  private final com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics metrics;

  /**
   * Constructor with dependency injection.
   *
   * @param inventoryService inventory management service
   * @param queueService queue and rate limiting service
   * @param orderService order management service
   * @param antiFraudService anti-fraud and risk assessment service
   * @param activityService activity management service
   * @param orderMessageProducer Kafka message producer for orders
   * @param metrics metrics service for recording operations
   */
  public SeckillController(
      InventoryService inventoryService,
      QueueService queueService,
      OrderService orderService,
      AntiFraudService antiFraudService,
      ActivityService activityService,
      OrderMessageProducer orderMessageProducer,
      com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics metrics) {
    this.inventoryService = inventoryService;
    this.queueService = queueService;
    this.orderService = orderService;
    this.antiFraudService = antiFraudService;
    this.activityService = activityService;
    this.orderMessageProducer = orderMessageProducer;
    this.metrics = metrics;
  }

  /**
   * 秒杀入口接口 Seckill entry endpoint.
   *
   * <p>POST /api/seckill/{skuId}
   *
   * <p>Implements the complete three-layer funnel flow:
   *
   * <ol>
   *   <li>Anti-fraud assessment (L2/L3 risk control)
   *   <li>Queue token acquisition (rate limiting)
   *   <li>Atomic stock deduction (Redis Lua script)
   *   <li>Kafka message production (async order persistence)
   * </ol>
   *
   * <p>Response Codes:
   *
   * <ul>
   *   <li>200 - Success, order created
   *   <li>202 - Queuing, retry later
   *   <li>403 - Blocked by anti-fraud
   *   <li>410 - Sold out
   *   <li>420 - Activity not started
   *   <li>421 - Activity ended
   *   <li>422 - Purchase limit exceeded
   *   <li>423 - Captcha required
   *   <li>429 - Rate limited
   *   <li>503 - System busy
   * </ul>
   *
   * @param skuId the SKU identifier
   * @param userId the user identifier (from request header or param)
   * @param quantity the quantity to purchase (default 1)
   * @param deviceId the device identifier (optional)
   * @param captchaCode the captcha code (required if flagged)
   * @param request the HTTP servlet request for extracting metadata
   * @return ResponseEntity with SeckillResponse
   */
  @PostMapping("/{skuId}")
  public ResponseEntity<SeckillResponse> seckill(
      @PathVariable String skuId,
      @RequestParam String userId,
      @RequestParam(defaultValue = "1") int quantity,
      @RequestParam(required = false) String deviceId,
      @RequestParam(required = false) String captchaCode,
      @RequestParam(defaultValue = "WEB") String source,
      HttpServletRequest request) {

    logger.info(
        "Seckill request received: userId={}, skuId={}, quantity={}, source={}",
        userId,
        skuId,
        quantity,
        source);

    // Record request and measure duration
    return metrics
        .getSeckillRequestDuration()
        .record(
            () -> {
              try {
                // Step 1: Check purchase limit
                if (activityService.hasReachedPurchaseLimit(userId, skuId)) {
                  logger.warn("User {} has reached purchase limit for SKU {}", userId, skuId);
                  metrics.recordSeckillRequest(false);
                  return ResponseEntity.status(422)
                      .body(SeckillResponse.error(422, "超过限购数量", null));
                }

                // Step 2: Anti-fraud assessment (L2/L3)
                DeviceFingerprint fingerprint =
                    deviceId != null ? DeviceFingerprint.ofDeviceId(deviceId) : null;

                SeckillRequest seckillRequest =
                    SeckillRequest.builder()
                        .userId(userId)
                        .skuId(skuId)
                        .quantity(quantity)
                        .deviceFingerprint(fingerprint)
                        .ipAddress(getClientIp(request))
                        .userAgent(request.getHeader("User-Agent"))
                        .timestamp(System.currentTimeMillis())
                        .source(source)
                        .build();

                RiskAssessment riskAssessment = antiFraudService.assess(seckillRequest);

                // Handle risk assessment result
                if (riskAssessment.shouldBlock()) {
                  logger.warn(
                      "Request blocked by anti-fraud: userId={}, reason={}",
                      userId,
                      riskAssessment.reason());
                  metrics.recordSeckillRequest(false);
                  return ResponseEntity.status(HttpStatus.FORBIDDEN)
                      .body(SeckillResponse.error(403, "请求被拒绝: " + riskAssessment.reason(), null));
                }

                if (riskAssessment.requiresCaptcha()) {
                  // Verify captcha if provided
                  if (captchaCode == null || captchaCode.isBlank()) {
                    logger.info("Captcha required for userId={}", userId);
                    metrics.recordSeckillRequest(false);
                    return ResponseEntity.status(423)
                        .body(SeckillResponse.error(423, "需要验证码", null));
                  }

                  if (!antiFraudService.verifyCaptcha(userId, captchaCode)) {
                    logger.warn("Invalid captcha for userId={}", userId);
                    metrics.recordSeckillRequest(false);
                    return ResponseEntity.status(423)
                        .body(SeckillResponse.error(423, "验证码错误", null));
                  }
                }

                // Step 3: Queue token acquisition (rate limiting)
                QueueResult queueResult = queueService.tryAcquireToken(userId, skuId);

                if (queueResult.code() == 410) {
                  // Sold out
                  logger.info("SKU {} is sold out", skuId);
                  metrics.recordSeckillRequest(false);
                  return ResponseEntity.status(410)
                      .body(SeckillResponse.error(410, queueResult.message(), null));
                }

                if (queueResult.code() == 202) {
                  // Queuing - not counted as failure, just rate limited
                  logger.info(
                      "User {} is queuing for SKU {}, estimated wait: {}s",
                      userId,
                      skuId,
                      queueResult.estimatedWait());
                  return ResponseEntity.status(202)
                      .body(
                          SeckillResponse.queuing(
                              queueResult.message(),
                              queueResult.estimatedWait(),
                              queueResult.queueToken()));
                }

                // Step 4: Atomic stock deduction
                DeductResult deductResult = inventoryService.deductStock(skuId, userId, quantity);

                if (!deductResult.success()) {
                  if (deductResult.code()
                      == com.pingxin403.cuckoo.flashsale.service.dto.DeductResultCode
                          .OUT_OF_STOCK) {
                    // Notify queue service that SKU is sold out
                    queueService.notifySoldOut(skuId);
                    logger.info("SKU {} out of stock", skuId);
                    metrics.recordSeckillRequest(false);
                    return ResponseEntity.status(410)
                        .body(SeckillResponse.error(410, "商品已售罄", null));
                  } else {
                    logger.error(
                        "Stock deduction failed for SKU {}: {}", skuId, deductResult.code());
                    metrics.recordSeckillRequest(false);
                    return ResponseEntity.status(503)
                        .body(SeckillResponse.error(503, "系统繁忙，请稍后重试", null));
                  }
                }

                // Step 5: Send order message to Kafka (async processing)
                String orderId = deductResult.orderId();
                OrderMessage orderMessage =
                    new OrderMessage(
                        orderId,
                        userId,
                        skuId,
                        quantity,
                        System.currentTimeMillis(),
                        source,
                        generateTraceId());

                orderMessageProducer.send(orderMessage);

                // Step 6: Record user purchase for limit tracking
                activityService.recordUserPurchase(userId, skuId, quantity);

                logger.info(
                    "Seckill success: userId={}, skuId={}, orderId={}, remainingStock={}",
                    userId,
                    skuId,
                    orderId,
                    deductResult.remainingStock());

                // Record successful request
                metrics.recordSeckillRequest(true);

                return ResponseEntity.ok(
                    SeckillResponse.success("秒杀成功", orderId, deductResult.remainingStock()));

              } catch (Exception e) {
                logger.error("Seckill request failed: userId={}, skuId={}", userId, skuId, e);

                // Record failed request
                metrics.recordSeckillRequest(false);

                return ResponseEntity.status(500).body(SeckillResponse.error(500, "系统错误", null));
              }
            });
  }

  /**
   * 订单状态查询接口 Order status query endpoint.
   *
   * <p>GET /api/seckill/status/{orderId}
   *
   * <p>Allows users to check the current status of their order or queue position.
   *
   * @param orderId the order identifier or queue token
   * @return ResponseEntity with OrderStatusResult
   */
  @GetMapping("/status/{orderId}")
  public ResponseEntity<OrderStatusResult> queryStatus(@PathVariable String orderId) {
    logger.info("Status query request: orderId={}", orderId);

    try {
      OrderStatusResult statusResult = queueService.queryStatus(orderId);
      return ResponseEntity.ok(statusResult);
    } catch (Exception e) {
      logger.error("Status query failed: orderId={}", orderId, e);
      return ResponseEntity.status(500).body(null);
    }
  }

  // ==================== Activity Management Endpoints ====================

  /**
   * 创建秒杀活动 Create a new flash sale activity.
   *
   * <p>POST /api/seckill/activity
   *
   * @param request the activity creation request
   * @return ResponseEntity with created activity
   */
  @PostMapping("/activity")
  public ResponseEntity<SeckillActivity> createActivity(
      @RequestBody ActivityCreateRequest request) {
    logger.info("Create activity request: {}", request);

    try {
      SeckillActivity activity = activityService.createActivity(request);

      // Warmup stock to Redis
      inventoryService.warmupStock(activity.getSkuId(), activity.getTotalStock());

      logger.info("Activity created: activityId={}", activity.getActivityId());
      return ResponseEntity.status(HttpStatus.CREATED).body(activity);
    } catch (Exception e) {
      logger.error("Failed to create activity", e);
      return ResponseEntity.status(500).body(null);
    }
  }

  /**
   * 查询活动详情 Get activity details by ID.
   *
   * <p>GET /api/seckill/activity/{activityId}
   *
   * @param activityId the activity identifier
   * @return ResponseEntity with activity details
   */
  @GetMapping("/activity/{activityId}")
  public ResponseEntity<SeckillActivity> getActivity(@PathVariable String activityId) {
    logger.info("Get activity request: activityId={}", activityId);

    Optional<SeckillActivity> activity = activityService.getActivity(activityId);
    return activity.map(ResponseEntity::ok).orElseGet(() -> ResponseEntity.notFound().build());
  }

  /**
   * 查询所有活动 Get all activities.
   *
   * <p>GET /api/seckill/activity
   *
   * @return ResponseEntity with list of all activities
   */
  @GetMapping("/activity")
  public ResponseEntity<List<SeckillActivity>> getAllActivities() {
    logger.info("Get all activities request");

    try {
      List<SeckillActivity> activities = activityService.getAllActivities();
      return ResponseEntity.ok(activities);
    } catch (Exception e) {
      logger.error("Failed to get all activities", e);
      return ResponseEntity.status(500).body(null);
    }
  }

  /**
   * 更新活动 Update an existing activity.
   *
   * <p>PUT /api/seckill/activity/{activityId}
   *
   * @param activityId the activity identifier
   * @param request the update request
   * @return ResponseEntity with updated activity
   */
  @PutMapping("/activity/{activityId}")
  public ResponseEntity<SeckillActivity> updateActivity(
      @PathVariable String activityId, @RequestBody ActivityUpdateRequest request) {
    logger.info("Update activity request: activityId={}, request={}", activityId, request);

    try {
      SeckillActivity activity = activityService.updateActivity(activityId, request);
      logger.info("Activity updated: activityId={}", activityId);
      return ResponseEntity.ok(activity);
    } catch (IllegalArgumentException e) {
      logger.warn("Activity not found: activityId={}", activityId);
      return ResponseEntity.notFound().build();
    } catch (Exception e) {
      logger.error("Failed to update activity: activityId={}", activityId, e);
      return ResponseEntity.status(500).body(null);
    }
  }

  /**
   * 删除活动 Delete an activity.
   *
   * <p>DELETE /api/seckill/activity/{activityId}
   *
   * @param activityId the activity identifier
   * @return ResponseEntity with no content
   */
  @DeleteMapping("/activity/{activityId}")
  public ResponseEntity<Void> deleteActivity(@PathVariable String activityId) {
    logger.info("Delete activity request: activityId={}", activityId);

    try {
      boolean deleted = activityService.deleteActivity(activityId);
      if (deleted) {
        logger.info("Activity deleted: activityId={}", activityId);
        return ResponseEntity.noContent().build();
      } else {
        logger.warn("Activity not found: activityId={}", activityId);
        return ResponseEntity.notFound().build();
      }
    } catch (Exception e) {
      logger.error("Failed to delete activity: activityId={}", activityId, e);
      return ResponseEntity.status(500).build();
    }
  }

  /**
   * 手动开启活动 Manually start an activity.
   *
   * <p>POST /api/seckill/activity/{activityId}/start
   *
   * @param activityId the activity identifier
   * @return ResponseEntity with success status
   */
  @PostMapping("/activity/{activityId}/start")
  public ResponseEntity<ApiResponse> startActivity(@PathVariable String activityId) {
    logger.info("Start activity request: activityId={}", activityId);

    try {
      boolean started = activityService.startActivity(activityId);
      if (started) {
        logger.info("Activity started: activityId={}", activityId);
        return ResponseEntity.ok(new ApiResponse(true, "活动已开启"));
      } else {
        logger.warn("Failed to start activity: activityId={}", activityId);
        return ResponseEntity.badRequest().body(new ApiResponse(false, "活动开启失败"));
      }
    } catch (Exception e) {
      logger.error("Failed to start activity: activityId={}", activityId, e);
      return ResponseEntity.status(500).body(new ApiResponse(false, "系统错误"));
    }
  }

  /**
   * 手动结束活动 Manually end an activity.
   *
   * <p>POST /api/seckill/activity/{activityId}/end
   *
   * @param activityId the activity identifier
   * @return ResponseEntity with success status
   */
  @PostMapping("/activity/{activityId}/end")
  public ResponseEntity<ApiResponse> endActivity(@PathVariable String activityId) {
    logger.info("End activity request: activityId={}", activityId);

    try {
      boolean ended = activityService.endActivity(activityId);
      if (ended) {
        logger.info("Activity ended: activityId={}", activityId);
        return ResponseEntity.ok(new ApiResponse(true, "活动已结束"));
      } else {
        logger.warn("Failed to end activity: activityId={}", activityId);
        return ResponseEntity.badRequest().body(new ApiResponse(false, "活动结束失败"));
      }
    } catch (Exception e) {
      logger.error("Failed to end activity: activityId={}", activityId, e);
      return ResponseEntity.status(500).body(new ApiResponse(false, "系统错误"));
    }
  }

  // ==================== Helper Methods ====================

  /**
   * Extract client IP address from HTTP request.
   *
   * @param request the HTTP servlet request
   * @return the client IP address
   */
  private String getClientIp(HttpServletRequest request) {
    String ip = request.getHeader("X-Forwarded-For");
    if (ip == null || ip.isEmpty() || "unknown".equalsIgnoreCase(ip)) {
      ip = request.getHeader("X-Real-IP");
    }
    if (ip == null || ip.isEmpty() || "unknown".equalsIgnoreCase(ip)) {
      ip = request.getRemoteAddr();
    }
    // Handle multiple IPs in X-Forwarded-For
    if (ip != null && ip.contains(",")) {
      ip = ip.split(",")[0].trim();
    }
    return ip;
  }

  /**
   * Generate a unique trace ID for request tracking.
   *
   * @return a unique trace ID
   */
  private String generateTraceId() {
    return "trace-" + System.currentTimeMillis() + "-" + Thread.currentThread().getId();
  }

  // ==================== Response DTOs ====================

  /**
   * 秒杀响应 Seckill response DTO.
   *
   * @param code response code
   * @param message response message
   * @param orderId order ID (null if not successful)
   * @param remainingStock remaining stock (null if not applicable)
   * @param estimatedWait estimated wait time in seconds (null if not queuing)
   * @param queueToken queue token for status tracking (null if not queuing)
   */
  public record SeckillResponse(
      int code,
      String message,
      String orderId,
      Integer remainingStock,
      Integer estimatedWait,
      String queueToken) {

    public static SeckillResponse success(String message, String orderId, int remainingStock) {
      return new SeckillResponse(200, message, orderId, remainingStock, null, null);
    }

    public static SeckillResponse queuing(String message, int estimatedWait, String queueToken) {
      return new SeckillResponse(202, message, null, null, estimatedWait, queueToken);
    }

    public static SeckillResponse error(int code, String message, String orderId) {
      return new SeckillResponse(code, message, orderId, null, null, null);
    }
  }

  /**
   * 通用API响应 Generic API response DTO.
   *
   * @param success whether the operation was successful
   * @param message response message
   */
  public record ApiResponse(boolean success, String message) {}
}
