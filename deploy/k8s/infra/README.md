# Infrastructure Deployment with Helm

This directory contains infrastructure components for the IM Chat System and API Gateway. We use **community Helm charts** for deploying infrastructure components instead of maintaining custom manifests.

## Why Use Community Helm Charts?

- **Battle-tested**: Production-ready configurations maintained by the community
- **Feature-rich**: Built-in support for HA, monitoring, backups, etc.
- **Less maintenance**: No need to maintain custom Kubernetes manifests
- **Easy upgrades**: Simple version management with Helm

## Prerequisites

```bash
# Add Helm repositories
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add higress https://higress.io/helm-charts
helm repo update
```

## Infrastructure Components

### 1. etcd Cluster (Bitnami Chart)

etcd provides distributed key-value store for IM Chat System registry.

**Deploy:**
```bash
helm install im-etcd bitnami/etcd \
  --namespace im-chat-system \
  --create-namespace \
  -f deploy/k8s/infra/etcd-values.yaml
```

**Configuration:**
- 3-node cluster for high availability
- 10Gi persistent volume per node
- Auto-compaction enabled
- 8GB backend quota

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=etcd
```

**Access:**
```bash
# Port forward
kubectl port-forward -n im-chat-system svc/im-etcd 2379:2379

# Test connection
etcdctl --endpoints=http://localhost:2379 endpoint health
```

### 2. MySQL (Bitnami Chart)

**Deploy:**
```bash
helm install im-mysql bitnami/mysql \
  --namespace im-chat-system \
  -f deploy/k8s/infra/mysql-values.yaml
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=mysql
```

**Access:**
```bash
# Port forward
kubectl port-forward -n im-chat-system svc/im-mysql 3306:3306

# Get password
kubectl get secret -n im-chat-system im-mysql -o jsonpath="{.data.mysql-password}" | base64 -d
```

### 3. Redis (Bitnami Chart)

**Deploy:**
```bash
helm install im-redis bitnami/redis \
  --namespace im-chat-system \
  -f deploy/k8s/infra/redis-values.yaml
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=redis
```

**Access:**
```bash
# Port forward
kubectl port-forward -n im-chat-system svc/im-redis-master 6379:6379

# Test connection
redis-cli -h localhost -p 6379 PING
```

### 4. Kafka (Bitnami Chart - KRaft mode)

**Deploy:**
```bash
helm install im-kafka bitnami/kafka \
  --namespace im-chat-system \
  -f deploy/k8s/infra/kafka-values.yaml
```

**Create topics:**
```bash
# Wait for Kafka to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka -n im-chat-system --timeout=300s

# Create topics using kafka client pod
kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:3.6.0 --namespace im-chat-system --command -- sleep infinity

# Create group_msg topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic group_msg \
  --partitions 32 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# Create offline_msg topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic offline_msg \
  --partitions 64 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config min.insync.replicas=2

# Create membership_change topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic membership_change \
  --partitions 16 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# Verify topics
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --list
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=kafka
```

### 5. Higress API Gateway (Higress Chart)

Higress provides cloud-native API gateway with gRPC-Web support.

**Deploy:**
```bash
helm install higress higress/higress \
  --namespace higress-system \
  --create-namespace \
  -f deploy/k8s/infra/higress-values.yaml
```

**Configuration:**
- 2 controller replicas
- 2 gateway replicas
- Autoscaling enabled (2-10 replicas)
- LoadBalancer service type
- Prometheus metrics enabled

**Verify:**
```bash
kubectl get pods -n higress-system
kubectl get svc -n higress-system
```

**Access:**
```bash
# Get LoadBalancer IP
kubectl get svc -n higress-system higress-gateway

# Or port forward for testing
kubectl port-forward -n higress-system svc/higress-gateway 8080:80
```

## Database Migrations (Liquibase)

After MySQL is deployed, run database migrations:

```bash
# Build Liquibase image (if not already built)
docker build -t im-liquibase:latest apps/im-chat-system/migrations/

