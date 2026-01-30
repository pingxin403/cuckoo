package com.pingxin403.cuckoo.flashsale.repository;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Optional;

import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;

/**
 * 秒杀活动数据访问层 Repository interface for SeckillActivity entity operations.
 *
 * <p>Provides CRUD operations and custom queries for flash sale activity management.
 */
@Repository
public interface SeckillActivityRepository extends JpaRepository<SeckillActivity, Long> {

  /**
   * Find activity by unique activity ID.
   *
   * @param activityId the unique activity identifier
   * @return Optional containing the activity if found
   */
  Optional<SeckillActivity> findByActivityId(String activityId);

  /**
   * Find all activities for a specific SKU.
   *
   * @param skuId the SKU identifier
   * @return list of activities for the SKU
   */
  List<SeckillActivity> findBySkuId(String skuId);

  /**
   * Find activities by status.
   *
   * @param status the activity status
   * @return list of activities with the specified status
   */
  List<SeckillActivity> findByStatus(ActivityStatus status);

  /**
   * Find activities that should be started (start time has passed but status is NOT_STARTED).
   *
   * @param currentTime the current time
   * @return list of activities that should be started
   */
  @Query(
      "SELECT a FROM SeckillActivity a WHERE a.status = 0 AND a.startTime <= :currentTime AND"
          + " a.endTime > :currentTime")
  List<SeckillActivity> findActivitiesToStart(@Param("currentTime") LocalDateTime currentTime);

  /**
   * Find activities that should be ended (end time has passed or stock is depleted).
   *
   * @param currentTime the current time
   * @return list of activities that should be ended
   */
  @Query(
      "SELECT a FROM SeckillActivity a WHERE a.status = 1 AND (a.endTime <= :currentTime OR"
          + " a.remainingStock <= 0)")
  List<SeckillActivity> findActivitiesToEnd(@Param("currentTime") LocalDateTime currentTime);

  /**
   * Find active activity for a SKU (status is IN_PROGRESS).
   *
   * @param skuId the SKU identifier
   * @return Optional containing the active activity if found
   */
  @Query("SELECT a FROM SeckillActivity a WHERE a.skuId = :skuId AND a.status = 1")
  Optional<SeckillActivity> findActiveActivityBySkuId(@Param("skuId") String skuId);

  /**
   * Update activity status.
   *
   * @param activityId the activity identifier
   * @param status the new status
   * @return number of rows updated
   */
  @Modifying
  @Query("UPDATE SeckillActivity a SET a.status = :status WHERE a.activityId = :activityId")
  int updateStatus(@Param("activityId") String activityId, @Param("status") ActivityStatus status);

  /**
   * Decrement remaining stock for an activity.
   *
   * @param activityId the activity identifier
   * @param quantity the quantity to decrement
   * @return number of rows updated
   */
  @Modifying
  @Query(
      "UPDATE SeckillActivity a SET a.remainingStock = a.remainingStock - :quantity WHERE"
          + " a.activityId = :activityId AND a.remainingStock >= :quantity")
  int decrementStock(@Param("activityId") String activityId, @Param("quantity") int quantity);

  /**
   * Increment remaining stock for an activity (rollback).
   *
   * @param activityId the activity identifier
   * @param quantity the quantity to increment
   * @return number of rows updated
   */
  @Modifying
  @Query(
      "UPDATE SeckillActivity a SET a.remainingStock = a.remainingStock + :quantity WHERE"
          + " a.activityId = :activityId")
  int incrementStock(@Param("activityId") String activityId, @Param("quantity") int quantity);

  /**
   * Check if activity exists by activity ID.
   *
   * @param activityId the activity identifier
   * @return true if activity exists
   */
  boolean existsByActivityId(String activityId);

  /**
   * Find activities by status and start time before a given time.
   *
   * @param status the activity status
   * @param time the time threshold
   * @return list of activities
   */
  List<SeckillActivity> findByStatusAndStartTimeBefore(ActivityStatus status, LocalDateTime time);

  /**
   * Find activities by status and end time before a given time.
   *
   * @param status the activity status
   * @param time the time threshold
   * @return list of activities
   */
  List<SeckillActivity> findByStatusAndEndTimeBefore(ActivityStatus status, LocalDateTime time);

  /**
   * Find activities by status and remaining stock.
   *
   * @param status the activity status
   * @param remainingStock the remaining stock value
   * @return list of activities
   */
  List<SeckillActivity> findByStatusAndRemainingStock(ActivityStatus status, int remainingStock);
}
