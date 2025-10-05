#!/bin/bash
# Create a pool of TAP devices for Aetherium VMs
# This allows the worker to run without sudo by using pre-created TAP devices

set -e

# Configuration
BRIDGE_NAME="aetherium0"
TAP_PREFIX="aether-tap"
NUM_TAPS=10

echo "=== Creating TAP device pool for Aetherium ===\"
echo "Bridge: $BRIDGE_NAME"
echo "Number of TAP devices: $NUM_TAPS"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run with sudo"
    echo "Usage: sudo ./scripts/create-tap-pool.sh"
    exit 1
fi

# Check if bridge exists
if ! ip link show "$BRIDGE_NAME" &>/dev/null; then
    echo "Error: Bridge $BRIDGE_NAME does not exist"
    echo "Please run: sudo ./scripts/setup-network.sh"
    exit 1
fi

# Create TAP devices
for i in $(seq 0 $((NUM_TAPS - 1))); do
    TAP_NAME="${TAP_PREFIX}${i}"

    # Check if TAP already exists
    if ip link show "$TAP_NAME" &>/dev/null; then
        echo "✓ TAP device $TAP_NAME already exists"
        continue
    fi

    # Create TAP device
    echo "Creating TAP device: $TAP_NAME"
    ip tuntap add "$TAP_NAME" mode tap

    # Attach to bridge
    ip link set "$TAP_NAME" master "$BRIDGE_NAME"

    # Bring up
    ip link set "$TAP_NAME" up

    # Set permissions to allow non-root access
    chmod 666 "/sys/class/net/$TAP_NAME/tun_flags" 2>/dev/null || true

    echo "✓ Created and configured $TAP_NAME"
done

echo ""
echo "=== TAP device pool created successfully ===\"
echo "Created $NUM_TAPS TAP devices: ${TAP_PREFIX}0 to ${TAP_PREFIX}$((NUM_TAPS - 1))"
echo ""
echo "The worker can now use these TAP devices without requiring sudo."
