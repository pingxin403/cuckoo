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

