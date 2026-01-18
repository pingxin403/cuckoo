package com.myorg.template;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Main application class for {{SERVICE_NAME}}
 * 
 * This application starts a Spring Boot application with gRPC server support.
 * The gRPC server configuration is defined in application.yml.
 */
@SpringBootApplication
public class TemplateServiceApplication {
	
	private static final Logger logger = LoggerFactory.getLogger(TemplateServiceApplication.class);

	public static void main(String[] args) {
		logger.info("Starting {{SERVICE_NAME}}...");
		SpringApplication.run(TemplateServiceApplication.class, args);
		logger.info("{{SERVICE_NAME}} started successfully");
	}
}
