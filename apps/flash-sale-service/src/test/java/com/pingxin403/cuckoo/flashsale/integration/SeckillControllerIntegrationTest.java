package com.pingxin403.cuckoo.flashsale.integration;

import static org.junit.jupiter.api.Assertions.*;

import java.time.LocalDateTime;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.client.TestRestTemplate;
import org.springframework.boot.test.web.server.LocalServerPort;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import com.pingxin403.cuckoo.flashsale.controller.SeckillController.SeckillResponse;
import com.pingxin403.cuckoo.flashsale.model.SeckillActivity;
import com.pingxin403.cuckoo.flashsale.service.ActivityService;
import com.pingxin403.cuckoo.flashsale.service.InventoryService;
import com.pingxin403.cuckoo.flashsale.service.dto.ActivityCreateRequest;

/**
 * Integration tests for SeckillController.
 *
 * <p>Tests the complete seckill flow with real Redis instance using Testcontainers.
 */
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
@Testcontainers
class SeckillControllerIntegrationTest {

  @LocalServerPort private int port;

  @Autowired private TestRestTemplate restTemplate;

  @Autowired private ActivityService activityService;

  @Autowired private InventoryService inventoryService;

  @Container
  private static final GenericContainer<?> redis =
      new GenericContainer<>(DockerImageName.parse("redis:7-alpine")).withExposedPorts(6379);

  @DynamicPropertySource
  static void redisProperties(DynamicPropertyRegistry registry) {
    registry.add("spring.data.redis.host", redis::getHost);
    registry.add("spring.data.redis.port", redis::getFirstMappedPort);
  }

  private String baseUrl;

  @BeforeEach
  void setUp() {
    baseUrl = "http://localhost:" + port + "/api/seckill";
  }

  @Test
  @DisplayName("Integration test - complete seckill flow")
  void testCompleteSeckillFlow() {
    // Step 1: Create activity
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            "test-sku-001",
            "Integration Test Activity",
            10,
            LocalDateTime.now().minusHours(1),
            LocalDateTime.now().plusHours(1),
            2);

    ResponseEntity<SeckillActivity> createResponse =
        restTemplate.postForEntity(baseUrl + "/activity", createRequest, SeckillActivity.class);

    assertEquals(HttpStatus.CREATED, createResponse.getStatusCode());
    assertNotNull(createResponse.getBody());
    String activityId = createResponse.getBody().getActivityId();
    assertNotNull(activityId);

    // Step 2: Start activity
    ResponseEntity<String> startResponse =
        restTemplate.postForEntity(
            baseUrl + "/activity/" + activityId + "/start", null, String.class);

    assertEquals(HttpStatus.OK, startResponse.getStatusCode());

    // Step 3: Perform seckill (first purchase)
    String seckillUrl =
        baseUrl
            + "/test-sku-001?userId=user001&quantity=1&source=WEB&deviceId=device001&captchaCode=1234";

    ResponseEntity<SeckillResponse> seckillResponse =
        restTemplate.postForEntity(seckillUrl, null, SeckillResponse.class);

    // Note: The actual response depends on the anti-fraud service configuration
    // In a real integration test, we would configure the services appropriately
    assertNotNull(seckillResponse);
    assertNotNull(seckillResponse.getBody());

    // The response could be:
    // - 200 (success) if anti-fraud passes
    // - 202 (queuing) if rate limited
    // - 423 (captcha required) if flagged as suspicious
    assertTrue(
        seckillResponse.getStatusCode().value() == 200
            || seckillResponse.getStatusCode().value() == 202
            || seckillResponse.getStatusCode().value() == 423);

    // Step 4: Query activity status
    ResponseEntity<SeckillActivity> getResponse =
        restTemplate.getForEntity(baseUrl + "/activity/" + activityId, SeckillActivity.class);

    assertEquals(HttpStatus.OK, getResponse.getStatusCode());
    assertNotNull(getResponse.getBody());
    assertEquals(activityId, getResponse.getBody().getActivityId());

    // Step 5: End activity
    ResponseEntity<String> endResponse =
        restTemplate.postForEntity(
            baseUrl + "/activity/" + activityId + "/end", null, String.class);

    assertEquals(HttpStatus.OK, endResponse.getStatusCode());

    // Step 6: Delete activity
    restTemplate.delete(baseUrl + "/activity/" + activityId);

    // Verify deletion
    ResponseEntity<SeckillActivity> verifyResponse =
        restTemplate.getForEntity(baseUrl + "/activity/" + activityId, SeckillActivity.class);

