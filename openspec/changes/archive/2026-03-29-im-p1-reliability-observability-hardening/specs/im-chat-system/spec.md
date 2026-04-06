## ADDED Requirements

### Requirement: Dependency Resilience Policy
The system SHALL apply consistent timeout, retry, and circuit-breaker policies for critical dependency calls.

#### Scenario: Dependency timeout
- **WHEN** a critical downstream call exceeds timeout budget
- **THEN** request fails with classified timeout error
- **AND** timeout metric and trace attributes are emitted

#### Scenario: Dependency instability
- **WHEN** failure rate exceeds configured threshold
- **THEN** circuit breaker opens for the dependency path
- **AND** requests follow configured fallback behavior

### Requirement: Delivery Path Observability
The system SHALL expose observability signals for message delivery and cross-gateway forwarding.

#### Scenario: Message delivery execution
- **WHEN** message is routed and delivered (local or cross-gateway)
- **THEN** metrics include path type, result, and latency buckets
- **AND** structured logs include correlation identifiers

#### Scenario: ACK lifecycle tracking
- **WHEN** ACK state transitions occur
- **THEN** metrics reflect pending/success/timeout/late-ack counts
- **AND** traces include ACK transition events

### Requirement: Reliability Alert Signals
The system SHALL emit alert-ready metrics for key reliability risks.

#### Scenario: Reliability degradation
- **WHEN** ACK timeout rate, cross-gateway failure rate, or Kafka lag crosses threshold
- **THEN** metrics expose threshold-breaching values for alerting
- **AND** runbook-relevant context is logged

### Requirement: Message Delivery Guarantee
The system SHALL classify delivery outcomes with observable failure categories while preserving at-least-once semantics.

#### Scenario: Retry and classified failure
- **WHEN** delivery retries are exhausted
- **THEN** system records classified final failure reason
- **AND** fallback path is triggered according to policy

#### Scenario: Late ACK after fallback
- **WHEN** late ACK arrives after fallback routing
- **THEN** system records late-ack event
- **AND** duplicate display is prevented by dedup strategy
