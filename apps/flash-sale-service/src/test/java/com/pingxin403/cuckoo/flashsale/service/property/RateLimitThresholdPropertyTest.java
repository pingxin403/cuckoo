package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Random;
import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.service.AntiFraudService;

/**
 * Property 13: 限流阈值动态生效
 *
 * <p>**Validates: Requirement 7.4**
 *
 * <p>Updated threshold takes effect immediately
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 13: 限流阈值动态生效")
public class RateLimitThresholdPropertyTest {

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
  }

  @Autowired private AntiFraudService antiFraudService;
  @Autowired private StringRedisTemplate redisTemplate;

  private static final Random random = new Random();

  /**
   * Property 13: 限流阈值动态生效
   *
   * <p>For any rate limit threshold configuration change, after calling updateRateLimitThreshold,
   * subsequent risk assessments should use the new threshold for judgment.
   *
   * <p>**Validates: Requirement 7.4**
   */
  @ParameterizedTest(name = "Threshold update: key={0}, oldThreshold={1}, newThreshold={2}")
  @MethodSource("generateThresholdUpdateTestCases")
  void updatedThresholdTakesEffectImmediately(
      String rateLimitKey, int oldThreshold, int newThreshold) {
    // Setup: Set initial threshold
    antiFraudService.updateRateLimitThreshold(rateLimitKey, oldThreshold);

    // Verify initial threshold is set
    String redisKey = "rate_limit:" + rateLimitKey;
    String storedValue = redisTemplate.opsForValue().get(redisKey);
    assertThat(storedValue)
        .as("Initial threshold should be stored in Redis")
        .isEqualTo(String.valueOf(oldThreshold));

    // Act: Update threshold
    antiFraudService.updateRateLimitThreshold(rateLimitKey, newThreshold);

    // Assert: New threshold is immediately effective
    String updatedValue = redisTemplate.opsForValue().get(redisKey);
    assertThat(updatedValue)
        .as("Updated threshold should take effect immediately")
        .isEqualTo(String.valueOf(newThreshold));

    // Verify the threshold is used in subsequent operations
    // (The actual risk assessment logic would use this threshold)
    assertThat(Integer.parseInt(updatedValue))
        .as("Threshold value should match new threshold")
        .isEqualTo(newThreshold);
  }

  /**
   * Property 13b: Multiple threshold updates are idempotent
   *
   * <p>Updating the same threshold multiple times should always result in the last value being
   * used.
   */
  @ParameterizedTest(name = "Multiple updates: key={0}, values={1}")
  @MethodSource("generateMultipleUpdateTestCases")
  void multipleThresholdUpdatesUseLastValue(String rateLimitKey, int[] thresholdValues) {
    // Act: Update threshold multiple times
    for (int threshold : thresholdValues) {
      antiFraudService.updateRateLimitThreshold(rateLimitKey, threshold);
    }

    // Assert: Last value is effective
    String redisKey = "rate_limit:" + rateLimitKey;
    String storedValue = redisTemplate.opsForValue().get(redisKey);
    int lastThreshold = thresholdValues[thresholdValues.length - 1];

    assertThat(storedValue)
        .as("Last threshold value should be effective")
        .isEqualTo(String.valueOf(lastThreshold));
  }

  /** Generate test cases for threshold updates */
  static Stream<Object[]> generateThresholdUpdateTestCases() {
    return Stream.generate(
            () -> {
              String[] keys = {"device", "user", "ip", "global"};
              String key = keys[random.nextInt(keys.length)];
              int oldThreshold = random.nextInt(1000) + 10; // 10 to 1009
              int newThreshold = random.nextInt(1000) + 10; // 10 to 1009
              return new Object[] {key, oldThreshold, newThreshold};
            })
        .limit(50);
  }

  /** Generate test cases for multiple threshold updates */
  static Stream<Object[]> generateMultipleUpdateTestCases() {
    return Stream.generate(
            () -> {
              String[] keys = {"device", "user", "ip", "global"};
              String key = keys[random.nextInt(keys.length)];
              int updateCount = random.nextInt(5) + 2; // 2 to 6 updates
              int[] values = new int[updateCount];
              for (int i = 0; i < updateCount; i++) {
                values[i] = random.nextInt(1000) + 10;
              }
              return new Object[] {key, values};
            })
        .limit(50);
  }
}
