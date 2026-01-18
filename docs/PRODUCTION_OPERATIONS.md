# Production Operations Guide

This guide covers day-to-day operations, monitoring, and maintenance of the Monorepo Hello/TODO Services in production.

## Table of Contents

- [Quick Reference](#quick-reference)
- [Health Checks](#health-checks)
- [Monitoring](#monitoring)
- [Logging](#logging)
- [Scaling](#scaling)
- [Updates and Rollbacks](#updates-and-rollbacks)
- [Troubleshooting](#troubleshooting)
- [Incident Response](#incident-response)
- [Maintenance](#maintenance)

## Quick Reference

### Essential Commands

```bash
# Check overall health
./scripts/verify-production.sh

# View pod status
kubectl get pods -n production

# View logs
kubectl logs -n production -l app=hello-service --tail=100
kubectl logs -n production -l app=todo-service --tail=100

# Check resource usage
kubectl top pods -n production

# Restart a service
kubectl rollout restart deployment/hello-service -n production

# Scale a service
kubectl scale deployment/hello-service -n production --replicas=5
```

### Service Endpoints

**Internal (within cluster)**:
- Hello Service: `hello-service.production.svc.cluster.local:9090`
- TODO Service: `todo-service.production.svc.cluster.local:9091`

**External (via Ingress)**:
- API Gateway: `https://api.example.com`
- Hello Service: `https://api.example.com/api/hello`
- TODO Service: `https://api.example.com/api/todo`

## Health Checks

### Automated Verification

Run the production verification script:

```bash
./scripts/verify-production.sh
```

This checks:
- Namespace existence
- Deployment status
- Pod health
- Service endpoints
- Ingress configuration
- Resource usage
- Log errors
- Service connectivity

### Manual Health Checks

#### Check Deployment Status

```bash
# View all deployments
kubectl get deployments -n production

# Check specific deployment
kubectl describe deployment hello-service -n production

# Check rollout status
kubectl rollout status deployment/hello-service -n production
```

#### Check Pod Health

```bash
# List all pods
kubectl get pods -n production

# Check pod details
kubectl describe pod <pod-name> -n production

# Check pod events
kubectl get events -n production --field-selector involvedObject.name=<pod-name>
```

#### Check Service Health

```bash
# List services
kubectl get services -n production

# Check service endpoints
kubectl get endpoints -n production

# Test service connectivity
kubectl run test-pod --rm -it --image=busybox -n production -- sh
# Inside pod: wget -O- http://hello-service:9090
```

### Health Check Endpoints

Both services have health check probes configured:

**Hello Service**:
- Liveness: Process check
- Readiness: gRPC health check (if implemented)

**TODO Service**:
- Liveness: Process check
- Readiness: gRPC health check (if implemented)

## Monitoring

### Resource Monitoring

#### CPU and Memory Usage

```bash
# View resource usage for all pods
kubectl top pods -n production

# View resource usage for specific service
kubectl top pods -n production -l app=hello-service

# View node resource usage
kubectl top nodes
```

#### Resource Limits

Check if pods are hitting resource limits:

```bash
# Check for OOMKilled pods
kubectl get pods -n production -o json | jq '.items[] | select(.status.containerStatuses[].lastState.terminated.reason == "OOMKilled") | .metadata.name'

# Check resource requests vs limits
kubectl describe pod <pod-name> -n production | grep -A 5 "Limits\|Requests"
```

### Application Metrics

#### Request Rates

Monitor request rates through logs:

```bash
# Count requests per minute (Hello Service)
kubectl logs -n production -l app=hello-service --since=1m | grep "SayHello" | wc -l

# Count requests per minute (TODO Service)
kubectl logs -n production -l app=todo-service --since=1m | grep "CreateTodo\|ListTodos" | wc -l
```

#### Error Rates

Monitor error rates:

```bash
# Count errors in last hour
kubectl logs -n production -l app=hello-service --since=1h | grep -i "error\|exception" | wc -l
kubectl logs -n production -l app=todo-service --since=1h | grep -i "error\|exception" | wc -l
```

### Prometheus Integration (Optional)

If Prometheus is installed, add these annotations to deployments:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
```

Example queries:
```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

## Logging

### View Logs

#### Real-time Logs

```bash
# Follow logs for Hello Service
kubectl logs -n production -l app=hello-service -f

# Follow logs for TODO Service
kubectl logs -n production -l app=todo-service -f

# Follow logs from all containers
kubectl logs -n production -l app=hello-service -f --all-containers=true
```

#### Historical Logs

```bash
# Last 100 lines
kubectl logs -n production -l app=hello-service --tail=100

# Last hour
kubectl logs -n production -l app=hello-service --since=1h

# Specific time range (requires log aggregation)
kubectl logs -n production -l app=hello-service --since-time=2024-01-17T10:00:00Z
```

#### Previous Container Logs

If a container crashed:

```bash
kubectl logs -n production <pod-name> --previous
```

### Log Aggregation

For production, set up centralized logging:

**ELK Stack (Elasticsearch, Logstash, Kibana)**:
```bash
# Install ELK using Helm
helm repo add elastic https://helm.elastic.co
helm install elasticsearch elastic/elasticsearch -n logging
helm install kibana elastic/kibana -n logging
helm install filebeat elastic/filebeat -n logging
```

**Loki + Grafana**:
```bash
# Install Loki using Helm
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n logging
```

### Log Levels

Configure log levels via environment variables:

```yaml
env:
- name: LOG_LEVEL
  value: "info"  # debug, info, warn, error
```

## Scaling

### Manual Scaling

#### Scale Up

```bash
# Scale Hello Service to 5 replicas
kubectl scale deployment hello-service -n production --replicas=5

# Scale TODO Service to 3 replicas
kubectl scale deployment todo-service -n production --replicas=3

# Verify scaling
kubectl get deployments -n production
```

#### Scale Down

```bash
# Scale down during low traffic
kubectl scale deployment hello-service -n production --replicas=2
kubectl scale deployment todo-service -n production --replicas=2
```

### Horizontal Pod Autoscaler (HPA)

#### Create HPA

```yaml
# hpa-hello-service.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hello-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

Apply:
```bash
kubectl apply -f hpa-hello-service.yaml
```

#### Monitor HPA

```bash
# View HPA status
kubectl get hpa -n production

# Describe HPA
kubectl describe hpa hello-service-hpa -n production

# Watch HPA in action
watch kubectl get hpa -n production
```

### Vertical Pod Autoscaler (VPA)

For automatic resource limit adjustments:

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: hello-service-vpa
  namespace: production
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-service
  updatePolicy:
    updateMode: "Auto"
```

## Updates and Rollbacks

### Rolling Updates

#### Update Image

```bash
# Update Hello Service image
kubectl set image deployment/hello-service \
  hello-service=registry.example.com/hello-service:v1.1.0 \
  -n production

# Update TODO Service image
kubectl set image deployment/todo-service \
  todo-service=registry.example.com/todo-service:v1.1.0 \
  -n production
```

#### Monitor Rollout

```bash
# Watch rollout progress
kubectl rollout status deployment/hello-service -n production

# View rollout history
kubectl rollout history deployment/hello-service -n production
```

### Rollback

#### Rollback to Previous Version

```bash
# Rollback Hello Service
kubectl rollout undo deployment/hello-service -n production

# Rollback TODO Service
kubectl rollout undo deployment/todo-service -n production
```

#### Rollback to Specific Revision

```bash
# View revision history
kubectl rollout history deployment/hello-service -n production

# Rollback to specific revision
kubectl rollout undo deployment/hello-service -n production --to-revision=3
```

### Zero-Downtime Deployments

Ensure zero-downtime with proper configuration:

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  minReadySeconds: 10
```

## Troubleshooting

### Common Issues

#### Pods Not Starting

**Check pod status:**
```bash
kubectl get pods -n production
kubectl describe pod <pod-name> -n production
```

**Common causes:**
1. ImagePullBackOff: Image not found
2. CrashLoopBackOff: Application crashes on startup
3. Pending: Insufficient resources

**Solutions:**
```bash
# Check events
kubectl get events -n production --sort-by='.lastTimestamp'

# Check logs
kubectl logs <pod-name> -n production

# Check resource availability
kubectl describe nodes
```

#### High Memory Usage

**Check memory usage:**
```bash
kubectl top pods -n production
```

**Increase memory limits:**
```bash
kubectl set resources deployment hello-service \
  --limits=memory=1Gi \
  --requests=memory=512Mi \
  -n production
```

#### Service Unavailable

**Check service endpoints:**
```bash
kubectl get endpoints -n production
kubectl describe service hello-service -n production
```

**Test connectivity:**
```bash
kubectl run test-pod --rm -it --image=nicolaka/netshoot -n production -- bash
# Inside pod: curl -v telnet://hello-service:9090
```

### Debug Commands

```bash
# Get shell in pod
kubectl exec -it <pod-name> -n production -- sh

# Run command in pod
kubectl exec <pod-name> -n production -- env

# Copy files from pod
kubectl cp production/<pod-name>:/path/to/file ./local-file

# Port forward for debugging
kubectl port-forward -n production <pod-name> 9090:9090
```

## Incident Response

### Incident Severity Levels

**P0 - Critical**: Complete service outage
**P1 - High**: Partial service degradation
**P2 - Medium**: Minor issues, no user impact
**P3 - Low**: Cosmetic issues

### Response Procedures

#### P0 - Critical Incident

1. **Acknowledge**: Confirm incident in monitoring system
2. **Assess**: Check service status
   ```bash
   ./scripts/verify-production.sh
   kubectl get pods -n production
   ```
3. **Mitigate**: Quick fixes
   ```bash
   # Restart failing service
   kubectl rollout restart deployment/hello-service -n production
   
   # Scale up if needed
   kubectl scale deployment/hello-service -n production --replicas=5
   
   # Rollback if recent deployment
   kubectl rollout undo deployment/hello-service -n production
   ```
4. **Communicate**: Update status page
5. **Resolve**: Fix root cause
6. **Post-mortem**: Document incident

#### P1 - High Priority

1. Check logs for errors
2. Scale up if performance issue
3. Apply fix during next maintenance window

### Emergency Contacts

- On-call engineer: [Contact info]
- Platform team: [Contact info]
- Cloud provider support: [Contact info]

## Maintenance

### Regular Maintenance Tasks

#### Daily

- [ ] Check pod health
- [ ] Review error logs
- [ ] Monitor resource usage
- [ ] Check for alerts

```bash
# Daily health check
./scripts/verify-production.sh
```

#### Weekly

- [ ] Review metrics and trends
- [ ] Check for security updates
- [ ] Review and clean up old resources
- [ ] Test backup and restore

```bash
# Clean up completed pods
kubectl delete pods --field-selector=status.phase=Succeeded -n production

# Clean up evicted pods
kubectl delete pods --field-selector=status.phase=Failed -n production
```

#### Monthly

- [ ] Update dependencies
- [ ] Review and optimize resource limits
- [ ] Conduct disaster recovery drill
- [ ] Review and update documentation

### Backup and Restore

#### Backup Configuration

```bash
# Backup all resources
kubectl get all -n production -o yaml > backup-$(date +%Y%m%d).yaml

# Backup specific resources
kubectl get deployment,service,ingress -n production -o yaml > backup-config.yaml
```

#### Restore from Backup

```bash
# Restore from backup
kubectl apply -f backup-20240117.yaml
```

### Security Updates

#### Update Base Images

1. Update Dockerfiles with new base image versions
2. Rebuild images
3. Test in staging
4. Deploy to production

```bash
# Build new images
make docker-build

# Tag and push
docker tag hello-service:latest registry.example.com/hello-service:v1.1.0
docker push registry.example.com/hello-service:v1.1.0

# Deploy
kubectl set image deployment/hello-service \
  hello-service=registry.example.com/hello-service:v1.1.0 \
  -n production
```

#### Scan for Vulnerabilities

```bash
# Scan images
trivy image registry.example.com/hello-service:v1.0.0
trivy image registry.example.com/todo-service:v1.0.0
```

## Best Practices

1. **Always test in staging first**
2. **Use rolling updates for zero-downtime**
3. **Monitor during and after deployments**
4. **Keep rollback plan ready**
5. **Document all changes**
6. **Regular backups**
7. **Security updates promptly**
8. **Capacity planning**

## Additional Resources

- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Production Checklist](https://kubernetes.io/docs/setup/best-practices/cluster-large/)
- [Troubleshooting Guide](https://kubernetes.io/docs/tasks/debug/)

