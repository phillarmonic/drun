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

# Run race tests with timeout (10 minutes max)
echo "⏱️  Running race condition tests with 10-minute timeout..."

if ./scripts/with-timeout.sh 600 go test -race -cover -coverprofile=coverage/coverage.out ./internal/...; then
    echo "✅ Race condition tests completed successfully"
else
    exit_code=$?
    if [ $exit_code -eq 124 ]; then
        echo "❌ Race condition tests timed out after 10 minutes!"
        echo "This may indicate a deadlock or infinite loop in the code."
        exit 1
    else
        echo "❌ Race condition tests failed with exit code $exit_code"
        exit $exit_code
    fi
fi

# Show coverage summary (like xdrun test recipe)
echo "📊 Coverage Summary:"
go tool cover -func=coverage/coverage.out | tail -1

# Check build with version info (like GHA)
echo "🔨 Testing build..."
go build -ldflags "-X main.version=ci-build -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/xdrun ./cmd/xdrun

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

# Test basic functionality
echo "🧪 Testing binary..."
./bin/xdrun --version

if [ $? -ne 0 ]; then
    echo "❌ Binary test failed!"
    exit 1
fi

# Clean up
rm -f bin/xdrun
rm -rf coverage/

echo "✅ CI tests passed!"
