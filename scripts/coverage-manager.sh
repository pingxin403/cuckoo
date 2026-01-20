#!/bin/bash

# Coverage Manager Script
# Provides unified interface for running test coverage across all app types
# Usage: ./scripts/coverage-manager.sh [app-name] [--verify]
#
# If app-name is omitted, runs coverage for all apps with test-coverage.sh scripts
# If --verify is passed, also verifies coverage thresholds

set -e

APP_NAME="$1"
VERIFY_MODE=false

# Check for --verify flag
if [ "$1" = "--verify" ] || [ "$2" = "--verify" ]; then
    VERIFY_MODE=true
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Normalize app name (support short names)
normalize_app_name() {
    local input_name="$1"
    
    # First check if it's already a full app name
    if [ -d "apps/$input_name" ]; then
        echo "$input_name"
        return
    fi
    
    # Search for app with matching short_name in metadata.yaml
    for app_dir in apps/*/; do
        if [ -f "$app_dir/metadata.yaml" ]; then
            local short_name=$(grep "^  short_name:" "$app_dir/metadata.yaml" | awk '{print $2}' | tr -d '[:space:]')
            if [ "$short_name" = "$input_name" ]; then
                basename "$app_dir"
                return
            fi
        fi
    done
    
    # If no match found, return the input as-is
    echo "$input_name"
}

# Auto-detect app type
detect_app_type() {
    local app_dir="$1"
    
    # Priority 1: Check metadata.yaml
    if [ -f "$app_dir/metadata.yaml" ]; then
        local type=$(grep "^  type:" "$app_dir/metadata.yaml" | awk '{print $2}' | tr -d '[:space:]')
        if [ -n "$type" ]; then
            echo "$type"
            return
        fi
    fi
    
    # Priority 2: Check .apptype file
    if [ -f "$app_dir/.apptype" ]; then
        cat "$app_dir/.apptype" | tr -d '[:space:]'
        return
    fi
    
    # Priority 3: Detect by file characteristics
    if [ -f "$app_dir/build.gradle" ] || [ -f "$app_dir/pom.xml" ]; then
        echo "java"
    elif [ -f "$app_dir/go.mod" ]; then
        echo "go"
    elif [ -f "$app_dir/package.json" ]; then
        echo "node"
    else
        echo ""
    fi
}

# Get all apps that have coverage support
get_coverage_apps() {
    local apps=""
    
    for app_dir in apps/*/; do
        local app=$(basename "$app_dir")
        local app_type=$(detect_app_type "$app_dir")
        
        # Check if app has coverage support
        case $app_type in
            java)
                # Java apps with Gradle/Maven
                if [ -f "$app_dir/gradlew" ] || [ -f "$app_dir/mvnw" ]; then
                    apps="$apps $app"
                fi
                ;;
            go)
                # Go apps with test-coverage.sh script
                if [ -f "$app_dir/scripts/test-coverage.sh" ]; then
                    apps="$apps $app"
                fi
                ;;
            node)
                # Node apps with test script
                if [ -f "$app_dir/package.json" ] && grep -q '"test"' "$app_dir/package.json"; then
                    apps="$apps $app"
                fi
                ;;
        esac
    done
    
    echo "$apps" | xargs
}

# Run coverage for Java app
run_java_coverage() {
    local app=$1
    local app_dir="apps/$app"
    
    log_info "Running coverage for Java app: $app"
    
    if [ -f "$app_dir/gradlew" ]; then
        if [ "$VERIFY_MODE" = true ]; then
            (cd "$app_dir" && ./gradlew test jacocoTestReport jacocoTestCoverageVerification) || return 1
            log_info "Coverage report: $app_dir/build/reports/jacoco/test/html/index.html"
        else
            (cd "$app_dir" && ./gradlew test jacocoTestReport) || return 1
            log_info "Coverage report: $app_dir/build/reports/jacoco/test/html/index.html"
        fi
    elif [ -f "$app_dir/mvnw" ]; then
        if [ "$VERIFY_MODE" = true ]; then
            (cd "$app_dir" && ./mvnw test jacoco:report jacoco:check) || return 1
        else
            (cd "$app_dir" && ./mvnw test jacoco:report) || return 1
        fi
        log_info "Coverage report: $app_dir/target/site/jacoco/index.html"
    else
        log_error "No build tool found for $app"
        return 1
    fi
}

