package com.pingxin403.cuckoo.flashsale.service.impl;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.HashOperations;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.data.redis.core.script.DefaultRedisScript;

import com.pingxin403.cuckoo.flashsale.service.dto.DeviceFingerprint;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAction;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskAssessment;
import com.pingxin403.cuckoo.flashsale.service.dto.RiskLevel;
import com.pingxin403.cuckoo.flashsale.service.dto.SeckillRequest;

/**
 * Unit tests for L2 rate limiting with Redis token bucket in AntiFraudServiceImpl.
 *
 * <p>Validates Requirements: 3.2, 7.4
 */
@ExtendWith(MockitoExtension.class)
@DisplayName("AntiFraudService L2 Rate Limiting Tests")
class AntiFraudServiceL2RateLimitTest {

  @Mock private StringRedisTemplate stringRedisTemplate;
  @Mock private DefaultRedisScript<Long> tokenBucketScript;
  @Mock private ValueOperations<String, String> valueOperations;
  @Mock private HashOperations<String, Object, Object> hashOperations;

  private AntiFraudServiceImpl antiFraudService;

  @BeforeEach
  void setUp() {
    lenient().when(stringRedisTemplate.opsForValue()).thenReturn(valueOperations);
    lenient().when(stringRedisTemplate.opsForHash()).thenReturn(hashOperations);
    antiFraudService = new AntiFraudServiceImpl(stringRedisTemplate, tokenBucketScript);
  }

  @Test
  @DisplayName("Should require captcha when L2 rate limit exceeded (no token available)")
  void shouldRequireCaptchaWhenL2RateLimitExceeded() {
    // Given
    DeviceFingerprint fingerprint =
        DeviceFingerprint.builder()
            .deviceId("device123")
            .platform("Windows")
            .browserName("Chrome")
            .build();

    SeckillRequest request =
        SeckillRequest.builder()
            .userId("user123")
            .skuId("sku123")
            .quantity(1)
            .deviceFingerprint(fingerprint)
            .ipAddress("192.168.1.1")
            .source("WEB")
            .build();

    // When: Token bucket returns 0 (no token available)
    when(stringRedisTemplate.execute(
            eq(tokenBucketScript), anyList(), anyString(), anyString(), anyString()))
        .thenReturn(0L);

    // Then
    RiskAssessment result = antiFraudService.assess(request);

    assertEquals(RiskLevel.MEDIUM, result.level());
    assertEquals(RiskAction.CAPTCHA, result.action());
    assertTrue(result.requiresCaptcha());
    assertEquals("请求频率超过阈值，请完成验证", result.reason());

    verify(stringRedisTemplate)
        .execute(eq(tokenBucketScript), anyList(), eq("100"), eq("10"), anyString());
  }

  @Test
  @DisplayName("Should pass when L2 rate limit not exceeded (token acquired)")
  void shouldPassWhenL2RateLimitNotExceeded() {
    // Given
    DeviceFingerprint fingerprint =
        DeviceFingerprint.builder()
            .deviceId("device123")
            .platform("Windows")
            .browserName("Chrome")
            .canvasFingerprint("canvas123")
            .webglFingerprint("webgl123")
            .build();

    SeckillRequest request =
        SeckillRequest.builder()
            .userId("user123")
            .skuId("sku123")
            .quantity(1)
            .deviceFingerprint(fingerprint)
            .ipAddress("192.168.1.1")
            .source("WEB")
            .build();

    // When: Token bucket returns 1 (token acquired)
    when(stringRedisTemplate.execute(
            eq(tokenBucketScript), anyList(), anyString(), anyString(), anyString()))
        .thenReturn(1L);
    when(hashOperations.get(anyString(), eq("score"))).thenReturn("10");
    when(valueOperations.increment(anyString())).thenReturn(5L);

    // Then
    RiskAssessment result = antiFraudService.assess(request);

    assertEquals(RiskLevel.LOW, result.level());
    assertEquals(RiskAction.PASS, result.action());
    assertFalse(result.shouldBlock());
    assertFalse(result.requiresCaptcha());
  }

  @Test
  @DisplayName("Should update token bucket capacity threshold dynamically")
  void shouldUpdateTokenBucketCapacity() {
    // When
    antiFraudService.updateRateLimitThreshold("token_bucket_capacity", 200);

    // Then: New capacity should be used
    DeviceFingerprint fingerprint = DeviceFingerprint.builder().deviceId("device123").build();

    SeckillRequest request =
        SeckillRequest.builder()
            .userId("user123")
            .skuId("sku123")
            .deviceFingerprint(fingerprint)
            .build();

    when(stringRedisTemplate.execute(
            eq(tokenBucketScript), anyList(), anyString(), anyString(), anyString()))
        .thenReturn(1L);
    when(hashOperations.get(anyString(), eq("score"))).thenReturn("10");
    when(valueOperations.increment(anyString())).thenReturn(5L);

    antiFraudService.assess(request);

    verify(stringRedisTemplate)
        .execute(eq(tokenBucketScript), anyList(), eq("200"), anyString(), anyString());
  }

  @Test
  @DisplayName("Should update token refill rate threshold dynamically")
  void shouldUpdateTokenRefillRate() {
    // When
    antiFraudService.updateRateLimitThreshold("token_refill_rate", 20);

    // Then: New refill rate should be used
    DeviceFingerprint fingerprint = DeviceFingerprint.builder().deviceId("device123").build();

    SeckillRequest request =
        SeckillRequest.builder()
            .userId("user123")
            .skuId("sku123")
            .deviceFingerprint(fingerprint)
            .build();

    when(stringRedisTemplate.execute(
            eq(tokenBucketScript), anyList(), anyString(), anyString(), anyString()))
        .thenReturn(1L);
    when(hashOperations.get(anyString(), eq("score"))).thenReturn("10");
    when(valueOperations.increment(anyString())).thenReturn(5L);

    antiFraudService.assess(request);

    verify(stringRedisTemplate)
        .execute(eq(tokenBucketScript), anyList(), anyString(), eq("20"), anyString());
  }
}
