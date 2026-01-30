package com.pingxin403.cuckoo.flashsale.model;

import java.time.LocalDateTime;
import java.util.Objects;

import com.pingxin403.cuckoo.flashsale.model.enums.OrderStatus;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Index;
import jakarta.persistence.PrePersist;
import jakarta.persistence.Table;

/**
 * 秒杀订单实体类 Entity class representing a flash sale (seckill) order.
 *
 * <p>This entity maps to the seckill_order table and tracks the complete lifecycle of a flash sale
 * order from creation through payment or cancellation.
 */
@Entity
@Table(
    name = "seckill_order",
    indexes = {
      @Index(name = "idx_user_id", columnList = "user_id"),
      @Index(name = "idx_sku_id", columnList = "sku_id"),
      @Index(name = "idx_status_created", columnList = "status, created_at"),
      @Index(name = "idx_activity_id", columnList = "activity_id")
    })
public class SeckillOrder {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "order_id", nullable = false, unique = true, length = 64)
  private String orderId;

  @Column(name = "user_id", nullable = false, length = 64)
  private String userId;

  @Column(name = "sku_id", nullable = false, length = 64)
  private String skuId;

  @Column(name = "activity_id", nullable = false, length = 64)
  private String activityId;

  @Column(name = "quantity", nullable = false)
  private Integer quantity = 1;

  @Enumerated(EnumType.ORDINAL)
  @Column(name = "status", nullable = false)
  private OrderStatus status = OrderStatus.PENDING_PAYMENT;

  @Column(name = "created_at", updatable = false)
  private LocalDateTime createdAt;

  @Column(name = "paid_at")
  private LocalDateTime paidAt;

  @Column(name = "cancelled_at")
  private LocalDateTime cancelledAt;

  @Column(name = "source", length = 16)
  private String source;

  @Column(name = "trace_id", length = 64)
  private String traceId;

  /** Default constructor for JPA. */
  public SeckillOrder() {}

  /**
   * Constructor with required fields.
   *
   * @param orderId unique order identifier
   * @param userId user identifier
   * @param skuId SKU identifier
   * @param activityId activity identifier
   * @param quantity order quantity
   */
  public SeckillOrder(
      String orderId, String userId, String skuId, String activityId, Integer quantity) {
    this.orderId = orderId;
    this.userId = userId;
    this.skuId = skuId;
    this.activityId = activityId;
    this.quantity = quantity;
  }

  @PrePersist
  protected void onCreate() {
    this.createdAt = LocalDateTime.now();
  }

  // Getters and Setters

  public Long getId() {
    return id;
  }

  public void setId(Long id) {
    this.id = id;
  }

  public String getOrderId() {
    return orderId;
  }

  public void setOrderId(String orderId) {
    this.orderId = orderId;
  }

  public String getUserId() {
    return userId;
  }

  public void setUserId(String userId) {
    this.userId = userId;
  }

  public String getSkuId() {
    return skuId;
  }

  public void setSkuId(String skuId) {
    this.skuId = skuId;
  }

  public String getActivityId() {
    return activityId;
  }

  public void setActivityId(String activityId) {
    this.activityId = activityId;
  }

  public Integer getQuantity() {
    return quantity;
  }

  public void setQuantity(Integer quantity) {
    this.quantity = quantity;
  }

  public OrderStatus getStatus() {
    return status;
  }

  public void setStatus(OrderStatus status) {
    this.status = status;
  }

  public LocalDateTime getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(LocalDateTime createdAt) {
    this.createdAt = createdAt;
  }

  public LocalDateTime getPaidAt() {
    return paidAt;
  }

  public void setPaidAt(LocalDateTime paidAt) {
    this.paidAt = paidAt;
  }

  public LocalDateTime getCancelledAt() {
    return cancelledAt;
  }

  public void setCancelledAt(LocalDateTime cancelledAt) {
    this.cancelledAt = cancelledAt;
  }

  public String getSource() {
    return source;
  }

  public void setSource(String source) {
    this.source = source;
  }

  public String getTraceId() {
    return traceId;
  }

  public void setTraceId(String traceId) {
    this.traceId = traceId;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    SeckillOrder that = (SeckillOrder) o;
    return Objects.equals(orderId, that.orderId);
  }

  @Override
  public int hashCode() {
    return Objects.hash(orderId);
  }

  @Override
  public String toString() {
    return "SeckillOrder{"
        + "id="
        + id
        + ", orderId='"
        + orderId
        + '\''
        + ", userId='"
        + userId
        + '\''
        + ", skuId='"
        + skuId
        + '\''
        + ", activityId='"
        + activityId
        + '\''
        + ", quantity="
        + quantity
        + ", status="
        + status
        + ", createdAt="
        + createdAt
        + ", source='"
        + source
        + '\''
        + '}';
  }
}
