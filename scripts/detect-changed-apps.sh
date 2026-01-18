#!/bin/bash

# Script to detect which apps have changed based on git diff
# Usage: ./scripts/detect-changed-apps.sh [base-branch]
# Returns: Space-separated list of changed app names

set -e

# Default base branch
BASE_BRANCH="${1:-main}"

# Get the list of changed files
if git rev-parse --verify HEAD >/dev/null 2>&1; then
    # If we're in a git repo with commits
    if git rev-parse --verify "$BASE_BRANCH" >/dev/null 2>&1; then
        # Compare with base branch
        CHANGED_FILES=$(git diff --name-only "$BASE_BRANCH"...HEAD 2>/dev/null || git diff --name-only HEAD)
    else
        # No base branch, use staged + unstaged changes
        CHANGED_FILES=$(git diff --name-only HEAD 2>/dev/null || echo "")
        if [ -z "$CHANGED_FILES" ]; then
            CHANGED_FILES=$(git ls-files)
        fi
    fi
else
    # Not a git repo or no commits yet
    echo "hello-service todo-service web"
    exit 0
fi

# Extract unique app names from changed files
CHANGED_APPS=""

# Check each app directory
for app_dir in apps/*/; do
    if [ -d "$app_dir" ]; then
        app_name=$(basename "$app_dir")
        
        # Check if any changed files are in this app directory
        if echo "$CHANGED_FILES" | grep -q "^apps/$app_name/"; then
            CHANGED_APPS="$CHANGED_APPS $app_name"
        fi
    fi
done

# Check if API changes affect all services
if echo "$CHANGED_FILES" | grep -q "^api/"; then
    # API changes affect all backend services
    CHANGED_APPS="$CHANGED_APPS hello-service todo-service"
fi

# Check if shared libs change
if echo "$CHANGED_FILES" | grep -q "^libs/"; then
    # Lib changes might affect all apps
    CHANGED_APPS="$CHANGED_APPS hello-service todo-service web"
fi

# Remove duplicates and trim
CHANGED_APPS=$(echo "$CHANGED_APPS" | tr ' ' '\n' | sort -u | tr '\n' ' ' | xargs)

# If no apps changed, return all apps (for safety)
if [ -z "$CHANGED_APPS" ]; then
    echo -n "hello-service todo-service web"
else
    echo -n "$CHANGED_APPS"
fi
