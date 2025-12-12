# VM Creation Issues

## Bridge Not Found

**Error:**
```
bridge aetherium0 does not exist
```

**Solution:**
```bash
sudo ./scripts/setup-network.sh
```

This creates the bridge and NAT rules (one-time only).

## TAP Device Creation Fails

**Error:**
```
failed to create TAP device (needs CAP_NET_ADMIN)
```

**Solution - Option 1 (Simplest):**
```bash
sudo ./scripts/start-worker.sh
```

**Solution - Option 2 (One-time):**
```bash
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

## DNS Resolution Fails in VM

**Error:**
```
Temporary failure resolving 'github.com'
```

**Solution:**
```bash
sudo ./scripts/setup-rootfs-once.sh
```

Then create new VMs (existing VMs use old rootfs).

## Rootfs Not Found

**Error:**
```
rootfs not found at /var/firecracker/rootfs.ext4
```

**Solution:**
```bash
# Prepare rootfs with tools
sudo ./scripts/prepare-rootfs-with-tools.sh
```

This takes 5-10 minutes but creates a complete rootfs.

## VM Boot Timeout

**Error:**
```
VM did not boot within timeout period
```

**Check:**
1. Kernel path is correct
2. Rootfs exists and is readable
3. Firecracker is installed: `which firecracker`
4. vhost-vsock module is loaded: `lsmod | grep vhost_vsock`

**Solution:**
```bash
# Check Firecracker installation
sudo which firecracker

# Load vsock module
sudo modprobe vhost_vsock

# Check logs
cat /tmp/aetherium-vm-*.sock.log
```

## KVM Not Available

**Error:**
```
KVM device not available: /dev/kvm
```

**Check:**
```bash
ls -l /dev/kvm
```

**Solution:**
```bash
# Add user to kvm group
sudo usermod -aG kvm $USER
newgrp kvm

# Or enable KVM if on VM
sudo modprobe kvm
```

## VM Runs but No Network

**Check:**
```bash
# Inside VM, run:
ip addr show

# If eth0 has no IP, network auto-config failed
```

**Solution:**
1. Run setup: `sudo ./scripts/setup-rootfs-once.sh`
2. Create new VM (uses updated rootfs)

## Tool Installation Timeout

**Error:**
```
tool installation timeout after 20m
```

**Solution:**
- Use pre-built rootfs: `sudo ./scripts/prepare-rootfs-with-tools.sh`
- Or increase timeout in worker config

## Database Connection Failed

**Error:**
```
failed to connect to database
```

**Check:**
```bash
# Is PostgreSQL running?
docker ps | grep postgres

# Is it accepting connections?
docker exec aetherium-postgres pg_isready
```

**Solution:**
```bash
# Start infrastructure
docker-compose up -d

# Run migrations
migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" \
        -path ./migrations up
```

## Queue Connection Failed

**Error:**
```
failed to connect to queue
```

**Check:**
```bash
# Is Redis running?
docker ps | grep redis

# Can we connect?
docker exec aetherium-redis redis-cli ping
```

**Solution:**
```bash
# Start Redis
docker run -d --name aetherium-redis \
  -p 6379:6379 \
  redis:7-alpine
```

## Worker Exits Immediately

**Check logs:**
```bash
# Run worker with verbose output
./bin/worker -v

# Check permissions
ls -la /dev/vhost-vsock
ls -la /dev/kvm
```

**Permissions Needed:**
```bash
# Grant capabilities
sudo setcap cap_net_admin+ep ./bin/worker
sudo setcap cap_sys_admin+ep ./bin/worker

# Or run with sudo
sudo ./bin/worker
```
