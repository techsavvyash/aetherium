#!/bin/bash
set -e

echo "╔════════════════════════════════════════╗"
echo "║  Firecracker Vsock Test Script        ║"
echo "╚════════════════════════════════════════╝"
echo ""

# Check if running as root for modprobe
if [ "$EUID" -ne 0 ]; then
    echo "This script needs sudo to load kernel modules"
    echo "Run: sudo $0"
    exit 1
fi

echo "1. Checking vsock kernel support..."
if lsmod | grep -q vhost_vsock; then
    echo "   ✓ vhost-vsock module already loaded"
else
    echo "   Loading vhost-vsock module..."
    modprobe vhost-vsock
    if lsmod | grep -q vhost_vsock; then
        echo "   ✓ vhost-vsock module loaded successfully"
    else
        echo "   ✗ Failed to load vhost-vsock module"
        exit 1
    fi
fi

echo ""
echo "2. Checking /dev/vhost-vsock..."
if [ -c /dev/vhost-vsock ]; then
    echo "   ✓ /dev/vhost-vsock exists"
    ls -l /dev/vhost-vsock
else
    echo "   ✗ /dev/vhost-vsock not found"
    echo "   Your kernel may not support vhost-vsock"
    exit 1
fi

echo ""
echo "3. Setting permissions on /dev/vhost-vsock..."
chmod 666 /dev/vhost-vsock
echo "   ✓ Permissions set (world-readable/writable for testing)"

echo ""
echo "4. Cleaning up old sockets..."
rm -f /tmp/firecracker-test-vm.sock*
echo "   ✓ Cleaned up"

echo ""
echo "╔════════════════════════════════════════╗"
echo "║  ✓ Vsock setup complete!               ║"
echo "╚════════════════════════════════════════╝"
echo ""
echo "You can now run the test as a regular user:"
echo "  ./bin/fc-test"
echo ""
