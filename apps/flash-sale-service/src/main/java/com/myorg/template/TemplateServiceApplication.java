package com.myorg.template;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Main application class for flash-sale-service
 * 
 * This application starts a Spring Boot application with gRPC server support.
 * The gRPC server configuration is defined in application.yml.
 */
@SpringBootApplication
public class UflashUsaleUserviceServiceApplication {
	
	private static final Logger logger = LoggerFactory.getLogger(UflashUsaleUserviceServiceApplication.class);

	public static void main(String[] args) {
		logger.info("Starting flash-sale-service...");
		SpringApplication.run(UflashUsaleUserviceServiceApplication.class, args);
		logger.info("flash-sale-service started successfully");
	}
}
