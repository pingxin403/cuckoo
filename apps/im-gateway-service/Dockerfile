# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/im-gateway-service/Dockerfile -t im-gateway-service:latest .

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

# Copy shared libraries and API definitions (required for go mod download with replace directives)
COPY libs/ /libs/
COPY api/gen /api/gen

# Copy go mod files
COPY apps/im-gateway-service/go.mod apps/im-gateway-service/go.sum ./
RUN go mod download

# Copy proto files (Single Source of Truth)
COPY api/v1 /api/v1

# Generate proto code (adjust proto file names as needed)
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
COPY apps/im-gateway-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o im-gateway-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/im-gateway-service .

# Expose gRPC port
EXPOSE 9093

# Run the service
ENTRYPOINT ["./im-gateway-service"]
