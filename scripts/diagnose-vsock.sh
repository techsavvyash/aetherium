#!/bin/bash
# Diagnostic script to check vsock configuration

echo "=== Vsock Diagnostic Tool ==="
echo ""

# Check host vsock
echo "1. Host Vsock Status:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
if lsmod | grep -q vhost_vsock; then
    echo "✓ vhost-vsock module loaded"
    lsmod | grep vsock | sed 's/^/  /'
else
    echo "✗ vhost-vsock module NOT loaded"
    echo "  Run: sudo modprobe vhost-vsock"
fi
echo ""

if [ -c /dev/vhost-vsock ]; then
    echo "✓ /dev/vhost-vsock device exists"
    ls -l /dev/vhost-vsock | sed 's/^/  /'
else
    echo "✗ /dev/vhost-vsock device NOT found"
fi
echo ""

# Check kernel
echo "2. Guest Kernel Status:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
KERNEL_PATH="/var/firecracker/vmlinux"
if [ -f "$KERNEL_PATH" ]; then
    echo "✓ Kernel found: $KERNEL_PATH"
    ls -lh "$KERNEL_PATH" | awk '{print "  Size: " $5 ", Modified: " $6 " " $7 " " $8}'

    # Check kernel version
    VERSION=$(strings "$KERNEL_PATH" | grep "Linux version" | head -1)
    echo "  Version: $VERSION" | head -c 100
    echo ""

    # Check for vsock symbols
    if strings "$KERNEL_PATH" | grep -q "virtio_vsock"; then
        echo "✓ Kernel contains vsock symbols"
        echo "  Found symbols:"
        strings "$KERNEL_PATH" | grep "virtio_vsock" | head -5 | sed 's/^/    /'
    else
        echo "✗ No vsock symbols found in kernel"
    fi
else
    echo "✗ Kernel not found at $KERNEL_PATH"
fi
echo ""

# Check kernel config if available
echo "3. Kernel Configuration:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
CONFIG_URL="https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.13/x86_64/vmlinux-5.10.239.config"
echo "Checking config from: $CONFIG_URL"
VSOCK_CONFIG=$(curl -s "$CONFIG_URL" | grep -i "CONFIG_VIRTIO_VSOCK")
if [ -n "$VSOCK_CONFIG" ]; then
    echo "✓ Kernel config has vsock support:"
    echo "$VSOCK_CONFIG" | sed 's/^/  /'
else
    echo "✗ Could not verify vsock config"
fi
echo ""

# Check agent in rootfs
echo "4. Agent Deployment (requires sudo):"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "To check if agent is deployed in rootfs, run:"
echo "  sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt"
echo "  ls -lh /mnt/usr/local/bin/fc-agent"
echo "  cat /mnt/etc/systemd/system/fc-agent.service"
echo "  sudo umount /mnt"
echo ""

# Check running VMs
echo "5. Running Firecracker VMs:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
if pgrep -a firecracker > /dev/null; then
    echo "✓ Firecracker processes found:"
    pgrep -a firecracker | sed 's/^/  /'
else
    echo "○ No running Firecracker VMs"
fi
echo ""

echo "6. Vsock Connections:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━"
if command -v ss &> /dev/null; then
    VSOCK_CONNS=$(ss -xa 2>/dev/null | grep vsock || true)
    if [ -n "$VSOCK_CONNS" ]; then
        echo "✓ Active vsock connections:"
        echo "$VSOCK_CONNS" | sed 's/^/  /'
    else
        echo "○ No active vsock connections"
    fi
else
    echo "⚠ 'ss' command not available"
fi
echo ""

echo "=== Recommendations ==="
echo ""
if lsmod | grep -q vhost_vsock && [ -c /dev/vhost-vsock ] && strings "$KERNEL_PATH" 2>/dev/null | grep -q "virtio_vsock"; then
    echo "✓ Host and kernel configuration looks good"
    echo ""
    echo "Next steps:"
    echo "1. Ensure agent is deployed: sudo ./scripts/setup-and-test.sh"
    echo "2. Check VM logs at: /tmp/firecracker-test-vm.sock.log (after running test)"
    echo "3. Look for agent startup messages and vsock initialization"
else
    echo "⚠ Some components are missing or misconfigured"
    echo ""
    echo "Run these commands to fix:"
    if ! lsmod | grep -q vhost_vsock; then
        echo "  sudo modprobe vhost-vsock"
    fi
    if [ ! -c /dev/vhost-vsock ]; then
        echo "  sudo chmod 666 /dev/vhost-vsock"
    fi
    echo "  sudo ./scripts/download-vsock-kernel.sh"
    echo "  sudo ./scripts/setup-and-test.sh"
fi
