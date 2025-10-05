#!/bin/bash
# Check virtio devices in guest

if [ "$EUID" -ne 0 ]; then
    echo "Must run as root"
    exit 1
fi

ROOTFS="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/mnt"

mount -o loop "$ROOTFS" "$MOUNT_POINT"

# Add script to check virtio devices at boot
cat > "$MOUNT_POINT/usr/local/bin/check-virtio.sh" << 'CHECKEOF'
#!/bin/bash
exec >> /var/log/virtio-check.log 2>&1

echo "=== Virtio Device Check $(date) ==="

echo "1. All virtio devices:"
ls -la /sys/bus/virtio/devices/ 2>/dev/null || echo "No virtio bus found"

echo ""
echo "2. Virtio drivers:"
ls -la /sys/bus/virtio/drivers/ 2>/dev/null || echo "No virtio drivers"

echo ""
echo "3. Vsock specific:"
find /sys -name "*vsock*" 2>/dev/null || echo "No vsock in sysfs"

echo ""
echo "4. PCI devices (should be empty with pci=off):"
ls -la /sys/bus/pci/devices/ 2>/dev/null || echo "No PCI bus (expected)"

echo ""
echo "5. Platform devices:"
ls -la /sys/bus/platform/devices/ | grep virtio || echo "No virtio platform devices"

echo ""
echo "6. Check dmesg for vsock:"
dmesg | grep -i vsock || echo "No vsock in dmesg"

echo ""
echo "7. Check dmesg for virtio:"
dmesg | grep -i virtio | head -20

echo "=== End Check ==="
CHECKEOF

chmod +x "$MOUNT_POINT/usr/local/bin/check-virtio.sh"

# Create service to run before debug
cat > "$MOUNT_POINT/etc/systemd/system/virtio-check.service" << 'SVCEOF'
[Unit]
Description=Virtio Device Check
DefaultDependencies=no
Before=vsock-debug.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/check-virtio.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
SVCEOF

# Enable service
mkdir -p "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants"
ln -sf /etc/systemd/system/virtio-check.service \
    "$MOUNT_POINT/etc/systemd/system/multi-user.target.wants/virtio-check.service"

# Update dependencies
sed -i 's/Before=fc-agent.service/Before=virtio-check.service/' \
    "$MOUNT_POINT/etc/systemd/system/vsock-debug.service" 2>/dev/null || true

umount "$MOUNT_POINT"

echo "✓ Virtio check added"
echo ""
echo "After running test, check with:"
echo "  sudo bash -c 'mount -o loop /var/firecracker/rootfs.ext4 /mnt && cat /mnt/var/log/virtio-check.log && umount /mnt'"
