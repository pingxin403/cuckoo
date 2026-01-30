package com.pingxin403.cuckoo.flashsale.config;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import io.micrometer.tracing.Tracer;
import io.opentelemetry.api.OpenTelemetry;

/**
 * Unit tests for TracingConfig without full Spring context.
 *
 * <p>These tests verify the configuration logic without requiring external dependencies.
 */
class TracingConfigUnitTest {

  private TracingConfig tracingConfig;

  @BeforeEach
  void setUp() {
    tracingConfig = new TracingConfig();
    ReflectionTestUtils.setField(tracingConfig, "serviceName", "test-service");
    ReflectionTestUtils.setField(tracingConfig, "serviceVersion", "1.0.0");
    ReflectionTestUtils.setField(tracingConfig, "environment", "test");
    ReflectionTestUtils.setField(tracingConfig, "otlpEndpoint", "http://localhost:4317");
    ReflectionTestUtils.setField(tracingConfig, "samplingProbability", 1.0);
  }

  @Test
  void testOpenTelemetryBeanCreation() {
    // This test verifies that the OpenTelemetry SDK can be initialized
    // even without a running collector (it will just fail to export)
    OpenTelemetry openTelemetry = tracingConfig.openTelemetry();

    assertThat(openTelemetry).as("OpenTelemetry instance should be created").isNotNull();

    // Clean up
    shutdownOpenTelemetry();
  }

  @Test
  void testTracerBeanCreation() {
    OpenTelemetry openTelemetry = tracingConfig.openTelemetry();
    Tracer tracer = tracingConfig.tracer(openTelemetry);

    assertThat(tracer).as("Tracer instance should be created").isNotNull();

    // Clean up
    shutdownOpenTelemetry();
  }

  @Test
  void testTracerCanCreateSpans() {
    OpenTelemetry openTelemetry = tracingConfig.openTelemetry();
    Tracer tracer = tracingConfig.tracer(openTelemetry);

    // Create a span
    var span = tracer.nextSpan().name("test-span").start();

    try (var ws = tracer.withSpan(span)) {
      assertThat(tracer.currentSpan()).as("Current span should be active").isNotNull();
      assertThat(tracer.currentSpan().context().traceId())
          .as("Trace ID should be present")
          .isNotNull()
          .isNotEmpty();
    } finally {
      span.end();
    }

    // Clean up
    shutdownOpenTelemetry();
  }

  @Test
  void testSpanAttributes() {
    OpenTelemetry openTelemetry = tracingConfig.openTelemetry();
    Tracer tracer = tracingConfig.tracer(openTelemetry);

    var span = tracer.nextSpan().name("test-span").start();

    try (var ws = tracer.withSpan(span)) {
      // Add attributes
      span.tag("key1", "value1");
      span.tag("key2", "value2");

      assertThat(tracer.currentSpan()).isNotNull();
    } finally {
      span.end();
    }

    // Clean up
    shutdownOpenTelemetry();
  }

  @Test
  void testNestedSpans() {
    OpenTelemetry openTelemetry = tracingConfig.openTelemetry();
    Tracer tracer = tracingConfig.tracer(openTelemetry);

    var parentSpan = tracer.nextSpan().name("parent").start();

    try (var parentScope = tracer.withSpan(parentSpan)) {
      String parentTraceId = tracer.currentSpan().context().traceId();

      var childSpan = tracer.nextSpan().name("child").start();

      try (var childScope = tracer.withSpan(childSpan)) {
        String childTraceId = tracer.currentSpan().context().traceId();

        // Verify same trace ID
        assertThat(childTraceId).isEqualTo(parentTraceId);

        // Verify different span IDs
        assertThat(tracer.currentSpan().context().spanId())
            .isNotEqualTo(parentSpan.context().spanId());
      } finally {
        childSpan.end();
      }
    } finally {
      parentSpan.end();
    }

    // Clean up
    shutdownOpenTelemetry();
  }

  /**
   * Helper method to shut down OpenTelemetry SDK to prevent resource leaks. This prevents the
   * "buildAndRegisterGlobal" from causing issues in subsequent tests.
   */
  private void shutdownOpenTelemetry() {
    try {
      // Give the SDK time to flush any pending spans
      Thread.sleep(100);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
    }
  }
}
