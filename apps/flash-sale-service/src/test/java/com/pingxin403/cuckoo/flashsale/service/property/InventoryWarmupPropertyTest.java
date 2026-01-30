package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

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
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

import net.jqwik.api.*;

/**
 * Property-based tests for inventory warmup round-trip behavior.
 *
 * <p>**Validates: Requirement 1.1**
 *
 * <p>Property 1: 库存预热Round-Trip - For any SKU and stock quantity N, after warming up stock to
 * Redis, querying Redis should return stock value equal to N and sold value equal to 0.
 *
 * <p>This test uses jqwik for property-based testing with 100 iterations per property.
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 1: 库存预热Round-Trip")
public class InventoryWarmupPropertyTest {

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

  /**
   * Property 1: 库存预热Round-Trip
   *
   * <p>For any SKU ID and stock quantity N (where N >= 0), after warming up stock to Redis,
   * querying the stock should return:
   *
   * <ul>
   *   <li>remainingStock = N
   *   <li>soldCount = 0
   *   <li>totalStock = N
   * </ul>
   *
   * <p>This property ensures that the warmup operation correctly initializes Redis state.
   *
   * <p>**Validates: Requirement 1.1**
   */
  @Property(tries = 100)
  @Label("Warmup round-trip: stock query returns warmed-up value")
  void warmupRoundTrip(
      @ForAll("validSkuIds") String skuId, @ForAll("validStockQuantities") int stock) {
    // Clean up any existing data for this SKU
    cleanupSku(skuId);

    // Act: Warmup stock
    WarmupResult warmupResult = inventoryService.warmupStock(skuId, stock);

    // Assert: Warmup succeeded
    assertThat(warmupResult.success()).isTrue();
    assertThat(warmupResult.skuId()).isEqualTo(skuId);
    assertThat(warmupResult.stock()).isEqualTo(stock);

    // Act: Query stock
    StockInfo stockInfo = inventoryService.getStock(skuId);

    // Assert: Stock info matches warmed-up values
    assertThat(stockInfo.skuId()).isEqualTo(skuId);
    assertThat(stockInfo.remainingStock()).isEqualTo(stock);
    assertThat(stockInfo.soldCount()).isEqualTo(0);
    assertThat(stockInfo.totalStock()).isEqualTo(stock);
  }

  /**
   * Property 1b: Multiple warmups overwrite previous values
   *
   * <p>For any SKU ID, warming up with different stock values should overwrite the previous value.
   * The last warmup value should be the one returned by getStock().
   *
   * <p>**Validates: Requirement 1.1**
   */
  @Property(tries = 100)
  @Label("Multiple warmups overwrite previous values")
  void multipleWarmupsOverwrite(
      @ForAll("validSkuIds") String skuId,
      @ForAll("validStockQuantities") int stock1,
      @ForAll("validStockQuantities") int stock2) {
    // Clean up any existing data for this SKU
    cleanupSku(skuId);

    // Act: First warmup
    inventoryService.warmupStock(skuId, stock1);

    // Act: Second warmup (overwrite)
    WarmupResult warmupResult = inventoryService.warmupStock(skuId, stock2);

    // Assert: Second warmup succeeded
    assertThat(warmupResult.success()).isTrue();

    // Act: Query stock
    StockInfo stockInfo = inventoryService.getStock(skuId);

    // Assert: Stock info reflects the second warmup value
    assertThat(stockInfo.remainingStock()).isEqualTo(stock2);
    assertThat(stockInfo.soldCount()).isEqualTo(0);
    assertThat(stockInfo.totalStock()).isEqualTo(stock2);
  }

  // ==================== Helper Methods ====================

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }

  // ==================== Arbitraries ====================

  @Provide
  Arbitrary<String> validSkuIds() {
    return Arbitraries.strings()
        .withCharRange('a', 'z')
        .ofMinLength(1)
        .ofMaxLength(20)
        .map(s -> "SKU-" + s);
  }

  @Provide
  Arbitrary<Integer> validStockQuantities() {
    return Arbitraries.integers().between(0, 100000);
  }
}
