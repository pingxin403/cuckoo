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

import com.pingxin403.cuckoo.flashsale.service.QueueService;
import com.pingxin403.cuckoo.flashsale.service.dto.QueueResult;

/**
 * Property 9: 令牌桶流量控制
 *
 * <p>**Validates: Requirements 4.1, 4.4**
 *
 * <p>Token available → code 200, no token → code 202
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 9: 令牌桶流量控制")
public class TokenBucketFlowControlPropertyTest {

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
  }

  @Autowired private QueueService queueService;
  @Autowired private StringRedisTemplate redisTemplate;

  private static final Random random = new Random();

  /**
   * Property 9a: Token available returns code 200
   *
   * <p>When tokens are available in the bucket, tryAcquireToken should return code 200
   *
   * <p>**Validates: Requirements 4.1, 4.4**
   */
  @ParameterizedTest(name = "Token available: userId={0}, skuId={1}, availableTokens={2}")
  @MethodSource("generateTokenAvailableTestCases")
  void tokenAvailableReturnsCode200(String userId, String skuId, int availableTokens) {
    cleanupSku(skuId);

    // Setup: Set available tokens in Redis
    String tokenKey = "token_bucket:" + skuId;
    redisTemplate.opsForValue().set(tokenKey, String.valueOf(availableTokens));

    // Act: Try to acquire token
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Assert: Should get token (code 200)
    assertThat(result.code())
        .as("When %d tokens available, should return code 200", availableTokens)
        .isEqualTo(200);
    assertThat(result.message()).contains("获得令牌");
    assertThat(result.estimatedWait()).isEqualTo(0);
    assertThat(result.queueToken()).isNotNull();
  }

  /**
   * Property 9b: No token available returns code 202
   *
   * <p>When no tokens are available in the bucket, tryAcquireToken should return code 202 (queuing)
   *
   * <p>**Validates: Requirements 4.1, 4.4**
   */
  @ParameterizedTest(name = "No token: userId={0}, skuId={1}")
  @MethodSource("generateNoTokenTestCases")
  void noTokenAvailableReturnsCode202(String userId, String skuId) {
    cleanupSku(skuId);

    // Setup: Set zero tokens in Redis
    String tokenKey = "token_bucket:" + skuId;
    redisTemplate.opsForValue().set(tokenKey, "0");

    // Act: Try to acquire token
    QueueResult result = queueService.tryAcquireToken(userId, skuId);

    // Assert: Should be queuing (code 202)
    assertThat(result.code()).as("When no tokens available, should return code 202").isEqualTo(202);
    assertThat(result.message()).contains("排队");
    assertThat(result.estimatedWait()).isGreaterThanOrEqualTo(0);
  }

  /** Generate test cases with available tokens */
  static Stream<Object[]> generateTokenAvailableTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  "USER-" + random.nextInt(10000),
                  "SKU-TOKEN-" + random.nextInt(1000),
                  random.nextInt(1000) + 1 // 1 to 1000 tokens
                })
        .limit(50);
  }

  /** Generate test cases with no tokens */
  static Stream<Object[]> generateNoTokenTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  "USER-" + random.nextInt(10000), "SKU-NOTOKEN-" + random.nextInt(1000)
                })
        .limit(50);
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("token_bucket:" + skuId);
    redisTemplate.delete("token_bucket_rate:" + skuId);
    redisTemplate.delete("token_bucket_last:" + skuId);
    redisTemplate.delete("sold_out:" + skuId);
  }
}
