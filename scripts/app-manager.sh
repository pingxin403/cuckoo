#!/bin/bash

# Application Manager Script
# Provides unified interface for managing apps in the monorepo
# Usage: ./scripts/app-manager.sh <command> [app-name]
#
# Commands:
#   test [app]       - Run tests for app(s)
#   build [app]      - Build app(s)
#   run [app]        - Run app(s) locally
#   docker [app]     - Build Docker image(s)
#   lint [app]       - Run linters for app(s)
#   lint-fix [app]   - Auto-fix lint errors for app(s)
#   clean [app]      - Clean build artifacts for app(s)
#   format [app]     - Format code for app(s)
#   list             - List all available apps
#
# If app-name is omitted, operates on changed apps (detected via git)

set -e

COMMAND="$1"
APP_NAME="$2"

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

# Normalize app name (support short names like "hello" -> "hello-service")
# Now reads from metadata.yaml instead of hardcoded mappings
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

# Auto-detect app type based on files
# Priority: metadata.yaml > .apptype (legacy) > file detection
detect_app_type() {
    local app_dir="$1"
    
    # Priority 1: Check metadata.yaml (preferred)
    if [ -f "$app_dir/metadata.yaml" ]; then
        local type=$(grep "^  type:" "$app_dir/metadata.yaml" | awk '{print $2}' | tr -d '[:space:]')
        if [ -n "$type" ]; then
            echo "$type"
            return
        fi
    fi
    
    # Priority 2: Check .apptype file (legacy support)
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

# Get app type
get_app_type() {
    local app=$(normalize_app_name "$1")
    local app_dir=$(get_app_path "$app")
    
    if [ -z "$app_dir" ] || [ ! -d "$app_dir" ]; then
        echo ""
        return
    fi
    
    detect_app_type "$app_dir"
}

# Get app path
get_app_path() {
    local app=$(normalize_app_name "$1")
    
    # Check if app directory exists
    if [ -d "apps/$app" ]; then
        echo "apps/$app"
    else
        echo ""
    fi
}

# Get list of all apps dynamically
get_all_apps() {
    for app_dir in apps/*/; do
        if [ -d "$app_dir" ]; then
            basename "$app_dir"
        fi
    done | tr '\n' ' ' | xargs
}

# Get list of apps to operate on
get_apps() {
    if [ -n "$APP_NAME" ]; then
        # Specific app provided - normalize the name
        local normalized_app=$(normalize_app_name "$APP_NAME")
        local app_type=$(get_app_type "$normalized_app")
        if [ -z "$app_type" ]; then
            log_error "Unknown app: $APP_NAME" >&2
            log_info "Available apps: $(get_all_apps)" >&2
            
            # Show available short names
            local short_names=""
            for app_dir in apps/*/; do
                if [ -f "$app_dir/metadata.yaml" ]; then
                    local short_name=$(grep "^  short_name:" "$app_dir/metadata.yaml" | awk '{print $2}' | tr -d '[:space:]')
                    if [ -n "$short_name" ]; then
                        short_names="$short_names $short_name"
                    fi
                fi
            done
            if [ -n "$short_names" ]; then
                log_info "Short names:$short_names" >&2
            fi
            exit 1
        fi
        echo "$normalized_app"
    else
        # Detect changed apps
        log_info "No app specified, detecting changed apps..." >&2
        local CHANGED_APPS=$(./scripts/detect-changed-apps.sh 2>/dev/null)
        log_info "Changed apps: $CHANGED_APPS" >&2
        echo "$CHANGED_APPS"
    fi
}

# Test command
cmd_test() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Testing $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                (cd "$app_path" && ./gradlew test) || return 1
            elif [ -f "$app_path/mvnw" ]; then
                (cd "$app_path" && ./mvnw test) || return 1
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            (cd "$app_path" && go test ./... -timeout=10m) || return 1
            ;;
        node)
            (cd "$app_path" && npm test -- --run) || return 1
            ;;
        migration)
            log_info "Skipping tests for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Tests passed for $app"
}

# Build command
cmd_build() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Building $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                (cd "$app_path" && ./gradlew build -x test -x jacocoTestCoverageVerification) || return 1
            elif [ -f "$app_path/mvnw" ]; then
                (cd "$app_path" && ./mvnw clean package -DskipTests) || return 1
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            (cd "$app_path" && go build -o bin/$app .) || return 1
            ;;
        node)
            (cd "$app_path" && npm run build) || return 1
            ;;
        migration)
            log_info "Skipping build for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Build completed for $app"
}

# Run command
cmd_run() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Running $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                cd "$app_path" && ./gradlew bootRun
            elif [ -f "$app_path/mvnw" ]; then
                cd "$app_path" && ./mvnw spring-boot:run
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            cd "$app_path" && go run .
            ;;
        node)
            cd "$app_path" && npm run dev
            ;;
        migration)
            log_info "Skipping run for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
}

# Docker build command
cmd_docker() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Building Docker image for $app..."
    
    case $app_type in
        migration)
            log_info "Skipping Docker build for migration-only app: $app"
            ;;
        *)
            docker build -t "$app:latest" "$app_path"
            log_success "Docker image built for $app"
            ;;
    esac
}