# Create Kubernetes Job for migration
kubectl create job im-liquibase-migration \
  --image=im-liquibase:latest \
  --namespace=im-chat-system \
  -- liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://im-mysql:3306/im_chat?useSSL=false \
  --username=im_service \
  --password=im_service_password \
  --driver=com.mysql.cj.jdbc.Driver \
  --contexts=prod \
  update

# Check migration status
kubectl logs -n im-chat-system job/im-liquibase-migration
```

## Complete Deployment Script

Use the provided script to deploy all infrastructure components:

```bash
# Deploy everything
./deploy/k8s/infra/deploy-all.sh

# Or with custom namespace
NAMESPACE=my-namespace ./deploy/k8s/infra/deploy-all.sh
```

The script will:
1. Create namespace
2. Add Helm repositories
3. Deploy etcd cluster (Bitnami chart)
4. Deploy MySQL (Bitnami chart)
5. Deploy Redis (Bitnami chart)
6. Deploy Kafka (Bitnami chart)
7. Create Kafka topics
8. Deploy Higress API Gateway (Higress chart)

## Uninstall

```bash
# Uninstall Helm releases
helm uninstall im-etcd -n im-chat-system
helm uninstall im-mysql -n im-chat-system
helm uninstall im-redis -n im-chat-system
helm uninstall im-kafka -n im-chat-system
helm uninstall higress -n higress-system

# Delete namespaces (WARNING: This deletes all data!)
kubectl delete namespace im-chat-system
kubectl delete namespace higress-system
```

## Monitoring

All Bitnami charts come with built-in Prometheus metrics exporters. Enable them:

```bash
# etcd with metrics
helm upgrade im-etcd bitnami/etcd \
  --namespace im-chat-system \
  -f deploy/k8s/infra/etcd-values.yaml \
  --set metrics.enabled=true

# MySQL with metrics
helm upgrade im-mysql bitnami/mysql \
  --namespace im-chat-system \
  -f deploy/k8s/infra/mysql-values.yaml \
  --set metrics.enabled=true

# Redis with metrics
helm upgrade im-redis bitnami/redis \
  --namespace im-chat-system \
  -f deploy/k8s/infra/redis-values.yaml \
  --set metrics.enabled=true

# Kafka with metrics
helm upgrade im-kafka bitnami/kafka \
  --namespace im-chat-system \
  -f deploy/k8s/infra/kafka-values.yaml \
  --set metrics.kafka.enabled=true
```

Higress has metrics enabled by default on port 15020.

## References

- [Bitnami etcd Chart](https://github.com/bitnami/charts/tree/main/bitnami/etcd)
- [Bitnami MySQL Chart](https://github.com/bitnami/charts/tree/main/bitnami/mysql)
- [Bitnami Redis Chart](https://github.com/bitnami/charts/tree/main/bitnami/redis)
- [Bitnami Kafka Chart](https://github.com/bitnami/charts/tree/main/bitnami/kafka)
- [Higress Documentation](https://higress.io/docs/)
- [Higress Helm Chart](https://github.com/alibaba/higress/tree/main/helm)



**Deploy:**
```bash
# Create namespace
kubectl create namespace im-chat-system

# Install MySQL with custom values
helm install im-mysql bitnami/mysql \
  --namespace im-chat-system \
  --set auth.rootPassword=im_root_password \
  --set auth.database=im_chat \
  --set auth.username=im_service \
  --set auth.password=im_service_password \
  --set primary.persistence.size=20Gi \
  --set primary.resources.requests.memory=512Mi \
  --set primary.resources.requests.cpu=250m \
  --set primary.resources.limits.memory=2Gi \
  --set primary.resources.limits.cpu=1000m \
  --set primary.configuration="[mysqld]\nmax_connections=500\ndefault_authentication_plugin=mysql_native_password"
