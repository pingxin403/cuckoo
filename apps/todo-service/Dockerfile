# Multi-stage build for TODO service

# Stage 1: Build
FROM golang:1.21-alpine AS build

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o todo-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/todo-service .

# Expose gRPC port
EXPOSE 9091

# Run the service
ENTRYPOINT ["./todo-service"]
