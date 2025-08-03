#!/bin/bash
# Helper script to fix common lint issues

set -e

echo "🔧 Lint Fixer Tool"
echo ""

# Function to fix errcheck in a file
fix_errcheck() {
    local file=$1
    echo "Fixing errcheck in $file..."
    
    # Create backup
    cp "$file" "$file.bak"
    
    # Fix defer Close() patterns
    sed -i 's/defer \(.*\)\.Close()/defer func() { _ = \1.Close() }()/g' "$file"
    
    echo "✅ Fixed defer Close() patterns"
}

# Function to fix file permissions
fix_permissions() {
    local file=$1
    echo "Fixing file permissions in $file..."
    
    # Fix directory permissions (0755 -> 0750)
    sed -i 's/os\.MkdirAll(\(.*\), 0755)/os.MkdirAll(\1, 0750)/g' "$file"
    
    # Fix file permissions (0644 -> 0600)
    sed -i 's/os\.OpenFile(\(.*\), \(.*\), 0644)/os.OpenFile(\1, \2, 0600)/g' "$file"
    sed -i 's/os\.WriteFile(\(.*\), \(.*\), 0644)/os.WriteFile(\1, \2, 0600)/g' "$file"
    
    echo "✅ Fixed file permissions"
}

# Function to show manual fixes needed
show_manual_fixes() {
    echo ""
    echo "📝 Manual fixes needed:"
    echo ""
    echo "1. Context keys - Replace string keys with typed keys:"
    echo "   type contextKey string"
    echo "   const traceIDKey contextKey = \"trace_id\""
    echo ""
    echo "2. Unused parameters - Add _ prefix:"
    echo "   func handler(w http.ResponseWriter, _ *http.Request)"
    echo ""
    echo "3. Package comments - Add at top of file:"
    echo "   // Package logger provides structured logging"
    echo "   package logger"
    echo ""
}

# Main menu
case "${1:-menu}" in
    errcheck)
        fix_errcheck "$2"
        ;;
    permissions)
        fix_permissions "$2"
        ;;
    check)
        echo "Checking $2..."
        mise exec golangci-lint -- golangci-lint run "$2"
        ;;
    all)
        echo "🚀 Running all automatic fixes..."
        
        # Auto-fix with golangci-lint
        mise exec golangci-lint -- golangci-lint run --fix ./...
        
        # Fix permissions in all files
        find . -name "*.go" -type f | while read -r file; do
            fix_permissions "$file"
        done
        
        echo "✅ Automatic fixes completed"
        show_manual_fixes
        ;;
    *)
        echo "Usage:"
        echo "  ./scripts/fix-lint.sh errcheck <file>     - Fix errcheck issues"
        echo "  ./scripts/fix-lint.sh permissions <file>  - Fix file permissions"
        echo "  ./scripts/fix-lint.sh check <file>        - Check specific file"
        echo "  ./scripts/fix-lint.sh all                 - Run all automatic fixes"
        echo ""
        echo "Example:"
        echo "  ./scripts/fix-lint.sh errcheck internal/logger/file.go"
        ;;
esac