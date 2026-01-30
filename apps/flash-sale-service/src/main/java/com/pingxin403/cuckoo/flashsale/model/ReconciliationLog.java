package com.pingxin403.cuckoo.flashsale.model;

import java.time.LocalDateTime;
import java.util.Objects;

import com.pingxin403.cuckoo.flashsale.model.enums.ReconciliationStatus;

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
 * 对账记录实体类 Entity class representing a reconciliation log entry.
 *
 * <p>This entity maps to the reconciliation_log table and records the results of periodic
 * reconciliation checks between Redis stock data and MySQL order data.
 */
@Entity
@Table(
    name = "reconciliation_log",
    indexes = {
      @Index(name = "idx_sku_id", columnList = "sku_id"),
      @Index(name = "idx_status", columnList = "status")
    })
public class ReconciliationLog {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "sku_id", nullable = false, length = 64)
  private String skuId;

  @Column(name = "redis_stock", nullable = false)
  private Integer redisStock;

  @Column(name = "redis_sold", nullable = false)
  private Integer redisSold;

  @Column(name = "mysql_order_count", nullable = false)
  private Integer mysqlOrderCount;

  @Column(name = "discrepancy_count")
  private Integer discrepancyCount = 0;

  @Enumerated(EnumType.ORDINAL)
  @Column(name = "status")
  private ReconciliationStatus status = ReconciliationStatus.NORMAL;

  @Column(name = "details", columnDefinition = "JSON")
  private String details;

  @Column(name = "created_at", updatable = false)
  private LocalDateTime createdAt;

  /** Default constructor for JPA. */
  public ReconciliationLog() {}

  /**
   * Constructor with required fields.
   *
   * @param skuId SKU identifier
   * @param redisStock current stock in Redis
   * @param redisSold sold count in Redis
   * @param mysqlOrderCount order count in MySQL
   */
  public ReconciliationLog(
      String skuId, Integer redisStock, Integer redisSold, Integer mysqlOrderCount) {
    this.skuId = skuId;
    this.redisStock = redisStock;
    this.redisSold = redisSold;
    this.mysqlOrderCount = mysqlOrderCount;
  }

  /**
   * Factory method to create a normal reconciliation log (no discrepancy).
   *
   * @param skuId SKU identifier
   * @param redisStock current stock in Redis
   * @param redisSold sold count in Redis
   * @param mysqlOrderCount order count in MySQL
   * @return new ReconciliationLog instance with NORMAL status
   */
  public static ReconciliationLog createNormal(
      String skuId, Integer redisStock, Integer redisSold, Integer mysqlOrderCount) {
    ReconciliationLog log = new ReconciliationLog(skuId, redisStock, redisSold, mysqlOrderCount);
    log.setStatus(ReconciliationStatus.NORMAL);
    log.setDiscrepancyCount(0);
    return log;
  }

  /**
   * Factory method to create a discrepancy reconciliation log.
   *
   * @param skuId SKU identifier
   * @param redisStock current stock in Redis
   * @param redisSold sold count in Redis
   * @param mysqlOrderCount order count in MySQL
   * @param discrepancyCount number of discrepancies found
   * @param details JSON string with discrepancy details
   * @return new ReconciliationLog instance with DISCREPANCY status
   */
  public static ReconciliationLog createDiscrepancy(
      String skuId,
      Integer redisStock,
      Integer redisSold,
      Integer mysqlOrderCount,
      Integer discrepancyCount,
      String details) {
    ReconciliationLog log = new ReconciliationLog(skuId, redisStock, redisSold, mysqlOrderCount);
    log.setStatus(ReconciliationStatus.DISCREPANCY);
    log.setDiscrepancyCount(discrepancyCount);
    log.setDetails(details);
    return log;
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

  public String getSkuId() {
    return skuId;
  }

  public void setSkuId(String skuId) {
    this.skuId = skuId;
  }

  public Integer getRedisStock() {
    return redisStock;
  }

  public void setRedisStock(Integer redisStock) {
    this.redisStock = redisStock;
  }

  public Integer getRedisSold() {
    return redisSold;
  }

  public void setRedisSold(Integer redisSold) {
    this.redisSold = redisSold;
  }

  public Integer getMysqlOrderCount() {
    return mysqlOrderCount;
  }

  public void setMysqlOrderCount(Integer mysqlOrderCount) {
    this.mysqlOrderCount = mysqlOrderCount;
  }

  public Integer getDiscrepancyCount() {
    return discrepancyCount;
  }

  public void setDiscrepancyCount(Integer discrepancyCount) {
    this.discrepancyCount = discrepancyCount;
  }

  public ReconciliationStatus getStatus() {
    return status;
  }

  public void setStatus(ReconciliationStatus status) {
    this.status = status;
  }

  public String getDetails() {
    return details;
  }

  public void setDetails(String details) {
    this.details = details;
  }

  public LocalDateTime getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(LocalDateTime createdAt) {
    this.createdAt = createdAt;
  }

  /**
   * Check if this reconciliation found any discrepancies.
   *
   * @return true if discrepancies were found
   */
  public boolean hasDiscrepancy() {
    return status == ReconciliationStatus.DISCREPANCY || discrepancyCount > 0;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    ReconciliationLog that = (ReconciliationLog) o;
    return Objects.equals(id, that.id);
  }

  @Override
  public int hashCode() {
    return Objects.hash(id);
  }

  @Override
  public String toString() {
    return "ReconciliationLog{"
        + "id="
        + id
        + ", skuId='"
        + skuId
        + '\''
        + ", redisStock="
        + redisStock
        + ", redisSold="
        + redisSold
        + ", mysqlOrderCount="
        + mysqlOrderCount
        + ", discrepancyCount="
        + discrepancyCount
        + ", status="
        + status
        + ", createdAt="
        + createdAt
        + '}';
  }
}
