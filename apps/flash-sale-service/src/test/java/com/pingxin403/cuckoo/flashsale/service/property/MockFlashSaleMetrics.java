package com.pingxin403.cuckoo.flashsale.service.property;

import com.pingxin403.cuckoo.flashsale.config.MetricsConfig;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;

/** Mock implementation of FlashSaleMetrics for testing. */
public class MockFlashSaleMetrics extends MetricsConfig.FlashSaleMetrics {

  public MockFlashSaleMetrics() {
    super(new SimpleMeterRegistry());
  }
}
