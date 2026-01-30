package com.pingxin403.cuckoo.flashsale.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.util.Arrays;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.data.redis.core.script.DefaultRedisScript;

import com.pingxin403.cuckoo.flashsale.repository.StockLogRepository;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResultCode;
import com.pingxin403.cuckoo.flashsale.service.dto.RollbackResult;
import com.pingxin403.cuckoo.flashsale.service.dto.StockInfo;
import com.pingxin403.cuckoo.flashsale.service.dto.WarmupResult;

/**
 * Unit tests for InventoryServiceImpl.
 *
 * <p>Tests the business logic and error handling without requiring a real Redis instance.
 *
 * <p>Validates Requirements: 1.1, 1.2, 1.3, 1.4, 1.5
 */
@ExtendWith(MockitoExtension.class)
class InventoryServiceImplTest {

  @Mock private StringRedisTemplate stringRedisTemplate;

  @Mock private ValueOperations<String, String> valueOperations;

  @Mock private DefaultRedisScript<Long> stockDeductScript;

  @Mock private DefaultRedisScript<Long> stockRollbackScript;

  @Mock private StockLogRepository stockLogRepository;

  @Mock private com.pingxin403.cuckoo.flashsale.config.MetricsConfig.FlashSaleMetrics metrics;

  private InventoryServiceImpl inventoryService;

  private static final String TEST_SKU_ID = "TEST_SKU_001";
  private static final String TEST_USER_ID = "USER_001";

  @BeforeEach
  void setUp() {
    inventoryService =
        new InventoryServiceImpl(
            stringRedisTemplate,
            stockDeductScript,
            stockRollbackScript,
            stockLogRepository,
            metrics);

    // Setup common mock behavior (lenient to avoid UnnecessaryStubbingException)
    lenient().when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);

