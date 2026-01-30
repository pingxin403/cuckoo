package com.pingxin403.cuckoo.flashsale.model;

/**
 * 订单消息记录 Record representing an order message for Kafka.
 *
 * <p>This record is used to send order information to Kafka after successful stock deduction. The
 * message contains all necessary information for downstream order processing.
 *
 * @param orderId unique order identifier
 * @param userId user identifier (used for partition routing)
 * @param skuId SKU identifier
 * @param quantity order quantity
 * @param timestamp message creation timestamp in milliseconds
 * @param source order source (APP, WEB, H5)
 * @param traceId distributed tracing identifier
 */
public record OrderMessage(
    String orderId,
    String userId,
    String skuId,
    int quantity,
    long timestamp,
    String source,
    String traceId) {

  /**
   * Builder pattern for creating OrderMessage instances.
   *
   * @return a new Builder instance
   */
  public static Builder builder() {
    return new Builder();
  }

  /** Builder class for OrderMessage. */
  public static class Builder {
    private String orderId;
    private String userId;
    private String skuId;
    private int quantity = 1;
    private long timestamp = System.currentTimeMillis();
    private String source;
    private String traceId;

    public Builder orderId(String orderId) {
      this.orderId = orderId;
      return this;
    }

    public Builder userId(String userId) {
      this.userId = userId;
      return this;
    }

    public Builder skuId(String skuId) {
      this.skuId = skuId;
      return this;
    }

    public Builder quantity(int quantity) {
      this.quantity = quantity;
      return this;
    }

    public Builder timestamp(long timestamp) {
      this.timestamp = timestamp;
      return this;
    }

    public Builder source(String source) {
      this.source = source;
      return this;
    }

    public Builder traceId(String traceId) {
      this.traceId = traceId;
      return this;
    }

    public OrderMessage build() {
      return new OrderMessage(orderId, userId, skuId, quantity, timestamp, source, traceId);
    }
  }
}
