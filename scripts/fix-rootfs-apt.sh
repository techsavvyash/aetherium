#!/bin/bash
# Fix apt/dpkg directories in the rootfs

set -e

ROOTFS_PATH="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/tmp/aetherium-rootfs-mount"

echo "=== Fixing apt/dpkg directories in rootfs ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run with sudo"
    echo "Usage: sudo ./scripts/fix-rootfs-apt.sh"
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

echo ""
echo "=== Creating missing apt/dpkg directories ===\"

# Create apt directories
mkdir -p "$MOUNT_POINT/var/lib/apt/lists/partial"
mkdir -p "$MOUNT_POINT/var/lib/apt/lists/auxfiles"
mkdir -p "$MOUNT_POINT/var/cache/apt/archives/partial"

# Create dpkg directories
mkdir -p "$MOUNT_POINT/var/lib/dpkg"
mkdir -p "$MOUNT_POINT/var/lib/dpkg/updates"
mkdir -p "$MOUNT_POINT/var/lib/dpkg/info"

# Create dpkg status file if missing
if [ ! -f "$MOUNT_POINT/var/lib/dpkg/status" ]; then
    touch "$MOUNT_POINT/var/lib/dpkg/status"
    echo "✓ Created /var/lib/dpkg/status"
fi

# Create dpkg available file if missing
if [ ! -f "$MOUNT_POINT/var/lib/dpkg/available" ]; then
    touch "$MOUNT_POINT/var/lib/dpkg/available"
    echo "✓ Created /var/lib/dpkg/available"
fi

# Set permissions
chmod 755 "$MOUNT_POINT/var/lib/apt/lists"
chmod 755 "$MOUNT_POINT/var/lib/apt/lists/partial"
chmod 755 "$MOUNT_POINT/var/cache/apt/archives"
chmod 755 "$MOUNT_POINT/var/cache/apt/archives/partial"
chmod 755 "$MOUNT_POINT/var/lib/dpkg"
chmod 644 "$MOUNT_POINT/var/lib/dpkg/status" 2>/dev/null || true
chmod 644 "$MOUNT_POINT/var/lib/dpkg/available" 2>/dev/null || true

echo "✓ Created all required apt/dpkg directories"

# Configure DNS
echo ""
echo "=== Configuring DNS ===\"
cat > "$MOUNT_POINT/etc/resolv.conf" << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF
echo "✓ Configured DNS servers"

# Verify
echo ""
echo "=== Verification ===\"
echo "Checking directories:"
for dir in /var/lib/apt/lists/partial /var/lib/dpkg /var/cache/apt/archives/partial; do
    if [ -d "$MOUNT_POINT$dir" ]; then
        echo "  ✓ $dir exists"
    else
        echo "  ✗ $dir MISSING"
    fi
done

echo ""
echo "Checking files:"
for file in /var/lib/dpkg/status /var/lib/dpkg/available /etc/resolv.conf; do
    if [ -f "$MOUNT_POINT$file" ]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file MISSING"
    fi
done

# Unmount
echo ""
echo "Unmounting rootfs..."
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"

echo ""
echo "=== Rootfs fixed successfully ===\"
echo ""
echo "The rootfs now has:"
echo "  ✓ Complete apt/dpkg directory structure"
echo "  ✓ DNS configuration"
echo ""
echo "New VMs will be able to use apt-get and install packages."
