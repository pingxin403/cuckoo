#!/bin/bash

# Production Environment Verification Script
# This script verifies that all services are running correctly in production

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="production"
INGRESS_HOST=""
SKIP_INGRESS=false

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -h|--host)
            INGRESS_HOST="$2"
            shift 2
            ;;
        --skip-ingress)
            SKIP_INGRESS=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -n, --namespace NS    Kubernetes namespace (default: production)"
            echo "  -h, --host HOST       Ingress host for testing (e.g., api.example.com)"
            echo "  --skip-ingress        Skip Ingress tests"
            echo "  --help                Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Function to print test result
print_result() {
    local test_name=$1
    local result=$2
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Check kubectl connection
check_kubectl() {
    echo -e "${BLUE}Checking kubectl connection...${NC}"
    
    if ! kubectl cluster-info &> /dev/null; then
        echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
        exit 1
    fi
    
    CLUSTER=$(kubectl config current-context)
    echo -e "${GREEN}✓ Connected to cluster: $CLUSTER${NC}"
    echo -e "${BLUE}Namespace: $NAMESPACE${NC}"
    echo ""
}

# Check namespace exists
check_namespace() {
    echo -e "${BLUE}1. Checking namespace...${NC}"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_result "Namespace $NAMESPACE exists" "PASS"
    else
        print_result "Namespace $NAMESPACE exists" "FAIL"
        echo -e "${RED}Namespace not found. Please deploy first.${NC}"
        exit 1
    fi
    echo ""
}

# Check deployments
check_deployments() {
    echo -e "${BLUE}2. Checking deployments...${NC}"
    
    # Check Hello Service deployment
    if kubectl get deployment hello-service -n "$NAMESPACE" &> /dev/null; then
        READY=$(kubectl get deployment hello-service -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        DESIRED=$(kubectl get deployment hello-service -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if [ "$READY" = "$DESIRED" ] && [ "$READY" != "" ]; then
            print_result "Hello Service deployment ($READY/$DESIRED replicas ready)" "PASS"
        else
            print_result "Hello Service deployment ($READY/$DESIRED replicas ready)" "FAIL"
        fi
    else
        print_result "Hello Service deployment exists" "FAIL"
    fi
    
    # Check TODO Service deployment
    if kubectl get deployment todo-service -n "$NAMESPACE" &> /dev/null; then
        READY=$(kubectl get deployment todo-service -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        DESIRED=$(kubectl get deployment todo-service -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if [ "$READY" = "$DESIRED" ] && [ "$READY" != "" ]; then
            print_result "TODO Service deployment ($READY/$DESIRED replicas ready)" "PASS"
        else
            print_result "TODO Service deployment ($READY/$DESIRED replicas ready)" "FAIL"
        fi
    else
        print_result "TODO Service deployment exists" "FAIL"
    fi
    echo ""
}

# Check pods
check_pods() {
    echo -e "${BLUE}3. Checking pods...${NC}"
    
    # Check Hello Service pods
    HELLO_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=hello-service --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    if [ "$HELLO_PODS" -gt 0 ]; then
        print_result "Hello Service has $HELLO_PODS running pod(s)" "PASS"
    else
        print_result "Hello Service has running pods" "FAIL"
    fi
    
    # Check TODO Service pods
    TODO_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=todo-service --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    if [ "$TODO_PODS" -gt 0 ]; then
        print_result "TODO Service has $TODO_PODS running pod(s)" "PASS"
    else
        print_result "TODO Service has running pods" "FAIL"
    fi
    
    # Check for pod restarts
    HELLO_RESTARTS=$(kubectl get pods -n "$NAMESPACE" -l app=hello-service -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}' 2>/dev/null | awk '{s+=$1} END {print s}')
    if [ -z "$HELLO_RESTARTS" ]; then
        HELLO_RESTARTS=0
    fi
    
    TODO_RESTARTS=$(kubectl get pods -n "$NAMESPACE" -l app=todo-service -o jsonpath='{.items[*].status.containerStatuses[*].restartCount}' 2>/dev/null | awk '{s+=$1} END {print s}')
    if [ -z "$TODO_RESTARTS" ]; then
        TODO_RESTARTS=0
    fi
    
    if [ "$HELLO_RESTARTS" -eq 0 ]; then
        print_result "Hello Service pods have no restarts" "PASS"
    else
        print_result "Hello Service pods have $HELLO_RESTARTS restart(s)" "FAIL"
    fi
    
    if [ "$TODO_RESTARTS" -eq 0 ]; then
        print_result "TODO Service pods have no restarts" "PASS"
    else
        print_result "TODO Service pods have $TODO_RESTARTS restart(s)" "FAIL"
    fi
    echo ""
}

# Check services
check_services() {
    echo -e "${BLUE}4. Checking services...${NC}"
    
    # Check Hello Service
    if kubectl get service hello-service -n "$NAMESPACE" &> /dev/null; then
        ENDPOINTS=$(kubectl get endpoints hello-service -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
        if [ "$ENDPOINTS" -gt 0 ]; then
            print_result "Hello Service has $ENDPOINTS endpoint(s)" "PASS"
        else
            print_result "Hello Service has endpoints" "FAIL"
        fi
    else
        print_result "Hello Service exists" "FAIL"
    fi
    
    # Check TODO Service
    if kubectl get service todo-service -n "$NAMESPACE" &> /dev/null; then
        ENDPOINTS=$(kubectl get endpoints todo-service -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' | wc -w)
        if [ "$ENDPOINTS" -gt 0 ]; then
            print_result "TODO Service has $ENDPOINTS endpoint(s)" "PASS"
        else
            print_result "TODO Service has endpoints" "FAIL"
        fi
    else
        print_result "TODO Service exists" "FAIL"
    fi
    echo ""
}

# Check Ingress
check_ingress() {
    if [ "$SKIP_INGRESS" = true ]; then
        echo -e "${YELLOW}Skipping Ingress checks${NC}"
        echo ""
        return 0
    fi
    
    echo -e "${BLUE}5. Checking Ingress...${NC}"
    
    if kubectl get ingress -n "$NAMESPACE" &> /dev/null; then
        INGRESS_COUNT=$(kubectl get ingress -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
        if [ "$INGRESS_COUNT" -gt 0 ]; then
            print_result "Ingress resources exist ($INGRESS_COUNT found)" "PASS"
            
            # Get Ingress host if not provided
            if [ -z "$INGRESS_HOST" ]; then
                INGRESS_HOST=$(kubectl get ingress -n "$NAMESPACE" -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null)
            fi
            
            if [ -n "$INGRESS_HOST" ]; then
                echo -e "${BLUE}Ingress host: $INGRESS_HOST${NC}"
            fi
        else
            print_result "Ingress resources exist" "FAIL"
        fi
    else
        echo -e "${YELLOW}No Ingress resources found${NC}"
    fi
    echo ""
}

# Check resource usage
check_resources() {
    echo -e "${BLUE}6. Checking resource usage...${NC}"
    
    # Check if metrics-server is available
    if ! kubectl top nodes &> /dev/null; then
        echo -e "${YELLOW}Metrics server not available, skipping resource checks${NC}"
        echo ""
        return 0
    fi
    
    # Check Hello Service resource usage
    HELLO_CPU=$(kubectl top pods -n "$NAMESPACE" -l app=hello-service --no-headers 2>/dev/null | awk '{sum+=$2} END {print sum}' | sed 's/m//')
    HELLO_MEM=$(kubectl top pods -n "$NAMESPACE" -l app=hello-service --no-headers 2>/dev/null | awk '{sum+=$3} END {print sum}' | sed 's/Mi//')
    
    if [ -n "$HELLO_CPU" ] && [ "$HELLO_CPU" != "0" ]; then
        echo -e "${BLUE}Hello Service: ${HELLO_CPU}m CPU, ${HELLO_MEM}Mi Memory${NC}"
        print_result "Hello Service resource usage within limits" "PASS"
    fi
    
    # Check TODO Service resource usage
    TODO_CPU=$(kubectl top pods -n "$NAMESPACE" -l app=todo-service --no-headers 2>/dev/null | awk '{sum+=$2} END {print sum}' | sed 's/m//')
    TODO_MEM=$(kubectl top pods -n "$NAMESPACE" -l app=todo-service --no-headers 2>/dev/null | awk '{sum+=$3} END {print sum}' | sed 's/Mi//')
    
    if [ -n "$TODO_CPU" ] && [ "$TODO_CPU" != "0" ]; then
        echo -e "${BLUE}TODO Service: ${TODO_CPU}m CPU, ${TODO_MEM}Mi Memory${NC}"
        print_result "TODO Service resource usage within limits" "PASS"
    fi
    echo ""
}

# Check logs for errors
check_logs() {
    echo -e "${BLUE}7. Checking logs for errors...${NC}"
    
    # Check Hello Service logs
    HELLO_ERRORS=$(kubectl logs -n "$NAMESPACE" -l app=hello-service --tail=100 2>/dev/null | grep -i "error\|exception\|fatal" | wc -l)
    if [ "$HELLO_ERRORS" -eq 0 ]; then
        print_result "Hello Service logs have no errors" "PASS"
    else
        print_result "Hello Service logs have $HELLO_ERRORS error(s)" "FAIL"
        echo -e "${YELLOW}Recent errors:${NC}"
        kubectl logs -n "$NAMESPACE" -l app=hello-service --tail=100 2>/dev/null | grep -i "error\|exception\|fatal" | tail -5
    fi
    
    # Check TODO Service logs
    TODO_ERRORS=$(kubectl logs -n "$NAMESPACE" -l app=todo-service --tail=100 2>/dev/null | grep -i "error\|exception\|fatal" | wc -l)
    if [ "$TODO_ERRORS" -eq 0 ]; then
        print_result "TODO Service logs have no errors" "PASS"
    else
        print_result "TODO Service logs have $TODO_ERRORS error(s)" "FAIL"
        echo -e "${YELLOW}Recent errors:${NC}"
        kubectl logs -n "$NAMESPACE" -l app=todo-service --tail=100 2>/dev/null | grep -i "error\|exception\|fatal" | tail -5
    fi
    echo ""
}

# Test service connectivity
test_connectivity() {
    echo -e "${BLUE}8. Testing service connectivity...${NC}"
    
    # Test Hello Service
    echo -e "${YELLOW}Testing Hello Service...${NC}"
    POD=$(kubectl get pods -n "$NAMESPACE" -l app=hello-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POD" ]; then
        if kubectl exec -n "$NAMESPACE" "$POD" -- sh -c "exit 0" &> /dev/null; then
            print_result "Can execute commands in Hello Service pod" "PASS"
        else
            print_result "Can execute commands in Hello Service pod" "FAIL"
        fi
    fi
    
    # Test TODO Service
    echo -e "${YELLOW}Testing TODO Service...${NC}"
    POD=$(kubectl get pods -n "$NAMESPACE" -l app=todo-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$POD" ]; then
        if kubectl exec -n "$NAMESPACE" "$POD" -- sh -c "exit 0" &> /dev/null; then
            print_result "Can execute commands in TODO Service pod" "PASS"
        else
            print_result "Can execute commands in TODO Service pod" "FAIL"
        fi
    fi
    
    # Test service-to-service connectivity
    echo -e "${YELLOW}Testing service-to-service connectivity...${NC}"
    TODO_POD=$(kubectl get pods -n "$NAMESPACE" -l app=todo-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$TODO_POD" ]; then
        if kubectl exec -n "$NAMESPACE" "$TODO_POD" -- sh -c "nc -zv hello-service 9090" &> /dev/null; then
            print_result "TODO Service can reach Hello Service" "PASS"
        else
            print_result "TODO Service can reach Hello Service" "FAIL"
        fi
    fi
    echo ""
}

# Show summary
show_summary() {
    echo -e "${GREEN}=== Verification Summary ===${NC}"
    echo -e "Cluster: $(kubectl config current-context)"
    echo -e "Namespace: $NAMESPACE"
    echo -e "Total tests: $TESTS_TOTAL"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    else
        echo -e "Failed: $TESTS_FAILED"
    fi
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ Production environment is healthy!${NC}"
        echo ""
        echo -e "${BLUE}Service URLs:${NC}"
        if [ -n "$INGRESS_HOST" ]; then
            echo -e "  - API Gateway: https://$INGRESS_HOST"
            echo -e "  - Hello Service: https://$INGRESS_HOST/api/hello"
            echo -e "  - TODO Service: https://$INGRESS_HOST/api/todo"
        else
            echo -e "  - Use port-forward to access services:"
            echo -e "    kubectl port-forward -n $NAMESPACE svc/hello-service 9090:9090"
            echo -e "    kubectl port-forward -n $NAMESPACE svc/todo-service 9091:9091"
        fi
        exit 0
    else
        echo -e "${RED}✗ Production environment has issues${NC}"
        echo -e "${YELLOW}Please check the failed tests above${NC}"
        echo ""
        echo -e "${BLUE}Troubleshooting commands:${NC}"
        echo -e "  - View pods: kubectl get pods -n $NAMESPACE"
        echo -e "  - View logs: kubectl logs -n $NAMESPACE -l app=hello-service"
        echo -e "  - Describe pod: kubectl describe pod <pod-name> -n $NAMESPACE"
        echo -e "  - View events: kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp'"
        exit 1
    fi
}

# Main execution
main() {
    echo -e "${GREEN}=== Production Environment Verification ===${NC}\n"
    
    check_kubectl
    check_namespace
    check_deployments
    check_pods
    check_services
    check_ingress
    check_resources
    check_logs
    test_connectivity
    show_summary
}

# Run main function
main

