# P1 Stability Acceptance Report

## Change

- `im-p1-reliability-observability-hardening`

## Scope Covered

- Reliability hardening: timeout/retry/circuit-breaker/connection reuse/error classification
- Observability hardening: delivery/cross-gateway/ACK metrics and key tracing attributes
- Quality validation: dependency jitter/timeout/fault injection recovery checks
- Alert readiness: ACK timeout, cross-gateway failure, Kafka lag

## Verification Evidence

### 1) Alert rule validation

```bash
promtool check rules deploy/docker/prometheus-alerts.yml
```

Result:
- `SUCCESS: 37 rules found`

### 2) Gateway lint

```bash
make lint APP=im-gateway-service
```

Result:
- Passed, `0 issues`

### 3) Gateway tests

```bash
make test APP=im-gateway-service
```

Result:
- Passed
- Coverage script final status: all thresholds met under integration-aware policy

### 4) Module-level unit tests for changed packages

```bash
cd apps/im-gateway-service
go test ./metrics ./service
```

Result:
- Passed

## Reliability Alert Rules Delivered (P1-3.3)

- `HighAckTimeoutRate`
  - Expression based on `im_gateway_ack_timeouts_total / im_gateway_messages_delivered_total`
- `HighCrossGatewayForwardFailureRate`
  - Expression based on `im_gateway_cross_gateway_forward_total{result="failure"}` share
- `HighKafkaConsumerLag`
  - Expression based on `max(im_gateway_kafka_consumer_lag{topic=~"group_msg|read_receipt|membership_change"})`

## Documentation Updated (P1-4.1)

- `apps/im-gateway-service/metrics/README.md`
  - Added ACK lifecycle metrics (`pending/success/late`)
  - Added cross-gateway forwarding metrics and query examples
  - Added Kafka consumer lag/error metrics and query examples
- `apps/im-gateway-service/DEPLOYMENT.md`
  - Added P1 alert thresholds for cross-gateway failure and Kafka lag
  - Added consolidated PromQL snippets for three P1 reliability alerts

## Acceptance Decision

- P1 reliability/observability hardening acceptance criteria are satisfied for implemented scope.
- Remaining task state in `tasks.md`: only `4.2` transitions to done with this report.
