#!/bin/bash

# Unified Dependency Manager
# Manages dependencies across Go, Java, and Node.js projects

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Action and target
ACTION=${1:-install}
TARGET=${2:-all}

# Load version requirements
if [ -f "$ROOT_DIR/.tool-versions" ]; then
    source "$ROOT_DIR/.tool-versions"
fi

# Function to print colored messages
print_info() {
    echo -e "${BLUE}ℹ ${1}${NC}"
}

print_success() {
    echo -e "${GREEN}✓ ${1}${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ ${1}${NC}"
}

print_error() {
    echo -e "${RED}✗ ${1}${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# ===== Go Dependency Management =====

deps_go_install() {
    print_info "Installing Go dependencies..."
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/go.mod" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && go mod download && go mod verify) || {
                print_error "Failed to install Go dependencies for $app_name"
                return 1
            }
            count=$((count + 1))
        fi
    done
    
    # Also handle libs
    for dir in "$ROOT_DIR"/libs/*/; do
        if [ -f "$dir/go.mod" ]; then
            local lib_name=$(basename "$dir")
            print_info "  → libs/$lib_name"
            (cd "$dir" && go mod download && go mod verify) || {
                print_error "Failed to install Go dependencies for libs/$lib_name"
                return 1
            }
            count=$((count + 1))
        fi
    done
    
    print_success "Installed Go dependencies for $count modules"
}

deps_go_update() {
    print_info "Updating Go dependencies..."
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/go.mod" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && go get -u ./... && go mod tidy) || {
                print_warning "Failed to update Go dependencies for $app_name"
            }
            count=$((count + 1))
        fi
    done
    
    # Also handle libs
    for dir in "$ROOT_DIR"/libs/*/; do
        if [ -f "$dir/go.mod" ]; then
            local lib_name=$(basename "$dir")
            print_info "  → libs/$lib_name"
            (cd "$dir" && go get -u ./... && go mod tidy) || {
                print_warning "Failed to update Go dependencies for libs/$lib_name"
            }
            count=$((count + 1))
        fi
    done
    
    print_success "Updated Go dependencies for $count modules"
}

deps_go_clean() {
    print_info "Cleaning Go dependencies..."
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/go.mod" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && go clean -modcache) || true
            count=$((count + 1))
        fi
    done
    
    for dir in "$ROOT_DIR"/libs/*/; do
        if [ -f "$dir/go.mod" ]; then
            local lib_name=$(basename "$dir")
            print_info "  → libs/$lib_name"
            (cd "$dir" && go clean -modcache) || true
            count=$((count + 1))
        fi
    done
    
    print_success "Cleaned Go dependencies for $count modules"
}

deps_go_verify() {
    print_info "Verifying Go dependencies..."
    local count=0
    local errors=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/go.mod" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && go mod verify) || {
                print_error "Verification failed for $app_name"
                errors=$((errors + 1))
            }
            count=$((count + 1))
        fi
    done
    
    for dir in "$ROOT_DIR"/libs/*/; do
        if [ -f "$dir/go.mod" ]; then
            local lib_name=$(basename "$dir")
            print_info "  → libs/$lib_name"
            (cd "$dir" && go mod verify) || {
                print_error "Verification failed for libs/$lib_name"
                errors=$((errors + 1))
            }
            count=$((count + 1))
        fi
    done
    
    if [ $errors -eq 0 ]; then
        print_success "Verified Go dependencies for $count modules"
    else
        print_error "Verification failed for $errors modules"
        return 1
    fi
}

