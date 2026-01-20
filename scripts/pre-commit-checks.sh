#!/bin/bash

# Pre-commit Quality Checks Script
# This script runs all quality checks that should pass before committing code
# Can be run manually with 'make pre-commit' or automatically via git hook

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Pre-Commit Quality Checks ===${NC}\n"

# Track if any checks failed
FAILED=0
CHECKS_RUN=0

# Function to check if files changed in a directory
files_changed_in() {
    local dir=$1
    if [ -d ".git" ]; then
        git diff --cached --name-only 2>/dev/null | grep -q "^${dir}/" && return 0
        git diff --name-only 2>/dev/null | grep -q "^${dir}/" && return 0
    fi
    # If not in git or no changes, check if directory exists
    [ -d "$dir" ] && return 0
    return 1
}

# 1. Check tool versions
echo -e "${BLUE}[1/6] Checking tool versions...${NC}"
if ./scripts/check-versions.sh > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Tool versions are correct${NC}\n"
else
    echo -e "${YELLOW}⚠ Tool version mismatch detected${NC}"
    echo -e "${YELLOW}  Run 'make check-versions' for details${NC}\n"
fi
CHECKS_RUN=$((CHECKS_RUN + 1))

# 2. Verify Protobuf generated code
if files_changed_in "api"; then
    echo -e "${BLUE}[2/6] Verifying Protobuf generated code...${NC}"
    if make proto > /dev/null 2>&1 && git diff --exit-code apps/*/gen apps/*/src/gen > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Protobuf code is up to date${NC}\n"
    else
        echo -e "${RED}✗ Protobuf code is out of date${NC}"
        echo -e "${YELLOW}  Run 'make proto' to regenerate${NC}\n"
        FAILED=1
    fi
    CHECKS_RUN=$((CHECKS_RUN + 1))
else
    echo -e "${BLUE}[2/6] Skipping Protobuf check (no API changes)${NC}\n"
fi

# 3. Run linters for all changed services
echo -e "${BLUE}[3/6] Running linters...${NC}"

# Java linting (Hello Service)
if files_changed_in "apps/hello-service/src"; then
    echo -e "${YELLOW}  Checking Java code (hello-service)...${NC}"
    if cd apps/hello-service && ./gradlew spotlessCheck --quiet 2>&1; then
        echo -e "${GREEN}  ✓ Java formatting is correct${NC}"
    else
        echo -e "${RED}  ✗ Java formatting issues found${NC}"
        echo -e "${YELLOW}    Run 'make lint-fix APP=hello' to fix${NC}"
        FAILED=1
    fi
    cd ../..
fi

# Go linting (TODO Service)
if files_changed_in "apps/todo-service"; then
    echo -e "${YELLOW}  Checking Go code (todo-service)...${NC}"
    
    # Check formatting
    if cd apps/todo-service && [ -z "$(gofmt -l . 2>/dev/null | grep -v '^gen/')" ]; then
        echo -e "${GREEN}  ✓ Go formatting is correct${NC}"
    else
        echo -e "${RED}  ✗ Go formatting issues found${NC}"
        echo -e "${YELLOW}    Run 'make format APP=todo' to fix${NC}"
        FAILED=1
    fi
    
    # Run golangci-lint if available
    if command -v golangci-lint >/dev/null 2>&1; then
        LINT_OUTPUT=$(golangci-lint run ./... 2>&1)
        if echo "$LINT_OUTPUT" | grep -q "0 issues"; then
            echo -e "${GREEN}  ✓ Go linting passed${NC}"
        else
            echo -e "${RED}  ✗ Go linting issues found${NC}"
            echo -e "${YELLOW}    Run 'golangci-lint run' for details${NC}"
            FAILED=1
        fi
    fi
    cd ../..
fi

# TypeScript linting (Web)
if files_changed_in "apps/web/src"; then
    echo -e "${YELLOW}  Checking TypeScript code (web)...${NC}"
    if cd apps/web && npm run lint --silent 2>&1; then
        echo -e "${GREEN}  ✓ TypeScript linting passed${NC}"
    else
        echo -e "${RED}  ✗ TypeScript linting issues found${NC}"
        echo -e "${YELLOW}    Run 'make lint-fix APP=web' to fix${NC}"
        FAILED=1
    fi
    cd ../..
fi

echo ""
CHECKS_RUN=$((CHECKS_RUN + 1))

# 4. Run tests for changed services
echo -e "${BLUE}[4/6] Running tests...${NC}"

# Java tests (Hello Service)
if files_changed_in "apps/hello-service"; then
    echo -e "${YELLOW}  Testing Java code (hello-service)...${NC}"
    if cd apps/hello-service && ./gradlew test --quiet 2>&1; then
        echo -e "${GREEN}  ✓ Java tests passed${NC}"
    else
        echo -e "${RED}  ✗ Java tests failed${NC}"
        echo -e "${YELLOW}    Run 'make test APP=hello' for details${NC}"
        FAILED=1
    fi
    cd ../..
fi

# Go tests (TODO Service)
if files_changed_in "apps/todo-service"; then
    echo -e "${YELLOW}  Testing Go code (todo-service)...${NC}"
    if cd apps/todo-service && go test ./... -short 2>&1 | grep -q "PASS\|ok"; then
        echo -e "${GREEN}  ✓ Go tests passed${NC}"
    else
        echo -e "${RED}  ✗ Go tests failed${NC}"
        echo -e "${YELLOW}    Run 'make test APP=todo' for details${NC}"
        FAILED=1
    fi
    cd ../..
