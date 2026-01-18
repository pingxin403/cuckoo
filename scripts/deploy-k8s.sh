#!/bin/bash

# Kubernetes Deployment Script
# This script deploys the Monorepo Hello/TODO Services to a Kubernetes cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
OVERLAY="production"
NAMESPACE="production"
DRY_RUN=false
SKIP_BUILD=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--overlay)
            OVERLAY="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -o, --overlay OVERLAY    Kustomize overlay to use (default: production)"
            echo "  -n, --namespace NS       Kubernetes namespace (default: production)"
            echo "  --dry-run                Show what would be deployed without applying"
            echo "  --skip-build             Skip Docker image build step"
            echo "  -h, --help               Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Check if kubectl is installed
check_kubectl() {
    echo -e "${BLUE}Checking kubectl installation...${NC}"
    
    if ! command -v kubectl &> /dev/null; then
        echo -e "${RED}Error: kubectl is not installed${NC}"
        echo -e "${YELLOW}Please install kubectl from https://kubernetes.io/docs/tasks/tools/${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ kubectl is installed${NC}"
}

# Check if kustomize is available
check_kustomize() {
    echo -e "${BLUE}Checking kustomize availability...${NC}"
    
    # Check if kustomize is available via kubectl
    if kubectl kustomize --help &> /dev/null; then
        echo -e "${GREEN}✓ kustomize is available via kubectl${NC}"
        return 0
    fi
    
    # Check if standalone kustomize is installed
    if command -v kustomize &> /dev/null; then
        echo -e "${GREEN}✓ kustomize is installed${NC}"
        return 0
    fi
    
    echo -e "${RED}Error: kustomize is not available${NC}"
    echo -e "${YELLOW}Install with: kubectl version (kustomize is built-in for kubectl 1.14+)${NC}"
    echo -e "${YELLOW}Or install standalone: https://kubectl.docs.kubernetes.io/installation/kustomize/${NC}"
    exit 1
}

# Check cluster connection
check_cluster() {
    echo -e "${BLUE}Checking Kubernetes cluster connection...${NC}"
    
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        echo -e "${YELLOW}Please check your kubeconfig and cluster connection${NC}"
        exit 1
    fi
    
    CLUSTER=$(kubectl config current-context)
    echo -e "${GREEN}✓ Connected to cluster: $CLUSTER${NC}"
    
    # Warn if deploying to production
    if [[ "$OVERLAY" == "production" ]]; then
        echo -e "${YELLOW}⚠ WARNING: You are about to deploy to PRODUCTION${NC}"
        echo -e "${YELLOW}Cluster: $CLUSTER${NC}"
        echo -e "${YELLOW}Namespace: $NAMESPACE${NC}"
        read -p "Are you sure you want to continue? (yes/no): " -r
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            echo -e "${BLUE}Deployment cancelled${NC}"
            exit 0
        fi
    fi
}

# Build Docker images
build_images() {
    if [ "$SKIP_BUILD" = true ]; then
        echo -e "${YELLOW}Skipping Docker image build${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Building Docker images...${NC}"
    
    if make docker-build; then
        echo -e "${GREEN}✓ Docker images built successfully${NC}"
    else
        echo -e "${RED}✗ Failed to build Docker images${NC}"
        exit 1
    fi
}

# Create namespace if it doesn't exist
create_namespace() {
    echo -e "${BLUE}Checking namespace: $NAMESPACE${NC}"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        echo -e "${GREEN}✓ Namespace $NAMESPACE exists${NC}"
    else
        echo -e "${YELLOW}Creating namespace: $NAMESPACE${NC}"
        if [ "$DRY_RUN" = false ]; then
            kubectl create namespace "$NAMESPACE"
            echo -e "${GREEN}✓ Namespace created${NC}"
        else
            echo -e "${BLUE}[DRY RUN] Would create namespace: $NAMESPACE${NC}"
        fi
    fi
}

# Validate Kustomize configuration
validate_kustomize() {
    echo -e "${BLUE}Validating Kustomize configuration...${NC}"
    
    KUSTOMIZE_PATH="k8s/overlays/$OVERLAY"
    
    if [ ! -d "$KUSTOMIZE_PATH" ]; then
        echo -e "${RED}Error: Kustomize overlay not found: $KUSTOMIZE_PATH${NC}"
        exit 1
    fi
    
    if kubectl kustomize "$KUSTOMIZE_PATH" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Kustomize configuration is valid${NC}"
    else
        echo -e "${RED}✗ Kustomize configuration has errors${NC}"
        kubectl kustomize "$KUSTOMIZE_PATH"
        exit 1
    fi
}

