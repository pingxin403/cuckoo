# Multi-stage build for arbiter mock service
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY tools/arbiter-mock/ ./tools/arbiter-mock/
COPY libs/ ./libs/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o arbiter-mock ./tools/arbiter-mock/

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/arbiter-mock .

# Expose port
EXPOSE 9999

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:9999/health || exit 1

# Run the binary
CMD ["./arbiter-mock"]