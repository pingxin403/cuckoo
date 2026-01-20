.PHONY: help init check-env check-versions proto gen-proto gen-proto-go gen-proto-java gen-proto-ts verify-proto \
        build test lint lint-fix format docker-build run clean list-apps create \
        test-coverage verify-coverage \
        dev pre-commit verify-auto-detection prepare-k8s-resources

# Default target
help:
	@echo "Monorepo Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  init               - Initialize development environment and install dependencies"
	@echo "  check-env          - Check if all required tools are installed"
	@echo "  check-versions     - Verify tool versions match requirements"
	@echo "  verify-auto-detection - Verify app type auto-detection works correctly"
	@echo "  proto              - Generate code from Protobuf definitions (all languages)"
	@echo "  verify-proto       - Verify generated code is up to date (for CI)"
	@echo ""
	@echo "  Quality & Testing:"
	@echo "  pre-commit         - Run all pre-commit quality checks (lint, test, security)"
	@echo "  lint [APP=name]    - Run linters for app(s)"
	@echo "  lint-fix [APP=name] - Auto-fix lint errors for app(s)"
	@echo "  format [APP=name]  - Format code for app(s)"
	@echo "  test [APP=name]    - Run tests for app(s)"
	@echo "  test-coverage [APP=name] - Run tests with coverage for app(s)"
	@echo "  verify-coverage [APP=name] - Verify coverage thresholds for app(s)"
	@echo ""
	@echo "  Kubernetes:"
	@echo "  prepare-k8s-resources - Prepare K8s resources for Kustomize build"
	@echo ""
	@echo "  App Management (supports APP=<name> or auto-detects changed apps):"
	@echo "  list-apps          - List all available apps"
	@echo "  create             - Create a new app from template"
	@echo "  build [APP=name]   - Build app(s)"
	@echo "  docker-build [APP=name] - Build Docker image(s)"
	@echo "  run [APP=name]     - Run app(s) locally"
	@echo "  clean [APP=name]   - Clean build artifacts for app(s)"
	@echo ""
	@echo "  dev                - Start all services in development mode"
	@echo ""
	@echo "Examples:"
	@echo "  make pre-commit                # Run all quality checks before commit"
	@echo "  make proto                     # Generate code from Protobuf"
	@echo "  make test APP=hello            # Test specific app (short name)"
	@echo "  make test-coverage APP=shortener # Run coverage for specific app"
	@echo "  make test-coverage             # Run coverage for all apps"
	@echo "  make lint-fix                  # Fix linting issues in all changed apps"
	@echo "  make build APP=todo            # Build specific app (short name)"

# Initialization
init:
	@echo "Initializing development environment..."
	@./scripts/init.sh

# Environment check
check-env:
	@./scripts/check-env.sh

# Verify auto-detection functionality
verify-auto-detection:
	@./scripts/verify-auto-detection.sh

# Version check
check-versions:
	@./scripts/check-versions.sh

# Protobuf code generation
proto: gen-proto-go gen-proto-java gen-proto-ts
	@echo "✅ Protobuf code generation completed for all languages"

# Legacy alias for backward compatibility
gen-proto: proto
	@echo "Note: 'gen-proto' is deprecated. Use 'make proto' instead."

# Convenience aliases for CI (without gen- prefix)
proto-go: gen-proto-go
proto-java: gen-proto-java
proto-ts: gen-proto-ts

gen-proto-go:
	@./scripts/proto-generator.sh go

gen-proto-java:
	@./scripts/proto-generator.sh java

gen-proto-ts:
	@./scripts/proto-generator.sh ts

# CI verification
verify-proto:
	@echo "Verifying generated code is up to date..."
	@$(MAKE) proto
	@git diff --exit-code apps/*/gen apps/*/src/main/java-gen apps/*/src/gen || \
	  (echo "Generated code is out of date. Run 'make proto' and commit changes." && exit 1)

# App management targets (new unified interface)
list-apps:
	@./scripts/app-manager.sh list

create:
	@echo "Create a new app from template"
	@echo ""
	@read -p "App type (java/go/node): " app_type; \
	read -p "App name (e.g., user-service): " app_name; \
	read -p "Port (default: auto-assign): " port; \
	read -p "Description: " description; \
	if [ "$$app_type" = "java" ]; then \
		read -p "Java package (default: com.pingxin403.cuckoo.$$app_name): " package; \
	fi; \
	if [ "$$app_type" = "go" ]; then \
		read -p "Go module (default: github.com/pingxin403/cuckoo/apps/$$app_name): " module; \
	fi; \
	read -p "Team name (default: platform-team): " team; \
	cmd="./scripts/create-app.sh $$app_type $$app_name"; \
	[ -n "$$port" ] && cmd="$$cmd --port $$port"; \
	[ -n "$$description" ] && cmd="$$cmd --description \"$$description\""; \
	[ -n "$$package" ] && cmd="$$cmd --package $$package"; \
	[ -n "$$module" ] && cmd="$$cmd --module $$module"; \
	[ -n "$$team" ] && cmd="$$cmd --team $$team"; \
	eval $$cmd

build:
ifdef APP
	@./scripts/app-manager.sh build $(APP)
else
	@./scripts/app-manager.sh build
endif

test:
ifdef APP
	@./scripts/app-manager.sh test $(APP)
else
	@./scripts/app-manager.sh test
endif

lint:
ifdef APP
	@./scripts/app-manager.sh lint $(APP)
else
	@./scripts/app-manager.sh lint
endif

lint-fix:
ifdef APP
	@./scripts/app-manager.sh lint-fix $(APP)
else
	@./scripts/app-manager.sh lint-fix
endif

format:
ifdef APP
	@./scripts/app-manager.sh format $(APP)
else
	@./scripts/app-manager.sh format
endif

docker-build:
ifdef APP
	@./scripts/app-manager.sh docker $(APP)
else
	@./scripts/app-manager.sh docker
endif

run:
ifdef APP
	@./scripts/app-manager.sh run $(APP)
else
	@echo "Error: APP parameter required for run command"
	@echo "Usage: make run APP=<app-name>"
	@echo "Available apps:"
	@./scripts/app-manager.sh list
	@exit 1
endif

clean:
ifdef APP
	@./scripts/app-manager.sh clean $(APP)
else
	@./scripts/app-manager.sh clean
endif

# Test coverage targets (unified interface)
test-coverage:
ifdef APP
	@./scripts/coverage-manager.sh $(APP)
else
	@./scripts/coverage-manager.sh
endif

# Verify coverage thresholds (for CI)
verify-coverage:
ifdef APP
	@./scripts/coverage-manager.sh $(APP) --verify
else
	@./scripts/coverage-manager.sh --verify
endif

# Development mode
dev:
	@echo "Starting all services in development mode..."
	@./scripts/dev.sh

# Pre-commit checks (run all quality checks before commit)
pre-commit:
	@echo "Running pre-commit checks..."
	@./scripts/pre-commit-checks.sh

# Kubernetes resource preparation
prepare-k8s-resources:
	@echo "Preparing K8s resources for Kustomize..."
	@./scripts/prepare-k8s-resources.sh
