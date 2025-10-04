#!/bin/bash
set -e

echo "╔════════════════════════════════════════╗"
echo "║  Firecracker Agent Setup Script       ║"
echo "╚════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root (use sudo)"
    exit 1
fi

ROOTFS_PATH="${ROOTFS_PATH:-/var/firecracker/rootfs.ext4}"
AGENT_BINARY="${AGENT_BINARY:-./bin/fc-agent}"
MOUNT_POINT="/tmp/fc-rootfs-mount"

echo "Configuration:"
echo "  Rootfs: $ROOTFS_PATH"
echo "  Agent:  $AGENT_BINARY"
echo "  Mount:  $MOUNT_POINT"
echo ""

# Check if agent binary exists
if [ ! -f "$AGENT_BINARY" ]; then
    echo "✗ Agent binary not found: $AGENT_BINARY"
    echo ""
    echo "Please build it first:"
    echo "  go build -o bin/fc-agent ./cmd/fc-agent"
    exit 1
fi

# Check if rootfs exists
if [ ! -f "$ROOTFS_PATH" ]; then
    echo "✗ Rootfs not found: $ROOTFS_PATH"
    echo ""
    echo "Please run the installation script first:"
    echo "  ./scripts/install-firecracker.sh"
    exit 1
fi

echo "1. Creating mount point..."
mkdir -p "$MOUNT_POINT"

echo "2. Mounting rootfs..."
mount -o loop "$ROOTFS_PATH" "$MOUNT_POINT"

# Ensure cleanup on exit
trap "umount $MOUNT_POINT 2>/dev/null || true; rmdir $MOUNT_POINT 2>/dev/null || true" EXIT

echo "3. Copying agent binary..."
cp "$AGENT_BINARY" "$MOUNT_POINT/usr/local/bin/fc-agent"
chmod +x "$MOUNT_POINT/usr/local/bin/fc-agent"

echo "4. Creating systemd service..."
cat > "$MOUNT_POINT/etc/systemd/system/fc-agent.service" <<EOF
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
EOF

echo "5. Enabling agent service..."
# Create symlink to enable service
mkdir -p "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants"
ln -sf /etc/systemd/system/fc-agent.service \
    "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/fc-agent.service"

echo "6. Unmounting rootfs..."
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"

echo ""
echo "✓ Firecracker agent setup complete!"
echo ""
echo "The agent will automatically start when the VM boots."
echo "It listens on port 9999 for command execution requests."
echo ""
echo "Next steps:"
echo "  1. Start a Firecracker VM:"
echo "     ./bin/firecracker-demo"
echo ""
echo "  2. Or use the vm-cli:"
echo "     ./bin/vm-cli init firecracker"
echo "     ./bin/vm-cli create my-vm"
echo "     ./bin/vm-cli start my-vm"
echo "     ./bin/vm-cli exec my-vm echo 'Hello from Firecracker!'"
