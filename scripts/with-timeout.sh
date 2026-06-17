#!/bin/bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <seconds> <command> [args...]" >&2
    exit 2
fi

seconds="$1"
shift

if command -v timeout >/dev/null 2>&1; then
    exec timeout "$seconds" "$@"
fi

if command -v gtimeout >/dev/null 2>&1; then
    exec gtimeout "$seconds" "$@"
fi

if command -v python3 >/dev/null 2>&1; then
    exec python3 - "$seconds" "$@" <<'PY'
import subprocess
import sys

timeout_seconds = float(sys.argv[1])
command = sys.argv[2:]

try:
    completed = subprocess.run(command, timeout=timeout_seconds)
    raise SystemExit(completed.returncode)
except subprocess.TimeoutExpired:
    raise SystemExit(124)
PY
fi

echo "No timeout implementation available; install timeout, gtimeout, or python3." >&2
exit 127
