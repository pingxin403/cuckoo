package com.pingxin403.cuckoo.flashsale.util;

import java.util.Map;
import java.util.function.Supplier;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;

/**
 * Utility class for distributed tracing operations.
 *
 * <p>Provides helper methods for: - Creating and managing spans - Adding attributes to spans -
 * Recording errors - Executing operations within spans
 *
 * <p>This follows the observability patterns from libs/observability.
 */
@Component
public class TracingUtil {

  private static final Logger logger = LoggerFactory.getLogger(TracingUtil.class);

  private final Tracer tracer;

  public TracingUtil(Tracer tracer) {
    this.tracer = tracer;
  }

  /**
   * Executes an operation within a new span.
   *
   * <p>The span is automatically ended when the operation completes, even if an exception is
   * thrown. Errors are automatically recorded in the span.
   *
   * @param spanName the name of the span
   * @param operation the operation to execute
   * @param <T> the return type
   * @return the result of the operation
   * @throws RuntimeException if the operation throws an exception
   */
  public <T> T executeInSpan(String spanName, Supplier<T> operation) {
    Span span = tracer.nextSpan().name(spanName).start();
    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      return operation.get();
    } catch (Exception e) {
      recordError(span, e);
      throw e;
    } finally {
      span.end();
    }
  }

  /**
   * Executes an operation within a new span with attributes.
   *
   * @param spanName the name of the span
   * @param attributes attributes to add to the span
   * @param operation the operation to execute
   * @param <T> the return type
   * @return the result of the operation
   */
  public <T> T executeInSpan(
      String spanName, Map<String, String> attributes, Supplier<T> operation) {
    Span span = tracer.nextSpan().name(spanName).start();
    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      addAttributes(span, attributes);
      return operation.get();
    } catch (Exception e) {
      recordError(span, e);
      throw e;
    } finally {
      span.end();
    }
  }

  /**
   * Executes a void operation within a new span.
   *
   * @param spanName the name of the span
   * @param operation the operation to execute
   */
  public void executeInSpan(String spanName, Runnable operation) {
    Span span = tracer.nextSpan().name(spanName).start();
    try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
      operation.run();
    } catch (Exception e) {
      recordError(span, e);
      throw e;
    } finally {
      span.end();
    }
  }

  /**
   * Adds attributes to the current span.
   *
   * @param attributes the attributes to add
   */
  public void addAttributes(Map<String, String> attributes) {
    Span currentSpan = tracer.currentSpan();
    if (currentSpan != null) {
      addAttributes(currentSpan, attributes);
    }
  }

  /**
   * Adds attributes to a specific span.
   *
   * @param span the span to add attributes to
   * @param attributes the attributes to add
   */
  public void addAttributes(Span span, Map<String, String> attributes) {
    if (span != null && attributes != null) {
      attributes.forEach(span::tag);
    }
  }

  /**
   * Adds a single attribute to the current span.
   *
   * @param key the attribute key
   * @param value the attribute value
   */
  public void addAttribute(String key, String value) {
    Span currentSpan = tracer.currentSpan();
    if (currentSpan != null) {
      currentSpan.tag(key, value);
    }
  }

  /**
   * Records an error in the current span.
   *
   * @param error the error to record
   */
  public void recordError(Throwable error) {
    Span currentSpan = tracer.currentSpan();
    if (currentSpan != null) {
      recordError(currentSpan, error);
    }
  }

  /**
   * Records an error in a specific span.
   *
   * @param span the span to record the error in
   * @param error the error to record
   */
  public void recordError(Span span, Throwable error) {
    if (span != null && error != null) {
      span.error(error);
      span.tag("error", "true");
      span.tag("error.type", error.getClass().getSimpleName());
      span.tag("error.message", error.getMessage() != null ? error.getMessage() : "");
    }
  }

  /**
   * Gets the current trace ID.
   *
   * @return the trace ID, or null if no span is active
   */
  public String getCurrentTraceId() {
    Span currentSpan = tracer.currentSpan();
    if (currentSpan != null && currentSpan.context() != null) {
      return currentSpan.context().traceId();
    }
    return null;
  }

  /**
   * Gets the current span ID.
   *
   * @return the span ID, or null if no span is active
   */
  public String getCurrentSpanId() {
    Span currentSpan = tracer.currentSpan();
    if (currentSpan != null && currentSpan.context() != null) {
      return currentSpan.context().spanId();
    }
    return null;
  }

  /**
   * Checks if tracing is currently active.
   *
   * @return true if a span is active, false otherwise
   */
  public boolean isTracingActive() {
    return tracer.currentSpan() != null;
  }
}