# Lint command
cmd_lint() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Linting $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                # Run Java quality checks: Spotless (formatting) and SpotBugs (bug detection)
                (cd "$app_path" && ./gradlew spotlessCheck spotbugsMain spotbugsTest) || return 1
            elif [ -f "$app_path/mvnw" ]; then
                log_warning "Maven linting not configured. Skipping..."
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            if command -v golangci-lint >/dev/null 2>&1; then
                (cd "$app_path" && golangci-lint run ./...) || return 1
            else
                log_warning "golangci-lint not found. Falling back to go vet..."
                (cd "$app_path" && go vet ./...) || return 1
            fi
            ;;
        node)
            (cd "$app_path" && npm run lint) || return 1
            ;;
        migration)
            log_info "Skipping lint for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Linting passed for $app"
}

# Lint-fix command (auto-fix lint errors)
cmd_lint-fix() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Auto-fixing lint errors for $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                # Run Spotless to auto-fix formatting issues
                (cd "$app_path" && ./gradlew spotlessApply) || return 1
                log_success "Spotless formatting applied"
                
                # Note: SpotBugs issues typically require manual fixes
                log_info "Note: SpotBugs issues may require manual fixes"
            elif [ -f "$app_path/mvnw" ]; then
                log_warning "Maven lint-fix not configured. Skipping..."
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            if command -v golangci-lint >/dev/null 2>&1; then
                (cd "$app_path" && golangci-lint run --fix ./...) || return 1
            else
                log_warning "golangci-lint not found. Falling back to gofmt..."
                if command -v goimports >/dev/null 2>&1; then
                    (cd "$app_path" && gofmt -w . && goimports -w .) || return 1
                else
                    (cd "$app_path" && gofmt -w .) || return 1
                fi
            fi
            ;;
        node)
            (cd "$app_path" && npm run lint -- --fix) || return 1
            ;;
        migration)
            log_info "Skipping lint-fix for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Lint errors auto-fixed for $app"
}

# Format command
cmd_format() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Formatting $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                (cd "$app_path" && ./gradlew spotlessApply) || return 1
            elif [ -f "$app_path/mvnw" ]; then
                log_warning "Maven formatting not configured. Skipping..."
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            # Use gofmt (always available) and goimports if available
            (cd "$app_path" && gofmt -w .) || return 1
            if command -v goimports >/dev/null 2>&1; then
                (cd "$app_path" && goimports -w .) || return 1
            else
                log_warning "goimports not found. Only gofmt was applied."
                log_warning "Install goimports: go install golang.org/x/tools/cmd/goimports@latest"
            fi
            ;;
        node)
            # Use npm run format which should be configured in package.json
            if [ -f "$app_path/package.json" ]; then
                (cd "$app_path" && npm run format) || return 1
            else
                log_error "package.json not found"
                return 1
            fi
            ;;
        migration)
            log_info "Skipping format for migration-only app: $app"
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Formatting completed for $app"
}

# Clean command
cmd_clean() {
    local app=$1
    local app_type=$(get_app_type "$app")
    local app_path=$(get_app_path "$app")
    
    log_info "Cleaning $app ($app_type)..."
    
    case $app_type in
        java)
            if [ -f "$app_path/gradlew" ]; then
                (cd "$app_path" && ./gradlew clean) || return 1
            elif [ -f "$app_path/mvnw" ]; then
                (cd "$app_path" && ./mvnw clean) || return 1
            else
                log_error "No build tool found (gradlew or mvnw)"
                return 1
            fi
            ;;
        go)
            (cd "$app_path" && rm -rf bin/) || return 1
            ;;
        node)
            (cd "$app_path" && rm -rf dist/ node_modules/.vite) || return 1
            ;;
        *)
            log_error "Unknown app type: $app_type"
            return 1
            ;;
    esac
    
    log_success "Cleaned $app"
}

# List command
cmd_list() {
    log_info "Available apps:"
    for app in $(get_all_apps); do
        local app_type=$(get_app_type "$app")
        echo "  - $app ($app_type)"
    done
}

# Main execution
main() {
    if [ -z "$COMMAND" ]; then
        log_error "No command specified"
        echo "Usage: $0 <command> [app-name]"
        echo ""
        echo "Commands:"
        echo "  test [app]       - Run tests"
        echo "  build [app]      - Build app"
        echo "  run [app]        - Run app locally"
        echo "  docker [app]     - Build Docker image"
        echo "  lint [app]       - Run linters"
        echo "  lint-fix [app]   - Auto-fix lint errors"
        echo "  clean [app]      - Clean build artifacts"
        echo "  format [app]     - Format code"
        echo "  list             - List all apps"
        exit 1
    fi
    
    case $COMMAND in
        list)
            cmd_list
            ;;
        test|build|run|docker|lint|lint-fix|clean|format)
            APPS=$(get_apps)
            FAILED_APPS=""
            
            for app in $APPS; do
                echo ""
                log_info "========================================="
                log_info "Processing: $app"
                log_info "========================================="
                
                if cmd_$COMMAND "$app"; then
                    log_success "✓ $COMMAND completed for $app"
                else
                    log_error "✗ $COMMAND failed for $app"
                    FAILED_APPS="$FAILED_APPS $app"
                fi
            done
            
            echo ""
            log_info "========================================="
            log_info "Summary"
            log_info "========================================="
            
            if [ -n "$FAILED_APPS" ]; then
                log_error "Failed apps:$FAILED_APPS"
                exit 1
            else
                log_success "All apps processed successfully!"
            fi
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            exit 1
            ;;
    esac
}

main
