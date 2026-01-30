package com.pingxin403.cuckoo.flashsale.config;

import java.util.HashMap;
import java.util.Map;

import org.apache.kafka.clients.admin.AdminClientConfig;
import org.apache.kafka.clients.admin.NewTopic;
import org.apache.kafka.clients.consumer.ConsumerConfig;
import org.apache.kafka.clients.producer.ProducerConfig;
import org.apache.kafka.common.serialization.StringDeserializer;
import org.apache.kafka.common.serialization.StringSerializer;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.config.ConcurrentKafkaListenerContainerFactory;
import org.springframework.kafka.config.TopicBuilder;
import org.springframework.kafka.core.ConsumerFactory;
import org.springframework.kafka.core.DefaultKafkaConsumerFactory;
import org.springframework.kafka.core.DefaultKafkaProducerFactory;
import org.springframework.kafka.core.KafkaAdmin;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.core.ProducerFactory;
import org.springframework.kafka.listener.ContainerProperties;
import org.springframework.kafka.support.serializer.ErrorHandlingDeserializer;
import org.springframework.kafka.support.serializer.JsonDeserializer;
import org.springframework.kafka.support.serializer.JsonSerializer;

import com.pingxin403.cuckoo.flashsale.model.DlqMessage;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * Kafka configuration for Flash Sale Service.
 *
 * <p>Configures Kafka producers, consumers, and topics for:
 *
 * <ul>
 *   <li>Order message queue (seckill-orders)
 *   <li>Dead letter queue (seckill-dlq)
 * </ul>
 */
@Configuration
public class KafkaConfig {

  @Value("${spring.kafka.bootstrap-servers}")
  private String bootstrapServers;

  @Value("${flash-sale.kafka.order-topic:seckill-orders}")
  private String orderTopic;

  @Value("${flash-sale.kafka.dlq-topic:seckill-dlq}")
  private String dlqTopic;

  @Value("${flash-sale.kafka.partitions:100}")
  private int partitions;

  @Value("${flash-sale.kafka.replication-factor:3}")
  private int replicationFactor;

  @Value("${spring.kafka.consumer.group-id:flash-sale-consumer}")
  private String consumerGroupId;

  @Value("${spring.kafka.listener.concurrency:3}")
  private int listenerConcurrency;

