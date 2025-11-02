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

# Export environment variables for distributed mode
export POSTGRES_HOST=${POSTGRES_HOST:-localhost}
export POSTGRES_PORT=${POSTGRES_PORT:-5434}
export POSTGRES_USER=${POSTGRES_USER:-aetherium}
export POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-aetherium}
export POSTGRES_DB=${POSTGRES_DB:-aetherium}
export REDIS_ADDR=${REDIS_ADDR:-localhost:6380}
export CONSUL_ADDR=${CONSUL_ADDR:-localhost:8500}
export CONSUL_DATACENTER=${CONSUL_DATACENTER:-dc1}
export CONSUL_SCHEME=${CONSUL_SCHEME:-http}
export CONSUL_SERVICE_NAME=${CONSUL_SERVICE_NAME:-aetherium-worker}

# Worker configuration
export WORKER_ID=${WORKER_ID:-worker-$(hostname)-$$}
export WORKER_ZONE=${WORKER_ZONE:-us-west-1a}
export WORKER_ADDRESS=${WORKER_ADDRESS:-$(hostname):8081}
export WORKER_LABELS=${WORKER_LABELS:-env=dev,tier=compute}
export WORKER_CAPABILITY=${WORKER_CAPABILITY:-firecracker}
export WORKER_CPU_CORES=${WORKER_CPU_CORES:-$(nproc)}
export WORKER_MEMORY_MB=${WORKER_MEMORY_MB:-16384}
export WORKER_DISK_GB=${WORKER_DISK_GB:-500}
export WORKER_MAX_VMS=${WORKER_MAX_VMS:-10}
export HEARTBEAT_INTERVAL_SECONDS=${HEARTBEAT_INTERVAL_SECONDS:-10}

# Firecracker paths
export KERNEL_PATH=${KERNEL_PATH:-/var/firecracker/vmlinux}
export ROOTFS_TEMPLATE=${ROOTFS_TEMPLATE:-/var/firecracker/rootfs.ext4}
export SOCKET_DIR=${SOCKET_DIR:-/tmp}
export DEFAULT_VCPU=${DEFAULT_VCPU:-1}
export DEFAULT_MEMORY_MB=${DEFAULT_MEMORY_MB:-512}

echo "Worker Configuration:"
echo "  ID:      $WORKER_ID"
echo "  Zone:    $WORKER_ZONE"
echo "  Address: $WORKER_ADDRESS"
echo "  Labels:  $WORKER_LABELS"
echo "  CPU:     $WORKER_CPU_CORES cores"
echo "  Memory:  $WORKER_MEMORY_MB MB"
echo "  Max VMs: $WORKER_MAX_VMS"
echo "  Consul:  $CONSUL_ADDR"
echo ""

# Run worker
cd "$PROJECT_ROOT"
exec "$WORKER_BIN"
