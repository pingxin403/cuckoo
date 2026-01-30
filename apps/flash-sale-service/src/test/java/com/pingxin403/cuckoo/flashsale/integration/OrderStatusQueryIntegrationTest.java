package com.pingxin403.cuckoo.flashsale.integration;

import static org.junit.jupiter.api.Assertions.*;

import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.MySQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.dto.OrderStatusResult;
import com.redis.testcontainers.RedisContainer;

/**
 * Integration test for order status query functionality.
 *
 * <p>Validates Requirement: 4.3 - Order status query interface with Redis caching
 *
 * <p>This test verifies the complete flow:
 *
 * <ol>
 *   <li>OrderService creates an order and caches status to Redis
 *   <li>QueueService queries the status from Redis cache
 *   <li>Status updates are reflected in the cache
 * </ol>
 */
@SpringBootTest
@ActiveProfiles("test")
@Testcontainers
@org.junit.jupiter.api.Disabled("Disabled due to Testcontainers Docker requirement")
@DisplayName("Order Status Query Integration Tests")
class OrderStatusQueryIntegrationTest {

  @Container
  static MySQLContainer<?> mysql =
      new MySQLContainer<>("mysql:8.0")
          .withDatabaseName("flash_sale_test")
          .withUsername("test")
          .withPassword("test");

  @Container
  static RedisContainer redis = new RedisContainer("redis:7-alpine").withExposedPorts(6379);

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.datasource.url", mysql::getJdbcUrl);
    registry.add("spring.datasource.username", mysql::getUsername);
    registry.add("spring.datasource.password", mysql::getPassword);
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", () -> redis.getMappedPort(6379).toString());
  }

  @Autowired private OrderService orderService;

  @Autowired private QueueService queueService;

  @Autowired private StringRedisTemplate stringRedisTemplate;

  private static final String TEST_ORDER_ID = "test-order-integration-123";

  @BeforeEach
  void setUp() {
    // Clean up any existing test data
    stringRedisTemplate.delete("order_status:" + TEST_ORDER_ID);
  }

  @AfterEach
  void tearDown() {
    // Clean up test data
    stringRedisTemplate.delete("order_status:" + TEST_ORDER_ID);
  }

  @Test
  @DisplayName("Should cache order status when order is created and query returns cached status")
  void shouldCacheOrderStatusOnCreationAndQueryFromCache() {
    // Given: Create an order
    OrderMessage message =
        new OrderMessage(
            TEST_ORDER_ID,
            "user-integration-test",
            "sku-integration-test",
            1,
            System.currentTimeMillis(),
            "WEB",
            "trace-integration-test");

    // When: Create order (should cache status to Redis)
    orderService.createOrder(message);

    // Then: Verify status is cached in Redis
    String cachedStatus = stringRedisTemplate.opsForValue().get("order_status:" + TEST_ORDER_ID);
    assertNotNull(cachedStatus, "Order status should be cached in Redis");
    assertEquals("PENDING_PAYMENT", cachedStatus);

    // And: Query status through QueueService (should read from cache)
    OrderStatusResult result = queueService.queryStatus(TEST_ORDER_ID);

    assertNotNull(result);
    assertEquals(TEST_ORDER_ID, result.orderId());
    assertEquals(OrderStatus.PENDING_PAYMENT, result.status());
    assertEquals("待支付", result.message());
  }

  @Test
  @DisplayName("Should update cached status when order status changes")
  void shouldUpdateCachedStatusOnStatusChange() {
    // Given: Create an order
    OrderMessage message =
        new OrderMessage(
            TEST_ORDER_ID,
            "user-integration-test",
            "sku-integration-test",
            1,
            System.currentTimeMillis(),
            "WEB",
            "trace-integration-test");

    orderService.createOrder(message);

    // Verify initial status
    OrderStatusResult initialResult = queueService.queryStatus(TEST_ORDER_ID);
    assertEquals(OrderStatus.PENDING_PAYMENT, initialResult.status());

    // When: Update order status to PAID
    boolean updated = orderService.updateStatus(TEST_ORDER_ID, OrderStatus.PAID);
    assertTrue(updated, "Order status should be updated successfully");

    // Then: Verify cached status is updated
    String cachedStatus = stringRedisTemplate.opsForValue().get("order_status:" + TEST_ORDER_ID);
    assertEquals("PAID", cachedStatus);

    // And: Query returns updated status
    OrderStatusResult updatedResult = queueService.queryStatus(TEST_ORDER_ID);
    assertEquals(OrderStatus.PAID, updatedResult.status());
    assertEquals("已支付", updatedResult.message());
  }

  @Test
  @DisplayName("Should set TTL on cached order status")
  void shouldSetTTLOnCachedOrderStatus() {
    // Given: Create an order
    OrderMessage message =
        new OrderMessage(
            TEST_ORDER_ID,
            "user-integration-test",
            "sku-integration-test",
            1,
            System.currentTimeMillis(),
            "WEB",
            "trace-integration-test");

    // When: Create order
    orderService.createOrder(message);

    // Then: Verify TTL is set (24 hours = 86400 seconds)
    Long ttl = stringRedisTemplate.getExpire("order_status:" + TEST_ORDER_ID, TimeUnit.SECONDS);
    assertNotNull(ttl, "TTL should be set on cached order status");
    assertTrue(ttl > 0, "TTL should be positive");
    assertTrue(ttl <= 24 * 60 * 60, "TTL should be less than or equal to 24 hours (86400 seconds)");
    assertTrue(
        ttl > 24 * 60 * 60 - 10,
        "TTL should be close to 24 hours (allowing 10 seconds for test execution)");
  }

  @Test
  @DisplayName("Should return not found when order does not exist in cache")
  void shouldReturnNotFoundWhenOrderNotInCache() {
    // Given: No order exists with this ID
    String nonExistentOrderId = "non-existent-order-999";

    // When: Query status
    OrderStatusResult result = queueService.queryStatus(nonExistentOrderId);

    // Then: Should return not found
    assertNotNull(result);
    assertEquals(nonExistentOrderId, result.orderId());
    assertNull(result.status());
    assertEquals("订单不存在", result.message());
  }
}
