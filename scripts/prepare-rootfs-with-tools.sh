#!/bin/bash
set -e

echo "=== Aetherium Rootfs Preparation Script ==="
echo "This script prepares a rootfs image with essential tools pre-installed"
echo ""

# Configuration
ROOTFS_SIZE="${ROOTFS_SIZE:-2G}"
ROOTFS_PATH="${ROOTFS_PATH:-/var/firecracker/rootfs.ext4}"
MOUNT_POINT="/tmp/aetherium-rootfs-mount"
UBUNTU_VERSION="${UBUNTU_VERSION:-22.04}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

echo "Step 1: Creating rootfs image..."
if [ -f "$ROOTFS_PATH" ]; then
    echo "  Backing up existing rootfs to ${ROOTFS_PATH}.backup"
    mv "$ROOTFS_PATH" "${ROOTFS_PATH}.backup"
fi

# Create sparse file
dd if=/dev/zero of="$ROOTFS_PATH" bs=1 count=0 seek="$ROOTFS_SIZE"
mkfs.ext4 "$ROOTFS_PATH"
echo "✓ Rootfs image created: $ROOTFS_PATH"

echo ""
echo "Step 2: Mounting rootfs..."
mkdir -p "$MOUNT_POINT"
mount -o loop "$ROOTFS_PATH" "$MOUNT_POINT"
echo "✓ Mounted at $MOUNT_POINT"

# Ensure cleanup on exit
trap "umount $MOUNT_POINT 2>/dev/null || true; rmdir $MOUNT_POINT 2>/dev/null || true" EXIT

echo ""
echo "Step 3: Installing Ubuntu base system..."
debootstrap --arch=amd64 "$UBUNTU_VERSION" "$MOUNT_POINT" http://archive.ubuntu.com/ubuntu/
echo "✓ Ubuntu base system installed"

echo ""
echo "Step 4: Configuring system..."

# Setup chroot environment
mount --bind /dev "$MOUNT_POINT/dev"
mount --bind /proc "$MOUNT_POINT/proc"
mount --bind /sys "$MOUNT_POINT/sys"

# Create setup script to run in chroot
cat > "$MOUNT_POINT/tmp/setup.sh" << 'SETUP_SCRIPT'
#!/bin/bash
set -e

export DEBIAN_FRONTEND=noninteractive

# Configure DNS
echo "nameserver 8.8.8.8" > /etc/resolv.conf
echo "nameserver 8.8.4.4" >> /etc/resolv.conf

# Configure apt sources
cat > /etc/apt/sources.list << EOF
deb http://archive.ubuntu.com/ubuntu/ jammy main restricted universe multiverse
deb http://archive.ubuntu.com/ubuntu/ jammy-updates main restricted universe multiverse
deb http://archive.ubuntu.com/ubuntu/ jammy-security main restricted universe multiverse
EOF

# Update package lists
apt-get update

# Install essential packages
echo "Installing essential packages..."
apt-get install -y \
    systemd \
    systemd-sysv \
    ca-certificates \
    curl \
    wget \
    git \
    build-essential \
    sudo \
    openssh-server \
    nano \
    vim \
    net-tools \
    iputils-ping \
    dnsutils \
    unzip

# Install Node.js 20.x
echo "Installing Node.js..."
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt-get install -y nodejs

# Install Bun
echo "Installing Bun..."
curl -fsSL https://bun.sh/install | bash
export BUN_INSTALL="/root/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
echo 'export BUN_INSTALL="/root/.bun"' >> /root/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> /root/.bashrc

# Install Claude Code (requires npm)
echo "Installing Claude Code..."
npm install -g claude-code || echo "Warning: Claude Code installation may require specific configuration"

# Create systemd service for fc-agent
cat > /etc/systemd/system/fc-agent.service << EOF
[Unit]
Description=Firecracker Agent for Command Execution
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

# Enable fc-agent service
systemctl enable fc-agent.service

# Configure root password (change this in production!)
echo "root:aetherium" | chpasswd

# Configure SSH
sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
systemctl enable ssh

# Clean up
apt-get clean
rm -rf /var/lib/apt/lists/*

echo "✓ System configuration complete"
SETUP_SCRIPT

chmod +x "$MOUNT_POINT/tmp/setup.sh"

echo "Running setup in chroot..."
chroot "$MOUNT_POINT" /tmp/setup.sh

# Cleanup chroot mounts
umount "$MOUNT_POINT/dev" || true
umount "$MOUNT_POINT/proc" || true
umount "$MOUNT_POINT/sys" || true

echo ""
echo "Step 5: Copying fc-agent binary..."
if [ -f "bin/fc-agent" ]; then
    cp bin/fc-agent "$MOUNT_POINT/usr/local/bin/fc-agent"
    chmod +x "$MOUNT_POINT/usr/local/bin/fc-agent"
    echo "✓ fc-agent copied"
else
    echo "Warning: bin/fc-agent not found. Build it first with: go build -o bin/fc-agent ./cmd/fc-agent"
fi

echo ""
echo "Step 6: Creating fstab..."
cat > "$MOUNT_POINT/etc/fstab" << EOF
/dev/vda / ext4 defaults 0 1
EOF

echo ""
echo "Step 7: Final cleanup and unmounting..."
sync
umount "$MOUNT_POINT"
rmdir "$MOUNT_POINT"

echo ""
echo "========================================="
echo "✓ Rootfs preparation complete!"
echo "========================================="
echo ""
echo "Rootfs location: $ROOTFS_PATH"
echo "Rootfs size: $(du -h $ROOTFS_PATH | cut -f1)"
echo ""
echo "Pre-installed tools:"
echo "  - Ubuntu $UBUNTU_VERSION base system"
echo "  - Git"
echo "  - Node.js (latest LTS)"
echo "  - Bun (latest)"
echo "  - Claude Code (if npm install succeeded)"
echo "  - Build essentials"
echo "  - SSH server"
echo "  - fc-agent service (will auto-start)"
echo ""
echo "Default credentials:"
echo "  Username: root"
echo "  Password: aetherium"
echo ""
echo "Note: Change the root password in production!"
echo ""
echo "To use this rootfs with Firecracker, ensure your worker is configured with:"
echo "  rootfs_template: $ROOTFS_PATH"
echo ""
