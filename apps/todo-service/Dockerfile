# Multi-stage build for Go service
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/todo-service/Dockerfile -t todo-service:latest .

# Stage 1: Build
FROM golang:1.25-alpine AS build

WORKDIR /app/apps/todo-service

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files first (for better caching)
COPY apps/todo-service/go.mod apps/todo-service/go.sum ./

# Copy shared libraries and proto code to match go.mod replace directives
# go.mod has: replace github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
# From /app/apps/todo-service, ../../api/gen/go resolves to /app/api/gen/go
COPY api/gen/ /app/api/gen/
COPY libs/ /app/libs/

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY apps/todo-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o todo-service .

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from build stage
COPY --from=build /app/apps/todo-service/todo-service .

# Expose gRPC port
EXPOSE 9091

# Run the service
ENTRYPOINT ["./todo-service"]
