# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/auth-service/Dockerfile -t auth-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

WORKDIR /app

# Install git (required for go mod download)
RUN apk add --no-cache git

# Copy shared libraries and API generated code
COPY libs/ ./libs/
COPY api/gen/go/ ./api/gen/go/

# Copy go mod files
COPY apps/auth-service/go.mod apps/auth-service/go.sum ./apps/auth-service/
WORKDIR /app/apps/auth-service
RUN go mod download

# Copy source code
COPY apps/auth-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/auth-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/bin/auth-service .

# Expose gRPC port
EXPOSE 9095

# Run the service
ENTRYPOINT ["./auth-service"]
