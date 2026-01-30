package com.pingxin403.cuckoo.flashsale.model.enums;

/**
 * 秒杀订单状态枚举 Order status enumeration for flash sale orders.
 *
 * <p>Maps to the status field in seckill_order table: - 0: PENDING_PAYMENT (待支付) - 1: PAID (已支付) -
 * 2: CANCELLED (已取消) - 3: TIMEOUT (超时)
 */
public enum OrderStatus {
  /** 待支付 - Order is pending payment */
  PENDING_PAYMENT(0, "待支付"),

  /** 已支付 - Order has been paid */
  PAID(1, "已支付"),

  /** 已取消 - Order has been cancelled */
  CANCELLED(2, "已取消"),

  /** 超时 - Order has timed out */
  TIMEOUT(3, "超时");

  private final int code;
  private final String description;

  OrderStatus(int code, String description) {
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
   * Get OrderStatus from database code value.
   *
   * @param code the database code value
   * @return the corresponding OrderStatus
   * @throws IllegalArgumentException if code is invalid
   */
  public static OrderStatus fromCode(int code) {
    for (OrderStatus status : values()) {
      if (status.code == code) {
        return status;
      }
    }
    throw new IllegalArgumentException("Invalid OrderStatus code: " + code);
  }
}
