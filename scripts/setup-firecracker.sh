#!/bin/bash
set -e

echo "=========================================="
echo "Firecracker Setup Script"
echo "=========================================="
echo

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${RED}Please do not run this script as root${NC}"
    echo "Run as regular user, will use sudo when needed"
    exit 1
fi

# 1. Create directories
echo "1. Creating directories..."
sudo mkdir -p /var/firecracker
sudo chown $USER:$USER /var/firecracker
echo -e "${GREEN}✓${NC} Directories created"
echo

# 2. Download Firecracker
echo "2. Downloading Firecracker v1.7.0..."
if [ ! -f "/usr/bin/firecracker" ]; then
    cd /tmp
    wget -q https://github.com/firecracker-microvm/firecracker/releases/download/v1.7.0/firecracker-v1.7.0-x86_64.tgz
    tar -xzf firecracker-v1.7.0-x86_64.tgz
    sudo mv release-v1.7.0-x86_64/firecracker-v1.7.0-x86_64 /usr/bin/firecracker
    sudo chmod +x /usr/bin/firecracker
    rm -rf firecracker-v1.7.0-x86_64.tgz release-v1.7.0-x86_64
    echo -e "${GREEN}✓${NC} Firecracker installed to /usr/bin/firecracker"
else
    echo -e "${YELLOW}⚠${NC} Firecracker already installed"
fi
echo

# 3. Download Linux kernel
echo "3. Downloading Linux kernel..."
if [ ! -f "/var/firecracker/vmlinux" ]; then
    wget -q -O /var/firecracker/vmlinux \
        https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/vmlinux-5.10.217
    echo -e "${GREEN}✓${NC} Kernel downloaded to /var/firecracker/vmlinux"
else
    echo -e "${YELLOW}⚠${NC} Kernel already exists"
fi
echo

# 4. Download or create rootfs
echo "4. Setting up rootfs..."
if [ ! -f "/var/firecracker/rootfs.ext4" ]; then
    echo "   Downloading Ubuntu 22.04 rootfs (this may take a while)..."
    wget -q -O /var/firecracker/rootfs.ext4 \
        https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/ubuntu-22.04.ext4
    echo -e "${GREEN}✓${NC} Rootfs downloaded to /var/firecracker/rootfs.ext4"
else
    echo -e "${YELLOW}⚠${NC} Rootfs already exists"
fi
echo

# 5. Check KVM access
echo "5. Checking KVM access..."
if [ -c "/dev/kvm" ]; then
    if groups | grep -q kvm; then
        echo -e "${GREEN}✓${NC} User is in kvm group"
    else
        echo -e "${YELLOW}⚠${NC} Adding user to kvm group..."
        sudo usermod -aG kvm $USER
        echo -e "${GREEN}✓${NC} User added to kvm group"
        echo -e "${YELLOW}⚠${NC} You need to log out and back in for group changes to take effect"
    fi
else
    echo -e "${RED}✗${NC} /dev/kvm not found - KVM support required"
    exit 1
fi
echo

# 6. Verify installation
echo "6. Verifying installation..."
echo "   Firecracker: $(firecracker --version 2>&1 | head -1)"
echo "   Kernel size: $(du -h /var/firecracker/vmlinux | cut -f1)"
echo "   Rootfs size: $(du -h /var/firecracker/rootfs.ext4 | cut -f1)"
echo

echo "=========================================="
echo -e "${GREEN}Setup Complete!${NC}"
echo "=========================================="
echo
echo "Next steps:"
echo "  1. If you were added to kvm group, log out and back in"
echo "  2. Run: ./bin/fc-cli create test-vm"
echo "  3. Run: ./bin/fc-cli start test-vm"
echo
