package com.pingxin403.cuckoo.flashsale.config;

import java.time.Duration;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import io.micrometer.tracing.Tracer;
import io.micrometer.tracing.otel.bridge.OtelCurrentTraceContext;
import io.micrometer.tracing.otel.bridge.OtelTracer;
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.common.AttributeKey;
import io.opentelemetry.api.common.Attributes;
import io.opentelemetry.api.trace.propagation.W3CTraceContextPropagator;
import io.opentelemetry.context.propagation.ContextPropagators;
import io.opentelemetry.exporter.otlp.trace.OtlpGrpcSpanExporter;
import io.opentelemetry.sdk.OpenTelemetrySdk;
import io.opentelemetry.sdk.resources.Resource;
import io.opentelemetry.sdk.trace.SdkTracerProvider;
import io.opentelemetry.sdk.trace.export.BatchSpanProcessor;
import io.opentelemetry.sdk.trace.samplers.Sampler;

/**
 * Distributed tracing configuration using OpenTelemetry and Jaeger.
 *
 * <p>This configuration integrates with the project's observability stack (OpenTelemetry + Jaeger)
 * following the patterns from libs/observability.
 *
 * <p>Features: - Jaeger integration via OTLP exporter - W3C Trace Context propagation - Automatic
 * trace-log correlation - Configurable sampling rate - Batch span processing for performance
 *
 * <p>Configuration properties: - management.otlp.tracing.endpoint: OTLP endpoint (default:
 * http://localhost:4317) - management.tracing.sampling.probability: Sample rate (default: 0.1 =
 * 10%) - spring.application.name: Service name - DEPLOYMENT_ENVIRONMENT: Environment name
 *
 * @see <a href="../../../../../../../../../../libs/observability/README.md">Observability
 *     Library</a>
 * @see <a
 *     href="../../../../../../../../../../libs/observability/OPENTELEMETRY_GUIDE.md">OpenTelemetry
 *     Guide</a>
 */
@Configuration
@ConditionalOnProperty(
    name = "management.tracing.enabled",
    havingValue = "true",
    matchIfMissing = true)
public class TracingConfig {

  private static final Logger logger = LoggerFactory.getLogger(TracingConfig.class);

  @Value("${spring.application.name:flash-sale-service}")
  private String serviceName;

  @Value("${spring.application.version:1.0.0}")
  private String serviceVersion;

  @Value("${DEPLOYMENT_ENVIRONMENT:local}")
  private String environment;

  @Value("${management.otlp.tracing.endpoint:http://localhost:4317}")
  private String otlpEndpoint;

  @Value("${management.tracing.sampling.probability:0.1}")
  private double samplingProbability;

  /**
   * Creates the OpenTelemetry SDK instance with Jaeger OTLP exporter.
   *
   * <p>This bean configures: - Service resource attributes (name, version, environment) - OTLP gRPC
   * span exporter to Jaeger - Batch span processor for efficient export - Probability-based sampler
   * - W3C Trace Context propagation
   *
   * @return configured OpenTelemetry instance
   */
  @Bean
  public OpenTelemetry openTelemetry() {
    logger.info(
        "Initializing OpenTelemetry tracing: service={}, version={}, environment={}, "
            + "endpoint={}, samplingRate={}",
        serviceName,
        serviceVersion,
        environment,
        otlpEndpoint,
        samplingProbability);

    // Create resource with service attributes
    Resource resource =
        Resource.getDefault()
            .merge(
                Resource.create(
                    Attributes.builder()
                        .put(AttributeKey.stringKey("service.name"), serviceName)
                        .put(AttributeKey.stringKey("service.version"), serviceVersion)
                        .put(AttributeKey.stringKey("deployment.environment"), environment)
                        .build()));

    // Configure OTLP gRPC span exporter to Jaeger
    OtlpGrpcSpanExporter spanExporter =
        OtlpGrpcSpanExporter.builder()
            .setEndpoint(otlpEndpoint)
            .setTimeout(Duration.ofSeconds(10))
            .build();

    // Configure batch span processor for efficient export
    BatchSpanProcessor spanProcessor =
        BatchSpanProcessor.builder(spanExporter)
            .setScheduleDelay(Duration.ofSeconds(5)) // Export every 5 seconds
            .setMaxQueueSize(2048) // Buffer up to 2048 spans
            .setMaxExportBatchSize(512) // Export in batches of 512
            .setExporterTimeout(Duration.ofSeconds(30))
            .build();

    // Configure sampler based on probability
    Sampler sampler = Sampler.traceIdRatioBased(samplingProbability);

    // Build tracer provider
    SdkTracerProvider tracerProvider =
        SdkTracerProvider.builder()
            .addSpanProcessor(spanProcessor)
            .setResource(resource)
            .setSampler(sampler)
            .build();

    // Build OpenTelemetry SDK with W3C Trace Context propagation
    // Note: Using build() instead of buildAndRegisterGlobal() to avoid
    // conflicts when multiple instances are created (e.g., in tests)
    OpenTelemetrySdk openTelemetry =
        OpenTelemetrySdk.builder()
            .setTracerProvider(tracerProvider)
            .setPropagators(ContextPropagators.create(W3CTraceContextPropagator.getInstance()))
            .build();

    // Add shutdown hook to flush spans on application shutdown
    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> {
                  logger.info("Shutting down OpenTelemetry tracing...");
                  tracerProvider.close();
                  logger.info("OpenTelemetry tracing shutdown complete");
                }));

    logger.info("OpenTelemetry tracing initialized successfully");
    return openTelemetry;
  }

  /**
   * Creates a Micrometer Tracer bridge to OpenTelemetry.
   *
   * <p>This allows using Micrometer's tracing API while exporting to OpenTelemetry/Jaeger. The
   * bridge automatically: - Propagates trace context in HTTP headers - Correlates traces with logs
   * (adds traceId and spanId to MDC) - Integrates with Spring Boot Actuator
   *
   * @param openTelemetry the OpenTelemetry instance
   * @return Micrometer Tracer
   */
  @Bean
  public Tracer tracer(OpenTelemetry openTelemetry) {
    io.opentelemetry.api.trace.Tracer otelTracer =
        openTelemetry.getTracer(serviceName, serviceVersion);

    OtelCurrentTraceContext otelCurrentTraceContext = new OtelCurrentTraceContext();

    return new OtelTracer(otelTracer, otelCurrentTraceContext, null);
  }
}
