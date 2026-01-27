# IM Chat System - Operational Runbooks

**Version**: 1.0  
**Last Updated**: 2026-01-25  
**Maintainer**: Platform Team

## Table of Contents

1. [Overview](#overview)
2. [Runbook 1: Handle Gateway Node Failure](#runbook-1-handle-gateway-node-failure)
3. [Runbook 2: Handle Database Outage](#runbook-2-handle-database-outage)
4. [Runbook 3: Handle Kafka Outage](#runbook-3-handle-kafka-outage)
5. [Runbook 4: Scale Cluster Up](#runbook-4-scale-cluster-up)
6. [Runbook 5: Scale Cluster Down](#runbook-5-scale-cluster-down)
7. [Runbook 6: Investigate Message Delivery Issues](#runbook-6-investigate-message-delivery-issues)
8. [Emergency Contacts](#emergency-contacts)
9. [Post-Incident Procedures](#post-incident-procedures)

---

## Overview

This document contains operational runbooks for the IM Chat System. Each runbook provides step-by-step procedures for handling common operational scenarios, including failures, scaling operations, and troubleshooting.

### Severity Levels

- **P0 (Critical)**: Complete service outage, immediate action required
- **P1 (High)**: Partial service degradation, action required within 1 hour
- **P2 (Medium)**: Minor issues, action required within 4 hours
- **P3 (Low)**: Non-urgent issues, action required within 24 hours

### Prerequisites

Before executing any runbook, ensure you have:
- Access to Kubernetes cluster (`kubectl` configured)
- Access to monitoring dashboards (Grafana)
- Access to logging system (Loki/ELK)
- Access to alerting system (Alertmanager)
- VPN connection to production environment (if required)
- Incident tracking system access (Jira/PagerDuty)

---

## Runbook 1: Handle Gateway Node Failure

**Severity**: P1 (High)  
**Estimated Time**: 15-30 minutes  
**Impact**: Partial service degradation, affected users will reconnect

### Symptoms

- Alert: `IMGatewayNodeDown` triggered
- Grafana dashboard shows Gateway node(s) offline
- Increased connection errors in logs
- Users reporting disconnections

### Diagnosis Steps

1. **Verify the issue**:
   ```bash
   # Check Gateway pod status
   kubectl get pods -n im-system -l app=im-gateway-service
   
   # Check pod events
   kubectl describe pod <gateway-pod-name> -n im-system
   
   # Check pod logs
   kubectl logs <gateway-pod-name> -n im-system --tail=100
   ```

2. **Check node health**:
   ```bash
   # Check node status
   kubectl get nodes
   
   # Check node resources
   kubectl top nodes
   ```

3. **Check metrics**:
   - Open Grafana → IM Gateway Connections dashboard
   - Verify active connections dropped
   - Check error rates increased

### Resolution Steps

#### Scenario A: Pod Crashed (OOMKilled, CrashLoopBackOff)

1. **Check resource limits**:
   ```bash
   kubectl describe pod <gateway-pod-name> -n im-system | grep -A 5 "Limits"
   ```

2. **Increase resource limits** (if OOMKilled):
   ```bash
   # Edit deployment
   kubectl edit deployment im-gateway-service -n im-system
   
   # Update memory limit (e.g., from 512Mi to 1Gi)
   resources:
     limits:
       memory: 1Gi
       cpu: 1000m
     requests:
       memory: 512Mi
       cpu: 500m
   ```

3. **Restart the pod**:
   ```bash
   kubectl rollout restart deployment im-gateway-service -n im-system
   ```

#### Scenario B: Node Failure

1. **Cordon the failed node**:
   ```bash
   kubectl cordon <node-name>
   ```

2. **Drain the node** (evict pods):
   ```bash
   kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
   ```

3. **Verify pods rescheduled**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service -o wide
   ```

4. **Investigate node issue**:
   - Check node logs: `journalctl -u kubelet -n 100`
   - Check system resources: `top`, `df -h`, `free -m`
   - Contact infrastructure team if hardware issue

#### Scenario C: Network Partition

1. **Check network connectivity**:
   ```bash
   # From Gateway pod to etcd
   kubectl exec -it <gateway-pod-name> -n im-system -- nc -zv etcd-0.etcd 2379
   
   # From Gateway pod to Redis
   kubectl exec -it <gateway-pod-name> -n im-system -- nc -zv redis-master 6379
   ```

2. **Check network policies**:
   ```bash
   kubectl get networkpolicies -n im-system
   ```

3. **Restart networking** (if needed):
   ```bash
   # Restart pod to reset network
   kubectl delete pod <gateway-pod-name> -n im-system
   ```

### Verification

1. **Check pod status**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service
   # All pods should be Running
   ```

2. **Check health endpoint**:
   ```bash
   kubectl exec -it <gateway-pod-name> -n im-system -- curl http://localhost:8080/health
   # Should return: {"status":"ok"}
   ```

3. **Monitor metrics**:
   - Active connections recovering
   - Error rate returning to normal (<1%)
   - Message delivery latency normal (P99 < 200ms)

4. **Test client connection**:
   ```bash
   # Use test client to connect
   wscat -c ws://im-gateway.example.com/ws
   ```

### Rollback Plan

If the issue persists after resolution:

1. **Rollback deployment** (if recent change):
   ```bash
   kubectl rollout undo deployment im-gateway-service -n im-system
   ```

2. **Scale up replicas** (temporary mitigation):
   ```bash
   kubectl scale deployment im-gateway-service -n im-system --replicas=5
   ```

### Post-Incident Actions

1. Update incident ticket with root cause
2. Document any configuration changes
3. Review and update monitoring alerts if needed
4. Schedule post-mortem meeting if P0/P1 incident

---

## Runbook 2: Handle Database Outage

**Severity**: P0 (Critical)  
**Estimated Time**: 30-60 minutes  
**Impact**: Offline message storage unavailable, read receipts unavailable

### Symptoms

- Alert: `MySQLDown` or `MySQLHighConnectionErrors` triggered
- Services logging database connection errors
- Offline messages not being stored
- Read receipts not being recorded

### Diagnosis Steps

1. **Check MySQL pod status**:
   ```bash
   kubectl get pods -n im-system -l app=mysql
   kubectl describe pod <mysql-pod-name> -n im-system
   ```

2. **Check MySQL logs**:
   ```bash
   kubectl logs <mysql-pod-name> -n im-system --tail=200
   ```

3. **Check database connectivity**:
   ```bash
   # From IM Service pod
   kubectl exec -it <im-service-pod> -n im-system -- \
     mysql -h mysql -u im_service -p<password> -e "SELECT 1"
   ```

4. **Check disk space**:
   ```bash
   kubectl exec -it <mysql-pod-name> -n im-system -- df -h
   ```

5. **Check connection pool**:
   ```bash
   # Check active connections
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<root-password> -e "SHOW PROCESSLIST"
   ```

### Resolution Steps

#### Scenario A: MySQL Pod Crashed

1. **Check crash reason**:
   ```bash
   kubectl logs <mysql-pod-name> -n im-system --previous
   ```

2. **Restart MySQL pod**:
   ```bash
   kubectl delete pod <mysql-pod-name> -n im-system
   # StatefulSet will recreate it
   ```

3. **Wait for pod to be ready**:
   ```bash
   kubectl wait --for=condition=ready pod/<mysql-pod-name> -n im-system --timeout=300s
   ```

#### Scenario B: Disk Full

1. **Check disk usage**:
   ```bash
   kubectl exec -it <mysql-pod-name> -n im-system -- du -sh /var/lib/mysql/*
   ```

2. **Clean up old data** (if safe):
   ```bash
   # Run TTL cleanup manually
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<password> im_chat -e \
     "DELETE FROM offline_messages WHERE created_at < DATE_SUB(NOW(), INTERVAL 7 DAY) LIMIT 10000"
   ```

3. **Expand PVC** (if needed):
   ```bash
   # Edit PVC
   kubectl edit pvc mysql-data-mysql-0 -n im-system
   # Increase storage size (e.g., from 50Gi to 100Gi)
   ```

#### Scenario C: Too Many Connections

1. **Check connection count**:
   ```bash
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<password> -e "SHOW STATUS LIKE 'Threads_connected'"
   ```

2. **Kill idle connections**:
   ```bash
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<password> -e \
     "SELECT CONCAT('KILL ', id, ';') FROM information_schema.processlist WHERE command='Sleep' AND time > 300"
   ```

3. **Increase max_connections** (if needed):
   ```bash
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<password> -e "SET GLOBAL max_connections = 500"
   ```

4. **Update ConfigMap** (permanent fix):
   ```bash
   kubectl edit configmap mysql-config -n im-system
   # Add: max_connections = 500
   ```

#### Scenario D: Replication Lag (if using replica)

1. **Check replication status**:
   ```bash
   kubectl exec -it <mysql-replica-pod> -n im-system -- \
     mysql -u root -p<password> -e "SHOW SLAVE STATUS\G"
   ```

2. **Check lag**:
   ```bash
   # Look for: Seconds_Behind_Master
   ```

3. **Skip problematic transaction** (if stuck):
   ```bash
   kubectl exec -it <mysql-replica-pod> -n im-system -- \
     mysql -u root -p<password> -e "STOP SLAVE; SET GLOBAL SQL_SLAVE_SKIP_COUNTER = 1; START SLAVE"
   ```

### Verification

1. **Check MySQL is running**:
   ```bash
   kubectl get pods -n im-system -l app=mysql
   # Should be Running and Ready
   ```

2. **Test database connection**:
   ```bash
   kubectl exec -it <im-service-pod> -n im-system -- \
     mysql -h mysql -u im_service -p<password> -e "SELECT COUNT(*) FROM offline_messages"
   ```

3. **Check service logs**:
   ```bash
   kubectl logs <im-service-pod> -n im-system --tail=50 | grep -i "database"
   # Should not see connection errors
   ```

4. **Monitor metrics**:
   - Database connection errors: 0
   - Offline message write rate: normal
   - Query latency: < 100ms

### Rollback Plan

If database cannot be recovered:

1. **Restore from backup**:
   ```bash
   # Stop services writing to database
   kubectl scale deployment im-service -n im-system --replicas=0
   kubectl scale deployment offline-worker -n im-system --replicas=0
   
   # Restore backup
   kubectl exec -it <mysql-pod-name> -n im-system -- \
     mysql -u root -p<password> im_chat < /backup/im_chat_backup.sql
   
   # Restart services
   kubectl scale deployment im-service -n im-system --replicas=3
   kubectl scale deployment offline-worker -n im-system --replicas=2
   ```

2. **Failover to replica** (if available):
   ```bash
   # Promote replica to master
   kubectl exec -it <mysql-replica-pod> -n im-system -- \
     mysql -u root -p<password> -e "STOP SLAVE; RESET SLAVE ALL"
   
   # Update service to point to new master
   kubectl edit service mysql -n im-system
   ```

### Post-Incident Actions

1. Review database backup strategy
2. Check disk space monitoring alerts
3. Review connection pool configuration
4. Schedule database maintenance window if needed

---

## Runbook 3: Handle Kafka Outage

**Severity**: P0 (Critical)  
**Estimated Time**: 20-45 minutes  
**Impact**: Group messages not delivered, offline messages not queued

### Symptoms

- Alert: `KafkaBrokerDown` or `KafkaHighProducerErrors` triggered
- Group messages not being delivered
- Offline messages not being queued
- Services logging Kafka connection errors

### Diagnosis Steps

1. **Check Kafka broker status**:
   ```bash
   kubectl get pods -n im-system -l app=kafka
   kubectl describe pod <kafka-pod-name> -n im-system
   ```

2. **Check Kafka logs**:
   ```bash
   kubectl logs <kafka-pod-name> -n im-system --tail=200
   ```

3. **Check Kafka topics**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-topics.sh --bootstrap-server localhost:9092 --list
   ```

4. **Check consumer groups**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list
   ```

5. **Check topic lag**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
     --group gateway-group --describe
   ```

### Resolution Steps

#### Scenario A: Kafka Broker Crashed

1. **Check crash reason**:
   ```bash
   kubectl logs <kafka-pod-name> -n im-system --previous
   ```

2. **Restart Kafka broker**:
   ```bash
   kubectl delete pod <kafka-pod-name> -n im-system
   # StatefulSet will recreate it
   ```

3. **Wait for broker to rejoin cluster**:
   ```bash
   kubectl wait --for=condition=ready pod/<kafka-pod-name> -n im-system --timeout=300s
   ```

#### Scenario B: Disk Full

1. **Check disk usage**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- df -h /var/lib/kafka
   ```

2. **Clean up old segments** (if retention policy not working):
   ```bash
   # Check topic retention
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-configs.sh --bootstrap-server localhost:9092 \
     --entity-type topics --entity-name group_msg --describe
   
   # Update retention (if needed)
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-configs.sh --bootstrap-server localhost:9092 \
     --entity-type topics --entity-name group_msg \
     --alter --add-config retention.ms=3600000
   ```

3. **Expand PVC** (if needed):
   ```bash
   kubectl edit pvc kafka-data-kafka-0 -n im-system
   # Increase storage size
   ```

#### Scenario C: Under-Replicated Partitions

1. **Check partition status**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-topics.sh --bootstrap-server localhost:9092 \
     --describe --under-replicated-partitions
   ```

2. **Trigger leader election**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-leader-election.sh --bootstrap-server localhost:9092 \
     --election-type PREFERRED --all-topic-partitions
   ```

3. **Reassign partitions** (if broker permanently lost):
   ```bash
   # Generate reassignment plan
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-reassign-partitions.sh --bootstrap-server localhost:9092 \
     --topics-to-move-json-file /tmp/topics.json \
     --broker-list "0,1,2" --generate
   
   # Execute reassignment
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-reassign-partitions.sh --bootstrap-server localhost:9092 \
     --reassignment-json-file /tmp/reassignment.json --execute
   ```

#### Scenario D: Consumer Lag

1. **Check consumer lag**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
     --group gateway-group --describe
   ```

2. **Scale up consumers** (if lag increasing):
   ```bash
   # Scale Gateway service (consumes group_msg)
   kubectl scale deployment im-gateway-service -n im-system --replicas=5
   
   # Scale Offline Worker (consumes offline_msg)
   kubectl scale deployment offline-worker -n im-system --replicas=3
   ```

3. **Reset consumer offset** (if stuck):
   ```bash
   # Stop consumers first
   kubectl scale deployment im-gateway-service -n im-system --replicas=0
   
   # Reset offset to latest
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
     --group gateway-group --reset-offsets --to-latest --all-topics --execute
   
   # Restart consumers
   kubectl scale deployment im-gateway-service -n im-system --replicas=3
   ```

### Verification

1. **Check Kafka brokers**:
   ```bash
   kubectl get pods -n im-system -l app=kafka
   # All brokers should be Running and Ready
   ```

2. **Check topic health**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-topics.sh --bootstrap-server localhost:9092 --describe
   # All partitions should have leader and in-sync replicas
   ```

3. **Test message production**:
   ```bash
   kubectl exec -it <kafka-pod-name> -n im-system -- \
     kafka-console-producer.sh --bootstrap-server localhost:9092 --topic group_msg
   # Type test message and press Ctrl+C
   ```

4. **Monitor metrics**:
   - Kafka producer errors: 0
   - Under-replicated partitions: 0
   - Consumer lag: decreasing

### Rollback Plan

If Kafka cluster cannot be recovered:

1. **Deploy new Kafka cluster** (parallel):
   ```bash
   # Deploy new cluster with different name
   kubectl apply -f deploy/k8s/infra/kafka-new.yaml
   ```

2. **Update service configurations**:
   ```bash
   # Update IM Service to use new Kafka
   kubectl set env deployment/im-service -n im-system \
     KAFKA_BROKERS=kafka-new-0.kafka-new:9092,kafka-new-1.kafka-new:9092
   ```

3. **Migrate data** (if needed):
   - Use MirrorMaker 2 to replicate topics
   - Or accept data loss for transient messages

### Post-Incident Actions

1. Review Kafka monitoring and alerting
2. Check disk space and retention policies
3. Review replication factor configuration
4. Schedule Kafka cluster upgrade if needed

---

## Runbook 4: Scale Cluster Up

**Severity**: P3 (Low) - Planned operation  
**Estimated Time**: 30-45 minutes  
**Impact**: None (capacity increase)

### When to Scale Up

- Active connections approaching 80% of capacity
- Message delivery latency increasing (P99 > 150ms)
- CPU/Memory usage consistently > 70%
- Anticipating traffic spike (e.g., product launch)

### Pre-Scaling Checklist

1. **Review current metrics**:
   - Current active connections per Gateway node
   - Current CPU/Memory usage
   - Current message throughput
   - Current error rates

2. **Calculate target capacity**:
   - Target connections per node: 80,000 (80% of 100K max)
   - Target CPU usage: < 70%
   - Target memory usage: < 70%

3. **Check infrastructure capacity**:
   - Available Kubernetes nodes
   - Available IP addresses
   - Load balancer capacity

4. **Schedule maintenance window** (optional):
   - Notify users if significant scaling
   - Prepare rollback plan

### Scaling Steps

#### Step 1: Scale Gateway Service

1. **Check current replica count**:
   ```bash
   kubectl get deployment im-gateway-service -n im-system
   ```

2. **Calculate new replica count**:
   ```
   New replicas = Current replicas × (Current load / Target load)
   Example: 3 × (90% / 70%) = 4 replicas
   ```

3. **Scale deployment**:
   ```bash
   kubectl scale deployment im-gateway-service -n im-system --replicas=5
   ```

4. **Monitor rollout**:
   ```bash
   kubectl rollout status deployment im-gateway-service -n im-system
   ```

5. **Verify new pods**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service -o wide
   ```

#### Step 2: Scale IM Service (if needed)

1. **Check current load**:
   ```bash
   kubectl top pods -n im-system -l app=im-service
   ```

2. **Scale if CPU/Memory > 70%**:
   ```bash
   kubectl scale deployment im-service -n im-system --replicas=5
   ```

#### Step 3: Scale Infrastructure (if needed)

**Scale etcd** (if Registry lookups slow):
```bash
# etcd is StatefulSet, scaling requires careful planning
# Typically run 3 or 5 nodes (odd number for quorum)
kubectl scale statefulset etcd -n im-system --replicas=5
```

**Scale Redis** (if cache hit rate low):
```bash
# Add read replicas
kubectl scale deployment redis-replica -n im-system --replicas=3
```

**Scale Kafka** (if high producer lag):
```bash
# Kafka scaling requires partition reassignment
# See Kafka documentation for detailed steps
kubectl scale statefulset kafka -n im-system --replicas=5
```

#### Step 4: Update Load Balancer

1. **Check load balancer configuration**:
   ```bash
   kubectl get service im-gateway-service -n im-system -o yaml
   ```

2. **Verify new endpoints registered**:
   ```bash
   kubectl get endpoints im-gateway-service -n im-system
   ```

3. **Test load distribution**:
   ```bash
   # Connect multiple clients and verify distribution
   for i in {1..10}; do
     curl -s http://im-gateway.example.com/health | jq .hostname
   done
   ```

#### Step 5: Update Auto-Scaling (if using HPA)

1. **Check current HPA**:
   ```bash
   kubectl get hpa -n im-system
   ```

2. **Update HPA max replicas**:
   ```bash
   kubectl patch hpa im-gateway-service-hpa -n im-system \
     --patch '{"spec":{"maxReplicas":10}}'
   ```

### Verification

1. **Check all pods running**:
   ```bash
   kubectl get pods -n im-system
   # All pods should be Running and Ready
   ```

2. **Check resource usage decreased**:
   ```bash
   kubectl top pods -n im-system
   # CPU/Memory should be < 70%
   ```

3. **Check connection distribution**:
   - Open Grafana → IM Gateway Connections dashboard
   - Verify connections distributed across all nodes
   - Each node should have similar connection count

4. **Check metrics**:
   - Message delivery latency: P99 < 150ms
   - Error rate: < 1%
   - Active connections per node: < 80K

5. **Test client connections**:
   ```bash
   # Run load test with increased connections
   cd apps/im-gateway-service/load_test
   ./run-load-tests.sh quick
   ```

### Post-Scaling Actions

1. **Update capacity planning document**:
   - Document new capacity
   - Update scaling thresholds
   - Update cost estimates

2. **Update monitoring alerts**:
   - Adjust alert thresholds if needed
   - Update capacity alerts

3. **Review auto-scaling configuration**:
   - Verify HPA working correctly
   - Adjust scaling policies if needed

4. **Document scaling decision**:
   - Reason for scaling
   - Metrics before/after
   - Any issues encountered

### Rollback Plan

If scaling causes issues:

1. **Scale back to original replica count**:
   ```bash
   kubectl scale deployment im-gateway-service -n im-system --replicas=3
   ```

2. **Check for issues**:
   - Review logs for errors
   - Check metrics for anomalies

3. **Investigate root cause**:
   - Resource constraints?
   - Configuration issues?
   - Infrastructure limitations?

---

## Runbook 5: Scale Cluster Down

**Severity**: P3 (Low) - Planned operation  
**Estimated Time**: 30-45 minutes  
**Impact**: None if done correctly

### When to Scale Down

- Active connections consistently < 50% of capacity
- CPU/Memory usage consistently < 40%
- Cost optimization initiative
- Off-peak hours (if predictable traffic pattern)

### Pre-Scaling Checklist

1. **Review traffic patterns**:
   - Check last 7 days of traffic
   - Identify peak hours
   - Verify current load is not temporary dip

2. **Calculate safe capacity**:
   - Ensure remaining capacity can handle peak load
   - Leave 30% headroom for spikes
   - Consider upcoming events

3. **Check dependencies**:
   - Verify no ongoing incidents
   - Check no planned traffic spikes
   - Verify monitoring is working

4. **Schedule maintenance window** (recommended):
   - Choose low-traffic period
   - Notify team members
   - Prepare rollback plan

### Scaling Steps

#### Step 1: Graceful Connection Draining

1. **Identify pods to remove**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service -o wide
   # Choose pods to drain
   ```

2. **Mark pod for draining** (if supported):
   ```bash
   # Set pod annotation to stop accepting new connections
   kubectl annotate pod <gateway-pod-name> -n im-system \
     drain=true
   ```

3. **Wait for connections to drain** (or force close):
   ```bash
   # Monitor active connections on pod
   kubectl exec -it <gateway-pod-name> -n im-system -- \
     curl http://localhost:8080/stats | jq .active_connections
   
   # Wait until connections < 1000 or timeout (5 minutes)
   ```

#### Step 2: Scale Down Gateway Service

1. **Calculate new replica count**:
   ```
   New replicas = Current replicas × (Target load / Current load)
   Example: 5 × (70% / 40%) = 3 replicas (round up for safety)
   ```

2. **Scale deployment**:
   ```bash
   kubectl scale deployment im-gateway-service -n im-system --replicas=3
   ```

3. **Monitor rollout**:
   ```bash
   kubectl rollout status deployment im-gateway-service -n im-system
   ```

4. **Verify remaining pods healthy**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service
   ```

#### Step 3: Scale Down IM Service (if needed)

1. **Check current load**:
   ```bash
   kubectl top pods -n im-system -l app=im-service
   ```

2. **Scale if CPU/Memory < 40%**:
   ```bash
   kubectl scale deployment im-service -n im-system --replicas=3
   ```

#### Step 4: Scale Down Infrastructure (if needed)

**Scale Redis replicas** (if cache hit rate high):
```bash
kubectl scale deployment redis-replica -n im-system --replicas=1
```

**Note**: Do NOT scale down etcd or Kafka unless absolutely necessary, as they require quorum and careful planning.

#### Step 5: Update Auto-Scaling (if using HPA)

1. **Update HPA min/max replicas**:
   ```bash
   kubectl patch hpa im-gateway-service-hpa -n im-system \
     --patch '{"spec":{"minReplicas":3,"maxReplicas":8}}'
   ```

### Verification

1. **Check remaining pods healthy**:
   ```bash
   kubectl get pods -n im-system
   # All pods should be Running and Ready
   ```

2. **Check resource usage acceptable**:
   ```bash
   kubectl top pods -n im-system
   # CPU/Memory should be < 70%
   ```

3. **Check connection distribution**:
   - Open Grafana → IM Gateway Connections dashboard
   - Verify connections distributed across remaining nodes
   - No single node overloaded

4. **Check metrics**:
   - Message delivery latency: P99 < 200ms
   - Error rate: < 1%
   - Active connections per node: < 80K

5. **Monitor for 30 minutes**:
   - Watch for any degradation
   - Check error rates
   - Verify no alerts triggered

### Post-Scaling Actions

1. **Update capacity planning document**:
   - Document new capacity
   - Update cost savings
   - Update scaling thresholds

2. **Monitor closely for 24 hours**:
   - Watch for traffic spikes
   - Be ready to scale up quickly
   - Keep team on standby

3. **Document scaling decision**:
   - Reason for scaling down
   - Metrics before/after
   - Cost savings achieved

### Rollback Plan

If scaling down causes issues:

1. **Immediately scale back up**:
   ```bash
   kubectl scale deployment im-gateway-service -n im-system --replicas=5
   ```

2. **Check for issues**:
   - Review logs for errors
   - Check metrics for degradation
   - Verify all services healthy

3. **Investigate root cause**:
   - Was capacity calculation wrong?
   - Unexpected traffic spike?
   - Infrastructure issue?

---

## Runbook 6: Investigate Message Delivery Issues

**Severity**: P1 (High)  
**Estimated Time**: 30-60 minutes  
**Impact**: Users not receiving messages

### Symptoms

- Users reporting messages not delivered
- Alert: `MessageDeliveryFailureRateHigh` triggered
- Increased message delivery latency
- Offline message queue growing

### Diagnosis Steps

#### Step 1: Identify Scope

1. **Check if issue is widespread or isolated**:
   ```bash
   # Check error rate in Grafana
   # Open: IM Gateway Messages dashboard
   # Look at: Message Delivery Success Rate
   ```

2. **Check affected users**:
   ```bash
   # Search logs for specific user
   kubectl logs -n im-system -l app=im-gateway-service | grep "user_id=<user_id>"
   ```

3. **Check affected message types**:
   - Private messages only?
   - Group messages only?
   - All messages?

#### Step 2: Check Gateway Service

1. **Check Gateway pods status**:
   ```bash
   kubectl get pods -n im-system -l app=im-gateway-service
   ```

2. **Check Gateway logs**:
   ```bash
   kubectl logs -n im-system -l app=im-gateway-service --tail=200 | grep -i error
   ```

3. **Check active connections**:
   ```bash
   kubectl exec -it <gateway-pod-name> -n im-system -- \
     curl http://localhost:8080/stats | jq .active_connections
   ```

4. **Check if user is connected**:
   ```bash
   # Search Registry for user
   kubectl exec -it etcd-0 -n im-system -- \
     etcdctl get /registry/users/<user_id> --prefix
   ```

#### Step 3: Check IM Service

1. **Check IM Service pods status**:
   ```bash
   kubectl get pods -n im-system -l app=im-service
   ```

2. **Check IM Service logs**:
   ```bash
   kubectl logs -n im-system -l app=im-service --tail=200 | grep -i error
   ```

3. **Check message routing**:
   ```bash
   # Look for routing errors
   kubectl logs -n im-system -l app=im-service | grep "route.*failed"
   ```

#### Step 4: Check Infrastructure

1. **Check etcd (Registry)**:
   ```bash
   # Check etcd health
   kubectl exec -it etcd-0 -n im-system -- etcdctl endpoint health
   
   # Check etcd performance
   kubectl exec -it etcd-0 -n im-system -- etcdctl endpoint status --write-out=table
   ```

2. **Check Redis (Deduplication)**:
   ```bash
   # Check Redis connectivity
   kubectl exec -it <gateway-pod-name> -n im-system -- redis-cli -h redis-master ping
   
   # Check Redis memory
   kubectl exec -it redis-master-0 -n im-system -- redis-cli info memory
   ```

3. **Check Kafka (Group Messages)**:
   ```bash
   # Check Kafka brokers
   kubectl get pods -n im-system -l app=kafka
   
   # Check consumer lag
   kubectl exec -it kafka-0 -n im-system -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
     --group gateway-group --describe
   ```

4. **Check MySQL (Offline Messages)**:
   ```bash
   # Check MySQL connectivity
   kubectl exec -it <im-service-pod> -n im-system -- \
     mysql -h mysql -u im_service -p<password> -e "SELECT 1"
   
   # Check offline message queue size
   kubectl exec -it mysql-0 -n im-system -- \
     mysql -u root -p<password> im_chat -e \
     "SELECT COUNT(*) FROM offline_messages WHERE created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)"
   ```

#### Step 5: Check Network

1. **Check network connectivity**:
   ```bash
   # From Gateway to IM Service
   kubectl exec -it <gateway-pod-name> -n im-system -- \
     nc -zv im-service 9094
   
   # From IM Service to Gateway
   kubectl exec -it <im-service-pod> -n im-system -- \
     nc -zv im-gateway-service 9093
   ```

2. **Check DNS resolution**:
   ```bash
   kubectl exec -it <gateway-pod-name> -n im-system -- \
     nslookup im-service.im-system.svc.cluster.local
   ```

3. **Check network policies**:
   ```bash
   kubectl get networkpolicies -n im-system
   ```

### Common Issues and Solutions

#### Issue 1: User Not Connected to Gateway

**Symptoms**: Messages going to offline queue immediately

**Solution**:
1. Check if user's WebSocket connection is active
2. Check if user is registered in Registry
3. Ask user to reconnect client

#### Issue 2: Registry Lookup Failing

**Symptoms**: Logs show "user not found in registry"

**Solution**:
```bash
# Check etcd health
kubectl exec -it etcd-0 -n im-system -- etcdctl endpoint health

# Manually register user (temporary)
kubectl exec -it etcd-0 -n im-system -- \
  etcdctl put /registry/users/<user_id>/<device_id> <gateway_address>

# Restart Gateway pod to re-register
kubectl delete pod <gateway-pod-name> -n im-system
```

#### Issue 3: Message Deduplication Blocking

**Symptoms**: Logs show "duplicate message detected"

**Solution**:
```bash
# Check Redis dedup set
kubectl exec -it redis-master-0 -n im-system -- \
  redis-cli EXISTS dedup:<msg_id>

# Clear dedup entry (if false positive)
kubectl exec -it redis-master-0 -n im-system -- \
  redis-cli DEL dedup:<msg_id>

# Check Redis memory (might be evicting keys)
kubectl exec -it redis-master-0 -n im-system -- \
  redis-cli info memory
```

#### Issue 4: Kafka Consumer Lag

**Symptoms**: Group messages delayed

**Solution**:
```bash
# Check consumer lag
kubectl exec -it kafka-0 -n im-system -- \
  kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group gateway-group --describe

# Scale up Gateway service
kubectl scale deployment im-gateway-service -n im-system --replicas=5

# Reset offset if stuck (last resort)
kubectl scale deployment im-gateway-service -n im-system --replicas=0
kubectl exec -it kafka-0 -n im-system -- \
  kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group gateway-group --reset-offsets --to-latest --topic group_msg --execute
kubectl scale deployment im-gateway-service -n im-system --replicas=3
```

#### Issue 5: Database Connection Pool Exhausted

**Symptoms**: Logs show "too many connections"

**Solution**:
```bash
# Check active connections
kubectl exec -it mysql-0 -n im-system -- \
  mysql -u root -p<password> -e "SHOW PROCESSLIST"

# Kill idle connections
kubectl exec -it mysql-0 -n im-system -- \
  mysql -u root -p<password> -e \
  "KILL <connection_id>"

# Increase connection pool (temporary)
kubectl set env deployment/im-service -n im-system \
  DB_MAX_OPEN_CONNS=50

# Restart service
kubectl rollout restart deployment im-service -n im-system
```

#### Issue 6: Sequence Number Collision

**Symptoms**: Logs show "sequence number already exists"

**Solution**:
```bash
# Check Redis sequence counter
kubectl exec -it redis-master-0 -n im-system -- \
  redis-cli GET seq:private:<user1>:<user2>

# Check MySQL sequence snapshot
kubectl exec -it mysql-0 -n im-system -- \
  mysql -u root -p<password> im_chat -e \
  "SELECT * FROM sequence_snapshots WHERE conversation_id='<conversation_id>'"

# Reset sequence (if corrupted)
kubectl exec -it redis-master-0 -n im-system -- \
  redis-cli DEL seq:private:<user1>:<user2>
```

### Verification

1. **Send test message**:
   ```bash
   # Use test client to send message
   # Verify message delivered successfully
   ```

2. **Check metrics**:
   - Message delivery success rate: > 99%
   - Message delivery latency: P99 < 200ms
   - Error rate: < 1%

3. **Check logs**:
   ```bash
   kubectl logs -n im-system -l app=im-gateway-service --tail=50
   # Should not see delivery errors
   ```

4. **Monitor for 15 minutes**:
   - Watch for recurring issues
   - Verify metrics stable

### Escalation

If issue cannot be resolved:

1. **Escalate to on-call engineer**:
   - Provide incident summary
   - Share diagnostic findings
   - Share relevant logs and metrics

2. **Engage vendor support** (if infrastructure issue):
   - etcd, Kafka, MySQL issues
   - Kubernetes cluster issues

3. **Consider emergency mitigation**:
   - Route all messages to offline queue
   - Disable problematic features
   - Scale up resources

### Post-Incident Actions

1. **Document root cause**:
   - What caused the issue?
   - Why wasn't it detected earlier?
   - How can we prevent it?

2. **Update monitoring**:
   - Add missing alerts
   - Adjust alert thresholds
   - Add new metrics

3. **Update runbook**:
   - Add new troubleshooting steps
   - Document new solutions
   - Update escalation procedures

4. **Schedule post-mortem**:
   - Review incident timeline
   - Identify action items
   - Assign owners

---

## Emergency Contacts

### On-Call Rotation

- **Primary On-Call**: Check PagerDuty schedule
- **Secondary On-Call**: Check PagerDuty schedule
- **Escalation Manager**: Check PagerDuty schedule

### Team Contacts

- **Platform Team Lead**: platform-lead@example.com
- **SRE Team**: sre-team@example.com
- **DevOps Team**: devops-team@example.com

### Vendor Support

- **Kubernetes Support**: k8s-support@example.com
- **Cloud Provider Support**: cloud-support@example.com
- **Database Support**: db-support@example.com

### Communication Channels

- **Incident Slack Channel**: #incidents
- **Platform Slack Channel**: #platform-team
- **Status Page**: https://status.example.com

---

## Post-Incident Procedures

### Immediate Actions (Within 1 hour)

1. **Update incident ticket**:
   - Mark as resolved
   - Add resolution summary
   - Document root cause

2. **Notify stakeholders**:
   - Send resolution notification
   - Update status page
   - Post in Slack channels

3. **Verify resolution**:
   - Check metrics returned to normal
   - Verify no recurring issues
   - Monitor for 1 hour

### Short-Term Actions (Within 24 hours)

1. **Document incident**:
   - Create incident report
   - Include timeline
   - Include impact assessment
   - Include resolution steps

2. **Review monitoring**:
   - Check if alerts worked correctly
   - Identify missing alerts
   - Update alert thresholds

3. **Update runbooks**:
   - Add new troubleshooting steps
   - Document new solutions
   - Update escalation procedures

### Long-Term Actions (Within 1 week)

1. **Schedule post-mortem**:
   - Invite relevant stakeholders
   - Review incident timeline
   - Identify root cause
   - Create action items

2. **Implement preventive measures**:
   - Fix underlying issues
   - Improve monitoring
   - Update documentation
   - Conduct training

3. **Track action items**:
   - Assign owners
   - Set deadlines
   - Track progress
   - Verify completion

### Post-Mortem Template

```markdown
# Incident Post-Mortem: [Incident Title]

**Date**: YYYY-MM-DD  
**Duration**: X hours  
**Severity**: P0/P1/P2  
**Incident Commander**: [Name]

## Summary

[Brief description of what happened]

## Impact

- **Users Affected**: X users
- **Services Affected**: [List services]
- **Duration**: X hours
- **Revenue Impact**: $X (if applicable)

## Timeline

- **HH:MM** - [Event description]
- **HH:MM** - [Event description]
- **HH:MM** - [Event description]

## Root Cause

[Detailed explanation of root cause]

## Resolution

[How the issue was resolved]

## What Went Well

- [Thing that went well]
- [Thing that went well]

## What Went Wrong

- [Thing that went wrong]
- [Thing that went wrong]

## Action Items

1. [Action item] - Owner: [Name] - Due: [Date]
2. [Action item] - Owner: [Name] - Due: [Date]
3. [Action item] - Owner: [Name] - Due: [Date]

## Lessons Learned

[Key takeaways from the incident]
```

---

## Appendix

### Useful Commands

**Check all pods in namespace**:
```bash
kubectl get pods -n im-system -o wide
```

**Check pod logs**:
```bash
kubectl logs <pod-name> -n im-system --tail=100 -f
```

**Check pod resource usage**:
```bash
kubectl top pods -n im-system
```

**Execute command in pod**:
```bash
kubectl exec -it <pod-name> -n im-system -- <command>
```

**Port forward to pod**:
```bash
kubectl port-forward <pod-name> -n im-system 8080:8080
```

**Check service endpoints**:
```bash
kubectl get endpoints <service-name> -n im-system
```

**Check events**:
```bash
kubectl get events -n im-system --sort-by='.lastTimestamp'
```

### Monitoring Dashboards

- **Grafana**: https://grafana.example.com
  - IM Gateway Connections: `/d/im-gateway-connections`
  - IM Gateway Messages: `/d/im-gateway-messages`
  - IM Gateway Health: `/d/im-gateway-health`
  - IM Gateway SLO: `/d/im-gateway-slo`

- **Prometheus**: https://prometheus.example.com
  - Alerts: `/alerts`
  - Targets: `/targets`

- **Alertmanager**: https://alertmanager.example.com
  - Active Alerts: `/`

### Log Queries

**Loki queries** (in Grafana):

```logql
# All errors in last hour
{namespace="im-system"} |= "error" | json

# Gateway connection errors
{namespace="im-system", app="im-gateway-service"} |= "connection" |= "error"

# Message delivery failures
{namespace="im-system"} |= "delivery" |= "failed"

# Database errors
{namespace="im-system"} |= "database" |= "error"
```

### Metric Queries

**Prometheus queries**:

```promql
# Active connections per Gateway node
sum(im_gateway_active_connections) by (pod)

# Message delivery success rate
rate(im_gateway_messages_delivered_total{status="success"}[5m]) / 
rate(im_gateway_messages_delivered_total[5m])

# P99 message delivery latency
histogram_quantile(0.99, rate(im_gateway_message_delivery_duration_seconds_bucket[5m]))

# Error rate
rate(im_gateway_errors_total[5m])
```

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-25  
**Next Review**: 2026-04-25  
**Owner**: Platform Team
