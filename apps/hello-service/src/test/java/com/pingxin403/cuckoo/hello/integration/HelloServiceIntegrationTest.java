package com.pingxin403.cuckoo.hello.integration;

import static org.junit.jupiter.api.Assertions.*;

import java.util.concurrent.TimeUnit;

import org.junit.jupiter.api.*;

import com.myorg.api.v1.HelloRequest;
import com.myorg.api.v1.HelloResponse;
import com.myorg.api.v1.HelloServiceGrpc;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

/**
 * Integration tests for Hello Service Tests the service running in Docker with real gRPC
 * communication
 *
 * <p>Note: This test expects the service to be already running (via docker-compose) Run with:
 * ./scripts/run-integration-tests.sh
 */
@Tag("integration")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class HelloServiceIntegrationTest {

  private static ManagedChannel channel;
  private static HelloServiceGrpc.HelloServiceBlockingStub blockingStub;

  private static final String GRPC_HOST = System.getenv().getOrDefault("GRPC_HOST", "localhost");
  private static final int GRPC_PORT =
      Integer.parseInt(System.getenv().getOrDefault("GRPC_PORT", "9090"));

  @BeforeAll
  static void setUp() {
    // Create gRPC channel
    channel = ManagedChannelBuilder.forAddress(GRPC_HOST, GRPC_PORT).usePlaintext().build();

    blockingStub = HelloServiceGrpc.newBlockingStub(channel);
  }

  @AfterAll
  static void tearDown() throws InterruptedException {
    if (channel != null) {
      channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }
  }

  @Test
  @Order(1)
  @DisplayName("Test SayHello with valid name")
  void testSayHelloWithName() {
    // Given
    String name = "Alice";
    HelloRequest request = HelloRequest.newBuilder().setName(name).build();

    // When
    HelloResponse response = blockingStub.sayHello(request);

    // Then
    assertNotNull(response);
    assertEquals("Hello, Alice!", response.getMessage());
  }

  @Test
  @Order(2)
  @DisplayName("Test SayHello with empty name")
  void testSayHelloWithEmptyName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("").build();

    // When
    HelloResponse response = blockingStub.sayHello(request);

    // Then
    assertNotNull(response);
    assertEquals("Hello, World!", response.getMessage());
  }

  @Test
  @Order(3)
  @DisplayName("Test SayHello with no name field")
  void testSayHelloWithNoName() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().build();

    // When
    HelloResponse response = blockingStub.sayHello(request);

    // Then
    assertNotNull(response);
    assertEquals("Hello, World!", response.getMessage());
  }

  @Test
  @Order(4)
  @DisplayName("Test SayHello with special characters")
  void testSayHelloWithSpecialCharacters() {
    // Given
    String[] specialNames = {
      "张三", // Chinese characters
      "José", // Accented characters
      "O'Brien", // Apostrophe
      "Jean-Luc", // Hyphen
      "user@example.com" // Email format
    };

    // When & Then
    for (String name : specialNames) {
      HelloRequest request = HelloRequest.newBuilder().setName(name).build();

      HelloResponse response = blockingStub.sayHello(request);

      assertNotNull(response);
      assertEquals("Hello, " + name + "!", response.getMessage());
    }
  }

  @Test
  @Order(5)
  @DisplayName("Test SayHello with long name")
  void testSayHelloWithLongName() {
    // Given
    String longName = "A".repeat(1000); // 1000 character name
    HelloRequest request = HelloRequest.newBuilder().setName(longName).build();

    // When
    HelloResponse response = blockingStub.sayHello(request);

    // Then
    assertNotNull(response);
    assertEquals("Hello, " + longName + "!", response.getMessage());
  }

  @Test
  @Order(6)
  @DisplayName("Test concurrent requests")
  void testConcurrentRequests() throws InterruptedException {
    // Given
    int numRequests = 10;
    Thread[] threads = new Thread[numRequests];
    boolean[] results = new boolean[numRequests];

    // When
    for (int i = 0; i < numRequests; i++) {
      final int index = i;
      threads[i] =
          new Thread(
              () -> {
                try {
                  HelloRequest request = HelloRequest.newBuilder().setName("User" + index).build();

                  HelloResponse response = blockingStub.sayHello(request);

                  results[index] = response.getMessage().equals("Hello, User" + index + "!");
                } catch (Exception e) {
                  results[index] = false;
                }
              });
      threads[i].start();
    }

    // Wait for all threads to complete
    for (Thread thread : threads) {
      thread.join();
    }

    // Then
    for (boolean result : results) {
      assertTrue(result, "All concurrent requests should succeed");
    }
  }

  @Test
  @Order(7)
  @DisplayName("Test service availability")
  void testServiceAvailability() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("Test").build();

    // When & Then - Should not throw exception
    assertDoesNotThrow(
        () -> {
          HelloResponse response = blockingStub.sayHello(request);
          assertNotNull(response);
        });
  }

  @Test
  @Order(8)
  @DisplayName("Test response time")
  void testResponseTime() {
    // Given
    HelloRequest request = HelloRequest.newBuilder().setName("Performance Test").build();

    // When
    long startTime = System.currentTimeMillis();
    HelloResponse response = blockingStub.sayHello(request);
    long endTime = System.currentTimeMillis();
    long duration = endTime - startTime;

    // Then
    assertNotNull(response);
    assertTrue(duration < 100, "Response time should be less than 100ms, was: " + duration + "ms");
  }
}
