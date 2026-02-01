# Multi-Region Active-Active MVP Makefile
# Provides convenient commands for managing the dual-region environment

.PHONY: help start stop restart status logs clean test chaos monitor network build

# Default target
help: ## Show this help message
	@echo "Multi-Region Active-Active MVP Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make start          # Start the entire environment"
	@echo "  make chaos          # Run chaos engineering tests"
	@echo "  make monitor        # Start real-time monitoring"
	@echo "  make logs service=im-service-a  # View specific service logs"

start: ## Start the multi-region environment
	@echo "🚀 Starting Multi-Region Active-Active MVP..."
	./scripts/start-mvp.sh

stop: ## Stop all services
	@echo "🛑 Stopping all services..."
	docker-compose down

restart: ## Restart all services
	@echo "🔄 Restarting all services..."
	docker-compose restart

status: ## Show service status
	@echo "📊 Service Status:"
	docker-compose ps
	@echo ""
	@echo "🏥 Health Checks:"
	@curl -s http://localhost:8080/health | jq '.status // "unhealthy"' | xargs echo "Region A:"
	@curl -s http://localhost:8081/health | jq '.status // "unhealthy"' | xargs echo "Region B:"
	@curl -s http://localhost:9999/health | jq '.status // "unhealthy"' | xargs echo "Arbiter:"

logs: ## Show logs (use service=<name> for specific service)
ifdef service
	@echo "📋 Logs for $(service):"
	docker-compose logs -f --tail=50 $(service)
else
	@echo "📋 All service logs:"
	docker-compose logs -f --tail=20
endif

clean: ## Clean up everything (containers, volumes, networks)
	@echo "🧹 Cleaning up everything..."
	docker-compose down --volumes --remove-orphans
	docker system prune -f
	@echo "✅ Cleanup completed"

build: ## Build all services
	@echo "🏗️  Building all services..."
	docker-compose build --no-cache

test: ## Run basic functionality tests
	@echo "🧪 Running basic tests..."
	./scripts/chaos-test.sh basic

chaos: ## Run chaos engineering tests
	@echo "🔥 Running chaos engineering tests..."
	./scripts/chaos-test.sh all

monitor: ## Start real-time monitoring
	@echo "📊 Starting real-time monitoring..."
	./scripts/monitor.sh

network: ## Run network tests
	@echo "🌐 Running network tests..."
	./scripts/network-test.sh

# Service-specific commands
start-region-a: ## Start only Region A services
	@echo "🚀 Starting Region A..."
	docker-compose up -d im-service-a im-gateway-a redis-region-a

start-region-b: ## Start only Region B services
	@echo "🚀 Starting Region B..."
	docker-compose up -d im-service-b im-gateway-b redis-region-b

stop-region-a: ## Stop Region A services (simulate failure)
	@echo "💥 Stopping Region A (simulating failure)..."
	docker-compose stop im-service-a im-gateway-a redis-region-a

stop-region-b: ## Stop Region B services (simulate failure)
	@echo "💥 Stopping Region B (simulating failure)..."
	docker-compose stop im-service-b im-gateway-b redis-region-b

# Monitoring shortcuts
prometheus: ## Open Prometheus in browser
	@echo "📊 Opening Prometheus..."
	@which open >/dev/null && open http://localhost:9090 || echo "Open http://localhost:9090 in your browser"

grafana: ## Open Grafana in browser
	@echo "📈 Opening Grafana..."
	@which open >/dev/null && open http://localhost:3000 || echo "Open http://localhost:3000 in your browser (admin/admin)"

arbiter: ## Show arbiter status
	@echo "🏛️  Arbiter Status:"
	@curl -s http://localhost:9999/status | jq '.' || echo "❌ Arbiter not available"

# Development helpers
shell-a: ## Open shell in Region A service
	docker exec -it im-service-region-a /bin/sh

shell-b: ## Open shell in Region B service
	docker exec -it im-service-region-b /bin/sh

shell-chaos: ## Open shell in network chaos container
	docker exec -it network-chaos /bin/sh

