# IM Chat System

**Status**: In Development  
**Owner**: Platform Team  
**Last Updated**: 2026-02-01

## Purpose

Real-time instant messaging system providing private and group chat capabilities with offline message delivery, read receipts, and multi-device synchronization. The system demonstrates scalable microservices architecture with WebSocket connections, message routing, and persistent storage.

## Requirements

### Requirement: IM Service Architecture

The system SHALL provide a distributed instant messaging architecture with message routing, offline storage, and real-time delivery capabilities.

#### Scenario: Service initialization
- **WHEN** the IM system is initialized
- **THEN** the system SHALL create IM Service (`apps/im-service/`) for message routing and offline persistence
- **AND** the system SHALL create IM Gateway Service (`apps/im-gateway-service/`) for WebSocket connections
- **AND** the system SHALL create Auth Service (`apps/auth-service/`) for user authentication
- **AND** the system SHALL create User Service (`apps/user-service/`) for user management
- **AND** the system SHALL configure MySQL database for message persistence
- **AND** the system SHALL configure Redis for session state and caching
- **AND** the system SHALL configure Kafka for asynchronous message processing
- **AND** the system SHALL configure etcd for service discovery and coordination

#### Scenario: Service communication
- **WHEN** services communicate
- **THEN** the system SHALL use gRPC for inter-service communication
- **AND** the system SHALL use WebSocket for client connections
- **AND** the system SHALL use Kafka for asynchronous message queuing
- **AND** the system SHALL use etcd for service registry and health checks

### Requirement: Real-time Message Routing

The system SHALL provide efficient message routing with support for both online and offline delivery.

#### Scenario: Private message routing - online recipient
- **WHEN** a user sends a private message to an online recipient
- **THEN** the IM Service SHALL lookup the recipient's gateway location in etcd registry
- **AND** the system SHALL route the message directly to the recipient's WebSocket connection
- **AND** the system SHALL assign a sequence number for message ordering
- **AND** the system SHALL return delivery confirmation to the sender
- **AND** the system SHALL complete delivery within 100ms for optimal user experience

#### Scenario: Private message routing - offline recipient
- **WHEN** a user sends a private message to an offline recipient
- **THEN** the IM Service SHALL detect the recipient is offline via registry lookup
- **AND** the system SHALL publish the message to Kafka offline_msg topic
- **AND** the system SHALL store the message in MySQL offline_messages table
- **AND** the system SHALL return offline delivery status to the sender
- **AND** the system SHALL deliver the message when recipient comes online

#### Scenario: Group message routing
- **WHEN** a user sends a group message
- **THEN** the IM Service SHALL publish the message to Kafka group_msg topic
- **AND** the system SHALL fan out the message to all group members
- **AND** the system SHALL handle online members via direct WebSocket delivery
- **AND** the system SHALL handle offline members via offline message storage
- **AND** the system SHALL maintain consistent sequence numbers across all members

#### Scenario: Message deduplication
- **WHEN** duplicate messages are received
- **THEN** the system SHALL use Redis-based deduplication to prevent duplicate processing
- **AND** the system SHALL maintain deduplication cache with configurable TTL
- **AND** the system SHALL return success for already-processed messages

### Requirement: WebSocket Gateway Management

The system SHALL provide scalable WebSocket connection management with session state synchronization.

#### Scenario: WebSocket connection establishment
- **WHEN** a client establishes a WebSocket connection
- **THEN** the Gateway Service SHALL authenticate the user via Auth Service
- **AND** the system SHALL register the connection in etcd service registry
- **AND** the system SHALL store session state in Redis for multi-device support
- **AND** the system SHALL assign the connection to a specific gateway instance
- **AND** the system SHALL enable message routing to this connection

#### Scenario: Multi-device session management
- **WHEN** a user connects from multiple devices
- **THEN** the system SHALL maintain separate WebSocket connections for each device
- **AND** the system SHALL synchronize session state across all devices via Redis
- **AND** the system SHALL deliver messages to all active connections
- **AND** the system SHALL handle device-specific read receipts and presence

