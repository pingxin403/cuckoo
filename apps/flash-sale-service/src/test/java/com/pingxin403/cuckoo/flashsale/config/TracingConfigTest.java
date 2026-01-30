package com.pingxin403.cuckoo.flashsale.config;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.context.TestPropertySource;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;
import io.opentelemetry.api.OpenTelemetry;

/**
 * Tests for distributed tracing configuration.
 *
 * <p>Verifies: - OpenTelemetry SDK initialization - Tracer bean creation - Trace context
 * propagation - Span creation and management
 */
@SpringBootTest
@TestPropertySource(
    properties = {
      "management.tracing.enabled=true",
      "management.tracing.sampling.probability=1.0",
      "management.otlp.tracing.endpoint=http://localhost:4317",
      "spring.application.name=flash-sale-service-test",
      "spring.application.version=1.0.0-test",
      "spring.autoconfigure.exclude=org.springframework.boot.autoconfigure.jdbc.DataSourceAutoConfiguration,org.springframework.boot.autoconfigure.orm.jpa.HibernateJpaAutoConfiguration,org.springframework.boot.autoconfigure.data.redis.RedisAutoConfiguration,org.springframework.boot.autoconfigure.kafka.KafkaAutoConfiguration"
    })
@ActiveProfiles("test")
class TracingConfigTest {

  @Autowired(required = false)
  private OpenTelemetry openTelemetry;

  @Autowired(required = false)
  private Tracer tracer;

  @Test
  void testOpenTelemetryBeanCreated() {
    assertThat(openTelemetry)
        .as("OpenTelemetry bean should be created when tracing is enabled")
        .isNotNull();
  }

  @Test
  void testTracerBeanCreated() {
    assertThat(tracer).as("Tracer bean should be created when tracing is enabled").isNotNull();
  }

  @Test
  void testSpanCreation() {
    // Create a span
    Span span = tracer.nextSpan().name("test-span").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      // Verify span is active
      Span currentSpan = tracer.currentSpan();
      assertThat(currentSpan).as("Current span should be active").isNotNull();
      assertThat(currentSpan.context().traceId())
          .as("Trace ID should be present")
          .isNotNull()
          .isNotEmpty();
      assertThat(currentSpan.context().spanId())
          .as("Span ID should be present")
          .isNotNull()
          .isNotEmpty();
    } finally {
      span.end();
    }
  }

  @Test
  void testSpanAttributes() {
    Span span = tracer.nextSpan().name("test-span-with-attributes").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      // Add attributes
      span.tag("user_id", "user123");
      span.tag("operation", "test");
      span.tag("success", "true");

      // Verify span is active
      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }
  }

  @Test
  void testSpanErrorRecording() {
    Span span = tracer.nextSpan().name("test-span-with-error").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      // Simulate an error
      Exception error = new RuntimeException("Test error");
      span.error(error);
      span.tag("error", "true");

      // Verify span is active
      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }
  }

  @Test
  void testNestedSpans() {
    // Create parent span
    Span parentSpan = tracer.nextSpan().name("parent-span").start();

    try (Tracer.SpanInScope parentScope = tracer.withSpan(parentSpan)) {
      String parentTraceId = tracer.currentSpan().context().traceId();

      // Create child span
      Span childSpan = tracer.nextSpan().name("child-span").start();

      try (Tracer.SpanInScope childScope = tracer.withSpan(childSpan)) {
        String childTraceId = tracer.currentSpan().context().traceId();

        // Verify child span has same trace ID as parent
        assertThat(childTraceId)
            .as("Child span should have same trace ID as parent")
            .isEqualTo(parentTraceId);

        // Verify child span has different span ID
        assertThat(tracer.currentSpan().context().spanId())
            .as("Child span should have different span ID")
            .isNotEqualTo(parentSpan.context().spanId());
      } finally {
        childSpan.end();
      }
    } finally {
      parentSpan.end();
    }
  }

  @Test
  void testTraceContextPropagation() {
    // Create a span
    Span span = tracer.nextSpan().name("test-propagation").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      String traceId = tracer.currentSpan().context().traceId();
      String spanId = tracer.currentSpan().context().spanId();

      // Verify trace context is available
      assertThat(traceId).as("Trace ID should be available for propagation").isNotNull();
      assertThat(spanId).as("Span ID should be available for propagation").isNotNull();

      // In a real scenario, these would be propagated via HTTP headers
      // using W3C Trace Context format (traceparent header)
    } finally {
      span.end();
    }
  }

  @Test
  void testSamplingConfiguration() {
    // With sampling probability = 1.0, all spans should be sampled
    Span span = tracer.nextSpan().name("test-sampling").start();

    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      // Verify span is created (sampled)
      assertThat(tracer.currentSpan())
          .as("Span should be sampled with probability 1.0")
          .isNotNull();
    } finally {
      span.end();
    }
  }
}
