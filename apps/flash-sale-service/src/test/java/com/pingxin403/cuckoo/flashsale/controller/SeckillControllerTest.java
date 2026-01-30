package com.pingxin403.cuckoo.flashsale.controller;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.time.LocalDateTime;
import java.util.Arrays;
import java.util.List;
import java.util.Optional;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.mock.web.MockHttpServletRequest;

import com.pingxin403.cuckoo.flashsale.controller.SeckillController.ApiResponse;
import com.pingxin403.cuckoo.flashsale.controller.SeckillController.SeckillResponse;
import com.pingxin403.cuckoo.flashsale.kafka.OrderMessageProducer;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.service.ActivityService;
import com.pingxin403.cuckoo.flashsale.service.AntiFraudService;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

/**
 * Unit tests for SeckillController.
 *
 * <p>Tests all REST endpoints including:
 *
 * <ul>
 *   <li>Seckill entry endpoint with various scenarios
 *   <li>Order status query endpoint
 *   <li>Activity management CRUD operations
 * </ul>
 */
@ExtendWith(MockitoExtension.class)
class SeckillControllerTest {

  @Mock private InventoryService inventoryService;
  @Mock private QueueService queueService;
  @Mock private OrderService orderService;
  @Mock private AntiFraudService antiFraudService;
  @Mock private ActivityService activityService;
  @Mock private OrderMessageProducer orderMessageProducer;
  @Mock private com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics metrics;

  private SeckillController controller;
  private MockHttpServletRequest httpRequest;

  @BeforeEach
  void setUp() {
    controller =
        new SeckillController(
            inventoryService,
            queueService,
            orderService,
            antiFraudService,
            activityService,
            orderMessageProducer,
            metrics);

    httpRequest = new MockHttpServletRequest();
    httpRequest.setRemoteAddr("192.168.1.1");
    httpRequest.addHeader("User-Agent", "Mozilla/5.0");

    // Mock timer for metrics (lenient to avoid UnnecessaryStubbingException)
    io.micrometer.core.instrument.Timer timer = mock(io.micrometer.core.instrument.Timer.class);
    lenient().when(metrics.getSeckillRequestDuration()).thenReturn(timer);
    lenient()
        .when(timer.record(any(java.util.function.Supplier.class)))
        .thenAnswer(
            invocation -> {
              java.util.function.Supplier<?> supplier = invocation.getArgument(0);
              return supplier.get();
            });
  }

  // ==================== Seckill Entry Tests ====================