#### Scenario: Connection health monitoring
- **WHEN** monitoring WebSocket connections
- **THEN** the system SHALL implement heartbeat/ping-pong mechanism
- **AND** the system SHALL detect and clean up stale connections
- **AND** the system SHALL update etcd registry when connections are lost
- **AND** the system SHALL gracefully handle connection failures and reconnections

### Requirement: Offline Message System

The system SHALL provide reliable offline message storage and delivery with configurable retention policies.

#### Scenario: Offline message storage
- **WHEN** messages are stored for offline users
- **THEN** the system SHALL use Kafka offline worker to batch process messages
- **AND** the system SHALL store messages in MySQL with expiration timestamps
- **AND** the system SHALL support configurable message TTL (default 7 days)
- **AND** the system SHALL handle storage failures with retry mechanisms
- **AND** the system SHALL provide batch insertion for performance optimization

#### Scenario: Offline message retrieval
- **WHEN** an offline user comes online
- **THEN** the system SHALL retrieve messages using cursor-based pagination
- **AND** the system SHALL deliver messages in sequence number order
- **AND** the system SHALL support incremental message loading
- **AND** the system SHALL mark messages as delivered after successful transmission
- **AND** the system SHALL handle large message backlogs efficiently

#### Scenario: Message expiration and cleanup
- **WHEN** managing message lifecycle
- **THEN** the system SHALL automatically delete expired messages
- **AND** the system SHALL run cleanup tasks in configurable intervals
- **AND** the system SHALL provide metrics on storage usage and cleanup operations
- **AND** the system SHALL support manual message deletion for GDPR compliance

### Requirement: Sequence Number Management

The system SHALL provide consistent message ordering using distributed sequence generation.

#### Scenario: Sequence number generation
- **WHEN** messages require sequence numbers
- **THEN** the system SHALL use Redis-based sequence generator for each conversation
- **AND** the system SHALL ensure monotonic increasing sequences per conversation
- **AND** the system SHALL support both private and group conversation sequences
- **AND** the system SHALL handle sequence generation failures with appropriate error handling

#### Scenario: Message ordering consistency
- **WHEN** messages are delivered
- **THEN** the system SHALL maintain consistent ordering across all participants
- **AND** the system SHALL use sequence numbers for client-side message sorting
- **AND** the system SHALL handle out-of-order delivery scenarios
- **AND** the system SHALL provide sequence gap detection for message integrity

### Requirement: Read Receipt System

The system SHALL provide read receipt tracking with privacy controls and efficient delivery.

#### Scenario: Read receipt generation
- **WHEN** a user reads a message
- **THEN** the system SHALL generate a read receipt event
- **AND** the system SHALL publish the event to Kafka read_receipt_events topic
- **AND** the system SHALL update read status in MySQL database
- **AND** the system SHALL notify the message sender if read receipts are enabled

#### Scenario: Read receipt privacy
- **WHEN** handling read receipts
- **THEN** the system SHALL respect user privacy settings for read receipt visibility
- **AND** the system SHALL allow users to disable read receipt sending
- **AND** the system SHALL allow users to disable read receipt receiving
- **AND** the system SHALL provide granular controls per conversation

#### Scenario: Bulk read operations
- **WHEN** users mark conversations as read
- **THEN** the system SHALL support bulk read receipt updates
- **AND** the system SHALL efficiently update multiple message read statuses
- **AND** the system SHALL provide conversation-level read status tracking
- **AND** the system SHALL optimize database operations for large conversations

### Requirement: User Registry and Service Discovery

The system SHALL provide efficient user location tracking and service discovery using etcd.

