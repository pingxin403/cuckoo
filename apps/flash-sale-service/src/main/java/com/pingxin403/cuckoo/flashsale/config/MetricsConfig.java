package com.pingxin403.cuckoo.flashsale.config;

import java.util.concurrent.atomic.AtomicInteger;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;

/**
 * Metrics configuration for flash sale service.
 *
 * <p>Configures Prometheus metrics exposure following the project's observability patterns.
 *
 * <p>Exposed metrics:
 *
 * <ul>
 *   <li>QPS (Queries Per Second) - via request counters
 *   <li>Response time - via request duration timers
 *   <li>Success rate - via success/failure counters
 *   <li>Inventory remaining - via inventory gauges
 *   <li>Queue length - via queue length gauges
 * </ul>
 *
 * <p>Validates Requirement: 7.1
 */
@Configuration
public class MetricsConfig {

  /** Metric name prefix for flash sale service */
  private static final String METRIC_PREFIX = "flash_sale";

  /**
   * Create metrics service bean.
   *
   * @param meterRegistry Micrometer meter registry
   * @return configured metrics service
   */
  @Bean
  public FlashSaleMetrics flashSaleMetrics(MeterRegistry meterRegistry) {
    return new FlashSaleMetrics(meterRegistry);
  }

  /**
   * Flash sale metrics service.
   *
   * <p>Provides methods to record metrics for flash sale operations.
   */
  public static class FlashSaleMetrics {

    private final MeterRegistry meterRegistry;

    // Counters for QPS and success rate
    private final Counter seckillRequestsTotal;
    private final Counter seckillSuccessTotal;
    private final Counter seckillFailureTotal;
    private final Counter inventoryDeductionsTotal;
    private final Counter inventoryRollbacksTotal;
    private final Counter queueTokenAcquiredTotal;
    private final Counter queueTokenRejectedTotal;

    // Timers for response time
    private final Timer seckillRequestDuration;
    private final Timer inventoryDeductionDuration;
    private final Timer queueAcquisitionDuration;

    // Gauges for inventory and queue length
    private final AtomicInteger currentQueueLength = new AtomicInteger(0);

    /**
     * Constructor.
     *
     * @param meterRegistry Micrometer meter registry
     */
    public FlashSaleMetrics(MeterRegistry meterRegistry) {
      this.meterRegistry = meterRegistry;

      // Initialize counters
      this.seckillRequestsTotal =
          Counter.builder(METRIC_PREFIX + ".requests.total")
              .description("Total number of seckill requests")
              .register(meterRegistry);

      this.seckillSuccessTotal =
          Counter.builder(METRIC_PREFIX + ".requests.success")
              .description("Total number of successful seckill requests")
              .register(meterRegistry);

      this.seckillFailureTotal =
          Counter.builder(METRIC_PREFIX + ".requests.failure")
              .description("Total number of failed seckill requests")
              .register(meterRegistry);

      this.inventoryDeductionsTotal =
          Counter.builder(METRIC_PREFIX + ".inventory.deductions.total")
              .description("Total number of inventory deductions")
              .register(meterRegistry);

      this.inventoryRollbacksTotal =
          Counter.builder(METRIC_PREFIX + ".inventory.rollbacks.total")
              .description("Total number of inventory rollbacks")
              .register(meterRegistry);

      this.queueTokenAcquiredTotal =
          Counter.builder(METRIC_PREFIX + ".queue.tokens.acquired")
              .description("Total number of queue tokens acquired")
              .register(meterRegistry);

      this.queueTokenRejectedTotal =
          Counter.builder(METRIC_PREFIX + ".queue.tokens.rejected")
              .description("Total number of queue tokens rejected (queuing)")
              .register(meterRegistry);

      // Initialize timers
      this.seckillRequestDuration =
          Timer.builder(METRIC_PREFIX + ".request.duration")
              .description("Duration of seckill requests")
              .register(meterRegistry);

      this.inventoryDeductionDuration =
          Timer.builder(METRIC_PREFIX + ".inventory.deduction.duration")
              .description("Duration of inventory deduction operations")
              .register(meterRegistry);

      this.queueAcquisitionDuration =
          Timer.builder(METRIC_PREFIX + ".queue.acquisition.duration")
              .description("Duration of queue token acquisition")
              .register(meterRegistry);

      // Initialize gauges
      Gauge.builder(METRIC_PREFIX + ".queue.length", currentQueueLength, AtomicInteger::get)
          .description("Current queue length (estimated)")
          .register(meterRegistry);
    }

    /**
     * Record a seckill request.
     *
     * @param success whether the request was successful
     */
    public void recordSeckillRequest(boolean success) {
      seckillRequestsTotal.increment();
      if (success) {
        seckillSuccessTotal.increment();
      } else {
        seckillFailureTotal.increment();
      }
    }

    /**
     * Record inventory deduction.
     *
     * @param skuId the SKU identifier
     * @param success whether the deduction was successful
     */
    public void recordInventoryDeduction(String skuId, boolean success) {
      if (success) {
        inventoryDeductionsTotal.increment();
      }
    }

    /**
     * Record inventory rollback.
     *
     * @param skuId the SKU identifier
     */
    public void recordInventoryRollback(String skuId) {
      inventoryRollbacksTotal.increment();
    }

    /**
     * Record queue token acquisition.
     *
     * @param acquired whether the token was acquired
     */
    public void recordQueueTokenAcquisition(boolean acquired) {
      if (acquired) {
        queueTokenAcquiredTotal.increment();
      } else {
        queueTokenRejectedTotal.increment();
      }
    }

    /**
     * Update queue length gauge.
     *
     * @param length current queue length
     */
    public void updateQueueLength(int length) {
      currentQueueLength.set(length);
    }

    /**
     * Register a gauge for inventory remaining for a specific SKU.
     *
     * @param skuId the SKU identifier
     * @param inventorySupplier supplier function to get current inventory
     */
    public void registerInventoryGauge(
        String skuId, java.util.function.Supplier<Number> inventorySupplier) {
      Gauge.builder(METRIC_PREFIX + ".inventory.remaining", inventorySupplier)
          .description("Remaining inventory for SKU")
          .tag("sku_id", skuId)
          .register(meterRegistry);
    }

    /**
     * Get seckill request duration timer.
     *
     * @return timer for recording request duration
     */
    public Timer getSeckillRequestDuration() {
      return seckillRequestDuration;
    }

    /**
     * Get inventory deduction duration timer.
     *
     * @return timer for recording deduction duration
     */
    public Timer getInventoryDeductionDuration() {
      return inventoryDeductionDuration;
    }

    /**
     * Get queue acquisition duration timer.
     *
     * @return timer for recording queue acquisition duration
     */
    public Timer getQueueAcquisitionDuration() {
      return queueAcquisitionDuration;
    }

    /**
     * Get meter registry for custom metrics.
     *
     * @return meter registry
     */
    public MeterRegistry getMeterRegistry() {
      return meterRegistry;
    }
  }
}
