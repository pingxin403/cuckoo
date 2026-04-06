package com.pingxin403.cuckoo.flashsale.scheduled;

import static org.mockito.ArgumentMatchers.anyMap;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;

import java.util.List;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import com.pingxin403.cuckoo.flashsale.alert.AlertPublisher;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationResult;

@ExtendWith(MockitoExtension.class)
class ReconciliationAlertServiceTest {

  @Mock private AlertPublisher alertPublisher;

  @Test
  @DisplayName("should publish warning for discrepancy alert")
  void shouldPublishWarningForDiscrepancyAlert() {
    ReconciliationAlertService service = new ReconciliationAlertService(alertPublisher);
    ReconciliationReport report =
        ReconciliationReport.from(List.of(ReconciliationResult.failure("SKU001", 100, 50, 45, List.of())));

    service.sendDiscrepancyAlert(report);

    verify(alertPublisher)
        .publishWarning(
            eq("reconciliation_discrepancy_detected"), eq("Reconciliation discrepancies detected"), anyMap());
  }

  @Test
  @DisplayName("should publish critical alert when pause is required")
  void shouldPublishCriticalAlertWhenPauseRequired() {
    ReconciliationAlertService service = new ReconciliationAlertService(alertPublisher);
    ReflectionTestUtils.setField(service, "criticalThreshold", 10);

    ReconciliationReport report =
        ReconciliationReport.from(
            List.of(
                ReconciliationResult.failure("SKU001", 100, 50, 45, List.of()),
                ReconciliationResult.failure("SKU002", 100, 50, 45, List.of())));

    service.pauseActivitiesAndNotify(report);

    verify(alertPublisher)
        .publishCritical(
            eq("reconciliation_activity_pause_required"),
            eq("Critical reconciliation discrepancy threshold reached"),
            anyMap());
  }

  @Test
  @DisplayName("should publish critical alert for reconciliation execution error")
  void shouldPublishCriticalAlertForReconciliationError() {
    ReconciliationAlertService service = new ReconciliationAlertService(alertPublisher);

    service.sendReconciliationErrorAlert(new RuntimeException("reconciliation failed"));

    verify(alertPublisher)
        .publishCritical(eq("reconciliation_execution_failed"), eq("reconciliation failed"), anyMap());
  }
}
