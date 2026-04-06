#!/bin/bash
# Deploy all infrastructure components using Helm

set -e

NAMESPACE="${NAMESPACE:-im-chat-system}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_MIGRATIONS="${RUN_MIGRATIONS:-false}"
MIGRATION_JOB_NAME="${MIGRATION_JOB_NAME:-im-liquibase-migration}"
MIGRATION_IMAGE="${MIGRATION_IMAGE:-im-liquibase:latest}"
MIGRATION_CHANGELOG="${MIGRATION_CHANGELOG:-changelog/db.changelog-master.yaml}"
MIGRATION_DB_NAME="${MIGRATION_DB_NAME:-im_chat}"
MIGRATION_DB_USER="${MIGRATION_DB_USER:-im_service}"
MIGRATION_DB_PASSWORD="${MIGRATION_DB_PASSWORD:-im_service_password}"
MIGRATION_CONTEXTS="${MIGRATION_CONTEXTS:-prod}"
MIGRATION_TIMEOUT="${MIGRATION_TIMEOUT:-300s}"
MIGRATION_CLEANUP_ON_SUCCESS="${MIGRATION_CLEANUP_ON_SUCCESS:-false}"
MIGRATION_DRY_RUN="${MIGRATION_DRY_RUN:-false}"

echo "=========================================="
echo "Deploying IM Chat System Infrastructure"
echo "Namespace: $NAMESPACE"
echo "=========================================="

# Create namespace
echo ""
echo "Creating namespace..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Add Helm repositories
echo ""
echo "Adding Helm repositories..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add higress https://higress.io/helm-charts
helm repo update

# Deploy etcd (Bitnami Helm chart)
echo ""
echo "Deploying etcd cluster..."
helm upgrade --install im-etcd bitnami/etcd \
  --namespace $NAMESPACE \
  -f $SCRIPT_DIR/etcd-values.yaml \
  --wait \
  --timeout 10m

# Wait for etcd to be ready
echo "Waiting for etcd to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=etcd -n $NAMESPACE --timeout=300s || true

# Deploy MySQL
echo ""
echo "Deploying MySQL..."
helm upgrade --install im-mysql bitnami/mysql \
  --namespace $NAMESPACE \
  -f $SCRIPT_DIR/mysql-values.yaml \
  --wait \
  --timeout 10m

# Deploy Redis
echo ""
echo "Deploying Redis..."
helm upgrade --install im-redis bitnami/redis \
  --namespace $NAMESPACE \
  -f $SCRIPT_DIR/redis-values.yaml \
  --wait \
  --timeout 10m

# Deploy Kafka
echo ""
echo "Deploying Kafka (KRaft mode)..."
helm upgrade --install im-kafka bitnami/kafka \
  --namespace $NAMESPACE \
  -f $SCRIPT_DIR/kafka-values.yaml \
  --wait \
  --timeout 10m

# Wait for Kafka to be fully ready
echo ""
echo "Waiting for Kafka to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka -n $NAMESPACE --timeout=300s

# Create Kafka topics
echo ""
echo "Creating Kafka topics..."
kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:3.6.0 --namespace $NAMESPACE --command -- sleep infinity 2>/dev/null || true
kubectl wait --for=condition=ready pod/kafka-client -n $NAMESPACE --timeout=60s

# Create group_msg topic
kubectl exec -it kafka-client -n $NAMESPACE -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic group_msg \
  --partitions 32 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# Create offline_msg topic
kubectl exec -it kafka-client -n $NAMESPACE -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic offline_msg \
  --partitions 64 \
  --replication-factor 3 \
  --config retention.ms=604800000 \
  --config min.insync.replicas=2

# Create membership_change topic
kubectl exec -it kafka-client -n $NAMESPACE -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --create --if-not-exists \
  --topic membership_change \
  --partitions 16 \
  --replication-factor 3 \
  --config retention.ms=3600000 \
  --config min.insync.replicas=2

