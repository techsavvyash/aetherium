#!/bin/sh
# =============================================================================
# Aetherium Worker Entrypoint
# =============================================================================
# Performs pre-flight checks, registers with Consul, and starts the worker.
# =============================================================================

set -e

# Colors for logging
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"; }
error() { echo "${RED}[ERROR]${NC} $1" >&2; }
warn() { echo "${YELLOW}[WARN]${NC} $1"; }
success() { echo "${GREEN}[OK]${NC} $1"; }

# =============================================================================
# Pre-flight Checks
# =============================================================================
preflight_checks() {
    log "Running pre-flight checks..."
    FAILED=0

    # Check KVM
    if [ -e /dev/kvm ]; then
        success "KVM device available"
    else
        error "KVM device not found at /dev/kvm"
        error "Ensure node has KVM support and device is mounted"
        FAILED=1
    fi

    # Check vhost-vsock
    if [ -e /dev/vhost-vsock ]; then
        success "vhost-vsock device available"
    else
        warn "vhost-vsock device not found - VM communication may fail"
        warn "Run 'modprobe vhost_vsock' on host"
    fi

    # Check Firecracker binary
    if [ -x /usr/local/bin/firecracker ] || [ -x /var/firecracker/firecracker ]; then
        success "Firecracker binary found"
    else
        error "Firecracker binary not found"
        error "Expected at /usr/local/bin/firecracker or /var/firecracker/firecracker"
        FAILED=1
    fi

    # Check kernel
    if [ -f /var/firecracker/vmlinux ]; then
        success "Kernel image found at /var/firecracker/vmlinux"
    else
        error "Kernel image not found at /var/firecracker/vmlinux"
        FAILED=1
    fi

    # Check rootfs
    if [ -f /var/firecracker/rootfs.ext4 ]; then
        success "Root filesystem found at /var/firecracker/rootfs.ext4"
    else
        error "Root filesystem not found at /var/firecracker/rootfs.ext4"
        FAILED=1
    fi

    if [ $FAILED -eq 1 ]; then
        error "Pre-flight checks failed. Ensure node is properly configured."
        error "See docs/KUBERNETES.md for bare-metal node setup."
        exit 1
    fi

    success "All pre-flight checks passed"
}

# =============================================================================
# Network Setup
# =============================================================================
setup_network() {
    log "Setting up network..."

    # Check if bridge already exists
    if ip link show aetherium0 >/dev/null 2>&1; then
        log "Bridge aetherium0 already exists"
    else
        log "Creating bridge aetherium0..."
        ip link add name aetherium0 type bridge || true
        ip addr add 172.16.0.1/24 dev aetherium0 || true
        ip link set aetherium0 up || true
    fi

    # Enable IP forwarding
    echo 1 > /proc/sys/net/ipv4/ip_forward || true

    # Setup NAT for VM internet access
    iptables -t nat -C POSTROUTING -s 172.16.0.0/24 -o eth0 -j MASQUERADE 2>/dev/null || \
        iptables -t nat -A POSTROUTING -s 172.16.0.0/24 -o eth0 -j MASQUERADE || true

    success "Network setup complete"
}

# =============================================================================
# Consul Registration
# =============================================================================
register_with_consul() {
    if [ -z "$CONSUL_ADDR" ]; then
        log "CONSUL_ADDR not set, skipping Consul registration"
        return
    fi

    log "Registering with Consul at $CONSUL_ADDR..."

    # Get node info
    NODE_NAME="${NODE_NAME:-$(hostname)}"
    POD_IP="${POD_IP:-$(hostname -i)}"
    WORKER_PORT="${WORKER_PORT:-8081}"

    # Build service registration JSON
    SERVICE_JSON=$(cat <<EOF
{
    "ID": "aetherium-worker-${NODE_NAME}",
    "Name": "aetherium-worker",
    "Tags": ["worker", "firecracker", "kvm"],
    "Address": "${POD_IP}",
    "Port": ${WORKER_PORT},
    "Meta": {
        "node": "${NODE_NAME}",
        "version": "${VERSION:-unknown}",
        "kvm_enabled": "true"
    },
    "Check": {
        "HTTP": "http://${POD_IP}:${WORKER_PORT}/health",
        "Interval": "10s",
        "Timeout": "5s",
        "DeregisterCriticalServiceAfter": "1m"
    }
}
EOF
)

    # Register with Consul
    RETRIES=5
    for i in $(seq 1 $RETRIES); do
        if curl -sf -X PUT \
            -H "Content-Type: application/json" \
            -d "$SERVICE_JSON" \
            "http://${CONSUL_ADDR}/v1/agent/service/register"; then
            success "Registered with Consul as aetherium-worker-${NODE_NAME}"
            return
        fi
        warn "Consul registration attempt $i/$RETRIES failed, retrying..."
        sleep 2
    done

    warn "Failed to register with Consul after $RETRIES attempts"
    warn "Worker will start but may not be discoverable"
}

# =============================================================================
# Consul Deregistration (on shutdown)
# =============================================================================
deregister_from_consul() {
    if [ -z "$CONSUL_ADDR" ]; then
        return
    fi

    log "Deregistering from Consul..."
    NODE_NAME="${NODE_NAME:-$(hostname)}"

    curl -sf -X PUT \
        "http://${CONSUL_ADDR}/v1/agent/service/deregister/aetherium-worker-${NODE_NAME}" || true
}

# =============================================================================
# Signal Handlers
# =============================================================================
cleanup() {
    log "Received shutdown signal..."
    deregister_from_consul
    exit 0
}

trap cleanup SIGTERM SIGINT

# =============================================================================
# Main
# =============================================================================
main() {
    log "=============================================="
    log "  Aetherium Worker Starting"
    log "=============================================="
    log "Node: ${NODE_NAME:-$(hostname)}"
    log "Config: ${CONFIG_PATH:-/etc/aetherium/config.yaml}"
    log "Consul: ${CONSUL_ADDR:-disabled}"
    log "=============================================="

    preflight_checks
    setup_network
    register_with_consul

    log "Starting worker process..."
    exec "$@"
}

main "$@"
