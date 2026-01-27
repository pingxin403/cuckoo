# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/shortener-service/Dockerfile -t shortener-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy centralized proto generated code
COPY api/gen/go /app/api/gen/go

# Copy go mod files
COPY apps/shortener-service/go.mod apps/shortener-service/go.sum ./
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
COPY --from=build /app/shortener-service .

# Expose ports
EXPOSE 9092 8080 9090

# Run the service
ENTRYPOINT ["./shortener-service"]
