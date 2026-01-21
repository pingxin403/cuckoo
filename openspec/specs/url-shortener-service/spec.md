# URL Shortener Service

**Status**: Implemented  
**Owner**: Backend Team  
**Last Updated**: 2026-01-21

## Purpose

High-performance URL shortening microservice that transforms long URLs into short, unique identifiers and provides fast redirection. Designed to handle 500,000+ QPS for redirects with P99 latency under 10ms, featuring multi-tier caching, rate limiting, and comprehensive analytics.

## Requirements

### Requirement: Short Code Generation

The system SHALL generate unique, unpredictable short codes for long URLs.

#### Scenario: Valid URL submission
- **WHEN** a user submits a valid long URL
- **THEN** the ID Generator SHALL generate a unique 7-character short code using Base62 encoding (0-9, a-z, A-Z)
- **AND** the Shortener Service SHALL return the complete short URL within 10ms

#### Scenario: Unpredictable code generation
- **WHEN** generating a short code
- **THEN** the ID Generator SHALL ensure the code is unpredictable and non-sequential to prevent enumeration attacks

#### Scenario: Collision handling
- **WHEN** a short code collision is detected
- **THEN** the ID Generator SHALL regenerate a new code and retry until a unique code is found
- **AND** the system SHALL attempt a maximum of 3 retries

#### Scenario: URL validation
- **WHEN** a long URL is submitted
- **THEN** the Shortener Service SHALL validate that the URL is properly formatted before creating a short code

#### Scenario: Dynamic expansion
- **WHEN** collision rate exceeds 0.1%
- **THEN** the ID Generator SHALL support dynamic code length expansion from 7 to 8 characters

### Requirement: URL Mapping Storage

The system SHALL reliably store URL mappings with ACID guarantees.

#### Scenario: Mapping persistence
- **WHEN** a new URL mapping is created
- **THEN** the Storage Layer SHALL persist the mapping to MySQL with ACID guarantees
- **AND** the system SHALL wait for MySQL write confirmation before returning success

#### Scenario: Unique constraint enforcement
- **WHEN** storing URL mappings
- **THEN** the Storage Layer SHALL use short code as primary key with unique index to prevent duplicates

#### Scenario: Metadata recording
- **WHEN** storing a URL mapping
- **THEN** the Storage Layer SHALL record creation timestamp, expiration time (if set), and creator IP address

#### Scenario: Long URL support
- **WHEN** storing URL mappings
- **THEN** the Storage Layer SHALL support long URLs up to 2048 characters in length

#### Scenario: Write failure handling
- **WHEN** a database write fails
- **THEN** the Shortener Service SHALL return an error and NOT return a short code to the user

### Requirement: High-Performance Redirection

The system SHALL provide fast redirection with minimal latency.

#### Scenario: Valid short code redirect
- **WHEN** a user accesses a valid short code
- **THEN** the Redirect Service SHALL return an HTTP 302 redirect response within 10ms (P99 latency)

#### Scenario: Cache hierarchy lookup
- **WHEN** a short code is requested
- **THEN** the Redirect Service SHALL first check L1 cache (local memory) before querying L2 (Redis) or L3 (database)

#### Scenario: Cache miss backfill
- **WHEN** a cache miss occurs
- **THEN** the Redirect Service SHALL query the next cache layer and backfill the previous layers

#### Scenario: High throughput support
- **WHEN** handling redirect operations
- **THEN** the Redirect Service SHALL support a minimum throughput of 500,000 requests per second

#### Scenario: Not found response
- **WHEN** a short code is not found in any layer
- **THEN** the Redirect Service SHALL return an HTTP 404 response within 5ms

### Requirement: Multi-Tier Caching

The system SHALL implement multi-tier caching for high read throughput with minimal database load.

#### Scenario: L1 cache implementation
- **WHEN** caching URL mappings
- **THEN** the Cache Layer SHALL implement L1 caching using local in-memory storage (Ristretto) with 1-hour TTL for hot short codes

#### Scenario: L2 cache implementation
- **WHEN** caching URL mappings
- **THEN** the Cache Layer SHALL implement L2 caching using Redis Cluster with 7-day TTL for all active short codes

#### Scenario: Write-through caching
- **WHEN** a URL mapping is created
- **THEN** the Shortener Service SHALL immediately write to both Redis and MySQL

#### Scenario: Graceful degradation
- **WHEN** a cache entry expires or is invalidated
- **THEN** the Cache Layer SHALL allow graceful degradation to the next cache layer without service interruption

#### Scenario: Cache hit ratio
- **WHEN** measuring cache performance
- **THEN** the Cache Layer SHALL maintain a cache hit ratio of at least 95% for L1+L2 combined

