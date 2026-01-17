#!/bin/bash

# Script to install Git hooks

set -e

echo "Installing Git hooks..."

# Get the repository root directory
REPO_ROOT=$(git rev-parse --show-toplevel)

# Create .git/hooks directory if it doesn't exist
mkdir -p "$REPO_ROOT/.git/hooks"

# Install pre-commit hook
if [ -f "$REPO_ROOT/.githooks/pre-commit" ]; then
    ln -sf "$REPO_ROOT/.githooks/pre-commit" "$REPO_ROOT/.git/hooks/pre-commit"
    chmod +x "$REPO_ROOT/.git/hooks/pre-commit"
    echo "✓ Pre-commit hook installed"
else
    echo "✗ Pre-commit hook not found at .githooks/pre-commit"
    exit 1
fi

# Configure Git to use .githooks directory (Git 2.9+)
if git config core.hooksPath .githooks 2>/dev/null; then
    echo "✓ Git configured to use .githooks directory"
else
    echo "⚠ Could not configure Git hooks path (Git 2.9+ required)"
    echo "  Hooks have been symlinked to .git/hooks/ instead"
fi

echo ""
echo "Git hooks installed successfully!"
echo ""
echo "To bypass hooks temporarily, use: git commit --no-verify"