# Show what will be deployed
show_resources() {
    echo -e "${BLUE}Resources to be deployed:${NC}"
    kubectl kustomize "k8s/overlays/$OVERLAY" | grep -E "^kind:|^  name:" | sed 's/^/  /'
}

# Deploy to Kubernetes
deploy() {
    echo -e "${BLUE}Deploying to Kubernetes...${NC}"
    
    KUSTOMIZE_PATH="k8s/overlays/$OVERLAY"
    
    if [ "$DRY_RUN" = true ]; then
        echo -e "${BLUE}[DRY RUN] Would apply:${NC}"
        kubectl kustomize "$KUSTOMIZE_PATH"
        return 0
    fi
    
    if kubectl apply -k "$KUSTOMIZE_PATH"; then
        echo -e "${GREEN}✓ Deployment successful${NC}"
    else
        echo -e "${RED}✗ Deployment failed${NC}"
        exit 1
    fi
}

# Wait for deployments to be ready
wait_for_deployments() {
    if [ "$DRY_RUN" = true ]; then
        echo -e "${BLUE}[DRY RUN] Would wait for deployments${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Waiting for deployments to be ready...${NC}"
    
    # Wait for Hello Service
    echo -e "${YELLOW}Waiting for hello-service...${NC}"
    if kubectl rollout status deployment/hello-service -n "$NAMESPACE" --timeout=5m; then
        echo -e "${GREEN}✓ hello-service is ready${NC}"
    else
        echo -e "${RED}✗ hello-service failed to become ready${NC}"
        kubectl get pods -n "$NAMESPACE" -l app=hello-service
        exit 1
    fi
    
    # Wait for TODO Service
    echo -e "${YELLOW}Waiting for todo-service...${NC}"
    if kubectl rollout status deployment/todo-service -n "$NAMESPACE" --timeout=5m; then
        echo -e "${GREEN}✓ todo-service is ready${NC}"
    else
        echo -e "${RED}✗ todo-service failed to become ready${NC}"
        kubectl get pods -n "$NAMESPACE" -l app=todo-service
        exit 1
    fi
}

# Verify deployment
verify_deployment() {
    if [ "$DRY_RUN" = true ]; then
        echo -e "${BLUE}[DRY RUN] Would verify deployment${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Verifying deployment...${NC}"
    
    # Check pods
    echo -e "${YELLOW}Pods:${NC}"
    kubectl get pods -n "$NAMESPACE" -l "app in (hello-service,todo-service)"
    
    # Check services
    echo -e "${YELLOW}Services:${NC}"
    kubectl get services -n "$NAMESPACE" -l "app in (hello-service,todo-service)"
    
    # Check ingress (if exists)
    if kubectl get ingress -n "$NAMESPACE" &> /dev/null; then
        echo -e "${YELLOW}Ingress:${NC}"
        kubectl get ingress -n "$NAMESPACE"
    fi
}

# Show logs
show_logs() {
    if [ "$DRY_RUN" = true ]; then
        return 0
    fi
    
    echo -e "${BLUE}Recent logs:${NC}"
    
    echo -e "${YELLOW}Hello Service logs:${NC}"
    kubectl logs -n "$NAMESPACE" -l app=hello-service --tail=10 --prefix=true || true
    
    echo -e "${YELLOW}TODO Service logs:${NC}"
    kubectl logs -n "$NAMESPACE" -l app=todo-service --tail=10 --prefix=true || true
}

# Main execution
main() {
    echo -e "${GREEN}=== Kubernetes Deployment ===${NC}\n"
    
    # Pre-flight checks
    check_kubectl
    check_kustomize
    check_cluster
    
    # Build images
    build_images
    
    # Create namespace
    create_namespace
    
    # Validate configuration
    validate_kustomize
    
    # Show resources
    show_resources
    
    # Deploy
    deploy
    
    # Wait for deployments
    wait_for_deployments
    
    # Verify
    verify_deployment
    
    # Show logs
    show_logs
    
    # Summary
    echo -e "\n${GREEN}=== Deployment Complete ===${NC}"
    echo -e "${BLUE}Cluster: $(kubectl config current-context)${NC}"
    echo -e "${BLUE}Namespace: $NAMESPACE${NC}"
    echo -e "${BLUE}Overlay: $OVERLAY${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  1. Check pod status: kubectl get pods -n $NAMESPACE"
    echo -e "  2. View logs: kubectl logs -n $NAMESPACE -l app=hello-service"
    echo -e "  3. Test services: kubectl port-forward -n $NAMESPACE svc/hello-service 9090:9090"
    echo -e "  4. Access via Ingress (if configured)"
}

# Run main function
main

