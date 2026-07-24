#!/usr/bin/env bash
set -euo pipefail

echo "🔒 Running the v2 language compatibility contract..."
go test ./internal/compatibility
echo "✅ v2 language compatibility contract passed"
