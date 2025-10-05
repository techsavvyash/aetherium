#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$SCRIPT_DIR/.."

echo "╔════════════════════════════════════════════════════════╗"
echo "║  Firecracker Complete Setup & Test                    ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (sudo)"
    exit 1
fi

# Get the actual user (not root when using sudo)
ACTUAL_USER=${SUDO_USER:-$USER}
ACTUAL_HOME=$(getent passwd "$ACTUAL_USER" | cut -d: -f6)

echo "Running as: $ACTUAL_USER"
echo "Project dir: $PROJECT_DIR"
echo ""

# Step 1: Download kernel with vsock
echo "Step 1: Downloading kernel with vsock support..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

KERNEL_DIR="/var/firecracker"
KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.13/x86_64/vmlinux-5.10.239"

mkdir -p "$KERNEL_DIR"

# Backup old kernel if exists
if [ -f "$KERNEL_DIR/vmlinux" ]; then
    echo "Backing up current kernel..."
    cp "$KERNEL_DIR/vmlinux" "$KERNEL_DIR/vmlinux.backup-$(date +%s)"
fi

echo "Downloading: $KERNEL_URL"
wget -q --show-progress -O "$KERNEL_DIR/vmlinux" "$KERNEL_URL"

if [ -f "$KERNEL_DIR/vmlinux" ]; then
    echo "✓ Kernel downloaded ($(ls -lh $KERNEL_DIR/vmlinux | awk '{print $5}'))"
else
    echo "❌ Kernel download failed"
    exit 1
fi
echo ""

# Step 2: Load vsock module
echo "Step 2: Loading vhost-vsock kernel module..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if lsmod | grep -q vhost_vsock; then
    echo "✓ vhost-vsock already loaded"
else
    modprobe vhost-vsock
    echo "✓ vhost-vsock loaded"
fi

if [ -c /dev/vhost-vsock ]; then
    chmod 666 /dev/vhost-vsock
    echo "✓ /dev/vhost-vsock permissions set"
else
    echo "❌ /dev/vhost-vsock not found"
    exit 1
fi
echo ""

# Step 3: Build agent
echo "Step 3: Building fc-agent..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cd "$PROJECT_DIR"

# Find go binary from user's environment
GO_BIN=$(sudo -u "$ACTUAL_USER" bash -l -c 'which go 2>/dev/null' || true)

if [ -z "$GO_BIN" ]; then
    echo "❌ Go not found in user's PATH"
    echo "Tried: sudo -u $ACTUAL_USER bash -l -c 'which go'"
    echo ""
    echo "Debug info:"
    sudo -u "$ACTUAL_USER" bash -l -c 'echo "PATH: $PATH"' || true
    exit 1
fi

echo "Using Go: $GO_BIN"

sudo -u "$ACTUAL_USER" bash -l -c "cd $PROJECT_DIR && $GO_BIN build -o bin/fc-agent ./cmd/fc-agent"
sudo -u "$ACTUAL_USER" bash -l -c "cd $PROJECT_DIR && $GO_BIN build -o bin/fc-test ./cmd/fc-test"
echo "✓ Binaries built"
echo ""

# Step 4: Deploy agent to rootfs
echo "Step 4: Deploying agent to rootfs..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

ROOTFS_PATH="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/tmp/fc-rootfs-mount"

if [ ! -f "$ROOTFS_PATH" ]; then
    echo "❌ Rootfs not found: $ROOTFS_PATH"
    echo "Run: sudo ./scripts/install-firecracker.sh first"
    exit 1
fi

mkdir -p "$MOUNT_POINT"
mount -o loop "$ROOTFS_PATH" "$MOUNT_POINT"

# Cleanup trap
trap "umount $MOUNT_POINT 2>/dev/null || true; rmdir $MOUNT_POINT 2>/dev/null || true" EXIT

# Copy agent
cp "$PROJECT_DIR/bin/fc-agent" "$MOUNT_POINT/usr/local/bin/fc-agent"
chmod +x "$MOUNT_POINT/usr/local/bin/fc-agent"
echo "✓ Agent binary copied"

# Create systemd service
cat > "$MOUNT_POINT/etc/systemd/system/fc-agent.service" <<'EOF'
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

mkdir -p "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants"
ln -sf /etc/systemd/system/fc-agent.service \
    "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/fc-agent.service"

echo "✓ Systemd service configured"

# Unmount
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"
echo ""

# Step 5: Clean up old sockets
echo "Step 5: Cleaning up..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f /tmp/firecracker-test-vm.sock*
echo "✓ Old sockets removed"
echo ""

echo "╔════════════════════════════════════════════════════════╗"
echo "║  ✓ Setup Complete!                                     ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""
echo "Now testing as user $ACTUAL_USER..."
echo ""

# Step 6: Run test as actual user
cd "$PROJECT_DIR"
sudo -u "$ACTUAL_USER" ./bin/fc-test

exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo ""
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║  ✓✓✓ SUCCESS! Everything works! ✓✓✓                   ║"
    echo "╚════════════════════════════════════════════════════════╝"
else
    echo ""
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║  ✗ Test failed - see output above                     ║"
    echo "╚════════════════════════════════════════════════════════╝"
fi

exit $exit_code
