package com.pingxin403.cuckoo.flashsale.service.property;

import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.service.impl.InventoryServiceImpl;
import com.redis.testcontainers.RedisContainer;

/**
 * Base class for property-based tests that provides shared Redis infrastructure.
 *
 * <p>This class manages a single Redis container that is shared across all property tests to avoid
 * the overhead of starting/stopping containers for each test.
 */
public abstract class PropertyTestBase {

  protected static RedisContainer redis;
  protected static InventoryServiceImpl inventoryService;
  protected static StringRedisTemplate redisTemplate;
  private static boolean initialized = false;

  /**
   * Initialize the shared test infrastructure.
   *
   * <p>This method is idempotent and can be called multiple times safely.
   */
  protected static synchronized void initializeInfrastructure() {
    if (!initialized) {
      try {
        redis = new RedisContainer(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);
        redis.start();

        TestRedisConfig testRedisConfig =
            new TestRedisConfig(redis.getHost(), redis.getFirstMappedPort());
        redisTemplate = testRedisConfig.stringRedisTemplate();

        // Load Lua scripts
        DefaultRedisScript<Long> stockDeductScript = testRedisConfig.stockDeductScript();
        DefaultRedisScript<Long> stockRollbackScript = testRedisConfig.stockRollbackScript();

        // Create mock dependencies
        MockStockLogRepository mockStockLogRepository = new MockStockLogRepository();
        MockFlashSaleMetrics mockMetrics = new MockFlashSaleMetrics();

        inventoryService =
            new InventoryServiceImpl(
                redisTemplate,
                stockDeductScript,
                stockRollbackScript,
                mockStockLogRepository,
                mockMetrics);

        initialized = true;

        // Add shutdown hook to stop container
        Runtime.getRuntime()
            .addShutdownHook(
                new Thread(
                    () -> {
                      if (redis != null) {
                        redis.stop();
                      }
                    }));
      } catch (Exception e) {
        throw new RuntimeException("Failed to initialize test infrastructure", e);
      }
    }
  }

  /**
   * Clean up Redis keys for a specific SKU.
   *
   * @param skuId the SKU identifier
   */
  protected static void cleanupSku(String skuId) {
    if (redisTemplate != null) {
      redisTemplate.delete("stock:sku_" + skuId);
      redisTemplate.delete("sold:sku_" + skuId);
    }
  }

  /** Clean up all Redis keys. */
  protected static void cleanupAll() {
    if (redisTemplate != null && redisTemplate.getConnectionFactory() != null) {
      try {
        redisTemplate.getConnectionFactory().getConnection().serverCommands().flushAll();
      } catch (Exception e) {
        // Ignore cleanup errors
      }
    }
  }
}
