#!/bin/bash

# Proto Generator Script
# Automatically generates protobuf code for all apps based on their type
# Usage: ./scripts/proto-generator.sh [language] [app-name]
#
# Languages: go, java, ts, all (default)
# If app-name is omitted, generates for all apps

set -e

LANGUAGE="${1:-all}"
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

# Find proto files for an app
find_proto_files() {
    local app=$1
    local app_dir="apps/$app"
    local proto_files=""
    
    # Check if app has proto configuration in metadata.yaml
    if [ -f "$app_dir/metadata.yaml" ]; then
        # Look for proto_files section
        local in_proto_section=false
        while IFS= read -r line; do
            if [[ "$line" =~ ^[[:space:]]*proto_files: ]]; then
                in_proto_section=true
                continue
            fi
            if [ "$in_proto_section" = true ]; then
                # Check if we've left the proto_files section
                if [[ "$line" =~ ^[[:space:]]*[a-z_]+: ]] && [[ ! "$line" =~ ^[[:space:]]*- ]]; then
                    break
                fi
                # Extract proto file name
                if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*(.+)$ ]]; then
                    local proto_file="${BASH_REMATCH[1]}"
                    proto_files="$proto_files $proto_file"
                fi
            fi
        done < "$app_dir/metadata.yaml"
    fi
    
    # If no proto files found in metadata, try to infer from app name
    if [ -z "$proto_files" ]; then
        # Common patterns: hello-service -> hello.proto, todo-service -> todo.proto
        local base_name=$(echo "$app" | sed 's/-service$//')
        if [ -f "api/v1/${base_name}.proto" ]; then
            proto_files="${base_name}.proto"
        fi
        
        # Also check for service-specific proto
        if [ -f "api/v1/${app}.proto" ]; then
            proto_files="$proto_files ${app}.proto"
        fi
    fi
    
    echo "$proto_files" | xargs
}

# Generate Go proto code for an app
generate_go_proto() {
    local app=$1
    local app_dir="apps/$app"
    local proto_files=$(find_proto_files "$app")
    
    if [ -z "$proto_files" ]; then
        log_warning "No proto files found for $app, skipping Go generation"
        return 0
    fi
    
    log_info "Generating Go proto for $app"
    
    for proto_file in $proto_files; do
        local proto_name=$(basename "$proto_file" .proto)
        local gen_dir="$app_dir/gen/${proto_name}pb"
        
        log_info "  - Generating from $proto_file -> $gen_dir"
        
        mkdir -p "$gen_dir"
        
        protoc --go_out="$gen_dir" \
               --go_opt=paths=source_relative \
               --go-grpc_out="$gen_dir" \
               --go-grpc_opt=paths=source_relative \
               -I api/v1 \
               "api/v1/$proto_file" || return 1
    done
    
    log_success "Go proto generated for $app"
}

# Generate Java proto code for an app
generate_java_proto() {
    local app=$1
    local app_dir="apps/$app"
    local proto_files=$(find_proto_files "$app")
    
    if [ -z "$proto_files" ]; then
        log_warning "No proto files found for $app, skipping Java generation"
        return 0
    fi
    
    log_info "Generating Java proto for $app"
    log_info "Note: Java code generation is typically handled by Maven/Gradle plugins"
    
    # Check if protoc-gen-grpc-java is available
    if ! command -v protoc-gen-grpc-java >/dev/null 2>&1; then
        log_warning "protoc-gen-grpc-java not found. Java code will be generated by Maven/Gradle"
        return 0
    fi
    
    local gen_dir="$app_dir/src/main/java-gen"
    mkdir -p "$gen_dir"
    
    for proto_file in $proto_files; do
        log_info "  - Generating from $proto_file -> $gen_dir"
        
        protoc --java_out="$gen_dir" \
               --grpc-java_out="$gen_dir" \
               -I api/v1 \
               "api/v1/$proto_file" || return 1
    done
    
    log_success "Java proto generated for $app"
}

