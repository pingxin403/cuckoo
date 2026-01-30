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
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResultCode;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;

/**
 * Property 3: 库存扣减返回值正确性
 *
 * <p>**Validates: Requirements 1.3, 1.4**
 *
 * <p>Deduction returns correct status and remaining stock
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 3: 库存扣减返回值正确性")
public class StockDeductionResultPropertyTest {

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
   * Property 3a: Sufficient stock returns SUCCESS
   *
   * <p>When remaining stock >= requested quantity, should return SUCCESS and correct remaining
   * stock
   *
   * <p>**Validates: Requirement 1.3**
   */
  @ParameterizedTest(name = "Sufficient stock: stock={0}, quantity={1}")
  @MethodSource("generateSufficientStockCases")
  void sufficientStockReturnsSuccess(int stock, int quantity) {
    String skuId = "SKU-SUFFICIENT-" + System.nanoTime();
    cleanupSku(skuId);

    // Setup: Warmup stock
    inventoryService.warmupStock(skuId, stock);

    // Act: Deduct stock
    DeductResult result = inventoryService.deductStock(skuId, "USER-1", quantity);

    // Assert: Success with correct remaining stock
    assertThat(result.success()).isTrue();
    assertThat(result.code()).isEqualTo(DeductResultCode.SUCCESS);
    assertThat(result.remainingStock()).isEqualTo(stock - quantity);
    assertThat(result.orderId()).isNotNull();

    // Verify Redis state
    StockInfo stockInfo = inventoryService.getStock(skuId);
    assertThat(stockInfo.remainingStock()).isEqualTo(stock - quantity);
    assertThat(stockInfo.soldCount()).isEqualTo(quantity);
  }

  /**
   * Property 3b: Insufficient stock returns OUT_OF_STOCK
   *
   * <p>When remaining stock < requested quantity, should return OUT_OF_STOCK and stock remains
   * unchanged
   *
   * <p>**Validates: Requirement 1.4**
   */
  @ParameterizedTest(name = "Insufficient stock: stock={0}, quantity={1}")
  @MethodSource("generateInsufficientStockCases")
  void insufficientStockReturnsOutOfStock(int stock, int quantity) {
    String skuId = "SKU-INSUFFICIENT-" + System.nanoTime();
    cleanupSku(skuId);

    // Setup: Warmup stock
    inventoryService.warmupStock(skuId, stock);

    // Act: Attempt to deduct more than available
    DeductResult result = inventoryService.deductStock(skuId, "USER-1", quantity);

    // Assert: Out of stock, no deduction
    assertThat(result.success()).isFalse();
    assertThat(result.code()).isEqualTo(DeductResultCode.OUT_OF_STOCK);
    assertThat(result.orderId()).isNull();

    // Verify stock unchanged
    StockInfo stockInfo = inventoryService.getStock(skuId);
    assertThat(stockInfo.remainingStock()).isEqualTo(stock);
    assertThat(stockInfo.soldCount()).isEqualTo(0);
  }

  /** Generate test cases where stock >= quantity (sufficient) */
  static Stream<Object[]> generateSufficientStockCases() {
    return Stream.generate(
            () -> {
              int stock = random.nextInt(1000) + 1; // 1 to 1000
              int quantity = random.nextInt(stock) + 1; // 1 to stock
              return new Object[] {stock, quantity};
            })
        .limit(50);
  }

  /** Generate test cases where stock < quantity (insufficient) */
  static Stream<Object[]> generateInsufficientStockCases() {
    return Stream.generate(
            () -> {
              int stock = random.nextInt(100); // 0 to 99
              int quantity = stock + random.nextInt(100) + 1; // stock + 1 to stock + 100
              return new Object[] {stock, quantity};
            })
        .limit(50);
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }
}
