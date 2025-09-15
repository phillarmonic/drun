#!/bin/bash
# Quick test script for CI environments
set -euo pipefail

echo "🧪 Running CI Test Suite..."

# Install golangci-lint if not present
if ! command -v golangci-lint &> /dev/null; then
    echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

# Run linting first
echo "Running golangci-lint..."
golangci-lint run ./...

# Run unit tests with coverage
echo "Running unit tests..."
go test -race -cover -coverprofile=coverage.out ./internal/...

# Check build
echo "Testing build..."
go build -o bin/drun ./cmd/drun

# Test basic functionality
echo "Testing binary..."
./bin/drun --version

# Clean up
rm -f bin/drun coverage.out

echo "✅ CI tests passed!"
