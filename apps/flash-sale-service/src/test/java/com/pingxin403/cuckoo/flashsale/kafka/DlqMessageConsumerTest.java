package com.pingxin403.cuckoo.flashsale.kafka;

import static org.mockito.ArgumentMatchers.anyMap;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;

import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.kafka.support.Acknowledgment;

import com.pingxin403.cuckoo.flashsale.alert.AlertPublisher;
import com.pingxin403.cuckoo.flashsale.model.DlqMessage;
import com.pingxin403.cuckoo.flashsale.model.OrderMessage;

@ExtendWith(MockitoExtension.class)
class DlqMessageConsumerTest {

  @Mock private AlertPublisher alertPublisher;

  @Mock private Acknowledgment acknowledgment;

  @Test
  @DisplayName("should publish warning alert for dlq message")
  void shouldPublishWarningAlertForDlqMessage() {
    DlqMessageConsumer consumer = new DlqMessageConsumer(alertPublisher);
    DlqMessage message =
        DlqMessage.create(
            OrderMessage.builder()
                .orderId("order-1")
                .userId("user-1")
                .skuId("sku-1")
                .quantity(1)
                .timestamp(System.currentTimeMillis())
                .source("WEB")
                .traceId("trace-1")
                .build(),
            "test error",
            3,
            "seckill-orders",
            0,
            100);
    ConsumerRecord<String, DlqMessage> record =
        new ConsumerRecord<>("seckill-dlq", 0, 100, "order-1", message);

    consumer.consume(record, acknowledgment);

    verify(alertPublisher)
        .publishWarning(eq("flash_sale_dlq_message"), eq("DLQ message received"), anyMap());
    verify(acknowledgment).acknowledge();
  }
}
