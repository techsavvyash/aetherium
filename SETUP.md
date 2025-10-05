# Aetherium Setup Guide

This guide covers the **one-time setup** required for Aetherium. After this setup, all VM operations are automatic.

## Philosophy

Aetherium follows these principles:
1. **One-time setup** - Infrastructure configuration is done once
2. **Automatic operation** - Daily usage requires no manual intervention
3. **Self-documenting errors** - When something needs fixing, the code tells you exactly what to do

## One-Time Setup (Run Once)

### Step 1: Build Binaries

```bash
# Build all components
go build -o bin/worker cmd/worker/main.go
go build -o bin/api-gateway cmd/api-gateway/main.go
go build -o bin/aether-cli cmd/cli/main.go
```

### Step 2: Start Infrastructure

```bash
# Start PostgreSQL and Redis
docker-compose up -d
```

### Step 3: Setup Network (Requires Sudo - ONE TIME)

```bash
# Configure bridge, NAT, and iptables
sudo ./scripts/setup-network.sh
```

This creates:
- Bridge: `aetherium0` (172.16.0.1/24)
- NAT rules for internet access
- IP forwarding enabled
- iptables forwarding rules

**This only needs to be run once per system!**

### Step 4: Setup Rootfs Template (Requires Sudo - ONE TIME)

```bash
# Configure DNS and auto-configuration in the rootfs
sudo ./scripts/setup-rootfs-once.sh
```

This configures the rootfs template to:
- Automatically configure DNS on boot
- Parse DNS from kernel parameters
- Self-configure networking

**This only needs to be run once! All VMs use this template.**

## Daily Usage (No Sudo Required After Setup)

### Start the Worker

The worker needs network capabilities to create TAP devices:

```bash
# Option 1: Run with sudo (simplest for development)
sudo ./scripts/start-worker.sh

# Option 2: Grant capabilities (run once, then no sudo needed)
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

### Start the API Gateway (Optional)

```bash
./bin/api-gateway
```

### Create VMs

```bash
# Create a VM - no manual intervention required!
./bin/aether-cli -type vm:create -name my-vm

# Execute commands
VM_ID="..." # from previous command or database
./bin/aether-cli -type vm:execute -vm-id "$VM_ID" -cmd ls -args "-la"

# Delete VM
./bin/aether-cli -type vm:delete -vm-id "$VM_ID"
```

## What Happens Automatically?

When you create a VM, the system automatically:

1. **Creates a TAP device** with unique name (e.g., `aether-bf0f8499`)
2. **Allocates an IP** from the 172.16.0.0/24 subnet
3. **Generates a MAC address** based on VM ID
4. **Attaches TAP to bridge** for connectivity
5. **Configures kernel parameters** with IP and DNS
6. **Boots the VM** with Firecracker
7. **Auto-configures DNS** inside the VM (via systemd service we added)
8. **Installs tools** (git, nodejs, bun, claude-code) via package manager
9. **Stores VM metadata** in PostgreSQL
10. **Queues tasks** in Redis

## Troubleshooting

### Bridge Not Found

**Error:** `bridge aetherium0 does not exist`

**Solution:** Run the one-time network setup:
```bash
sudo ./scripts/setup-network.sh
```

### TAP Device Creation Fails

**Error:** `failed to create TAP device (needs CAP_NET_ADMIN)`

**Solution:** The worker needs network privileges:
```bash
# Option 1: Use start script
sudo ./scripts/start-worker.sh

# Option 2: Grant capability (one-time)
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

### DNS Resolution Fails in VM

**Error:** `Temporary failure resolving 'github.com'`

**Solution:** Run the one-time rootfs setup:
```bash
sudo ./scripts/setup-rootfs-once.sh
```

Then create a new VM (existing VMs use old rootfs).

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Host System                          │
│                                                          │
│  ┌──────────────┐    ┌─────────────┐                   │
│  │   Worker     │───▶│  PostgreSQL │                   │
│  │ (needs sudo) │    └─────────────┘                   │
│  └──────┬───────┘                                       │
│         │ creates                                       │
│         ▼                                               │
│  ┌─────────────────────────────────────┐               │
│  │  aetherium0 Bridge (172.16.0.1/24) │               │
│  │  ┌───────────┐  ┌──────────┐       │               │
│  │  │ aether-vm1│  │aether-vm2│       │               │
│  │  │172.16.0.2 │  │172.16.0.3│       │               │
│  │  └─────┬─────┘  └────┬─────┘       │               │
│  └────────┼─────────────┼─────────────┘               │
│           │ NAT         │ NAT                          │
│           ▼             ▼                              │
│  ┌────────────────────────────┐                       │
│  │   Internet (via NAT)       │                       │
│  └────────────────────────────┘                       │
│                                                         │
│  VMs:                                                  │
│  ┌──────────────────────────────┐                     │
│  │  Firecracker microVM         │                     │
│  │  • Auto-configured DNS        │                     │
│  │  • Auto-configured IP         │                     │
│  │  • Tool installation          │                     │
│  │  • vsock agent for commands   │                     │
│  └──────────────────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

## Files Overview

### One-Time Setup Scripts (Require Sudo)
- `scripts/setup-network.sh` - Configure bridge, NAT, iptables (run once)
- `scripts/setup-rootfs-once.sh` - Configure rootfs template (run once)

### Runtime Scripts
- `scripts/start-worker.sh` - Start worker with sudo (daily use)

### Code Components (Automatic)
- `pkg/network/network.go` - TAP device management (automatic)
- `pkg/vmm/firecracker/firecracker.go` - VM creation with network (automatic)
- `pkg/tools/installer.go` - Tool installation (automatic)
- `pkg/worker/worker.go` - Task orchestration (automatic)

## Summary

**One-time setup (2 commands):**
```bash
sudo ./scripts/setup-network.sh      # Network infrastructure
sudo ./scripts/setup-rootfs-once.sh  # Rootfs template
```

**Daily usage (1 command):**
```bash
sudo ./scripts/start-worker.sh       # Start worker
```

Then all VM operations work automatically with no manual intervention!
