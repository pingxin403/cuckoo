#!/usr/bin/env bash

# build-image.sh - Unified Docker image builder for monorepo
#
# This script provides a consistent interface for building Docker images
# with proper proto code validation, app type detection, and error handling.
#
# Usage:
#   ./scripts/build-image.sh [APP_NAME] [OPTIONS]
#
# Arguments:
#   APP_NAME    - Optional. Name of app to build. If omitted, builds changed apps.
#
# Options:
#   --tag TAG   - Custom tag to apply to image (default: git SHA)
#   --no-cache  - Disable Docker layer caching
#   --push      - Push image after building (for CI)
#
# Exit Codes:
#   0  - Success
#   1  - Proto validation failed
#   2  - App detection failed
#   3  - Docker build failed
#   4  - Docker not available
#
# Examples:
#   ./scripts/build-image.sh hello
#   ./scripts/build-image.sh hello --tag v1.0.0
#   ./scripts/build-image.sh --no-cache
#   ./scripts/build-image.sh hello --push

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_NAME=""
CUSTOM_TAG=""
NO_CACHE=false
PUSH=false
SUCCESSFUL_BUILDS=()
FAILED_BUILDS=()
SKIPPED_APPS=()

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
    echo -e "${GREEN}✅${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}⚠️${NC} $*"
}

log_error() {
    echo -e "${RED}❌${NC} $*" >&2
}

# Parse command-line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --tag)
                CUSTOM_TAG="$2"
                shift 2
                ;;
            --no-cache)
                NO_CACHE=true
                shift
                ;;
            --push)
                PUSH=true
                shift
                ;;
            --help|-h)
                head -n 30 "$0" | grep "^#" | sed 's/^# \?//'
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                exit 2
                ;;
            *)
                if [[ -z "$APP_NAME" ]]; then
                    APP_NAME="$1"
                else
                    log_error "Multiple app names provided: $APP_NAME and $1"
                    exit 2
                fi
                shift
                ;;
        esac
    done
}

# Validate Docker is available and running
validate_docker() {
    log_info "Validating Docker availability..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        log_error "Please install Docker: https://docs.docker.com/get-docker/"
        return 4
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        log_error "Please start Docker and try again"
        return 4
    fi
    
    # Check if BuildKit is available
    if ! docker buildx version &> /dev/null; then
        log_warning "Docker BuildKit (buildx) is not available"
        log_warning "Falling back to standard docker build (slower)"
    fi
    
    log_success "Docker is available"
    return 0
}

# Validate proto code is up to date
validate_proto_code() {
    log_info "Validating proto code..."
    
    cd "$REPO_ROOT"
    
    # Generate proto code
    log_info "Generating proto code..."
    if ! make proto > /dev/null 2>&1; then
        log_error "Proto generation failed"
        log_error "Run 'make proto' to see detailed error"
        return 1
    fi
    
    # Check if generated code is up to date
    if ! git diff --exit-code api/gen > /dev/null 2>&1; then
        log_error "Proto code is out of date"
        log_error "Generated files differ from committed files:"
        git diff --name-only api/gen | sed 's/^/  - /'
        log_error ""
        log_error "To fix this:"
        log_error "  1. Run: make proto"
        log_error "  2. Review changes: git diff api/gen"
        log_error "  3. Commit changes: git add api/gen && git commit -m 'chore: update proto generated code'"
        return 1
    fi
    
    log_success "Proto code is up to date"
    return 0
}

