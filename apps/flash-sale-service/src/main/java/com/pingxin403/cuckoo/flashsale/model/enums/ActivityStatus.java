package com.pingxin403.cuckoo.flashsale.model.enums;

/**
 * 秒杀活动状态枚举 Activity status enumeration for flash sale activities.
 *
 * <p>Maps to the status field in seckill_activity table: - 0: NOT_STARTED (未开始) - 1: IN_PROGRESS
 * (进行中) - 2: ENDED (已结束)
 */
public enum ActivityStatus {
  /** 活动未开始 - Activity has not started yet */
  NOT_STARTED(0, "未开始"),

  /** 活动进行中 - Activity is currently in progress */
  IN_PROGRESS(1, "进行中"),

  /** 活动已结束 - Activity has ended */
  ENDED(2, "已结束");

  private final int code;
  private final String description;

  ActivityStatus(int code, String description) {
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
   * Get ActivityStatus from database code value.
   *
   * @param code the database code value
   * @return the corresponding ActivityStatus
   * @throws IllegalArgumentException if code is invalid
   */
  public static ActivityStatus fromCode(int code) {
    for (ActivityStatus status : values()) {
      if (status.code == code) {
        return status;
      }
    }
    throw new IllegalArgumentException("Invalid ActivityStatus code: " + code);
  }
}
