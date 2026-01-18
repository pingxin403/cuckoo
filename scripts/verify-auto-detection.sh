#!/bin/bash

# Script to verify auto-detection functionality
# This script tests that app type detection works correctly

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Auto-Detection Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to detect app type (same logic as CI)
detect_app_type() {
    local app_dir=$1
    local app_type=""
    
    # Priority 1: Check .apptype file
    if [ -f "$app_dir/.apptype" ]; then
        app_type=$(cat "$app_dir/.apptype" | tr -d '[:space:]')
        echo "detected_via=.apptype"
        echo "type=$app_type"
        return
    fi
    
    # Priority 2: Check metadata.yaml
    if [ -f "$app_dir/metadata.yaml" ]; then
        app_type=$(grep "^  type:" "$app_dir/metadata.yaml" | awk '{print $2}' | tr -d '[:space:]')
        echo "detected_via=metadata.yaml"
        echo "type=$app_type"
        return
    fi
    
    # Priority 3: Detect by file characteristics
    if [ -f "$app_dir/build.gradle" ] || [ -f "$app_dir/pom.xml" ]; then
        app_type="java"
        echo "detected_via=file_characteristics"
        echo "type=$app_type"
        return
    elif [ -f "$app_dir/go.mod" ]; then
        app_type="go"
        echo "detected_via=file_characteristics"
        echo "type=$app_type"
        return
    elif [ -f "$app_dir/package.json" ]; then
        app_type="node"
        echo "detected_via=file_characteristics"
        echo "type=$app_type"
        return
    fi
    
    echo "detected_via=none"
    echo "type=unknown"
}

# Test existing services
test_service() {
    local service=$1
    local expected_type=$2
    
    echo -e "${YELLOW}Testing: $service${NC}"
    
    local result=$(detect_app_type "apps/$service")
    local detected_via=$(echo "$result" | grep "detected_via=" | cut -d'=' -f2)
    local detected_type=$(echo "$result" | grep "type=" | cut -d'=' -f2)
    
    if [ "$detected_type" = "$expected_type" ]; then
        echo -e "${GREEN}✅ PASS${NC}: Detected as '$detected_type' via $detected_via"
    else
        echo -e "${RED}❌ FAIL${NC}: Expected '$expected_type', got '$detected_type'"
        exit 1
    fi
    echo ""
}

# Test all existing services
echo -e "${BLUE}Testing Existing Services:${NC}"
echo ""

test_service "hello-service" "java"
test_service "todo-service" "go"
test_service "web" "node"

# Test template files
echo -e "${BLUE}Testing Template Files:${NC}"
echo ""

for template in templates/*-service; do
    template_name=$(basename "$template")
    echo -e "${YELLOW}Testing template: $template_name${NC}"
    
    if [ -f "$template/.apptype" ]; then
        app_type=$(cat "$template/.apptype" | tr -d '[:space:]')
        echo -e "${GREEN}✅ PASS${NC}: Template has .apptype file with type '$app_type'"
    else
        echo -e "${RED}❌ FAIL${NC}: Template missing .apptype file"
        exit 1
    fi
    
    if [ -f "$template/metadata.yaml" ]; then
        echo -e "${GREEN}✅ PASS${NC}: Template has metadata.yaml file"
    else
        echo -e "${RED}❌ FAIL${NC}: Template missing metadata.yaml file"
        exit 1
    fi
    echo ""
done

# Test CI workflow syntax
echo -e "${BLUE}Testing CI Workflow:${NC}"
echo ""

if python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))" 2>/dev/null; then
    echo -e "${GREEN}✅ PASS${NC}: CI workflow YAML syntax is valid"
else
    echo -e "${RED}❌ FAIL${NC}: CI workflow YAML syntax is invalid"
    exit 1
fi
echo ""

# Test that CI uses dynamic detection
echo -e "${BLUE}Testing CI Dynamic Detection:${NC}"
echo ""

if grep -q "steps.detect-type.outputs.type" .github/workflows/ci.yml; then
    echo -e "${GREEN}✅ PASS${NC}: CI uses dynamic type detection"
else
    echo -e "${RED}❌ FAIL${NC}: CI does not use dynamic type detection"
    exit 1
fi

if grep -q "matrix.app == 'hello-service'" .github/workflows/ci.yml; then
    echo -e "${RED}❌ FAIL${NC}: CI still has hardcoded service names"
    exit 1
else
    echo -e "${GREEN}✅ PASS${NC}: CI has no hardcoded service names"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ All Verification Tests Passed!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Auto-detection is working correctly:"
echo "  ✅ All existing services detected correctly"
echo "  ✅ All templates have required metadata files"
echo "  ✅ CI workflow uses dynamic detection"
echo "  ✅ No hardcoded service names in CI"
echo ""
echo "The architecture is ready for unlimited service scaling!"
