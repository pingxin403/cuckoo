# Business End-to-End Tests

## Overview

This document describes the business E2E test suite for the multi-region active-active IM system. These tests validate complete business workflows across regions, ensuring that the system meets all functional requirements from a user perspective.

## Test Coverage

The business E2E test suite (`business_e2e_test.go`) validates the following requirements:

### 1. Cross-Region Direct Message (Requirement 9.3.1)

**Test**: `TestCrossRegionDirectMessage`

**Validates**: 跨地域单聊消息的发送、接收和确认流程

**Scenario**:
- User A in Region A sends a message to User B in Region B
- Message is stored in Region A with HLC-based global ID
- Message is synchronized to Region B
- User B receives the message in Region B
- User B sends acknowledgment back to Region A
- Verify end-to-end message flow and consistency

**Key Validations**:
- ✓ Message stored correctly in source region
- ✓ Message synced to target region within acceptable latency
- ✓ Message content and metadata preserved across regions
- ✓ Acknowledgment flows back to source region
- ✓ Data consistency maintained in both regions

### 2. Cross-Region Group Chat (Requirement 9.3.2)

**Test**: `TestCrossRegionGroupChat`

**Validates**: 跨地域群聊消息的广播和排序正确性

**Scenario**:
- Create a group with members in both Region A and Region B
- Members from both regions send messages concurrently
- Messages are broadcast to all regions
- Verify message ordering using HLC timestamps
- Verify all members can see all messages

**Key Validations**:
- ✓ Group created and synced across regions
- ✓ Messages from both regions are broadcast correctly
- ✓ Message ordering preserved using HLC
- ✓ All members have consistent view of messages
- ✓ Concurrent messages handled correctly

### 3. Offline Message Push (Requirement 9.3.3)

**Test**: `TestOfflineMessagePush`

**Validates**: 离线消息在用户从不同地域上线时的推送正确性

**Scenario**:
- User is offline, messages are stored in offline queue
- Offline messages are synced to both regions
- User comes online in Region B (different from message origin)
- Offline messages are retrieved and pushed to user
- Verify message order and content
- User switches to Region A and receives new messages

**Key Validations**:
- ✓ Offline messages stored with TTL
- ✓ Offline messages synced across regions
- ✓ User can retrieve messages from any region
- ✓ Message order preserved during push
- ✓ Offline queue cleared after delivery
- ✓ User can switch regions and receive messages

### 4. Multi-Device Sync (Requirement 9.3.4)

**Test**: `TestMultiDeviceSync`

**Validates**: 多设备登录场景下消息同步的一致性

**Scenario**:
- User logs in on multiple devices (mobile, desktop, tablet) in different regions
- Message is sent to the user
- Message is synced to all device regions
- Verify all devices can see the message
- Mark message as read on one device
- Verify read status synced to all devices
- Test device offline/online scenario

**Key Validations**:
- ✓ Multiple device sessions created across regions
- ✓ Messages synced to all device regions
- ✓ All devices have consistent view of messages
- ✓ Read receipts synced across all devices
- ✓ Offline device receives messages after coming online
- ✓ Final consistency maintained across all devices

### 5. Failover Recovery (Requirements 4.1, 4.2, RPO/RTO)

**Test**: `TestFailoverRecovery`

**Validates**: 故障转移恢复后数据一致性

**Scenario**:
- Establish baseline messages in both regions
- Simulate Region A failure
- Continue operations in Region B during failover
- Verify Region B can serve all requests
- Restore Region A
- Sync failover messages to Region A
- Verify data consistency after recovery
- Verify message ordering preserved
- Validate RTO < 30 seconds
- Validate RPO ≈ 0 (no data loss)

**Key Validations**:
- ✓ Baseline data accessible in both regions
- ✓ System continues operating during region failure
- ✓ Failover region serves all requests
- ✓ Failed region recovers successfully
- ✓ Data synced after recovery
- ✓ Data consistency maintained (no data loss)
- ✓ Message ordering preserved
- ✓ RTO requirement met (< 30 seconds)
- ✓ RPO requirement met (≈ 0 data loss)

## Running the Tests

### Prerequisites

1. **Multi-region infrastructure running**:
   ```bash
   cd deploy/docker
   ./start-multi-region.sh start
   ./start-multi-region.sh test
   ```

2. **Required services**:
   - Region A: IM Service, Gateway, Redis, etcd
   - Region B: IM Service, Gateway, Redis, etcd
   - Shared: MySQL, Kafka

### Execute Tests

```bash
# Run all business E2E tests
cd tests/e2e/multi-region
go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 20m

# Run specific test
go test -v -tags=e2e -run TestBusinessEndToEndVerification/CrossRegionDirectMessage

# Run with verbose output
go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 20m -v
```

### Quick Test Script

```bash
#!/bin/bash
# tests/e2e/multi-region/run-business-tests.sh

set -e

echo "Starting multi-region infrastructure..."
cd ../../../deploy/docker
./start-multi-region.sh start
./start-multi-region.sh test

echo "Running business E2E tests..."
cd ../../tests/e2e/multi-region
go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 20m

echo "Cleaning up..."
cd ../../../deploy/docker
./start-multi-region.sh stop

echo "✓ Business E2E tests completed"
```

## Test Data Flow

### Direct Message Flow
```
User A (Region A) → Message → Region A Storage
                              ↓ Sync
                         Region B Storage → User B (Region B)
                              ↓ Ack
User A (Region A) ← Ack ← Region A Storage
```

