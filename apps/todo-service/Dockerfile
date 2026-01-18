# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/todo-service/Dockerfile -t todo-service:latest .

# Stage 1: Proto generation and build
FROM golang:1.25-alpine AS build

WORKDIR /app

# Install protoc and required tools
ARG PROTOC_VERSION=28.3
ARG PROTOC_GEN_GO_VERSION=v1.36.6
ARG PROTOC_GEN_GO_GRPC_VERSION=v1.5.1

RUN apk add --no-cache curl unzip git && \
    # Install protoc
    PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
    curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP} && \
    unzip -o ${PROTOC_ZIP} -d /usr/local bin/protoc && \
    unzip -o ${PROTOC_ZIP} -d /usr/local 'include/*' && \
    rm -f ${PROTOC_ZIP} && \
    # Install Go protoc plugins
    go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} && \
    # Verify installations
    protoc --version

# Copy go mod files
COPY apps/todo-service/go.mod apps/todo-service/go.sum ./
RUN go mod download

# Copy proto files (Single Source of Truth)
COPY api/v1 /api/v1

# Generate proto code
RUN mkdir -p gen/hellopb gen/todopb && \
    protoc --proto_path=/api/v1 \
           --go_out=./gen/hellopb --go_opt=paths=source_relative \
           --go-grpc_out=./gen/hellopb --go-grpc_opt=paths=source_relative \
           /api/v1/hello.proto && \
    protoc --proto_path=/api/v1 \
           --go_out=./gen/todopb --go_opt=paths=source_relative \
           --go-grpc_out=./gen/todopb --go-grpc_opt=paths=source_relative \
           /api/v1/todo.proto

# Copy source code
COPY apps/todo-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o todo-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/todo-service .

# Expose gRPC port
EXPOSE 9091

# Run the service
ENTRYPOINT ["./todo-service"]
