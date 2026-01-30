package com.pingxin403.cuckoo.flashsale.scheduled;

import static org.mockito.Mockito.*;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import com.pingxin403.cuckoo.flashsale.service.OrderService;

/**
 * Unit tests for OrderTimeoutScheduledTask.
 *
 * <p>Validates: Requirement 5.3
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("OrderTimeoutScheduledTask Tests")
class OrderTimeoutScheduledTaskTest {

  @Mock private OrderService orderService;

  private OrderTimeoutScheduledTask scheduledTask;

  @BeforeEach
  void setUp() {
    scheduledTask = new OrderTimeoutScheduledTask(orderService);
    // Set default timeout to 10 minutes
    ReflectionTestUtils.setField(scheduledTask, "timeoutMinutes", 10);
  }

  @Test
  @DisplayName("Should call orderService.handleTimeoutOrders with configured timeout")
  void shouldCallHandleTimeoutOrdersWithConfiguredTimeout() {
    // Given
    when(orderService.handleTimeoutOrders(10)).thenReturn(5);

    // When
    scheduledTask.handleTimeoutOrders();

    // Then
    verify(orderService).handleTimeoutOrders(10);
  }

  @Test
  @DisplayName("Should handle exception from orderService gracefully")
  void shouldHandleExceptionGracefully() {
    // Given
    when(orderService.handleTimeoutOrders(anyInt()))
        .thenThrow(new RuntimeException("Database error"));

    // When - should not throw exception
    scheduledTask.handleTimeoutOrders();

    // Then
    verify(orderService).handleTimeoutOrders(10);
  }

  @Test
  @DisplayName("Should use custom timeout minutes from configuration")
  void shouldUseCustomTimeoutMinutes() {
    // Given
    ReflectionTestUtils.setField(scheduledTask, "timeoutMinutes", 15);
    when(orderService.handleTimeoutOrders(15)).thenReturn(3);

    // When
    scheduledTask.handleTimeoutOrders();

    // Then
    verify(orderService).handleTimeoutOrders(15);
  }

  @Test
  @DisplayName("Should handle zero timeout orders gracefully")
  void shouldHandleZeroTimeoutOrdersGracefully() {
    // Given
    when(orderService.handleTimeoutOrders(10)).thenReturn(0);

    // When
    scheduledTask.handleTimeoutOrders();

    // Then
    verify(orderService).handleTimeoutOrders(10);
  }
}
