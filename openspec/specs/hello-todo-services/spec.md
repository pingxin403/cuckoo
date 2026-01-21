# Hello and TODO Services

**Status**: Implemented  
**Owner**: Platform Team  
**Last Updated**: 2026-01-21

## Purpose

Multi-language microservices providing greeting functionality (Hello Service in Java/Spring Boot) and task management (TODO Service in Go). These services demonstrate the monorepo's capability to support heterogeneous technology stacks with unified API contracts using Protocol Buffers.

## Requirements

### Requirement: Monorepo Project Structure

The system SHALL provide a standardized monorepo structure that supports multiple programming languages and services in a single repository.

#### Scenario: Directory structure initialization
- **WHEN** the monorepo is initialized
- **THEN** the system SHALL create root directories including `api/`, `apps/`, `libs/`, `tools/`, and `scripts/`
- **AND** the system SHALL create subdirectories for web (React), hello-service (Java), and todo-service (Go) under `apps/`
- **AND** the system SHALL configure build tool root configuration files
- **AND** the system SHALL create `.gitignore` excluding build artifacts (target/, bin/, node_modules/)
- **AND** the system SHALL create CODEOWNERS file defining code ownership per directory

#### Scenario: Multi-language support
- **WHEN** services in different languages are added to the monorepo
- **THEN** the system SHALL support unified build and dependency management for Java, Go, and TypeScript
- **AND** the system SHALL enable independent building, testing, and deployment of each service

### Requirement: Java Hello Service

The system SHALL provide a greeting service implemented in Java using Spring Boot and gRPC.

#### Scenario: Service initialization
- **WHEN** the Hello Service is initialized
- **THEN** the system SHALL create a Maven or Gradle project in `apps/hello-service/`
- **AND** the system SHALL configure Spring Boot and gRPC server dependencies
- **AND** the system SHALL create main application class and configuration files
- **AND** the system SHALL configure Protobuf and gRPC plugins

#### Scenario: Service startup
- **WHEN** the Hello Service starts
- **THEN** the system SHALL listen on port 9090
- **AND** the system SHALL output startup logs confirming successful initialization

#### Scenario: Greeting with name
- **WHEN** a user sends a Hello request containing a name
- **THEN** the Hello Service SHALL return a greeting message including that name

#### Scenario: Greeting without name
- **WHEN** a user sends a Hello request with an empty name
- **THEN** the Hello Service SHALL return a default greeting message

#### Scenario: gRPC integration
- **WHEN** the Hello Service is running
- **THEN** the service SHALL implement the Protobuf-defined Hello service interface
- **AND** the service SHALL be registered in Spring Boot gRPC framework
- **AND** the web app SHALL be able to call the service through gRPC-Web or REST proxy

### Requirement: Go TODO Service

The system SHALL provide a task management service implemented in Go with gRPC support.

#### Scenario: Service initialization
- **WHEN** the TODO Service is initialized
- **THEN** the system SHALL create a Go module with go.mod in `apps/todo-service/`
- **AND** the system SHALL create main.go entry file
- **AND** the system SHALL configure gRPC server framework
- **AND** the system SHALL include necessary dependencies (gRPC, Protobuf runtime)

#### Scenario: Service startup
- **WHEN** the TODO Service starts
- **THEN** the system SHALL listen on port 9091
- **AND** the system SHALL output startup logs confirming successful initialization

#### Scenario: Create TODO item
- **WHEN** a user creates a new TODO item
- **THEN** the TODO Service SHALL save the item
- **AND** the service SHALL return a unique identifier for the item

#### Scenario: List TODO items
- **WHEN** a user queries the TODO list
- **THEN** the TODO Service SHALL return all TODO items

#### Scenario: Update TODO item
- **WHEN** a user updates a TODO item
- **THEN** the TODO Service SHALL modify the corresponding item
- **AND** the service SHALL return the update result

#### Scenario: Delete TODO item
- **WHEN** a user deletes a TODO item
- **THEN** the TODO Service SHALL remove the corresponding item
- **AND** the service SHALL return the deletion result

#### Scenario: Data persistence
- **WHEN** TODO items are managed
- **THEN** the TODO Service SHALL use in-memory storage or simple persistence mechanism
- **AND** the service SHALL implement the Protobuf-defined TODO service interface

### Requirement: React Frontend Application

The system SHALL provide a web-based user interface for interacting with Hello and TODO services.

#### Scenario: Frontend initialization
- **WHEN** the web application is initialized
- **THEN** the system SHALL create a React + TypeScript project in `apps/web/`
- **AND** the system SHALL configure package.json with necessary dependencies
- **AND** the system SHALL create basic project structure (src/, public/)
- **AND** the system SHALL configure TypeScript compilation options

#### Scenario: Development server
- **WHEN** the web app development server starts
- **THEN** the system SHALL display the default page in a browser

#### Scenario: Hello service interaction
- **WHEN** the web app is running
- **THEN** the system SHALL provide a Hello service interface with input field and submit button
- **AND** WHEN a user enters a name and submits
- **THEN** the system SHALL display the greeting message returned by the service

#### Scenario: TODO list display
- **WHEN** the web app is running
- **THEN** the system SHALL provide a TODO list display interface
- **AND** the system SHALL provide an input interface for creating new TODO items
- **AND** the system SHALL provide update and delete buttons for each TODO item

