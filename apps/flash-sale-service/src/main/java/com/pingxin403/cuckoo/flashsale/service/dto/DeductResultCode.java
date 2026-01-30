package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 库存扣减结果状态码枚举 Result code enumeration for stock deduction operations.
 *
 * <p>Indicates the outcome of a stock deduction attempt:
 *
 * <ul>
 *   <li>SUCCESS - Stock deduction completed successfully
 *   <li>OUT_OF_STOCK - Insufficient stock available
 *   <li>SYSTEM_ERROR - System error occurred (e.g., Redis failure)
 * </ul>
 */
public enum DeductResultCode {
  /** 扣减成功 - Stock deduction completed successfully */
  SUCCESS("扣减成功"),

  /** 库存不足 - Insufficient stock available */
  OUT_OF_STOCK("库存不足"),

  /** 系统错误 - System error occurred */
  SYSTEM_ERROR("系统错误");

  private final String description;

  DeductResultCode(String description) {
    this.description = description;
  }

  public String getDescription() {
    return description;
  }
}
