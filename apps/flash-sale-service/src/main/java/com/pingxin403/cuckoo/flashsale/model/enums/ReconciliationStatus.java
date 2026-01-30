package com.pingxin403.cuckoo.flashsale.model.enums;

/**
 * 对账状态枚举 Reconciliation status enumeration for reconciliation log entries.
 *
 * <p>Maps to the status field in reconciliation_log table: - 0: NORMAL (正常) - 1: DISCREPANCY (有差异)
 * - 2: FIXED (已修复)
 */
public enum ReconciliationStatus {
  /** 正常 - No discrepancy found */
  NORMAL(0, "正常"),

  /** 有差异 - Discrepancy detected */
  DISCREPANCY(1, "有差异"),

  /** 已修复 - Discrepancy has been fixed */
  FIXED(2, "已修复");

  private final int code;
  private final String description;

  ReconciliationStatus(int code, String description) {
    this.code = code;
    this.description = description;
  }

  public int getCode() {
    return code;
  }

  public String getDescription() {
    return description;
  }

  /**
   * Get ReconciliationStatus from database code value.
   *
   * @param code the database code value
   * @return the corresponding ReconciliationStatus
   * @throws IllegalArgumentException if code is invalid
   */
  public static ReconciliationStatus fromCode(int code) {
    for (ReconciliationStatus status : values()) {
      if (status.code == code) {
        return status;
      }
    }
    throw new IllegalArgumentException("Invalid ReconciliationStatus code: " + code);
  }
}
