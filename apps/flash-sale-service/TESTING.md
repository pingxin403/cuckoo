# Testing Guide - Flash Sale Service

Complete testing guide for the flash-sale-service, covering unit tests, property-based tests, and integration tests.

## Table of Contents

- [Overview](#overview)
- [Test Execution](#test-execution)
- [Unit Testing](#unit-testing)
- [Property-Based Testing](#property-based-testing)
- [Integration Testing](#integration-testing)
- [Coverage Requirements](#coverage-requirements)
- [Troubleshooting](#troubleshooting)
- [CI/CD Integration](#cicd-integration)

## Overview

The flash-sale-service has three types of tests:

1. **Unit Tests** (168 tests) - Fast tests without external dependencies
2. **Property-Based Tests** (15 tests) - Parameterized tests requiring Docker
3. **Integration Tests** (10 tests) - End-to-end tests requiring Docker

### Test Statistics

| Test Type | Count | Time | Docker Required |
|-----------|-------|------|-----------------|
| Unit tests | 168 | ~20-30s | No |
| Property tests | 15 | ~3-5min | Yes |
| Integration tests | 10 | ~2-3min | Yes |
| **Total** | **193** | **~5-10min** | **Partial** |

## Test Execution

### Quick Commands

```bash
# Run unit tests only (default, no Docker required)
make test APP=flash-sale-service
# or
./gradlew test

# Run all tests including Docker-dependent ones
./gradlew testAll

# Run only Docker-dependent tests (integration + property)
./gradlew testDocker

# Run specific test class
./gradlew test --tests "InventoryServiceImplTest"

# Run with coverage report
./gradlew test jacocoTestReport
```

### Test Task Configuration

The build is configured with three test tasks:

1. **`test`** (default) - Excludes Docker-dependent tests
   - Excludes: `*IntegrationTest`, `*PropertyTest`, `TracingConfigTest`, `TracingUtilTest`
   - Coverage verification: Disabled
   - Use for: Fast feedback during development

2. **`testAll`** - Runs all tests
   - Includes: All unit, integration, and property tests
   - Coverage verification: Enabled
   - Use for: Pre-commit validation, CI/CD

3. **`testDocker`** - Runs only Docker-dependent tests
   - Includes: `*IntegrationTest`, `*PropertyTest`
   - Use for: Testing infrastructure integration

### Coverage Reports

After running tests with coverage:

```bash
# Generate HTML report
./gradlew test jacocoTestReport

# Open report in browser
open build/reports/jacoco/test/html/index.html
```

## Unit Testing

### Framework

- **JUnit 5** (Jupiter) for test structure
- **AssertJ** for fluent assertions
- **Mockito** for mocking dependencies
- **Spring Boot Test** for Spring context

### Test Structure

```
src/test/java/com/pingxin403/cuckoo/flashsale/
├── service/
│   └── impl/
│       ├── InventoryServiceImplTest.java
│       ├── OrderServiceImplTest.java
│       ├── QueueServiceImplTest.java
│       ├── AntiFraudServiceImplTest.java
│       ├── ActivityServiceImplTest.java
│       └── ReconciliationServiceImplTest.java
├── controller/
│   └── SeckillControllerTest.java
├── scheduled/
│   ├── OrderTimeoutScheduledTaskTest.java
│   └── ReconciliationScheduledTaskTest.java
└── config/
    └── MetricsConfigTest.java
```

### Example Unit Test

```java
@ExtendWith(MockitoExtension.class)
class InventoryServiceImplTest {
    
    @Mock
    private RedisTemplate<String, String> redisTemplate;
    
    @Mock
    private ValueOperations<String, String> valueOperations;
    
    @InjectMocks
    private InventoryServiceImpl inventoryService;
    
    @Test
    @DisplayName("Warmup stock successfully stores inventory in Redis")
    void warmupStock_withValidInput_storesInRedis() {
        // Arrange
        String skuId = "SKU001";
        int totalStock = 1000;
        when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        
        // Act
        inventoryService.warmupStock(skuId, totalStock);
        
        // Assert
        verify(valueOperations).set("stock:sku_" + skuId, "1000");
        verify(valueOperations).set("sold:sku_" + skuId, "0");
    }
}
```

### Best Practices

1. **Test Naming**: Use `methodName_scenario_expectedResult` format
2. **AAA Pattern**: Arrange-Act-Assert structure
3. **One Assertion Focus**: Keep tests focused on single behavior
4. **Mock External Dependencies**: Use Mockito for Redis, Kafka, MySQL
5. **Test Edge Cases**: Empty inputs, nulls, boundaries, max values

## Property-Based Testing

### Overview

Property-based tests validate universal correctness properties across many randomly generated inputs. The flash-sale-service has 15 property tests covering critical business logic.

### Framework

- **JUnit 5 @ParameterizedTest** with **@MethodSource**
- **Random data generation** using `Stream.generate()`
- **100+ iterations** per property
- **Testcontainers** for Redis, Kafka, MySQL

### Property Tests Implemented

| Property | File | Requirements | Docker |
|----------|------|--------------|--------|
| 1. 库存预热Round-Trip | InventoryWarmupRoundTripPropertyTest | 1.1 | Yes |
| 2. 库存扣减原子性 | StockDeductionAtomicityPropertyTest | 1.2, 6.1 | Yes |
| 3. 库存扣减返回值正确性 | StockDeductionResultPropertyTest | 1.3, 1.4 | Yes |
| 4. 超时回滚Round-Trip | StockRollbackRoundTripPropertyTest | 1.5, 5.3 | Yes |
| 5. Kafka消息发送一致性 | KafkaMessageConsistencyPropertyTest | 2.1 | Yes |
| 6. Kafka分区路由一致性 | KafkaPartitionRoutingPropertyTest | 2.2 | Yes |
| 7. 批量写入正确性 | BatchWriteCorrectnessPropertyTest | 2.3 | Yes |
| 8. 风险等级动作映射 | RiskLevelActionMappingPropertyTest | 3.4-3.6 | No |
| 9. 令牌桶流量控制 | TokenBucketFlowControlPropertyTest | 4.1, 4.4 | Yes |
| 10. 订单状态流转正确性 | OrderStatusTransitionPropertyTest | 5.1, 5.2 | Yes |
| 11. 订单状态变更幂等性 | OrderStatusIdempotencyPropertyTest | 5.4 | Yes |
| 12. 对账差异检测 | ReconciliationDiscrepancyPropertyTest | 6.5 | Yes |
| 13. 限流阈值动态生效 | RateLimitThresholdPropertyTest | 7.4 | Yes |
| 14. 活动状态自动管理 | ActivityStatusManagementPropertyTest | 8.2, 8.3 | Yes |
| 15. 限购拦截 | PurchaseLimitPropertyTest | 8.6 | Yes |

### Example Property Test

```java
@SpringBootTest
@Testcontainers
@Tag("Feature: flash-sale-system, Property 1: 库存预热Round-Trip")
class InventoryWarmupRoundTripPropertyTest {
    
    @Container
    static GenericContainer<?> redis = new GenericContainer<>("redis:7-alpine")
        .withExposedPorts(6379);
    
    @ParameterizedTest
    @MethodSource("generateTestData")
    @DisplayName("Property 1: For any SKU and stock N, warmup then query returns N")
    void warmupRoundTrip(TestData data) {
        // Arrange
        String skuId = data.skuId();
        int stock = data.stock();
        
        // Act
        inventoryService.warmupStock(skuId, stock);
        StockInfo result = inventoryService.getStock(skuId);
        
        // Assert
        assertThat(result.totalStock()).isEqualTo(stock);
    }
    
    static Stream<TestData> generateTestData() {
        Random random = new Random();
        return Stream.generate(() -> new TestData(
            "SKU" + random.nextInt(1000),
            random.nextInt(10000) + 1
        )).limit(100);
    }
}
```

### Running Property Tests

```bash
# Run all property tests (requires Docker)
./gradlew test --tests "*PropertyTest"

# Run specific property test
./gradlew test --tests "InventoryWarmupRoundTripPropertyTest"

# Run without Docker (only Property 8)
./gradlew test --tests "RiskLevelActionMappingPropertyTest"
```

## Integration Testing

### Overview

Integration tests validate end-to-end flows using real infrastructure via Testcontainers. The test suite includes 10 comprehensive scenarios covering all system requirements.

### Infrastructure

**Testcontainers** automatically manages:
- **Redis 7-alpine** - Inventory, caching, rate limiting
- **Kafka (Confluent 7.4.0)** - Message queue
- **MySQL 8.0** - Persistent storage

### Integration Test Cases

| Test # | Description | Requirements | Time |
|--------|-------------|--------------|------|
| 1 | Complete seckill flow - single user | 1.1-1.3, 2.1, 4.1, 5.1, 8.1-8.2 | ~10s |
| 2 | Concurrent requests - no overselling | 1.2, 6.1 | ~15s |
| 3 | Purchase limit enforcement | 8.5, 8.6 | ~8s |
| 4 | Order timeout and rollback | 1.5, 5.3 | ~12s |
| 5 | Data reconciliation | 6.4-6.7 | ~10s |
| 6 | Activity lifecycle management | 8.1-8.4 | ~10s |
| 7 | Kafka message flow | 2.1-2.3 | ~12s |
| 8 | Batch order creation | 2.3 | ~15s |
| 9 | Stock sold out notification | 4.6 | ~8s |
| 10 | Complete end-to-end flow | All (1.1-8.6) | ~20s |

### Example Integration Test

```java
@SpringBootTest
@Testcontainers
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class CompleteSeckillFlowIntegrationTest {
    
    @Container
    static GenericContainer<?> redis = new GenericContainer<>("redis:7-alpine")
        .withExposedPorts(6379);
    
    @Container
    static KafkaContainer kafka = new KafkaContainer(
        DockerImageName.parse("confluentinc/cp-kafka:7.4.0")
    );
    
    @Container
    static MySQLContainer<?> mysql = new MySQLContainer<>("mysql:8.0")
        .withDatabaseName("seckill_test");
    
    @Test
    @Order(1)
    @DisplayName("Test 1: Complete seckill flow - single user success")
    void testCompleteSeckillFlow() throws InterruptedException {
        // Step 1: Create activity
        ActivityCreateRequest request = new ActivityCreateRequest(
            testSkuId, "iPhone 15", 100, 
            LocalDateTime.now(), LocalDateTime.now().plusHours(1),
            5
        );
        testActivityId = activityService.createActivity(request);
        
        // Step 2: Warmup inventory
        inventoryService.warmupStock(testSkuId, 100);
        
        // Step 3: Acquire token
        QueueResult queueResult = queueService.tryAcquireToken(testSkuId, testUserId);
        assertEquals(200, queueResult.code());
        
        // Step 4: Deduct stock
        DeductResult deductResult = inventoryService.deductStock(
            testSkuId, testUserId, 1
        );
        assertEquals(DeductResultCode.SUCCESS, deductResult.code());
        
        // Step 5: Verify Kafka message and order creation
        await()
            .atMost(Duration.ofSeconds(10))
            .pollInterval(Duration.ofMillis(500))
            .untilAsserted(() -> {
                Optional<SeckillOrder> order = orderService.getOrder(orderId);
                assertTrue(order.isPresent());
                assertEquals(OrderStatus.PENDING_PAYMENT, order.get().getStatus());
            });
    }
}
```

### Running Integration Tests

```bash
# Prerequisites: Docker must be running
docker ps

# Run all integration tests
./gradlew test --tests "*IntegrationTest"

# Run specific integration test
./gradlew test --tests "CompleteSeckillFlowIntegrationTest"

# Run with detailed output
./gradlew test --tests "*IntegrationTest" --info
```

### First Run

The first run will download Docker images (~1.3 GB total):
- redis:7-alpine (~30 MB)
- confluentinc/cp-kafka:7.4.0 (~800 MB)
- mysql:8.0 (~500 MB)

Subsequent runs use cached images and start quickly.

## Coverage Requirements

### Thresholds

- **Overall coverage**: 60% minimum (unit tests only)
- **Service classes**: 70% minimum (excludes DTOs, entities, config)

### Coverage Verification

```bash
# Run tests with coverage
./gradlew test jacocoTestReport

# Verify thresholds (disabled for unit tests)
./gradlew testAll jacocoTestCoverageVerification
```

### Viewing Coverage

```bash
# Open HTML report
open build/reports/jacoco/test/html/index.html

# Check specific package
./gradlew test jacocoTestReport --info | grep "flashsale.service"
```

## Troubleshooting

### Docker Not Available

**Error**: `Could not find a valid Docker environment`

**Solution**:
1. Start Docker Desktop
2. Verify: `docker ps`
3. Check permissions (Linux): `sudo chmod 666 /var/run/docker.sock`

### Port Conflicts

**Error**: `Port already in use`

**Solution**:
```bash
# Testcontainers uses random ports by default
# If using fixed ports, stop conflicting services

# Check for zombie containers
docker ps -a

# Clean up
docker stop $(docker ps -aq)
docker rm $(docker ps -aq)
```

### Tests Fail Intermittently

**Kafka Consumer Lag**:
- Tests use Awaitility with 10-20s timeouts
- Increase timeout if needed
- Check Kafka consumer logs

**Database Constraints**:
- Tests use `create-drop` schema management
- Ensure no manual schema modifications
- Check for foreign key violations

### Slow Test Execution

```bash
# Run tests in parallel
./gradlew test --parallel

# Profile execution
./gradlew test --profile
open build/reports/profile/profile-*.html

# Run only unit tests for fast feedback
make test APP=flash-sale-service
```

### Coverage Below Threshold

```bash
# Generate detailed report
./gradlew test jacocoTestReport

# Identify uncovered code
open build/reports/jacoco/test/html/index.html

# Add tests for uncovered code
# Or adjust thresholds in build.gradle if appropriate
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Flash Sale Service Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-java@v3
        with:
          java-version: '17'
          distribution: 'temurin'
      
      - name: Run unit tests
        run: make test APP=flash-sale-service
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-java@v3
        with:
          java-version: '17'
          distribution: 'temurin'
      
      - name: Run all tests
        run: |
          cd apps/flash-sale-service
          ./gradlew testAll
```

### Test Strategy

- **PR Checks**: Run unit tests only (fast feedback)
- **Merge to Main**: Run all tests including integration
- **Nightly Builds**: Run full test suite with coverage verification
- **Release**: Run all tests + performance tests

## Resources

- [JUnit 5 User Guide](https://junit.org/junit5/docs/current/user-guide/)
- [AssertJ Documentation](https://assertj.github.io/doc/)
- [Mockito Documentation](https://javadoc.io/doc/org.mockito/mockito-core/latest/org/mockito/Mockito.html)
- [Testcontainers](https://www.testcontainers.org/)
- [Awaitility](https://github.com/awaitility/awaitility)
- [Spring Boot Testing](https://docs.spring.io/spring-boot/docs/current/reference/html/features.html#features.testing)

## Summary

The flash-sale-service has comprehensive test coverage:

- **168 unit tests** for fast feedback without Docker
- **15 property tests** validating correctness properties
- **10 integration tests** for end-to-end validation

Use `make test APP=flash-sale-service` for daily development, and `./gradlew testAll` before committing to ensure all tests pass.

---

**Last Updated**: January 30, 2025
