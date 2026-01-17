#!/bin/bash

# Initialization script for Monorepo Hello/TODO Services
# This script sets up the development environment and installs all dependencies

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Monorepo Hello/TODO Services - Environment Setup ===${NC}\n"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     PLATFORM=Linux;;
    Darwin*)    PLATFORM=Mac;;
    *)          PLATFORM="UNKNOWN:${OS}"
esac

echo -e "${BLUE}Detected platform: ${PLATFORM}${NC}\n"

# Track installation status
ERRORS=0

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check version
check_version() {
    local cmd=$1
    local min_version=$2
    local current_version=$($cmd 2>&1 | head -1)
    echo -e "${GREEN}✓ $cmd installed: $current_version${NC}"
}

# Check Java
echo -e "${BLUE}Checking Java...${NC}"
if command_exists java; then
    JAVA_VERSION=$(java -version 2>&1 | head -1 | cut -d'"' -f2 | cut -d'.' -f1)
    if [ "$JAVA_VERSION" -ge 17 ]; then
        check_version "java -version" "17"
    else
        echo -e "${RED}✗ Java version is too old. Need Java 17+${NC}"
        echo -e "${YELLOW}  Install: https://adoptium.net/ or use SDKMAN: https://sdkman.io/${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}✗ Java not found${NC}"
    echo -e "${YELLOW}  Install: https://adoptium.net/ or use SDKMAN: https://sdkman.io/${NC}"
    ERRORS=$((ERRORS + 1))
fi

# Check Go
echo -e "\n${BLUE}Checking Go...${NC}"
if command_exists go; then
    check_version "go version" "1.21"
else
    echo -e "${RED}✗ Go not found${NC}"
    if [ "$PLATFORM" = "Mac" ]; then
        echo -e "${YELLOW}  Install: brew install go${NC}"
    else
        echo -e "${YELLOW}  Install: https://golang.org/dl/${NC}"
    fi
    ERRORS=$((ERRORS + 1))
fi

# Check Node.js
echo -e "\n${BLUE}Checking Node.js...${NC}"
if command_exists node; then
    NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$NODE_VERSION" -ge 18 ]; then
        check_version "node -v" "18"
    else
        echo -e "${RED}✗ Node.js version is too old. Need Node 18+${NC}"
        echo -e "${YELLOW}  Install: https://nodejs.org/ or use nvm: https://github.com/nvm-sh/nvm${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}✗ Node.js not found${NC}"
    echo -e "${YELLOW}  Install: https://nodejs.org/ or use nvm: https://github.com/nvm-sh/nvm${NC}"
    ERRORS=$((ERRORS + 1))
fi

# Check npm
if command_exists npm; then
    check_version "npm -v" "8"
else
    echo -e "${RED}✗ npm not found (should come with Node.js)${NC}"
    ERRORS=$((ERRORS + 1))
fi

# Check protoc
echo -e "\n${BLUE}Checking Protocol Buffers compiler...${NC}"
if command_exists protoc; then
    check_version "protoc --version" "3"
else
    echo -e "${YELLOW}⚠ protoc not found${NC}"
    if [ "$PLATFORM" = "Mac" ]; then
        echo -e "${YELLOW}  Installing via Homebrew...${NC}"
        if command_exists brew; then
            brew install protobuf
            echo -e "${GREEN}✓ protoc installed${NC}"
        else
            echo -e "${RED}✗ Homebrew not found. Install manually: https://grpc.io/docs/protoc-installation/${NC}"
            ERRORS=$((ERRORS + 1))
        fi
    else
        echo -e "${YELLOW}  Install: https://grpc.io/docs/protoc-installation/${NC}"
        ERRORS=$((ERRORS + 1))
    fi
fi

# Check Envoy (optional)
echo -e "\n${BLUE}Checking Envoy (optional)...${NC}"
if command_exists envoy; then
    check_version "envoy --version" ""
else
    echo -e "${YELLOW}⚠ Envoy not found (optional but recommended)${NC}"
    if [ "$PLATFORM" = "Mac" ]; then
        echo -e "${YELLOW}  To install: brew install envoy${NC}"
    else
        echo -e "${YELLOW}  To install: https://www.envoyproxy.io/docs/envoy/latest/start/install${NC}"
    fi
