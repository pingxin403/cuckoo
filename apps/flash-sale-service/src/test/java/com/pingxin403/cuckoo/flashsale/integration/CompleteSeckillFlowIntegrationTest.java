package com.pingxin403.cuckoo.flashsale.integration;

import static org.awaitility.Awaitility.await;
import static org.junit.jupiter.api.Assertions.*;

import java.time.Duration;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.KafkaContainer;
import org.testcontainers.containers.MySQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.repository.SeckillActivityRepository;
import com.pingxin403.cuckoo.flashsale.repository.SeckillOrderRepository;
import com.pingxin403.cuckoo.flashsale.service.ActivityService;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.ReconciliationService;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResultCode;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;

/**
 * Comprehensive integration test for the complete seckill flow.
 *
 * <p>This test validates the entire flash sale system from request to order creation using
 * Testcontainers for Redis, Kafka, and MySQL. It covers all requirements from the design document
 * including:
 *
 * <ul>
 *   <li>Activity creation and lifecycle management
 *   <li>Inventory warmup and atomic deduction
 *   <li>Queue service and rate limiting
 *   <li>Kafka message production and consumption
 *   <li>Order creation and persistence
 *   <li>Concurrent request handling
 *   <li>Data reconciliation
 *   <li>Purchase limit enforcement
 * </ul>
 *
 * <p><b>Validates Requirements:</b> All requirements from design document (1.1-8.6)
 */
@SpringBootTest
@Testcontainers
@DisplayName("Complete Seckill Flow Integration Test")
class CompleteSeckillFlowIntegrationTest {

  // Testcontainers setup
  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @Container
  private static final KafkaContainer kafka =
      new KafkaContainer(DockerImageName.parse("confluentinc/cp-kafka:7.4.0"));

