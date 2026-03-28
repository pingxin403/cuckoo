## ADDED Requirements

### Requirement: Gateway Runtime Wiring Completeness
The system SHALL initialize and wire IM Gateway runtime dependencies before accepting production traffic.

#### Scenario: Gateway starts with required dependencies
- **WHEN** im-gateway-service starts in a standard environment
- **THEN** auth, registry, and IM clients are initialized successfully
- **AND** gateway runtime and Kafka consumer paths are started with expected configuration

#### Scenario: Gateway startup fails fast on missing critical dependencies
- **WHEN** any required dependency cannot be initialized
- **THEN** startup fails with explicit error logs
- **AND** service does not report ready state

### Requirement: Cross-Gateway Message Delivery
The system SHALL deliver messages to recipient devices connected on remote gateway nodes.

#### Scenario: Recipient devices are connected on another gateway node
- **WHEN** a private message targets a user whose active device is not local to the sender gateway
- **THEN** gateway forwards message through cross-gateway path
- **AND** remote device receives the message once

#### Scenario: Mixed local and remote devices for same user
- **WHEN** recipient has devices on local and remote gateway nodes
- **THEN** local devices receive direct push
- **AND** remote devices receive cross-gateway forwarded push

### Requirement: Cross-Gateway Read Receipt Delivery
The system SHALL deliver read receipts to sender devices across gateway nodes.

#### Scenario: Sender device is connected on remote gateway
- **WHEN** recipient marks message as read
- **THEN** read receipt is forwarded through cross-gateway path
- **AND** sender receives read receipt on remote-connected devices

### Requirement: WebSocket Origin Allowlist Enforcement
The gateway SHALL enforce configurable Origin allowlist checks on WebSocket upgrade requests.

#### Scenario: Allowed origin attempts websocket upgrade
- **WHEN** request origin matches configured allowlist
- **THEN** upgrade is accepted

#### Scenario: Disallowed origin attempts websocket upgrade
- **WHEN** request origin does not match configured allowlist
- **THEN** upgrade is rejected
- **AND** rejection reason is logged for audit

### Requirement: Message Delivery ACK State Closure
The gateway SHALL process message delivery ACK events and correlate them to pending delivery states.

#### Scenario: ACK arrives before timeout
- **WHEN** recipient ACK for a pushed message is received within timeout window
- **THEN** pending state is marked delivered
- **AND** delivery status is returned as success

#### Scenario: ACK timeout
- **WHEN** no ACK is received within configured timeout
- **THEN** pending state is marked timeout
- **AND** failure status is returned for fallback handling

## ADDED Requirements

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
