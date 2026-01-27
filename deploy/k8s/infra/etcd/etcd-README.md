# etcd Cluster Deployment for IM Chat System (LEGACY - ARCHIVED)

**⚠️ DEPRECATED**: This directory contains legacy documentation for the custom etcd StatefulSet deployment. 

**Current Deployment Method**: We now use the **Bitnami etcd Helm chart** for production deployments. See the parent directory's README for current deployment instructions.

---

## Legacy Documentation (For Reference Only)

This directory previously contained Kubernetes manifests for deploying a 3-node etcd cluster that serves as the Registry service for the IM Chat System.

## Overview

The etcd cluster provides:
- **User-to-Gateway Mapping**: Maintains `user_id → (gateway_node, device_id)` mappings
- **Multi-Device Support**: Multiple entries per user_id for different devices
- **TTL-Based Cleanup**: Automatic removal of stale entries (90s TTL)
- **Watch API**: Real-time cache invalidation for Gateway nodes
- **High Availability**: 3-node cluster with Raft consensus

**Validates Requirements:**
- 7.6: etcd cluster with 3 or 5 nodes for high availability
- 10.3: Quorum-based consensus for fault tolerance

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  etcd Cluster                       │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ etcd-0   │  │ etcd-1   │  │ etcd-2   │        │
│  │ (Leader) │  │(Follower)│  │(Follower)│        │
│  │ 10Gi PV  │  │ 10Gi PV  │  │ 10Gi PV  │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
│       │             │             │                │
│       └─────────────┴─────────────┘                │
│              Raft Consensus                        │
└─────────────────────────────────────────────────────┘
         │                    │
         │ Client Port 2379   │ Peer Port 2380
         │                    │
    ┌────▼────────────────────▼────┐
    │   Gateway Nodes / IM Service  │
    └───────────────────────────────┘
```

## Components

### 1. Headless Service (`etcd`)
- **Purpose**: Stable network identity for StatefulSet pods
- **Type**: ClusterIP None (headless)
- **Ports**:
  - 2379: Client communication
  - 2380: Peer communication (Raft)
- **DNS**: `etcd-{0,1,2}.etcd.default.svc.cluster.local`

### 2. Client Service (`etcd-client`)
- **Purpose**: Load-balanced endpoint for client applications
- **Type**: ClusterIP
- **Port**: 2379
- **DNS**: `etcd-client.default.svc.cluster.local`

### 3. StatefulSet (`etcd`)
- **Replicas**: 3 (odd number for quorum)
- **Image**: `quay.io/coreos/etcd:v3.5.11`
- **Storage**: 10Gi persistent volume per pod
- **Resources**:
  - Requests: 250m CPU, 512Mi memory
  - Limits: 500m CPU, 1Gi memory

## Configuration

### Environment Variables

| Variable | Value | Description |
|----------|-------|-------------|
| `ETCD_INITIAL_CLUSTER_TOKEN` | `etcd-cluster-im-chat` | Unique cluster identifier |
| `ETCD_INITIAL_CLUSTER_STATE` | `new` | Initial cluster state |
| `ETCD_INITIAL_CLUSTER` | `etcd-0=http://...,etcd-1=...,etcd-2=...` | Initial cluster members |
| `ETCD_LISTEN_PEER_URLS` | `http://0.0.0.0:2380` | Peer communication endpoint |
| `ETCD_LISTEN_CLIENT_URLS` | `http://0.0.0.0:2379` | Client communication endpoint |
| `ETCD_AUTO_COMPACTION_RETENTION` | `1` | Auto-compact every 1 hour |
| `ETCD_QUOTA_BACKEND_BYTES` | `8589934592` | 8GB backend quota |

### Storage

- **Volume Claim Template**: 10Gi per pod
- **Access Mode**: ReadWriteOnce
- **Storage Class**: Default (can be customized)
- **Mount Path**: `/var/run/etcd`

### Health Checks

**Liveness Probe:**
- Endpoint: `GET /health` on port 2379
- Initial Delay: 30s
- Period: 10s
- Timeout: 5s
- Failure Threshold: 3

**Readiness Probe:**
- Endpoint: `GET /health` on port 2379
- Initial Delay: 10s
- Period: 5s
- Timeout: 3s
- Failure Threshold: 3

## Deployment

### Prerequisites

1. Kubernetes cluster (v1.24+)
2. kubectl configured
3. Sufficient storage (30Gi total for 3 nodes)
4. Namespace created (default or custom)

### Deploy etcd Cluster

