#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════════"
echo "  Complete Firecracker Vsock Setup & Test"
echo "═══════════════════════════════════════════════════════"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (sudo)"
    exit 1
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTFS_PATH="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/tmp/fc-rootfs-mount"
AGENT_BINARY="${SCRIPT_DIR}/../bin/fc-agent"

echo "Step 1: Checking vsock kernel support..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if lsmod | grep -q vhost_vsock; then
    echo "✓ vhost-vsock module already loaded"
else
    echo "Loading vhost-vsock module..."
    modprobe vhost-vsock
    if lsmod | grep -q vhost_vsock; then
        echo "✓ vhost-vsock module loaded successfully"
    else
        echo "❌ Failed to load vhost-vsock module"
        echo "Your kernel may not have vsock support compiled in."
        exit 1
    fi
fi
echo ""

echo "Step 2: Checking /dev/vhost-vsock..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ -c /dev/vhost-vsock ]; then
    echo "✓ /dev/vhost-vsock exists"
    ls -l /dev/vhost-vsock
else
    echo "❌ /dev/vhost-vsock not found"
    echo "Your kernel may not support vhost-vsock"
    exit 1
fi
echo ""

echo "Step 3: Setting permissions on /dev/vhost-vsock..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
chmod 666 /dev/vhost-vsock
echo "✓ Permissions set (world-readable/writable for testing)"
echo ""

echo "Step 4: Deploying updated agent to rootfs..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if agent binary exists
if [ ! -f "$AGENT_BINARY" ]; then
    echo "❌ Agent binary not found: $AGENT_BINARY"
    echo "Run 'make go-build' first"
    exit 1
fi

# Create mount point
mkdir -p "$MOUNT_POINT"

# Mount rootfs
echo "Mounting rootfs..."
mount -o loop "$ROOTFS_PATH" "$MOUNT_POINT"

# Setup cleanup trap
trap "umount $MOUNT_POINT 2>/dev/null || true; rmdir $MOUNT_POINT 2>/dev/null || true" EXIT

# Copy agent
echo "Copying updated agent binary..."
cp "$AGENT_BINARY" "$MOUNT_POINT/usr/local/bin/fc-agent"
chmod +x "$MOUNT_POINT/usr/local/bin/fc-agent"

# Verify systemd service
if [ -f "$MOUNT_POINT/etc/systemd/system/fc-agent.service" ]; then
    echo "✓ Service file exists"
else
    echo "Creating systemd service..."
    cat > "$MOUNT_POINT/etc/systemd/system/fc-agent.service" <<'SVCEOF'
[Unit]
Description=Firecracker Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fc-agent
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SVCEOF

    mkdir -p "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants"
    ln -sf /etc/systemd/system/fc-agent.service \
        "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/fc-agent.service"
    echo "✓ Service created and enabled"
fi

# Unmount
echo "Unmounting rootfs..."
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"
echo "✓ Agent deployed successfully"
echo ""

echo "Step 5: Cleaning up old VM sockets..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f /tmp/firecracker-test-vm.sock*
echo "✓ Cleaned up"
echo ""

echo "═══════════════════════════════════════════════════════"
echo "  ✓ Setup Complete!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Now run the test as your regular user:"
echo "  ./bin/fc-test"
echo ""
