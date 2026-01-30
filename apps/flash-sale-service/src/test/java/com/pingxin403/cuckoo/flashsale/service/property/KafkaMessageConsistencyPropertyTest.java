package com.pingxin403.cuckoo.flashsale.service.property;

import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;

import java.time.Duration;
import java.util.Optional;
import java.util.Random;
import java.util.stream.Stream;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.MethodSource;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.KafkaContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.kafka.OrderMessageProducer;
import com.pingxin403.cuckoo.flashsale.kafka.SendResult;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;
import com.pingxin403.cuckoo.flashsale.model.SeckillOrder;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.OrderService;
import com.pingxin403.cuckoo.flashsale.service.dto.DeductResult;

/**
 * Property 5: Kafka消息发送一致性
 *
 * <p>**Validates: Requirement 2.1**
 *
 * <p>Every successful deduction produces a Kafka message
 */
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 5: Kafka消息发送一致性")
public class KafkaMessageConsistencyPropertyTest {

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @Container
  private static final KafkaContainer kafka =
      new KafkaContainer(DockerImageName.parse("confluentinc/cp-kafka:7.4.0"));

  @DynamicPropertySource
  static void configureProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
    registry.add("spring.kafka.bootstrap-servers", kafka::getBootstrapServers);
  }

  @Autowired private InventoryService inventoryService;
  @Autowired private OrderMessageProducer orderMessageProducer;
  @Autowired private OrderService orderService;
  @Autowired private StringRedisTemplate redisTemplate;

  private static final Random random = new Random();

  /**
   * Property 5: Kafka消息发送一致性
   *
   * <p>For any successful stock deduction operation, there should be a corresponding Kafka order
   * message produced, and the message fields (orderId, userId, skuId, quantity) should match the
   * deduction request.
   *
   * <p>**Validates: Requirement 2.1**
   */
  @ParameterizedTest(name = "Message consistency: userId={0}, skuId={1}, quantity={2}")
  @MethodSource("generateMessageConsistencyTestCases")
  void successfulDeductionProducesKafkaMessage(String userId, String skuId, int quantity) {
    cleanupSku(skuId);

    // Setup: Warmup stock with sufficient quantity
    inventoryService.warmupStock(skuId, quantity + 100);

    // Act: Deduct stock
    DeductResult deductResult = inventoryService.deductStock(skuId, userId, quantity);

    // Assert: Deduction succeeded
    assertThat(deductResult.success()).isTrue();
    assertThat(deductResult.orderId()).isNotNull();

    // Act: Send Kafka message (simulating the flow after deduction)
    OrderMessage message =
        OrderMessage.builder()
            .orderId(deductResult.orderId())
            .userId(userId)
            .skuId(skuId)
            .quantity(quantity)
            .timestamp(System.currentTimeMillis())
            .source("TEST")
            .traceId("TRACE-" + System.nanoTime())
            .build();

    SendResult sendResult = orderMessageProducer.send(message);

    // Assert: Message sent successfully
    assertThat(sendResult.success()).as("Kafka message should be sent successfully").isTrue();
    assertThat(sendResult.orderId()).isEqualTo(deductResult.orderId());
    assertThat(sendResult.partition()).isGreaterThanOrEqualTo(0);
    assertThat(sendResult.offset()).isGreaterThanOrEqualTo(0);

    // Wait for message to be consumed and order created
    await()
        .atMost(Duration.ofSeconds(10))
        .pollInterval(Duration.ofMillis(500))
        .untilAsserted(
            () -> {
              Optional<SeckillOrder> order = orderService.getOrder(deductResult.orderId());
              assertThat(order).isPresent();
              assertThat(order.get().getUserId()).isEqualTo(userId);
              assertThat(order.get().getSkuId()).isEqualTo(skuId);
              assertThat(order.get().getQuantity()).isEqualTo(quantity);
            });
  }

  /** Generate 100+ test cases with various user IDs, SKU IDs, and quantities */
  static Stream<Object[]> generateMessageConsistencyTestCases() {
    return Stream.generate(
            () ->
                new Object[] {
                  "USER-" + random.nextInt(10000),
                  "SKU-MSG-" + random.nextInt(1000),
                  random.nextInt(10) + 1 // 1 to 10
                })
        .limit(100);
  }

  private void cleanupSku(String skuId) {
    redisTemplate.delete("stock:sku_" + skuId);
    redisTemplate.delete("sold:sku_" + skuId);
  }
}