#### Scenario: Proactive invalidation
- **WHEN** a short code is deleted or expires
- **THEN** the Redirect Service SHALL proactively invalidate all cache layers to prevent stale redirects

### Requirement: Expiration and Lifecycle Management

The system SHALL support expiration times for temporary short links.

#### Scenario: Optional expiration setting
- **WHEN** creating a short link
- **THEN** the Shortener Service SHALL accept an optional expiration time parameter (in seconds or ISO 8601 format)

#### Scenario: Expired link access
- **WHEN** a short code has expired
- **THEN** the Redirect Service SHALL return an HTTP 410 Gone response with an appropriate error message

#### Scenario: Expiration time range
- **WHEN** setting expiration times
- **THEN** the Shortener Service SHALL support expiration times ranging from 1 hour to 10 years

#### Scenario: Permanent links
- **WHEN** no expiration time is specified
- **THEN** the Shortener Service SHALL create a permanent short link with no expiration

#### Scenario: Efficient cleanup
- **WHEN** managing expired mappings
- **THEN** the Storage Layer SHALL index the expiration time field to enable efficient cleanup

### Requirement: Rate Limiting and Abuse Prevention

The system SHALL prevent abuse through rate limiting and validation.

#### Scenario: Creation rate limit
- **WHEN** a client exceeds 100 short link creation requests per minute
- **THEN** the API Gateway SHALL return an HTTP 429 Too Many Requests response

#### Scenario: Per-IP rate limiting
- **WHEN** enforcing rate limits
- **THEN** the Shortener Service SHALL implement per-IP rate limiting using a token bucket algorithm

#### Scenario: Suspicious pattern detection
- **WHEN** a suspicious pattern is detected (e.g., rapid creation of links to the same domain)
- **THEN** the Shortener Service SHALL log the activity for review

#### Scenario: Malicious URL validation
- **WHEN** validating long URLs
- **THEN** the Shortener Service SHALL validate that URLs do not contain obvious malicious patterns (e.g., javascript: protocol)

#### Scenario: Retry-After header
- **WHEN** rate limiting is triggered
- **THEN** the API Gateway SHALL include a Retry-After header indicating when the client can retry

### Requirement: Click Analytics

The system SHALL provide basic click statistics for short links.

#### Scenario: Asynchronous event recording
- **WHEN** a short link is accessed
- **THEN** the Redirect Service SHALL asynchronously record the click event to a message queue (Kafka) without blocking the redirect

#### Scenario: Non-blocking analytics
- **WHEN** recording click events
- **THEN** the Redirect Service SHALL NOT block the redirect response while recording analytics

#### Scenario: Event data capture
- **WHEN** recording a click event
- **THEN** the Redirect Service SHALL capture timestamp, short code, source IP address, and user agent

#### Scenario: Statistics API
- **WHEN** retrieving click statistics
- **THEN** the Shortener Service SHALL provide an API endpoint with a disclaimer about eventual consistency (typical delay: 1-5 seconds)

#### Scenario: Analytics failure resilience
- **WHEN** analytics recording fails
- **THEN** the Redirect Service SHALL still complete the redirect successfully

#### Scenario: Count accuracy levels
- **WHEN** providing click counts
- **THEN** the Shortener Service SHALL distinguish between "real-time approximate count" (Redis counter) and "accurate count" (batch-processed from Kafka)

### Requirement: Custom Short Codes

The system SHALL optionally support custom short codes for branded links.

#### Scenario: Custom code validation
- **WHEN** custom short code feature is enabled AND a user provides a custom code
- **THEN** the Shortener Service SHALL validate that the code is available and meets naming requirements

#### Scenario: Reserved keyword rejection
- **WHEN** custom short code feature is enabled
- **THEN** the Shortener Service SHALL reject custom codes containing profanity, reserved keywords (admin, api, health, metrics), or high-value brand names

#### Scenario: Length constraints
- **WHEN** custom short code feature is enabled
- **THEN** the Shortener Service SHALL enforce a minimum length of 4 characters and maximum of 20 characters

#### Scenario: Conflict handling
- **WHEN** custom short code feature is enabled AND a custom code conflicts with an existing code
- **THEN** the Shortener Service SHALL return an HTTP 409 Conflict error

#### Scenario: Character restrictions
- **WHEN** custom short code feature is enabled
- **THEN** the Shortener Service SHALL allow only alphanumeric characters and hyphens in custom codes

#### Scenario: Manual review
- **WHEN** custom short code feature is enabled
- **THEN** the Shortener Service SHALL implement manual review for custom codes matching sensitive patterns (e.g., payment-related terms)

### Requirement: Service Integration and API Design

