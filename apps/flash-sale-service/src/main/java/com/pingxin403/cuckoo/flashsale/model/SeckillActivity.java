package com.pingxin403.cuckoo.flashsale.model;

import java.time.LocalDateTime;
import java.util.Objects;

import com.pingxin403.cuckoo.flashsale.model.enums.ActivityStatus;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Index;
import jakarta.persistence.PrePersist;
import jakarta.persistence.PreUpdate;
import jakarta.persistence.Table;

/**
 * 秒杀活动实体类 Entity class representing a flash sale (seckill) activity.
 *
 * <p>This entity maps to the seckill_activity table and contains all configuration for a flash sale
 * event including stock, timing, and purchase limits.
 */
@Entity
@Table(
    name = "seckill_activity",
    indexes = {
      @Index(name = "idx_sku_id", columnList = "sku_id"),
      @Index(name = "idx_start_time", columnList = "start_time"),
      @Index(name = "idx_status", columnList = "status")
    })
public class SeckillActivity {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "activity_id", nullable = false, unique = true, length = 64)
  private String activityId;

  @Column(name = "sku_id", nullable = false, length = 64)
  private String skuId;

  @Column(name = "activity_name", nullable = false, length = 256)
  private String activityName;

  @Column(name = "total_stock", nullable = false)
  private Integer totalStock;

  @Column(name = "remaining_stock", nullable = false)
  private Integer remainingStock;

  @Column(name = "start_time", nullable = false)
  private LocalDateTime startTime;

  @Column(name = "end_time", nullable = false)
  private LocalDateTime endTime;

  @Column(name = "purchase_limit")
  private Integer purchaseLimit = 1;

  @Enumerated(EnumType.ORDINAL)
  @Column(name = "status")
  private ActivityStatus status = ActivityStatus.NOT_STARTED;

  @Column(name = "created_at", updatable = false)
  private LocalDateTime createdAt;

  @Column(name = "updated_at")
  private LocalDateTime updatedAt;

  /** Default constructor for JPA. */
  public SeckillActivity() {}

  /**
   * Constructor with required fields.
   *
   * @param activityId unique activity identifier
   * @param skuId SKU identifier
   * @param activityName name of the activity
   * @param totalStock total stock available
   * @param startTime activity start time
   * @param endTime activity end time
   */
  public SeckillActivity(
      String activityId,
      String skuId,
      String activityName,
      Integer totalStock,
      LocalDateTime startTime,
      LocalDateTime endTime) {
    this.activityId = activityId;
    this.skuId = skuId;
    this.activityName = activityName;
    this.totalStock = totalStock;
    this.remainingStock = totalStock;
    this.startTime = startTime;
    this.endTime = endTime;
  }

  @PrePersist
  protected void onCreate() {
    LocalDateTime now = LocalDateTime.now();
    this.createdAt = now;
    this.updatedAt = now;
  }

  @PreUpdate
  protected void onUpdate() {
    this.updatedAt = LocalDateTime.now();
  }

  // Getters and Setters

  public Long getId() {
    return id;
  }

  public void setId(Long id) {
    this.id = id;
  }

  public String getActivityId() {
    return activityId;
  }

  public void setActivityId(String activityId) {
    this.activityId = activityId;
  }

  public String getSkuId() {
    return skuId;
  }

  public void setSkuId(String skuId) {
    this.skuId = skuId;
  }

  public String getActivityName() {
    return activityName;
  }

  public void setActivityName(String activityName) {
    this.activityName = activityName;
  }

  public Integer getTotalStock() {
    return totalStock;
  }

  public void setTotalStock(Integer totalStock) {
    this.totalStock = totalStock;
  }

  public Integer getRemainingStock() {
    return remainingStock;
  }

  public void setRemainingStock(Integer remainingStock) {
    this.remainingStock = remainingStock;
  }

  public LocalDateTime getStartTime() {
    return startTime;
  }

  public void setStartTime(LocalDateTime startTime) {
    this.startTime = startTime;
  }

  public LocalDateTime getEndTime() {
    return endTime;
  }

  public void setEndTime(LocalDateTime endTime) {
    this.endTime = endTime;
  }

  public Integer getPurchaseLimit() {
    return purchaseLimit;
  }

  public void setPurchaseLimit(Integer purchaseLimit) {
    this.purchaseLimit = purchaseLimit;
  }

  public ActivityStatus getStatus() {
    return status;
  }

  public void setStatus(ActivityStatus status) {
    this.status = status;
  }

  public LocalDateTime getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(LocalDateTime createdAt) {
    this.createdAt = createdAt;
  }

  public LocalDateTime getUpdatedAt() {
    return updatedAt;
  }

  public void setUpdatedAt(LocalDateTime updatedAt) {
    this.updatedAt = updatedAt;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    SeckillActivity that = (SeckillActivity) o;
    return Objects.equals(activityId, that.activityId);
  }

  @Override
  public int hashCode() {
    return Objects.hash(activityId);
  }

  @Override
  public String toString() {
    return "SeckillActivity{"
        + "id="
        + id
        + ", activityId='"
        + activityId
        + '\''
        + ", skuId='"
        + skuId
        + '\''
        + ", activityName='"
        + activityName
        + '\''
        + ", totalStock="
        + totalStock
        + ", remainingStock="
        + remainingStock
        + ", startTime="
        + startTime
        + ", endTime="
        + endTime
        + ", purchaseLimit="
        + purchaseLimit
        + ", status="
        + status
        + '}';
  }
}