# Run coverage for Go app
run_go_coverage() {
    local app=$1
    local app_dir="apps/$app"
    
    log_info "Running coverage for Go app: $app"
    
    if [ -f "$app_dir/scripts/test-coverage.sh" ]; then
        (cd "$app_dir" && ./scripts/test-coverage.sh) || return 1
        log_info "Coverage report: $app_dir/coverage.html"
    else
        # Fallback to basic go test with coverage
        log_warning "No test-coverage.sh script found, using basic go test"
        (cd "$app_dir" && go test -v -race -coverprofile=coverage.out ./...) || return 1
        (cd "$app_dir" && go tool cover -html=coverage.out -o coverage.html)
        log_info "Coverage report: $app_dir/coverage.html"
        
        if [ "$VERIFY_MODE" = true ]; then
            log_warning "Coverage verification not available without test-coverage.sh script"
        fi
    fi
}

# Run coverage for Node app
run_node_coverage() {
    local app=$1
    local app_dir="apps/$app"
    
    log_info "Running coverage for Node app: $app"
    
    if [ -f "$app_dir/package.json" ]; then
        # Check if coverage script exists
        if grep -q '"test:coverage"' "$app_dir/package.json"; then
            (cd "$app_dir" && npm run test:coverage) || return 1
        else
            # Fallback to regular test
            log_warning "No test:coverage script found, using regular test"
            (cd "$app_dir" && npm test -- --run) || return 1
        fi
        
        # Try to find coverage report
        if [ -d "$app_dir/coverage" ]; then
            log_info "Coverage report: $app_dir/coverage/index.html"
        fi
    else
        log_error "No package.json found for $app"
        return 1
    fi
}

# Run coverage for a single app
run_app_coverage() {
    local app=$1
    local app_dir="apps/$app"
    local app_type=$(detect_app_type "$app_dir")
    
    if [ ! -d "$app_dir" ]; then
        log_error "App directory not found: $app_dir"
        return 1
    fi
    
    case $app_type in
        java)
            run_java_coverage "$app"
            ;;
        go)
            run_go_coverage "$app"
            ;;
        node)
            run_node_coverage "$app"
            ;;
        *)
            log_error "Unknown or unsupported app type: $app_type"
            return 1
            ;;
    esac
}

# Main execution
main() {
    if [ -n "$APP_NAME" ] && [ "$APP_NAME" != "--verify" ]; then
        # Run coverage for specific app
        local normalized_app=$(normalize_app_name "$APP_NAME")
        
        log_info "========================================="
        log_info "Running coverage for: $normalized_app"
        log_info "========================================="
        
        if run_app_coverage "$normalized_app"; then
            log_success "✓ Coverage completed for $normalized_app"
        else
            log_error "✗ Coverage failed for $normalized_app"
            exit 1
        fi
    else
        # Run coverage for all apps
        local apps=$(get_coverage_apps)
        
        if [ -z "$apps" ]; then
            log_warning "No apps with coverage support found"
            exit 0
        fi
        
        log_info "Apps with coverage support: $apps"
        echo ""
        
        local failed_apps=""
        
        for app in $apps; do
            log_info "========================================="
            log_info "Running coverage for: $app"
            log_info "========================================="
            
            if run_app_coverage "$app"; then
                log_success "✓ Coverage completed for $app"
            else
                log_error "✗ Coverage failed for $app"
                failed_apps="$failed_apps $app"
            fi
            echo ""
        done
        
        log_info "========================================="
        log_info "Summary"
        log_info "========================================="
        
        if [ -n "$failed_apps" ]; then
            log_error "Failed apps:$failed_apps"
            exit 1
        else
            log_success "All coverage tests passed!"
        fi
    fi
}

main