The system SHALL provide a gRPC API following monorepo conventions.

#### Scenario: gRPC API definition
- **WHEN** the API is defined
- **THEN** the Shortener Service SHALL expose a gRPC API defined in Protocol Buffers following the monorepo's contract-first design pattern

#### Scenario: API Gateway integration
- **WHEN** the API is accessed
- **THEN** the Shortener Service SHALL accept requests through the API Gateway at the path /api/shortener

#### Scenario: CreateShortLink method
- **WHEN** the API is called
- **THEN** the Shortener Service SHALL provide a CreateShortLink RPC method accepting a long URL and optional parameters (expiration, custom code)

#### Scenario: GetLinkInfo method
- **WHEN** the API is called
- **THEN** the Shortener Service SHALL provide a GetLinkInfo RPC method returning metadata for a short code (creation time, expiration, click count)

#### Scenario: Error handling conventions
- **WHEN** errors occur
- **THEN** the Shortener Service SHALL follow the monorepo's error handling conventions and return structured error responses

### Requirement: High Availability and Fault Tolerance

The system SHALL remain available during partial failures.

#### Scenario: Redis unavailability
- **WHEN** Redis is unavailable
- **THEN** the Redirect Service SHALL gracefully degrade to querying MySQL directly

#### Scenario: MySQL unavailability
- **WHEN** MySQL is unavailable
- **THEN** the Shortener Service SHALL return an HTTP 503 Service Unavailable response with a retry-after header

#### Scenario: Horizontal scaling
- **WHEN** scaling the service
- **THEN** the Shortener Service SHALL support horizontal scaling with multiple instances running concurrently

#### Scenario: Instance failure handling
- **WHEN** a service instance fails
- **THEN** the API Gateway SHALL automatically route traffic to healthy instances

#### Scenario: Uptime target
- **WHEN** measuring availability
- **THEN** the Shortener Service SHALL achieve 99.99% uptime (maximum 52 minutes downtime per year)

### Requirement: Monitoring and Observability

The system SHALL provide comprehensive monitoring and alerting capabilities.

#### Scenario: Prometheus metrics
- **WHEN** monitoring the service
- **THEN** the Shortener Service SHALL expose Prometheus metrics including request rate, error rate, latency percentiles, and cache hit ratios

#### Scenario: Latency alerting
- **WHEN** P99 latency exceeds 10ms for redirect operations
- **THEN** the Shortener Service SHALL trigger an alert

#### Scenario: Cache hit ratio alerting
- **WHEN** cache hit ratio falls below 95%
- **THEN** the Shortener Service SHALL trigger an alert

#### Scenario: Structured logging
- **WHEN** logging events
- **THEN** the Shortener Service SHALL log all errors and warnings to structured logs (JSON format) for centralized log aggregation

#### Scenario: Health checks
- **WHEN** Kubernetes probes the service
- **THEN** the Shortener Service SHALL provide health check endpoints (/health and /ready) for liveness and readiness probes

#### Scenario: Distributed tracing
- **WHEN** tracking requests
- **THEN** the Shortener Service SHALL implement distributed tracing using OpenTelemetry to track request flows across components

#### Scenario: Cache stampede metrics
- **WHEN** monitoring cache behavior
- **THEN** the Shortener Service SHALL expose metrics for cache stampede events and singleflight group wait times

### Requirement: Cache Stampede Protection

The system SHALL protect against cache stampede scenarios.

#### Scenario: Request coalescing
- **WHEN** multiple concurrent requests arrive for the same expired cache key
- **THEN** the Redirect Service SHALL use singleflight pattern to coalesce requests into a single database query

#### Scenario: Multi-layer coalescing
- **WHEN** implementing request coalescing
- **THEN** the Redirect Service SHALL implement request coalescing at both L1 and L2 cache layers

#### Scenario: Stampede detection
- **WHEN** a cache stampede is detected (>10 concurrent requests for same key)
- **THEN** the Redirect Service SHALL log the event for monitoring

#### Scenario: TTL jitter
- **WHEN** setting cache TTLs
- **THEN** the Redirect Service SHALL set a random jitter (±10%) on cache TTLs to prevent synchronized expiration

#### Scenario: In-flight query waiting
- **WHEN** a database query is in-flight for a cache miss
- **THEN** the Redirect Service SHALL allow subsequent requests to wait for the result rather than issuing duplicate queries

### Requirement: Data Consistency and Durability

The system SHALL ensure URL mappings are never lost.

#### Scenario: ACID guarantees
- **WHEN** storing data
- **THEN** the Storage Layer SHALL use MySQL with InnoDB engine to provide ACID transaction guarantees

