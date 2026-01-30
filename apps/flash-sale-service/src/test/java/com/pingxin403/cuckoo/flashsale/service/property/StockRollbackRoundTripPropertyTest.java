package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Random;
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
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;

/**
 * Property 4: 超时回滚Round-Trip
 *
 * <p>**Validates: Requirements 1.5, 5.3**
 *
 * <p>Deduct then rollback restores original stock
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 4: 超时回滚Round-Trip")
public class StockRollbackRoundTripPropertyTest {

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
   * Property 4: 超时回滚Round-Trip
   *
   * <p>For any successful stock deduction, executing rollback should restore the stock to the
   * pre-deduction amount. That is: deduct(quantity) → rollback(quantity) results in stock =
   * original stock.
   *
   * <p>**Validates: Requirements 1.5, 5.3**
   */
  @ParameterizedTest(name = "Rollback round-trip: initialStock={0}, quantity={1}")
  @MethodSource("generateRollbackTestCases")
  void deductThenRollbackRestoresStock(int initialStock, int quantity) {
    String skuId = "SKU-ROLLBACK-" + System.nanoTime();
    cleanupSku(skuId);

    // Setup: Warmup stock
    inventoryService.warmupStock(skuId, initialStock);

    // Record initial state
    StockInfo beforeDeduct = inventoryService.getStock(skuId);
    assertThat(beforeDeduct.remainingStock()).isEqualTo(initialStock);
    assertThat(beforeDeduct.soldCount()).isEqualTo(0);

    // Act: Deduct stock
    DeductResult deductResult = inventoryService.deductStock(skuId, "USER-1", quantity);
    assertThat(deductResult.success()).isTrue();

    // Verify deduction
    StockInfo afterDeduct = inventoryService.getStock(skuId);
    assertThat(afterDeduct.remainingStock()).isEqualTo(initialStock - quantity);
    assertThat(afterDeduct.soldCount()).isEqualTo(quantity);

    // Act: Rollback stock
    RollbackResult rollbackResult =
        inventoryService.rollbackStock(skuId, deductResult.orderId(), quantity);
    assertThat(rollbackResult.success()).isTrue();

    // Assert: Stock restored to original
    StockInfo afterRollback = inventoryService.getStock(skuId);
    assertThat(afterRollback.remainingStock())
        .as("Stock should be restored to original amount")
        .isEqualTo(initialStock);
    assertThat(afterRollback.soldCount()).as("Sold count should be restored to 0").isEqualTo(0);
  }

  /** Generate 100+ test cases with various stock and quantity combinations */
  static Stream<Object[]> generateRollbackTestCases() {
    return Stream.generate(
            () -> {
              int stock = random.nextInt(1000) + 1; // 1 to 1000
              int quantity = random.nextInt(stock) + 1; // 1 to stock
              return new Object[] {stock, quantity};
            })
        .limit(100);
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }
}
