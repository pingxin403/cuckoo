package com.pingxin403.cuckoo.flashsale.alert;

import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class LoggingAlertPublisher implements AlertPublisher {

  private static final Logger logger = LoggerFactory.getLogger(LoggingAlertPublisher.class);

  @Override
  public void publishWarning(String title, String message, Map<String, Object> metadata) {
    logger.warn("ALERT_WARNING title={} message={} metadata={}", title, message, metadata);
  }

  @Override
  public void publishCritical(String title, String message, Map<String, Object> metadata) {
    logger.error("ALERT_CRITICAL title={} message={} metadata={}", title, message, metadata);
  }
}
