#!/bin/bash

# Docker Build Verification Script
# This script builds and verifies Docker images for all services

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Docker is installed and running
check_docker() {
    echo -e "${BLUE}Checking Docker installation...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker is not installed${NC}"
        echo -e "${YELLOW}Please install Docker from https://www.docker.com/get-started${NC}"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        echo -e "${RED}Error: Docker daemon is not running${NC}"
        echo -e "${YELLOW}Please start Docker Desktop or the Docker daemon${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker is installed and running${NC}"
}

# Build Hello Service image
build_hello_service() {
    echo -e "\n${BLUE}Building Hello Service Docker image...${NC}"
    
    # Ensure Protobuf code is generated
    echo -e "${YELLOW}Ensuring Protobuf code is generated...${NC}"
    make gen-proto-java || true
    
    # Build the image
    if docker build -t hello-service:latest -f apps/hello-service/Dockerfile apps/hello-service; then
        echo -e "${GREEN}✓ Hello Service image built successfully${NC}"
        
        # Get image size
        SIZE=$(docker images hello-service:latest --format "{{.Size}}")
        echo -e "${BLUE}Image size: $SIZE${NC}"
        
        return 0
    else
        echo -e "${RED}✗ Failed to build Hello Service image${NC}"
        return 1
    fi
}

# Build TODO Service image
build_todo_service() {
    echo -e "\n${BLUE}Building TODO Service Docker image...${NC}"
    
    # Ensure Protobuf code is generated
    echo -e "${YELLOW}Ensuring Protobuf code is generated...${NC}"
    make gen-proto-go || true
    
    # Build the image
    if docker build -t todo-service:latest -f apps/todo-service/Dockerfile apps/todo-service; then
        echo -e "${GREEN}✓ TODO Service image built successfully${NC}"
        
        # Get image size
        SIZE=$(docker images todo-service:latest --format "{{.Size}}")
        echo -e "${BLUE}Image size: $SIZE${NC}"
        
        return 0
    else
        echo -e "${RED}✗ Failed to build TODO Service image${NC}"
        return 1
    fi
}

# Verify image can run
verify_image() {
    local image=$1
    local port=$2
    local container_name=$3
    
    echo -e "\n${BLUE}Verifying $image can run...${NC}"
    
    # Remove existing container if it exists
    docker rm -f $container_name 2>/dev/null || true
    
    # Run the container
    if docker run -d --name $container_name -p $port:$port $image; then
        echo -e "${GREEN}✓ Container started${NC}"
        
        # Wait a few seconds for startup
        echo -e "${YELLOW}Waiting for service to start...${NC}"
        sleep 5
        
        # Check if container is still running
        if docker ps | grep -q $container_name; then
            echo -e "${GREEN}✓ Container is running${NC}"
            
            # Show logs
            echo -e "${BLUE}Container logs:${NC}"
            docker logs $container_name | tail -n 10
            
            # Stop and remove container
            docker stop $container_name >/dev/null 2>&1
            docker rm $container_name >/dev/null 2>&1
            
            return 0
        else
            echo -e "${RED}✗ Container stopped unexpectedly${NC}"
            echo -e "${BLUE}Container logs:${NC}"
            docker logs $container_name
            docker rm $container_name >/dev/null 2>&1
            return 1
        fi
    else
        echo -e "${RED}✗ Failed to start container${NC}"
        return 1
    fi
}

# List built images
list_images() {
    echo -e "\n${BLUE}Built images:${NC}"
    docker images | grep -E "hello-service|todo-service|REPOSITORY"
}

# Main execution
main() {
    echo -e "${GREEN}=== Docker Build Verification ===${NC}\n"
    
    # Check Docker
    check_docker
    
    # Build images
    HELLO_SUCCESS=0
    TODO_SUCCESS=0
    
    if build_hello_service; then
        HELLO_SUCCESS=1
    fi
    
    if build_todo_service; then
        TODO_SUCCESS=1
    fi
    
    # Verify images can run
    if [ $HELLO_SUCCESS -eq 1 ]; then
        verify_image "hello-service:latest" "9090" "hello-service-test"
    fi
    
    if [ $TODO_SUCCESS -eq 1 ]; then
        verify_image "todo-service:latest" "9091" "todo-service-test"
    fi
    
    # List images
    list_images
    
    # Summary
    echo -e "\n${GREEN}=== Build Summary ===${NC}"
    
    if [ $HELLO_SUCCESS -eq 1 ]; then
        echo -e "${GREEN}✓ Hello Service image: hello-service:latest${NC}"
    else
        echo -e "${RED}✗ Hello Service image build failed${NC}"
    fi
    
    if [ $TODO_SUCCESS -eq 1 ]; then
        echo -e "${GREEN}✓ TODO Service image: todo-service:latest${NC}"
    else
        echo -e "${RED}✗ TODO Service image build failed${NC}"
    fi
    
    echo ""
    
    if [ $HELLO_SUCCESS -eq 1 ] && [ $TODO_SUCCESS -eq 1 ]; then
        echo -e "${GREEN}✓ All images built successfully!${NC}"
        echo ""
        echo -e "${BLUE}Next steps:${NC}"
        echo -e "  1. Tag images for your registry:"
        echo -e "     docker tag hello-service:latest registry.example.com/hello-service:v1.0.0"
        echo -e "     docker tag todo-service:latest registry.example.com/todo-service:v1.0.0"
        echo ""
        echo -e "  2. Push images to registry:"
        echo -e "     docker push registry.example.com/hello-service:v1.0.0"
        echo -e "     docker push registry.example.com/todo-service:v1.0.0"
        echo ""
        echo -e "  3. Deploy to Kubernetes:"
        echo -e "     kubectl apply -k k8s/overlays/production"
        exit 0
    else
        echo -e "${RED}✗ Some images failed to build${NC}"
        exit 1
    fi
}

# Run main function
main

