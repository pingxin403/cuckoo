package com.pingxin403.cuckoo.flashsale.service;

import java.util.List;
import java.util.Optional;

import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest;

/**
 * 活动服务接口 Activity service interface for managing flash sale activities.
 *
 * <p>Provides CRUD operations and lifecycle management for flash sale activities.
 *
 * <p>Validates Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6
 */
public interface ActivityService {

  /**
   * 创建秒杀活动 Create a new flash sale activity.
   *
   * <p>Validates: Requirement 8.1
   *
   * @param request the activity creation request
   * @return the created activity
   */
  SeckillActivity createActivity(ActivityCreateRequest request);

  /**
   * 查询活动 Get an activity by ID.
   *
   * <p>Validates: Requirement 8.1
   *
   * @param activityId the activity identifier
   * @return the activity if found
   */
  Optional<SeckillActivity> getActivity(String activityId);

  /**
   * 查询活动列表 Get all activities.
   *
   * <p>Validates: Requirement 8.1
   *
   * @return list of all activities
   */
  List<SeckillActivity> getAllActivities();

  /**
   * 更新活动 Update an existing activity.
   *
   * <p>Validates: Requirement 8.1
   *
   * @param activityId the activity identifier
   * @param request the update request
   * @return the updated activity
   */
  SeckillActivity updateActivity(String activityId, ActivityUpdateRequest request);

  /**
   * 删除活动 Delete an activity.
   *
   * <p>Validates: Requirement 8.1
   *
   * @param activityId the activity identifier
   * @return true if deleted successfully
   */
  boolean deleteActivity(String activityId);

  /**
   * 手动开启活动 Manually start an activity.
   *
   * <p>Validates: Requirement 8.2
   *
   * @param activityId the activity identifier
   * @return true if started successfully
   */
  boolean startActivity(String activityId);

  /**
   * 手动结束活动 Manually end an activity.
   *
   * <p>Validates: Requirement 8.3, 8.4
   *
   * @param activityId the activity identifier
   * @return true if ended successfully
   */
  boolean endActivity(String activityId);

  /**
   * 自动管理活动状态 Automatically manage activity status based on time.
   *
   * <p>This method should be called periodically to:
   *
   * <ul>
   *   <li>Start activities when start time is reached
   *   <li>End activities when end time is reached or stock is depleted
   * </ul>
   *
   * <p>Validates: Requirements 8.2, 8.3
   *
   * @return number of activities whose status was updated
   */
  int autoManageActivityStatus();

  /**
   * 检查用户限购 Check if user has reached purchase limit for an activity.
   *
   * <p>Validates: Requirements 8.5, 8.6
   *
   * @param userId the user identifier
   * @param skuId the SKU identifier
   * @return true if user has reached limit
   */
  boolean hasReachedPurchaseLimit(String userId, String skuId);

  /**
   * 记录用户购买 Record a user purchase for limit tracking.
   *
   * <p>Validates: Requirements 8.5, 8.6
   *
   * @param userId the user identifier
   * @param skuId the SKU identifier
   * @param quantity the quantity purchased
   */
  void recordUserPurchase(String userId, String skuId, int quantity);
}
