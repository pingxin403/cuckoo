package com.pingxin403.cuckoo.flashsale.scheduled;

import java.time.LocalDateTime;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;
import com.pingxin403.cuckoo.flashsale.repository.SeckillActivityRepository;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;

@Component
public class InventoryWarmupScheduledTask {

  private static final Logger logger = LoggerFactory.getLogger(InventoryWarmupScheduledTask.class);

  private final InventoryService inventoryService;
  private final SeckillActivityRepository activityRepository;

  @Value("${flash-sale.warmup.hours-before-start:1}")
  private int hoursBeforeStart;

  public InventoryWarmupScheduledTask(
      InventoryService inventoryService, SeckillActivityRepository activityRepository) {
    this.inventoryService = inventoryService;
    this.activityRepository = activityRepository;
  }

  @Scheduled(cron = "${flash-sale.warmup.cron:0 0 * * * ?}")
  public void warmupUpcomingInventory() {
    logger.debug("Starting scheduled inventory warmup");

    try {
      LocalDateTime now = LocalDateTime.now();
      LocalDateTime targetTime = now.plusHours(hoursBeforeStart);

      List<SeckillActivity> upcomingActivities =
          activityRepository.findByStatusAndStartTimeBetween(
              ActivityStatus.NOT_STARTED, now, targetTime);

      if (upcomingActivities.isEmpty()) {
        logger.debug("No upcoming activities found for warmup");
        return;
      }

      logger.info("Found {} upcoming activities for warmup", upcomingActivities.size());

      for (SeckillActivity activity : upcomingActivities) {
        try {
          inventoryService.warmupStock(activity.getSkuId(), activity.getTotalStock());
          logger.info(
              "Warmup completed: skuId={}, activityId={}, stock={}",
              activity.getSkuId(),
              activity.getActivityId(),
              activity.getTotalStock());
        } catch (Exception e) {
          logger.error(
              "Warmup failed: skuId={}, activityId={}",
              activity.getSkuId(),
              activity.getActivityId(),
              e);
        }
      }
    } catch (Exception e) {
      logger.error("Error during scheduled inventory warmup", e);
    }
  }
}
