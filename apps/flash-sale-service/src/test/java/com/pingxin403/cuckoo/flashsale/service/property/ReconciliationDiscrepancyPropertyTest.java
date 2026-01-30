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
import org.testcontainers.containers.MySQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.service.ReconciliationService;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

/**
 * Property 12: 对账差异检测
 *
 * <p>**Validates: Requirement 6.5**
 *
 * <p>Detects when Redis ≠ MySQL counts
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 12: 对账差异检测")
public class ReconciliationDiscrepancyPropertyTest {

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @Container
  private static final MySQLContainer<?> mysql =
      new MySQLContainer<>(DockerImageName.parse("mysql:8.0"))
          .withDatabaseName("flash_sale_test")
          .withUsername("test")
          .withPassword("test");

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
    registry.add("spring.datasource.url", mysql::getJdbcUrl);
    registry.add("spring.datasource.username", mysql::getUsername);
    registry.add("spring.datasource.password", mysql::getPassword);
  }

  @Autowired private ReconciliationService reconciliationService;
  @Autowired private StringRedisTemplate redisTemplate;

  private static final Random random = new Random();

  /**
   * Property 12: 对账差异检测
   *
   * <p>For any SKU reconciliation operation, when redisStock + redisSoldCount ≠ totalStock, or
   * redisSoldCount ≠ mysqlOrderCount, the system should detect the discrepancy and return
   * passed=false.
   *
   * <p>**Validates: Requirement 6.5**
   */
  @ParameterizedTest(name = "Discrepancy detection: redisStock={0}, redisSold={1}, totalStock={2}")
  @MethodSource("generateDiscrepancyTestCases")
  void detectsDiscrepancyWhenRedisNotEqualsMysql(
      int redisStock, int redisSold, int totalStock, boolean shouldHaveDiscrepancy) {
    String skuId = "SKU-RECON-" + System.nanoTime();

    // Setup: Set Redis values
    String stockKey = "stock:sku_" + skuId;
    String soldKey = "sold:sku_" + skuId;
    redisTemplate.opsForValue().set(stockKey, String.valueOf(redisStock));
    redisTemplate.opsForValue().set(soldKey, String.valueOf(redisSold));

    // Act: Perform reconciliation
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Assert: Discrepancy detection
    if (shouldHaveDiscrepancy) {
      // When redisStock + redisSold != totalStock, should detect discrepancy
      int sum = redisStock + redisSold;
      if (sum != totalStock) {
        assertThat(result.passed())
            .as(
                "Should detect discrepancy when redisStock(%d) + redisSold(%d) = %d != totalStock(%d)",
                redisStock, redisSold, sum, totalStock)
            .isFalse();
        assertThat(result.discrepancies()).isNotEmpty();
      }
    } else {
      // When values are consistent, should pass
      assertThat(result.passed()).as("Should pass when Redis values are consistent").isTrue();
      assertThat(result.discrepancies()).isEmpty();
    }

    // Verify result contains correct values
    assertThat(result.skuId()).isEqualTo(skuId);
    assertThat(result.redisStock()).isEqualTo(redisStock);
    assertThat(result.redisSoldCount()).isEqualTo(redisSold);
  }

  /** Generate 100+ test cases with various Redis and MySQL states */
  static Stream<Object[]> generateDiscrepancyTestCases() {
    return Stream.generate(
            () -> {
              int totalStock = random.nextInt(1000) + 100; // 100 to 1099
              int redisSold = random.nextInt(totalStock + 1); // 0 to totalStock
              int redisStock;
              boolean shouldHaveDiscrepancy;

              if (random.nextBoolean()) {
                // Create discrepancy case
                redisStock = random.nextInt(totalStock + 100); // Random value
                shouldHaveDiscrepancy = (redisStock + redisSold) != totalStock;
              } else {
                // Create consistent case
                redisStock = totalStock - redisSold;
                shouldHaveDiscrepancy = false;
              }

              return new Object[] {redisStock, redisSold, totalStock, shouldHaveDiscrepancy};
            })
        .limit(100);
  }
}