deps_go_audit() {
    print_info "Auditing Go dependencies for security issues..."
    if ! command_exists govulncheck; then
        print_warning "govulncheck not installed. Installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    local count=0
    local issues=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/go.mod" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && govulncheck ./...) || {
                print_warning "Security issues found in $app_name"
                issues=$((issues + 1))
            }
            count=$((count + 1))
        fi
    done
    
    for dir in "$ROOT_DIR"/libs/*/; do
        if [ -f "$dir/go.mod" ]; then
            local lib_name=$(basename "$dir")
            print_info "  → libs/$lib_name"
            (cd "$dir" && govulncheck ./...) || {
                print_warning "Security issues found in libs/$lib_name"
                issues=$((issues + 1))
            }
            count=$((count + 1))
        fi
    done
    
    if [ $issues -eq 0 ]; then
        print_success "No security issues found in $count Go modules"
    else
        print_warning "Security issues found in $issues modules"
    fi
}

# ===== Java Dependency Management =====

deps_java_install() {
    print_info "Installing Java dependencies..."
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/build.gradle" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && ./gradlew dependencies --refresh-dependencies) || {
                print_error "Failed to install Java dependencies for $app_name"
                return 1
            }
            count=$((count + 1))
        fi
    done
    
    print_success "Installed Java dependencies for $count modules"
}

deps_java_update() {
    print_info "Updating Java dependencies..."
    print_warning "Java dependency updates require manual review"
    print_info "To check for updates, you can:"
    print_info "  1. Add the versions plugin to build.gradle:"
    print_info "     plugins { id 'com.github.ben-manes.versions' version '0.51.0' }"
    print_info "  2. Run: ./gradlew dependencyUpdates"
    print_info "  3. Or use: ./gradlew dependencies to view current versions"
    echo ""
    
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/build.gradle" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name: ./gradlew dependencies"
            count=$((count + 1))
        fi
    done
    
    print_success "Found $count Java modules"
    print_info "Update versions manually in build.gradle files"
}

deps_java_clean() {
    print_info "Cleaning Java dependencies..."
    local count=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/build.gradle" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && ./gradlew clean) || true
            count=$((count + 1))
        fi
    done
    
    print_success "Cleaned Java dependencies for $count modules"
}

deps_java_verify() {
    print_info "Verifying Java dependencies..."
    local count=0
    local errors=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/build.gradle" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && ./gradlew dependencies --verify-metadata) || {
                print_error "Verification failed for $app_name"
                errors=$((errors + 1))
            }
            count=$((count + 1))
        fi
    done
    
    if [ $errors -eq 0 ]; then
        print_success "Verified Java dependencies for $count modules"
    else
        print_error "Verification failed for $errors modules"
        return 1
    fi
}

deps_java_audit() {
    print_info "Auditing Java dependencies for security issues..."
    local count=0
    local issues=0
    for dir in "$ROOT_DIR"/apps/*/; do
        if [ -f "$dir/build.gradle" ]; then
            local app_name=$(basename "$dir")
            print_info "  → $app_name"
            (cd "$dir" && ./gradlew dependencyCheckAnalyze) || {
                print_warning "Security issues found in $app_name"
                issues=$((issues + 1))
            }
            count=$((count + 1))
        fi
    done
    
    if [ $issues -eq 0 ]; then
        print_success "No security issues found in $count Java modules"
    else
        print_warning "Security issues found in $issues modules"
    fi
}

# ===== Node.js Dependency Management =====

deps_node_install() {
    print_info "Installing Node.js dependencies..."
    if [ -d "$ROOT_DIR/apps/web" ]; then
        print_info "  → apps/web"
        (cd "$ROOT_DIR/apps/web" && npm ci) || {
            print_error "Failed to install Node.js dependencies"
            return 1
        }
        print_success "Installed Node.js dependencies"
    else
        print_warning "No Node.js projects found"
    fi
}

deps_node_update() {
    print_info "Updating Node.js dependencies..."
    if [ -d "$ROOT_DIR/apps/web" ]; then
        print_info "  → apps/web"
        (cd "$ROOT_DIR/apps/web" && npm update) || {
            print_warning "Failed to update Node.js dependencies"
        }
        print_success "Updated Node.js dependencies"
    else
        print_warning "No Node.js projects found"
    fi
}

deps_node_clean() {
    print_info "Cleaning Node.js dependencies..."
    if [ -d "$ROOT_DIR/apps/web" ]; then
        print_info "  → apps/web"
        (cd "$ROOT_DIR/apps/web" && rm -rf node_modules package-lock.json) || true
        print_success "Cleaned Node.js dependencies"
    else
        print_warning "No Node.js projects found"
    fi
}

deps_node_verify() {
    print_info "Verifying Node.js dependencies..."
    if [ -d "$ROOT_DIR/apps/web" ]; then
        print_info "  → apps/web"
        (cd "$ROOT_DIR/apps/web" && npm ls) || {
            print_error "Verification failed"
            return 1
        }
        print_success "Verified Node.js dependencies"
    else
        print_warning "No Node.js projects found"
    fi
}

deps_node_audit() {
    print_info "Auditing Node.js dependencies for security issues..."
    if [ -d "$ROOT_DIR/apps/web" ]; then
        print_info "  → apps/web"
        (cd "$ROOT_DIR/apps/web" && npm audit) || {
            print_warning "Security issues found"
        }
    else
        print_warning "No Node.js projects found"
    fi
}

