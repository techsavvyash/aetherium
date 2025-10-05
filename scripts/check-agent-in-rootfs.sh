#!/bin/bash
# Check if agent is deployed in rootfs

set -e

ROOTFS="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/mnt"

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root (use sudo)"
    exit 1
fi

echo "Checking agent deployment in rootfs..."
echo ""

# Mount rootfs
echo "Mounting $ROOTFS to $MOUNT_POINT..."
mount -o loop "$ROOTFS" "$MOUNT_POINT"

echo ""
echo "1. Agent Binary:"
echo "━━━━━━━━━━━━━━━━━━━━"
if [ -f "$MOUNT_POINT/usr/local/bin/fc-agent" ]; then
    echo "✓ Agent binary exists"
    ls -lh "$MOUNT_POINT/usr/local/bin/fc-agent"

    # Check if it's executable
    if [ -x "$MOUNT_POINT/usr/local/bin/fc-agent" ]; then
        echo "✓ Binary is executable"
    else
        echo "✗ Binary is NOT executable"
    fi
else
    echo "✗ Agent binary NOT found at /usr/local/bin/fc-agent"
fi

echo ""
echo "2. Systemd Service:"
echo "━━━━━━━━━━━━━━━━━━━━"
if [ -f "$MOUNT_POINT/etc/systemd/system/fc-agent.service" ]; then
    echo "✓ Systemd service exists"
    echo ""
    cat "$MOUNT_POINT/etc/systemd/system/fc-agent.service"
    echo ""

    # Check if enabled
    if [ -L "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/fc-agent.service" ]; then
        echo "✓ Service is enabled (will start on boot)"
    else
        echo "✗ Service is NOT enabled"
        echo "  To enable: systemctl enable fc-agent (inside VM)"
    fi
else
    echo "✗ Systemd service NOT found"
fi

echo ""
echo "3. Systemd Service Link:"
echo "━━━━━━━━━━━━━━━━━━━━"
ls -l "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/" 2>/dev/null | grep fc-agent || echo "○ No fc-agent link found"

echo ""
echo "4. Check for systemd-enable in rootfs:"
echo "━━━━━━━━━━━━━━━━━━━━"
if [ -d "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants" ]; then
    echo "✓ multi-user.target.wants directory exists"
else
    echo "✗ multi-user.target.wants directory missing"
fi

# Unmount
echo ""
echo "Unmounting..."
umount "$MOUNT_POINT"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "If agent binary and service exist but service is not enabled:"
echo "  This is likely the problem!"
echo ""
echo "Fix: Re-run setup script to enable the service:"
echo "  sudo ./scripts/setup-and-test.sh"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