#### Scenario: User registration
- **WHEN** users connect to gateway services
- **THEN** the system SHALL register user location in etcd with TTL
- **AND** the system SHALL include gateway instance information
- **AND** the system SHALL support multiple concurrent connections per user
- **AND** the system SHALL automatically clean up stale registrations

#### Scenario: User lookup for message routing
- **WHEN** routing messages to users
- **THEN** the system SHALL query etcd for user gateway locations
- **AND** the system SHALL handle users connected to multiple gateways
- **AND** the system SHALL provide fast lookup with sub-millisecond response times
- **AND** the system SHALL cache frequently accessed user locations

#### Scenario: Service health monitoring
- **WHEN** monitoring service health
- **THEN** the system SHALL use etcd for service health checks and coordination
- **AND** the system SHALL implement distributed health checking
- **AND** the system SHALL provide service discovery for inter-service communication
- **AND** the system SHALL handle service failures and automatic failover

### Requirement: Message Filtering and Content Moderation

The system SHALL provide configurable content filtering and sensitive word detection.

#### Scenario: Sensitive word filtering
- **WHEN** messages contain sensitive content
- **THEN** the system SHALL apply configurable filtering rules
- **AND** the system SHALL support multiple filtering actions (block, replace, audit)
- **AND** the system SHALL maintain language-specific word lists
- **AND** the system SHALL provide audit logs for filtered content
- **AND** the system SHALL allow administrative override and whitelist management

#### Scenario: Content moderation policies
- **WHEN** applying content moderation
- **THEN** the system SHALL support configurable moderation policies per group/user
- **AND** the system SHALL provide real-time content scanning
- **AND** the system SHALL integrate with external moderation services if needed
- **AND** the system SHALL maintain moderation audit trails for compliance

### Requirement: Observability and Monitoring

The system SHALL provide comprehensive observability with metrics, logging, and tracing.

#### Scenario: Metrics collection
- **WHEN** the system is running
- **THEN** the system SHALL collect metrics on message throughput, latency, and error rates
- **AND** the system SHALL monitor WebSocket connection counts and health
- **AND** the system SHALL track offline message queue depths and processing rates
- **AND** the system SHALL provide business metrics on user activity and engagement
- **AND** the system SHALL export metrics in Prometheus format

#### Scenario: Distributed tracing
- **WHEN** processing messages
- **THEN** the system SHALL provide distributed tracing across all services
- **AND** the system SHALL trace message flow from client to delivery
- **AND** the system SHALL include timing information for performance analysis
- **AND** the system SHALL support trace sampling for production environments

#### Scenario: Structured logging
- **WHEN** logging system events
- **THEN** the system SHALL use structured JSON logging
- **AND** the system SHALL include correlation IDs for request tracking
- **AND** the system SHALL provide configurable log levels per service
- **AND** the system SHALL support centralized log aggregation

### Requirement: Security and Authentication

The system SHALL provide secure authentication and authorization with JWT tokens and TLS encryption.

#### Scenario: User authentication
- **WHEN** users authenticate
- **THEN** the Auth Service SHALL validate credentials and issue JWT tokens
- **AND** the system SHALL support token refresh mechanisms
- **AND** the system SHALL validate tokens on all API requests
- **AND** the system SHALL provide secure token storage recommendations

#### Scenario: Message encryption
- **WHEN** handling sensitive messages
- **THEN** the system SHALL support optional end-to-end encryption
- **AND** the system SHALL provide key management for encrypted conversations
- **AND** the system SHALL ensure encrypted messages remain encrypted at rest
- **AND** the system SHALL support forward secrecy for enhanced security

#### Scenario: Transport security
- **WHEN** transmitting data
- **THEN** the system SHALL use TLS for all external communications
- **AND** the system SHALL encrypt WebSocket connections
- **AND** the system SHALL secure inter-service gRPC communications
- **AND** the system SHALL validate certificates and prevent man-in-the-middle attacks

### Requirement: Performance and Scalability

The system SHALL support high throughput and horizontal scaling with efficient resource utilization.