# ===== Protobuf Tools =====

deps_proto_install() {
    print_info "Installing Protobuf tools..."
    if command_exists go; then
        go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION:-latest}
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION:-latest}
        print_success "Installed Protobuf tools"
    else
        print_error "Go not found. Cannot install Protobuf tools"
        return 1
    fi
}

# ===== Main Logic =====

case $ACTION in
    install)
        case $TARGET in
            all)
                deps_go_install
                deps_java_install
                deps_node_install
                deps_proto_install
                ;;
            go)
                deps_go_install
                ;;
            java)
                deps_java_install
                ;;
            node)
                deps_node_install
                ;;
            proto)
                deps_proto_install
                ;;
            *)
                print_error "Unknown target: $TARGET"
                print_info "Valid targets: all, go, java, node, proto"
                exit 1
                ;;
        esac
        ;;
    
    update)
        case $TARGET in
            all)
                deps_go_update
                deps_java_update
                deps_node_update
                ;;
            go)
                deps_go_update
                ;;
            java)
                deps_java_update
                ;;
            node)
                deps_node_update
                ;;
            *)
                print_error "Unknown target: $TARGET"
                print_info "Valid targets: all, go, java, node"
                exit 1
                ;;
        esac
        ;;
    
    clean)
        case $TARGET in
            all)
                deps_go_clean
                deps_java_clean
                deps_node_clean
                ;;
            go)
                deps_go_clean
                ;;
            java)
                deps_java_clean
                ;;
            node)
                deps_node_clean
                ;;
            *)
                print_error "Unknown target: $TARGET"
                print_info "Valid targets: all, go, java, node"
                exit 1
                ;;
        esac
        ;;
    
    verify)
        case $TARGET in
            all)
                deps_go_verify
                deps_java_verify
                deps_node_verify
                ;;
            go)
                deps_go_verify
                ;;
            java)
                deps_java_verify
                ;;
            node)
                deps_node_verify
                ;;
            *)
                print_error "Unknown target: $TARGET"
                print_info "Valid targets: all, go, java, node"
                exit 1
                ;;
        esac
        ;;
    
    audit)
        case $TARGET in
            all)
                deps_go_audit
                deps_java_audit
                deps_node_audit
                ;;
            go)
                deps_go_audit
                ;;
            java)
                deps_java_audit
                ;;
            node)
                deps_node_audit
                ;;
            *)
                print_error "Unknown target: $TARGET"
                print_info "Valid targets: all, go, java, node"
                exit 1
                ;;
        esac
        ;;
    
    status)
        print_info "Dependency Status:"
        echo ""
        print_info "Go modules:"
        find "$ROOT_DIR/apps" "$ROOT_DIR/libs" -name "go.mod" 2>/dev/null | while read -r mod; do
            echo "  - $(dirname "$mod" | sed "s|$ROOT_DIR/||")"
        done
        echo ""
        print_info "Java modules:"
        find "$ROOT_DIR/apps" -name "build.gradle" 2>/dev/null | while read -r gradle; do
            echo "  - $(dirname "$gradle" | sed "s|$ROOT_DIR/||")"
        done
        echo ""
        print_info "Node.js modules:"
        find "$ROOT_DIR/apps" -name "package.json" 2>/dev/null | while read -r pkg; do
            echo "  - $(dirname "$pkg" | sed "s|$ROOT_DIR/||")"
        done
        ;;
    
    *)
        print_error "Unknown action: $ACTION"
        echo ""
        echo "Usage: $0 <action> [target]"
        echo ""
        echo "Actions:"
        echo "  install  - Install dependencies"
        echo "  update   - Update dependencies"
        echo "  clean    - Clean dependencies"
        echo "  verify   - Verify dependencies"
        echo "  audit    - Security audit"
        echo "  status   - Show dependency status"
        echo ""
        echo "Targets:"
        echo "  all      - All languages (default)"
        echo "  go       - Go modules only"
        echo "  java     - Java modules only"
        echo "  node     - Node.js modules only"
        echo "  proto    - Protobuf tools only (install action only)"
        echo ""
        echo "Examples:"
        echo "  $0 install all"
        echo "  $0 update go"
        echo "  $0 audit node"
        exit 1
        ;;
esac

print_success "Done!"
