#!/bin/bash
set -e

echo "=========================================="
echo "Installing Firecracker v1.7.0"
echo "=========================================="
echo ""

# Check if running with sudo
if [ "$EUID" -eq 0 ]; then
    echo "Please run this script as a regular user (it will ask for sudo when needed)"
    exit 1
fi

# Create Firecracker directory
echo "1. Creating /var/firecracker directory..."
sudo mkdir -p /var/firecracker
sudo chown $USER:$USER /var/firecracker
echo "   ✓ Directory created"
echo ""

# Download Firecracker binary
echo "2. Downloading Firecracker v1.7.0..."
cd /tmp
if [ ! -f "firecracker-v1.7.0-x86_64.tgz" ]; then
    wget -q https://github.com/firecracker-microvm/firecracker/releases/download/v1.7.0/firecracker-v1.7.0-x86_64.tgz
fi
echo "   ✓ Downloaded"
echo ""

# Extract and install
echo "3. Installing Firecracker binary..."
tar -xzf firecracker-v1.7.0-x86_64.tgz
sudo cp release-v1.7.0-x86_64/firecracker-v1.7.0-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker
rm -rf release-v1.7.0-x86_64 firecracker-v1.7.0-x86_64.tgz
echo "   ✓ Installed to /usr/local/bin/firecracker"
echo ""

# Download kernel
echo "4. Downloading Linux kernel for Firecracker..."
if [ ! -f "/var/firecracker/vmlinux" ] || [ ! -s "/var/firecracker/vmlinux" ]; then
    echo "   Downloading kernel (~21MB)..."
    wget -O /var/firecracker/vmlinux \
        https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin
    echo "   ✓ Kernel downloaded"
else
    echo "   ✓ Kernel already exists"
fi
echo ""

# Download rootfs
echo "5. Downloading root filesystem (Ubuntu 22.04)..."
if [ ! -f "/var/firecracker/rootfs.ext4" ]; then
    wget -q -O /var/firecracker/rootfs.ext4 \
        https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/ubuntu-22.04.ext4
    echo "   ✓ Rootfs downloaded (~200MB)"
else
    echo "   ✓ Rootfs already exists"
fi
echo ""

# Configure KVM access
echo "6. Configuring KVM access..."
if groups | grep -q kvm; then
    echo "   ✓ User already in kvm group"
else
    sudo usermod -aG kvm $USER
    echo "   ✓ Added user to kvm group"
    echo "   ⚠ You need to log out and back in for group changes to take effect"
fi
echo ""

# Verify installation
echo "7. Verifying installation..."
firecracker --version
echo ""

# Check KVM
if [ -c /dev/kvm ]; then
    echo "   ✓ /dev/kvm exists"
    ls -l /dev/kvm
else
    echo "   ✗ /dev/kvm not found - KVM may not be enabled"
fi
echo ""

# List downloaded files
echo "8. Downloaded files:"
ls -lh /var/firecracker/
echo ""

echo "=========================================="
echo "✓ Firecracker Installation Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. If you were added to kvm group, log out and back in"
echo "2. Verify KVM access: ls -l /dev/kvm"
echo "3. Test Firecracker: firecracker --version"
echo "4. Run demo: ./bin/vm-cli init firecracker"
echo ""
