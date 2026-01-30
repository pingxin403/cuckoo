# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/im-service/Dockerfile -t im-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

# IMPORTANT: WORKDIR must be /app/apps/im-service to match go.mod replace directives
# go.mod has: replace github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
# From /app/apps/im-service, ../../api/gen/go resolves to /app/api/gen/go
WORKDIR /app/apps/im-service

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files first (for better caching)
COPY apps/im-service/go.mod apps/im-service/go.sum ./

# Copy shared libraries and proto code to match go.mod replace directives
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/

# Download dependencies
RUN go mod download

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
COPY --from=build /app/apps/im-service/im-service .

# Expose gRPC port
EXPOSE 9094

# Expose HTTP port for health checks and metrics
EXPOSE 8080

# Run the service (includes both gRPC server and offline worker)
ENTRYPOINT ["./im-service"]
