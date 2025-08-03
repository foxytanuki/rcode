#!/bin/bash
# Generate lint report by category

echo "📊 Lint Report"
echo "=============="
echo ""

# Count by linter
echo "Issues by Linter:"
mise exec golangci-lint -- golangci-lint run ./... --timeout=30s 2>&1 | \
    grep -E "^\S+\.go:[0-9]+:[0-9]+:" | \
    sed -E 's/.*\(([^)]+)\)$/\1/' | \
    sort | uniq -c | sort -rn

echo ""
echo "Files with Most Issues:"
mise exec golangci-lint -- golangci-lint run ./... --timeout=30s 2>&1 | \
    grep -E "^\S+\.go:[0-9]+" | \
    cut -d: -f1 | \
    sort | uniq -c | sort -rn | head -10

echo ""
echo "Quick Fix Commands:"
echo "-------------------"
echo "# Auto-fix what's possible:"
echo "mise exec golangci-lint -- golangci-lint run --fix ./..."
echo ""
echo "# Check specific linter:"
echo "mise exec golangci-lint -- golangci-lint run --disable-all -E errcheck ./..."
echo "mise exec golangci-lint -- golangci-lint run --disable-all -E gosec ./..."
echo ""
echo "# Fix permissions globally:"
echo "find . -name '*.go' -exec sed -i 's/0755/0750/g; s/0644/0600/g' {} \;"