#!/bin/bash
# CI test script that matches GitHub Actions workflow exactly
set -euo pipefail

echo "🧪 Running CI Test Suite..."

# Download dependencies first (like GHA)
echo "📦 Downloading dependencies..."
go mod download

# Install golangci-lint if not present
if ! command -v golangci-lint &> /dev/null; then
    echo "⚙️  Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

# Run linting with timeout (like GHA)
echo "🔍 Running golangci-lint..."
golangci-lint run --timeout=5m ./...

if [ $? -ne 0 ]; then
    echo "❌ Linting failed!"
    exit 1
fi

# Run unit tests with coverage
echo "🧪 Running unit tests..."
mkdir -p coverage
go test -race -cover -coverprofile=coverage/coverage.out ./internal/...

if [ $? -ne 0 ]; then
    echo "❌ Tests failed!"
    exit 1
fi

# Show coverage summary (like drun test recipe)
echo "📊 Coverage Summary:"
go tool cover -func=coverage/coverage.out | tail -1

# Check build with version info (like GHA)
echo "🔨 Testing build..."
go build -ldflags "-X main.version=ci-build -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/drun ./cmd/drun

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# Test basic functionality
echo "🧪 Testing binary..."
./bin/drun --version

if [ $? -ne 0 ]; then
    echo "❌ Binary test failed!"
    exit 1
fi

# Clean up
rm -f bin/drun
rm -rf coverage/

echo "✅ CI tests passed!"
