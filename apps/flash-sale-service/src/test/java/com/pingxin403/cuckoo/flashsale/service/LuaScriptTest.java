package com.pingxin403.cuckoo.flashsale.service;

import static org.junit.jupiter.api.Assertions.*;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.data.redis.core.script.RedisScript;

class LuaScriptTest {

  private RedisScript<Long> deductScript;
  private RedisScript<Long> rollbackScript;

  @BeforeEach
  void setUp() {
    deductScript = RedisScript.of(
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
        """, Long.class);

    rollbackScript = RedisScript.of(
        """
        local stock = redis.call('GET', KEYS[1])
        if stock == false then
            stock = '0'
        end
        local rollback = tonumber(ARGV[1])
        local newStock = redis.call('INCRBY', KEYS[1], rollback)
        return newStock
        """, Long.class);
  }

  @Test
  void testDeductScript_Exists() {
    assertNotNull(deductScript);
    String scriptAsString = deductScript.getScriptAsString();
    assertNotNull(scriptAsString);
    assertTrue(scriptAsString.contains("DECRBY"));
    assertTrue(scriptAsString.contains("GET"));
    assertTrue(scriptAsString.contains("return -1"));
  }

  @Test
  void testRollbackScript_Exists() {
    assertNotNull(rollbackScript);
    String scriptAsString = rollbackScript.getScriptAsString();
    assertNotNull(scriptAsString);
    assertTrue(scriptAsString.contains("INCRBY"));
    assertTrue(scriptAsString.contains("GET"));
  }

  @Test
  void testDeductScript_ReturnType() {
    assertEquals(Long.class, deductScript.getResultType());
  }

  @Test
  void testRollbackScript_ReturnType() {
    assertEquals(Long.class, rollbackScript.getResultType());
  }
}