  /**
   * Creates the Kafka admin client for topic management.
   *
   * @return configured KafkaAdmin
   */
  @Bean
  public KafkaAdmin kafkaAdmin() {
    Map<String, Object> configs = new HashMap<>();
    configs.put(AdminClientConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
    return new KafkaAdmin(configs);
  }

  /**
   * Creates the main order topic for flash sale orders.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Partitions: 100 (production) for high throughput
   *   <li>Replication: 3 for durability
   *   <li>Retention: 7 days
   * </ul>
   *
   * @return the order topic configuration
   */
  @Bean
  public NewTopic orderTopic() {
    return TopicBuilder.name(orderTopic)
        .partitions(partitions)
        .replicas(replicationFactor)
        .config("retention.ms", "604800000") // 7 days
        .config("min.insync.replicas", String.valueOf(Math.max(1, replicationFactor - 1)))
        .build();
  }

  /**
   * Creates the dead letter queue topic for failed messages.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Partitions: 10 (lower than main topic)
   *   <li>Replication: same as main topic
   *   <li>Retention: 30 days for investigation
   * </ul>
   *
   * @return the DLQ topic configuration
   */
  @Bean
  public NewTopic dlqTopic() {
    return TopicBuilder.name(dlqTopic)
        .partitions(10)
        .replicas(replicationFactor)
        .config("retention.ms", "2592000000") // 30 days
        .build();
  }

  /**
   * Creates the producer factory with idempotent configuration.
   *
   * @return configured ProducerFactory
   */
  @Bean
  public ProducerFactory<String, Object> producerFactory() {
    Map<String, Object> configProps = new HashMap<>();
    configProps.put(ProducerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
    configProps.put(ProducerConfig.KEY_SERIALIZER_CLASS_CONFIG, StringSerializer.class);
    configProps.put(ProducerConfig.VALUE_SERIALIZER_CLASS_CONFIG, JsonSerializer.class);

    // Idempotent producer settings for exactly-once semantics
    configProps.put(ProducerConfig.ENABLE_IDEMPOTENCE_CONFIG, true);
    configProps.put(ProducerConfig.ACKS_CONFIG, "all");
    configProps.put(ProducerConfig.RETRIES_CONFIG, 3);
    configProps.put(ProducerConfig.MAX_IN_FLIGHT_REQUESTS_PER_CONNECTION, 5);

    // Performance tuning
    configProps.put(ProducerConfig.BATCH_SIZE_CONFIG, 16384);
    configProps.put(ProducerConfig.LINGER_MS_CONFIG, 5);
    configProps.put(ProducerConfig.BUFFER_MEMORY_CONFIG, 33554432);

    return new DefaultKafkaProducerFactory<>(configProps);
  }

  /**
   * Creates the KafkaTemplate for sending messages.
   *
   * @return configured KafkaTemplate
   */
  @Bean
  public KafkaTemplate<String, Object> kafkaTemplate() {
    return new KafkaTemplate<>(producerFactory());
  }

  /**
   * Creates the consumer factory for OrderMessage deserialization.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Auto-commit disabled for manual acknowledgment
   *   <li>Error handling deserializer for graceful error handling
   *   <li>JSON deserialization with trusted packages
   * </ul>
   *
   * @return configured ConsumerFactory for OrderMessage
   */
  @Bean
  public ConsumerFactory<String, OrderMessage> consumerFactory() {
    Map<String, Object> configProps = new HashMap<>();
    configProps.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
    configProps.put(ConsumerConfig.GROUP_ID_CONFIG, consumerGroupId);
    configProps.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
    configProps.put(ConsumerConfig.ENABLE_AUTO_COMMIT_CONFIG, false);

    // Use ErrorHandlingDeserializer for graceful error handling
    configProps.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG, ErrorHandlingDeserializer.class);
    configProps.put(
        ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, ErrorHandlingDeserializer.class);
    configProps.put(ErrorHandlingDeserializer.KEY_DESERIALIZER_CLASS, StringDeserializer.class);
    configProps.put(ErrorHandlingDeserializer.VALUE_DESERIALIZER_CLASS, JsonDeserializer.class);

    // JSON deserializer configuration
    configProps.put(JsonDeserializer.TRUSTED_PACKAGES, "com.pingxin403.cuckoo.flashsale.model");
    configProps.put(JsonDeserializer.VALUE_DEFAULT_TYPE, OrderMessage.class.getName());
    configProps.put(JsonDeserializer.USE_TYPE_INFO_HEADERS, false);

    // Performance tuning
    configProps.put(ConsumerConfig.MAX_POLL_RECORDS_CONFIG, 500);
    configProps.put(ConsumerConfig.FETCH_MIN_BYTES_CONFIG, 1);
    configProps.put(ConsumerConfig.FETCH_MAX_WAIT_MS_CONFIG, 500);

    return new DefaultKafkaConsumerFactory<>(configProps);
  }

  /**
   * Creates the Kafka listener container factory with manual acknowledgment.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Manual immediate acknowledgment mode
   *   <li>Configurable concurrency for parallel processing
   * </ul>
   *
   * @return configured ConcurrentKafkaListenerContainerFactory
   */
  @Bean
  public ConcurrentKafkaListenerContainerFactory<String, OrderMessage>
      kafkaListenerContainerFactory() {
    ConcurrentKafkaListenerContainerFactory<String, OrderMessage> factory =
        new ConcurrentKafkaListenerContainerFactory<>();
    factory.setConsumerFactory(consumerFactory());
    factory.setConcurrency(listenerConcurrency);

    // Manual immediate acknowledgment for reliable processing
    factory.getContainerProperties().setAckMode(ContainerProperties.AckMode.MANUAL_IMMEDIATE);

    return factory;
  }

  /**
   * Creates the consumer factory for DlqMessage deserialization.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Auto-commit disabled for manual acknowledgment
   *   <li>Error handling deserializer for graceful error handling
   *   <li>JSON deserialization with trusted packages
   * </ul>
   *
   * @return configured ConsumerFactory for DlqMessage
   */
  @Bean
  public ConsumerFactory<String, DlqMessage> dlqConsumerFactory() {
    Map<String, Object> configProps = new HashMap<>();
    configProps.put(ConsumerConfig.BOOTSTRAP_SERVERS_CONFIG, bootstrapServers);
    configProps.put(ConsumerConfig.GROUP_ID_CONFIG, consumerGroupId + "-dlq");
    configProps.put(ConsumerConfig.AUTO_OFFSET_RESET_CONFIG, "earliest");
    configProps.put(ConsumerConfig.ENABLE_AUTO_COMMIT_CONFIG, false);

    // Use ErrorHandlingDeserializer for graceful error handling
    configProps.put(ConsumerConfig.KEY_DESERIALIZER_CLASS_CONFIG, ErrorHandlingDeserializer.class);
    configProps.put(
        ConsumerConfig.VALUE_DESERIALIZER_CLASS_CONFIG, ErrorHandlingDeserializer.class);
    configProps.put(ErrorHandlingDeserializer.KEY_DESERIALIZER_CLASS, StringDeserializer.class);
    configProps.put(ErrorHandlingDeserializer.VALUE_DESERIALIZER_CLASS, JsonDeserializer.class);

    // JSON deserializer configuration - trust both OrderMessage and DlqMessage
    configProps.put(JsonDeserializer.TRUSTED_PACKAGES, "com.pingxin403.cuckoo.flashsale.model");
    configProps.put(JsonDeserializer.VALUE_DEFAULT_TYPE, DlqMessage.class.getName());
    configProps.put(JsonDeserializer.USE_TYPE_INFO_HEADERS, false);

    // Lower throughput settings for DLQ - these are exceptional cases
    configProps.put(ConsumerConfig.MAX_POLL_RECORDS_CONFIG, 10);
    configProps.put(ConsumerConfig.FETCH_MIN_BYTES_CONFIG, 1);
    configProps.put(ConsumerConfig.FETCH_MAX_WAIT_MS_CONFIG, 1000);

    return new DefaultKafkaConsumerFactory<>(configProps);
  }

  /**
   * Creates the Kafka listener container factory for DLQ messages with manual acknowledgment.
   *
   * <p>Configuration:
   *
   * <ul>
   *   <li>Manual immediate acknowledgment mode
   *   <li>Single thread concurrency (DLQ messages are exceptional)
   * </ul>
   *
   * @return configured ConcurrentKafkaListenerContainerFactory for DLQ
   */
  @Bean
  public ConcurrentKafkaListenerContainerFactory<String, DlqMessage>
      dlqKafkaListenerContainerFactory() {
    ConcurrentKafkaListenerContainerFactory<String, DlqMessage> factory =
        new ConcurrentKafkaListenerContainerFactory<>();
    factory.setConsumerFactory(dlqConsumerFactory());
    factory.setConcurrency(1); // Single thread for DLQ processing

    // Manual immediate acknowledgment for reliable processing
    factory.getContainerProperties().setAckMode(ContainerProperties.AckMode.MANUAL_IMMEDIATE);

    return factory;
  }
}
