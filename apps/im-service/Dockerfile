# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/im-service/Dockerfile -t im-service:latest .

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
COPY apps/im-service/go.mod apps/im-service/go.sum ./
RUN go mod download

# Copy proto files (Single Source of Truth)
COPY api/v1 /api/v1

# Generate proto code
RUN mkdir -p gen/impb gen/im_servicepb && \
    protoc --proto_path=/api/v1 \
           --go_out=./gen/impb --go_opt=paths=source_relative \
           --go-grpc_out=./gen/impb --go-grpc_opt=paths=source_relative \
           /api/v1/im.proto

# Copy source code
COPY apps/im-service/ ./

# Build the application (single binary with integrated worker)
RUN CGO_ENABLED=0 GOOS=linux go build -o im-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/im-service .

# Expose gRPC port
EXPOSE 9094

# Expose HTTP port for health checks and metrics
EXPOSE 8080

# Run the service (includes both gRPC server and offline worker)
ENTRYPOINT ["./im-service"]
