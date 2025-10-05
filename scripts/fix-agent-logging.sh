#!/bin/bash
# Fix agent service to log to file instead of journal

if [ "$EUID" -ne 0 ]; then
    echo "Must run as root"
    exit 1
fi

ROOTFS="/var/firecracker/rootfs.ext4"
MOUNT_POINT="/mnt"

echo "Fixing agent service logging..."

mount -o loop "$ROOTFS" "$MOUNT_POINT"

# Update service to log to file
cat > "$MOUNT_POINT/etc/systemd/system/fc-agent.service" << 'EOF'
[Unit]
Description=Firecracker Agent
After=network.target vsock-debug.service

[Service]
Type=simple
ExecStart=/usr/local/bin/fc-agent
Restart=no
StandardOutput=append:/var/log/fc-agent.log
StandardError=append:/var/log/fc-agent.log

[Install]
WantedBy=multi-user.target
EOF

echo "✓ Service updated to log to /var/log/fc-agent.log"

umount "$MOUNT_POINT"

echo ""
echo "Now run the test and check agent logs with:"
echo "  sudo bash -c 'mount -o loop /var/firecracker/rootfs.ext4 /mnt && cat /mnt/var/log/fc-agent.log && umount /mnt'"