```bash
# Deploy to default namespace
kubectl apply -f k8s/base/etcd-statefulset.yaml

# Or deploy to custom namespace
kubectl apply -f k8s/base/etcd-statefulset.yaml -n im-chat

# Verify deployment
kubectl get statefulset etcd
kubectl get pods -l app=etcd
kubectl get pvc -l app=etcd
kubectl get svc etcd etcd-client
```

### Wait for Cluster Ready

```bash
# Wait for all pods to be ready
kubectl wait --for=condition=ready pod -l app=etcd --timeout=300s

# Check pod status
kubectl get pods -l app=etcd -o wide
```

Expected output:
```
NAME     READY   STATUS    RESTARTS   AGE
etcd-0   1/1     Running   0          2m
etcd-1   1/1     Running   0          2m
etcd-2   1/1     Running   0          2m
```

## Testing

### Run Automated Tests

```bash
# Run comprehensive test suite
./scripts/test-etcd-cluster.sh

# Or with custom namespace
NAMESPACE=im-chat ./scripts/test-etcd-cluster.sh
```

The test script validates:
1. ✓ Cluster health
2. ✓ Member list
3. ✓ Leader election
4. ✓ Write operations
5. ✓ Read operations
6. ✓ TTL/lease functionality
7. ✓ Watch mechanism
8. ✓ Cluster consistency
9. ✓ Cleanup

### Manual Testing

#### 1. Check Cluster Health

```bash
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-0.etcd:2379,http://etcd-1.etcd:2379,http://etcd-2.etcd:2379 \
  endpoint health
```

Expected output:
```
http://etcd-0.etcd:2379 is healthy: successfully committed proposal: took = 2.345ms
http://etcd-1.etcd:2379 is healthy: successfully committed proposal: took = 2.567ms
http://etcd-2.etcd:2379 is healthy: successfully committed proposal: took = 2.123ms
```

#### 2. Check Leader Election

```bash
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-0.etcd:2379,http://etcd-1.etcd:2379,http://etcd-2.etcd:2379 \
  endpoint status --write-out=table
```

Expected output:
```
+---------------------------+------------------+---------+---------+-----------+
|         ENDPOINT          |        ID        | VERSION | DB SIZE | IS LEADER |
+---------------------------+------------------+---------+---------+-----------+
| http://etcd-0.etcd:2379   | 8e9e05c52164694d | 3.5.11  | 20 kB   | true      |
| http://etcd-1.etcd:2379   | 91bc3c398fb3c146 | 3.5.11  | 20 kB   | false     |
| http://etcd-2.etcd:2379   | fd422379fda50e48 | 3.5.11  | 20 kB   | false     |
+---------------------------+------------------+---------+---------+-----------+
```

#### 3. Test Write/Read Operations

```bash
# Write a key
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  put /registry/users/user123/device-abc "gateway-1.cluster.local:8080"

# Read the key
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  get /registry/users/user123/device-abc
```

#### 4. Test TTL (Lease) Functionality

```bash
# Create a lease (90 seconds)
LEASE_ID=$(kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  lease grant 90 | grep -oP 'lease \K[0-9a-f]+')

# Put key with lease
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  put /registry/users/user456/device-xyz "gateway-2.cluster.local:8080" \
  --lease=${LEASE_ID}

# Check TTL
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  lease timetolive ${LEASE_ID}
```

#### 5. Test Watch Mechanism

```bash
# Terminal 1: Start watching
kubectl exec -it etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  watch /registry/users/ --prefix

# Terminal 2: Trigger events
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  put /registry/users/user789/device-123 "gateway-3.cluster.local:8080"
```

## Operations

### Scaling

**Note**: etcd requires odd number of nodes (3, 5, 7) for quorum.

To scale to 5 nodes:

```bash
# Update StatefulSet replicas
kubectl scale statefulset etcd --replicas=5

# Update ETCD_INITIAL_CLUSTER environment variable
# Add: etcd-3=http://etcd-3.etcd:2380,etcd-4=http://etcd-4.etcd:2380
```

### Backup

```bash
# Create snapshot
kubectl exec etcd-0 -- etcdctl \
  --endpoints=http://etcd-client:2379 \
  snapshot save /var/run/etcd/backup-$(date +%Y%m%d-%H%M%S).db

# Copy snapshot to local
kubectl cp etcd-0:/var/run/etcd/backup-*.db ./etcd-backup.db
```

### Restore

