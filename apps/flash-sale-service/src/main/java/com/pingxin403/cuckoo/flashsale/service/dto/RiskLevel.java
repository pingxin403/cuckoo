package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 风险等级枚举 Risk level enumeration for anti-fraud assessment.
 *
 * <p>Defines three risk levels based on user behavior and device fingerprint analysis:
 *
 * <ul>
 *   <li>LOW - Normal user, passes without additional verification
 *   <li>MEDIUM - Suspicious user, requires captcha verification
 *   <li>HIGH - High-risk user, request is blocked
 * </ul>
 *
 * <p>Validates Requirements: 3.4, 3.5, 3.6
 */
public enum RiskLevel {
  /** 正常用户，无感通过 - Normal user, passes without additional verification */
  LOW(0, "低风险"),

  /** 可疑用户，需要验证码 - Suspicious user, requires captcha verification */
  MEDIUM(1, "中风险"),

  /** 高风险用户，直接拒绝 - High-risk user, request is blocked */
  HIGH(2, "高风险");

  private final int code;
  private final String description;

  RiskLevel(int code, String description) {
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
   * Get RiskLevel from code value.
   *
   * @param code the code value
   * @return the corresponding RiskLevel
   * @throws IllegalArgumentException if code is invalid
   */
  public static RiskLevel fromCode(int code) {
    for (RiskLevel level : values()) {
      if (level.code == code) {
        return level;
      }
    }
    throw new IllegalArgumentException("Invalid RiskLevel code: " + code);
  }
}