# Generate TypeScript proto code for an app
generate_ts_proto() {
    local app=$1
    local app_dir="apps/$app"
    
    if [ ! -d "$app_dir" ]; then
        log_error "App directory not found: $app_dir"
        return 1
    fi
    
    log_info "Generating TypeScript proto for $app"
    
    if [ -f "$app_dir/package.json" ]; then
        # Check if gen-proto script exists
        if grep -q '"gen-proto"' "$app_dir/package.json"; then
            (cd "$app_dir" && npm run gen-proto) || return 1
            log_success "TypeScript proto generated for $app"
        else
            log_warning "No gen-proto script found in package.json for $app"
        fi
    else
        log_warning "No package.json found for $app"
    fi
}

# Get all apps that need proto generation for a specific language
get_proto_apps() {
    local lang=$1
    local apps=""
    
    for app_dir in apps/*/; do
        local app=$(basename "$app_dir")
        local app_type=$(detect_app_type "$app_dir")
        local proto_files=$(find_proto_files "$app")
        
        # Skip if no proto files
        [ -z "$proto_files" ] && continue
        
        # Match app type with language
        case $lang in
            go)
                [ "$app_type" = "go" ] && apps="$apps $app"
                ;;
            java)
                [ "$app_type" = "java" ] && apps="$apps $app"
                ;;
            ts)
                [ "$app_type" = "node" ] && apps="$apps $app"
                ;;
            all)
                apps="$apps $app"
                ;;
        esac
    done
    
    echo "$apps" | xargs
}

# Generate proto for a single app
generate_app_proto() {
    local app=$1
    local lang=$2
    local app_dir="apps/$app"
    local app_type=$(detect_app_type "$app_dir")
    
    if [ ! -d "$app_dir" ]; then
        log_error "App directory not found: $app_dir"
        return 1
    fi
    
    case $lang in
        go)
            if [ "$app_type" = "go" ]; then
                generate_go_proto "$app"
            else
                log_warning "$app is not a Go app, skipping"
            fi
            ;;
        java)
            if [ "$app_type" = "java" ]; then
                generate_java_proto "$app"
            else
                log_warning "$app is not a Java app, skipping"
            fi
            ;;
        ts)
            if [ "$app_type" = "node" ]; then
                generate_ts_proto "$app"
            else
                log_warning "$app is not a Node.js app, skipping"
            fi
            ;;
        all)
            case $app_type in
                go)
                    generate_go_proto "$app"
                    ;;
                java)
                    generate_java_proto "$app"
                    ;;
                node)
                    generate_ts_proto "$app"
                    ;;
                *)
                    log_warning "Unknown app type for $app: $app_type"
                    ;;
            esac
            ;;
    esac
}

# Main execution
main() {
    if [ -n "$APP_NAME" ]; then
        # Generate for specific app
        local normalized_app=$(normalize_app_name "$APP_NAME")
        
        log_info "========================================="
        log_info "Generating proto for: $normalized_app"
        log_info "Language: $LANGUAGE"
        log_info "========================================="
        
        if generate_app_proto "$normalized_app" "$LANGUAGE"; then
            log_success "✓ Proto generation completed for $normalized_app"
        else
            log_error "✗ Proto generation failed for $normalized_app"
            exit 1
        fi
    else
        # Generate for all apps
        local apps=$(get_proto_apps "$LANGUAGE")
        
        if [ -z "$apps" ]; then
            log_warning "No apps found for proto generation (language: $LANGUAGE)"
            exit 0
        fi
        
        log_info "Apps for proto generation: $apps"
        log_info "Language: $LANGUAGE"
        echo ""
        
        local failed_apps=""
        
        for app in $apps; do
            log_info "========================================="
            log_info "Generating proto for: $app"
            log_info "========================================="
            
            if generate_app_proto "$app" "$LANGUAGE"; then
                log_success "✓ Proto generation completed for $app"
            else
                log_error "✗ Proto generation failed for $app"
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
            log_success "All proto generation completed!"
        fi
    fi
}

main
