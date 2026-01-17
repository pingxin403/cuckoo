#!/bin/bash

# Environment check script
# Verifies that all required tools are installed and properly configured

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Environment Check ===${NC}\n"

# Track status
REQUIRED_OK=true
OPTIONAL_OK=true

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check required tool
check_required() {
    local tool=$1
    local min_version=$2
    local install_hint=$3
    
    if command_exists "$tool"; then
        echo -e "${GREEN}✓ $tool${NC}"
        return 0
    else
        echo -e "${RED}✗ $tool (REQUIRED)${NC}"
        echo -e "${YELLOW}  Install: $install_hint${NC}"
        REQUIRED_OK=false
        return 1
    fi
}

# Function to check optional tool
check_optional() {
    local tool=$1
    local install_hint=$2
    
    if command_exists "$tool"; then
        echo -e "${GREEN}✓ $tool${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ $tool (optional)${NC}"
        echo -e "${YELLOW}  Install: $install_hint${NC}"
        OPTIONAL_OK=false
        return 1
    fi
}

# Check required tools
echo -e "${BLUE}Required Tools:${NC}"
check_required "java" "17" "https://adoptium.net/ or SDKMAN (https://sdkman.io/)"
check_required "go" "1.21" "https://golang.org/dl/ or 'brew install go'"
check_required "node" "18" "https://nodejs.org/ or nvm (https://github.com/nvm-sh/nvm)"
check_required "npm" "8" "Comes with Node.js"
check_required "protoc" "3" "https://grpc.io/docs/protoc-installation/ or 'brew install protobuf'"

# Check Go tools
echo -e "\n${BLUE}Go Tools:${NC}"
check_required "protoc-gen-go" "" "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
check_required "protoc-gen-go-grpc" "" "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"

# Check optional tools
echo -e "\n${BLUE}Optional Tools:${NC}"
check_optional "envoy" "brew install envoy (macOS) or https://www.envoyproxy.io/docs/envoy/latest/start/install"
check_optional "golangci-lint" "brew install golangci-lint or https://golangci-lint.run/usage/install/"
check_optional "docker" "https://docs.docker.com/get-docker/"
check_optional "kubectl" "https://kubernetes.io/docs/tasks/tools/"

# Note: Optional tools don't affect REQUIRED_OK status

# Check versions
echo -e "\n${BLUE}Version Information:${NC}"
if command_exists java; then
    java -version 2>&1 | head -1
fi
if command_exists go; then
    go version
fi
if command_exists node; then
    echo "Node: $(node -v)"
fi
if command_exists npm; then
    echo "npm: $(npm -v)"
fi
if command_exists protoc; then
    protoc --version
fi
if command_exists envoy; then
    envoy --version 2>&1 | head -1
fi

# Check project structure
echo -e "\n${BLUE}Project Structure:${NC}"
if [ -d "api/v1" ]; then
    echo -e "${GREEN}✓ api/v1/ directory exists${NC}"
else
    echo -e "${RED}✗ api/v1/ directory missing${NC}"
    REQUIRED_OK=false
fi

if [ -d "apps/hello-service" ]; then
    echo -e "${GREEN}✓ apps/hello-service/ directory exists${NC}"
else
    echo -e "${RED}✗ apps/hello-service/ directory missing${NC}"
    REQUIRED_OK=false
fi

if [ -d "apps/todo-service" ]; then
    echo -e "${GREEN}✓ apps/todo-service/ directory exists${NC}"
else
    echo -e "${RED}✗ apps/todo-service/ directory missing${NC}"
    REQUIRED_OK=false
fi

if [ -d "apps/web" ]; then
    echo -e "${GREEN}✓ apps/web/ directory exists${NC}"
else
    echo -e "${RED}✗ apps/web/ directory missing${NC}"
    REQUIRED_OK=false
fi

# Check if frontend dependencies are installed
if [ -d "apps/web/node_modules" ]; then
    echo -e "${GREEN}✓ Frontend dependencies installed${NC}"
else
    echo -e "${YELLOW}⚠ Frontend dependencies not installed${NC}"
    echo -e "${YELLOW}  Run: cd apps/web && npm install${NC}"
fi

# Summary
echo -e "\n${BLUE}=== Summary ===${NC}"
if [ "$REQUIRED_OK" = true ]; then
    echo -e "${GREEN}✓ All required tools are installed${NC}"
    
    if [ "$OPTIONAL_OK" = true ]; then
        echo -e "${GREEN}✓ All optional tools are installed${NC}"
        echo -e "\n${GREEN}Environment is fully configured! 🎉${NC}"
    else
        echo -e "${YELLOW}⚠ Some optional tools are missing${NC}"
        echo -e "\n${YELLOW}Environment is ready for basic development.${NC}"
        echo -e "${YELLOW}Install optional tools for full functionality.${NC}"
    fi
    
    echo -e "\n${BLUE}Next steps:${NC}"
    echo -e "  1. Run: ${GREEN}make init${NC} (if not done already)"
    echo -e "  2. Run: ${GREEN}make build${NC}"
    echo -e "  3. Run: ${GREEN}./scripts/dev.sh${NC}"
    exit 0
else
    echo -e "${RED}✗ Some required tools are missing${NC}"
    echo -e "\n${RED}Please install missing tools and run this check again.${NC}"
    echo -e "\n${BLUE}Quick fix:${NC}"
    echo -e "  Run: ${GREEN}make init${NC}"
    exit 1
fi