# Normalize app name (support short names)
normalize_app_name() {
    local input_name="$1"
    
    # Check if full name exists
    if [[ -d "$REPO_ROOT/apps/$input_name" ]]; then
        echo "$input_name"
        return 0
    fi
    
    # Search for short_name in metadata.yaml files
    for app_dir in "$REPO_ROOT/apps"/*; do
        if [[ -f "$app_dir/metadata.yaml" ]]; then
            local short_name
            short_name=$(grep "^  short_name:" "$app_dir/metadata.yaml" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
            if [[ "$short_name" == "$input_name" ]]; then
                basename "$app_dir"
                return 0
            fi
        fi
    done
    
    # Return input as-is if no match
    echo "$input_name"
}

# Detect app type
detect_app_type() {
    local app_name="$1"
    local app_dir="$REPO_ROOT/apps/$app_name"
    
    # Priority 1: Check metadata.yaml
    if [[ -f "$app_dir/metadata.yaml" ]]; then
        local app_type
        app_type=$(grep "^  type:" "$app_dir/metadata.yaml" 2>/dev/null | awk '{print $2}' | tr -d '[:space:]')
        if [[ -n "$app_type" ]]; then
            echo "$app_type"
            return 0
        fi
    fi
    
    # Priority 2: Check .apptype file (legacy)
    if [[ -f "$app_dir/.apptype" ]]; then
        cat "$app_dir/.apptype" | tr -d '[:space:]'
        return 0
    fi
    
    # Priority 3: Detect by file characteristics
    if [[ -f "$app_dir/build.gradle" ]] || [[ -f "$app_dir/pom.xml" ]]; then
        echo "java"
        return 0
    elif [[ -f "$app_dir/go.mod" ]]; then
        echo "go"
        return 0
    elif [[ -f "$app_dir/package.json" ]]; then
        echo "node"
        return 0
    else
        echo "unknown"
        return 0
    fi
}

# Detect changed apps
detect_changed_apps() {
    cd "$REPO_ROOT"
    
    if [[ -x "$REPO_ROOT/scripts/detect-changed-apps.sh" ]]; then
        "$REPO_ROOT/scripts/detect-changed-apps.sh" 2>/dev/null || echo ""
    else
        # Fallback: list all apps
        log_warning "detect-changed-apps.sh not found, building all apps"
        ls -1 "$REPO_ROOT/apps" | tr '\n' ' '
    fi
}

# Tag image with appropriate tags
tag_image() {
    local app_name="$1"
    
    cd "$REPO_ROOT"
    
    # Get git commit SHA
    local git_sha
    git_sha=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
    
    # Tag with commit SHA
    docker tag "$app_name:latest" "$app_name:$git_sha" 2>/dev/null || true
    log_info "Tagged: $app_name:$git_sha"
    
    # Get current branch
    local git_branch
    git_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
    
    if [[ "$git_branch" == "main" ]]; then
        log_info "On main branch, keeping 'latest' tag"
    elif [[ -n "$git_branch" ]] && [[ "$git_branch" != "HEAD" ]]; then
        docker tag "$app_name:latest" "$app_name:$git_branch" 2>/dev/null || true
        log_info "Tagged: $app_name:$git_branch"
    fi
    
    # Apply custom tag if provided
    if [[ -n "$CUSTOM_TAG" ]]; then
        docker tag "$app_name:latest" "$app_name:$CUSTOM_TAG" 2>/dev/null || true
        log_info "Tagged: $app_name:$CUSTOM_TAG"
    fi
}

# Build Docker image for an app
build_image() {
    local app_name="$1"
    local app_type="$2"
    
    log_info "Building Docker image for $app_name ($app_type)..."
    
    # Skip Node.js apps (no Docker images)
    if [[ "$app_type" == "node" ]]; then
        log_info "Skipping Node.js app (no Docker image)"
        SKIPPED_APPS+=("$app_name (node)")
        return 0
    fi
    
    # Skip unknown types
    if [[ "$app_type" == "unknown" ]]; then
        log_warning "Skipping app with unknown type"
        SKIPPED_APPS+=("$app_name (unknown)")
        return 0
    fi
    
    # Check if Dockerfile exists
    local dockerfile="$REPO_ROOT/apps/$app_name/Dockerfile"
    if [[ ! -f "$dockerfile" ]]; then
        log_error "Dockerfile not found: $dockerfile"
        FAILED_BUILDS+=("$app_name (no Dockerfile)")
        return 3
    fi
    
    cd "$REPO_ROOT"
    
    # Build command
    local build_cmd="docker"
    local build_args=()
    
    # Use buildx if available
    if docker buildx version &> /dev/null; then
        build_args+=("buildx" "build")
        export DOCKER_BUILDKIT=1
    else
        build_args+=("build")
    fi
    
    build_args+=(
        "--file" "$dockerfile"
        "--tag" "$app_name:latest"
        "--progress" "plain"
    )
    
    if [[ "$NO_CACHE" == "true" ]]; then
        build_args+=("--no-cache")
    fi
    
    build_args+=(".")
    
    # Build image
    log_info "Running: $build_cmd ${build_args[*]}"
    if "$build_cmd" "${build_args[@]}"; then
        log_success "Docker image built: $app_name:latest"
        
        # Tag image
        tag_image "$app_name"
        
        # Push if requested
        if [[ "$PUSH" == "true" ]]; then
            log_info "Pushing image..."
            docker push "$app_name:latest"
            log_success "Image pushed: $app_name:latest"
        fi
        
        SUCCESSFUL_BUILDS+=("$app_name")
        return 0
    else
        log_error "Docker build failed for $app_name"
        FAILED_BUILDS+=("$app_name")
        return 3
    fi
}

# Build multiple apps
build_multiple_apps() {
    local apps=("$@")
    local total=${#apps[@]}
    local current=0
    
    log_info "Building $total app(s)..."
    echo ""
    
    for app in "${apps[@]}"; do
        current=$((current + 1))
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_info "[$current/$total] Processing: $app"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Normalize app name
        local normalized_app
        normalized_app=$(normalize_app_name "$app")
        
        # Check if app exists
        if [[ ! -d "$REPO_ROOT/apps/$normalized_app" ]]; then
            log_error "App not found: $app"
            FAILED_BUILDS+=("$app (not found)")
            continue
        fi
        
        # Detect app type
        local app_type
        app_type=$(detect_app_type "$normalized_app")
        
        # Build image (continue on failure)
        build_image "$normalized_app" "$app_type" || true
        
        echo ""
    done
}

# Display build summary
display_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Build Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ ${#SUCCESSFUL_BUILDS[@]} -gt 0 ]]; then
        log_success "Successful builds (${#SUCCESSFUL_BUILDS[@]}):"
        for app in "${SUCCESSFUL_BUILDS[@]}"; do
            echo "  ✅ $app"
        done
        echo ""
    fi
    
    if [[ ${#SKIPPED_APPS[@]} -gt 0 ]]; then
        log_info "Skipped apps (${#SKIPPED_APPS[@]}):"
        for app in "${SKIPPED_APPS[@]}"; do
            echo "  ⏭️  $app"
        done
        echo ""
    fi
    
    if [[ ${#FAILED_BUILDS[@]} -gt 0 ]]; then
        log_error "Failed builds (${#FAILED_BUILDS[@]}):"
        for app in "${FAILED_BUILDS[@]}"; do
            echo "  ❌ $app"
        done
        echo ""
    fi
    
    local total=$((${#SUCCESSFUL_BUILDS[@]} + ${#FAILED_BUILDS[@]} + ${#SKIPPED_APPS[@]}))
    echo "Total: $total apps processed"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main function
main() {
    parse_args "$@"
    
    # Validate Docker
    validate_docker || exit $?
    
    # Validate proto code
    validate_proto_code || exit $?
    
    # Determine which apps to build
    local apps_to_build=()
    if [[ -n "$APP_NAME" ]]; then
        apps_to_build=("$APP_NAME")
    else
        # Detect changed apps
        local changed_apps
        changed_apps=$(detect_changed_apps)
        if [[ -z "$changed_apps" ]]; then
            log_warning "No changed apps detected"
            log_info "Specify an app name to build: ./scripts/build-image.sh <app-name>"
            exit 0
        fi
        read -ra apps_to_build <<< "$changed_apps"
    fi
    
    # Build apps
    build_multiple_apps "${apps_to_build[@]}"
    
    # Display summary
    display_summary
    
    # Exit with error if any builds failed
    if [[ ${#FAILED_BUILDS[@]} -gt 0 ]]; then
        exit 3
    fi
    
    exit 0
}

# Run main function
main "$@"
