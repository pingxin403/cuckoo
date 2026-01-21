#!/bin/bash

# Script to create a new app from template
# Usage: ./scripts/create-app.sh <app-type> <app-name> [options]

set -e

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

# Show usage
show_usage() {
    echo "Usage: $0 <app-type> <app-name> [options]"
    echo ""
    echo "App Types:"
    echo "  java    - Create a Java/Spring Boot service"
    echo "  go      - Create a Go service"
    echo "  node    - Create a Node.js/React application"
    echo ""
    echo "Options:"
    echo "  --port <port>           - gRPC port (default: auto-assign)"
    echo "  --description <desc>    - Service description"
    echo "  --package <package>     - Java package name (for Java apps)"
    echo "  --module <module>       - Go module path (for Go apps)"
    echo "  --proto <proto-file>    - Protobuf file name (without .proto)"
    echo "  --team <team-name>      - Team name for ownership"
    echo ""
    echo "Examples:"
    echo "  $0 java user-service --port 9092 --description 'User management service'"
    echo "  $0 go payment-service --port 9093 --module github.com/myorg/cuckoo/apps/payment-service"
    echo "  $0 node admin-dashboard --description 'Admin dashboard application'"
}

# Parse arguments
APP_TYPE="$1"
APP_NAME="$2"
shift 2 || true

# Default values
PORT=""
DESCRIPTION=""
PACKAGE=""
MODULE=""
PROTO_FILE=""
TEAM="platform-team"

# Parse options
while [[ $# -gt 0 ]]; do
    case $1 in
        --port)
            PORT="$2"
            shift 2
            ;;
        --description)
            DESCRIPTION="$2"
            shift 2
            ;;
        --package)
            PACKAGE="$2"
            shift 2
            ;;
        --module)
            MODULE="$2"
            shift 2
            ;;
        --proto)
            PROTO_FILE="$2"
            shift 2
            ;;
        --team)
            TEAM="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate inputs
if [ -z "$APP_TYPE" ] || [ -z "$APP_NAME" ]; then
    log_error "App type and name are required"
    show_usage
    exit 1
fi

if [[ ! "$APP_TYPE" =~ ^(java|go|node)$ ]]; then
    log_error "Invalid app type: $APP_TYPE"
    log_info "Valid types: java, go, node"
    exit 1
fi

# Check if app already exists
if [ -d "apps/$APP_NAME" ]; then
    log_error "App already exists: apps/$APP_NAME"
    exit 1
fi

