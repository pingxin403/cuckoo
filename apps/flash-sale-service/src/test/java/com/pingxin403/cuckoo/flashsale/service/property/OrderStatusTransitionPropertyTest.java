package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Optional;
import java.util.Random;
import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.MySQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.service.OrderService;

/**
 * Property 10: 订单状态流转正确性
 *
 * <p>**Validates: Requirements 5.1, 5.2**
 *
 * <p>Order states follow valid transitions
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 10: 订单状态流转正确性")
public class OrderStatusTransitionPropertyTest {

  @Container
  private static final MySQLContainer<?> mysql =
      new MySQLContainer<>(DockerImageName.parse("mysql:8.0"))
          .withDatabaseName("flash_sale_test")
          .withUsername("test")
          .withPassword("test");

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.datasource.url", mysql::getJdbcUrl);
    registry.add("spring.datasource.username", mysql::getUsername);
    registry.add("spring.datasource.password", mysql::getPassword);
  }

  @Autowired private OrderService orderService;

  private static final Random random = new Random();

  /**
   * Property 10a: New orders start with PENDING_PAYMENT
   *
   * <p>For any order, when created, the status should be PENDING_PAYMENT
   *
   * <p>**Validates: Requirement 5.1**
   */
  @ParameterizedTest(name = "New order: orderId={0}")
  @MethodSource("generateNewOrderTestCases")
  void newOrdersStartWithPendingPayment(String orderId, String userId, String skuId) {
    // Act: Create order
    OrderMessage message =
        OrderMessage.builder()
            .orderId(orderId)
            .userId(userId)
            .skuId(skuId)
            .quantity(random.nextInt(10) + 1)
            .timestamp(System.currentTimeMillis())
            .source("TEST")
            .traceId("TRACE-" + System.nanoTime())
            .build();

    SeckillOrder order = orderService.createOrder(message);

    // Assert: Status is PENDING_PAYMENT
    assertThat(order.getStatus())
        .as("New order should have status PENDING_PAYMENT")
        .isEqualTo(OrderStatus.PENDING_PAYMENT);

    // Verify in database
    Optional<SeckillOrder> retrieved = orderService.getOrder(orderId);
    assertThat(retrieved).isPresent();
    assertThat(retrieved.get().getStatus()).isEqualTo(OrderStatus.PENDING_PAYMENT);
  }

  /**
   * Property 10b: Valid status transitions
   *
   * <p>Order status should follow valid transitions: PENDING_PAYMENT → PAID or PENDING_PAYMENT →
   * CANCELLED/TIMEOUT
   *
   * <p>**Validates: Requirement 5.2**
   */
  @ParameterizedTest(name = "Status transition: {0} → {1}")
  @MethodSource("generateStatusTransitionTestCases")
  void orderStatusFollowsValidTransitions(
      OrderStatus initialStatus, OrderStatus targetStatus, boolean shouldSucceed) {
    String orderId = "ORD-TRANSITION-" + System.nanoTime();

    // Create order
    OrderMessage message =
        OrderMessage.builder()
            .orderId(orderId)
            .userId("USER-" + random.nextInt(1000))
            .skuId("SKU-" + random.nextInt(100))
            .quantity(1)
            .timestamp(System.currentTimeMillis())
            .source("TEST")
            .traceId("TRACE-" + System.nanoTime())
            .build();

    orderService.createOrder(message);

    // Act: Update status
    boolean result = orderService.updateStatus(orderId, targetStatus);

    // Assert: Transition result matches expected
    if (shouldSucceed) {
      assertThat(result)
          .as("Transition from %s to %s should succeed", initialStatus, targetStatus)
          .isTrue();

      Optional<SeckillOrder> order = orderService.getOrder(orderId);
      assertThat(order).isPresent();
      assertThat(order.get().getStatus()).isEqualTo(targetStatus);
    }
  }

  /** Generate test cases for new order creation */
  static Stream<Object[]> generateNewOrderTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  "ORD-NEW-" + System.nanoTime(),
                  "USER-" + random.nextInt(10000),
                  "SKU-" + random.nextInt(1000)
                })
        .limit(50);
  }

  /** Generate test cases for status transitions */
  static Stream<Object[]> generateStatusTransitionTestCases() {
    return Stream.of(
            // Valid transitions (repeated multiple times)
            Stream.generate(
                    () -> new Object[] {OrderStatus.PENDING_PAYMENT, OrderStatus.PAID, true})
                .limit(17),
            Stream.generate(
                    () -> new Object[] {OrderStatus.PENDING_PAYMENT, OrderStatus.CANCELLED, true})
                .limit(17),
            Stream.generate(
                    () -> new Object[] {OrderStatus.PENDING_PAYMENT, OrderStatus.TIMEOUT, true})
                .limit(16))
        .flatMap(s -> s);
  }
}
