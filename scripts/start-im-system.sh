#!/bin/bash

# Simple script to start IM Chat System

set -e

echo "=========================================="
echo "Starting IM Chat System"
echo "=========================================="
echo ""

# Step 1: Start infrastructure
echo "Step 1: Starting infrastructure (etcd, MySQL, Redis, Kafka)..."
docker compose -f deploy/docker/docker-compose.infra.yml up -d

echo "Waiting for infrastructure to be ready (30 seconds)..."
sleep 30

# Step 2: Run database migrations
echo ""
echo "Step 2: Running database migrations..."
docker compose -f deploy/docker/docker-compose.infra.yml up liquibase

# Step 3: Start services
echo ""
echo "Step 3: Starting IM services..."
docker compose -f deploy/docker/docker-compose.infra.yml \
                -f deploy/docker/docker-compose.services.yml up -d

echo ""
echo "=========================================="
echo "✓ IM Chat System Started"
echo "=========================================="
echo ""
echo "Services:"
echo "  - Auth Service:        http://localhost:9095"
echo "  - User Service:        http://localhost:9096"
echo "  - IM Service:          http://localhost:9094 (gRPC + HTTP:8080)"
echo "    * Message Router:    gRPC on port 9094"
echo "    * Offline Worker:    Background Kafka consumer"
echo "  - Gateway Service:     http://localhost:9093 (gRPC)"
echo "  - Gateway WebSocket:   ws://localhost:8082"
echo ""
echo "Infrastructure:"
echo "  - etcd:   localhost:2379"
echo "  - MySQL:  localhost:3306 (database: im_chat)"
echo "  - Redis:  localhost:6379"
echo "  - Kafka:  localhost:9092"
echo ""
echo "Check status:"
echo "  docker compose -f deploy/docker/docker-compose.infra.yml \\"
echo "                 -f deploy/docker/docker-compose.services.yml ps"
echo ""
echo "View logs:"
echo "  docker logs im-service              # Both routing and worker"
echo "  docker logs im-gateway-service"
echo ""
echo "Check worker stats:"
echo "  curl http://localhost:8080/stats    # IM Service worker metrics"
echo ""
echo "Stop all:"
echo "  docker compose -f deploy/docker/docker-compose.infra.yml \\"
echo "                 -f deploy/docker/docker-compose.services.yml down"
