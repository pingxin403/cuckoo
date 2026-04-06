package com.pingxin403.cuckoo.flashsale.service;

import static org.junit.jupiter.api.Assertions.*;

import java.util.List;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;

class LuaScriptTest {

  private StringRedisTemplate redisTemplate;
  private DefaultRedisScript<Long> deductScript;
  private DefaultRedisScript<Long> rollbackScript;

  @BeforeEach
  void setUp() {
    deductScript = new DefaultRedisScript<>();
    deductScript.setResultType(Long.class);
    deductScript.setScriptText(
        """
        local stock = redis.call('GET', KEYS[1])
        if stock == false then
            stock = '0'
        end
        local deduct = tonumber(ARGV[1])
        if tonumber(stock) >= deduct then
            local newStock = redis.call('DECRBY', KEYS[1], deduct)
            return newStock
        else
            return -1
        end
        """);

    rollbackScript = new DefaultRedisScript<>();
    rollbackScript.setResultType(Long.class);
    rollbackScript.setScriptText(
        """
        local stock = redis.call('GET', KEYS[1])
        if stock == false then
            stock = '0'
        end
        local rollback = tonumber(ARGV[1])
        local newStock = redis.call('INCRBY', KEYS[1], rollback)
        return newStock
        """);
  }

  @Test
  void testDeductScript_SufficientStock() {
    String skuId = "test-sku-001";
    redisTemplate.opsForValue().set("stock:" + skuId, "100");

    Long result =
        redisTemplate.execute(deductScript, List.of("stock:" + skuId), String.valueOf(10));

    assertTrue(result >= 0, "Should return remaining stock");
  }

  @Test
  void testDeductScript_InsufficientStock() {
    String skuId = "test-sku-002";
    redisTemplate.opsForValue().set("stock:" + skuId, "5");

    Long result =
        redisTemplate.execute(deductScript, List.of("stock:" + skuId), String.valueOf(10));

    assertEquals(-1, result, "Should return -1 when insufficient stock");
  }

  @Test
  void testDeductScript_Atomicity() {
    String skuId = "test-sku-003";
    redisTemplate.opsForValue().set("stock:" + skuId, "100");

    for (int i = 0; i < 10; i++) {
      redisTemplate.execute(deductScript, List.of("stock:" + skuId), String.valueOf(1));
    }

    String remaining = redisTemplate.opsForValue().get("stock:" + skuId);
    assertEquals("90", remaining, "Stock should be decremented atomically");
  }

  @Test
  void testRollbackScript() {
    String skuId = "test-sku-004";
    redisTemplate.opsForValue().set("stock:" + skuId, "90");

    Long result =
        redisTemplate.execute(rollbackScript, List.of("stock:" + skuId), String.valueOf(10));

    assertEquals(100, result, "Stock should be rolled back");
  }
}
