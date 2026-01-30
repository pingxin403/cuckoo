package com.pingxin403.cuckoo.flashsale.util;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.util.Map;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.TestPropertySource;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;

/**
 * Tests for TracingUtil helper class.
 *
 * <p>Verifies: - Span creation and management - Attribute handling - Error recording - Trace
 * context retrieval
 */
@SpringBootTest
@TestPropertySource(
    properties = {
      "management.tracing.enabled=true",
      "management.tracing.sampling.probability=1.0",
      "management.otlp.tracing.endpoint=http://localhost:4317",
      "spring.autoconfigure.exclude=org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration,org.springframework.boot.autoconfigure.orm.jpa.HibernateJpaAutoConfiguration,org.springframework.boot.autoconfigure.data.redis.RedisAutoConfiguration,org.springframework.boot.autoconfigure.kafka.KafkaAutoConfiguration"
    })
@ActiveProfiles("test")
class TracingUtilTest {

  @Autowired private TracingUtil tracingUtil;

  @Autowired private Tracer tracer;

  @Test
  void testExecuteInSpanWithReturnValue() {
    String result =
        tracingUtil.executeInSpan(
            "test-operation",
            () -> {
              // Verify span is active
              assertThat(tracer.currentSpan()).isNotNull();
              return "success";
            });

    assertThat(result).isEqualTo("success");
  }

  @Test
  void testExecuteInSpanWithAttributes() {
    Map<String, String> attributes = Map.of("user_id", "user123", "operation", "test");

    String result =
        tracingUtil.executeInSpan(
            "test-operation-with-attributes",
            attributes,
            () -> {
              // Verify span is active
              assertThat(tracer.currentSpan()).isNotNull();
              return "success";
            });

    assertThat(result).isEqualTo("success");
  }

  @Test
  void testExecuteInSpanVoid() {
    final boolean[] executed = {false};

    tracingUtil.executeInSpan(
        "test-void-operation",
        () -> {
          // Verify span is active
          assertThat(tracer.currentSpan()).isNotNull();
          executed[0] = true;
        });

    assertThat(executed[0]).isTrue();
  }

  @Test
  void testExecuteInSpanWithError() {
    assertThatThrownBy(
            () ->
                tracingUtil.executeInSpan(
                    "test-error-operation",
                    () -> {
                      throw new RuntimeException("Test error");
                    }))
        .isInstanceOf(RuntimeException.class)
        .hasMessage("Test error");
  }

  @Test
  void testAddAttributesToCurrentSpan() {
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      Map<String, String> attributes = Map.of("key1", "value1", "key2", "value2");

      tracingUtil.addAttributes(attributes);

      // Verify span is still active
      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }
  }

  @Test
  void testAddSingleAttribute() {
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      tracingUtil.addAttribute("test_key", "test_value");

      // Verify span is still active
      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }
  }

  @Test
  void testRecordError() {
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      Exception error = new RuntimeException("Test error");
      tracingUtil.recordError(error);

      // Verify span is still active
      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }
  }

  @Test
  void testGetCurrentTraceId() {
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      String traceId = tracingUtil.getCurrentTraceId();

      assertThat(traceId).as("Trace ID should be available").isNotNull().isNotEmpty();
    } finally {
      span.end();
    }
  }

  @Test
  void testGetCurrentSpanId() {
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      String spanId = tracingUtil.getCurrentSpanId();

      assertThat(spanId).as("Span ID should be available").isNotNull().isNotEmpty();
    } finally {
      span.end();
    }
  }

  @Test
  void testIsTracingActive() {
    // No span active
    assertThat(tracingUtil.isTracingActive()).isFalse();

    // Create span
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      assertThat(tracingUtil.isTracingActive()).isTrue();
    } finally {
      span.end();
    }

    // Span ended
    assertThat(tracingUtil.isTracingActive()).isFalse();
  }

  @Test
  void testNestedSpansWithTracingUtil() {
    String result =
        tracingUtil.executeInSpan(
            "parent-operation",
            () -> {
              String parentTraceId = tracingUtil.getCurrentTraceId();

              return tracingUtil.executeInSpan(
                  "child-operation",
                  () -> {
                    String childTraceId = tracingUtil.getCurrentTraceId();

                    // Verify same trace ID
                    assertThat(childTraceId).isEqualTo(parentTraceId);

                    return "nested-success";
                  });
            });

    assertThat(result).isEqualTo("nested-success");
  }

  @Test
  void testAddAttributesWithNullSpan() {
    // Should not throw exception when no span is active
    tracingUtil.addAttributes(Map.of("key", "value"));
    tracingUtil.addAttribute("key", "value");
  }

  @Test
  void testRecordErrorWithNullSpan() {
    // Should not throw exception when no span is active
    tracingUtil.recordError(new RuntimeException("Test error"));
  }

  @Test
  void testGetTraceIdWithNoActiveSpan() {
    String traceId = tracingUtil.getCurrentTraceId();
    assertThat(traceId).isNull();
  }

  @Test
  void testGetSpanIdWithNoActiveSpan() {
    String spanId = tracingUtil.getCurrentSpanId();
    assertThat(spanId).isNull();
  }
}
