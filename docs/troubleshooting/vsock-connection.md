# Vsock Connection Timeout

## Problem

VM command execution fails with:
```
Cannot connect to VM agent via vsock: connection timeout
```

## Root Cause

The guest kernel lacks `CONFIG_VIRTIO_VSOCK=y` enabled.

## Solution

### Step 1: Download vsock-enabled kernel

```bash
sudo ./scripts/download-vsock-kernel.sh
```

This:
- Downloads kernel v5.10.217 with vsock support
- Backs up current kernel to `/var/firecracker/vmlinux.old`
- Replaces `/var/firecracker/vmlinux` with vsock version

### Step 2: Deploy agent

```bash
sudo ./scripts/complete-vsock-test.sh
```

This:
- Loads vhost-vsock module
- Deploys updated agent to rootfs
- Configures systemd service

### Step 3: Test

```bash
./bin/fc-test
```

Expected output:
```
=== Firecracker VM Test ===

1. Creating VM...
   ✓ VM created: test-vm

2. Starting VM...
   ✓ VM started

3. Waiting for boot...
   ✓ Boot complete

4. Testing command execution...
   Exit Code: 0
   Stdout: Hello from Firecracker!

✓ SUCCESS! Command execution works!
```

### If Test Fails

```bash
# Check VM logs
cat /tmp/firecracker-test-vm.sock.log

# Run diagnostics
./scripts/diagnose-vsock.sh
```

## Verify Kernel Config

```bash
# Check for vsock symbols
strings /var/firecracker/vmlinux | grep -i vsock
```

## Alternative: Use Docker

If you want to skip vsock setup:

```bash
./bin/docker-demo
```

Docker containers have built-in networking, so command execution works immediately.

## Technical Details

**Host Requirements:**
- vhost-vsock kernel module loaded
- `/dev/vhost-vsock` device available
- Go vsock library (mdlayler/vsock)

**Guest Requirements:**
- `CONFIG_VIRTIO_VSOCK=y` in kernel
- Agent listening on vsock port 9999
- Systemd service to start agent

## Agent Fallback

The agent tries both transports:
1. Try vsock (preferred)
2. Fallback to TCP (if vsock unavailable)

However, TCP requires network configuration inside the VM.
