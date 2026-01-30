package com.pingxin403.cuckoo.flashsale.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.ClassPathResource;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.data.redis.serializer.GenericJackson2JsonRedisSerializer;
import org.springframework.data.redis.serializer.StringRedisSerializer;
import org.springframework.scripting.support.ResourceScriptSource;

/**
 * Redis configuration for Flash Sale Service.
 *
 * <p>Configures Redis templates and Lua scripts for:
 *
 * <ul>
 *   <li>Atomic inventory operations (stock deduction, rollback)
 *   <li>Token bucket rate limiting
 *   <li>User purchase limit tracking
 *   <li>Order status caching
 * </ul>
 */
@Configuration
public class RedisConfig {

  /**
   * Creates a RedisTemplate for general object storage.
   *
   * @param connectionFactory the Redis connection factory
   * @return configured RedisTemplate
   */
  @Bean
  public RedisTemplate<String, Object> redisTemplate(RedisConnectionFactory connectionFactory) {
    RedisTemplate<String, Object> template = new RedisTemplate<>();
    template.setConnectionFactory(connectionFactory);

    // Use String serializer for keys
    template.setKeySerializer(new StringRedisSerializer());
    template.setHashKeySerializer(new StringRedisSerializer());

    // Use JSON serializer for values
    template.setValueSerializer(new GenericJackson2JsonRedisSerializer());
    template.setHashValueSerializer(new GenericJackson2JsonRedisSerializer());

    template.afterPropertiesSet();
    return template;
  }

  /**
   * Creates a StringRedisTemplate for string-based operations.
   *
   * @param connectionFactory the Redis connection factory
   * @return configured StringRedisTemplate
   */
  @Bean
  public StringRedisTemplate stringRedisTemplate(RedisConnectionFactory connectionFactory) {
    return new StringRedisTemplate(connectionFactory);
  }

  /**
   * Loads the stock deduction Lua script.
   *
   * <p>This script performs atomic inventory check and deduction: - KEYS[1] = stock:sku_{skuId} -
   * KEYS[2] = sold:sku_{skuId} - ARGV[1] = quantity to deduct - Returns: remaining stock (>0),
   * 0=out of stock, -1=error
   *
   * @return the stock deduction script
   */
  @Bean
  public DefaultRedisScript<Long> stockDeductScript() {
    DefaultRedisScript<Long> script = new DefaultRedisScript<>();
    script.setScriptSource(new ResourceScriptSource(new ClassPathResource("lua/stock_deduct.lua")));
    script.setResultType(Long.class);
    return script;
  }

  /**
   * Loads the stock rollback Lua script.
   *
   * <p>This script performs atomic inventory rollback: - KEYS[1] = stock:sku_{skuId} - KEYS[2] =
   * sold:sku_{skuId} - ARGV[1] = quantity to rollback - Returns: new stock count after rollback
   *
   * @return the stock rollback script
   */
  @Bean
  public DefaultRedisScript<Long> stockRollbackScript() {
    DefaultRedisScript<Long> script = new DefaultRedisScript<>();
    script.setScriptSource(
        new ResourceScriptSource(new ClassPathResource("lua/stock_rollback.lua")));
    script.setResultType(Long.class);
    return script;
  }

  /**
   * Loads the token bucket acquire Lua script.
   *
   * <p>This script implements token bucket algorithm for rate limiting: - KEYS[1] =
   * token_bucket:{skuId} - KEYS[2] = token_bucket_last:{skuId} - ARGV[1] = bucket capacity -
   * ARGV[2] = refill rate (tokens/second) - ARGV[3] = current timestamp (ms) - Returns: 1=token
   * acquired, 0=no tokens available
   *
   * @return the token bucket script
   */
  @Bean
  public DefaultRedisScript<Long> tokenBucketScript() {
    DefaultRedisScript<Long> script = new DefaultRedisScript<>();
    script.setScriptSource(new ResourceScriptSource(new ClassPathResource("lua/token_bucket.lua")));
    script.setResultType(Long.class);
    return script;
  }
}