```bash
# Copy snapshot to pod
kubectl cp ./etcd-backup.db etcd-0:/var/run/etcd/restore.db

# Stop etcd (scale down)
kubectl scale statefulset etcd --replicas=0

# Restore from snapshot (requires manual intervention)
# See: https://etcd.io/docs/v3.5/op-guide/recovery/

# Scale back up
kubectl scale statefulset etcd --replicas=3
```

### Monitoring

#### View Logs

```bash
# View logs for all pods
kubectl logs -l app=etcd --tail=100 -f

# View logs for specific pod
kubectl logs etcd-0 -f
```

#### Check Metrics

```bash
# Get etcd metrics
kubectl exec etcd-0 -- curl -s http://localhost:2379/metrics
```

Key metrics to monitor:
- `etcd_server_has_leader`: Should be 1
- `etcd_server_leader_changes_seen_total`: Should be low
- `etcd_disk_backend_commit_duration_seconds`: Should be < 100ms
- `etcd_network_peer_round_trip_time_seconds`: Should be < 50ms

### Troubleshooting

#### Pods Not Starting

```bash
# Check pod events
kubectl describe pod etcd-0

# Check logs
kubectl logs etcd-0

# Common issues:
# - Insufficient storage
# - Port conflicts
# - Network policies blocking peer communication
```

#### Cluster Not Forming

```bash
# Check if all pods can reach each other
kubectl exec etcd-0 -- ping etcd-1.etcd
kubectl exec etcd-0 -- ping etcd-2.etcd

# Check peer URLs
kubectl exec etcd-0 -- etcdctl member list

# Verify ETCD_INITIAL_CLUSTER matches actual pods
kubectl get pods -l app=etcd -o yaml | grep ETCD_INITIAL_CLUSTER
```

#### Leader Election Issues

```bash
# Check leader status
kubectl exec etcd-0 -- etcdctl endpoint status --write-out=table

# Check for network partitions
kubectl exec etcd-0 -- etcdctl alarm list

# If quorum lost, may need to force new cluster
# WARNING: This is destructive, only use as last resort
```

#### Performance Issues

```bash
# Check disk latency
kubectl exec etcd-0 -- etcdctl check perf

# Check database size
kubectl exec etcd-0 -- etcdctl endpoint status --write-out=table

# Compact and defragment if needed
kubectl exec etcd-0 -- etcdctl compact $(kubectl exec etcd-0 -- etcdctl endpoint status --write-out=json | jq -r '.[0].Status.header.revision')
kubectl exec etcd-0 -- etcdctl defrag
```

## Integration with IM Services

### Connection Configuration

Gateway and IM services should connect using:

```yaml
registry:
  endpoints:
    - etcd-0.etcd.default.svc.cluster.local:2379
    - etcd-1.etcd.default.svc.cluster.local:2379
    - etcd-2.etcd.default.svc.cluster.local:2379
  # Or use load-balanced endpoint:
  # - etcd-client.default.svc.cluster.local:2379
  dial_timeout: 5s
  request_timeout: 3s
```

### Example Go Client

```go
import (
    clientv3 "go.etcd.io/etcd/client/v3"
)

func NewRegistryClient() (*clientv3.Client, error) {
    return clientv3.New(clientv3.Config{
        Endpoints:   []string{
            "etcd-0.etcd.default.svc.cluster.local:2379",
            "etcd-1.etcd.default.svc.cluster.local:2379",
            "etcd-2.etcd.default.svc.cluster.local:2379",
        },
        DialTimeout: 5 * time.Second,
    })
}
```

## Security Considerations

**Current Configuration**: HTTP (no TLS)
- Suitable for development and internal cluster communication
- All traffic stays within Kubernetes cluster network

**Production Recommendations**:
1. Enable TLS for client and peer communication
2. Use Kubernetes secrets for certificates
3. Enable RBAC authentication
4. Restrict network policies to only allow IM services

## Cleanup

```bash
# Delete etcd cluster
kubectl delete -f k8s/base/etcd-statefulset.yaml

# Delete persistent volumes (WARNING: Data loss!)
kubectl delete pvc -l app=etcd
```

## References

- [etcd Documentation](https://etcd.io/docs/v3.5/)
- [etcd Operator Guide](https://etcd.io/docs/v3.5/op-guide/)
- [Kubernetes StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [IM Chat System Design](../../.kiro/specs/im-chat-system/design.md)

## Support

For issues or questions:
1. Check logs: `kubectl logs -l app=etcd`
2. Run test script: `./scripts/test-etcd-cluster.sh`
3. Review etcd documentation
4. Check IM Chat System design document
