#!/bin/bash

# Add build tags to all property test files
# This allows us to skip slow property-based tests during normal testing

set -e

echo "Adding build tags to property test files..."

# Find all *_property_test.go files
find apps -name "*_property_test.go" | while read -r file; do
    # Check if file already has the build tag
    if ! grep -q "//go:build property" "$file"; then
        echo "Processing: $file"
        
        # Create temp file with build tags
        {
            echo "//go:build property"
            echo "// +build property"
            echo ""
            cat "$file"
        } > "$file.tmp"
        
        # Replace original file
        mv "$file.tmp" "$file"
        
        echo "  ✓ Added build tags"
    else
        echo "Skipping: $file (already has build tags)"
    fi
done

echo "Done! Property test files now require -tags=property to run."
