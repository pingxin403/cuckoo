package com.pingxin403.cuckoo.flashsale.service.dto;

import java.time.LocalDateTime;

/**
 * Request DTO for creating a flash sale activity.
 *
 * @param skuId SKU identifier
 * @param activityName activity name
 * @param totalStock total stock for the activity
 * @param startTime activity start time
 * @param endTime activity end time
 * @param purchaseLimit purchase limit per user (default: 1)
 */
public record ActivityCreateRequest(
    String skuId,
    String activityName,
    int totalStock,
    LocalDateTime startTime,
    LocalDateTime endTime,
    Integer purchaseLimit) {

  public ActivityCreateRequest {
    if (purchaseLimit == null) {
      purchaseLimit = 1;
    }
  }

  public int getPurchaseLimit() {
    return purchaseLimit != null ? purchaseLimit : 1;
  }
}