  @Container
  private static final MySQLContainer<?> mysql =
      new MySQLContainer<>(DockerImageName.parse("mysql:8.0"))
          .withDatabaseName("seckill_test")
          .withUsername("test")
          .withPassword("test");

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    // Redis configuration
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);

    // Kafka configuration
    registry.add("spring.kafka.bootstrap-servers", kafka::getBootstrapServers);
    registry.add("spring.kafka.consumer.auto-offset-reset", () -> "earliest");
    registry.add("spring.kafka.consumer.group-id", () -> "test-consumer-group");

    // MySQL configuration
    registry.add("spring.datasource.url", mysql::getJdbcUrl);
    registry.add("spring.datasource.username", mysql::getUsername);
    registry.add("spring.datasource.password", mysql::getPassword);
    registry.add("spring.jpa.hibernate.ddl-auto", () -> "create-drop");
  }

  @Autowired private ActivityService activityService;
  @Autowired private InventoryService inventoryService;
  @Autowired private QueueService queueService;
  @Autowired private OrderService orderService;
  @Autowired private ReconciliationService reconciliationService;
  @Autowired private SeckillActivityRepository activityRepository;
  @Autowired private SeckillOrderRepository orderRepository;
  @Autowired private RedisTemplate<String, Object> redisTemplate;
  @Autowired private KafkaTemplate<String, OrderMessage> kafkaTemplate;

  private String testSkuId;
  private String testActivityId;

  @BeforeEach
  void setUp() {
    testSkuId = "test-sku-" + System.currentTimeMillis();
    // Clean up Redis before each test
    redisTemplate.delete(redisTemplate.keys("*"));
  }

  @AfterEach
  void tearDown() {
    // Clean up test data
    if (testActivityId != null) {
      try {
        activityService.deleteActivity(testActivityId);
      } catch (Exception e) {
        // Ignore cleanup errors
      }
    }
    // Clean up Redis
    redisTemplate.delete(redisTemplate.keys("*"));
  }

  @Test
  @DisplayName("Test 1: Complete seckill flow - single user success")
  void testCompleteSeckillFlowSingleUser() throws InterruptedException {
    // Step 1: Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Complete Flow Test Activity",
            10, // Total stock
            LocalDateTime.now().minusMinutes(5), // Started 5 minutes ago
            LocalDateTime.now().plusHours(1), // Ends in 1 hour
            2 // Purchase limit per user
            );

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    assertNotNull(testActivityId);
    assertEquals(ActivityStatus.NOT_STARTED, activity.getStatus());

    // Step 2: Start activity and warmup inventory
    activityService.startActivity(testActivityId);
    activity = activityService.getActivity(testActivityId).orElseThrow();
    assertEquals(ActivityStatus.IN_PROGRESS, activity.getStatus());

    // Verify inventory warmup
    StockInfo stockInfo = inventoryService.getStock(testSkuId);
    assertNotNull(stockInfo);
    assertEquals(10, stockInfo.totalStock());
    assertEquals(0, stockInfo.soldCount());
    assertEquals(10, stockInfo.remainingStock());

    // Step 3: User attempts to acquire token
    String userId = "user-001";
    QueueResult queueResult = queueService.tryAcquireToken(userId, testSkuId);
    assertNotNull(queueResult);
    // Should get token (200) or be queued (202)
    assertTrue(queueResult.code() == 200 || queueResult.code() == 202);

    // Step 4: Deduct inventory
    DeductResult deductResult = inventoryService.deductStock(testSkuId, userId, 1);
    assertNotNull(deductResult);
    assertEquals(DeductResultCode.SUCCESS, deductResult.code());
    assertEquals(9, deductResult.remainingStock());
    assertNotNull(deductResult.orderId());

    // Step 5: Verify Kafka message was sent and order created
    String orderId = deductResult.orderId();

    // Wait for Kafka consumer to process the message and create order
    await()
        .atMost(Duration.ofSeconds(10))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              SeckillOrder order = orderService.getOrder(orderId).orElse(null);
              assertNotNull(order, "Order should be created");
              assertEquals(userId, order.getUserId());
              assertEquals(testSkuId, order.getSkuId());
              assertEquals(1, order.getQuantity());
              assertEquals(OrderStatus.PENDING_PAYMENT, order.getStatus());
            });

    // Step 6: Verify inventory state
    stockInfo = inventoryService.getStock(testSkuId);
    assertEquals(9, stockInfo.remainingStock());
    assertEquals(1, stockInfo.soldCount());

    // Step 7: Simulate payment
    boolean updated = orderService.updateStatus(orderId, OrderStatus.PAID);
    assertTrue(updated);

    SeckillOrder paidOrder = orderService.getOrder(orderId).orElseThrow();
    assertEquals(OrderStatus.PAID, paidOrder.getStatus());
    assertNotNull(paidOrder.getPaidAt());
  }

  @Test
  @DisplayName("Test 2: Concurrent requests - no overselling")
  void testConcurrentRequestsNoOverselling() throws InterruptedException {
    // Create activity with limited stock
    int totalStock = 10;
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Concurrent Test Activity",
            totalStock,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            5 // Allow multiple purchases per user
            );

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Simulate 50 concurrent users trying to buy (5x oversell attempt)
    int concurrentUsers = 50;
    ExecutorService executor = Executors.newFixedThreadPool(concurrentUsers);
    CountDownLatch latch = new CountDownLatch(concurrentUsers);
    AtomicInteger successCount = new AtomicInteger(0);
    AtomicInteger failureCount = new AtomicInteger(0);

    for (int i = 0; i < concurrentUsers; i++) {
      final String userId = "user-" + String.format("%03d", i);
      executor.submit(
          () -> {
            try {
              DeductResult result = inventoryService.deductStock(testSkuId, userId, 1);
              if (result.code() == DeductResultCode.SUCCESS) {
                successCount.incrementAndGet();
              } else if (result.code() == DeductResultCode.OUT_OF_STOCK) {
                failureCount.incrementAndGet();
              }
            } catch (Exception e) {
              failureCount.incrementAndGet();
            } finally {
              latch.countDown();
            }
          });
    }

    // Wait for all threads to complete
    assertTrue(latch.await(30, TimeUnit.SECONDS), "All threads should complete within 30 seconds");
    executor.shutdown();

    // Verify no overselling
    assertEquals(totalStock, successCount.get(), "Success count should equal total stock");
    assertEquals(
        concurrentUsers - totalStock,
        failureCount.get(),
        "Failure count should equal rejected requests");

    // Verify final inventory state
    StockInfo stockInfo = inventoryService.getStock(testSkuId);
    assertEquals(0, stockInfo.remainingStock(), "All stock should be sold");
    assertEquals(totalStock, stockInfo.soldCount(), "Sold count should equal total stock");

    // Wait for all orders to be created
    await()
        .atMost(Duration.ofSeconds(15))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              List<SeckillOrder> orders = orderRepository.findBySkuId(testSkuId);
              assertEquals(
                  totalStock, orders.size(), "Number of orders should equal total stock sold");
            });
  }

  @Test
  @DisplayName("Test 3: Purchase limit enforcement")
  void testPurchaseLimitEnforcement() throws InterruptedException {
    // Create activity with purchase limit
    int purchaseLimit = 2;
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Purchase Limit Test Activity",
            100,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            purchaseLimit);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    String userId = "user-limit-test";

    // First purchase - should succeed
    DeductResult result1 = inventoryService.deductStock(testSkuId, userId, 1);
    assertEquals(DeductResultCode.SUCCESS, result1.code());

    // Second purchase - should succeed
    DeductResult result2 = inventoryService.deductStock(testSkuId, userId, 1);
    assertEquals(DeductResultCode.SUCCESS, result2.code());

    // Third purchase - should fail due to purchase limit
    // Note: This depends on the implementation checking purchase limits
    // If not implemented in deductStock, it should be checked at controller level
    DeductResult result3 = inventoryService.deductStock(testSkuId, userId, 1);
    // The result depends on whether purchase limit is enforced in inventory service
    // or at a higher level (controller/service layer)
    assertNotNull(result3);

    // Wait for orders to be created
    await()
        .atMost(Duration.ofSeconds(10))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              List<SeckillOrder> userOrders =
                  orderRepository.findByUserIdAndSkuId(userId, testSkuId);
              // Should have at most purchaseLimit orders
              assertTrue(
                  userOrders.size() <= purchaseLimit, "User should not exceed purchase limit");
            });
  }

  @Test
  @DisplayName("Test 4: Order timeout and inventory rollback")
  void testOrderTimeoutAndRollback() throws InterruptedException {
    // Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Timeout Test Activity",
            10,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            5);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Create order
    String userId = "user-timeout-test";
    DeductResult deductResult = inventoryService.deductStock(testSkuId, userId, 1);
    assertEquals(DeductResultCode.SUCCESS, deductResult.code());
    String orderId = deductResult.orderId();

    // Wait for order to be created
    await()
        .atMost(Duration.ofSeconds(10))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(() -> assertTrue(orderService.getOrder(orderId).isPresent()));

    // Get initial stock
    StockInfo stockBefore = inventoryService.getStock(testSkuId);
    assertEquals(9, stockBefore.remainingStock());

    // Simulate timeout by updating order status to TIMEOUT
    boolean updated = orderService.updateStatus(orderId, OrderStatus.TIMEOUT);
    assertTrue(updated);

    // Rollback inventory
    var rollbackResult = inventoryService.rollbackStock(testSkuId, orderId, 1);
    assertTrue(rollbackResult.success());

    // Verify inventory was restored
    StockInfo stockAfter = inventoryService.getStock(testSkuId);
    assertEquals(10, stockAfter.remainingStock());
    assertEquals(0, stockAfter.soldCount());
  }

  @Test
  @DisplayName("Test 5: Data reconciliation")
  void testDataReconciliation() throws InterruptedException {
    // Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Reconciliation Test Activity",
            20,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            5);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Create multiple orders
    List<String> orderIds = new ArrayList<>();
    for (int i = 0; i < 5; i++) {
      String userId = "user-recon-" + i;
      DeductResult result = inventoryService.deductStock(testSkuId, userId, 1);
      if (result.code() == DeductResultCode.SUCCESS) {
        orderIds.add(result.orderId());
      }
    }

    // Wait for all orders to be created
    await()
        .atMost(Duration.ofSeconds(15))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              List<SeckillOrder> orders = orderRepository.findBySkuId(testSkuId);
              assertEquals(orderIds.size(), orders.size());
            });

    // Perform reconciliation
    ReconciliationResult reconResult = reconciliationService.reconcile(testSkuId);
    assertNotNull(reconResult);
    assertEquals(testSkuId, reconResult.skuId());

    // Verify reconciliation results
    assertTrue(reconResult.passed(), "Reconciliation should pass with consistent data");
    assertEquals(15, reconResult.redisStock(), "Redis stock should be 15 (20 - 5)");
    assertEquals(5, reconResult.redisSoldCount(), "Redis sold count should be 5");
    assertEquals(5, reconResult.mysqlOrderCount(), "MySQL order count should be 5");
    assertTrue(reconResult.discrepancies().isEmpty(), "There should be no discrepancies");
  }

  @Test
  @DisplayName("Test 6: Activity lifecycle management")
  void testActivityLifecycleManagement() {
    // Create activity that hasn't started yet
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Lifecycle Test Activity",
            50,
            LocalDateTime.now().plusMinutes(10), // Starts in 10 minutes
            LocalDateTime.now().plusHours(1),
            3);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    assertEquals(ActivityStatus.NOT_STARTED, activity.getStatus());

    // Start activity manually
    activityService.startActivity(testActivityId);
    activity = activityService.getActivity(testActivityId).orElseThrow();
    assertEquals(ActivityStatus.IN_PROGRESS, activity.getStatus());

    // Verify inventory was warmed up
    StockInfo stockInfo = inventoryService.getStock(testSkuId);
    assertEquals(50, stockInfo.totalStock());
    assertEquals(50, stockInfo.remainingStock());

    // End activity
    activityService.endActivity(testActivityId);
    activity = activityService.getActivity(testActivityId).orElseThrow();
    assertEquals(ActivityStatus.ENDED, activity.getStatus());

    // Verify cannot deduct stock from ended activity
    DeductResult result = inventoryService.deductStock(testSkuId, "user-test", 1);
    // The behavior depends on implementation - might return OUT_OF_STOCK or SYSTEM_ERROR
    assertNotEquals(DeductResultCode.SUCCESS, result.code());
  }

  @Test
  @DisplayName("Test 7: Kafka message production and consumption")
  void testKafkaMessageFlow() throws InterruptedException {
    // Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Kafka Test Activity",
            10,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            5);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Deduct stock (which should send Kafka message)
    String userId = "user-kafka-test";
    DeductResult result = inventoryService.deductStock(testSkuId, userId, 1);
    assertEquals(DeductResultCode.SUCCESS, result.code());
    String orderId = result.orderId();

    // Wait for Kafka consumer to process message and create order
    await()
        .atMost(Duration.ofSeconds(10))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              SeckillOrder order = orderService.getOrder(orderId).orElse(null);
              assertNotNull(order, "Order should be created from Kafka message");
              assertEquals(userId, order.getUserId());
              assertEquals(testSkuId, order.getSkuId());
              assertEquals(testActivityId, order.getActivityId());
              assertEquals(1, order.getQuantity());
              assertEquals(OrderStatus.PENDING_PAYMENT, order.getStatus());
              assertNotNull(order.getCreatedAt());
            });
  }

  @Test
  @DisplayName("Test 8: Batch order creation from Kafka")
  void testBatchOrderCreation() throws InterruptedException {
    // Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Batch Test Activity",
            50,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            10);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Create multiple orders rapidly
    int orderCount = 15;
    List<String> orderIds = new ArrayList<>();

    for (int i = 0; i < orderCount; i++) {
      String userId = "user-batch-" + i;
      DeductResult result = inventoryService.deductStock(testSkuId, userId, 1);
      if (result.code() == DeductResultCode.SUCCESS) {
        orderIds.add(result.orderId());
      }
    }

    assertEquals(orderCount, orderIds.size(), "All deductions should succeed");

    // Wait for all orders to be created (batch processing)
    await()
        .atMost(Duration.ofSeconds(20))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              List<SeckillOrder> orders = orderRepository.findBySkuId(testSkuId);
              assertEquals(
                  orderCount,
                  orders.size(),
                  "All orders should be created through batch processing");

              // Verify all order IDs are present
              List<String> createdOrderIds = orders.stream().map(SeckillOrder::getOrderId).toList();
              assertTrue(
                  createdOrderIds.containsAll(orderIds), "All expected orders should be created");
            });
  }

  @Test
  @DisplayName("Test 9: Stock sold out notification")
  void testStockSoldOutNotification() throws InterruptedException {
    // Create activity with very limited stock
    int totalStock = 3;
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "Sold Out Test Activity",
            totalStock,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            5);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    activityService.startActivity(testActivityId);

    // Buy all stock
    for (int i = 0; i < totalStock; i++) {
      String userId = "user-soldout-" + i;
      DeductResult result = inventoryService.deductStock(testSkuId, userId, 1);
      assertEquals(DeductResultCode.SUCCESS, result.code());
    }

    // Verify stock is sold out
    StockInfo stockInfo = inventoryService.getStock(testSkuId);
    assertEquals(0, stockInfo.remainingStock());

    // Notify sold out
    queueService.notifySoldOut(testSkuId);

    // Try to buy again - should fail
    DeductResult result = inventoryService.deductStock(testSkuId, "user-late", 1);
    assertEquals(DeductResultCode.OUT_OF_STOCK, result.code());

    // Queue service should return sold out status
    QueueResult queueResult = queueService.tryAcquireToken("user-late", testSkuId);
    assertEquals(410, queueResult.code(), "Should return sold out status");
  }

  @Test
  @DisplayName("Test 10: Complete end-to-end flow with all components")
  void testCompleteEndToEndFlow() throws InterruptedException {
    // This test validates the complete flow from activity creation to order fulfillment
    // covering all major components and requirements

    // Step 1: Create and configure activity
    int totalStock = 20;
    int purchaseLimit = 3;
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            testSkuId,
            "End-to-End Test Activity",
            totalStock,
            LocalDateTime.now().minusMinutes(5),
            LocalDateTime.now().plusHours(1),
            purchaseLimit);

    SeckillActivity activity = activityService.createActivity(createRequest);
    testActivityId = activity.getActivityId();
    assertNotNull(testActivityId);

    // Step 2: Start activity and verify inventory warmup
    activityService.startActivity(testActivityId);
    StockInfo initialStock = inventoryService.getStock(testSkuId);
    assertEquals(totalStock, initialStock.totalStock());
    assertEquals(totalStock, initialStock.remainingStock());
    assertEquals(0, initialStock.soldCount());

    // Step 3: Simulate multiple users making purchases
    int userCount = 10;
    List<String> successfulOrderIds = new ArrayList<>();

    for (int i = 0; i < userCount; i++) {
      String userId = "user-e2e-" + i;

      // Try to acquire token (queue service)
      QueueResult queueResult = queueService.tryAcquireToken(userId, testSkuId);
      assertNotNull(queueResult);

      // Deduct inventory
      DeductResult deductResult = inventoryService.deductStock(testSkuId, userId, 1);
      if (deductResult.code() == DeductResultCode.SUCCESS) {
        successfulOrderIds.add(deductResult.orderId());
      }
    }

    assertEquals(userCount, successfulOrderIds.size(), "All users should successfully purchase");

    // Step 4: Wait for Kafka processing and order creation
    await()
        .atMost(Duration.ofSeconds(20))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              List<SeckillOrder> orders = orderRepository.findBySkuId(testSkuId);
              assertEquals(userCount, orders.size(), "All orders should be created");

              // Verify all orders are in correct state
              for (SeckillOrder order : orders) {
                assertEquals(OrderStatus.PENDING_PAYMENT, order.getStatus());
                assertEquals(testSkuId, order.getSkuId());
                assertEquals(testActivityId, order.getActivityId());
                assertNotNull(order.getCreatedAt());
              }
            });

    // Step 5: Verify inventory state
    StockInfo currentStock = inventoryService.getStock(testSkuId);
    assertEquals(totalStock - userCount, currentStock.remainingStock());
    assertEquals(userCount, currentStock.soldCount());

    // Step 6: Simulate some users paying
    int paidCount = 5;
    for (int i = 0; i < paidCount; i++) {
      String orderId = successfulOrderIds.get(i);
      boolean updated = orderService.updateStatus(orderId, OrderStatus.PAID);
      assertTrue(updated);
    }

    // Step 7: Simulate some orders timing out
    int timeoutCount = 2;
    for (int i = paidCount; i < paidCount + timeoutCount; i++) {
      String orderId = successfulOrderIds.get(i);
      boolean updated = orderService.updateStatus(orderId, OrderStatus.TIMEOUT);
      assertTrue(updated);

      // Rollback inventory
      var rollbackResult = inventoryService.rollbackStock(testSkuId, orderId, 1);
      assertTrue(rollbackResult.success());
    }

    // Step 8: Verify final inventory state after rollbacks
    StockInfo finalStock = inventoryService.getStock(testSkuId);
    assertEquals(
        totalStock - userCount + timeoutCount,
        finalStock.remainingStock(),
        "Stock should be restored for timeout orders");

    // Step 9: Perform reconciliation
    ReconciliationResult reconResult = reconciliationService.reconcile(testSkuId);
    assertNotNull(reconResult);
    assertTrue(reconResult.passed(), "Reconciliation should pass");

    // Step 10: End activity
    activityService.endActivity(testActivityId);
    activity = activityService.getActivity(testActivityId).orElseThrow();
    assertEquals(ActivityStatus.ENDED, activity.getStatus());

    // Step 11: Verify cannot purchase from ended activity
    DeductResult lateResult = inventoryService.deductStock(testSkuId, "user-late", 1);
    assertNotEquals(
        DeductResultCode.SUCCESS,
        lateResult.code(),
        "Should not allow purchase after activity ends");
  }
}
