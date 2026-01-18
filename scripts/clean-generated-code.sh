#!/bin/bash

# Script to remove generated proto code from git
# This is a one-time migration script to implement the new unified proto generation strategy

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo "🧹 Cleaning generated proto code from git..."
echo ""

# Remove Go generated code
if [ -d "apps/todo-service/gen" ]; then
    echo "📦 Removing Go generated code: apps/todo-service/gen/"
    git rm -r apps/todo-service/gen/ 2>/dev/null || true
fi

# Remove TypeScript generated code
if [ -d "apps/web/src/gen" ]; then
    echo "📦 Removing TypeScript generated code: apps/web/src/gen/"
    git rm -r apps/web/src/gen/ 2>/dev/null || true
fi

# Remove Java generated code (if any)
if [ -d "apps/hello-service/build/generated" ]; then
    echo "📦 Removing Java generated code: apps/hello-service/build/generated/"
    git rm -r apps/hello-service/build/generated/ 2>/dev/null || true
fi

echo ""
echo "✅ Generated code removed from git"
echo ""
echo "📝 Next steps:"
echo "   1. Commit these changes: git commit -m 'chore: remove generated proto code from git'"
echo "   2. Run 'make proto' to regenerate code locally"
echo "   3. Verify builds work: make build"
echo "   4. Push changes to trigger CI"
echo ""
echo "ℹ️  Generated code will now be created during build time (not committed to git)"
echo "   See docs/PROTO_GENERATION_STRATEGY.md for details"
