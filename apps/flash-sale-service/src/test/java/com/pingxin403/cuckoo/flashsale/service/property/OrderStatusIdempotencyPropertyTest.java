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
 * Property 11: 订单状态变更幂等性
 *
 * <p>**Validates: Requirement 5.4**
 *
 * <p>Repeated status changes produce same result
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 11: 订单状态变更幂等性")
public class OrderStatusIdempotencyPropertyTest {

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
   * Property 11: 订单状态变更幂等性
   *
   * <p>For any order status change operation, repeating the same status change request should
   * produce the same result (return true or false), and should not cause data inconsistency.
   *
   * <p>**Validates: Requirement 5.4**
   */
  @ParameterizedTest(name = "Idempotency: targetStatus={0}, repeatCount={1}")
  @MethodSource("generateIdempotencyTestCases")
  void repeatedStatusChangesProduceSameResult(OrderStatus targetStatus, int repeatCount) {
    String orderId = "ORD-IDEMPOTENT-" + System.nanoTime();

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

    // Act: Update status multiple times
    Boolean firstResult = null;
    OrderStatus statusAfterFirst = null;

    for (int i = 0; i < repeatCount; i++) {
      boolean result = orderService.updateStatus(orderId, targetStatus);

      if (i == 0) {
        firstResult = result;
        Optional<SeckillOrder> order = orderService.getOrder(orderId);
        assertThat(order).isPresent();
        statusAfterFirst = order.get().getStatus();
      } else {
        // Assert: Subsequent calls produce same result
        assertThat(result)
            .as("Call %d should produce same result as first call", i + 1)
            .isEqualTo(firstResult);
      }
    }

    // Assert: Final status matches first update result
    Optional<SeckillOrder> finalOrder = orderService.getOrder(orderId);
    assertThat(finalOrder).isPresent();
    assertThat(finalOrder.get().getStatus())
        .as("Final status should match status after first update")
        .isEqualTo(statusAfterFirst);
  }

  /** Generate 100+ test cases with various target statuses and repeat counts */
  static Stream<Object[]> generateIdempotencyTestCases() {
    return Stream.generate(
            () -> {
              OrderStatus[] statuses = {
                OrderStatus.PAID, OrderStatus.CANCELLED, OrderStatus.TIMEOUT
              };
              OrderStatus targetStatus = statuses[random.nextInt(statuses.length)];
              int repeatCount = random.nextInt(5) + 2; // 2 to 6 repeats
              return new Object[] {targetStatus, repeatCount};
            })
        .limit(100);
  }
}
