#!/bin/bash
# One-time rootfs setup for Aetherium VMs
# This script configures the rootfs template to automatically handle DNS and networking

set -e

ROOTFS_PATH="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/tmp/aetherium-rootfs-mount"

echo "=== One-time Rootfs Setup for Aetherium ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run with sudo"
    echo "Usage: sudo ./scripts/setup-rootfs-once.sh"
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
echo "=== Fixing apt/dpkg directories ===\"

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
else
    echo "✓ /var/lib/dpkg/status exists"
fi

# Create dpkg available file if missing
if [ ! -f "$MOUNT_POINT/var/lib/dpkg/available" ]; then
    touch "$MOUNT_POINT/var/lib/dpkg/available"
    echo "✓ Created /var/lib/dpkg/available"
else
    echo "✓ /var/lib/dpkg/available exists"
fi

# Set permissions
chmod 755 "$MOUNT_POINT/var/lib/apt/lists" 2>/dev/null || true
chmod 755 "$MOUNT_POINT/var/lib/apt/lists/partial" 2>/dev/null || true
chmod 755 "$MOUNT_POINT/var/cache/apt/archives" 2>/dev/null || true
chmod 755 "$MOUNT_POINT/var/cache/apt/archives/partial" 2>/dev/null || true
chmod 755 "$MOUNT_POINT/var/lib/dpkg" 2>/dev/null || true
chmod 644 "$MOUNT_POINT/var/lib/dpkg/status" 2>/dev/null || true
chmod 644 "$MOUNT_POINT/var/lib/dpkg/available" 2>/dev/null || true

echo "✓ Created all required apt/dpkg directories"

echo ""
echo "=== Configuring DNS ===\"

# Create resolv.conf with DNS servers
cat > "$MOUNT_POINT/etc/resolv.conf" << 'EOF'
# DNS configuration for Aetherium VMs
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF

echo "✓ Created /etc/resolv.conf with DNS servers"

echo ""
echo "=== Creating network configuration script ===\"

# Create a systemd service that ensures DNS is configured on boot
cat > "$MOUNT_POINT/etc/systemd/system/aetherium-network.service" << 'EOF'
[Unit]
Description=Aetherium Network Configuration
After=network-pre.target
Before=network.target
DefaultDependencies=no

[Service]
Type=oneshot
ExecStart=/usr/local/bin/aetherium-configure-network.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

# Create the configuration script
mkdir -p "$MOUNT_POINT/usr/local/bin"
cat > "$MOUNT_POINT/usr/local/bin/aetherium-configure-network.sh" << 'EOF'
#!/bin/bash
# Aetherium network configuration script
# This runs on every boot to ensure DNS is properly configured

# Parse kernel cmdline for DNS servers
DNS1="8.8.8.8"
DNS2="8.8.4.4"

# Extract DNS from kernel parameters if provided
CMDLINE=$(cat /proc/cmdline)
if [[ $CMDLINE =~ ip=[^:]*:[^:]*:[^:]*:[^:]*:[^:]*:[^:]*:[^:]*:([^:]+):([^ ]+) ]]; then
    DNS1="${BASH_REMATCH[1]}"
    DNS2="${BASH_REMATCH[2]}"
fi

# Write resolv.conf
cat > /etc/resolv.conf << RESOLV
# Configured by Aetherium
nameserver $DNS1
nameserver $DNS2
nameserver 1.1.1.1
RESOLV

echo "✓ DNS configured: $DNS1, $DNS2"
EOF

chmod +x "$MOUNT_POINT/usr/local/bin/aetherium-configure-network.sh"

# Enable the service
ln -sf /etc/systemd/system/aetherium-network.service \
    "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/aetherium-network.service" 2>/dev/null || true

echo "✓ Created network configuration service"

echo ""
echo "=== Verifying configuration ===\"
echo ""
echo "DNS configuration (/etc/resolv.conf):"
cat "$MOUNT_POINT/etc/resolv.conf"
echo ""
echo "Network service:"
ls -l "$MOUNT_POINT/etc/systemd/system/aetherium-network.service"

# Unmount
echo ""
echo "Unmounting rootfs..."
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"

echo ""
echo "=== Rootfs setup complete ===\"
echo ""
echo "The rootfs is now configured with:"
echo "  ✓ Complete apt/dpkg directory structure"
echo "  ✓ Automatic DNS configuration on boot"
echo "  ✓ DNS server parsing from kernel parameters"
echo "  ✓ Fallback DNS: 8.8.8.8, 8.8.4.4, 1.1.1.1"
echo ""
echo "All new VMs will automatically have:"
echo "  ✓ Working package manager (apt-get)"
echo "  ✓ DNS resolution"
echo "  ✓ Internet connectivity (via NAT)"
echo ""
echo "No manual intervention required for future VMs!"
