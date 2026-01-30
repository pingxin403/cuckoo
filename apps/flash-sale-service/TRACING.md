# Distributed Tracing Configuration

This document describes the distributed tracing implementation for the Flash Sale Service using OpenTelemetry and Jaeger.

## Overview

The Flash Sale Service integrates with the project's observability stack (OpenTelemetry + Jaeger) to provide distributed tracing capabilities. This enables:

- **Request tracking**: Follow requests across multiple services
- **Performance analysis**: Identify bottlenecks and slow operations
- **Error diagnosis**: Trace errors back to their source
- **Trace-log correlation**: Automatically link logs to traces

## Architecture

```
Flash Sale Service
       ↓
  Micrometer Tracing
       ↓
  OpenTelemetry SDK
       ↓
  OTLP Exporter (gRPC)
       ↓
  Jaeger Collector
       ↓
  Jaeger UI (http://localhost:16686)
```

## Configuration

### Application Properties

The tracing configuration is in `application.yml`:

```yaml
management:
  tracing:
    enabled: true                                    # Enable tracing
    sampling:
      probability: ${TRACING_SAMPLE_RATE:0.1}       # Sample 10% of traces
  otlp:
    tracing:
      endpoint: ${OTEL_EXPORTER_OTLP_ENDPOINT:http://localhost:4317}
```

### Environment Variables

Configure tracing using environment variables:

```bash
# Enable/disable tracing
MANAGEMENT_TRACING_ENABLED=true

# Sampling rate (0.0 to 1.0)
TRACING_SAMPLE_RATE=0.1          # 10% of traces
TRACING_SAMPLE_RATE=1.0          # 100% of traces (development)

# OTLP endpoint
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# Service identification
SPRING_APPLICATION_NAME=flash-sale-service
SPRING_APPLICATION_VERSION=1.0.0
DEPLOYMENT_ENVIRONMENT=production
```

### Sampling Rates

Recommended sampling rates by environment:

| Environment | Sample Rate | Description |
|-------------|-------------|-------------|
| Development | 1.0 (100%) | Trace all requests for debugging |
| Staging | 0.5 (50%) | Balance visibility and cost |
| Production | 0.1 (10%) | Reduce overhead while maintaining visibility |
| High Traffic | 0.01 (1%) | Minimize overhead for very high traffic |

## Components

### TracingConfig

The `TracingConfig` class configures the OpenTelemetry SDK:

- **Resource attributes**: Service name, version, environment
- **OTLP exporter**: Sends traces to Jaeger via gRPC
- **Batch processor**: Efficiently exports spans in batches
- **Sampler**: Controls which traces are recorded
- **W3C propagation**: Standard trace context propagation

### TracingUtil

The `TracingUtil` helper class provides convenient methods for tracing:

```java
@Autowired
private TracingUtil tracingUtil;

// Execute operation in a span
String result = tracingUtil.executeInSpan("operation-name", () -> {
    // Your code here
    return "result";
});

// Execute with attributes
Map<String, String> attributes = Map.of(
    "user_id", "user123",
    "sku_id", "sku456"
);
tracingUtil.executeInSpan("deduct-stock", attributes, () -> {
    // Your code here
    return result;
});

// Add attributes to current span
tracingUtil.addAttribute("order_id", orderId);

// Record errors
try {
    // Your code
} catch (Exception e) {
    tracingUtil.recordError(e);
    throw e;
}

// Get trace context
String traceId = tracingUtil.getCurrentTraceId();
String spanId = tracingUtil.getCurrentSpanId();
```

## Usage Examples

### Basic Span Creation

```java
@Service
public class InventoryService {
    
    @Autowired
    private Tracer tracer;
    
    public DeductResult deductStock(String skuId, String userId, int quantity) {
        Span span = tracer.nextSpan().name("inventory.deduct").start();
        
        try (Tracer.SpanInScope ws = tracer.withSpan(span)) {
            // Add attributes
            span.tag("sku_id", skuId);
            span.tag("user_id", userId);
            span.tag("quantity", String.valueOf(quantity));
            
            // Execute operation
            DeductResult result = executeDeduction(skuId, userId, quantity);
            
            // Add result attributes
            span.tag("success", String.valueOf(result.success()));
            span.tag("remaining_stock", String.valueOf(result.remainingStock()));
            
            return result;
        } catch (Exception e) {
            span.error(e);
            span.tag("error", "true");
            throw e;
        } finally {
            span.end();
        }
    }
}
```

### Using TracingUtil

```java
@Service
public class OrderService {
    
    @Autowired
    private TracingUtil tracingUtil;
    
    public SeckillOrder createOrder(OrderMessage message) {
        return tracingUtil.executeInSpan("order.create", 
            Map.of(
                "order_id", message.orderId(),
                "user_id", message.userId(),
                "sku_id", message.skuId()
            ),
            () -> {
                // Create order
                SeckillOrder order = new SeckillOrder();
                order.setOrderId(message.orderId());
                order.setUserId(message.userId());
                order.setSkuId(message.skuId());
                
                // Save to database
                SeckillOrder saved = orderRepository.save(order);
                
                // Add result attribute
                tracingUtil.addAttribute("order_status", saved.getStatus().name());
                
                return saved;
            }
        );
    }
}
```

### Nested Spans