#### Scenario: Write confirmation
- **WHEN** a URL mapping is created
- **THEN** the Shortener Service SHALL wait for MySQL write confirmation before returning success to the client

#### Scenario: Replication for disaster recovery
- **WHEN** ensuring durability
- **THEN** the Storage Layer SHALL implement MySQL replication with at least one replica using semi-synchronous replication

#### Scenario: Source of truth
- **WHEN** cache and database are out of sync
- **THEN** the Redirect Service SHALL prioritize database data as the source of truth

#### Scenario: Eventual consistency
- **WHEN** synchronizing cache layers
- **THEN** the Shortener Service SHALL implement eventual consistency between cache layers with maximum staleness of 1 hour for non-expired links

#### Scenario: Database sharding
- **WHEN** write throughput exceeds 10,000 QPS
- **THEN** the Storage Layer SHALL support database sharding by short_code hash

### Requirement: Security and Input Validation

The system SHALL implement robust input validation and security controls.

#### Scenario: Protocol validation
- **WHEN** a long URL is submitted
- **THEN** the Shortener Service SHALL validate that it uses HTTP or HTTPS protocol only

#### Scenario: Length validation
- **WHEN** validating URLs
- **THEN** the Shortener Service SHALL reject URLs longer than 2048 characters

#### Scenario: Input sanitization
- **WHEN** processing user inputs
- **THEN** the Shortener Service SHALL sanitize all inputs to prevent SQL injection and XSS attacks

#### Scenario: Security headers
- **WHEN** a short code is accessed
- **THEN** the Redirect Service SHALL set appropriate security headers (X-Content-Type-Options, X-Frame-Options)

#### Scenario: Audit logging
- **WHEN** short links are created
- **THEN** the Shortener Service SHALL log all creation requests with source IP for audit and abuse investigation

#### Scenario: Threat intelligence integration
- **WHEN** validating URLs
- **THEN** the Shortener Service SHALL integrate with URL threat intelligence services to detect and block malicious URLs (phishing, malware)

#### Scenario: Malicious URL rejection
- **WHEN** a potentially malicious URL is detected
- **THEN** the Shortener Service SHALL reject the creation request and log the attempt for security review

### Requirement: Deployment and Configuration

The system SHALL integrate seamlessly with the existing monorepo infrastructure.

#### Scenario: Go service template
- **WHEN** implementing the service
- **THEN** the Shortener Service SHALL be implemented as a Go service following the monorepo's go-service template structure

#### Scenario: Service detection
- **WHEN** integrating with CI/CD
- **THEN** the Shortener Service SHALL use `.apptype` file and `metadata.yaml` for service detection and CI/CD integration

#### Scenario: Port assignment
- **WHEN** configuring the service
- **THEN** the Shortener Service SHALL be assigned port 9092 for gRPC communication

#### Scenario: Kubernetes manifests
- **WHEN** deploying to Kubernetes
- **THEN** the Shortener Service SHALL provide deployment manifests in the `k8s/` directory following monorepo conventions

#### Scenario: Environment configuration
- **WHEN** configuring the service
- **THEN** the Shortener Service SHALL support configuration via environment variables for database connection strings, Redis endpoints, and feature flags

### Requirement: Testing and Quality Assurance

The system SHALL maintain comprehensive test coverage for confident development.

#### Scenario: Test coverage targets
- **WHEN** measuring test coverage
- **THEN** the Shortener Service SHALL achieve at least 70% overall test coverage and 75% coverage for the service layer

#### Scenario: Unit tests
- **WHEN** testing business logic
- **THEN** the Shortener Service SHALL include unit tests for all core business logic (ID generation, validation, error handling)

#### Scenario: Property-based tests
- **WHEN** verifying correctness
- **THEN** the Shortener Service SHALL include property-based tests to verify correctness properties across random inputs

#### Scenario: Integration tests
- **WHEN** testing external dependencies
- **THEN** the Shortener Service SHALL include integration tests verifying interaction with Redis and MySQL

#### Scenario: Pre-commit checks
- **WHEN** committing code
- **THEN** the Shortener Service SHALL pass all pre-commit checks including linting (golangci-lint), formatting (gofmt), and security scanning

#### Scenario: Chaos engineering tests
- **WHEN** testing resilience
- **THEN** the Shortener Service SHALL include chaos engineering tests simulating Redis and MySQL failures to verify graceful degradation

#### Scenario: Load tests
- **WHEN** verifying performance
- **THEN** the Shortener Service SHALL include load tests verifying performance targets (500K QPS, P99 < 10ms) under sustained load

## References

- [Monorepo Architecture](./monorepo-architecture.md)
- [App Management System](./app-management-system.md)
- Implementation: `.kiro/specs/url-shortener-service/`