```

**Or use values file:**
```bash
helm install im-mysql bitnami/mysql \
  --namespace im-chat-system \
  -f deploy/infra/mysql-values.yaml
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=mysql
```

**Access:**
```bash
# Port forward
kubectl port-forward -n im-chat-system svc/im-mysql 3306:3306

# Get password
kubectl get secret -n im-chat-system im-mysql -o jsonpath="{.data.mysql-password}" | base64 -d
```

### 3. Redis (Bitnami Chart)

**Deploy:**
```bash
helm install im-redis bitnami/redis \
  --namespace im-chat-system \
  --set auth.enabled=false \
  --set master.persistence.size=10Gi \
  --set master.resources.requests.memory=256Mi \
  --set master.resources.requests.cpu=100m \
  --set master.resources.limits.memory=2Gi \
  --set master.resources.limits.cpu=500m \
  --set master.configuration="maxmemory 2gb\nmaxmemory-policy allkeys-lru\nappendonly yes"
```

**Or use values file:**
```bash
helm install im-redis bitnami/redis \
  --namespace im-chat-system \
  -f deploy/infra/redis-values.yaml
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=redis
```

**Access:**
```bash
# Port forward
kubectl port-forward -n im-chat-system svc/im-redis-master 6379:6379

# Test connection
redis-cli -h localhost -p 6379 PING
```

### 4. Kafka (Bitnami Chart - KRaft mode)

**Deploy:**
```bash
helm install im-kafka bitnami/kafka \
  --namespace im-chat-system \
  --set controller.replicaCount=3 \
  --set kraft.enabled=true \
  --set controller.persistence.size=20Gi \
  --set controller.resources.requests.memory=512Mi \
  --set controller.resources.requests.cpu=250m \
  --set controller.resources.limits.memory=2Gi \
  --set controller.resources.limits.cpu=1000m \
  --set listeners.client.protocol=PLAINTEXT \
  --set listeners.controller.protocol=PLAINTEXT \
  --set listeners.interbroker.protocol=PLAINTEXT
```

**Or use values file:**
```bash
helm install im-kafka bitnami/kafka \
  --namespace im-chat-system \
  -f deploy/infra/kafka-values.yaml
```

**Create topics:**
```bash
# Wait for Kafka to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka -n im-chat-system --timeout=300s

# Create topics using kafka client pod
kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:3.6.0 --namespace im-chat-system --command -- sleep infinity

# Create group_msg topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic group_msg \
  --partitions 32 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# Create offline_msg topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic offline_msg \
  --partitions 64 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config min.insync.replicas=2

# Create membership_change topic
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic membership_change \
  --partitions 16 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# Verify topics
kubectl exec -it kafka-client -n im-chat-system -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --list
```

**Verify:**
```bash
kubectl get pods -n im-chat-system -l app.kubernetes.io/name=kafka
```

## Helm Values Files

Create custom values files for each environment:

### `deploy/infra/mysql-values.yaml`
```yaml
auth:
  rootPassword: "im_root_password"
  database: "im_chat"
  username: "im_service"
  password: "im_service_password"

primary:
  persistence:
    size: 20Gi
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: 2Gi
      cpu: 1000m
  configuration: |
    [mysqld]
    max_connections=500
    default_authentication_plugin=mysql_native_password
```

### `deploy/infra/redis-values.yaml`
```yaml
auth:
  enabled: false

master:
  persistence:
    size: 10Gi
  resources:
    requests:
      memory: 256Mi
      cpu: 100m
    limits:
      memory: 2Gi
      cpu: 500m
  configuration: |
    maxmemory 2gb
    maxmemory-policy allkeys-lru
    appendonly yes
```

### `deploy/infra/kafka-values.yaml`
```yaml
controller:
  replicaCount: 3
  persistence:
    size: 20Gi
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: 2Gi
      cpu: 1000m

kraft:
  enabled: true

listeners:
  client:
    protocol: PLAINTEXT
  controller:
    protocol: PLAINTEXT
  interbroker:
    protocol: PLAINTEXT
