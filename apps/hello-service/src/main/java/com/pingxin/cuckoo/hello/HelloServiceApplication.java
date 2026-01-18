package com.pingxin.cuckoo.hello;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * HelloServiceApplication 是 Hello 服务的主应用类
 *
 * <p>该应用启动一个 Spring Boot 应用，并通过 gRPC 提供 Hello 服务。 gRPC 服务器配置在 application.yml 中，默认监听端口 9090。
 */
@SpringBootApplication
public class HelloServiceApplication {

  private static final Logger logger = LoggerFactory.getLogger(HelloServiceApplication.class);

  public static void main(String[] args) {
    logger.info("Starting Hello Service...");
    SpringApplication.run(HelloServiceApplication.class, args);
    logger.info("Hello Service started successfully");
  }
}
