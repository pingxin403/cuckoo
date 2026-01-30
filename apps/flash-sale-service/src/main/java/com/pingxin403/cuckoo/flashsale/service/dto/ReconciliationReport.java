package com.pingxin403.cuckoo.flashsale.service.dto;

import java.time.LocalDateTime;
import java.util.List;

/**
 * 全量对账报告 Full reconciliation report for all SKUs.
 *
 * <p>Contains the results of a full reconciliation run across all active SKUs.
 *
 * @param timestamp when the reconciliation was performed
 * @param totalSkus total number of SKUs checked
 * @param passedSkus number of SKUs that passed reconciliation
 * @param failedSkus number of SKUs with discrepancies
 * @param results list of individual SKU reconciliation results
 */
public record ReconciliationReport(
    LocalDateTime timestamp,
    int totalSkus,
    int passedSkus,
    int failedSkus,
    List<ReconciliationResult> results) {

  /**
   * Create a reconciliation report from a list of results.
   *
   * @param results list of individual SKU reconciliation results
   * @return ReconciliationReport
   */
  public static ReconciliationReport from(List<ReconciliationResult> results) {
    int total = results.size();
    int passed = (int) results.stream().filter(ReconciliationResult::passed).count();
    int failed = total - passed;

    return new ReconciliationReport(LocalDateTime.now(), total, passed, failed, results);
  }

  /**
   * Check if the overall reconciliation passed (no discrepancies).
   *
   * @return true if all SKUs passed reconciliation
   */
  public boolean allPassed() {
    return failedSkus == 0;
  }

  /**
   * Get the list of failed reconciliation results.
   *
   * @return list of results with discrepancies
   */
  public List<ReconciliationResult> getFailedResults() {
    return results.stream().filter(r -> !r.passed()).toList();
  }

  /**
   * Get the total number of discrepancies across all SKUs.
   *
   * @return total discrepancy count
   */
  public int getTotalDiscrepancies() {
    return results.stream().mapToInt(ReconciliationResult::getDiscrepancyCount).sum();
  }
}