### Group Chat Flow
```
Members (Region A) → Messages → Region A Storage
                                      ↓ Broadcast
Members (Region B) → Messages → Region B Storage
                                      ↓ Sync
                    All Members see all messages (HLC ordered)
```

### Offline Message Flow
```
Sender → Messages → Offline Queue (Region A)
                         ↓ Sync
                    Offline Queue (Region B)
                         ↓ User Online
                    Push to User (any region)
```

### Multi-Device Flow
```
Message → Region A Storage
              ↓ Sync
         Region B Storage
              ↓ Push
    Device 1 (Region A) ✓
    Device 2 (Region B) ✓
    Device 3 (Region A) ✓
```

### Failover Flow
```
Normal: Region A (Primary) ⟷ Region B (Secondary)
              ↓ Region A Fails
Failover: Region B (Active) - Region A (Failed)
              ↓ Region A Recovers
Recovery: Region A (Syncing) ← Region B (Active)
              ↓ Sync Complete
Normal: Region A (Primary) ⟷ Region B (Secondary)
```

## Expected Results

### Success Criteria

All tests should pass with the following outcomes:

✅ **Cross-Region Direct Message**
- Message delivery latency < 500ms (P99)
- Acknowledgment round-trip < 1 second
- 100% message consistency across regions

✅ **Cross-Region Group Chat**
- Message broadcast to all members
- Correct HLC-based ordering
- No message loss or duplication
- All members see consistent message history

✅ **Offline Message Push**
- All offline messages delivered
- Correct message order preserved
- TTL respected (30 days)
- Queue cleared after delivery
- Works from any region

✅ **Multi-Device Sync**
- All devices receive messages
- Read receipts synced across devices
- Offline devices catch up on reconnect
- Consistent state across all devices

✅ **Failover Recovery**
- RTO < 30 seconds
- RPO ≈ 0 (no data loss)
- Data consistency after recovery
- Message ordering preserved
- System fully operational after recovery

### Performance Metrics

| Metric | Target | Typical |
|--------|--------|---------|
| Direct message latency | < 500ms | 100-200ms |
| Group message broadcast | < 1s | 200-500ms |
| Offline message push | < 2s | 500ms-1s |
| Multi-device sync | < 500ms | 100-300ms |
| Failover detection | < 15s | 10-15s |
| Failover completion | < 30s | 20-30s |
| Recovery sync | < 60s | 30-60s |

## Troubleshooting

### Common Issues

#### 1. Test Timeout

**Symptom**: Tests timeout after 20 minutes

**Solution**:
```bash
# Increase timeout
go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 30m

# Check service health
cd deploy/docker
./start-multi-region.sh test
```

#### 2. Message Not Synced

**Symptom**: Messages not appearing in target region

**Solution**:
- Check network connectivity between regions
- Verify Redis replication is working
- Check sync latency metrics
- Review service logs for errors

#### 3. Inconsistent Data

**Symptom**: Data differs between regions

**Solution**:
- Verify HLC synchronization
- Check conflict resolution logs
- Ensure both regions are healthy
- Review sync status of messages

#### 4. Failover Test Fails

**Symptom**: Failover test doesn't complete

**Solution**:
- Verify geo router is running
- Check health check configuration
- Ensure failover is enabled
- Review failover logs

### Debug Mode

Enable verbose logging:

```bash
# Set environment variables
export E2E_DEBUG=true
export E2E_LOG_LEVEL=debug

# Run tests with debug output
go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 20m
```

### Manual Verification

Verify test data manually:

```bash
# Check messages in Region A
redis-cli -h localhost -p 6379 --scan --pattern "messages:*"

# Check messages in Region B  
redis-cli -h localhost -p 6379 --scan --pattern "messages:*"

# Check offline queues
redis-cli -h localhost -p 6379 --scan --pattern "offline:*"

# Check sessions
redis-cli -h localhost -p 6379 --scan --pattern "session:*"
```

## Integration with CI/CD

### GitHub Actions

```yaml
name: Business E2E Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  business-e2e:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Start Infrastructure
        run: |
          cd deploy/docker
          ./start-multi-region.sh start
          ./start-multi-region.sh test
      
      - name: Run Business E2E Tests
        run: |
          cd tests/e2e/multi-region
          go test -v -tags=e2e -run TestBusinessEndToEndVerification -timeout 20m
      
      - name: Cleanup
        if: always()
        run: |
          cd deploy/docker
          ./start-multi-region.sh stop
```

## Next Steps

After completing these business E2E tests:

1. **Performance Testing**: Run load tests with business scenarios
2. **Chaos Testing**: Inject failures during business operations
3. **Production Validation**: Run subset of tests in staging/production
4. **Monitoring**: Set up alerts based on test metrics
5. **Documentation**: Update operational runbooks with test insights

## References

- [Requirements Document](../../../.kiro/specs/multi-region-active-active/requirements.md)
- [Design Document](../../../.kiro/specs/multi-region-active-active/design.md)
- [Multi-Region Deployment Guide](../../../deploy/docker/MULTI_REGION_DEPLOYMENT.md)
- [End-to-End Verification Tests](./README.md)
- [Performance Tests](./performance_consistency_test.go)

## Support

For issues or questions:
1. Check this document's troubleshooting section
2. Review test output and logs
3. Check service health: `./start-multi-region.sh test`
4. Review service logs: `./start-multi-region.sh logs`
5. Consult the deployment guide
6. Open an issue with test output and logs
