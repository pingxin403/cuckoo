package com.pingxin403.cuckoo.flashsale.model;

import java.time.LocalDateTime;
import java.util.Objects;

import com.pingxin403.cuckoo.flashsale.model.enums.StockOperation;

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
 * 库存流水实体类 Entity class representing a stock operation log entry.
 *
 * <p>This entity maps to the stock_log table and records all stock operations (deductions and
 * rollbacks) for audit and reconciliation purposes.
 */
@Entity
@Table(
    name = "stock_log",
    indexes = {
      @Index(name = "idx_sku_id", columnList = "sku_id"),
      @Index(name = "idx_order_id", columnList = "order_id")
    })
public class StockLog {

  @Id
  @GeneratedValue(strategy = GenerationType.IDENTITY)
  private Long id;

  @Column(name = "sku_id", nullable = false, length = 64)
  private String skuId;

  @Column(name = "order_id", nullable = false, length = 64)
  private String orderId;

  @Enumerated(EnumType.ORDINAL)
  @Column(name = "operation", nullable = false)
  private StockOperation operation;

  @Column(name = "quantity", nullable = false)
  private Integer quantity;

  @Column(name = "before_stock", nullable = false)
  private Integer beforeStock;

  @Column(name = "after_stock", nullable = false)
  private Integer afterStock;

  @Column(name = "created_at", updatable = false)
  private LocalDateTime createdAt;

  /** Default constructor for JPA. */
  public StockLog() {}

  /**
   * Constructor with all required fields.
   *
   * @param skuId SKU identifier
   * @param orderId order identifier
   * @param operation stock operation type (DEDUCT or ROLLBACK)
   * @param quantity quantity affected
   * @param beforeStock stock before operation
   * @param afterStock stock after operation
   */
  public StockLog(
      String skuId,
      String orderId,
      StockOperation operation,
      Integer quantity,
      Integer beforeStock,
      Integer afterStock) {
    this.skuId = skuId;
    this.orderId = orderId;
    this.operation = operation;
    this.quantity = quantity;
    this.beforeStock = beforeStock;
    this.afterStock = afterStock;
  }

  /**
   * Factory method to create a deduction log entry.
   *
   * @param skuId SKU identifier
   * @param orderId order identifier
   * @param quantity quantity deducted
   * @param beforeStock stock before deduction
   * @param afterStock stock after deduction
   * @return new StockLog instance for deduction
   */
  public static StockLog createDeductLog(
      String skuId, String orderId, Integer quantity, Integer beforeStock, Integer afterStock) {
    return new StockLog(skuId, orderId, StockOperation.DEDUCT, quantity, beforeStock, afterStock);
  }

  /**
   * Factory method to create a rollback log entry.
   *
   * @param skuId SKU identifier
   * @param orderId order identifier
   * @param quantity quantity rolled back
   * @param beforeStock stock before rollback
   * @param afterStock stock after rollback
   * @return new StockLog instance for rollback
   */
  public static StockLog createRollbackLog(
      String skuId, String orderId, Integer quantity, Integer beforeStock, Integer afterStock) {
    return new StockLog(skuId, orderId, StockOperation.ROLLBACK, quantity, beforeStock, afterStock);
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

  public String getOrderId() {
    return orderId;
  }

  public void setOrderId(String orderId) {
    this.orderId = orderId;
  }

  public StockOperation getOperation() {
    return operation;
  }

  public void setOperation(StockOperation operation) {
    this.operation = operation;
  }

  public Integer getQuantity() {
    return quantity;
  }

  public void setQuantity(Integer quantity) {
    this.quantity = quantity;
  }

  public Integer getBeforeStock() {
    return beforeStock;
  }

  public void setBeforeStock(Integer beforeStock) {
    this.beforeStock = beforeStock;
  }

  public Integer getAfterStock() {
    return afterStock;
  }

  public void setAfterStock(Integer afterStock) {
    this.afterStock = afterStock;
  }

  public LocalDateTime getCreatedAt() {
    return createdAt;
  }

  public void setCreatedAt(LocalDateTime createdAt) {
    this.createdAt = createdAt;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    StockLog stockLog = (StockLog) o;
    return Objects.equals(id, stockLog.id);
  }

  @Override
  public int hashCode() {
    return Objects.hash(id);
  }

  @Override
  public String toString() {
    return "StockLog{"
        + "id="
        + id
        + ", skuId='"
        + skuId
        + '\''
        + ", orderId='"
        + orderId
        + '\''
        + ", operation="
        + operation
        + ", quantity="
        + quantity
        + ", beforeStock="
        + beforeStock
        + ", afterStock="
        + afterStock
        + ", createdAt="
        + createdAt
        + '}';
  }
}
