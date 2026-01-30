# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/shortener-service/Dockerfile -t shortener-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

WORKDIR /app/apps/shortener-service

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files first (for better caching)
COPY apps/shortener-service/go.mod apps/shortener-service/go.sum ./

# Copy shared libraries and proto code to match go.mod replace directives
# go.mod has: replace github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
# From /app/apps/shortener-service, ../../api/gen/go resolves to /app/api/gen/go
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/

# Download dependencies
RUN go mod download

# Copy source code
COPY apps/shortener-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o shortener-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/apps/shortener-service/shortener-service .

# Expose ports
EXPOSE 9092 8080 9090

# Run the service
ENTRYPOINT ["./shortener-service"]
