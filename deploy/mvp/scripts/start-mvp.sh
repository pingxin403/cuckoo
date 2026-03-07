#!/bin/bash

# Multi-Region Active-Active MVP Startup Script
# This script starts the complete dual-region simulation environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MVP_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$MVP_DIR")")"

echo "🚀 Starting Multi-Region Active-Active MVP Environment..."
echo "📁 Project Root: $PROJECT_ROOT"
echo "📁 MVP Directory: $MVP_DIR"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command_exists docker; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command_exists docker-compose; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Check if Docker daemon is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker daemon is not running. Please start Docker first."
    exit 1
fi

echo "✅ Prerequisites check passed"

# Navigate to MVP directory
cd "$MVP_DIR"

# Clean up any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose down --remove-orphans --volumes 2>/dev/null || true

# Build and start services
echo "🏗️  Building and starting services..."
docker-compose up -d --build

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 15

# Function to wait for service health
wait_for_service() {
    local service_name=$1
    local health_url=$2
    local max_attempts=30
    local attempt=1

    echo "🔍 Checking $service_name health..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$health_url" >/dev/null 2>&1; then
            echo "✅ $service_name is healthy"
            return 0
        fi
        
        echo "⏳ Attempt $attempt/$max_attempts: $service_name not ready yet..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "❌ $service_name failed to become healthy after $max_attempts attempts"
    return 1
}

# Check service health
echo "🏥 Performing health checks..."

# Check Arbiter Mock Service
wait_for_service "Arbiter Mock" "http://localhost:9999/health"

# Check Region A Gateway
wait_for_service "Region A Gateway" "http://localhost:8080/health"

# Check Region B Gateway  
wait_for_service "Region B Gateway" "http://localhost:8081/health"

# Check Prometheus
wait_for_service "Prometheus" "http://localhost:9090/-/healthy"

# Check Grafana
wait_for_service "Grafana" "http://localhost:3000/api/health"

# Display service status
echo ""
echo "🎉 Multi-Region Active-Active MVP Environment is ready!"
echo ""
echo "📊 Service Endpoints:"
echo "  Region A WebSocket:  ws://localhost:8080/ws"
echo "  Region B WebSocket:  ws://localhost:8081/ws"
echo "  Arbiter Service:     http://localhost:9999"
echo "  Prometheus:          http://localhost:9090"
echo "  Grafana:             http://localhost:3000 (admin/admin)"
echo ""
echo "🔧 Management Commands:"
echo "  View logs:           docker-compose logs -f [service-name]"
echo "  Stop environment:    docker-compose down"
echo "  Restart service:     docker-compose restart [service-name]"
echo ""
echo "🧪 Testing Commands:"
echo "  Run chaos test:      ./scripts/chaos-test.sh"
echo "  Monitor metrics:     ./scripts/monitor.sh"
echo "  Check network:       ./scripts/network-test.sh"
echo ""

# Show container status
echo "📦 Container Status:"
docker-compose ps

echo ""
echo "✅ Startup completed successfully!"
echo "💡 Tip: Run 'docker-compose logs -f' to see real-time logs from all services"