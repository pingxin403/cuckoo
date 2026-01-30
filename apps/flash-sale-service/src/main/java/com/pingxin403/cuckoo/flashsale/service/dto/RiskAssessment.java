package com.pingxin403.cuckoo.flashsale.service.dto;

/**
 * 风险评估结果 Risk assessment result record.
 *
 * <p>Contains the outcome of a risk assessment for a seckill request, including the risk level,
 * recommended action, and reason for the assessment.
 *
 * <p>Validates Requirements: 3.4, 3.5, 3.6
 *
 * @param level the assessed risk level (LOW, MEDIUM, HIGH)
 * @param action the recommended action (PASS, CAPTCHA, BLOCK)
 * @param reason the reason for the assessment result
 */
public record RiskAssessment(RiskLevel level, RiskAction action, String reason) {

  /**
   * Factory method to create a LOW risk assessment (PASS action).
   *
   * @param reason the reason for the assessment
   * @return RiskAssessment with LOW level and PASS action
   */
  public static RiskAssessment pass(String reason) {
    return new RiskAssessment(RiskLevel.LOW, RiskAction.PASS, reason);
  }

  /**
   * Factory method to create a MEDIUM risk assessment (CAPTCHA action).
   *
   * @param reason the reason for the assessment
   * @return RiskAssessment with MEDIUM level and CAPTCHA action
   */
  public static RiskAssessment captcha(String reason) {
    return new RiskAssessment(RiskLevel.MEDIUM, RiskAction.CAPTCHA, reason);
  }

  /**
   * Factory method to create a HIGH risk assessment (BLOCK action).
   *
   * @param reason the reason for the assessment
   * @return RiskAssessment with HIGH level and BLOCK action
   */
  public static RiskAssessment block(String reason) {
    return new RiskAssessment(RiskLevel.HIGH, RiskAction.BLOCK, reason);
  }

  /**
   * Factory method to create a risk assessment from a risk level.
   *
   * <p>Automatically maps the risk level to the corresponding action:
   *
   * <ul>
   *   <li>LOW -> PASS
   *   <li>MEDIUM -> CAPTCHA
   *   <li>HIGH -> BLOCK
   * </ul>
   *
   * @param level the risk level
   * @param reason the reason for the assessment
   * @return RiskAssessment with the appropriate action
   */
  public static RiskAssessment fromLevel(RiskLevel level, String reason) {
    return new RiskAssessment(level, RiskAction.fromRiskLevel(level), reason);
  }

  /**
   * Check if the request should be allowed to proceed.
   *
   * @return true if action is PASS
   */
  public boolean shouldPass() {
    return action == RiskAction.PASS;
  }

  /**
   * Check if captcha verification is required.
   *
   * @return true if action is CAPTCHA
   */
  public boolean requiresCaptcha() {
    return action == RiskAction.CAPTCHA;
  }

  /**
   * Check if the request should be blocked.
   *
   * @return true if action is BLOCK
   */
  public boolean shouldBlock() {
    return action == RiskAction.BLOCK;
  }
}