```

## Database Migrations (Liquibase)

After MySQL is deployed, run database migrations:

```bash
# Build Liquibase image (if not already built)
docker build -t im-liquibase:latest apps/im-chat-system/migrations/

# Create Kubernetes Job for migration
kubectl create job im-liquibase-migration \
  --image=im-liquibase:latest \
  --namespace=im-chat-system \
  -- liquibase \
  --changelog-file=changelog/db.changelog-master.yaml \
  --url=jdbc:mysql://im-mysql:3306/im_chat?useSSL=false \
  --username=im_service \
  --password=im_service_password \
  --driver=com.mysql.cj.jdbc.Driver \
  --contexts=prod \
  update

# Check migration status
kubectl logs -n im-chat-system job/im-liquibase-migration
```

## Complete Deployment Script

Use the provided script to deploy all infrastructure components:

```bash
# Deploy everything
./deploy/k8s/infra/deploy-all.sh

# Or with custom namespace
NAMESPACE=my-namespace ./deploy/k8s/infra/deploy-all.sh
```

The script will:
1. Create namespace
2. Add Helm repositories
3. Deploy etcd cluster (Bitnami chart)
4. Deploy MySQL (Bitnami chart)
5. Deploy Redis (Bitnami chart)
6. Deploy Kafka (Bitnami chart)
7. Create Kafka topics
8. Deploy Higress API Gateway (Higress chart)

## Uninstall

```bash
# Uninstall Helm releases
helm uninstall im-etcd -n im-chat-system
helm uninstall im-mysql -n im-chat-system
helm uninstall im-redis -n im-chat-system
helm uninstall im-kafka -n im-chat-system
helm uninstall higress -n higress-system

# Delete namespaces (WARNING: This deletes all data!)
kubectl delete namespace im-chat-system
kubectl delete namespace higress-system
```

## Monitoring

All Bitnami charts come with built-in Prometheus metrics exporters. Enable them:

```bash
# etcd with metrics
helm upgrade im-etcd bitnami/etcd \
  --namespace im-chat-system \
  -f deploy/k8s/infra/etcd-values.yaml \
  --set metrics.enabled=true

# MySQL with metrics
helm upgrade im-mysql bitnami/mysql \
  --namespace im-chat-system \
  -f deploy/k8s/infra/mysql-values.yaml \
  --set metrics.enabled=true

# Redis with metrics
helm upgrade im-redis bitnami/redis \
  --namespace im-chat-system \
  -f deploy/k8s/infra/redis-values.yaml \
  --set metrics.enabled=true

# Kafka with metrics
helm upgrade im-kafka bitnami/kafka \
  --namespace im-chat-system \
  -f deploy/k8s/infra/kafka-values.yaml \
  --set metrics.kafka.enabled=true
```

Higress has metrics enabled by default on port 15020.

## Helm Values Files

All infrastructure components use custom values files for configuration:

- `deploy/k8s/infra/etcd-values.yaml` - etcd cluster configuration
- `deploy/k8s/infra/mysql-values.yaml` - MySQL configuration
- `deploy/k8s/infra/redis-values.yaml` - Redis configuration
- `deploy/k8s/infra/kafka-values.yaml` - Kafka configuration
- `deploy/k8s/infra/higress-values.yaml` - Higress gateway configuration

## References

- [Bitnami etcd Chart](https://github.com/bitnami/charts/tree/main/bitnami/etcd)
- [Bitnami MySQL Chart](https://github.com/bitnami/charts/tree/main/bitnami/mysql)
- [Bitnami Redis Chart](https://github.com/bitnami/charts/tree/main/bitnami/redis)
- [Bitnami Kafka Chart](https://github.com/bitnami/charts/tree/main/bitnami/kafka)
- [Higress Documentation](https://higress.io/docs/)
- [Higress Helm Chart](https://github.com/alibaba/higress/tree/main/helm)
