package com.pingxin403.cuckoo.flashsale.scheduled;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.service.ReconciliationService;
import com.pingxin403.cuckoo.flashsale.service.dto.ReconciliationReport;

/**
 * Scheduled task for periodic reconciliation.
 *
 * <p>Executes full reconciliation every hour to ensure data consistency between Redis stock data
 * and MySQL order data.
 *
 * <p>Validates Requirements: 6.4, 6.5, 6.6
 *
 * <p>Configuration:
 *
 * <ul>
 *   <li>Execution frequency: Every hour (configurable via cron expression)
 *   <li>Alert threshold: Configurable via application properties
 *   <li>Auto-pause threshold: Configurable via application properties
 * </ul>
 */
@Component
public class ReconciliationScheduledTask {

  private static final Logger logger = LoggerFactory.getLogger(ReconciliationScheduledTask.class);

  private final ReconciliationService reconciliationService;
  private final ReconciliationAlertService alertService;

  /**
   * Constructor with dependency injection.
   *
   * @param reconciliationService the reconciliation service
   * @param alertService the alert service for notifications
   */
  public ReconciliationScheduledTask(
      ReconciliationService reconciliationService, ReconciliationAlertService alertService) {
    this.reconciliationService = reconciliationService;
    this.alertService = alertService;
  }

  /**
   * Execute hourly reconciliation task.
   *
   * <p>Runs every hour at the top of the hour (configurable via cron expression in
   * application.yml).
   *
   * <p>Validates: Requirements 6.4, 6.5, 6.6
   */
  @Scheduled(cron = "${flash-sale.reconciliation.cron:0 0 * * * ?}")
  public void executeHourlyReconciliation() {
    logger.info("Starting scheduled hourly reconciliation");

    try {
      // Execute full reconciliation
      ReconciliationReport report = reconciliationService.fullReconcile();

      // Log results
      logger.info(
          "Hourly reconciliation completed: total={}, passed={}, failed={}, discrepancies={}",
          report.totalSkus(),
          report.passedSkus(),
          report.failedSkus(),
          report.getTotalDiscrepancies());

      // Check if alert should be triggered
      if (!report.allPassed()) {
        logger.warn(
            "Reconciliation found discrepancies: failedSkus={}, totalDiscrepancies={}",
            report.failedSkus(),
            report.getTotalDiscrepancies());

        // Send alert notification
        alertService.sendDiscrepancyAlert(report);

        // Check if discrepancies exceed critical threshold
        if (alertService.shouldPauseActivity(report)) {
          logger.error(
              "Discrepancies exceed critical threshold! Pausing flash sale activities and"
                  + " notifying operations team.");
          alertService.pauseActivitiesAndNotify(report);
        }
      } else {
        logger.info("All reconciliations passed successfully");
      }

    } catch (Exception e) {
      logger.error("Error during scheduled reconciliation", e);
      // Send error alert
      alertService.sendReconciliationErrorAlert(e);
    }
  }

  /**
   * Execute manual reconciliation (for testing or on-demand execution).
   *
   * <p>This method can be triggered manually via JMX or REST API for immediate reconciliation.
   *
   * @return the reconciliation report
   */
  public ReconciliationReport executeManualReconciliation() {
    logger.info("Starting manual reconciliation");

    try {
      ReconciliationReport report = reconciliationService.fullReconcile();

      logger.info(
          "Manual reconciliation completed: total={}, passed={}, failed={}",
          report.totalSkus(),
          report.passedSkus(),
          report.failedSkus());

      return report;

    } catch (Exception e) {
      logger.error("Error during manual reconciliation", e);
      throw new RuntimeException("Manual reconciliation failed", e);
    }
  }
}
