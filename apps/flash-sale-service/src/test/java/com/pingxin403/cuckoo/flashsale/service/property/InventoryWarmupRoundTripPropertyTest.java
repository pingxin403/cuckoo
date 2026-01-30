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
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

/**
 * Property 1: 库存预热Round-Trip
 *
 * <p>**Validates: Requirements 1.1**
 *
 * <p>For any SKU and stock N, warmup then query should return N
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 1: 库存预热Round-Trip")
public class InventoryWarmupRoundTripPropertyTest {

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
   * Property 1: 库存预热Round-Trip
   *
   * <p>For any SKU ID and stock quantity N (where N >= 0), after warming up stock to Redis,
   * querying the stock should return remainingStock = N, soldCount = 0, totalStock = N
   *
   * <p>**Validates: Requirement 1.1**
   */
  @ParameterizedTest(name = "Warmup round-trip: skuId={0}, stock={1}")
  @MethodSource("generateWarmupTestCases")
  void warmupRoundTrip(String skuId, int stock) {
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

  /** Generate 100+ test cases with random SKU IDs and stock quantities */
  static Stream<Object[]> generateWarmupTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  generateSkuId(), random.nextInt(100001) // 0 to 100000
                })
        .limit(100);
  }

  private static String generateSkuId() {
    int length = random.nextInt(20) + 1; // 1 to 20 characters
    StringBuilder sb = new StringBuilder("SKU-");
    for (int i = 0; i < length; i++) {
      sb.append((char) ('a' + random.nextInt(26)));
    }
    return sb.toString();
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }
}
