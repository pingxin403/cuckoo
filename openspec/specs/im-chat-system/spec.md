# im-chat-system Specification

## Purpose
TBD - created by archiving change implement-im-p0-delivery-security-closure. Update Purpose after archive.
## Requirements
### Requirement: Group Chat Messaging
The System SHALL route group messages through Kafka-based broadcast and report delivery summary from available membership context.

#### Scenario: Group message publication
- **WHEN** a user sends a group message
- **THEN** IM Service publishes the message to `group_msg` topic
- **AND** publication failures are surfaced with error status and logs

#### Scenario: Group delivery summary reporting
- **WHEN** group message routing completes
- **THEN** IM Service returns available online/offline delivery summary
- **AND** summary source and limitations are observable in logs

### Requirement: Message Status Query
The System SHALL provide minimum viable message status query for routed messages.

#### Scenario: Status query for known message
- **WHEN** client queries status for a tracked message id
- **THEN** system returns known status state and timestamp metadata if available

#### Scenario: Status query for unknown message
- **WHEN** client queries status for an unknown message id
- **THEN** system returns not-found or pending-compatible response without server error

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

