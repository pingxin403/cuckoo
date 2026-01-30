package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;

import com.pingxin403.cuckoo.flashsale.service.dto.RiskAction;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskLevel;

/**
 * Property 8: 风险等级动作映射
 *
 * <p>**Validates: Requirements 3.4, 3.5, 3.6**
 *
 * <p>Risk level correctly maps to action (LOW→PASS, MEDIUM→CAPTCHA, HIGH→BLOCK)
 */
@Tag("Feature: flash-sale-system, Property 8: 风险等级动作映射")
public class RiskLevelActionMappingPropertyTest {

  /**
   * Property 8: 风险等级动作映射
   *
   * <p>For any risk assessment request:
   *
   * <ul>
   *   <li>When risk level is LOW, the action should be PASS
   *   <li>When risk level is MEDIUM, the action should be CAPTCHA
   *   <li>When risk level is HIGH, the action should be BLOCK
   * </ul>
   *
   * <p>**Validates: Requirements 3.4, 3.5, 3.6**
   */
  @ParameterizedTest(name = "Risk mapping: level={0}, expectedAction={1}")
  @MethodSource("generateRiskLevelMappingTestCases")
  void riskLevelCorrectlyMapsToAction(RiskLevel level, RiskAction expectedAction) {
    // Act: Create risk assessment from level
    RiskAssessment assessment = RiskAssessment.fromLevel(level, "Test reason");

    // Assert: Level and action match expected mapping
    assertThat(assessment.level()).isEqualTo(level);
    assertThat(assessment.action())
        .as("Risk level %s should map to action %s", level, expectedAction)
        .isEqualTo(expectedAction);

    // Verify using RiskAction.fromRiskLevel
    RiskAction mappedAction = RiskAction.fromRiskLevel(level);
    assertThat(mappedAction)
        .as("RiskAction.fromRiskLevel(%s) should return %s", level, expectedAction)
        .isEqualTo(expectedAction);

    // Verify helper methods
    switch (level) {
      case LOW:
        assertThat(assessment.shouldPass()).isTrue();
        assertThat(assessment.requiresCaptcha()).isFalse();
        assertThat(assessment.shouldBlock()).isFalse();
        break;
      case MEDIUM:
        assertThat(assessment.shouldPass()).isFalse();
        assertThat(assessment.requiresCaptcha()).isTrue();
        assertThat(assessment.shouldBlock()).isFalse();
        break;
      case HIGH:
        assertThat(assessment.shouldPass()).isFalse();
        assertThat(assessment.requiresCaptcha()).isFalse();
        assertThat(assessment.shouldBlock()).isTrue();
        break;
    }
  }

  /** Generate test cases covering all risk level to action mappings multiple times */
  static Stream<Object[]> generateRiskLevelMappingTestCases() {
    return Stream.of(
            // LOW -> PASS (repeated 34 times)
            Stream.generate(() -> new Object[] {RiskLevel.LOW, RiskAction.PASS}).limit(34),
            // MEDIUM -> CAPTCHA (repeated 33 times)
            Stream.generate(() -> new Object[] {RiskLevel.MEDIUM, RiskAction.CAPTCHA}).limit(33),
            // HIGH -> BLOCK (repeated 33 times)
            Stream.generate(() -> new Object[] {RiskLevel.HIGH, RiskAction.BLOCK}).limit(33))
        .flatMap(s -> s);
  }
}
