#!/bin/bash
set -euo pipefail

# Build script for local release testing
# Usage: ./scripts/build-release.sh [version]

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "🚀 Building drun release binaries"
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $DATE"
echo ""

# Create dist directory
mkdir -p dist

# Build matrix
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    # Set binary name
    if [ "$GOOS" = "windows" ]; then
        BINARY_NAME="xdrun-$GOOS-$GOARCH.exe"
        FILENAME="xdrun.exe"
    else
        BINARY_NAME="xdrun-$GOOS-$GOARCH"
        FILENAME="xdrun"
    fi
    
    echo "Building $BINARY_NAME..."
    
    # Build binary
    env GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
        -o "dist/$BINARY_NAME" \
        ./cmd/drun
    
    # Show file info before compression
    echo "  📦 Built: $(ls -lh dist/$BINARY_NAME | awk '{print $5}')"
    
    # Compress with UPX if available
    if command -v upx >/dev/null 2>&1; then
        echo "  🗜️  Compressing with UPX..."
        
        if [ "$GOOS" = "windows" ] && [ "$GOARCH" = "arm64" ]; then
            # Windows ARM64 - not supported by UPX yet
            echo "  ℹ️  Skipping UPX compression for Windows ARM64 (not supported by UPX)"
        elif [ "$GOOS" = "darwin" ]; then
            # macOS binaries - UPX may have issues, so allow failure
            if upx --best "dist/$BINARY_NAME" 2>/dev/null; then
                echo "  ✅ UPX compressed: $(ls -lh dist/$BINARY_NAME | awk '{print $5}')"
            else
                echo "  ⚠️  UPX compression failed for macOS binary (continuing without compression)"
            fi
        else
            # Linux and Windows x64 binaries - UPX should work
            if upx --best "dist/$BINARY_NAME" 2>/dev/null; then
                echo "  ✅ UPX compressed: $(ls -lh dist/$BINARY_NAME | awk '{print $5}')"
            else
                echo "  ⚠️  UPX compression failed for $GOOS $GOARCH binary (continuing without compression)"
            fi
        fi
    else
        echo "  ℹ️  UPX not found, skipping compression (install UPX for smaller binaries)"
    fi
done

echo ""
echo "🎉 All binaries built successfully!"
echo ""
echo "📦 Release artifacts:"
ls -la dist/

echo ""
echo "🧪 Testing local binary:"
if [[ "$OSTYPE" == "darwin"* ]]; then
    if [[ $(uname -m) == "arm64" ]]; then
        ./dist/xdrun-darwin-arm64 --version
    else
        ./dist/xdrun-darwin-amd64 --version
    fi
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if [[ $(uname -m) == "aarch64" ]]; then
        ./dist/xdrun-linux-arm64 --version
    else
        ./dist/xdrun-linux-amd64 --version
    fi
else
    echo "  ℹ️  Cannot test on this platform, but binaries are built"
fi

echo ""
echo "✨ Release build complete!"