  @Test
  @DisplayName("Seckill success - normal flow")
  void testSeckillSuccess() {
    // Given
    String userId = "user123";
    String skuId = "sku456";
    int quantity = 1;

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.pass("Normal user"));
    when(queueService.tryAcquireToken(userId, skuId)).thenReturn(QueueResult.acquired("token123"));
    when(inventoryService.deductStock(skuId, userId, quantity))
        .thenReturn(DeductResult.success(99, "order123"));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, quantity, null, null, "WEB", httpRequest);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(200, response.getBody().code());
    assertEquals("order123", response.getBody().orderId());
    assertEquals(99, response.getBody().remainingStock());

    verify(orderMessageProducer, times(1)).send(any());
    verify(activityService, times(1)).recordUserPurchase(userId, skuId, quantity);
  }

  @Test
  @DisplayName("Seckill blocked - purchase limit exceeded")
  void testSeckillPurchaseLimitExceeded() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(true);

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(422, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(422, response.getBody().code());
    assertTrue(response.getBody().message().contains("限购"));

    verify(antiFraudService, never()).assess(any());
    verify(inventoryService, never()).deductStock(any(), any(), anyInt());
  }

  @Test
  @DisplayName("Seckill blocked - high risk user")
  void testSeckillBlockedByAntiFraud() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.block("High risk detected"));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(HttpStatus.FORBIDDEN, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(403, response.getBody().code());

    verify(queueService, never()).tryAcquireToken(any(), any());
    verify(inventoryService, never()).deductStock(any(), any(), anyInt());
  }

  @Test
  @DisplayName("Seckill requires captcha - no captcha provided")
  void testSeckillRequiresCaptcha() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.captcha("Suspicious activity"));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(423, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(423, response.getBody().code());
    assertTrue(response.getBody().message().contains("验证码"));

    verify(queueService, never()).tryAcquireToken(any(), any());
  }

  @Test
  @DisplayName("Seckill with valid captcha")
  void testSeckillWithValidCaptcha() {
    // Given
    String userId = "user123";
    String skuId = "sku456";
    String captchaCode = "1234";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.captcha("Suspicious activity"));
    when(antiFraudService.verifyCaptcha(userId, captchaCode)).thenReturn(true);
    when(queueService.tryAcquireToken(userId, skuId)).thenReturn(QueueResult.acquired("token123"));
    when(inventoryService.deductStock(skuId, userId, 1))
        .thenReturn(DeductResult.success(99, "order123"));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, captchaCode, "WEB", httpRequest);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(200, response.getBody().code());

    verify(antiFraudService, times(1)).verifyCaptcha(userId, captchaCode);
  }

  @Test
  @DisplayName("Seckill queuing - no tokens available")
  void testSeckillQueuing() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.pass("Normal user"));
    when(queueService.tryAcquireToken(userId, skuId))
        .thenReturn(QueueResult.queuing(5, "queue123"));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(202, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(202, response.getBody().code());
    assertEquals(5, response.getBody().estimatedWait());
    assertEquals("queue123", response.getBody().queueToken());

    verify(inventoryService, never()).deductStock(any(), any(), anyInt());
  }

  @Test
  @DisplayName("Seckill sold out - from queue service")
  void testSeckillSoldOutFromQueue() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.pass("Normal user"));
    when(queueService.tryAcquireToken(userId, skuId)).thenReturn(QueueResult.soldOut());

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(410, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(410, response.getBody().code());

    verify(inventoryService, never()).deductStock(any(), any(), anyInt());
  }

  @Test
  @DisplayName("Seckill out of stock - from inventory service")
  void testSeckillOutOfStock() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.pass("Normal user"));
    when(queueService.tryAcquireToken(userId, skuId)).thenReturn(QueueResult.acquired("token123"));
    when(inventoryService.deductStock(skuId, userId, 1)).thenReturn(DeductResult.outOfStock(0));

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(410, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(410, response.getBody().code());

    verify(queueService, times(1)).notifySoldOut(skuId);
  }

  @Test
  @DisplayName("Seckill system error - inventory service failure")
  void testSeckillSystemError() {
    // Given
    String userId = "user123";
    String skuId = "sku456";

    when(activityService.hasReachedPurchaseLimit(userId, skuId)).thenReturn(false);
    when(antiFraudService.assess(any())).thenReturn(RiskAssessment.pass("Normal user"));
    when(queueService.tryAcquireToken(userId, skuId)).thenReturn(QueueResult.acquired("token123"));
    when(inventoryService.deductStock(skuId, userId, 1)).thenReturn(DeductResult.systemError());

    // When
    ResponseEntity<SeckillResponse> response =
        controller.seckill(skuId, userId, 1, null, null, "WEB", httpRequest);

    // Then
    assertEquals(503, response.getStatusCode().value());
    assertNotNull(response.getBody());
    assertEquals(503, response.getBody().code());
  }

  // ==================== Status Query Tests ====================

  @Test
  @DisplayName("Query order status - success")
  void testQueryStatusSuccess() {
    // Given
    String orderId = "order123";
    OrderStatusResult expectedResult =
        new OrderStatusResult(
            orderId,
            com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus.PENDING_PAYMENT,
            "待支付");

    when(queueService.queryStatus(orderId)).thenReturn(expectedResult);

    // When
    ResponseEntity<OrderStatusResult> response = controller.queryStatus(orderId);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(orderId, response.getBody().orderId());
    assertEquals(
        com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus.PENDING_PAYMENT,
        response.getBody().status());

    verify(queueService, times(1)).queryStatus(orderId);
  }

  @Test
  @DisplayName("Query order status - service exception")
  void testQueryStatusException() {
    // Given
    String orderId = "order123";
    when(queueService.queryStatus(orderId)).thenThrow(new RuntimeException("Service error"));

    // When
    ResponseEntity<OrderStatusResult> response = controller.queryStatus(orderId);

    // Then
    assertEquals(500, response.getStatusCode().value());
    assertNull(response.getBody());
  }

  // ==================== Activity Management Tests ====================

  @Test
  @DisplayName("Create activity - success")
  void testCreateActivitySuccess() {
    // Given
    ActivityCreateRequest request =
        new ActivityCreateRequest(
            "sku123",
            "Test Activity",
            100,
            LocalDateTime.now().plusDays(1),
            LocalDateTime.now().plusDays(2),
            1);

    SeckillActivity expectedActivity =
        new SeckillActivity(
            "activity123", "sku123", "Test Activity", 100, request.startTime(), request.endTime());

    when(activityService.createActivity(request)).thenReturn(expectedActivity);
    when(inventoryService.warmupStock("sku123", 100))
        .thenReturn(WarmupResult.success("sku123", 100));

    // When
    ResponseEntity<SeckillActivity> response = controller.createActivity(request);

    // Then
    assertEquals(HttpStatus.CREATED, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals("activity123", response.getBody().getActivityId());

    verify(activityService, times(1)).createActivity(request);
    verify(inventoryService, times(1)).warmupStock("sku123", 100);
  }

  @Test
  @DisplayName("Get activity - found")
  void testGetActivityFound() {
    // Given
    String activityId = "activity123";
    SeckillActivity activity =
        new SeckillActivity(
            activityId,
            "sku123",
            "Test Activity",
            100,
            LocalDateTime.now().plusDays(1),
            LocalDateTime.now().plusDays(2));

    when(activityService.getActivity(activityId)).thenReturn(Optional.of(activity));

    // When
    ResponseEntity<SeckillActivity> response = controller.getActivity(activityId);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(activityId, response.getBody().getActivityId());
  }

  @Test
  @DisplayName("Get activity - not found")
  void testGetActivityNotFound() {
    // Given
    String activityId = "nonexistent";
    when(activityService.getActivity(activityId)).thenReturn(Optional.empty());

    // When
    ResponseEntity<SeckillActivity> response = controller.getActivity(activityId);

    // Then
    assertEquals(HttpStatus.NOT_FOUND, response.getStatusCode());
  }

  @Test
  @DisplayName("Get all activities - success")
  void testGetAllActivitiesSuccess() {
    // Given
    List<SeckillActivity> activities =
        Arrays.asList(
            new SeckillActivity(
                "activity1",
                "sku1",
                "Activity 1",
                100,
                LocalDateTime.now(),
                LocalDateTime.now().plusDays(1)),
            new SeckillActivity(
                "activity2",
                "sku2",
                "Activity 2",
                200,
                LocalDateTime.now(),
                LocalDateTime.now().plusDays(1)));

    when(activityService.getAllActivities()).thenReturn(activities);

    // When
    ResponseEntity<List<SeckillActivity>> response = controller.getAllActivities();

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals(2, response.getBody().size());
  }

  @Test
  @DisplayName("Update activity - success")
  void testUpdateActivitySuccess() {
    // Given
    String activityId = "activity123";
    ActivityUpdateRequest request =
        new ActivityUpdateRequest(
            "Updated Activity", null, null, LocalDateTime.now().plusDays(3), null);

    SeckillActivity updatedActivity =
        new SeckillActivity(
            activityId,
            "sku123",
            "Updated Activity",
            100,
            LocalDateTime.now(),
            LocalDateTime.now().plusDays(3));

    when(activityService.updateActivity(activityId, request)).thenReturn(updatedActivity);

    // When
    ResponseEntity<SeckillActivity> response = controller.updateActivity(activityId, request);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertEquals("Updated Activity", response.getBody().getActivityName());
  }

  @Test
  @DisplayName("Update activity - not found")
  void testUpdateActivityNotFound() {
    // Given
    String activityId = "nonexistent";
    ActivityUpdateRequest request =
        new ActivityUpdateRequest("Updated Activity", null, null, null, null);

    when(activityService.updateActivity(activityId, request))
        .thenThrow(new IllegalArgumentException("Activity not found"));

    // When
    ResponseEntity<SeckillActivity> response = controller.updateActivity(activityId, request);

    // Then
    assertEquals(HttpStatus.NOT_FOUND, response.getStatusCode());
  }

  @Test
  @DisplayName("Delete activity - success")
  void testDeleteActivitySuccess() {
    // Given
    String activityId = "activity123";
    when(activityService.deleteActivity(activityId)).thenReturn(true);

    // When
    ResponseEntity<Void> response = controller.deleteActivity(activityId);

    // Then
    assertEquals(HttpStatus.NO_CONTENT, response.getStatusCode());
    verify(activityService, times(1)).deleteActivity(activityId);
  }

  @Test
  @DisplayName("Delete activity - not found")
  void testDeleteActivityNotFound() {
    // Given
    String activityId = "nonexistent";
    when(activityService.deleteActivity(activityId)).thenReturn(false);

    // When
    ResponseEntity<Void> response = controller.deleteActivity(activityId);

    // Then
    assertEquals(HttpStatus.NOT_FOUND, response.getStatusCode());
  }

  @Test
  @DisplayName("Start activity - success")
  void testStartActivitySuccess() {
    // Given
    String activityId = "activity123";
    when(activityService.startActivity(activityId)).thenReturn(true);

    // When
    ResponseEntity<ApiResponse> response = controller.startActivity(activityId);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertTrue(response.getBody().success());
    verify(activityService, times(1)).startActivity(activityId);
  }

  @Test
  @DisplayName("Start activity - failure")
  void testStartActivityFailure() {
    // Given
    String activityId = "activity123";
    when(activityService.startActivity(activityId)).thenReturn(false);

    // When
    ResponseEntity<ApiResponse> response = controller.startActivity(activityId);

    // Then
    assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
    assertNotNull(response.getBody());
    assertFalse(response.getBody().success());
  }

  @Test
  @DisplayName("End activity - success")
  void testEndActivitySuccess() {
    // Given
    String activityId = "activity123";
    when(activityService.endActivity(activityId)).thenReturn(true);

    // When
    ResponseEntity<ApiResponse> response = controller.endActivity(activityId);

    // Then
    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertTrue(response.getBody().success());
    verify(activityService, times(1)).endActivity(activityId);
  }

  @Test
  @DisplayName("End activity - failure")
  void testEndActivityFailure() {
    // Given
    String activityId = "activity123";
    when(activityService.endActivity(activityId)).thenReturn(false);

    // When
    ResponseEntity<ApiResponse> response = controller.endActivity(activityId);

    // Then
    assertEquals(HttpStatus.BAD_REQUEST, response.getStatusCode());
    assertNotNull(response.getBody());
    assertFalse(response.getBody().success());
  }
}
