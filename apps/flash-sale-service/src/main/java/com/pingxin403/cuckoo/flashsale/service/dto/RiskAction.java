package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 风险动作枚举 Risk action enumeration for anti-fraud response.
 *
 * <p>Defines the action to take based on risk assessment:
 *
 * <ul>
 *   <li>PASS - Allow request to proceed
 *   <li>CAPTCHA - Require captcha verification before proceeding
 *   <li>BLOCK - Block the request immediately
 * </ul>
 *
 * <p>Validates Requirements: 3.4, 3.5, 3.6
 */
public enum RiskAction {
  /** 允许通过 - Allow request to proceed */
  PASS(0, "允许通过"),

  /** 需要验证码 - Require captcha verification */
  CAPTCHA(1, "需要验证码"),

  /** 直接拒绝 - Block the request */
  BLOCK(2, "直接拒绝");

  private final int code;
  private final String description;

  RiskAction(int code, String description) {
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
   * Get RiskAction from code value.
   *
   * @param code the code value
   * @return the corresponding RiskAction
   * @throws IllegalArgumentException if code is invalid
   */
  public static RiskAction fromCode(int code) {
    for (RiskAction action : values()) {
      if (action.code == code) {
        return action;
      }
    }
    throw new IllegalArgumentException("Invalid RiskAction code: " + code);
  }

  /**
   * Map RiskLevel to corresponding RiskAction.
   *
   * <p>Mapping rules:
   *
   * <ul>
   *   <li>LOW -> PASS
   *   <li>MEDIUM -> CAPTCHA
   *   <li>HIGH -> BLOCK
   * </ul>
   *
   * @param level the risk level
   * @return the corresponding risk action
   */
  public static RiskAction fromRiskLevel(RiskLevel level) {
    return switch (level) {
      case LOW -> PASS;
      case MEDIUM -> CAPTCHA;
      case HIGH -> BLOCK;
    };
  }
}
