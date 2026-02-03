# Multi-Region Active-Active MVP Environment

This directory contains a complete Docker Compose setup for simulating a dual-region active-active architecture locally. It's designed to demonstrate the core concepts of cross-region synchronization, conflict resolution, and failover mechanisms without requiring cloud infrastructure.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Local MVP Environment                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    Cross-Region Network    ┌─────────────────┐        │
│  │   Region A      │◄──── (30-50ms latency) ────►│   Region B      │        │
│  │                 │                             │                 │        │
│  │ ┌─────────────┐ │                             │ ┌─────────────┐ │        │
│  │ │ IM Gateway  │ │                             │ │ IM Gateway  │ │        │
│  │ │ :8080       │ │                             │ │ :8081       │ │        │
│  │ └─────────────┘ │                             │ └─────────────┘ │        │
│  │ ┌─────────────┐ │                             │ ┌─────────────┐ │        │
│  │ │ IM Service  │ │                             │ │ IM Service  │ │        │
│  │ │ + HLC       │ │                             │ │ + HLC       │ │        │
│  │ └─────────────┘ │                             │ └─────────────┘ │        │
│  │ ┌─────────────┐ │                             │ ┌─────────────┐ │        │
│  │ │ Redis       │ │                             │ │ Redis       │ │        │
│  │ │ (Cache)     │ │                             │ │ (Cache)     │ │        │
│  │ └─────────────┘ │                             │ └─────────────┘ │        │
│  │ ┌─────────────┐ │                             │ ┌─────────────┐ │        │
│  │ │ SQLite      │ │                             │ │ SQLite      │ │        │
│  │ │ (Storage)   │ │                             │ │ (Storage)   │ │        │
│  │ └─────────────┘ │                             │ └─────────────┘ │        │
│  └─────────────────┘                             └─────────────────┘        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                        Shared Infrastructure                            │ │
│  │                                                                         │ │
│  │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │ │
│  │ │ Kafka       │ │ Zookeeper   │ │ Arbiter     │ │ Network     │       │ │
│  │ │ (Messaging) │ │ (Coord)     │ │ Mock        │ │ Chaos       │       │ │
│  │ │             │ │             │ │ :9999       │ │ (tc)        │       │ │
│  │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                           Monitoring                                    │ │
│  │                                                                         │ │
│  │ ┌─────────────┐ ┌─────────────┐                                        │ │
│  │ │ Prometheus  │ │ Grafana     │                                        │ │
│  │ │ :9090       │ │ :3000       │                                        │ │
│  │ └─────────────┘ └─────────────┘                                        │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker (20.10+)
- Docker Compose (2.0+)
- curl and jq (for testing scripts)
- At least 4GB RAM available for containers

### 1. Start the Environment

```bash
# Navigate to MVP directory
cd deploy/mvp

# Start all services
./scripts/start-mvp.sh
```

This will:
- Build and start all containers
- Configure network isolation and latency simulation
- Set up monitoring and observability
- Perform health checks on all services

### 2. Verify the Setup

```bash
# Check service health
curl http://localhost:8080/health  # Region A
curl http://localhost:8081/health  # Region B
curl http://localhost:9999/health  # Arbiter

# View real-time monitoring
./scripts/monitor.sh

# Test network configuration
./scripts/network-test.sh
```

### 3. Run Chaos Tests

```bash
# Run all chaos engineering tests
./scripts/chaos-test.sh

# Or run specific tests
./scripts/chaos-test.sh basic      # Basic functionality
./scripts/chaos-test.sh failover-a # Region A failure
./scripts/chaos-test.sh partition  # Network partition
```

## 📊 Monitoring and Observability

### Grafana Dashboard
- **URL**: http://localhost:3000
- **Credentials**: admin/admin
- **Dashboards**: Multi-Region Active-Active Overview

### Prometheus Metrics
- **URL**: http://localhost:9090
- **Key Metrics**:
  - `hlc_physical_time_ms` - HLC physical clock
  - `hlc_logical_time` - HLC logical counter
  - `sync_latency_seconds` - Cross-region sync latency
  - `conflict_total` - Message conflicts detected
  - `failover_events_total` - Failover events

### Arbiter Status
- **URL**: http://localhost:9999/status
- **Information**: Current primary region, health status, election history

## 🧪 Testing Scenarios

### 1. Basic Message Synchronization
```bash
# Send message to Region A
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -d '{"conversation_id": "test", "content": "Hello from A"}'

# Send message to Region B
curl -X POST http://localhost:8081/api/messages \
  -H "Content-Type: application/json" \
  -d '{"conversation_id": "test", "content": "Hello from B"}'

# Check synchronization
curl http://localhost:8080/api/messages/test
curl http://localhost:8081/api/messages/test
```

### 2. Network Latency Simulation
```bash
# Inject high latency (200ms)
docker exec network-chaos tc qdisc change dev eth0 root netem delay 200ms 50ms

# Test message sync with high latency
./scripts/chaos-test.sh latency

# Restore normal latency (40ms)
docker exec network-chaos tc qdisc change dev eth0 root netem delay 40ms 10ms
```