fi

# Stop if critical dependencies are missing
if [ $ERRORS -gt 0 ]; then
    echo -e "\n${RED}=== Setup Failed ===${NC}"
    echo -e "${RED}Please install missing dependencies and run this script again.${NC}"
    exit 1
fi

# Install Go tools
echo -e "\n${BLUE}Installing Go tools...${NC}"
if command_exists go; then
    echo -e "${YELLOW}Installing protoc-gen-go...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    echo -e "${GREEN}✓ protoc-gen-go installed${NC}"
    
    echo -e "${YELLOW}Installing protoc-gen-go-grpc...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    echo -e "${GREEN}✓ protoc-gen-go-grpc installed${NC}"
    
    # Optional: golangci-lint
    if ! command_exists golangci-lint; then
        echo -e "${YELLOW}Installing golangci-lint (optional)...${NC}"
        if [ "$PLATFORM" = "Mac" ]; then
            brew install golangci-lint 2>/dev/null || echo -e "${YELLOW}  Skipped (install manually if needed)${NC}"
        else
            curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
        fi
    fi
fi

# Install frontend dependencies
echo -e "\n${BLUE}Installing frontend dependencies...${NC}"
if [ -d "apps/web" ]; then
    cd apps/web
    echo -e "${YELLOW}Running npm install...${NC}"
    npm install
    echo -e "${GREEN}✓ Frontend dependencies installed${NC}"
    cd ../..
fi

# Generate Protobuf code
echo -e "\n${BLUE}Generating Protobuf code...${NC}"
if command_exists protoc; then
    make gen-proto 2>/dev/null || {
        echo -e "${YELLOW}⚠ Protobuf generation failed. You may need to run 'make gen-proto' manually.${NC}"
    }
else
    echo -e "${YELLOW}⚠ Skipping Protobuf generation (protoc not found)${NC}"
fi

# Install Git hooks
echo -e "\n${BLUE}Installing Git hooks...${NC}"
if [ -f "scripts/install-hooks.sh" ]; then
    ./scripts/install-hooks.sh
else
    echo -e "${YELLOW}⚠ Git hooks script not found${NC}"
fi

# Create logs directory
echo -e "\n${BLUE}Creating logs directory...${NC}"
mkdir -p logs
echo -e "${GREEN}✓ logs/ directory created${NC}"

# Summary
echo -e "\n${GREEN}=== Setup Complete! ===${NC}\n"

echo -e "${BLUE}Next steps:${NC}"
echo -e "  1. Build all services:    ${GREEN}make build${NC}"
echo -e "  2. Run tests:             ${GREEN}make test${NC}"
echo -e "  3. Start development:     ${GREEN}./scripts/dev.sh${NC}"
echo -e "     (or start services individually)"
echo -e ""
echo -e "${BLUE}Useful commands:${NC}"
echo -e "  - Generate Protobuf:      ${GREEN}make gen-proto${NC}"
echo -e "  - Run linters:            ${GREEN}make lint${NC}"
echo -e "  - Format code:            ${GREEN}make format${NC}"
echo -e "  - Build Docker images:    ${GREEN}make docker-build${NC}"
echo -e ""
echo -e "${BLUE}Documentation:${NC}"
echo -e "  - Quick start:            ${GREEN}README.md${NC}"
echo -e "  - Infrastructure:         ${GREEN}docs/INFRASTRUCTURE.md${NC}"
echo -e "  - Code quality:           ${GREEN}docs/CODE_QUALITY.md${NC}"
echo -e "  - Local verification:     ${GREEN}docs/LOCAL_SETUP_VERIFICATION.md${NC}"
echo -e ""

if ! command_exists envoy; then
    echo -e "${YELLOW}Note: Envoy is not installed. Frontend will not be able to communicate with backend services.${NC}"
    echo -e "${YELLOW}To install Envoy:${NC}"
    if [ "$PLATFORM" = "Mac" ]; then
        echo -e "  ${GREEN}brew install envoy${NC}"
    else
        echo -e "  ${GREEN}https://www.envoyproxy.io/docs/envoy/latest/start/install${NC}"
    fi
    echo -e ""
fi

echo -e "${GREEN}Happy coding! 🚀${NC}"
