package com.pingxin403.cuckoo.flashsale.scheduled;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.pingxin403.cuckoo.flashsale.alert.AlertPublisher;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;

/**
 * Alert service for reconciliation discrepancies.
 *
 * <p>Handles alert notifications when reconciliation detects data inconsistencies.
 *
 * <p>Validates Requirements: 6.6, 7.2, 7.5
 */
@Service
public class ReconciliationAlertService {

  private static final Logger logger = LoggerFactory.getLogger(ReconciliationAlertService.class);

  private final AlertPublisher alertPublisher;

  public ReconciliationAlertService(AlertPublisher alertPublisher) {
    this.alertPublisher = alertPublisher;
  }

  @Value("${flash-sale.reconciliation.alert-threshold:5}")
  private int alertThreshold;

  @Value("${flash-sale.reconciliation.critical-threshold:20}")
  private int criticalThreshold;

  /**
   * Send alert notification for reconciliation discrepancies.
   *
   * <p>Validates: Requirement 6.6
   *
   * @param report the reconciliation report
   */
  public void sendDiscrepancyAlert(ReconciliationReport report) {
    logger.warn(
        "ALERT: Reconciliation discrepancies detected - failedSkus={}, totalDiscrepancies={}",
        report.failedSkus(),
        report.getTotalDiscrepancies());

    alertPublisher.publishWarning(
        "reconciliation_discrepancy_detected",
        "Reconciliation discrepancies detected",
        java.util.Map.of(
            "failedSkus", report.failedSkus(),
            "totalDiscrepancies", report.getTotalDiscrepancies(),
            "timestamp", report.timestamp().toString()));
  }

  /**
   * Check if discrepancies exceed critical threshold requiring activity pause.
   *
   * <p>Validates: Requirement 6.6
   *
   * @param report the reconciliation report
   * @return true if activities should be paused
   */
  public boolean shouldPauseActivity(ReconciliationReport report) {
    return report.getTotalDiscrepancies() >= criticalThreshold;
  }

  /**
   * Pause flash sale activities and notify operations team.
   *
   * <p>Validates: Requirement 6.6
   *
   * @param report the reconciliation report
   */
  public void pauseActivitiesAndNotify(ReconciliationReport report) {
    logger.error(
        "CRITICAL: Pausing flash sale activities due to excessive discrepancies - count={}",
        report.getTotalDiscrepancies());

    alertPublisher.publishCritical(
        "reconciliation_activity_pause_required",
        "Critical reconciliation discrepancy threshold reached",
        java.util.Map.of(
            "failedSkus", report.failedSkus(),
            "totalDiscrepancies", report.getTotalDiscrepancies(),
            "criticalThreshold", criticalThreshold,
            "timestamp", report.timestamp().toString()));

    logger.error("Operations team notified. Manual intervention required.");
  }

  /**
   * Send alert for reconciliation execution errors.
   *
   * <p>Validates: Requirement 7.5
   *
   * @param error the error that occurred
   */
  public void sendReconciliationErrorAlert(Exception error) {
    logger.error("ALERT: Reconciliation execution failed", error);

    alertPublisher.publishCritical(
        "reconciliation_execution_failed",
        error.getMessage(),
        java.util.Map.of("exceptionType", error.getClass().getName()));
  }
}
