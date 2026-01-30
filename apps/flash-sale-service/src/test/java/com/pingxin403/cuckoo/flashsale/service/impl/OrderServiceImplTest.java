package com.pingxin403.cuckoo.flashsale.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.repository.SeckillOrderRepository;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.dto.BatchCreateResult;
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;

/**
 * Unit tests for OrderServiceImpl with Redis caching.
 *
 * <p>Validates Requirements: 4.3, 5.1, 5.2, 5.3, 5.4
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("OrderService Implementation Tests")
class OrderServiceImplTest {

  @Mock private SeckillOrderRepository orderRepository;
  @Mock private StringRedisTemplate stringRedisTemplate;
  @Mock private ValueOperations<String, String> valueOperations;
  @Mock private InventoryService inventoryService;

  private OrderServiceImpl orderService;

  @BeforeEach
  void setUp() {
    lenient().when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);
    orderService = new OrderServiceImpl(orderRepository, stringRedisTemplate, inventoryService);
  }

  // ==================== createOrder Tests ====================

  @Test
  @DisplayName("Should create order and cache status to Redis")
  void shouldCreateOrderAndCacheStatus() {
    // Given
    OrderMessage message =
        new OrderMessage(
            "order123", "user123", "sku123", 1, System.currentTimeMillis(), "WEB", "trace123");

    SeckillOrder savedOrder = new SeckillOrder();
    savedOrder.setOrderId("order123");
    savedOrder.setStatus(OrderStatus.PENDING_PAYMENT);

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.empty());
    when(orderRepository.save(any(SeckillOrder.class))).thenReturn(savedOrder);

    // When
    SeckillOrder result = orderService.createOrder(message);

    // Then
    assertNotNull(result);
    assertEquals("order123", result.getOrderId());
    assertEquals(OrderStatus.PENDING_PAYMENT, result.getStatus());

    // Verify Redis cache was set with TTL (Requirement 4.3)
    verify(valueOperations)
        .set(
            eq("order_status:order123"),
            eq("PENDING_PAYMENT"),
            eq(24 * 60 * 60L),
            eq(TimeUnit.SECONDS));
  }

  @Test
  @DisplayName("Should return existing order without creating duplicate")
  void shouldReturnExistingOrderWithoutDuplicate() {
    // Given
    OrderMessage message =
        new OrderMessage(
            "order123", "user123", "sku123", 1, System.currentTimeMillis(), "WEB", "trace123");

    SeckillOrder existingOrder = new SeckillOrder();
    existingOrder.setOrderId("order123");
    existingOrder.setStatus(OrderStatus.PENDING_PAYMENT);

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.of(existingOrder));

    // When
    SeckillOrder result = orderService.createOrder(message);

    // Then
    assertNotNull(result);
    assertEquals("order123", result.getOrderId());

    // Verify no new order was saved
    verify(orderRepository, never()).save(any(SeckillOrder.class));
    // Verify no Redis cache update (order already exists)
    verify(valueOperations, never()).set(anyString(), anyString(), anyLong(), any(TimeUnit.class));
  }

  // ==================== batchCreateOrders Tests ====================

  @Test
  @DisplayName("Should batch create orders and cache all statuses")
  void shouldBatchCreateOrdersAndCacheStatuses() {
    // Given
    OrderMessage msg1 =
        new OrderMessage("order1", "user1", "sku1", 1, System.currentTimeMillis(), "WEB", "trace1");
    OrderMessage msg2 =
        new OrderMessage("order2", "user2", "sku2", 1, System.currentTimeMillis(), "APP", "trace2");

    when(orderRepository.findByOrderId(anyString())).thenReturn(Optional.empty());
    when(orderRepository.save(any(SeckillOrder.class))).thenAnswer(inv -> inv.getArgument(0));

    // When
    BatchCreateResult result = orderService.batchCreateOrders(List.of(msg1, msg2));

    // Then
    assertTrue(result.isFullSuccess());
    assertEquals(2, result.successCount());
    assertEquals(0, result.failedOrderIds().size());

    // Verify both orders were cached to Redis (Requirement 4.3)
    verify(valueOperations, times(2))
        .set(anyString(), eq("PENDING_PAYMENT"), eq(24 * 60 * 60L), eq(TimeUnit.SECONDS));
  }

  @Test
  @DisplayName("Should handle empty batch gracefully")
  void shouldHandleEmptyBatchGracefully() {
    // When
    BatchCreateResult result = orderService.batchCreateOrders(List.of());

    // Then
    assertTrue(result.isFullSuccess());
    assertEquals(0, result.successCount());
  }

  // ==================== updateStatus Tests ====================

  @Test
  @DisplayName("Should update order status and cache to Redis")
  void shouldUpdateOrderStatusAndCache() {
    // Given
    SeckillOrder order = new SeckillOrder();
    order.setOrderId("order123");
    order.setStatus(OrderStatus.PENDING_PAYMENT);

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.of(order));
    when(orderRepository.save(any(SeckillOrder.class))).thenAnswer(inv -> inv.getArgument(0));

    // When
    boolean result = orderService.updateStatus("order123", OrderStatus.PAID);

    // Then
    assertTrue(result);
    assertEquals(OrderStatus.PAID, order.getStatus());
    assertNotNull(order.getPaidAt());

    // Verify Redis cache was updated (Requirement 4.3)
    verify(valueOperations)
        .set(eq("order_status:order123"), eq("PAID"), eq(24 * 60 * 60L), eq(TimeUnit.SECONDS));
  }

  @Test
  @DisplayName("Should be idempotent when updating to same status")
  void shouldBeIdempotentWhenUpdatingToSameStatus() {
    // Given
    SeckillOrder order = new SeckillOrder();
    order.setOrderId("order123");
    order.setStatus(OrderStatus.PAID);

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.of(order));

    // When
    boolean result = orderService.updateStatus("order123", OrderStatus.PAID);

    // Then
    assertTrue(result);

    // Verify no save was called (idempotent)
    verify(orderRepository, never()).save(any(SeckillOrder.class));
    // Verify no Redis update (already in target status)
    verify(valueOperations, never()).set(anyString(), anyString(), anyLong(), any(TimeUnit.class));
  }

  @Test
  @DisplayName("Should reject invalid status transition")
  void shouldRejectInvalidStatusTransition() {
    // Given
    SeckillOrder order = new SeckillOrder();
    order.setOrderId("order123");
    order.setStatus(OrderStatus.PAID);

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.of(order));

    // When: Try to transition from PAID to CANCELLED (invalid)
    boolean result = orderService.updateStatus("order123", OrderStatus.CANCELLED);

    // Then
    assertFalse(result);

    // Verify no save or cache update
    verify(orderRepository, never()).save(any(SeckillOrder.class));
    verify(valueOperations, never()).set(anyString(), anyString(), anyLong(), any(TimeUnit.class));
  }

  @Test
  @DisplayName("Should return false when order not found")
  void shouldReturnFalseWhenOrderNotFound() {
    // Given
    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.empty());

    // When
    boolean result = orderService.updateStatus("order123", OrderStatus.PAID);

    // Then
    assertFalse(result);
  }

  // ==================== handleTimeoutOrders Tests ====================

  @Test
  @DisplayName("Should timeout orders and update cache")
  void shouldTimeoutOrdersAndUpdateCache() {
    // Given
    SeckillOrder order1 = new SeckillOrder();
    order1.setOrderId("order1");
    order1.setSkuId("sku1");
    order1.setQuantity(1);
    order1.setStatus(OrderStatus.PENDING_PAYMENT);
    order1.setCreatedAt(LocalDateTime.now().minusMinutes(15));

    SeckillOrder order2 = new SeckillOrder();
    order2.setOrderId("order2");
    order2.setSkuId("sku2");
    order2.setQuantity(2);
    order2.setStatus(OrderStatus.PENDING_PAYMENT);
    order2.setCreatedAt(LocalDateTime.now().minusMinutes(20));

    when(orderRepository.findByStatusAndCreatedAtBefore(
            eq(OrderStatus.PENDING_PAYMENT), any(LocalDateTime.class)))
        .thenReturn(List.of(order1, order2));
    when(orderRepository.save(any(SeckillOrder.class))).thenAnswer(inv -> inv.getArgument(0));
    when(inventoryService.rollbackStock("sku1", "order1", 1))
        .thenReturn(RollbackResult.success("sku1", "order1", 1, 10));
    when(inventoryService.rollbackStock("sku2", "order2", 2))
        .thenReturn(RollbackResult.success("sku2", "order2", 2, 8));

    // When
    int count = orderService.handleTimeoutOrders(10);

    // Then
    assertEquals(2, count);
    assertEquals(OrderStatus.TIMEOUT, order1.getStatus());
    assertEquals(OrderStatus.TIMEOUT, order2.getStatus());
    assertNotNull(order1.getCancelledAt());
    assertNotNull(order2.getCancelledAt());

    // Verify both orders' status were cached to Redis (Requirement 4.3)
    verify(valueOperations, times(2))
        .set(anyString(), eq("TIMEOUT"), eq(24 * 60 * 60L), eq(TimeUnit.SECONDS));

    // Verify stock rollback was called for both orders (Requirement 5.3)
    verify(inventoryService).rollbackStock("sku1", "order1", 1);
    verify(inventoryService).rollbackStock("sku2", "order2", 2);
  }

  @Test
  @DisplayName("Should handle no timeout orders gracefully")
  void shouldHandleNoTimeoutOrdersGracefully() {
    // Given
    when(orderRepository.findByStatusAndCreatedAtBefore(
            eq(OrderStatus.PENDING_PAYMENT), any(LocalDateTime.class)))
        .thenReturn(List.of());

    // When
    int count = orderService.handleTimeoutOrders(10);

    // Then
    assertEquals(0, count);
  }

  // ==================== getOrder Tests ====================

  @Test
  @DisplayName("Should get order by orderId")
  void shouldGetOrderByOrderId() {
    // Given
    SeckillOrder order = new SeckillOrder();
    order.setOrderId("order123");

    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.of(order));

    // When
    Optional<SeckillOrder> result = orderService.getOrder("order123");

    // Then
    assertTrue(result.isPresent());
    assertEquals("order123", result.get().getOrderId());
  }

  @Test
  @DisplayName("Should return empty when order not found")
  void shouldReturnEmptyWhenOrderNotFound() {
    // Given
    when(orderRepository.findByOrderId("order123")).thenReturn(Optional.empty());

    // When
    Optional<SeckillOrder> result = orderService.getOrder("order123");

    // Then
    assertTrue(result.isEmpty());
  }
}
