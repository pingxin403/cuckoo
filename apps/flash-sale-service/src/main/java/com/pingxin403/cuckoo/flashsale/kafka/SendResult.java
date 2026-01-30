package com.pingxin403.cuckoo.flashsale.kafka;

/**
 * Kafka消息发送结果 Result record for Kafka message send operations.
 *
 * <p>Contains the outcome of sending an order message to Kafka, including partition and offset
 * information for successful sends.
 *
 * @param success whether the send operation succeeded
 * @param orderId the order identifier from the message
 * @param topic the Kafka topic the message was sent to
 * @param partition the partition the message was sent to (-1 if failed)
 * @param offset the offset of the message in the partition (-1 if failed)
 * @param errorMessage error message if the send failed, null otherwise
 */
public record SendResult(
    boolean success,
    String orderId,
    String topic,
    int partition,
    long offset,
    String errorMessage) {

  /**
   * Factory method to create a successful send result.
   *
   * @param orderId the order identifier
   * @param topic the Kafka topic
   * @param partition the partition number
   * @param offset the message offset
   * @return successful SendResult
   */
  public static SendResult success(String orderId, String topic, int partition, long offset) {
    return new SendResult(true, orderId, topic, partition, offset, null);
  }

  /**
   * Factory method to create a failed send result.
   *
   * @param orderId the order identifier
   * @param errorMessage the error message
   * @return failed SendResult
   */
  public static SendResult failure(String orderId, String errorMessage) {
    return new SendResult(false, orderId, null, -1, -1, errorMessage);
  }

  /**
   * Factory method to create a failed send result with topic information.
   *
   * @param orderId the order identifier
   * @param topic the Kafka topic
   * @param errorMessage the error message
   * @return failed SendResult
   */
  public static SendResult failure(String orderId, String topic, String errorMessage) {
    return new SendResult(false, orderId, topic, -1, -1, errorMessage);
  }
}
