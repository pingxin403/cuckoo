package com.pingxin403.cuckoo.flashsale.config;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;

/**
 * Unit tests for MetricsConfig and FlashSaleMetrics.
 *
 * <p>Validates that metrics are properly registered and recorded.
 */
class MetricsConfigTest {

  private MeterRegistry meterRegistry;
  private MetricsConfig.FlashSaleMetrics metrics;

  @BeforeEach
  void setUp() {
    meterRegistry = new SimpleMeterRegistry();
    metrics = new MetricsConfig.FlashSaleMetrics(meterRegistry);
  }

  @Test
  @DisplayName("Should register all required counters")
  void shouldRegisterCounters() {
    // Verify counters are registered
    assertThat(meterRegistry.find("flash_sale.requests.total").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.requests.success").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.requests.failure").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.inventory.deductions.total").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.inventory.rollbacks.total").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.queue.tokens.acquired").counter()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.queue.tokens.rejected").counter()).isNotNull();
  }

  @Test
  @DisplayName("Should register all required timers")
  void shouldRegisterTimers() {
    // Verify timers are registered
    assertThat(meterRegistry.find("flash_sale.request.duration").timer()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.inventory.deduction.duration").timer()).isNotNull();
    assertThat(meterRegistry.find("flash_sale.queue.acquisition.duration").timer()).isNotNull();
  }

  @Test
  @DisplayName("Should register queue length gauge")
  void shouldRegisterQueueLengthGauge() {
    // Verify gauge is registered
    Gauge gauge = meterRegistry.find("flash_sale.queue.length").gauge();
    assertThat(gauge).isNotNull();
    assertThat(gauge.value()).isEqualTo(0.0);
  }

  @Test
  @DisplayName("Should record successful seckill request")
  void shouldRecordSuccessfulRequest() {
    // Record successful request
    metrics.recordSeckillRequest(true);

    // Verify counters
    Counter totalCounter = meterRegistry.find("flash_sale.requests.total").counter();
    Counter successCounter = meterRegistry.find("flash_sale.requests.success").counter();
    Counter failureCounter = meterRegistry.find("flash_sale.requests.failure").counter();

    assertThat(totalCounter.count()).isEqualTo(1.0);
    assertThat(successCounter.count()).isEqualTo(1.0);
    assertThat(failureCounter.count()).isEqualTo(0.0);
  }

  @Test
  @DisplayName("Should record failed seckill request")
  void shouldRecordFailedRequest() {
    // Record failed request
    metrics.recordSeckillRequest(false);

    // Verify counters
    Counter totalCounter = meterRegistry.find("flash_sale.requests.total").counter();
    Counter successCounter = meterRegistry.find("flash_sale.requests.success").counter();
    Counter failureCounter = meterRegistry.find("flash_sale.requests.failure").counter();

    assertThat(totalCounter.count()).isEqualTo(1.0);
    assertThat(successCounter.count()).isEqualTo(0.0);
    assertThat(failureCounter.count()).isEqualTo(1.0);
  }

  @Test
  @DisplayName("Should record inventory deduction")
  void shouldRecordInventoryDeduction() {
    // Record successful deduction
    metrics.recordInventoryDeduction("SKU001", true);

    // Verify counter
    Counter counter = meterRegistry.find("flash_sale.inventory.deductions.total").counter();
    assertThat(counter.count()).isEqualTo(1.0);

    // Failed deduction should not increment counter
    metrics.recordInventoryDeduction("SKU001", false);
    assertThat(counter.count()).isEqualTo(1.0);
  }

  @Test
  @DisplayName("Should record inventory rollback")
  void shouldRecordInventoryRollback() {
    // Record rollback
    metrics.recordInventoryRollback("SKU001");

    // Verify counter
    Counter counter = meterRegistry.find("flash_sale.inventory.rollbacks.total").counter();
    assertThat(counter.count()).isEqualTo(1.0);
  }

  @Test
  @DisplayName("Should record queue token acquisition")
  void shouldRecordQueueTokenAcquisition() {
    // Record acquired token
    metrics.recordQueueTokenAcquisition(true);

    // Verify counters
    Counter acquiredCounter = meterRegistry.find("flash_sale.queue.tokens.acquired").counter();
    Counter rejectedCounter = meterRegistry.find("flash_sale.queue.tokens.rejected").counter();

    assertThat(acquiredCounter.count()).isEqualTo(1.0);
    assertThat(rejectedCounter.count()).isEqualTo(0.0);

    // Record rejected token
    metrics.recordQueueTokenAcquisition(false);

    assertThat(acquiredCounter.count()).isEqualTo(1.0);
    assertThat(rejectedCounter.count()).isEqualTo(1.0);
  }

  @Test
  @DisplayName("Should update queue length gauge")
  void shouldUpdateQueueLength() {
    // Update queue length
    metrics.updateQueueLength(42);

    // Verify gauge value
    Gauge gauge = meterRegistry.find("flash_sale.queue.length").gauge();
    assertThat(gauge.value()).isEqualTo(42.0);

    // Update again
    metrics.updateQueueLength(100);
    assertThat(gauge.value()).isEqualTo(100.0);
  }

  @Test
  @DisplayName("Should register inventory gauge for SKU")
  void shouldRegisterInventoryGauge() {
    // Register gauge with supplier
    metrics.registerInventoryGauge("SKU001", () -> 50);

    // Verify gauge is registered and returns correct value
    Gauge gauge =
        meterRegistry.find("flash_sale.inventory.remaining").tag("sku_id", "SKU001").gauge();
    assertThat(gauge).isNotNull();
    assertThat(gauge.value()).isEqualTo(50.0);
  }

  @Test
  @DisplayName("Should record request duration")
  void shouldRecordRequestDuration() {
    // Record duration using timer
    Timer timer = metrics.getSeckillRequestDuration();
    timer.record(
        () -> {
          try {
            Thread.sleep(10);
          } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
          }
        });

    // Verify timer recorded the duration
    assertThat(timer.count()).isEqualTo(1);
    assertThat(timer.totalTime(TimeUnit.MILLISECONDS)).isGreaterThan(0);
  }

  @Test
  @DisplayName("Should record inventory deduction duration")
  void shouldRecordInventoryDeductionDuration() {
    // Record duration
    Timer timer = metrics.getInventoryDeductionDuration();
    timer.record(100, TimeUnit.MILLISECONDS);

    // Verify timer
    assertThat(timer.count()).isEqualTo(1);
    assertThat(timer.totalTime(TimeUnit.MILLISECONDS)).isGreaterThanOrEqualTo(100);
  }

  @Test
  @DisplayName("Should record queue acquisition duration")
  void shouldRecordQueueAcquisitionDuration() {
    // Record duration
    Timer timer = metrics.getQueueAcquisitionDuration();
    timer.record(50, TimeUnit.MILLISECONDS);

    // Verify timer
    assertThat(timer.count()).isEqualTo(1);
    assertThat(timer.totalTime(TimeUnit.MILLISECONDS)).isGreaterThanOrEqualTo(50);
  }

  @Test
  @DisplayName("Should handle multiple concurrent metric recordings")
  void shouldHandleConcurrentRecordings() {
    // Record multiple metrics concurrently
    for (int i = 0; i < 100; i++) {
      metrics.recordSeckillRequest(i % 2 == 0);
      metrics.recordInventoryDeduction("SKU001", true);
      metrics.recordQueueTokenAcquisition(i % 3 == 0);
      metrics.updateQueueLength(i);
    }

    // Verify counters
    Counter totalCounter = meterRegistry.find("flash_sale.requests.total").counter();
    Counter successCounter = meterRegistry.find("flash_sale.requests.success").counter();
    Counter failureCounter = meterRegistry.find("flash_sale.requests.failure").counter();
    Counter deductionCounter =
        meterRegistry.find("flash_sale.inventory.deductions.total").counter();

    assertThat(totalCounter.count()).isEqualTo(100.0);
    assertThat(successCounter.count()).isEqualTo(50.0);
    assertThat(failureCounter.count()).isEqualTo(50.0);
    assertThat(deductionCounter.count()).isEqualTo(100.0);

    // Verify gauge has latest value
    Gauge gauge = meterRegistry.find("flash_sale.queue.length").gauge();
    assertThat(gauge.value()).isEqualTo(99.0);
  }
}
