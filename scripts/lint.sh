#!/bin/bash
# Run linter separately

echo "Running golangci-lint..."
mise exec golangci-lint -- golangci-lint run ./... --timeout=30s