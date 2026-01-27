#!/bin/bash
# Script to update references from tools/ to deploy/

echo "Updating references from tools/ to deploy/..."

# Update tools/envoy references to deploy/docker
find . -type f -name "*.md" -exec sed -i '' 's|tools/envoy/envoy-docker\.yaml|deploy/docker/envoy-config.yaml|g' {} +
find . -type f -name "*.md" -exec sed -i '' 's|tools/envoy/envoy-local\.yaml|deploy/docker/envoy-local-config.yaml|g' {} +
find . -type f -name "*.md" -exec sed -i '' 's|tools/envoy|deploy/docker|g' {} +

# Update tools/higress references to deploy/k8s/services/higress
find . -type f -name "*.md" -exec sed -i '' 's|tools/higress|deploy/k8s/services/higress|g' {} +

# Update tools/k8s references to deploy/k8s/services/higress
find . -type f -name "*.md" -exec sed -i '' 's|tools/k8s/ingress\.yaml|deploy/k8s/services/higress/higress-routes.yaml|g' {} +
find . -type f -name "*.md" -exec sed -i '' 's|tools/k8s|deploy/k8s/services/higress|g' {} +

# Update generic tools/ references
find . -type f -name "*.md" -exec sed -i '' 's|/tools/|/deploy/|g' {} +

echo "✅ References updated successfully"
echo ""
echo "Files updated:"
git diff --name-only | grep "\.md$" || echo "No markdown files changed"
