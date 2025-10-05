#!/bin/bash
# Start Aetherium Worker with network capabilities
#
# Usage: sudo ./scripts/start-worker.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKER_BIN="$PROJECT_ROOT/bin/worker"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run with sudo"
    echo "Usage: sudo ./scripts/start-worker.sh"
    exit 1
fi

# Check if worker binary exists
if [ ! -f "$WORKER_BIN" ]; then
    echo "Error: Worker binary not found at $WORKER_BIN"
    echo "Please run: go build -o bin/worker cmd/worker/main.go"
    exit 1
fi

echo "=== Starting Aetherium Worker with network capabilities ==="
echo "Worker binary: $WORKER_BIN"
echo ""

# Run worker
cd "$PROJECT_ROOT"
exec "$WORKER_BIN"