# Auto-assign port if not provided
if [ -z "$PORT" ]; then
    # Find the highest port number in use
    MAX_PORT=$(grep -r "port.*909" apps/*/k8s/*.yaml 2>/dev/null | grep -oE "909[0-9]" | sort -n | tail -1 || echo "9091")
    PORT=$((MAX_PORT + 1))
    log_info "Auto-assigned port: $PORT"
fi

# Set default description if not provided
if [ -z "$DESCRIPTION" ]; then
    DESCRIPTION="$APP_NAME service"
fi

# Set default proto file name
if [ -z "$PROTO_FILE" ]; then
    PROTO_FILE=$(echo "$APP_NAME" | tr '-' '_')
fi

# Set default package for Java
if [ -z "$PACKAGE" ] && [ "$APP_TYPE" = "java" ]; then
    PACKAGE="com.pingxin403.cuckoo.$(echo $APP_NAME | tr '-' '.')"
fi

# Set default module for Go
if [ -z "$MODULE" ] && [ "$APP_TYPE" = "go" ]; then
    MODULE="github.com/pingxin403/cuckoo/apps/$APP_NAME"
fi

log_info "========================================="
log_info "Creating new $APP_TYPE app: $APP_NAME"
log_info "========================================="
log_info "Port: $PORT"
log_info "Description: $DESCRIPTION"
[ -n "$PACKAGE" ] && log_info "Package: $PACKAGE"
[ -n "$MODULE" ] && log_info "Module: $MODULE"
log_info "Proto file: $PROTO_FILE.proto"
log_info "Team: $TEAM"
log_info "========================================="

# Copy template
log_info "Copying template..."
cp -r "templates/${APP_TYPE}-service" "apps/$APP_NAME"

# Generate short name (remove -service suffix if present)
SHORT_NAME=$(echo "$APP_NAME" | sed 's/-service$//')

# Convert app name to different formats (needed for file renaming)
APP_NAME_UPPER=$(echo "$APP_NAME" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
APP_NAME_CAMEL=$(echo "$APP_NAME" | sed -r 's/(^|-)([a-z])/\U\2/g')
APP_NAME_SNAKE=$(echo "$APP_NAME" | tr '-' '_')

# Rename template files to match the service name (before content replacement)
if [ "$APP_TYPE" = "go" ]; then
    if [ -f "apps/$APP_NAME/service/template_service.go" ]; then
        mv "apps/$APP_NAME/service/template_service.go" "apps/$APP_NAME/service/${APP_NAME_SNAKE}_service.go"
    fi
    if [ -f "apps/$APP_NAME/service/template_service_test.go" ]; then
        mv "apps/$APP_NAME/service/template_service_test.go" "apps/$APP_NAME/service/${APP_NAME_SNAKE}_service_test.go"
    fi
fi

# Replace placeholders
log_info "Replacing placeholders..."

# Function to replace in file
replace_in_file() {
    local file=$1
    if [ -f "$file" ]; then
        # Escape special characters for sed
        local MODULE_ESC=$(echo "$MODULE" | sed 's/[\/&]/\\&/g')
        local PACKAGE_ESC=$(echo "$PACKAGE" | sed 's/[\/&]/\\&/g')
        local DESC_ESC=$(echo "$DESCRIPTION" | sed 's/[\/&]/\\&/g')
        
        sed -i.bak \
            -e "s/{{SERVICE_NAME}}/$APP_NAME/g" \
            -e "s/{{SHORT_NAME}}/$SHORT_NAME/g" \
            -e "s/{{SERVICE_NAME_UPPER}}/$APP_NAME_UPPER/g" \
            -e "s/{{SERVICE_NAME_CAMEL}}/$APP_NAME_CAMEL/g" \
            -e "s/{{SERVICE_NAME_SNAKE}}/$APP_NAME_SNAKE/g" \
            -e "s/{{ServiceName}}/$APP_NAME_CAMEL/g" \
            -e "s/{{SERVICE_DESCRIPTION}}/$DESC_ESC/g" \
            -e "s/{{PORT}}/$PORT/g" \
            -e "s/{{GRPC_PORT}}/$PORT/g" \
            -e "s/{{PACKAGE_NAME}}/$PACKAGE_ESC/g" \
            -e "s/{{MODULE_PATH}}/$MODULE_ESC/g" \
            -e "s/{{PROTO_FILE}}/$PROTO_FILE/g" \
            -e "s/{{PROTO_PACKAGE}}/${PROTO_FILE}pb/g" \
            -e "s/{{TEAM_NAME}}/$TEAM/g" \
            -e "s/templatepb/${PROTO_FILE}pb/g" \
            -e "s/Template/$APP_NAME_CAMEL/g" \
            "$file"
        rm -f "${file}.bak"
    fi
}

# Replace in all files (including metadata.yaml)
# Note: .apptype is now deprecated in favor of metadata.yaml
find "apps/$APP_NAME" -type f | while read file; do
    # Skip .git files
    if [[ ! "$file" =~ \.git/ ]]; then
        replace_in_file "$file"
    fi
done

log_success "App created successfully!"

# Update app-manager.sh
log_info "Updating app-manager.sh..."
APP_TYPE_SHORT=$(echo "$APP_TYPE" | sed 's/-service//')

# Add to get_app_type function
sed -i.bak "/get_app_type() {/,/^}/ {
    /web) echo \"node\" ;;/a\\
        $APP_NAME) echo \"$APP_TYPE_SHORT\" ;;
}" scripts/app-manager.sh

# Add to get_app_path function
sed -i.bak "/get_app_path() {/,/^}/ {
    /web) echo \"apps\/web\" ;;/a\\
        $APP_NAME) echo \"apps/$APP_NAME\" ;;
}" scripts/app-manager.sh

# Add to get_all_apps function
sed -i.bak "s/echo \"hello-service todo-service web\"/echo \"hello-service todo-service web $APP_NAME\"/" scripts/app-manager.sh

rm -f scripts/app-manager.sh.bak

log_success "Updated app-manager.sh"

# Create protobuf file
log_info "Creating protobuf file..."
cat > "api/v1/$PROTO_FILE.proto" <<EOF
syntax = "proto3";

package ${PROTO_FILE}pb;

option go_package = "github.com/pingxin403/cuckoo/apps/$APP_NAME/gen/${PROTO_FILE}pb";
option java_package = "$PACKAGE.proto";
option java_multiple_files = true;

// $DESCRIPTION
service ${APP_NAME_CAMEL}Service {
  // Add your RPC methods here
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

message HealthCheckRequest {}

message HealthCheckResponse {
  string status = 1;
}
EOF

log_success "Created api/v1/$PROTO_FILE.proto"

# Generate protobuf code for Go apps
if [ "$APP_TYPE" = "go" ]; then
    log_info "Generating protobuf code..."
    mkdir -p "apps/$APP_NAME/gen/${PROTO_FILE}pb"
    if protoc --go_out="apps/$APP_NAME/gen/${PROTO_FILE}pb" \
           --go_opt=paths=source_relative \
           --go-grpc_out="apps/$APP_NAME/gen/${PROTO_FILE}pb" \
           --go-grpc_opt=paths=source_relative \
           -I api/v1 \
           "api/v1/$PROTO_FILE.proto" 2>/dev/null; then
        log_success "Protobuf code generated"
    else
        log_warning "Failed to generate protobuf code. You may need to run 'make gen-proto' manually."
    fi
fi

# Initialize Go module dependencies if it's a Go app (after protobuf generation)
if [ "$APP_TYPE" = "go" ]; then
    log_info "Initializing Go module dependencies..."
    (cd "apps/$APP_NAME" && go mod tidy) || log_warning "Failed to run go mod tidy. You may need to run it manually."
    log_success "Go module dependencies initialized"
fi

# Show next steps
log_info ""
log_info "========================================="
log_info "Next Steps:"
log_info "========================================="
log_info "1. Define your service API in api/v1/$PROTO_FILE.proto"
log_info "2. Generate protobuf code: make gen-proto"
log_info "3. Implement your service logic in apps/$APP_NAME"
log_info "4. Build your app: make build APP=$APP_NAME"
log_info "5. Test your app: make test APP=$APP_NAME"
log_info ""
log_info "Your app is now integrated with:"
log_info "  ✓ App management system (make test/build/lint/etc)"
log_info "  ✓ Auto-detection for changed apps"
log_info "  ✓ CI/CD pipeline"
log_info "  ✓ Testing framework with coverage"
log_info "  ✓ Docker build support"
log_info "  ✓ Kubernetes deployment"
log_info ""
log_success "App $APP_NAME created successfully!"