### 3. Region Failover
```bash
# Simulate Region A failure
docker stop im-service-region-a im-gateway-region-a

# Verify Region B takes over
curl http://localhost:9999/status

# Send messages to Region B only
curl -X POST http://localhost:8081/api/messages \
  -H "Content-Type: application/json" \
  -d '{"conversation_id": "failover", "content": "During failover"}'

# Restore Region A
docker start im-service-region-a im-gateway-region-a

# Verify data synchronization after recovery
```

### 4. Split-Brain Prevention
```bash
# Create network partition
docker exec network-chaos tc qdisc change dev eth0 root netem loss 100%

# Check arbiter decision
curl http://localhost:9999/status

# Restore network
docker exec network-chaos tc qdisc change dev eth0 root netem delay 40ms 10ms
```

## 🔧 Configuration

### Environment Variables

Each service can be configured through environment variables in `docker-compose.yml`:

#### IM Service
- `REGION_ID`: Region identifier (region-a, region-b)
- `PEER_REGION`: Peer region identifier
- `PEER_ENDPOINT`: Peer service endpoint
- `DATABASE_PATH`: SQLite database path
- `REDIS_ADDR`: Redis connection address
- `ARBITER_ENDPOINT`: Arbiter service endpoint

#### Network Chaos
- Latency: Configured via `tc netem delay` commands
- Packet Loss: Configured via `tc netem loss` commands
- Bandwidth: Configured via `tc htb` commands

### Network Configuration

Three isolated networks:
- `region-a-net` (172.20.0.0/24): Region A internal
- `region-b-net` (172.21.0.0/24): Region B internal  
- `cross-region-net` (172.22.0.0/24): Cross-region communication

## 📁 File Structure

```
deploy/mvp/
├── docker-compose.yml           # Main orchestration file
├── prometheus.yml               # Prometheus configuration
├── multi_region_alerts.yml      # Alert rules
├── grafana/
│   ├── datasources/
│   │   └── prometheus.yml       # Grafana datasource
│   └── dashboards/
│       ├── dashboard.yml        # Dashboard provider
│       └── multi-region-overview.json
├── scripts/
│   ├── start-mvp.sh            # Environment startup
│   ├── chaos-test.sh           # Chaos engineering tests
│   ├── monitor.sh              # Real-time monitoring
│   └── network-test.sh         # Network testing
└── README.md                   # This file
```

## 🐛 Troubleshooting

### Common Issues

1. **Services not starting**
   ```bash
   # Check container logs
   docker-compose logs [service-name]
   
   # Restart specific service
   docker-compose restart [service-name]
   ```

2. **Network latency not working**
   ```bash
   # Check tc configuration
   docker exec network-chaos tc qdisc show dev eth0
   
   # Reset network configuration
   docker exec network-chaos tc qdisc del dev eth0 root
   ```

3. **Health checks failing**
   ```bash
   # Check service endpoints
   curl -v http://localhost:8080/health
   curl -v http://localhost:8081/health
   
   # Check container status
   docker-compose ps
   ```

4. **Monitoring not working**
   ```bash
   # Check Prometheus targets
   curl http://localhost:9090/api/v1/targets
   
   # Check Grafana datasource
   curl http://localhost:3000/api/datasources
   ```

### Performance Tuning

1. **Increase container resources**
   ```yaml
   # In docker-compose.yml
   services:
     im-service-a:
       deploy:
         resources:
           limits:
             memory: 512M
             cpus: '0.5'
   ```

2. **Adjust sync intervals**
   ```bash
   # Environment variables in docker-compose.yml
   - SYNC_INTERVAL=1s
   - HEALTH_CHECK_INTERVAL=5s
   ```

## 🎯 Learning Objectives

This MVP demonstrates:

1. **HLC (Hybrid Logical Clock)** implementation for distributed ordering
2. **LWW (Last Write Wins)** conflict resolution with deterministic tiebreaking
3. **Network simulation** using Linux traffic control (tc)
4. **Split-brain prevention** using external arbiter
5. **Cross-region synchronization** patterns
6. **Chaos engineering** practices
7. **Observability** in distributed systems

## 🚧 Limitations

This is a simplified MVP for learning purposes:

- Uses SQLite instead of distributed databases
- Simulates network latency instead of real geographic distribution
- Mock arbiter instead of production-grade consensus
- Single-node services instead of clusters
- No persistent storage across container restarts

## 🔄 Next Steps

To evolve this MVP toward production:

1. Replace SQLite with MySQL/PostgreSQL replication
2. Implement real Kafka cross-cluster replication
3. Deploy to actual geographic regions
4. Add proper service mesh (Istio/Linkerd)
5. Implement comprehensive monitoring and alerting
6. Add automated failover and recovery procedures

## 📚 References

