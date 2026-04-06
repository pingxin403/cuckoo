package com.pingxin403.cuckoo.flashsale.alert;

import java.util.Map;

public interface AlertPublisher {

  void publishWarning(String title, String message, Map<String, Object> metadata);

  void publishCritical(String title, String message, Map<String, Object> metadata);
}
