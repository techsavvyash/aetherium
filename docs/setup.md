# Setup Guide

Complete setup instructions for Aetherium.

## Philosophy

- One-time setup for infrastructure configuration
- Automatic operation for daily usage
- Self-documenting errors with exact fixes

## One-Time Setup

### Build Binaries

```bash
go build -o bin/worker ./cmd/worker
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/aether-cli ./cmd/aether-cli
go build -o bin/fc-agent ./cmd/fc-agent
```

### Start Infrastructure

```bash
docker-compose up -d
```

Runs:
- PostgreSQL (port 5432)
- Redis (port 6379)
- Loki (port 3100, optional)

### Setup Network (Requires sudo, ONE TIME)

```bash
sudo ./scripts/setup-network.sh
```

Creates:
- Bridge: `aetherium0` (172.16.0.1/24)
- NAT rules for internet access
- IP forwarding and iptables rules

### Setup Rootfs (Requires sudo, ONE TIME)

```bash
sudo ./scripts/setup-rootfs-once.sh
```

Configures:
- DNS auto-configuration on boot
- Kernel parameter parsing
- Network self-configuration

## Daily Usage

### Start Worker

```bash
# Option 1: With sudo (simplest)
sudo ./scripts/start-worker.sh

# Option 2: Grant capability (one-time)
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

### Start API Gateway

```bash
./bin/api-gateway
```

### Create VMs

```bash
# Create VM
./bin/aether-cli -type vm:create -name my-vm

# Get VM ID from database
VM_ID=$(docker exec aetherium-postgres psql -U aetherium -tc \
  "SELECT id FROM vms WHERE name='my-vm' LIMIT 1;")

# Execute command
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd git -args "--version"

# Delete VM
./bin/aether-cli -type vm:delete -vm-id $VM_ID
```

## Automatic Operations

When creating a VM, the system automatically:

1. Creates TAP device (e.g., `aether-bf0f8499`)
2. Allocates IP from 172.16.0.0/24 subnet
3. Generates unique MAC address
4. Attaches TAP to bridge
5. Configures kernel parameters
6. Boots VM with Firecracker
7. Auto-configures DNS inside VM
8. Installs default tools (git, nodejs, bun, claude-code)
9. Stores VM metadata in PostgreSQL
10. Queues tasks in Redis

## Troubleshooting

**Bridge not found:**
```bash
sudo ./scripts/setup-network.sh
```

**TAP device creation fails:**
```bash
# Option 1: Use start script with sudo
sudo ./scripts/start-worker.sh

# Option 2: Grant capability
sudo setcap cap_net_admin+ep ./bin/worker
```

**DNS resolution fails in VM:**
```bash
sudo ./scripts/setup-rootfs-once.sh
# Create new VMs (existing VMs use old rootfs)
```

## Architecture

```
Host System
  ├── Worker (manages VMs)
  ├── API Gateway (port 8080)
  ├── PostgreSQL (state)
  ├── Redis (task queue)
  ├── Bridge: aetherium0 (172.16.0.1/24)
  │   ├── TAP1 -> aether-vm1 (172.16.0.2)
  │   └── TAP2 -> aether-vm2 (172.16.0.3)
  └── NAT -> Internet
```

## File Structure

**One-Time Setup Scripts:**
- `scripts/setup-network.sh` - Bridge and NAT (run once)
- `scripts/setup-rootfs-once.sh` - Rootfs configuration (run once)

**Runtime Scripts:**
- `scripts/start-worker.sh` - Start worker with sudo (daily)

**Automatic Components:**
- `pkg/network/network.go` - TAP device management
- `pkg/vmm/firecracker/firecracker.go` - VM creation
- `pkg/tools/installer.go` - Tool installation
- `pkg/worker/worker.go` - Task orchestration
