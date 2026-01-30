package com.pingxin403.cuckoo.flashsale.scheduled;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import java.util.List;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import com.pingxin403.cuckoo.flashsale.service.ReconciliationService;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

/**
 * Unit tests for ReconciliationScheduledTask.
 *
 * <p>Validates Requirements: 6.4, 6.5, 6.6
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("ReconciliationScheduledTask Tests")
class ReconciliationScheduledTaskTest {

  @Mock private ReconciliationService reconciliationService;

  @Mock private ReconciliationAlertService alertService;

  private ReconciliationScheduledTask scheduledTask;

  @BeforeEach
  void setUp() {
    scheduledTask = new ReconciliationScheduledTask(reconciliationService, alertService);
  }

  @Test
  @DisplayName("Should execute hourly reconciliation successfully when all pass")
  void testExecuteHourlyReconciliation_AllPass() {
    // Given
    ReconciliationResult result1 = ReconciliationResult.success("SKU001", 100, 50, 50);
    ReconciliationResult result2 = ReconciliationResult.success("SKU002", 200, 100, 100);
    ReconciliationReport report = ReconciliationReport.from(List.of(result1, result2));

    when(reconciliationService.fullReconcile()).thenReturn(report);

    // When
    scheduledTask.executeHourlyReconciliation();

    // Then
    verify(reconciliationService).fullReconcile();
    verify(alertService, never()).sendDiscrepancyAlert(any());
    verify(alertService, never()).shouldPauseActivity(any());
  }

  @Test
  @DisplayName("Should send alert when discrepancies detected")
  void testExecuteHourlyReconciliation_WithDiscrepancies() {
    // Given
    ReconciliationResult result1 = ReconciliationResult.success("SKU001", 100, 50, 50);
    ReconciliationResult result2 = ReconciliationResult.failure("SKU002", 200, 100, 95, List.of());
    ReconciliationReport report = ReconciliationReport.from(List.of(result1, result2));

    when(reconciliationService.fullReconcile()).thenReturn(report);
    when(alertService.shouldPauseActivity(report)).thenReturn(false);

    // When
    scheduledTask.executeHourlyReconciliation();

    // Then
    verify(reconciliationService).fullReconcile();
    verify(alertService).sendDiscrepancyAlert(report);
    verify(alertService).shouldPauseActivity(report);
    verify(alertService, never()).pauseActivitiesAndNotify(any());
  }

  @Test
  @DisplayName("Should pause activities when critical threshold exceeded")
  void testExecuteHourlyReconciliation_CriticalThreshold() {
    // Given
    ReconciliationResult result1 = ReconciliationResult.failure("SKU001", 100, 50, 45, List.of());
    ReconciliationResult result2 = ReconciliationResult.failure("SKU002", 200, 100, 95, List.of());
    ReconciliationReport report = ReconciliationReport.from(List.of(result1, result2));

    when(reconciliationService.fullReconcile()).thenReturn(report);
    when(alertService.shouldPauseActivity(report)).thenReturn(true);

    // When
    scheduledTask.executeHourlyReconciliation();

    // Then
    verify(reconciliationService).fullReconcile();
    verify(alertService).sendDiscrepancyAlert(report);
    verify(alertService).shouldPauseActivity(report);
    verify(alertService).pauseActivitiesAndNotify(report);
  }

  @Test
  @DisplayName("Should handle reconciliation errors gracefully")
  void testExecuteHourlyReconciliation_Error() {
    // Given
    RuntimeException error = new RuntimeException("Reconciliation failed");
    when(reconciliationService.fullReconcile()).thenThrow(error);

    // When
    scheduledTask.executeHourlyReconciliation();

    // Then
    verify(reconciliationService).fullReconcile();
    verify(alertService).sendReconciliationErrorAlert(error);
  }

  @Test
  @DisplayName("Should execute manual reconciliation successfully")
  void testExecuteManualReconciliation_Success() {
    // Given
    ReconciliationResult result1 = ReconciliationResult.success("SKU001", 100, 50, 50);
    ReconciliationReport report = ReconciliationReport.from(List.of(result1));

    when(reconciliationService.fullReconcile()).thenReturn(report);

    // When
    ReconciliationReport result = scheduledTask.executeManualReconciliation();

    // Then
    assertNotNull(result);
    assertEquals(1, result.totalSkus());
    verify(reconciliationService).fullReconcile();
  }

  @Test
  @DisplayName("Should throw exception on manual reconciliation error")
  void testExecuteManualReconciliation_Error() {
    // Given
    when(reconciliationService.fullReconcile())
        .thenThrow(new RuntimeException("Reconciliation failed"));

    // When/Then
    assertThrows(RuntimeException.class, () -> scheduledTask.executeManualReconciliation());
  }
}
