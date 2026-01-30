# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/im-gateway-service/Dockerfile -t im-gateway-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

# IMPORTANT: WORKDIR must be /app/apps/im-gateway-service to match go.mod replace directives
# go.mod has: replace github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
# From /app/apps/im-gateway-service, ../../api/gen/go resolves to /app/api/gen/go
WORKDIR /app/apps/im-gateway-service

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files first (for better caching)
COPY apps/im-gateway-service/go.mod apps/im-gateway-service/go.sum ./

# Copy shared libraries and proto code to match go.mod replace directives
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/

# Download dependencies
RUN go mod download

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
COPY --from=build /app/apps/im-gateway-service/im-gateway-service .

# Expose gRPC port
EXPOSE 9093

# Run the service
ENTRYPOINT ["./im-gateway-service"]
