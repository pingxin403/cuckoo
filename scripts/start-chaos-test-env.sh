#!/bin/bash

# Quick startup script for chaos testing environment
# This script starts the infrastructure and a service for testing

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Chaos Test Environment Setup${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Step 1: Start infrastructure
echo -e "${BLUE}[1/4]${NC} Starting infrastructure (MySQL, Redis)..."
cd deploy/docker
docker compose -f docker-compose.infra.yml up -d mysql redis
cd ../..

# Step 2: Wait for infrastructure to be healthy
echo -e "${BLUE}[2/4]${NC} Waiting for infrastructure to be healthy..."
echo "Waiting for MySQL..."
for i in {1..30}; do
    if docker compose -f deploy/docker/docker-compose.infra.yml exec -T mysql mysqladmin ping -h localhost -u root -proot_password &>/dev/null; then
        echo -e "${GREEN}✓${NC} MySQL is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗${NC} MySQL failed to start"
        exit 1
    fi
    sleep 1
done

echo "Waiting for Redis..."
for i in {1..30}; do
    if docker compose -f deploy/docker/docker-compose.infra.yml exec -T redis redis-cli ping &>/dev/null; then
        echo -e "${GREEN}✓${NC} Redis is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}✗${NC} Redis failed to start"
        exit 1
    fi
    sleep 1
done

# Step 3: Build service
echo -e "${BLUE}[3/4]${NC} Building shortener-service..."
cd apps/shortener-service
go build -o bin/shortener-service
cd ../..
echo -e "${GREEN}✓${NC} Service built"

# Step 4: Instructions
echo -e "\n${BLUE}[4/4]${NC} Setup complete!"
echo -e "\n${GREEN}Next steps:${NC}"
echo -e "1. Start the service:"
echo -e "   ${BLUE}cd apps/shortener-service && ./bin/shortener-service${NC}"
echo -e "\n2. In another terminal, run chaos tests:"
echo -e "   ${BLUE}./scripts/health-check-chaos-test.sh all${NC}"
echo -e "\n3. To stop infrastructure:"
echo -e "   ${BLUE}docker compose -f deploy/docker/docker-compose.infra.yml down${NC}"
echo ""
