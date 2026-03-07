#!/bin/bash

# Setup Load Test Environment
# This script sets up the complete load test environment with Docker Compose.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Load Test Environment Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${YELLOW}Warning: k6 is not installed${NC}"
    echo "Please install k6: https://k6.io/docs/getting-started/installation/"
    echo ""
    echo "macOS: brew install k6"
    echo "Linux: See https://k6.io/docs/getting-started/installation/"
    echo ""
    read -p "Continue without k6? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo -e "${GREEN}✓ k6 is installed${NC}"
fi

echo ""
echo -e "${YELLOW}Starting load test environment...${NC}"
echo ""

# Stop any existing containers
echo -e "${YELLOW}Stopping existing containers...${NC}"
docker-compose -f docker-compose.loadtest.yml down -v 2>/dev/null || true

# Build the service image
echo -e "${YELLOW}Building service image...${NC}"
cd ..
docker build -t shortener-service:loadtest .
cd load_test

# Start the environment
echo -e "${YELLOW}Starting services...${NC}"
docker-compose -f docker-compose.loadtest.yml up -d

# Wait for services to be healthy
echo -e "${YELLOW}Waiting for services to be healthy...${NC}"
echo ""

# Wait for MySQL
echo -n "MySQL: "
for i in {1..30}; do
    if docker-compose -f docker-compose.loadtest.yml exec -T mysql mysqladmin ping -h localhost -u root -prootpass > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Redis
echo -n "Redis: "
for i in {1..30}; do
    if docker-compose -f docker-compose.loadtest.yml exec -T redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Shortener Service
echo -n "Shortener Service: "
for i in {1..60}; do
    if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Prometheus
echo -n "Prometheus: "
for i in {1..30}; do
    if curl -s -f http://localhost:9090/-/healthy > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

# Wait for Grafana
echo -n "Grafana: "
for i in {1..30}; do
    if curl -s -f http://localhost:3000/api/health > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ready${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Load Test Environment Ready!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Services:"
echo -e "  Shortener Service: ${GREEN}http://localhost:8080${NC}"
echo -e "  Prometheus:        ${GREEN}http://localhost:9090${NC}"
echo -e "  Grafana:           ${GREEN}http://localhost:3000${NC} (admin/admin)"
echo ""
echo -e "Next Steps:"
echo -e "  1. Run individual test: ${YELLOW}k6 run cache-stampede.js${NC}"
echo -e "  2. Run all tests:       ${YELLOW}./run-all-tests.sh${NC}"
echo -e "  3. View metrics:        ${YELLOW}open http://localhost:9090${NC}"
echo -e "  4. View dashboard:      ${YELLOW}open http://localhost:3000${NC}"
echo ""
echo -e "To stop the environment:"
echo -e "  ${YELLOW}docker-compose -f docker-compose.loadtest.yml down -v${NC}"
echo ""
