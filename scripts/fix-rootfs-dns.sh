#!/bin/bash
# Fix DNS configuration in the Firecracker rootfs

set -e

ROOTFS_PATH="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/tmp/aetherium-rootfs-mount"

echo "=== Fixing DNS configuration in rootfs ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run with sudo"
    echo "Usage: sudo ./scripts/fix-rootfs-dns.sh"
    exit 1
fi

# Check if rootfs exists
if [ ! -f "$ROOTFS_PATH" ]; then
    echo "Error: Rootfs not found at $ROOTFS_PATH"
    exit 1
fi

# Create mount point
mkdir -p "$MOUNT_POINT"

# Mount the rootfs
echo "Mounting rootfs..."
mount -o loop "$ROOTFS_PATH" "$MOUNT_POINT"

# Configure DNS
echo "Configuring DNS servers..."
cat > "$MOUNT_POINT/etc/resolv.conf" << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF

echo "✓ DNS configuration added to /etc/resolv.conf"

# Verify
echo ""
echo "Current /etc/resolv.conf content:"
cat "$MOUNT_POINT/etc/resolv.conf"

# Unmount
echo ""
echo "Unmounting rootfs..."
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"

echo ""
echo "=== DNS configuration fixed successfully ==="
echo "The rootfs now has DNS servers configured."
echo "New VMs will be able to resolve domain names."