# List topics
echo ""
echo "Kafka topics created:"
kubectl exec -it kafka-client -n $NAMESPACE -- kafka-topics.sh \
  --bootstrap-server im-kafka:9092 \
  --list

# Deploy Higress API Gateway
echo ""
echo "Deploying Higress API Gateway..."
helm upgrade --install higress higress/higress \
  --namespace higress-system \
  --create-namespace \
  -f $SCRIPT_DIR/higress-values.yaml \
  --wait \
  --timeout 10m

# Wait for Higress to be ready
echo "Waiting for Higress to be ready..."
kubectl wait --for=condition=ready pod -l app=higress-gateway -n higress-system --timeout=300s || true

echo ""
if [ "$RUN_MIGRATIONS" = "true" ]; then
  echo "Running Liquibase migrations..."

  if [ -z "$MIGRATION_IMAGE" ]; then
    echo "ERROR: MIGRATION_IMAGE is required when RUN_MIGRATIONS=true"
    exit 1
  fi

  MIGRATION_URL="${MIGRATION_URL:-jdbc:mysql://im-mysql:3306/$MIGRATION_DB_NAME?useSSL=false}"

  kubectl delete job "$MIGRATION_JOB_NAME" -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true

  if [ "$MIGRATION_DRY_RUN" = "true" ]; then
    MIGRATION_COMMAND="updateSQL"
  else
    MIGRATION_COMMAND="update"
  fi

  kubectl create job "$MIGRATION_JOB_NAME" \
    --image="$MIGRATION_IMAGE" \
    --namespace="$NAMESPACE" \
    -- liquibase \
    --changelog-file="$MIGRATION_CHANGELOG" \
    --url="$MIGRATION_URL" \
    --username="$MIGRATION_DB_USER" \
    --password="$MIGRATION_DB_PASSWORD" \
    --driver=com.mysql.cj.jdbc.Driver \
    --contexts="$MIGRATION_CONTEXTS" \
    "$MIGRATION_COMMAND"

  if [ "$MIGRATION_DRY_RUN" = "true" ]; then
    kubectl logs -n "$NAMESPACE" -f "job/$MIGRATION_JOB_NAME"
  else
    kubectl wait --for=condition=complete --timeout="$MIGRATION_TIMEOUT" "job/$MIGRATION_JOB_NAME" -n "$NAMESPACE"
    kubectl logs -n "$NAMESPACE" "job/$MIGRATION_JOB_NAME"
  fi

  if [ "$MIGRATION_CLEANUP_ON_SUCCESS" = "true" ]; then
    kubectl delete job "$MIGRATION_JOB_NAME" -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
  fi
else
  echo "Skipping Liquibase migrations (RUN_MIGRATIONS=false)"
fi

echo ""
echo "=========================================="
echo "Infrastructure deployment complete!"
echo "=========================================="
echo ""
echo "Deployed components:"
echo "  ✅ etcd cluster (3 nodes) - Bitnami Helm chart"
echo "  ✅ MySQL database - Bitnami Helm chart"
if [ "$RUN_MIGRATIONS" = "true" ]; then
  if [ "$MIGRATION_DRY_RUN" = "true" ]; then
    echo "  ✅ Liquibase migration dry-run executed"
  else
    echo "  ✅ Liquibase migrations executed"
  fi
else
  echo "  ⏭️  Liquibase migrations skipped"
fi
echo "  ✅ Redis cache - Bitnami Helm chart"
echo "  ✅ Kafka cluster (3 brokers, KRaft mode) - Bitnami Helm chart"
echo "  ✅ Higress API Gateway - Higress Helm chart"
echo ""
echo "Next steps:"
echo "1. Run migrations by re-running with RUN_MIGRATIONS=true (optional MIGRATION_* overrides)"
echo "2. Deploy application services: kubectl apply -k deploy/k8s/overlays/..."
echo "3. Verify all pods are running: kubectl get pods -n $NAMESPACE"
echo "4. Check Higress gateway: kubectl get svc -n higress-system"
echo ""
