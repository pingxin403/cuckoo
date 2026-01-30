package com.pingxin403.cuckoo.flashsale.service.property;

import java.io.IOException;
import java.nio.charset.StandardCharsets;

import org.springframework.core.io.ClassPathResource;
import org.springframework.data.redis.connection.RedisStandaloneConfiguration;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.util.StreamUtils;

/** Test configuration for Redis connection and Lua scripts. */
public class TestRedisConfig {

  private final String host;
  private final int port;

  public TestRedisConfig(String host, int port) {
    this.host = host;
    this.port = port;
  }

  public StringRedisTemplate stringRedisTemplate() {
    RedisStandaloneConfiguration config = new RedisStandaloneConfiguration(host, port);
    LettuceConnectionFactory connectionFactory = new LettuceConnectionFactory(config);
    connectionFactory.afterPropertiesSet();

    StringRedisTemplate template = new StringRedisTemplate();
    template.setConnectionFactory(connectionFactory);
    template.afterPropertiesSet();
    return template;
  }

  public DefaultRedisScript<Long> stockDeductScript() {
    DefaultRedisScript<Long> script = new DefaultRedisScript<>();
    script.setResultType(Long.class);
    try {
      String scriptContent =
          StreamUtils.copyToString(
              new ClassPathResource("lua/stock_deduct.lua").getInputStream(),
              StandardCharsets.UTF_8);
      script.setScriptText(scriptContent);
    } catch (IOException e) {
      throw new RuntimeException("Failed to load stock_deduct.lua", e);
    }
    return script;
  }

  public DefaultRedisScript<Long> stockRollbackScript() {
    DefaultRedisScript<Long> script = new DefaultRedisScript<>();
    script.setResultType(Long.class);
    try {
      String scriptContent =
          StreamUtils.copyToString(
              new ClassPathResource("lua/stock_rollback.lua").getInputStream(),
              StandardCharsets.UTF_8);
      script.setScriptText(scriptContent);
    } catch (IOException e) {
      throw new RuntimeException("Failed to load stock_rollback.lua", e);
    }
    return script;
  }
}
