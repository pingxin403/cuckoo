# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the arbiter service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o arbiter-service ./cmd/arbiter-service

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 arbiter && \
    adduser -D -s /bin/sh -u 1001 -G arbiter arbiter

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/arbiter-service .

# Change ownership
RUN chown -R arbiter:arbiter /app

# Switch to non-root user
USER arbiter

# Expose port (if running as HTTP service)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the service
CMD ["./arbiter-service"]