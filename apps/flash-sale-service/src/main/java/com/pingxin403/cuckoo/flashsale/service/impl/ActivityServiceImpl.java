package com.pingxin403.cuckoo.flashsale.service.impl;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;
import com.pingxin403.cuckoo.flashsale.repository.SeckillActivityRepository;
import com.pingxin403.cuckoo.flashsale.service.ActivityService;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest;

/**
 * 活动服务实现 Implementation of ActivityService for managing flash sale activities.
 *
 * <p>Validates Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6
 */
@Service
public class ActivityServiceImpl implements ActivityService {

  private static final Logger logger = LoggerFactory.getLogger(ActivityServiceImpl.class);

  /** Redis key prefix for user purchase tracking */
  private static final String USER_PURCHASE_PREFIX = "user_purchase:";

  private final SeckillActivityRepository activityRepository;
  private final StringRedisTemplate stringRedisTemplate;

  public ActivityServiceImpl(
      SeckillActivityRepository activityRepository, StringRedisTemplate stringRedisTemplate) {
    this.activityRepository = activityRepository;
    this.stringRedisTemplate = stringRedisTemplate;
  }

  @Override
  @Transactional
  public SeckillActivity createActivity(ActivityCreateRequest request) {
    logger.info("Creating activity: skuId={}, name={}", request.skuId(), request.activityName());

    SeckillActivity activity = new SeckillActivity();
    activity.setActivityId(generateActivityId());
    activity.setSkuId(request.skuId());
    activity.setActivityName(request.activityName());
    activity.setTotalStock(request.totalStock());
    activity.setRemainingStock(request.totalStock());
    activity.setStartTime(request.startTime());
    activity.setEndTime(request.endTime());
    activity.setPurchaseLimit(request.getPurchaseLimit());
    activity.setStatus(ActivityStatus.NOT_STARTED);

    SeckillActivity saved = activityRepository.save(activity);
    logger.info("Activity created: activityId={}", saved.getActivityId());

    return saved;
  }

  @Override
  public Optional<SeckillActivity> getActivity(String activityId) {
    return activityRepository.findByActivityId(activityId);
  }

  @Override
  public List<SeckillActivity> getAllActivities() {
    return activityRepository.findAll();
  }

  @Override
  @Transactional
  public SeckillActivity updateActivity(String activityId, ActivityUpdateRequest request) {
    logger.info("Updating activity: activityId={}", activityId);

    SeckillActivity activity =
        activityRepository
            .findByActivityId(activityId)
            .orElseThrow(() -> new IllegalArgumentException("Activity not found: " + activityId));

    if (request.activityName() != null) {
      activity.setActivityName(request.activityName());
    }
    if (request.totalStock() != null) {
      activity.setTotalStock(request.totalStock());
    }
    if (request.startTime() != null) {
      activity.setStartTime(request.startTime());
    }
    if (request.endTime() != null) {
      activity.setEndTime(request.endTime());
    }
    if (request.purchaseLimit() != null) {
      activity.setPurchaseLimit(request.purchaseLimit());
    }

    SeckillActivity updated = activityRepository.save(activity);
    logger.info("Activity updated: activityId={}", activityId);

    return updated;
  }

  @Override
  @Transactional
  public boolean deleteActivity(String activityId) {
    logger.info("Deleting activity: activityId={}", activityId);

    Optional<SeckillActivity> activity = activityRepository.findByActivityId(activityId);
    if (activity.isEmpty()) {
      logger.warn("Activity not found for deletion: activityId={}", activityId);
      return false;
    }

    activityRepository.delete(activity.get());
    logger.info("Activity deleted: activityId={}", activityId);
    return true;
  }

  @Override
  @Transactional
  public boolean startActivity(String activityId) {
    logger.info("Manually starting activity: activityId={}", activityId);

    Optional<SeckillActivity> activityOpt = activityRepository.findByActivityId(activityId);
    if (activityOpt.isEmpty()) {
      logger.warn("Activity not found: activityId={}", activityId);
      return false;
    }

    SeckillActivity activity = activityOpt.get();
    if (activity.getStatus() != ActivityStatus.NOT_STARTED) {
      logger.warn(
          "Activity cannot be started, current status: activityId={}, status={}",
          activityId,
          activity.getStatus());
      return false;
    }

    activity.setStatus(ActivityStatus.IN_PROGRESS);
    activityRepository.save(activity);

    logger.info("Activity started: activityId={}", activityId);
    return true;
  }

