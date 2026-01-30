package com.pingxin403.cuckoo.flashsale;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Flash Sale (Seckill) Service Application.
 *
 * <p>This service handles high-concurrency flash sale scenarios with:
 *
 * <ul>
 *   <li>Redis-based atomic inventory management using Lua scripts
 *   <li>Kafka message queue for order processing and traffic shaping
 *   <li>MySQL persistence for orders and activity management
 *   <li>Multi-layer anti-fraud and rate limiting
 *   <li>Token bucket based queue management
 * </ul>
 *
 * <p>Architecture follows the "Three-Layer Funnel Model":
 *
 * <ol>
 *   <li>Anti-fraud layer - blocks bots and suspicious users
 *   <li>Queue layer - controls entry rate with token bucket
 *   <li>Inventory layer - atomic stock deduction with Redis Lua
 * </ol>
 */
@SpringBootApplication
@EnableScheduling
public class FlashSaleApplication {

  public static void main(String[] args) {
    SpringApplication.run(FlashSaleApplication.class, args);
  }
}
