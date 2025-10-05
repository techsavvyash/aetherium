#!/bin/bash
# Add debug script to rootfs to check vsock status at boot

if [ "$EUID" -ne 0 ]; then
    echo "Must run as root"
    exit 1
fi

ROOTFS="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/mnt"

echo "Adding vsock debug script to rootfs..."

mount -o loop "$ROOTFS" "$MOUNT_POINT"

# Create debug script
cat > "$MOUNT_POINT/usr/local/bin/vsock-debug.sh" << 'DEBUGEOF'
#!/bin/bash
# Debug vsock availability

exec >> /var/log/vsock-debug.log 2>&1

echo "=== Vsock Debug $(date) ==="

echo "1. Kernel modules:"
lsmod | grep -i vsock || echo "No vsock modules loaded"

echo ""
echo "2. Vsock devices:"
ls -l /dev/*vsock* 2>/dev/null || echo "No vsock devices found"

echo ""
echo "3. Vsock sockets:"
ss -xa | grep vsock || echo "No vsock sockets"

echo ""
echo "4. Network interfaces:"
ip addr show

echo ""
echo "5. Trying to load virtio_vsock module (if not loaded):"
modprobe virtio_vsock 2>&1 || echo "Could not load virtio_vsock"

echo ""
echo "6. Check again after modprobe:"
lsmod | grep vsock || echo "Still no vsock modules"
ls -l /dev/*vsock* 2>/dev/null || echo "Still no vsock devices"

echo ""
echo "7. Agent binary exists:"
ls -l /usr/local/bin/fc-agent 2>&1

echo ""
echo "8. Agent service status:"
systemctl status fc-agent --no-pager 2>&1 || echo "Service check failed"

echo ""
echo "=== End Debug ==="
DEBUGEOF

chmod +x "$MOUNT_POINT/usr/local/bin/vsock-debug.sh"

# Create systemd service to run debug script before agent
cat > "$MOUNT_POINT/etc/systemd/system/vsock-debug.service" << 'SVCEOF'
[Unit]
Description=Vsock Debug Logger
Before=fc-agent.service
DefaultDependencies=no

[Service]
Type=oneshot
ExecStart=/usr/local/bin/vsock-debug.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
SVCEOF

# Enable debug service
mkdir -p "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants"
ln -sf /etc/systemd/system/vsock-debug.service \
    "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/vsock-debug.service"

# Also update agent service to run after debug
sed -i 's/After=network.target/After=network.target vsock-debug.service/' \
    "$MOUNT_POINT/etc/systemd/system/fc-agent.service"

umount "$MOUNT_POINT"

echo "✓ Debug script added"
echo ""
echo "After running a test, check the debug log:"
echo "  sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt"
echo "  sudo cat /mnt/var/log/vsock-debug.log"
echo "  sudo umount /mnt"