# Network manipulation
inject-latency: ## Inject high network latency (200ms)
	@echo "📡 Injecting high latency (200ms)..."
	docker exec network-chaos tc qdisc change dev eth0 root netem delay 200ms 50ms
	@echo "✅ High latency injected"

restore-latency: ## Restore normal network latency (40ms)
	@echo "📡 Restoring normal latency (40ms)..."
	docker exec network-chaos tc qdisc change dev eth0 root netem delay 40ms 10ms
	@echo "✅ Normal latency restored"

inject-packet-loss: ## Inject 10% packet loss
	@echo "📦 Injecting 10% packet loss..."
	docker exec network-chaos tc qdisc change dev eth0 root netem delay 40ms 10ms loss 10%
	@echo "✅ Packet loss injected"

network-partition: ## Create complete network partition
	@echo "🚫 Creating network partition..."
	docker exec network-chaos tc qdisc change dev eth0 root netem loss 100%
	@echo "⚠️  Network partition active - run 'make restore-network' to restore"

restore-network: ## Restore normal network conditions
	@echo "🔧 Restoring normal network..."
	docker exec network-chaos tc qdisc del dev eth0 root 2>/dev/null || true
	docker exec network-chaos tc qdisc add dev eth0 root handle 1: prio
	docker exec network-chaos tc qdisc add dev eth0 parent 1:1 handle 10: netem delay 40ms 10ms
	@echo "✅ Network restored"

# Quick demos
demo-basic: ## Run basic demo scenario
	@echo "🎬 Running basic demo..."
	@echo "1. Sending message to Region A..."
	@curl -X POST http://localhost:8080/api/messages -H "Content-Type: application/json" -d '{"conversation_id": "demo", "content": "Hello from Region A", "sender_id": "demo-user"}' -s | jq '.'
	@sleep 2
	@echo "2. Sending message to Region B..."
	@curl -X POST http://localhost:8081/api/messages -H "Content-Type: application/json" -d '{"conversation_id": "demo", "content": "Hello from Region B", "sender_id": "demo-user"}' -s | jq '.'
	@sleep 2
	@echo "3. Checking synchronization..."
	@echo "Region A messages:"
	@curl -s http://localhost:8080/api/messages/demo | jq '.'
	@echo "Region B messages:"
	@curl -s http://localhost:8081/api/messages/demo | jq '.'

demo-failover: ## Run failover demo scenario
	@echo "🎬 Running failover demo..."
	@echo "1. Sending initial message..."
	@curl -X POST http://localhost:8080/api/messages -H "Content-Type: application/json" -d '{"conversation_id": "failover-demo", "content": "Before failover", "sender_id": "demo-user"}' -s >/dev/null
	@sleep 2
	@echo "2. Simulating Region A failure..."
	@make stop-region-a
	@sleep 5
	@echo "3. Sending message to Region B (should work)..."
	@curl -X POST http://localhost:8081/api/messages -H "Content-Type: application/json" -d '{"conversation_id": "failover-demo", "content": "During failover", "sender_id": "demo-user"}' -s | jq '.'
	@sleep 2
	@echo "4. Restoring Region A..."
	@make start-region-a
	@sleep 10
	@echo "5. Checking final state..."
	@curl -s http://localhost:8080/api/messages/failover-demo | jq '.'

# Validation
validate: ## Validate the environment setup
	@echo "🔍 Validating environment..."
	@echo "Checking Docker..."
	@docker --version >/dev/null && echo "✅ Docker OK" || echo "❌ Docker not found"
	@echo "Checking Docker Compose..."
	@docker-compose --version >/dev/null && echo "✅ Docker Compose OK" || echo "❌ Docker Compose not found"
	@echo "Checking curl..."
	@curl --version >/dev/null && echo "✅ curl OK" || echo "❌ curl not found"
	@echo "Checking jq..."
	@jq --version >/dev/null && echo "✅ jq OK" || echo "❌ jq not found"
	@echo "Checking container status..."
	@docker-compose ps --format "table {{.Name}}\t{{.State}}"