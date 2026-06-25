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

go build -ldflags "${LDFLAGS}" -o bin/xdrun ./cmd/xdrun

echo "✅ Build complete: bin/xdrun"
echo "Test with: ./bin/xdrun --version"
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "${INSTALL_DIR}"
echo "Installing to ${INSTALL_DIR}/xdrun"
cp bin/xdrun "${INSTALL_DIR}/xdrun"
echo "✅ Installed to ${INSTALL_DIR}/xdrun"
"${INSTALL_DIR}/xdrun" --version
