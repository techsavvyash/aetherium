# Fixing Vsock Connection Issue

## Problem

The vsock connection times out because the guest kernel doesn't have vsock support compiled in.

```
Cannot connect to VM agent via vsock: connection timeout
```

## Root Cause

The kernel downloaded from `https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin` doesn't have `CONFIG_VIRTIO_VSOCK=y` enabled.

## Solution: Use Kernel with Vsock Support

### Step 1: Download vsock-enabled kernel

```bash
sudo ./scripts/download-vsock-kernel.sh
```

This will:
- Download kernel v5.10.217 with vsock support
- Backup your current kernel to `/var/firecracker/vmlinux.old`
- Replace `/var/firecracker/vmlinux` with the vsock-enabled version

### Step 2: Deploy the agent

```bash
sudo ./scripts/complete-vsock-test.sh
```

This ensures:
- vhost-vsock module is loaded on host
- Updated agent is deployed to VM rootfs
- Systemd service is configured

### Step 3: Test

```bash
# Quick test
./bin/fc-test

# OR test with full diagnostics (recommended)
./scripts/test-and-diagnose.sh
```

Expected output:
```
=== Firecracker VM Test ===

1. Creating VM...
   ✓ VM created: test-vm

2. Starting VM...
   ✓ VM started

3. Waiting for VM to boot and agent to start (20s)...
   Logs will be available at: /tmp/firecracker-test-vm.sock.log
   ✓ Boot complete

4. Testing command execution...
   Exit Code: 0
   Stdout: Hello from Firecracker!

✓ SUCCESS! Command execution works!
```

If the test fails:
```bash
# Check VM logs
cat /tmp/firecracker-test-vm.sock.log

# Run diagnostics
./scripts/diagnose-vsock.sh
```

## Alternative: Use Docker for Now

If you want to test the platform without dealing with vsock:

```bash
./bin/docker-demo
```

Docker containers have built-in networking, so command execution will work immediately.

## Technical Details

### What's Needed for Vsock

**Host Requirements:**
- ✅ `vhost-vsock` kernel module loaded
- ✅ `/dev/vhost-vsock` device available
- ✅ Go vsock library (`github.com/mdlayher/vsock`)

**Guest Requirements:**
- ❌ `CONFIG_VIRTIO_VSOCK=y` in kernel (this is what's missing)
- ✅ Agent listening on vsock port 9999
- ✅ Systemd service to start agent

### Checking Kernel Config

To verify if a kernel has vsock support:

```bash
# Extract config if available
scripts/firecracker-kernels/extract-config.sh /var/firecracker/vmlinux

# Or check for vsock symbols
strings /var/firecracker/vmlinux | grep -i vsock
```

### Agent Fallback Behavior

The agent tries both transports:

1. **First:** Try vsock (`vsock.Listen(port, nil)`)
2. **Fallback:** If vsock unavailable, use TCP (`net.Listen("tcp", ":9999")`)

However, TCP fallback won't work without network configuration.

## Summary

The vsock connection fails because:
1. Host has vsock support ✓
2. Agent has vsock code ✓
3. **Guest kernel lacks vsock drivers** ✗

**Fix:** Download kernel with `CONFIG_VIRTIO_VSOCK=y` enabled.
