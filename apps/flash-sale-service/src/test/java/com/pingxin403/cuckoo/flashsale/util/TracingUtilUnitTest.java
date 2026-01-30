package com.pingxin403.cuckoo.flashsale.util;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.*;

import java.util.Map;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.TraceContext;
import io.micrometer.tracing.Tracer;

/**
 * Unit tests for TracingUtil using mocks.
 *
 * <p>These tests verify the utility methods without requiring a full tracing setup.
 */
@ExtendWith(MockitoExtension.class)
class TracingUtilUnitTest {

  @Mock private Tracer tracer;

  @Mock private Span span;

  @Mock private TraceContext traceContext;

  @Mock private Tracer.SpanInScope spanInScope;

  private TracingUtil tracingUtil;

  @BeforeEach
  void setUp() {
    tracingUtil = new TracingUtil(tracer);
  }

  @Test
  void testExecuteInSpanWithReturnValue() {
    when(tracer.nextSpan()).thenReturn(span);
    when(span.name(anyString())).thenReturn(span);
    when(span.start()).thenReturn(span);
    when(tracer.withSpan(span)).thenReturn(spanInScope);

    String result = tracingUtil.executeInSpan("test-operation", () -> "success");

    assertThat(result).isEqualTo("success");
    verify(span).end();
  }

  @Test
  void testExecuteInSpanWithAttributes() {
    when(tracer.nextSpan()).thenReturn(span);
    when(span.name(anyString())).thenReturn(span);
    when(span.start()).thenReturn(span);
    when(tracer.withSpan(span)).thenReturn(spanInScope);

    Map<String, String> attributes = Map.of("key1", "value1", "key2", "value2");

    String result = tracingUtil.executeInSpan("test-operation", attributes, () -> "success");

    assertThat(result).isEqualTo("success");
    verify(span).tag("key1", "value1");
    verify(span).tag("key2", "value2");
    verify(span).end();
  }

  @Test
  void testExecuteInSpanVoid() {
    when(tracer.nextSpan()).thenReturn(span);
    when(span.name(anyString())).thenReturn(span);
    when(span.start()).thenReturn(span);
    when(tracer.withSpan(span)).thenReturn(spanInScope);

    final boolean[] executed = {false};

    tracingUtil.executeInSpan("test-operation", () -> executed[0] = true);

    assertThat(executed[0]).isTrue();
    verify(span).end();
  }

  @Test
  void testExecuteInSpanWithError() {
    when(tracer.nextSpan()).thenReturn(span);
    when(span.name(anyString())).thenReturn(span);
    when(span.start()).thenReturn(span);
    when(tracer.withSpan(span)).thenReturn(spanInScope);

    assertThatThrownBy(
            () ->
                tracingUtil.executeInSpan(
                    "test-operation",
                    () -> {
                      throw new RuntimeException("Test error");
                    }))
        .isInstanceOf(RuntimeException.class)
        .hasMessage("Test error");

    verify(span).error(any(RuntimeException.class));
    verify(span).end();
  }

  @Test
  void testAddAttributesToCurrentSpan() {
    when(tracer.currentSpan()).thenReturn(span);

    Map<String, String> attributes = Map.of("key1", "value1", "key2", "value2");

    tracingUtil.addAttributes(attributes);

    verify(span).tag("key1", "value1");
    verify(span).tag("key2", "value2");
  }

  @Test
  void testAddSingleAttribute() {
    when(tracer.currentSpan()).thenReturn(span);

    tracingUtil.addAttribute("test_key", "test_value");

    verify(span).tag("test_key", "test_value");
  }

  @Test
  void testRecordError() {
    when(tracer.currentSpan()).thenReturn(span);

    Exception error = new RuntimeException("Test error");
    tracingUtil.recordError(error);

    verify(span).error(error);
    verify(span).tag("error", "true");
    verify(span).tag("error.type", "RuntimeException");
    verify(span).tag("error.message", "Test error");
  }

  @Test
  void testGetCurrentTraceId() {
    when(tracer.currentSpan()).thenReturn(span);
    when(span.context()).thenReturn(traceContext);
    when(traceContext.traceId()).thenReturn("test-trace-id");

    String traceId = tracingUtil.getCurrentTraceId();

    assertThat(traceId).isEqualTo("test-trace-id");
  }

  @Test
  void testGetCurrentSpanId() {
    when(tracer.currentSpan()).thenReturn(span);
    when(span.context()).thenReturn(traceContext);
    when(traceContext.spanId()).thenReturn("test-span-id");

    String spanId = tracingUtil.getCurrentSpanId();

    assertThat(spanId).isEqualTo("test-span-id");
  }

  @Test
  void testIsTracingActive() {
    when(tracer.currentSpan()).thenReturn(null);
    assertThat(tracingUtil.isTracingActive()).isFalse();

    when(tracer.currentSpan()).thenReturn(span);
    assertThat(tracingUtil.isTracingActive()).isTrue();
  }

  @Test
  void testAddAttributesWithNullSpan() {
    when(tracer.currentSpan()).thenReturn(null);

    // Should not throw exception
    tracingUtil.addAttributes(Map.of("key", "value"));
    tracingUtil.addAttribute("key", "value");
  }

  @Test
  void testRecordErrorWithNullSpan() {
    when(tracer.currentSpan()).thenReturn(null);

    // Should not throw exception
    tracingUtil.recordError(new RuntimeException("Test error"));
  }

  @Test
  void testGetTraceIdWithNoActiveSpan() {
    when(tracer.currentSpan()).thenReturn(null);

    String traceId = tracingUtil.getCurrentTraceId();

    assertThat(traceId).isNull();
  }

  @Test
  void testGetSpanIdWithNoActiveSpan() {
    when(tracer.currentSpan()).thenReturn(null);

    String spanId = tracingUtil.getCurrentSpanId();

    assertThat(spanId).isNull();
  }
}