  @Override
  @Transactional
  public boolean endActivity(String activityId) {
    logger.info("Manually ending activity: activityId={}", activityId);

    Optional<SeckillActivity> activityOpt = activityRepository.findByActivityId(activityId);
    if (activityOpt.isEmpty()) {
      logger.warn("Activity not found: activityId={}", activityId);
      return false;
    }

    SeckillActivity activity = activityOpt.get();
    if (activity.getStatus() == ActivityStatus.ENDED) {
      logger.info("Activity already ended: activityId={}", activityId);
      return true;
    }

    activity.setStatus(ActivityStatus.ENDED);
    activityRepository.save(activity);

    logger.info("Activity ended: activityId={}", activityId);
    return true;
  }

  @Override
  @Transactional
  public int autoManageActivityStatus() {
    logger.debug("Auto-managing activity status");

    LocalDateTime now = LocalDateTime.now();
    int updatedCount = 0;

    // Start activities whose start time has been reached
    List<SeckillActivity> toStart =
        activityRepository.findByStatusAndStartTimeBefore(ActivityStatus.NOT_STARTED, now);
    for (SeckillActivity activity : toStart) {
      activity.setStatus(ActivityStatus.IN_PROGRESS);
      activityRepository.save(activity);
      updatedCount++;
      logger.info("Auto-started activity: activityId={}", activity.getActivityId());
    }

    // End activities whose end time has been reached
    List<SeckillActivity> toEnd =
        activityRepository.findByStatusAndEndTimeBefore(ActivityStatus.IN_PROGRESS, now);
    for (SeckillActivity activity : toEnd) {
      activity.setStatus(ActivityStatus.ENDED);
      activityRepository.save(activity);
      updatedCount++;
      logger.info("Auto-ended activity (time): activityId={}", activity.getActivityId());
    }

    // End activities whose stock is depleted
    List<SeckillActivity> soldOut =
        activityRepository.findByStatusAndRemainingStock(ActivityStatus.IN_PROGRESS, 0);
    for (SeckillActivity activity : soldOut) {
      activity.setStatus(ActivityStatus.ENDED);
      activityRepository.save(activity);
      updatedCount++;
      logger.info("Auto-ended activity (sold out): activityId={}", activity.getActivityId());
    }

    if (updatedCount > 0) {
      logger.info("Auto-managed {} activities", updatedCount);
    }

    return updatedCount;
  }

  @Override
  public boolean hasReachedPurchaseLimit(String userId, String skuId) {
    try {
      // Get activity for this SKU
      Optional<SeckillActivity> activityOpt = activityRepository.findActiveActivityBySkuId(skuId);
      if (activityOpt.isEmpty()) {
        logger.warn("No active activity found for skuId={}", skuId);
        return false;
      }

      SeckillActivity activity = activityOpt.get();
      int purchaseLimit = activity.getPurchaseLimit();

      // Get user's current purchase count from Redis
      String key = getUserPurchaseKey(skuId, userId);
      String value = stringRedisTemplate.opsForValue().get(key);
      int currentPurchases = value != null ? Integer.parseInt(value) : 0;

      boolean reachedLimit = currentPurchases >= purchaseLimit;
      if (reachedLimit) {
        logger.info(
            "User reached purchase limit: userId={}, skuId={}, current={}, limit={}",
            userId,
            skuId,
            currentPurchases,
            purchaseLimit);
      }

      return reachedLimit;

    } catch (Exception e) {
      logger.error("Error checking purchase limit: userId={}, skuId={}", userId, skuId, e);
      return false;
    }
  }

  @Override
  public void recordUserPurchase(String userId, String skuId, int quantity) {
    try {
      // Get activity for TTL calculation
      Optional<SeckillActivity> activityOpt = activityRepository.findActiveActivityBySkuId(skuId);
      if (activityOpt.isEmpty()) {
        logger.warn("No active activity found for recording purchase: skuId={}", skuId);
        return;
      }

      SeckillActivity activity = activityOpt.get();
      String key = getUserPurchaseKey(skuId, userId);

      // Increment purchase count
      stringRedisTemplate.opsForValue().increment(key, quantity);

      // Set TTL to activity end time
      LocalDateTime endTime = activity.getEndTime();
      long ttlSeconds = java.time.Duration.between(LocalDateTime.now(), endTime).getSeconds();
      if (ttlSeconds > 0) {
        stringRedisTemplate.expire(key, ttlSeconds, TimeUnit.SECONDS);
      }

      logger.debug(
          "Recorded user purchase: userId={}, skuId={}, quantity={}", userId, skuId, quantity);

    } catch (Exception e) {
      logger.error("Error recording user purchase: userId={}, skuId={}", userId, skuId, e);
    }
  }

  private String generateActivityId() {
    return "ACT-" + System.currentTimeMillis() + "-" + UUID.randomUUID().toString().substring(0, 8);
  }

  private String getUserPurchaseKey(String skuId, String userId) {
    return USER_PURCHASE_PREFIX + skuId + ":" + userId;
  }
}
