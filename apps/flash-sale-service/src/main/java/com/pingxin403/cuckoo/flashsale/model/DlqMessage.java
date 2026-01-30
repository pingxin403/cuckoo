package com.pingxin403.cuckoo.flashsale.model;

import java.time.Instant;

/**
 * 死信队列消息记录 Record representing a message sent to the dead letter queue.
 *
 * <p>This record wraps the original OrderMessage with additional metadata about the failure,
 * including the error message, retry count, and timestamp when the message was sent to DLQ.
 *
 * <p>Requirements: 2.4, 2.5
 *
 * @param originalMessage the original order message that failed processing
 * @param errorMessage the error message describing why processing failed
 * @param retryCount the number of retry attempts before sending to DLQ
 * @param timestamp the timestamp when the message was sent to DLQ (epoch millis)
 * @param topic the original topic the message came from
 * @param partition the original partition the message came from
 * @param offset the original offset of the message
 */
public record DlqMessage(
    OrderMessage originalMessage,
    String errorMessage,
    int retryCount,
    long timestamp,
    String topic,
    int partition,
    long offset) {

  /**
   * Creates a DlqMessage with the current timestamp.
   *
   * @param originalMessage the original order message
   * @param errorMessage the error message
   * @param retryCount the number of retries attempted
   * @param topic the original topic
   * @param partition the original partition
   * @param offset the original offset
   * @return a new DlqMessage with current timestamp
   */
  public static DlqMessage create(
      OrderMessage originalMessage,
      String errorMessage,
      int retryCount,
      String topic,
      int partition,
      long offset) {
    return new DlqMessage(
        originalMessage,
        errorMessage,
        retryCount,
        Instant.now().toEpochMilli(),
        topic,
        partition,
        offset);
  }

  /**
   * Builder pattern for creating DlqMessage instances.
   *
   * @return a new Builder instance
   */
  public static Builder builder() {
    return new Builder();
  }

  /** Builder class for DlqMessage. */
  public static class Builder {
    private OrderMessage originalMessage;
    private String errorMessage;
    private int retryCount;
    private long timestamp = Instant.now().toEpochMilli();
    private String topic;
    private int partition;
    private long offset;

    public Builder originalMessage(OrderMessage originalMessage) {
      this.originalMessage = originalMessage;
      return this;
    }

    public Builder errorMessage(String errorMessage) {
      this.errorMessage = errorMessage;
      return this;
    }

    public Builder retryCount(int retryCount) {
      this.retryCount = retryCount;
      return this;
    }

    public Builder timestamp(long timestamp) {
      this.timestamp = timestamp;
      return this;
    }

    public Builder topic(String topic) {
      this.topic = topic;
      return this;
    }

    public Builder partition(int partition) {
      this.partition = partition;
      return this;
    }

    public Builder offset(long offset) {
      this.offset = offset;
      return this;
    }

    public DlqMessage build() {
      return new DlqMessage(
          originalMessage, errorMessage, retryCount, timestamp, topic, partition, offset);
    }
  }
}
