## ADDED Requirements

### Requirement: Performance Baseline Governance
The system SHALL maintain reproducible performance baselines for IM critical paths before and after optimization.

#### Scenario: Baseline capture for critical paths
- **WHEN** performance governance is executed for a release cycle
- **THEN** baselines include throughput, P95/P99 latency, timeout rate, and error rate
- **AND** baselines are captured for private message, group message, cross-gateway delivery, and read-receipt paths

#### Scenario: Optimization without baseline is blocked
- **WHEN** a performance optimization proposal lacks baseline evidence
- **THEN** optimization is not approved for rollout
- **AND** evidence requirements are reported explicitly

### Requirement: Hot Path Optimization Validation
The system SHALL validate optimization impact on hotspot paths using benchmark and regression thresholds.

#### Scenario: Hot path optimization applied
- **WHEN** optimization is applied to group broadcast, cross-gateway forwarding, or ACK state management
- **THEN** benchmark results show measurable improvement against baseline
- **AND** no critical reliability regression is introduced

#### Scenario: Optimization causes reliability regression
- **WHEN** benchmark improvement exists but reliability metrics degrade beyond threshold
- **THEN** optimization is rejected or rolled back
- **AND** remediation actions are tracked

### Requirement: Capacity and Rollout Gate
The system SHALL enforce staging performance gates before production rollout.

#### Scenario: Staging gate passed
- **WHEN** staging load and failure-injection validation meet predefined thresholds
- **THEN** release is eligible for production rollout

#### Scenario: Staging gate failed
- **WHEN** any key threshold (latency, timeout, error, delivery success) is not met
- **THEN** production rollout is blocked
- **AND** failure report includes bottleneck and rollback plan

## MODIFIED Requirements

### Requirement: Group Chat Messaging
The system SHALL support scalable group broadcast under high concurrency with defined performance SLO guardrails.

#### Scenario: High-concurrency group broadcast
- **WHEN** group broadcast runs under high load conditions
- **THEN** system sustains target throughput within defined latency/error budgets
- **AND** degradation behavior is observable and bounded

### Requirement: Message Delivery Guarantee
The system SHALL preserve delivery guarantee semantics while applying performance optimizations.

#### Scenario: Optimized delivery path
- **WHEN** optimized path is active
- **THEN** at-least-once delivery semantics remain intact
- **AND** dedup behavior remains consistent with existing guarantees
