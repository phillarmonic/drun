#!/bin/bash
# Build script for drun with version information

set -euo pipefail

# Get version information
VERSION=${VERSION:-"2-dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

echo "Building drun..."
echo "Version: ${VERSION}"
echo "Commit: ${COMMIT}"
echo "Date: ${DATE}"
echo

go build -ldflags "${LDFLAGS}" -o bin/drun-cli ./cmd/drun

echo "✅ Build complete: bin/drun-cli"
echo "Test with: ./bin/drun-cli --version"
echo "Installing to /usr/local/bin/drun-cli"
sudo cp bin/drun-cli /usr/local/bin/drun-cli
echo "✅ Installed to /usr/local/bin/drun-cli"
/usr/local/bin/drun-cli --version