fi

# TypeScript tests (Web)
if files_changed_in "apps/web/src"; then
    echo -e "${YELLOW}  Testing TypeScript code (web)...${NC}"
    if cd apps/web && npm test -- --run --silent 2>&1; then
        echo -e "${GREEN}  ✓ TypeScript tests passed${NC}"
    else
        echo -e "${RED}  ✗ TypeScript tests failed${NC}"
        echo -e "${YELLOW}    Run 'make test APP=web' for details${NC}"
        FAILED=1
    fi
    cd ../..
fi

echo ""
CHECKS_RUN=$((CHECKS_RUN + 1))

# 5. Check for common issues
echo -e "${BLUE}[5/6] Checking for common issues...${NC}"

# Check for console.log in TypeScript
if files_changed_in "apps/web/src"; then
    if git diff --cached --name-only 2>/dev/null | xargs grep -n "console\.log" 2>/dev/null; then
        echo -e "${YELLOW}⚠ Found console.log statements${NC}"
        echo -e "${YELLOW}  Consider removing debug statements before commit${NC}"
    fi
fi

# Check for TODO/FIXME comments in changed files
if [ -d ".git" ]; then
    TODO_COUNT=$(git diff --cached --name-only 2>/dev/null | xargs grep -n "TODO\|FIXME" 2>/dev/null | wc -l)
    if [ "$TODO_COUNT" -gt 0 ]; then
        echo -e "${YELLOW}⚠ Found $TODO_COUNT TODO/FIXME comments${NC}"
        echo -e "${YELLOW}  Consider addressing them before commit${NC}"
    fi
fi

# Check for large files
if [ -d ".git" ]; then
    LARGE_FILES=$(git diff --cached --name-only 2>/dev/null | xargs ls -l 2>/dev/null | awk '$5 > 1048576 {print $9, $5}')
    if [ -n "$LARGE_FILES" ]; then
        echo -e "${YELLOW}⚠ Found large files (>1MB):${NC}"
        echo "$LARGE_FILES"
        echo -e "${YELLOW}  Consider if these should be committed${NC}"
    fi
fi

echo -e "${GREEN}✓ Common issues check completed${NC}\n"
CHECKS_RUN=$((CHECKS_RUN + 1))

# 6. Security checks
echo -e "${BLUE}[6/6] Running security checks...${NC}"

# Check for potential secrets
if [ -d ".git" ]; then
    # Get list of changed files (excluding documentation and test files)
    CHANGED_CODE_FILES=$(git diff --cached --name-only 2>/dev/null | \
        grep -v "\.md$" | \
        grep -v "\.txt$" | \
        grep -v "_test\.go$" | \
        grep -v "_test\.ts$" | \
        grep -v "_test\.js$" | \
        grep -v "Test\.java$" | \
        grep -v "^docs/" | \
        grep -v "^scripts/" | \
        grep -v "\.sh$")
    
    if [ -n "$CHANGED_CODE_FILES" ]; then
        # Check only code files for secrets
        SECRETS=""
        for file in $CHANGED_CODE_FILES; do
            FILE_SECRETS=$(git diff --cached -- "$file" 2>/dev/null | \
                grep -iE "(password|secret|api[_-]?key|token|credential)" | \
                grep -v "^-" | \
                grep -v "Password: \"\"" | \
                grep -v "// Empty password" | \
                grep -v "// Test config" | \
                grep -v "root_password" | \
                grep -v "test_password" | \
                grep -v "shortener_password" | \
                grep -v "credentials/insecure" | \
                grep -v "insecure.NewCredentials" | \
                grep -v "example.com" | \
                grep -v "localhost")
            if [ -n "$FILE_SECRETS" ]; then
                SECRETS="${SECRETS}
${FILE_SECRETS}"
            fi
        done
        
        if [ -n "$SECRETS" ]; then
            echo -e "${RED}✗ Potential secrets detected in code files:${NC}"
            echo "$SECRETS" | head -5
            echo -e "${RED}  Please review and remove any sensitive data${NC}"
            echo -e "${YELLOW}  If these are false positives (test data), you can:${NC}"
            echo -e "${YELLOW}  1. Review the changes carefully${NC}"
            echo -e "${YELLOW}  2. Use 'git commit --no-verify' to skip this check${NC}"
            FAILED=1
        else
            echo -e "${GREEN}✓ No obvious secrets detected in code files${NC}"
        fi
    else
        echo -e "${GREEN}✓ No code files changed (skipping secret scan)${NC}"
    fi
else
    echo -e "${GREEN}✓ Not a git repository (skipping secret scan)${NC}"
fi

echo ""
CHECKS_RUN=$((CHECKS_RUN + 1))

# Summary
echo -e "${BLUE}=== Summary ===${NC}"
echo -e "Checks run: $CHECKS_RUN"

if [ $FAILED -eq 1 ]; then
    echo -e "${RED}✗ Some checks failed${NC}"
    echo -e "${YELLOW}Please fix the issues above before committing${NC}"
    echo ""
    echo -e "${BLUE}Quick fixes:${NC}"
    echo -e "  - Format code:  ${GREEN}make format${NC}"
    echo -e "  - Fix linting:  ${GREEN}make lint-fix${NC}"
    echo -e "  - Run tests:    ${GREEN}make test${NC}"
    echo ""
    exit 1
else
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo -e "${GREEN}Ready to commit${NC}"
    exit 0
fi