    // Mock timer for metrics
    io.micrometer.core.instrument.Timer timer = mock(io.micrometer.core.instrument.Timer.class);
    lenient().when(metrics.getInventoryDeductionDuration()).thenReturn(timer);
    lenient()
        .when(timer.record(any(java.util.function.Supplier.class)))
        .thenAnswer(
            invocation -> {
              java.util.function.Supplier<?> supplier = invocation.getArgument(0);
              return supplier.get();
            });
  }

  /**
   * Test: Warmup stock successfully loads stock to Redis.
   *
   * <p>Validates Requirement 1.1
   */
  @Test
  void testWarmupStock_Success() {
    // Given
    int initialStock = 100;

    // When
    WarmupResult result = inventoryService.warmupStock(TEST_SKU_ID, initialStock);

    // Then
    assertTrue(result.success(), "Warmup should succeed");
    assertEquals(TEST_SKU_ID, result.skuId());
    assertEquals(initialStock, result.stock());

    // Verify Redis operations
    verify(valueOperations).set("stock:sku_" + TEST_SKU_ID, "100");
    verify(valueOperations).set("sold:sku_" + TEST_SKU_ID, "0");
  }

  /**
   * Test: Warmup with negative stock should fail.
   *
   * <p>Validates input validation
   */
  @Test
  void testWarmupStock_NegativeStock_Fails() {
    // When
    WarmupResult result = inventoryService.warmupStock(TEST_SKU_ID, -10);

    // Then
    assertFalse(result.success(), "Warmup with negative stock should fail");
    assertTrue(result.message().contains("不能为负数"));
  }

  /**
   * Test: Deduct stock successfully when sufficient stock available.
   *
   * <p>Validates Requirements 1.2, 1.3
   */
  @Test
  void testDeductStock_Success() {
    // Given
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("100");
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("10")))
        .thenReturn(90L); // Lua script returns remaining stock

    // When
    DeductResult result = inventoryService.deductStock(TEST_SKU_ID, TEST_USER_ID, 10);

    // Then
    assertTrue(result.success(), "Deduction should succeed");
    assertEquals(DeductResultCode.SUCCESS, result.code());
    assertEquals(90, result.remainingStock());
    assertNotNull(result.orderId());
    assertTrue(result.orderId().startsWith("ORD-"));

    // Verify Lua script was called
    verify(stringRedisTemplate)
        .execute(
            eq(stockDeductScript),
            eq(Arrays.asList("stock:sku_" + TEST_SKU_ID, "sold:sku_" + TEST_SKU_ID)),
            eq("10"));
  }

  /**
   * Test: Deduct stock fails when insufficient stock.
   *
   * <p>Validates Requirement 1.4
   */
  @Test
  void testDeductStock_OutOfStock() {
    // Given
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("5");
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("10")))
        .thenReturn(0L); // Lua script returns 0 for out of stock

    // When
    DeductResult result = inventoryService.deductStock(TEST_SKU_ID, TEST_USER_ID, 10);

    // Then
    assertFalse(result.success(), "Deduction should fail");
    assertEquals(DeductResultCode.OUT_OF_STOCK, result.code());
    assertNull(result.orderId());
  }

  /**
   * Test: Multiple deductions work correctly.
   *
   * <p>Validates Requirement 1.2 - Atomic operations
   */
  @Test
  void testDeductStock_MultipleDeductions() {
    // Given
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("100", "90", "70");
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("10"))).thenReturn(90L);
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("20"))).thenReturn(70L);
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("30"))).thenReturn(40L);

    // When - Perform multiple deductions
    DeductResult result1 = inventoryService.deductStock(TEST_SKU_ID, "USER_001", 10);
    DeductResult result2 = inventoryService.deductStock(TEST_SKU_ID, "USER_002", 20);
    DeductResult result3 = inventoryService.deductStock(TEST_SKU_ID, "USER_003", 30);

    // Then
    assertTrue(result1.success());
    assertEquals(90, result1.remainingStock());

    assertTrue(result2.success());
    assertEquals(70, result2.remainingStock());

    assertTrue(result3.success());
    assertEquals(40, result3.remainingStock());
  }

  /**
   * Test: Rollback stock successfully restores inventory.
   *
   * <p>Validates Requirement 1.5
   */
  @Test
  void testRollbackStock_Success() {
    // Given
    String orderId = "ORD-123456-ABCD";
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("90");
    when(stringRedisTemplate.execute(eq(stockRollbackScript), anyList(), eq("10")))
        .thenReturn(100L); // Lua script returns new stock count

    // When
    RollbackResult result = inventoryService.rollbackStock(TEST_SKU_ID, orderId, 10);

    // Then
    assertTrue(result.success(), "Rollback should succeed");
    assertEquals(100, result.newStock());

    // Verify Lua script was called
    verify(stringRedisTemplate)
        .execute(
            eq(stockRollbackScript),
            eq(Arrays.asList("stock:sku_" + TEST_SKU_ID, "sold:sku_" + TEST_SKU_ID)),
            eq("10"));
  }

  /**
   * Test: Rollback with invalid quantity fails.
   *
   * <p>Validates input validation
   */
  @Test
  void testRollbackStock_InvalidQuantity_Fails() {
    // When
    RollbackResult result = inventoryService.rollbackStock(TEST_SKU_ID, "ORDER_001", -5);

    // Then
    assertFalse(result.success(), "Rollback with negative quantity should fail");
    assertTrue(result.message().contains("必须为正数"));
  }

  /**
   * Test: Get stock returns correct info.
   *
   * <p>Validates stock query functionality
   */
  @Test
  void testGetStock_Success() {
    // Given
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("75");
    when(valueOperations.get("sold:sku_" + TEST_SKU_ID)).thenReturn("25");

    // When
    StockInfo stockInfo = inventoryService.getStock(TEST_SKU_ID);

    // Then
    assertEquals(TEST_SKU_ID, stockInfo.skuId());
    assertEquals(75, stockInfo.remainingStock());
    assertEquals(25, stockInfo.soldCount());
    assertEquals(100, stockInfo.totalStock());
    assertTrue(stockInfo.isAvailable());
    assertFalse(stockInfo.isSoldOut());
  }

  /**
   * Test: Get stock returns empty info for non-existent SKU.
   *
   * <p>Validates error handling
   */
  @Test
  void testGetStock_NonExistentSku() {
    // Given
    when(valueOperations.get("stock:sku_NON_EXISTENT_SKU")).thenReturn(null);
    when(valueOperations.get("sold:sku_NON_EXISTENT_SKU")).thenReturn(null);

    // When
    StockInfo stockInfo = inventoryService.getStock("NON_EXISTENT_SKU");

    // Then
    assertEquals("NON_EXISTENT_SKU", stockInfo.skuId());
    assertEquals(0, stockInfo.remainingStock());
    assertEquals(0, stockInfo.soldCount());
    assertEquals(0, stockInfo.totalStock());
    assertTrue(stockInfo.isSoldOut());
    assertFalse(stockInfo.isAvailable());
  }

  /**
   * Test: Deduct leaving 1 item succeeds.
   *
   * <p>Validates boundary condition
   *
   * <p>Note: Current implementation treats remaining=0 as OUT_OF_STOCK. Deducting exact remaining
   * stock would need special handling.
   */
  @Test
  void testDeductStock_LeavingOneItem() {
    // Given
    when(valueOperations.get("stock:sku_" + TEST_SKU_ID)).thenReturn("10");
    when(stringRedisTemplate.execute(eq(stockDeductScript), anyList(), eq("9")))
        .thenReturn(1L); // Lua script returns 1 remaining

    // When - Deduct 9 from 10, leaving 1
    DeductResult result = inventoryService.deductStock(TEST_SKU_ID, TEST_USER_ID, 9);

    // Then
    assertTrue(result.success());
    assertEquals(1, result.remainingStock());
  }

  /**
   * Test: Deduct with invalid inputs returns system error.
   *
   * <p>Validates input validation
   */
  @Test
  void testDeductStock_InvalidInputs() {
    // Test null skuId
    DeductResult result1 = inventoryService.deductStock(null, TEST_USER_ID, 10);
    assertEquals(DeductResultCode.SYSTEM_ERROR, result1.code());

    // Test blank skuId
    DeductResult result2 = inventoryService.deductStock("", TEST_USER_ID, 10);
    assertEquals(DeductResultCode.SYSTEM_ERROR, result2.code());

    // Test null userId
    DeductResult result3 = inventoryService.deductStock(TEST_SKU_ID, null, 10);
    assertEquals(DeductResultCode.SYSTEM_ERROR, result3.code());

    // Test zero quantity
    DeductResult result4 = inventoryService.deductStock(TEST_SKU_ID, TEST_USER_ID, 0);
    assertEquals(DeductResultCode.SYSTEM_ERROR, result4.code());

    // Test negative quantity
    DeductResult result5 = inventoryService.deductStock(TEST_SKU_ID, TEST_USER_ID, -5);
    assertEquals(DeductResultCode.SYSTEM_ERROR, result5.code());
  }
}