```java
@Service
public class SeckillService {
    
    @Autowired
    private TracingUtil tracingUtil;
    
    @Autowired
    private InventoryService inventoryService;
    
    @Autowired
    private OrderService orderService;
    
    public SeckillResult processSeckill(String skuId, String userId) {
        return tracingUtil.executeInSpan("seckill.process",
            Map.of("sku_id", skuId, "user_id", userId),
            () -> {
                // Deduct inventory (creates child span)
                DeductResult deductResult = inventoryService.deductStock(skuId, userId, 1);
                
                if (!deductResult.success()) {
                    return SeckillResult.outOfStock();
                }
                
                // Create order (creates child span)
                OrderMessage message = new OrderMessage(
                    generateOrderId(), userId, skuId, 1, 
                    System.currentTimeMillis(), "WEB", 
                    tracingUtil.getCurrentTraceId()
                );
                
                SeckillOrder order = orderService.createOrder(message);
                
                return SeckillResult.success(order.getOrderId());
            }
        );
    }
}
```

## Trace-Log Correlation

Logs automatically include trace context when using SLF4J:

```java
@Service
public class MyService {
    
    private static final Logger logger = LoggerFactory.getLogger(MyService.class);
    
    @Autowired
    private TracingUtil tracingUtil;
    
    public void processRequest(String userId) {
        tracingUtil.executeInSpan("process-request", () -> {
            // Logs will include traceId and spanId
            logger.info("Processing request for user: {}", userId);
            
            try {
                // Do work
                logger.debug("Work completed successfully");
            } catch (Exception e) {
                // Error logs include trace context
                logger.error("Failed to process request", e);
                tracingUtil.recordError(e);
                throw e;
            }
        });
    }
}
```

Log output includes trace context:

```
2025-01-24 12:34:56 INFO  [http-nio-8084-exec-1] [4bf92f3577b34da6a3ce929d0e0e4736,00f067aa0ba902b7] 
com.pingxin403.cuckoo.flashsale.service.MyService - Processing request for user: user123
```

Format: `[traceId,spanId]`

## Viewing Traces

### Jaeger UI

1. **Start Jaeger** (if not already running):
   ```bash
   docker run -d --name jaeger \
     -p 4317:4317 \
     -p 16686:16686 \
     jaegertracing/all-in-one:latest
   ```

2. **Access Jaeger UI**: http://localhost:16686

3. **Search for traces**:
   - Select service: `flash-sale-service`
   - Filter by operation: `seckill.process`, `inventory.deduct`, etc.
   - Filter by tags: `user_id`, `sku_id`, `error=true`

### Trace Analysis

In Jaeger UI, you can:

- **View trace timeline**: See the duration of each span
- **Inspect span details**: View attributes, events, and errors
- **Analyze dependencies**: Understand service interactions
- **Compare traces**: Identify performance regressions
- **Search by tags**: Find traces with specific attributes

## Performance Considerations

### Overhead

Tracing overhead depends on sampling rate:

| Sample Rate | Overhead | Use Case |
|-------------|----------|----------|
| 1.0 (100%) | ~5-10% | Development only |
| 0.5 (50%) | ~2-5% | Staging |
| 0.1 (10%) | ~0.5-1% | Production |
| 0.01 (1%) | ~0.05-0.1% | High traffic production |

### Optimization

The configuration includes optimizations:

- **Batch processing**: Spans exported in batches of 512
- **Async export**: Non-blocking span export
- **Queue buffering**: Up to 2048 spans buffered
- **Timeout**: 30s export timeout to prevent blocking

## Integration with Observability Stack

This tracing implementation follows the patterns from `libs/observability`:

- **OpenTelemetry SDK**: Standard observability framework
- **OTLP export**: Vendor-neutral export protocol
- **W3C propagation**: Standard trace context format
- **Micrometer bridge**: Spring Boot integration

See also:
- [libs/observability/README.md](../../libs/observability/README.md)
- [libs/observability/OPENTELEMETRY_GUIDE.md](../../libs/observability/OPENTELEMETRY_GUIDE.md)

## Troubleshooting

### Traces not appearing in Jaeger

1. **Check Jaeger is running**:
   ```bash
   curl http://localhost:16686
   ```

2. **Verify OTLP endpoint**:
   ```bash
   curl http://localhost:4317
   ```

3. **Check sampling rate**:
   ```yaml
   management.tracing.sampling.probability: 1.0  # 100% for testing
   ```

4. **Check application logs** for tracing errors

### High overhead

1. **Reduce sampling rate**:
   ```yaml
   management.tracing.sampling.probability: 0.1  # 10%
   ```

2. **Reduce span granularity**: Create fewer spans

3. **Check batch settings** in `TracingConfig`

### Missing trace context in logs

1. **Verify logging pattern** includes trace context:
   ```yaml
   logging.pattern.console: "%d{yyyy-MM-dd HH:mm:ss} %-5level [%thread] [%X{traceId:-},%X{spanId:-}] %logger{36} - %msg%n"
   ```

2. **Ensure span is active** when logging:
   ```java
   tracingUtil.executeInSpan("operation", () -> {
       logger.info("This log will include trace context");
   });
   ```

## Testing

Run tracing tests:

```bash
# Run all tracing tests
./gradlew test --tests "*TracingConfigTest"
./gradlew test --tests "*TracingUtilTest"

# Run with coverage
./gradlew test jacocoTestReport
```

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Micrometer Tracing](https://micrometer.io/docs/tracing)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Design Document - Requirement 7.3](../../.kiro/specs/flash-sale-system/design.md#73-distributed-tracing)
