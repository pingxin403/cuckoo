package com.pingxin403.cuckoo.flashsale.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.util.List;
import java.util.Optional;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.pingxin403.cuckoo.flashsale.model.ReconciliationLog;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;
import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;
import com.pingxin403.cuckoo.flashsale.model.enums.ReconciliationStatus;
import com.pingxin403.cuckoo.flashsale.repository.ReconciliationLogRepository;
import com.pingxin403.cuckoo.flashsale.repository.SeckillActivityRepository;
import com.pingxin403.cuckoo.flashsale.repository.SeckillOrderRepository;
import com.pingxin403.cuckoo.flashsale.service.dto.Discrepancy;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

/**
 * Unit tests for ReconciliationServiceImpl.
 *
 * <p>Tests the reconciliation logic for detecting and fixing data inconsistencies between Redis and
 * MySQL.
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("ReconciliationService Unit Tests")
class ReconciliationServiceImplTest {

  @Mock private StringRedisTemplate stringRedisTemplate;

  @Mock private ValueOperations<String, String> valueOperations;

  @Mock private SeckillActivityRepository activityRepository;

  @Mock private SeckillOrderRepository orderRepository;

  @Mock private ReconciliationLogRepository reconciliationLogRepository;

  private ObjectMapper objectMapper;

  private ReconciliationServiceImpl reconciliationService;

  @BeforeEach
  void setUp() {
    objectMapper = new ObjectMapper();
    reconciliationService =
        new ReconciliationServiceImpl(
            stringRedisTemplate,
            activityRepository,
            orderRepository,
            reconciliationLogRepository,
            objectMapper);

    // Setup default mock behavior (lenient to avoid unnecessary stubbing warnings)
    lenient().when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);
  }

  @Test
  @DisplayName("Should pass reconciliation when Redis and MySQL data match")
  void testReconcile_Success() {
    // Given
    String skuId = "SKU001";
    int redisStock = 100;
    int redisSold = 50;
    long mysqlOrderCount = 50L;
    int totalStock = 150;

    when(valueOperations.get("stock:sku_" + skuId)).thenReturn(String.valueOf(redisStock));
    when(valueOperations.get("sold:sku_" + skuId)).thenReturn(String.valueOf(redisSold));
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(mysqlOrderCount);

    SeckillActivity activity = new SeckillActivity();
    activity.setSkuId(skuId);
    activity.setTotalStock(totalStock);
    when(activityRepository.findActiveActivityBySkuId(skuId)).thenReturn(Optional.of(activity));

    // When
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Then
    assertTrue(result.passed(), "Reconciliation should pass");
    assertEquals(skuId, result.skuId());
    assertEquals(redisStock, result.redisStock());
    assertEquals(redisSold, result.redisSoldCount());
    assertEquals(mysqlOrderCount, result.mysqlOrderCount());
    assertTrue(result.discrepancies().isEmpty(), "Should have no discrepancies");

    // Verify reconciliation log was saved
    ArgumentCaptor<ReconciliationLog> logCaptor = ArgumentCaptor.forClass(ReconciliationLog.class);
    verify(reconciliationLogRepository).save(logCaptor.capture());
    ReconciliationLog savedLog = logCaptor.getValue();
    assertEquals(ReconciliationStatus.NORMAL, savedLog.getStatus());
  }

  @Test
  @DisplayName("Should detect order count mismatch")
  void testReconcile_OrderCountMismatch() {
    // Given
    String skuId = "SKU002";
    int redisStock = 100;
    int redisSold = 50;
    long mysqlOrderCount = 45L; // Mismatch!

    when(valueOperations.get("stock:sku_" + skuId)).thenReturn(String.valueOf(redisStock));
    when(valueOperations.get("sold:sku_" + skuId)).thenReturn(String.valueOf(redisSold));
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(mysqlOrderCount);
    when(activityRepository.findActiveActivityBySkuId(skuId)).thenReturn(Optional.empty());

    // When
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Then
    assertFalse(result.passed(), "Reconciliation should fail");
    assertEquals(1, result.discrepancies().size(), "Should have 1 discrepancy");
    assertEquals("ORDER_COUNT_MISMATCH", result.discrepancies().get(0).type());

    // Verify reconciliation log was saved with discrepancy
    ArgumentCaptor<ReconciliationLog> logCaptor = ArgumentCaptor.forClass(ReconciliationLog.class);
    verify(reconciliationLogRepository).save(logCaptor.capture());
    ReconciliationLog savedLog = logCaptor.getValue();
    assertEquals(ReconciliationStatus.DISCREPANCY, savedLog.getStatus());
    assertEquals(1, savedLog.getDiscrepancyCount());
  }

  @Test
  @DisplayName("Should detect total stock mismatch")
  void testReconcile_TotalStockMismatch() {
    // Given
    String skuId = "SKU003";
    int redisStock = 100;
    int redisSold = 50;
    long mysqlOrderCount = 50L;
    int totalStock = 200; // Mismatch! Should be 150

    when(valueOperations.get("stock:sku_" + skuId)).thenReturn(String.valueOf(redisStock));
    when(valueOperations.get("sold:sku_" + skuId)).thenReturn(String.valueOf(redisSold));
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(mysqlOrderCount);

    SeckillActivity activity = new SeckillActivity();
    activity.setSkuId(skuId);
    activity.setTotalStock(totalStock);
    when(activityRepository.findActiveActivityBySkuId(skuId)).thenReturn(Optional.of(activity));

    // When
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Then
    assertFalse(result.passed(), "Reconciliation should fail");
    assertEquals(1, result.discrepancies().size(), "Should have 1 discrepancy");
    assertEquals("TOTAL_STOCK_MISMATCH", result.discrepancies().get(0).type());
  }

  @Test
  @DisplayName("Should detect multiple discrepancies")
  void testReconcile_MultipleDiscrepancies() {
    // Given
    String skuId = "SKU004";
    int redisStock = 100;
    int redisSold = 50;
    long mysqlOrderCount = 45L; // Mismatch 1
    int totalStock = 200; // Mismatch 2

    when(valueOperations.get("stock:sku_" + skuId)).thenReturn(String.valueOf(redisStock));
    when(valueOperations.get("sold:sku_" + skuId)).thenReturn(String.valueOf(redisSold));
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(mysqlOrderCount);

    SeckillActivity activity = new SeckillActivity();
    activity.setSkuId(skuId);
    activity.setTotalStock(totalStock);
    when(activityRepository.findActiveActivityBySkuId(skuId)).thenReturn(Optional.of(activity));

    // When
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Then
    assertFalse(result.passed(), "Reconciliation should fail");
    assertEquals(2, result.discrepancies().size(), "Should have 2 discrepancies");
  }

  @Test
  @DisplayName("Should handle null or blank skuId")
  void testReconcile_NullSkuId() {
    // When
    ReconciliationResult result1 = reconciliationService.reconcile(null);
    ReconciliationResult result2 = reconciliationService.reconcile("");
    ReconciliationResult result3 = reconciliationService.reconcile("   ");

    // Then
    assertFalse(result1.passed());
    assertFalse(result2.passed());
    assertFalse(result3.passed());
  }

  @Test
  @DisplayName("Should handle Redis connection failure gracefully")
  void testReconcile_RedisFailure() {
    // Given
    String skuId = "SKU005";
    when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);
    when(valueOperations.get(anyString()))
        .thenThrow(new RuntimeException("Redis connection failed"));
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(0L);

    // When
    ReconciliationResult result = reconciliationService.reconcile(skuId);

    // Then
    // The implementation handles Redis failures gracefully by returning 0 for Redis values
    // and still completing the reconciliation
    assertTrue(
        result.passed(),
        "Should handle Redis failure gracefully and pass when MySQL count is also 0");
    assertEquals(0, result.redisStock());
    assertEquals(0, result.redisSoldCount());
  }

  @Test
  @DisplayName("Should perform full reconciliation for all activities")
  void testFullReconcile_Success() {
    // Given
    SeckillActivity activity1 = new SeckillActivity();
    activity1.setSkuId("SKU001");
    activity1.setTotalStock(150);

    SeckillActivity activity2 = new SeckillActivity();
    activity2.setSkuId("SKU002");
    activity2.setTotalStock(200);

    when(activityRepository.findByStatus(ActivityStatus.IN_PROGRESS))
        .thenReturn(List.of(activity1));
    when(activityRepository.findByStatus(ActivityStatus.ENDED)).thenReturn(List.of(activity2));

    // Mock Redis and MySQL data for both SKUs
    when(valueOperations.get("stock:sku_SKU001")).thenReturn("100");
    when(valueOperations.get("sold:sku_SKU001")).thenReturn("50");
    when(orderRepository.countBySkuIdAndStatusIn(eq("SKU001"), anyList())).thenReturn(50L);
    when(activityRepository.findActiveActivityBySkuId("SKU001")).thenReturn(Optional.of(activity1));

    when(valueOperations.get("stock:sku_SKU002")).thenReturn("150");
    when(valueOperations.get("sold:sku_SKU002")).thenReturn("50");
    when(orderRepository.countBySkuIdAndStatusIn(eq("SKU002"), anyList())).thenReturn(50L);
    when(activityRepository.findActiveActivityBySkuId("SKU002")).thenReturn(Optional.of(activity2));

    // When
    ReconciliationReport report = reconciliationService.fullReconcile();

    // Then
    assertNotNull(report);
    assertEquals(2, report.totalSkus());
    assertEquals(2, report.passedSkus());
    assertEquals(0, report.failedSkus());
    assertTrue(report.allPassed());
  }

  @Test
  @DisplayName("Should generate report with failures")
  void testFullReconcile_WithFailures() {
    // Given
    SeckillActivity activity1 = new SeckillActivity();
    activity1.setSkuId("SKU001");
    activity1.setTotalStock(150);

    when(activityRepository.findByStatus(ActivityStatus.IN_PROGRESS))
        .thenReturn(List.of(activity1));
    when(activityRepository.findByStatus(ActivityStatus.ENDED)).thenReturn(List.of());

    // Mock data with discrepancy
    when(valueOperations.get("stock:sku_SKU001")).thenReturn("100");
    when(valueOperations.get("sold:sku_SKU001")).thenReturn("50");
    when(orderRepository.countBySkuIdAndStatusIn(eq("SKU001"), anyList()))
        .thenReturn(45L); // Mismatch
    when(activityRepository.findActiveActivityBySkuId("SKU001")).thenReturn(Optional.of(activity1));

    // When
    ReconciliationReport report = reconciliationService.fullReconcile();

    // Then
    assertNotNull(report);
    assertEquals(1, report.totalSkus());
    assertEquals(0, report.passedSkus());
    assertEquals(1, report.failedSkus());
    assertFalse(report.allPassed());
    assertEquals(1, report.getFailedResults().size());
  }

  @Test
  @DisplayName("Should return empty report when no activities found")
  void testFullReconcile_NoActivities() {
    // Given
    when(activityRepository.findByStatus(ActivityStatus.IN_PROGRESS)).thenReturn(List.of());
    when(activityRepository.findByStatus(ActivityStatus.ENDED)).thenReturn(List.of());

    // When
    ReconciliationReport report = reconciliationService.fullReconcile();

    // Then
    assertNotNull(report);
    assertEquals(0, report.totalSkus());
    assertTrue(report.allPassed());
  }

  @Test
  @DisplayName("Should handle fixDiscrepancy with null input")
  void testFixDiscrepancy_NullInput() {
    // When
    boolean result = reconciliationService.fixDiscrepancy(null);

    // Then
    assertFalse(result, "Should return false for null input");
  }

  @Test
  @DisplayName("Should attempt to fix order count mismatch")
  void testFixDiscrepancy_OrderCountMismatch() {
    // Given
    String skuId = "SKU001";
    Discrepancy discrepancy = Discrepancy.orderCountMismatch(skuId, 50, 45);

    // When
    boolean result = reconciliationService.fixDiscrepancy(discrepancy);

    // Then
    assertTrue(result, "Should successfully fix order count mismatch");
    verify(valueOperations).set("sold:sku_" + skuId, "45");
  }

  @Test
  @DisplayName("Should attempt to fix total stock mismatch")
  void testFixDiscrepancy_TotalStockMismatch() {
    // Given
    String skuId = "SKU002";
    Discrepancy discrepancy = Discrepancy.totalStockMismatch(skuId, 200, 100, 50);

    // Mock MySQL order count
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(50L);

    // When
    boolean result = reconciliationService.fixDiscrepancy(discrepancy);

    // Then
    assertTrue(result, "Should successfully fix total stock mismatch");
    verify(valueOperations).set("stock:sku_" + skuId, "150"); // 200 - 50
    verify(valueOperations).set("sold:sku_" + skuId, "50");
  }

  @Test
  @DisplayName("Should attempt to fix stock mismatch")
  void testFixDiscrepancy_StockMismatch() {
    // Given
    String skuId = "SKU003";
    Discrepancy discrepancy = Discrepancy.stockMismatch(skuId, 100, 95);

    // When
    boolean result = reconciliationService.fixDiscrepancy(discrepancy);

    // Then
    assertTrue(result, "Should successfully fix stock mismatch");
    verify(valueOperations).set("stock:sku_" + skuId, "100");
  }

  @Test
  @DisplayName("Should handle unknown discrepancy type")
  void testFixDiscrepancy_UnknownType() {
    // Given
    Discrepancy discrepancy = new Discrepancy("UNKNOWN_TYPE", "Unknown discrepancy", null);

    // When
    boolean result = reconciliationService.fixDiscrepancy(discrepancy);

    // Then
    assertFalse(result, "Should return false for unknown discrepancy type");
  }

  @Test
  @DisplayName("Should verify only valid order statuses are counted")
  void testReconcile_OnlyCountsValidOrderStatuses() {
    // Given
    String skuId = "SKU006";
    when(valueOperations.get("stock:sku_" + skuId)).thenReturn("100");
    when(valueOperations.get("sold:sku_" + skuId)).thenReturn("50");
    when(orderRepository.countBySkuIdAndStatusIn(eq(skuId), anyList())).thenReturn(50L);
    when(activityRepository.findActiveActivityBySkuId(skuId)).thenReturn(Optional.empty());

    // When
    reconciliationService.reconcile(skuId);

    // Then
    ArgumentCaptor<List<OrderStatus>> statusCaptor = ArgumentCaptor.forClass(List.class);
    verify(orderRepository).countBySkuIdAndStatusIn(eq(skuId), statusCaptor.capture());

    List<OrderStatus> capturedStatuses = statusCaptor.getValue();
    assertEquals(2, capturedStatuses.size());
    assertTrue(capturedStatuses.contains(OrderStatus.PENDING_PAYMENT));
    assertTrue(capturedStatuses.contains(OrderStatus.PAID));
  }
}
