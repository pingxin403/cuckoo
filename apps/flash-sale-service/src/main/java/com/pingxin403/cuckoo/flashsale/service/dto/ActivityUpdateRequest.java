package com.pingxin403.cuckoo.flashsale.service.dto;

import java.time.LocalDateTime;

/**
 * Request DTO for updating a flash sale activity.
 *
 * @param activityName activity name (optional)
 * @param totalStock total stock (optional)
 * @param startTime start time (optional)
 * @param endTime end time (optional)
 * @param purchaseLimit purchase limit per user (optional)
 */
public record ActivityUpdateRequest(
    String activityName,
    Integer totalStock,
    LocalDateTime startTime,
    LocalDateTime endTime,
    Integer purchaseLimit) {}
