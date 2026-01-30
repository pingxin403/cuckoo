package com.pingxin403.cuckoo.flashsale.model.enums;

/**
 * 库存操作类型枚举 Stock operation type enumeration for stock log entries.
 *
 * <p>Maps to the operation field in stock_log table: - 1: DEDUCT (扣减) - 2: ROLLBACK (回滚)
 */
public enum StockOperation {
  /** 库存扣减 - Stock deduction operation */
  DEDUCT(1, "扣减"),

  /** 库存回滚 - Stock rollback operation */
  ROLLBACK(2, "回滚");

  private final int code;
  private final String description;

  StockOperation(int code, String description) {
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
   * Get StockOperation from database code value.
   *
   * @param code the database code value
   * @return the corresponding StockOperation
   * @throws IllegalArgumentException if code is invalid
   */
  public static StockOperation fromCode(int code) {
    for (StockOperation operation : values()) {
      if (operation.code == code) {
        return operation;
      }
    }
    throw new IllegalArgumentException("Invalid StockOperation code: " + code);
  }
}
