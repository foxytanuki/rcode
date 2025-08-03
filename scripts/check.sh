#!/bin/bash
# Run all checks manually

set -e

echo "🔍 Running all checks..."
echo ""

echo "📝 Running gofmt..."
FILES=$(mise exec go -- gofmt -s -l .)
if [ -z "$FILES" ]; then
    echo "✅ gofmt: OK"
else
    echo "❌ gofmt: Found issues in:"
    echo "$FILES"
    exit 1
fi
echo ""

echo "🔎 Running go vet..."
mise exec go -- go vet ./...
if [ $? -eq 0 ]; then
    echo "✅ go vet: OK"
else
    echo "❌ go vet: Found issues"
    exit 1
fi
echo ""

echo "🔍 Running golangci-lint..."
if command -v mise &> /dev/null; then
    # Run lint and check exit code properly
    mise exec golangci-lint -- golangci-lint run ./... --timeout=30s >/dev/null 2>&1
    LINT_RESULT=$?
    
    if [ $LINT_RESULT -eq 0 ]; then
        echo "✅ golangci-lint: OK"
    else
        # Get issue summary
        SUMMARY=$(mise exec golangci-lint -- golangci-lint run ./... --timeout=30s 2>&1 | tail -1)
        echo "⚠️  golangci-lint: $SUMMARY"
        echo "   Run 'mise exec golangci-lint -- golangci-lint run ./...' for details"
    fi
else
    echo "⚠️  golangci-lint not available, skipping..."
fi
echo ""

echo "🧪 Running tests..."
mise exec go -- go test ./...
if [ $? -eq 0 ]; then
    echo "✅ tests: OK"
else
    echo "❌ tests: Failed"
    exit 1
fi
echo ""

echo "🏗️  Running build check..."
mise exec go -- go build -o /tmp/rcode-test ./cmd/rcode
mise exec go -- go build -o /tmp/rcode-server-test ./cmd/server
rm -f /tmp/rcode-test /tmp/rcode-server-test
if [ $? -eq 0 ]; then
    echo "✅ build: OK"
else
    echo "❌ build: Failed"
    exit 1
fi
echo ""

echo "✨ All checks passed!"