#### Scenario: Message throughput
- **WHEN** handling high message volumes
- **THEN** the system SHALL support 10,000+ messages per second per service instance
- **AND** the system SHALL maintain sub-100ms message delivery latency
- **AND** the system SHALL efficiently batch database operations
- **AND** the system SHALL use connection pooling and resource optimization

#### Scenario: Horizontal scaling
- **WHEN** scaling the system
- **THEN** the system SHALL support adding multiple instances of each service
- **AND** the system SHALL distribute load across service instances
- **AND** the system SHALL maintain session affinity where required
- **AND** the system SHALL provide stateless service design for easy scaling

#### Scenario: Resource optimization
- **WHEN** optimizing resource usage
- **THEN** the system SHALL implement efficient memory management for WebSocket connections
- **AND** the system SHALL use Redis clustering for session state scaling
- **AND** the system SHALL optimize database queries and indexing
- **AND** the system SHALL provide resource usage monitoring and alerting

### Requirement: Data Consistency and Reliability

The system SHALL ensure data consistency and reliability across distributed components.

#### Scenario: Message delivery guarantees
- **WHEN** delivering messages
- **THEN** the system SHALL provide at-least-once delivery guarantees
- **AND** the system SHALL use idempotent message processing
- **AND** the system SHALL handle duplicate detection and elimination
- **AND** the system SHALL provide delivery confirmation mechanisms

#### Scenario: Database consistency
- **WHEN** storing message data
- **THEN** the system SHALL use database transactions for consistency
- **AND** the system SHALL handle concurrent access with appropriate locking
- **AND** the system SHALL provide data backup and recovery procedures
- **AND** the system SHALL maintain referential integrity across related data

#### Scenario: Failure recovery
- **WHEN** handling system failures
- **THEN** the system SHALL implement graceful degradation strategies
- **AND** the system SHALL provide automatic retry mechanisms with exponential backoff
- **AND** the system SHALL maintain service availability during partial failures
- **AND** the system SHALL provide manual recovery procedures for critical failures

## Implementation Notes

### Technology Stack
- **Backend Services**: Go with gRPC and Protocol Buffers
- **Database**: MySQL for persistent storage, Redis for caching and sessions
- **Message Queue**: Apache Kafka for asynchronous processing
- **Service Discovery**: etcd for coordination and health checks
- **WebSocket**: Gorilla WebSocket for real-time connections
- **Observability**: OpenTelemetry with Prometheus metrics

### Service Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │  Mobile Client  │    │  Desktop Client │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │    IM Gateway Service   │
                    │    (WebSocket + gRPC)   │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │      IM Service         │
                    │   (Message Routing)     │
                    └────────────┬────────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
    ┌─────▼─────┐         ┌─────▼─────┐         ┌─────▼─────┐
    │   MySQL   │         │   Redis   │         │   Kafka   │
    │(Messages) │         │(Sessions) │         │ (Queue)   │
    └───────────┘         └───────────┘         └───────────┘
```

### Database Schema
- **offline_messages**: Message storage with TTL and indexing
- **read_receipts**: Read status tracking per user per message
- **user_sessions**: Session state and device information
- **conversation_sequences**: Sequence number management per conversation

### API Contracts
- **IM Service**: `api/v1/im.proto` - Message routing and delivery
- **IM Gateway**: `api/v1/im-gateway.proto` - WebSocket and connection management
- **Auth Service**: `api/v1/auth.proto` - Authentication and authorization
- **User Service**: `api/v1/user.proto` - User management and profiles

## References

- [IM System Architecture](../../docs/architecture/IM_CHAT_SYSTEM.md)
- [Message Routing Design](../../apps/im-service/README.md)
- [WebSocket Gateway Design](../../apps/im-gateway-service/README.md)
- Implementation: `apps/im-service/`, `apps/im-gateway-service/`, `apps/auth-service/`, `apps/user-service/`