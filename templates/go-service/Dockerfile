# Multi-stage build for Go service

# Stage 1: Build
FROM golang:1.25-alpine AS build

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o {{SERVICE_NAME}} .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/{{SERVICE_NAME}} .

# Expose gRPC port
EXPOSE {{GRPC_PORT}}

# Run the service
ENTRYPOINT ["./{{SERVICE_NAME}}"]