#### Scenario: Real-time updates
- **WHEN** a user performs TODO operations
- **THEN** the web app SHALL update the interface display in real-time

#### Scenario: Error handling
- **WHEN** service calls fail
- **THEN** the web app SHALL display user-friendly error messages

### Requirement: Protobuf API Contracts

The system SHALL define unified API contracts using Protocol Buffers for consistent interface specifications.

#### Scenario: Contract definition
- **WHEN** API contracts are defined
- **THEN** the system SHALL create Protobuf definition files in `api/v1/`
- **AND** the system SHALL define Hello service interfaces (request and response messages)
- **AND** the system SHALL define TODO service interfaces (CRUD operation message types)
- **AND** the system SHALL use gRPC service definitions to declare service methods
- **AND** the system SHALL include clear comments explaining each message and service

### Requirement: Code Generation

The system SHALL automatically generate code from Protobuf definitions for Java, Go, and TypeScript.

#### Scenario: Java code generation
- **WHEN** Protobuf files are processed
- **THEN** the system SHALL generate Java code using protoc-gen-grpc-java
- **AND** the system SHALL output generated code to Maven/Gradle recognizable paths

#### Scenario: Go code generation
- **WHEN** Protobuf files are processed
- **THEN** the system SHALL generate Go code using protoc-gen-go and protoc-gen-go-grpc
- **AND** the system SHALL output generated code to Go module importable paths

#### Scenario: TypeScript code generation
- **WHEN** Protobuf files are processed
- **THEN** the system SHALL generate TypeScript code using ts-proto or grpc-web
- **AND** the system SHALL output generated code to frontend project importable paths

#### Scenario: Regeneration on changes
- **WHEN** Protobuf files are modified
- **THEN** the system SHALL regenerate corresponding code automatically

### Requirement: Service Communication and API Gateway

The system SHALL provide unified API gateway or proxy layer for frontend access to multiple backend services.

#### Scenario: Unified entry point
- **WHEN** the API gateway is configured
- **THEN** the system SHALL provide a unified entry point for frontend access
- **AND** the system SHALL route Hello service requests to Java server
- **AND** the system SHALL route TODO service requests to Go server

#### Scenario: Protocol support
- **WHEN** the API gateway handles requests
- **THEN** the system SHALL support gRPC-Web protocol or provide REST to gRPC conversion
- **AND** the system SHALL handle CORS configuration for cross-origin frontend access

#### Scenario: Proxy configuration
- **WHEN** using Envoy or Nginx as proxy
- **THEN** the system SHALL configure appropriate routing rules

### Requirement: Build and Development Workflow

The system SHALL provide efficient build and development workflows for rapid iteration and testing.

#### Scenario: Full project build
- **WHEN** building the entire project
- **THEN** the system SHALL provide commands to build Java, Go, and React components

#### Scenario: Individual service build
- **WHEN** building specific services
- **THEN** the system SHALL support building frontend, Hello service, or TODO service independently

#### Scenario: Incremental build
- **WHEN** code changes are made
- **THEN** the system SHALL support incremental builds, rebuilding only changed components

#### Scenario: Development mode
- **WHEN** running in development mode
- **THEN** the system SHALL support frontend hot reload
- **AND** the system SHALL provide scripts to start all services simultaneously

#### Scenario: Code quality
- **WHEN** code is committed
- **THEN** the system SHALL provide code formatting and lint checking tools for Java, Go, and TypeScript

#### Scenario: Containerization
- **WHEN** deploying services
- **THEN** the system SHALL support Docker containerized builds and deployments

### Requirement: Documentation and Examples

The system SHALL provide clear documentation and examples for new developers to quickly understand the project.

#### Scenario: Project documentation
- **WHEN** documentation is created
- **THEN** the system SHALL create README.md in root directory explaining project structure and quick start steps
- **AND** the system SHALL include instructions for running development servers (Java, Go, React)
- **AND** the system SHALL include instructions for building production versions
- **AND** the system SHALL include guidelines for adding new services supporting multiple languages
- **AND** the system SHALL provide Protobuf definition examples and comments in `api/` directory
- **AND** the system SHALL provide architecture diagrams and explanations for multi-language service coexistence

### Requirement: Extensibility and Governance

The system SHALL establish clear extension and governance mechanisms for healthy project scaling.

#### Scenario: New service addition
- **WHEN** developers add new services
- **THEN** the system SHALL provide standard process documentation
- **AND** the system SHALL require creating independent directories under `apps/`
- **AND** the system SHALL require defining Protobuf in `api/` directory first when adding new APIs

#### Scenario: Code quality enforcement
- **WHEN** code is committed
- **THEN** the system SHALL configure pre-commit hooks to check code format and basic standards
- **AND** the system SHALL explain code ownership and review processes in README

#### Scenario: Repository health
- **WHEN** monitoring repository health
- **THEN** the system SHALL provide suggestions for monitoring metrics (build time, code duplication rate)

## References

- [Monorepo Architecture](./monorepo-architecture.md)
- [App Management System](./app-management-system.md)
- Implementation: `.kiro/specs/monorepo-hello-todo/`
