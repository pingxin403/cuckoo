package com.pingxin403.cuckoo.flashsale.scheduled;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import com.pingxin403.cuckoo.flashsale.service.ActivityService;

/**
 * 活动状态自动管理定时任务 Scheduled task for auto-managing activity status.
 *
 * <p>This task runs periodically to:
 *
 * <ul>
 *   <li>Start activities when start time is reached
 *   <li>End activities when end time is reached
 *   <li>End activities when stock is depleted
 * </ul>
 *
 * <p>Validates: Requirements 8.2, 8.3
 */
@Component
public class ActivityStatusScheduledTask {

  private static final Logger logger = LoggerFactory.getLogger(ActivityStatusScheduledTask.class);

  private final ActivityService activityService;

  @Value("${flash-sale.activity.auto-manage-cron:0 * * * * ?}")
  private String cronExpression;

  public ActivityStatusScheduledTask(ActivityService activityService) {
    this.activityService = activityService;
  }

  /**
   * Executes activity status auto-management every minute.
   *
   * <p>Checks and updates activity statuses based on:
   *
   * <ul>
   *   <li>Start time reached -> NOT_STARTED -> IN_PROGRESS
   *   <li>End time reached -> IN_PROGRESS -> ENDED
   *   <li>Stock depleted -> IN_PROGRESS -> ENDED
   * </ul>
   *
   * <p>Validates: Requirements 8.2, 8.3
   */
  @Scheduled(cron = "${flash-sale.activity.auto-manage-cron:0 * * * * ?}")
  public void autoManageActivityStatus() {
    logger.debug("Starting scheduled activity status auto-management");

    try {
      int count = activityService.autoManageActivityStatus();

      if (count > 0) {
        logger.info("Auto-managed {} activities", count);
      } else {
        logger.debug("No activities required status update");
      }
    } catch (Exception e) {
      logger.error("Error auto-managing activity status", e);
    }
  }
}
