#!/bin/bash

# Multi-Region Documentation Reorganization Script
# This script reorganizes the multi-region documentation structure

set -e  # Exit on error

echo "========================================="
echo "Multi-Region Docs Reorganization"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if we're in the project root
if [ ! -d ".kiro" ] || [ ! -d "docs" ]; then
    print_error "Please run this script from the project root directory"
    exit 1
fi

echo "Step 1: Creating new directory structure..."
echo "-------------------------------------------"

# Create new directories
mkdir -p docs/operations/multi-region
print_success "Created docs/operations/multi-region/"

mkdir -p .kiro/specs/multi-region-active-active/blog
print_success "Created .kiro/specs/multi-region-active-active/blog/"

echo ""
echo "Step 2: Moving architecture documents..."
echo "-------------------------------------------"

# Move architecture overview
if [ -f "docs/multi-region-demo/architecture-overview.md" ]; then
    mv docs/multi-region-demo/architecture-overview.md \
       docs/architecture/MULTI_REGION_ACTIVE_ACTIVE.md
    print_success "Moved architecture-overview.md → MULTI_REGION_ACTIVE_ACTIVE.md"
else
    print_warning "architecture-overview.md not found, skipping"
fi

echo ""
echo "Step 3: Moving operations documents..."
echo "-------------------------------------------"

# Move troubleshooting handbook
if [ -f "docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md" ]; then
    mv docs/multi-region-demo/operations/TROUBLESHOOTING_HANDBOOK.md \
       docs/operations/multi-region/TROUBLESHOOTING.md
    print_success "Moved TROUBLESHOOTING_HANDBOOK.md → TROUBLESHOOTING.md"
else
    print_warning "TROUBLESHOOTING_HANDBOOK.md not found, skipping"
fi

# Move capacity planning guide
if [ -f "docs/multi-region-demo/operations/CAPACITY_PLANNING_GUIDE.md" ]; then
    mv docs/multi-region-demo/operations/CAPACITY_PLANNING_GUIDE.md \
       docs/operations/multi-region/CAPACITY_PLANNING.md
    print_success "Moved CAPACITY_PLANNING_GUIDE.md → CAPACITY_PLANNING.md"
else
    print_warning "CAPACITY_PLANNING_GUIDE.md not found, skipping"
fi

# Move performance tuning guide
if [ -f "docs/multi-region-demo/operations/PERFORMANCE_TUNING_GUIDE.md" ]; then
    mv docs/multi-region-demo/operations/PERFORMANCE_TUNING_GUIDE.md \
       docs/operations/multi-region/PERFORMANCE_TUNING.md
    print_success "Moved PERFORMANCE_TUNING_GUIDE.md → PERFORMANCE_TUNING.md"
else
    print_warning "PERFORMANCE_TUNING_GUIDE.md not found, skipping"
fi

echo ""
echo "Step 4: Moving blog articles..."
echo "-------------------------------------------"

# Move HLC implementation blog
if [ -f "docs/multi-region-demo/blog-hlc-implementation.md" ]; then
    mv docs/multi-region-demo/blog-hlc-implementation.md \
       .kiro/specs/multi-region-active-active/blog/hlc-implementation.md
    print_success "Moved blog-hlc-implementation.md → hlc-implementation.md"
else
    print_warning "blog-hlc-implementation.md not found, skipping"
fi

# Move conflict resolution blog
if [ -f "docs/multi-region-demo/blog-conflict-resolution.md" ]; then
    mv docs/multi-region-demo/blog-conflict-resolution.md \
       .kiro/specs/multi-region-active-active/blog/conflict-resolution.md
    print_success "Moved blog-conflict-resolution.md → conflict-resolution.md"
else
    print_warning "blog-conflict-resolution.md not found, skipping"
fi

# Move architecture decisions blog
if [ -f "docs/multi-region-demo/blog-architecture-decisions.md" ]; then
    mv docs/multi-region-demo/blog-architecture-decisions.md \
       .kiro/specs/multi-region-active-active/blog/architecture-decisions.md
    print_success "Moved blog-architecture-decisions.md → architecture-decisions.md"
else
    print_warning "blog-architecture-decisions.md not found, skipping"
fi

echo ""
echo "Step 5: Moving deployment documents..."
echo "-------------------------------------------"

# Move demo scenarios to deployment guide
if [ -f "docs/multi-region-demo/demo-scenarios.md" ]; then
    mv docs/multi-region-demo/demo-scenarios.md \
       docs/deployment/MULTI_REGION_DEPLOYMENT.md
    print_success "Moved demo-scenarios.md → MULTI_REGION_DEPLOYMENT.md"
else
    print_warning "demo-scenarios.md not found, skipping"
fi

echo ""
echo "Step 6: Moving README and summary documents..."
echo "-------------------------------------------"

# Move README to spec directory
if [ -f "docs/multi-region-demo/README.md" ]; then
    mv docs/multi-region-demo/README.md \
       .kiro/specs/multi-region-active-active/README.md
    print_success "Moved README.md to spec directory"
else
    print_warning "README.md not found, skipping"
fi

echo ""
echo "Step 7: Archiving remaining files..."
echo "-------------------------------------------"

# Archive monitoring dashboard (will be integrated later)
if [ -f "docs/multi-region-demo/monitoring-dashboard.md" ]; then
    mv docs/multi-region-demo/monitoring-dashboard.md \
       docs/archive/multi-region-monitoring-dashboard.md
    print_success "Archived monitoring-dashboard.md"
fi

# Archive quick reference (will be integrated later)
if [ -f "docs/multi-region-demo/QUICK_REFERENCE.md" ]; then
    mv docs/multi-region-demo/QUICK_REFERENCE.md \
       docs/archive/multi-region-quick-reference.md
    print_success "Archived QUICK_REFERENCE.md"
fi

# Archive demo package summary
if [ -f "docs/multi-region-demo/DEMO_PACKAGE_SUMMARY.md" ]; then
    mv docs/multi-region-demo/DEMO_PACKAGE_SUMMARY.md \
       docs/archive/multi-region-demo-package-summary.md
    print_success "Archived DEMO_PACKAGE_SUMMARY.md"
fi

echo ""
echo "Step 8: Cleaning up old directories..."
echo "-------------------------------------------"

# Remove empty operations directory
if [ -d "docs/multi-region-demo/operations" ]; then
    rmdir docs/multi-region-demo/operations 2>/dev/null || print_warning "operations directory not empty"
fi

# Remove multi-region-demo directory if empty
if [ -d "docs/multi-region-demo" ]; then
    rmdir docs/multi-region-demo 2>/dev/null && print_success "Removed empty docs/multi-region-demo/" || \
        print_warning "docs/multi-region-demo/ not empty, keeping it"
fi

echo ""
echo "========================================="
echo "Reorganization Complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Review the moved files"
echo "2. Create new index documents (README.md files)"
echo "3. Update document links"
echo "4. Run: git status to see changes"
echo "5. Run: git add . && git commit -m 'docs: reorganize multi-region documentation'"
echo ""
print_success "All done! Check docs/MULTI_REGION_DOCS_REORGANIZATION.md for details"
