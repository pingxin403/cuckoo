package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.HashMap;
import java.util.Map;
import java.util.Random;
import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.KafkaContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.kafka.OrderMessageProducer;
import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

/**
 * Property 6: Kafka分区路由一致性
 *
 * <p>**Validates: Requirement 2.2**
 *
 * <p>Same userId always routes to same partition
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 6: Kafka分区路由一致性")
public class KafkaPartitionRoutingPropertyTest {

  @Container
  private static final KafkaContainer kafka =
      new KafkaContainer(DockerImageName.parse("confluentinc/cp-kafka:7.4.0"));

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.kafka.bootstrap-servers", kafka::getBootstrapServers);
  }

  @Autowired private OrderMessageProducer orderMessageProducer;

  private static final Random random = new Random();

  /**
   * Property 6: Kafka分区路由一致性
   *
   * <p>For any userId, using the same userId to send multiple messages should always route to the
   * same Kafka partition. That is: partition(userId) is a deterministic function.
   *
   * <p>**Validates: Requirement 2.2**
   */
  @ParameterizedTest(name = "Partition routing: userId={0}, messageCount={1}")
  @MethodSource("generatePartitionRoutingTestCases")
  void sameUserIdAlwaysRoutesToSamePartition(String userId, int messageCount) {
    Map<String, Integer> userPartitionMap = new HashMap<>();

    // Act: Send multiple messages with the same userId
    for (int i = 0; i < messageCount; i++) {
      OrderMessage message =
          OrderMessage.builder()
              .orderId("ORD-" + System.nanoTime() + "-" + i)
              .userId(userId)
              .skuId("SKU-" + random.nextInt(1000))
              .quantity(random.nextInt(10) + 1)
              .timestamp(System.currentTimeMillis())
              .source("TEST")
              .traceId("TRACE-" + System.nanoTime())
              .build();

      SendResult result = orderMessageProducer.send(message);

      // Assert: Message sent successfully
      assertThat(result.success()).isTrue();
      assertThat(result.partition()).isGreaterThanOrEqualTo(0);

      // Record the partition for this userId
      if (!userPartitionMap.containsKey(userId)) {
        userPartitionMap.put(userId, result.partition());
      }

      // Assert: All messages for the same userId go to the same partition
      assertThat(result.partition())
          .as("All messages for userId %s should route to the same partition", userId)
          .isEqualTo(userPartitionMap.get(userId));
    }
  }

  /** Generate 100+ test cases with various user IDs and message counts */
  static Stream<Object[]> generatePartitionRoutingTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  "USER-" + random.nextInt(10000), // Random userId
                  random.nextInt(10) + 2 // 2 to 11 messages per user
                })
        .limit(100);
  }
}
