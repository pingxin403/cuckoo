package com.pingxin403.cuckoo.flashsale.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.RedisConnectionFailureException;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;

/**
 * Unit tests for QueueServiceImpl token bucket control.
 *
 * <p>Validates Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("QueueService Token Bucket Control Tests")
class QueueServiceImplTest {

  @Mock private StringRedisTemplate stringRedisTemplate;
  @Mock private ValueOperations<String, String> valueOperations;
  @Mock private com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics metrics;

  private QueueServiceImpl queueService;

  @BeforeEach
  void setUp() {
    lenient().when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);
    queueService = new QueueServiceImpl(stringRedisTemplate, metrics);
  }

  // ==================== tryAcquireToken Tests ====================

  @Test
  @DisplayName("Should return acquired when token is available")
  void shouldReturnAcquiredWhenTokenAvailable() {
    // Given
    String userId = "user123";
    String skuId = "sku123";

    // Mock: SKU not sold out
    when(valueOperations.get("sold_out:sku123")).thenReturn(null);

    // Mock: Token bucket not initialized, will be initialized
    when(stringRedisTemplate.hasKey("token_bucket:sku123")).thenReturn(false);

    // Mock: After initialization and refill, tokens available
    when(valueOperations.get("token_bucket:sku123"))
        .thenReturn("5000") // Initial capacity
        .thenReturn("5000") // After refill check
        .thenReturn("4999"); // After decrement

    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");
    when(valueOperations.get("token_bucket_last:sku123"))
        .thenReturn(String.valueOf(System.currentTimeMillis()));
    when(valueOperations.get("token_bucket_capacity:sku123")).thenReturn("5000");

    // Mock: Decrement succeeds (returns non-negative value)
    when(valueOperations.decrement("token_bucket:sku123")).thenReturn(4999L);

    // When
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Then
    assertEquals(200, result.code());
    assertEquals("获得令牌", result.message());
    assertEquals(0, result.estimatedWait());
    assertNotNull(result.queueToken());
    assertTrue(result.queueToken().startsWith("QT-"));

    verify(valueOperations).decrement("token_bucket:sku123");
  }

  @Test
  @DisplayName("Should return queuing when no tokens available")
  void shouldReturnQueuingWhenNoTokensAvailable() {
    // Given
    String userId = "user123";
    String skuId = "sku123";

    // Mock: SKU not sold out
    when(valueOperations.get("sold_out:sku123")).thenReturn(null);

    // Mock: Token bucket exists
    when(stringRedisTemplate.hasKey("token_bucket:sku123")).thenReturn(true);

    // Mock: No tokens available
    when(valueOperations.get("token_bucket:sku123"))
        .thenReturn("0") // Before refill
        .thenReturn("0") // After refill (no time elapsed)
        .thenReturn("0"); // For estimated wait time calculation

    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");
    when(valueOperations.get("token_bucket_last:sku123"))
        .thenReturn(String.valueOf(System.currentTimeMillis()));
    when(valueOperations.get("token_bucket_capacity:sku123")).thenReturn("5000");

    // Mock: Decrement returns negative (no tokens)
    when(valueOperations.decrement("token_bucket:sku123")).thenReturn(-1L);

    // When
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Then
    assertEquals(202, result.code());
    assertEquals("排队中", result.message());
    assertTrue(result.estimatedWait() >= 0);
    assertNotNull(result.queueToken());

    verify(valueOperations).decrement("token_bucket:sku123");
    verify(valueOperations).increment("token_bucket:sku123"); // Restore after failed attempt
  }

  @Test
  @DisplayName("Should return sold out when SKU is sold out")
  void shouldReturnSoldOutWhenSkuSoldOut() {
    // Given
    String userId = "user123";
    String skuId = "sku123";

    // Mock: SKU is sold out
    when(valueOperations.get("sold_out:sku123")).thenReturn("1");

    // When
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Then
    assertEquals(410, result.code());
    assertEquals("商品已售罄", result.message());
    assertEquals(0, result.estimatedWait());
    assertNull(result.queueToken());

    // Should not attempt to acquire token
    verify(valueOperations, never()).decrement(anyString());
  }

  @Test
  @DisplayName("Should return queuing when userId is null")
  void shouldReturnQueuingWhenUserIdNull() {
    // When
    QueueResult result = queueService.tryAcquireToken(null, "sku123");

    // Then
    assertEquals(202, result.code());
    assertEquals(0, result.estimatedWait());
  }

  @Test
  @DisplayName("Should return queuing when skuId is null")
  void shouldReturnQueuingWhenSkuIdNull() {
    // When
    QueueResult result = queueService.tryAcquireToken("user123", null);

    // Then
    assertEquals(202, result.code());
    assertEquals(0, result.estimatedWait());
  }

  @Test
  @DisplayName("Should return queuing when Redis connection fails")
  void shouldReturnQueuingWhenRedisConnectionFails() {
    // Given
    String userId = "user123";
    String skuId = "sku123";

    // Mock: Redis connection failure when checking sold out status
    // This will be caught by isSoldOut's catch block and return false
    // Then when trying to check if token bucket exists, throw exception
    when(valueOperations.get("sold_out:sku123")).thenReturn(null);
    when(stringRedisTemplate.hasKey("token_bucket:sku123")).thenReturn(true);

    // Mock: Throw exception during decrement (this will be caught by consumeToken)
    // But we need to throw it from a place that propagates to main catch
    // The only way is to throw RedisConnectionFailureException from decrement
    when(valueOperations.get("token_bucket:sku123")).thenReturn("100");
    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");
    when(valueOperations.get("token_bucket_last:sku123"))
        .thenReturn(String.valueOf(System.currentTimeMillis() - 1000)); // 1 second ago
    when(valueOperations.get("token_bucket_capacity:sku123")).thenReturn("5000");

    // Throw exception during decrement - this will be caught by consumeToken and return false
    // Then it will try to get estimated wait time, and we can throw there
    when(valueOperations.decrement("token_bucket:sku123"))
        .thenThrow(new RedisConnectionFailureException("Connection failed"));

    // When
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Then: consumeToken catches the exception and returns false, leading to queuing
    // But the estimated wait time calculation will use the mocked values
    assertEquals(202, result.code());
    assertEquals("排队中", result.message());
    // The estimated wait time will be calculated based on available tokens (100 > 0, so wait = 0)
    // But since decrement failed, consumeToken returns false, so it goes to queuing path
    assertNotNull(result.queueToken());
  }

  @Test
  @DisplayName("Should initialize token bucket on first request")
  void shouldInitializeTokenBucketOnFirstRequest() {
    // Given
    String userId = "user123";
    String skuId = "sku123";

    // Mock: SKU not sold out
    when(valueOperations.get("sold_out:sku123")).thenReturn(null);

    // Mock: Token bucket doesn't exist
    when(stringRedisTemplate.hasKey("token_bucket:sku123")).thenReturn(false);

    // Mock: After initialization
    when(valueOperations.get("token_bucket:sku123")).thenReturn("5000");
    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");
    when(valueOperations.get("token_bucket_last:sku123"))
        .thenReturn(String.valueOf(System.currentTimeMillis()));
    when(valueOperations.get("token_bucket_capacity:sku123")).thenReturn("5000");
    when(valueOperations.decrement("token_bucket:sku123")).thenReturn(4999L);

    // When
    queueService.tryAcquireToken(userId, skuId);

    // Then: Verify initialization (may be called multiple times during refill)
    verify(valueOperations, atLeastOnce()).set("token_bucket:sku123", "5000");
    verify(valueOperations).set("token_bucket_rate:sku123", "1000");
    verify(valueOperations).set("token_bucket_capacity:sku123", "5000");
    verify(valueOperations, atLeastOnce()).set(eq("token_bucket_last:sku123"), anyString());
  }

  // ==================== getEstimatedWaitTime Tests ====================

  @Test
  @DisplayName("Should return 0 wait time when tokens available")
  void shouldReturnZeroWaitTimeWhenTokensAvailable() {
    // Given
    String skuId = "sku123";

    // Mock: Tokens available
    when(valueOperations.get("token_bucket:sku123")).thenReturn("100");
    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");

    // When
    int waitTime = queueService.getEstimatedWaitTime(skuId);

    // Then
    assertEquals(0, waitTime);
  }

  @Test
  @DisplayName("Should calculate wait time based on queue depth")
  void shouldCalculateWaitTimeBasedOnQueueDepth() {
    // Given
    String skuId = "sku123";

    // Mock: Queue depth of 5000 (negative tokens), rate of 1000/sec
    when(valueOperations.get("token_bucket:sku123")).thenReturn("-5000");
    when(valueOperations.get("token_bucket_rate:sku123")).thenReturn("1000");

    // When
    int waitTime = queueService.getEstimatedWaitTime(skuId);

    // Then
    // Expected: 5000 / 1000 = 5 seconds
    assertEquals(5, waitTime);
  }

  @Test
  @DisplayName("Should return 0 when skuId is null")
  void shouldReturnZeroWhenSkuIdNullForWaitTime() {
    // When
    int waitTime = queueService.getEstimatedWaitTime(null);

    // Then
    assertEquals(0, waitTime);
  }

  @Test
  @DisplayName("Should return 0 when Redis connection fails for wait time")
  void shouldReturnZeroWhenRedisFailsForWaitTime() {
    // Given
    String skuId = "sku123";

    // Mock: Redis connection failure
    when(valueOperations.get(anyString()))
        .thenThrow(new RedisConnectionFailureException("Connection failed"));

    // When
    int waitTime = queueService.getEstimatedWaitTime(skuId);

    // Then
    assertEquals(0, waitTime);
  }

  @Test
  @DisplayName("Should handle invalid number format gracefully")
  void shouldHandleInvalidNumberFormatForWaitTime() {
    // Given
    String skuId = "sku123";

    // Mock: Invalid number format
    when(valueOperations.get("token_bucket:sku123")).thenReturn("invalid");

    // When
    int waitTime = queueService.getEstimatedWaitTime(skuId);

    // Then
    assertEquals(0, waitTime);
  }

  // ==================== notifySoldOut Tests ====================

  @Test
  @DisplayName("Should set sold out flag and clear token bucket")
  void shouldSetSoldOutFlagAndClearTokenBucket() {
    // Given
    String skuId = "sku123";

    // When
    queueService.notifySoldOut(skuId);

    // Then
    verify(valueOperations).set("sold_out:sku123", "1");
    verify(stringRedisTemplate).delete("token_bucket:sku123");
  }

  @Test
  @DisplayName("Should handle null skuId gracefully for notifySoldOut")
  void shouldHandleNullSkuIdForNotifySoldOut() {
    // When
    queueService.notifySoldOut(null);

    // Then: Should not throw exception
    verify(valueOperations, never()).set(anyString(), anyString());
  }

  @Test
  @DisplayName("Should handle Redis failure gracefully for notifySoldOut")
  void shouldHandleRedisFailureForNotifySoldOut() {
    // Given
    String skuId = "sku123";

    // Mock: Redis connection failure
    doThrow(new RedisConnectionFailureException("Connection failed"))
        .when(valueOperations)
        .set(anyString(), anyString());

    // When: Should not throw exception
    assertDoesNotThrow(() -> queueService.notifySoldOut(skuId));
  }

  // ==================== queryStatus Tests ====================

  @Test
  @DisplayName("Should return pending payment status")
  void shouldReturnPendingPaymentStatus() {
    // Given
    String orderId = "order123";

    // Mock: Order status is PENDING_PAYMENT
    when(valueOperations.get("order_status:order123")).thenReturn("PENDING_PAYMENT");

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertEquals(OrderStatus.PENDING_PAYMENT, result.status());
    assertEquals("待支付", result.message());
  }

  @Test
  @DisplayName("Should return paid status")
  void shouldReturnPaidStatus() {
    // Given
    String orderId = "order123";

    // Mock: Order status is PAID
    when(valueOperations.get("order_status:order123")).thenReturn("PAID");

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertEquals(OrderStatus.PAID, result.status());
    assertEquals("已支付", result.message());
  }

  @Test
  @DisplayName("Should return cancelled status")
  void shouldReturnCancelledStatus() {
    // Given
    String orderId = "order123";

    // Mock: Order status is CANCELLED
    when(valueOperations.get("order_status:order123")).thenReturn("CANCELLED");

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertEquals(OrderStatus.CANCELLED, result.status());
    assertEquals("已取消", result.message());
  }

  @Test
  @DisplayName("Should return timeout status")
  void shouldReturnTimeoutStatus() {
    // Given
    String orderId = "order123";

    // Mock: Order status is TIMEOUT
    when(valueOperations.get("order_status:order123")).thenReturn("TIMEOUT");

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertEquals(OrderStatus.TIMEOUT, result.status());
    assertEquals("超时取消", result.message());
  }

  @Test
  @DisplayName("Should return not found when order doesn't exist")
  void shouldReturnNotFoundWhenOrderDoesNotExist() {
    // Given
    String orderId = "order123";

    // Mock: Order not found
    when(valueOperations.get("order_status:order123")).thenReturn(null);

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertNull(result.status());
    assertEquals("订单不存在", result.message());
  }

  @Test
  @DisplayName("Should return not found when orderId is null")
  void shouldReturnNotFoundWhenOrderIdNull() {
    // When
    OrderStatusResult result = queueService.queryStatus(null);

    // Then
    assertNull(result.orderId());
    assertNull(result.status());
    assertEquals("订单不存在", result.message());
  }

  @Test
  @DisplayName("Should handle invalid status value gracefully")
  void shouldHandleInvalidStatusValue() {
    // Given
    String orderId = "order123";

    // Mock: Invalid status value
    when(valueOperations.get("order_status:order123")).thenReturn("INVALID_STATUS");

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertNull(result.status());
    assertEquals("订单不存在", result.message());
  }

  @Test
  @DisplayName("Should handle Redis failure gracefully for queryStatus")
  void shouldHandleRedisFailureForQueryStatus() {
    // Given
    String orderId = "order123";

    // Mock: Redis connection failure
    when(valueOperations.get(anyString()))
        .thenThrow(new RedisConnectionFailureException("Connection failed"));

    // When
    OrderStatusResult result = queueService.queryStatus(orderId);

    // Then
    assertEquals(orderId, result.orderId());
    assertNull(result.status());
    assertEquals("订单不存在", result.message());
  }
}
