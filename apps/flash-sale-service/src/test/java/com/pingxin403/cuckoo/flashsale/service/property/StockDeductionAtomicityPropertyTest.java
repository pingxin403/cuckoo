package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;

/**
 * Property 2: 库存扣减原子性-不超卖
 *
 * <p>**Validates: Requirements 1.2, 6.1**
 *
 * <p>Concurrent deductions never oversell
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 2: 库存扣减原子性-不超卖")
public class StockDeductionAtomicityPropertyTest {

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
  }

  @Autowired private InventoryService inventoryService;
  @Autowired private StringRedisTemplate redisTemplate;

  private static final Random random = new Random();

  /**
   * Property 2: 库存扣减原子性-不超卖
   *
   * <p>For any initial stock N and any number of concurrent deduction requests M (each deducting
   * 1), the final sold count should equal min(N, M), and remaining stock should equal max(0, N-M).
   * No negative stock allowed.
   *
   * <p>**Validates: Requirements 1.2, 6.1**
   */
  @ParameterizedTest(name = "Atomicity: initialStock={0}, concurrentRequests={1}")
  @MethodSource("generateAtomicityTestCases")
  void concurrentDeductionsNeverOversell(int initialStock, int concurrentRequests)
      throws InterruptedException {
    String skuId = "SKU-ATOMIC-" + System.nanoTime();
    cleanupSku(skuId);

    // Setup: Warmup stock
    inventoryService.warmupStock(skuId, initialStock);

    // Act: Perform concurrent deductions
    ExecutorService executor = Executors.newFixedThreadPool(20);
    CountDownLatch latch = new CountDownLatch(concurrentRequests);
    AtomicInteger successCount = new AtomicInteger(0);
    AtomicInteger failureCount = new AtomicInteger(0);

    for (int i = 0; i < concurrentRequests; i++) {
      final int userId = i;
      executor.submit(
          () -> {
            try {
              DeductResult result = inventoryService.deductStock(skuId, "USER-" + userId, 1);
              if (result.success()) {
                successCount.incrementAndGet();
              } else {
                failureCount.incrementAndGet();
              }
            } finally {
              latch.countDown();
            }
          });
    }

    latch.await();
    executor.shutdown();

    // Assert: Check final state
    StockInfo finalStock = inventoryService.getStock(skuId);

    int expectedSold = Math.min(initialStock, concurrentRequests);
    int expectedRemaining = Math.max(0, initialStock - concurrentRequests);

    assertThat(finalStock.soldCount())
        .as("Sold count should equal min(initialStock, requests)")
        .isEqualTo(expectedSold);

    assertThat(finalStock.remainingStock())
        .as("Remaining stock should equal max(0, initialStock - requests)")
        .isEqualTo(expectedRemaining);

    assertThat(finalStock.remainingStock())
        .as("Stock should never be negative")
        .isGreaterThanOrEqualTo(0);

    assertThat(successCount.get())
        .as("Success count should match sold count")
        .isEqualTo(expectedSold);

    assertThat(successCount.get() + failureCount.get())
        .as("All requests should be accounted for")
        .isEqualTo(concurrentRequests);
  }

  /** Generate 100+ test cases with various stock and concurrency levels */
  static Stream<Object[]> generateAtomicityTestCases() {
    List<Object[]> testCases = new ArrayList<>();

    // Edge cases
    testCases.add(new Object[] {0, 10}); // No stock, multiple requests
    testCases.add(new Object[] {1, 10}); // Single item, multiple requests
    testCases.add(new Object[] {10, 1}); // Multiple items, single request
    testCases.add(new Object[] {10, 10}); // Exact match

    // Random cases
    for (int i = 0; i < 96; i++) {
      int stock = random.nextInt(100) + 1; // 1 to 100
      int requests = random.nextInt(150) + 1; // 1 to 150
      testCases.add(new Object[] {stock, requests});
    }

    return testCases.stream();
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }
}
