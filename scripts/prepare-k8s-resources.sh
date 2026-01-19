#!/bin/bash
set -e

# Script to prepare K8s resources for Kustomize
# This copies app-specific K8s resources to the base directory
# so Kustomize can find them (Kustomize requires resources to be
# within or below the kustomization directory)

echo "Preparing K8s resources for Kustomize..."

# Create base directory if it doesn't exist
mkdir -p k8s/base

# Copy hello-service resources
if [ -f apps/hello-service/k8s/deployment.yaml ]; then
  cp apps/hello-service/k8s/deployment.yaml k8s/base/hello-service-deployment.yaml
  echo "✓ Copied hello-service deployment"
fi
if [ -f apps/hello-service/k8s/service.yaml ]; then
  cp apps/hello-service/k8s/service.yaml k8s/base/hello-service-service.yaml
  echo "✓ Copied hello-service service"
fi
if [ -f apps/hello-service/k8s/configmap.yaml ]; then
  cp apps/hello-service/k8s/configmap.yaml k8s/base/hello-service-configmap.yaml
  echo "✓ Copied hello-service configmap"
fi

# Copy todo-service resources
if [ -f apps/todo-service/k8s/deployment.yaml ]; then
  cp apps/todo-service/k8s/deployment.yaml k8s/base/todo-service-deployment.yaml
  echo "✓ Copied todo-service deployment"
fi
if [ -f apps/todo-service/k8s/service.yaml ]; then
  cp apps/todo-service/k8s/service.yaml k8s/base/todo-service-service.yaml
  echo "✓ Copied todo-service service"
fi

# Copy ingress
if [ -f tools/k8s/ingress.yaml ]; then
  cp tools/k8s/ingress.yaml k8s/base/ingress.yaml
  echo "✓ Copied ingress"
fi

echo ""
echo "✅ K8s resources prepared successfully"
echo ""
echo "Files in k8s/base/:"
ls -la k8s/base/*.yaml 2>/dev/null || echo "No YAML files found"
