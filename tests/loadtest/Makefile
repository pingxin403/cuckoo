.PHONY: help test test-short test-failover test-stability bench build run clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run all tests
	go test -v -timeout 10m

test-short: ## Run short tests only
	go test -v -short -timeout 2m

test-basic: ## Run basic load test
	go test -v -run TestBasicLoadTest -timeout 5m

test-failover: ## Run failover test
	go test -v -run TestFailoverLoadTest -timeout 10m

test-stability: ## Run long-running stability test
	go test -v -run TestLongRunningStabilityTest -timeout 30m

bench: ## Run benchmarks
	go test -bench=. -benchmem -benchtime=10s

build: ## Build the loadtest CLI tool
	go build -o bin/loadtest ./cmd/loadtest

run: build ## Run basic load test with CLI
	./bin/loadtest \
		-connections 1000 \
		-region-a-percent 50 \
		-rate 100 \
		-duration 1m \
		-rampup 10s \
		-cross-region 0.3 \
		-region-a ws://localhost:8080/ws \
		-region-b ws://localhost:8081/ws \
		-output results.json

run-failover: build ## Run failover test with CLI
	./bin/loadtest \
		-failover \
		-connections 1000 \
		-region-a-percent 50 \
		-rate 100 \
		-duration 2m \
		-rampup 10s \
		-cross-region 0.3 \
		-failover-delay 30s \
		-failover-recovery 5s \
		-failed-region region-a \
		-region-a ws://localhost:8080/ws \
		-region-b ws://localhost:8081/ws \
		-output failover-results.json

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f *.json
	go clean -testcache

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (if installed)
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

# Production load tests
prod-small: build ## Run small-scale production test (10K connections)
	./bin/loadtest \
		-connections 10000 \
		-region-a-percent 50 \
		-rate 1000 \
		-duration 5m \
		-rampup 1m \
		-cross-region 0.3 \
		-output prod-small-results.json

prod-medium: build ## Run medium-scale production test (50K connections)
	./bin/loadtest \
		-connections 50000 \
		-region-a-percent 50 \
		-rate 5000 \
		-duration 10m \
		-rampup 2m \
		-cross-region 0.3 \
		-output prod-medium-results.json

prod-large: build ## Run large-scale production test (100K connections)
	./bin/loadtest \
		-connections 100000 \
		-region-a-percent 50 \
		-rate 10000 \
		-duration 30m \
		-rampup 5m \
		-cross-region 0.3 \
		-output prod-large-results.json

prod-stability: build ## Run 24-hour stability test
	./bin/loadtest \
		-connections 100000 \
		-region-a-percent 50 \
		-rate 10000 \
		-duration 24h \
		-rampup 10m \
		-cross-region 0.3 \
		-output prod-stability-results.json