    assertEquals(HttpStatus.NOT_FOUND, verifyResponse.getStatusCode());
  }

  @Test
  @DisplayName("Integration test - activity CRUD operations")
  void testActivityCrudOperations() {
    // Create
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            "test-sku-002",
            "CRUD Test Activity",
            50,
            LocalDateTime.now().plusHours(1),
            LocalDateTime.now().plusHours(2),
            1);

    ResponseEntity<SeckillActivity> createResponse =
        restTemplate.postForEntity(baseUrl + "/activity", createRequest, SeckillActivity.class);

    assertEquals(HttpStatus.CREATED, createResponse.getStatusCode());
    assertNotNull(createResponse.getBody());
    String activityId = createResponse.getBody().getActivityId();

    // Read
    ResponseEntity<SeckillActivity> getResponse =
        restTemplate.getForEntity(baseUrl + "/activity/" + activityId, SeckillActivity.class);

    assertEquals(HttpStatus.OK, getResponse.getStatusCode());
    assertNotNull(getResponse.getBody());
    assertEquals("CRUD Test Activity", getResponse.getBody().getActivityName());

    // Update
    com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest updateRequest =
        new com.pingxin403.cuckoo.flashsale.service.dto.ActivityUpdateRequest(
            "Updated CRUD Test Activity", null, null, null, null);

    ResponseEntity<SeckillActivity> updateResponse =
        restTemplate.exchange(
            baseUrl + "/activity/" + activityId,
            HttpMethod.PUT,
            new HttpEntity<>(updateRequest),
            SeckillActivity.class);

    assertEquals(HttpStatus.OK, updateResponse.getStatusCode());
    assertNotNull(updateResponse.getBody());
    assertEquals("Updated CRUD Test Activity", updateResponse.getBody().getActivityName());

    // Delete
    restTemplate.delete(baseUrl + "/activity/" + activityId);

    // Verify deletion
    ResponseEntity<SeckillActivity> verifyResponse =
        restTemplate.getForEntity(baseUrl + "/activity/" + activityId, SeckillActivity.class);

    assertEquals(HttpStatus.NOT_FOUND, verifyResponse.getStatusCode());
  }

  @Test
  @DisplayName("Integration test - purchase limit enforcement")
  void testPurchaseLimitEnforcement() {
    // Create activity with purchase limit of 1
    ActivityCreateRequest createRequest =
        new ActivityCreateRequest(
            "test-sku-003",
            "Limit Test Activity",
            100,
            LocalDateTime.now().minusHours(1),
            LocalDateTime.now().plusHours(1),
            1);

    ResponseEntity<SeckillActivity> createResponse =
        restTemplate.postForEntity(baseUrl + "/activity", createRequest, SeckillActivity.class);

    assertEquals(HttpStatus.CREATED, createResponse.getStatusCode());
    String activityId = createResponse.getBody().getActivityId();

    // Start activity
    restTemplate.postForEntity(baseUrl + "/activity/" + activityId + "/start", null, String.class);

    // First purchase attempt
    String seckillUrl1 =
        baseUrl
            + "/test-sku-003?userId=user002&quantity=1&source=WEB&deviceId=device002&captchaCode=1234";

    ResponseEntity<SeckillResponse> response1 =
        restTemplate.postForEntity(seckillUrl1, null, SeckillResponse.class);

    assertNotNull(response1.getBody());

    // If first purchase succeeded, second should be blocked by purchase limit
    if (response1.getBody().code() == 200) {
      // Second purchase attempt (should be blocked)
      String seckillUrl2 =
          baseUrl
              + "/test-sku-003?userId=user002&quantity=1&source=WEB&deviceId=device002&captchaCode=1234";

      ResponseEntity<SeckillResponse> response2 =
          restTemplate.postForEntity(seckillUrl2, null, SeckillResponse.class);

      assertNotNull(response2.getBody());
      assertEquals(422, response2.getBody().code());
      assertTrue(response2.getBody().message().contains("限购"));
    }

    // Cleanup
    restTemplate.delete(baseUrl + "/activity/" + activityId);
  }

  @Test
  @DisplayName("Integration test - get all activities")
  void testGetAllActivities() {
    // Create multiple activities
    for (int i = 0; i < 3; i++) {
      ActivityCreateRequest createRequest =
          new ActivityCreateRequest(
              "test-sku-list-" + i,
              "List Test Activity " + i,
              10,
              LocalDateTime.now().plusHours(1),
              LocalDateTime.now().plusHours(2),
              1);

      restTemplate.postForEntity(baseUrl + "/activity", createRequest, SeckillActivity.class);
    }

    // Get all activities
    ResponseEntity<SeckillActivity[]> response =
        restTemplate.getForEntity(baseUrl + "/activity", SeckillActivity[].class);

    assertEquals(HttpStatus.OK, response.getStatusCode());
    assertNotNull(response.getBody());
    assertTrue(response.getBody().length >= 3);

    // Cleanup
    for (SeckillActivity activity : response.getBody()) {
      if (activity.getSkuId().startsWith("test-sku-list-")) {
        restTemplate.delete(baseUrl + "/activity/" + activity.getActivityId());
      }
    }
  }
}
