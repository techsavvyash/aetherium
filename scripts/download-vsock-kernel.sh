#!/bin/bash
set -e

echo "╔════════════════════════════════════════════════════════╗"
echo "║  Downloading Firecracker Kernel with Vsock Support    ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

KERNEL_DIR="/var/firecracker"
# Use 5.10 kernel for better compatibility with Ubuntu 22.04 rootfs
KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.13/x86_64/vmlinux-5.10.239"
KERNEL_PATH="$KERNEL_DIR/vmlinux-vsock"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ This script must be run as root (sudo)"
    exit 1
fi

# Create directory if it doesn't exist
mkdir -p "$KERNEL_DIR"

echo "Downloading kernel with vsock support..."
echo "URL: $KERNEL_URL"
echo "Destination: $KERNEL_PATH"
echo ""

# Download kernel
wget -O "$KERNEL_PATH" "$KERNEL_URL"

if [ -f "$KERNEL_PATH" ]; then
    echo "✓ Kernel downloaded successfully"
    ls -lh "$KERNEL_PATH"

    # Make a backup of the old kernel
    if [ -f "$KERNEL_DIR/vmlinux" ]; then
        echo ""
        echo "Backing up old kernel to vmlinux.old..."
        cp "$KERNEL_DIR/vmlinux" "$KERNEL_DIR/vmlinux.old"
    fi

    # Replace the current kernel
    echo "Replacing current kernel with vsock-enabled version..."
    cp "$KERNEL_PATH" "$KERNEL_DIR/vmlinux"

    echo ""
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║  ✓ Kernel with vsock support installed!               ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo ""
    echo "The new kernel has CONFIG_VIRTIO_VSOCK=y compiled in."
    echo ""
    echo "Next steps:"
    echo "1. Run: sudo ./scripts/complete-vsock-test.sh"
    echo "2. Test: ./bin/fc-test"
else
    echo "❌ Download failed"
    exit 1
fi
