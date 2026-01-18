#!/bin/bash

# Check if installed tool versions match required versions
# This script reads from .tool-versions and validates installed tools

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Load version requirements
source .tool-versions

echo "Checking tool versions..."
echo ""

ERRORS=0
WARNINGS=0

# Function to compare versions
version_ge() {
    # Returns 0 if $1 >= $2
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Check protoc
echo -n "Checking protoc... "
if command -v protoc >/dev/null 2>&1; then
    INSTALLED_PROTOC=$(protoc --version | grep -oE '[0-9]+\.[0-9]+' | head -1)
    REQUIRED_PROTOC=$(echo $PROTOC_VERSION | grep -oE '[0-9]+\.[0-9]+')
    
    if [ "$INSTALLED_PROTOC" = "$REQUIRED_PROTOC" ]; then
        echo -e "${GREEN}✓${NC} $INSTALLED_PROTOC (matches required $REQUIRED_PROTOC)"
    else
        echo -e "${RED}✗${NC} $INSTALLED_PROTOC (required: $REQUIRED_PROTOC)"
        echo "  Install: See docs/PROTO_TOOLS_VERSION.md"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    echo "  Install: brew install protobuf (macOS) or see docs/PROTO_TOOLS_VERSION.md"
    ERRORS=$((ERRORS + 1))
fi

# Check protoc-gen-go
echo -n "Checking protoc-gen-go... "
if command -v protoc-gen-go >/dev/null 2>&1; then
    INSTALLED_VERSION=$(protoc-gen-go --version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    
    if [ "$INSTALLED_VERSION" = "$PROTOC_GEN_GO_VERSION" ]; then
        echo -e "${GREEN}✓${NC} $INSTALLED_VERSION"
    else
        echo -e "${RED}✗${NC} $INSTALLED_VERSION (required: $PROTOC_GEN_GO_VERSION)"
        echo "  Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOC_GEN_GO_VERSION"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    echo "  Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOC_GEN_GO_VERSION"
    ERRORS=$((ERRORS + 1))
fi

# Check protoc-gen-go-grpc
echo -n "Checking protoc-gen-go-grpc... "
if command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    INSTALLED_VERSION=$(protoc-gen-go-grpc --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    REQUIRED_VERSION=$(echo $PROTOC_GEN_GO_GRPC_VERSION | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
    
    if [ "$INSTALLED_VERSION" = "$REQUIRED_VERSION" ]; then
        echo -e "${GREEN}✓${NC} $INSTALLED_VERSION"
    else
        echo -e "${RED}✗${NC} $INSTALLED_VERSION (required: $REQUIRED_VERSION)"
        echo "  Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$PROTOC_GEN_GO_GRPC_VERSION"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    echo "  Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$PROTOC_GEN_GO_GRPC_VERSION"
    ERRORS=$((ERRORS + 1))
fi

# Check Go
echo -n "Checking Go... "
if command -v go >/dev/null 2>&1; then
    INSTALLED_GO=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+')
    
    if version_ge "$INSTALLED_GO" "$GO_VERSION"; then
        echo -e "${GREEN}✓${NC} $INSTALLED_GO (>= $GO_VERSION)"
    else
        echo -e "${YELLOW}⚠${NC} $INSTALLED_GO (recommended: >= $GO_VERSION)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    ERRORS=$((ERRORS + 1))
fi

# Check Node.js
echo -n "Checking Node.js... "
if command -v node >/dev/null 2>&1; then
    INSTALLED_NODE=$(node --version | grep -oE '[0-9]+' | head -1)
    
    if [ "$INSTALLED_NODE" -ge "$NODE_VERSION" ]; then
        echo -e "${GREEN}✓${NC} v$INSTALLED_NODE (>= $NODE_VERSION)"
    else
        echo -e "${YELLOW}⚠${NC} v$INSTALLED_NODE (recommended: >= $NODE_VERSION)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    ERRORS=$((ERRORS + 1))
fi

# Check Java
echo -n "Checking Java... "
if command -v java >/dev/null 2>&1; then
    INSTALLED_JAVA=$(java -version 2>&1 | grep -oE 'version "[0-9]+' | grep -oE '[0-9]+')
    
    if [ "$INSTALLED_JAVA" -ge "$JAVA_VERSION" ]; then
        echo -e "${GREEN}✓${NC} $INSTALLED_JAVA (>= $JAVA_VERSION)"
    else
        echo -e "${YELLOW}⚠${NC} $INSTALLED_JAVA (recommended: >= $JAVA_VERSION)"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo -e "${RED}✗${NC} not installed"
    ERRORS=$((ERRORS + 1))
fi

# Summary
echo ""
echo "========================================="
if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All tools are correctly installed!${NC}"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠ $WARNINGS warning(s) found${NC}"
    echo "Your tools will work but may have minor differences from CI"
    exit 0
else
    echo -e "${RED}✗ $ERRORS error(s) found${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠ $WARNINGS warning(s) found${NC}"
    fi
    echo ""
    echo "Please install or update the required tools."
    echo "Run 'make init' to install missing tools automatically."
    exit 1
fi
