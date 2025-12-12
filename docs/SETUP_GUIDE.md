# Aetherium Complete Setup Guide

Complete reference for one-time setup and daily operations.

## Table of Contents
- [Quick Start](#quick-start)
- [One-Time Setup](#one-time-setup)
- [Daily Operations](#daily-operations)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)

## Quick Start

For experienced users:

```bash
# 1. Build binaries
make build

# 2. Start infrastructure
docker-compose up -d

# 3. One-time network setup
sudo ./scripts/setup-network.sh

# 4. One-time rootfs setup
sudo ./scripts/setup-rootfs-once.sh

# Then daily: start worker with network privileges
sudo ./bin/worker

# In another terminal: start API Gateway
./bin/api-gateway

# Create VMs
./bin/aether-cli -type vm:create -name my-vm
```

---

## One-Time Setup

### Philosophy

Aetherium follows these principles:
1. **One-time setup** - Infrastructure configuration done once
2. **Automatic operation** - Daily usage requires no manual intervention
3. **Self-documenting errors** - Clear error messages guide you to solutions

### Step 1: Build Binaries

```bash
cd /path/to/aetherium
make build
```

This builds:
- `bin/api-gateway` - REST API server
- `bin/worker` - Task worker daemon
- `bin/aether-cli` - Command-line tool
- `bin/fc-agent` - VM agent (for vsock communication)

### Step 2: Start Infrastructure

```bash
docker-compose up -d
```

This starts:
- **PostgreSQL** (aetherium-postgres) - State persistence
- **Redis** (aetherium-redis) - Task queue

Verify:
```bash
docker ps | grep aetherium
docker exec aetherium-postgres pg_isready
docker exec aetherium-redis redis-cli ping
```

### Step 3: Setup Network (Requires Sudo - Run Once)

```bash
sudo ./scripts/setup-network.sh
```

This creates:
- **Bridge**: `aetherium0` (172.16.0.1/24)
- **NAT rules** for internet access
- **IP forwarding** enabled
- **iptables rules** for packet routing

This only needs to be run **once per system**. The bridge and NAT persist across reboots.

Verify:
```bash
ip addr show aetherium0
ip link show | grep aether-
sudo iptables -t nat -L -n -v | grep 172.16
```

### Step 4: Setup Rootfs Template (Requires Sudo - Run Once)

```bash
sudo ./scripts/setup-rootfs-once.sh
```

This configures the rootfs template to:
- Automatically configure DNS on boot
- Parse DNS from kernel parameters
- Self-configure networking

This only needs to be run **once**. All VMs use this template.

---

## Daily Operations

### Start the Worker

The worker needs network capabilities to create TAP devices:

**Option 1: Run with sudo (simplest for development)**
```bash
sudo ./bin/worker
```

**Option 2: Grant capabilities (run once, then no sudo needed)**
```bash
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

**Option 3: Use startup script**
```bash
sudo ./scripts/start-worker.sh
```

### Start the API Gateway (Optional)

In another terminal:
```bash
./bin/api-gateway
```

The API Gateway exposes REST endpoints:
- `POST /api/v1/vms` - Create VM
- `GET /api/v1/vms` - List VMs
- `GET /api/v1/vms/{id}` - Get VM details
- `DELETE /api/v1/vms/{id}` - Delete VM
- `POST /api/v1/vms/{id}/execute` - Execute command

### Create and Manage VMs

```bash
# Create a VM
./bin/aether-cli -type vm:create -name my-vm -vcpus 2 -memory 2048

# Create VM with specific tools
./bin/aether-cli -type vm:create -name my-vm \
  -additional-tools go,python,rust \
  -tool-versions go:1.23.0,python:3.11

# List VMs
./bin/aether-cli -type vm:list

# Get VM details
./bin/aether-cli -type vm:get -vm-id <id>

# Execute command
./bin/aether-cli -type vm:execute -vm-id <id> -cmd git -args clone,https://github.com/user/repo

# Delete VM
./bin/aether-cli -type vm:delete -vm-id <id>
```

---

## Architecture

### Network Architecture

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

### VM Creation Process

When you create a VM, the system automatically:

1. **Creates a TAP device** with unique name (e.g., `aether-bf0f8499`)
2. **Allocates an IP** from the 172.16.0.0/24 subnet
3. **Generates a MAC address** based on VM ID
4. **Attaches TAP to bridge** for connectivity
5. **Configures kernel parameters** with IP and DNS
6. **Boots the VM** with Firecracker
7. **Auto-configures DNS** inside the VM (via systemd service)
8. **Installs tools** (git, nodejs, bun, claude-code) via package manager
9. **Stores VM metadata** in PostgreSQL
10. **Queues tasks** in Redis

### Component Interaction

```
┌──────────────────────────────────────────────────────────┐
│  Client (CLI or REST)                                   │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│  API Gateway (optional) or Task Service                  │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│  PostgreSQL: Store VM and task metadata                  │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│  Redis Queue: Enqueue tasks (Asynq)                      │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│  Worker: Process tasks                                   │
│  1. Pull task from Redis queue                           │
│  2. Execute task handler (create/execute/delete)         │
│  3. Store results in PostgreSQL                          │
└────────────────┬─────────────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────────────┐
│  Firecracker Orchestrator: Manage VM lifecycle           │
│  1. Create TAP device                                    │
│  2. Start Firecracker process                            │
│  3. Wait for boot                                        │
│  4. Execute commands via vsock                           │
│  5. Stop and delete VM                                   │
└──────────────────────────────────────────────────────────┘
```

---

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
# Option 1: Run with sudo
sudo ./bin/worker

# Option 2: Grant capability (one-time)
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker

# Option 3: Use startup script
sudo ./scripts/start-worker.sh
```

### DNS Resolution Fails in VM

**Error:** `Temporary failure resolving 'github.com'`

**Solution:** Run the one-time rootfs setup:
```bash
sudo ./scripts/setup-rootfs-once.sh
```

Then create a new VM (existing VMs use old rootfs).

### PostgreSQL Connection Fails

**Error:** `failed to connect to database: connection refused`

**Solution:** Check PostgreSQL is running:
```bash
docker ps | grep aetherium-postgres

# If not running, start it
docker-compose up -d

# Test connection
docker exec aetherium-postgres psql -U aetherium -d aetherium -c "SELECT version();"
```

### Redis Connection Fails

**Error:** `failed to connect to redis: connection refused`

**Solution:** Check Redis is running:
```bash
docker ps | grep aetherium-redis

# If not running, start it
docker-compose up -d

# Test connection
docker exec aetherium-redis redis-cli ping
```

### VM Creation Hangs

**Error:** Task never completes, VM stuck in "creating" state

**Solutions:**
1. Check worker is running: `ps aux | grep worker`
2. Check Redis queue: `docker exec aetherium-redis redis-cli LLEN asynq:queues:default`
3. Check Firecracker: `ps aux | grep firecracker`
4. Check logs: `tail -50 worker.log`

### Command Execution in VM Fails

**Error:** `Cannot connect to VM agent via vsock`

**Solutions:**
1. Verify kernel has vsock support: `ls -lh /var/firecracker/vmlinux`
2. Check vhost-vsock module: `lsmod | grep vhost_vsock`
3. SSH into VM and check agent: `ps aux | grep fc-agent`
4. Check agent logs in VM

---

## Environment Variables

### Worker

Required environment variables when running the worker:

```bash
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=aetherium
POSTGRES_PASSWORD=aetherium
POSTGRES_DB=aetherium

# Redis
REDIS_ADDR=localhost:6379

# Firecracker
KERNEL_PATH=/var/firecracker/vmlinux
ROOTFS_TEMPLATE=/var/firecracker/rootfs.ext4

# Optional
PROXY_ENABLED=false
```

### API Gateway

```bash
# API
API_PORT=8080

# PostgreSQL (same as worker)
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=aetherium
POSTGRES_PASSWORD=aetherium
POSTGRES_DB=aetherium

# Redis (same as worker)
REDIS_ADDR=localhost:6379
```

---

## File Structure Reference

```
aetherium/
├── bin/                          # Built binaries
│   ├── api-gateway
│   ├── worker
│   ├── aether-cli
│   └── fc-agent
│
├── cmd/                          # Source code entry points
│   ├── api-gateway/main.go
│   ├── worker/main.go
│   ├── aether-cli/main.go
│   └── fc-agent/main.go
│
├── pkg/                          # Core packages
│   ├── vmm/                      # VM orchestration
│   ├── queue/                    # Task queue (Asynq)
│   ├── storage/                  # Database (PostgreSQL)
│   ├── network/                  # Network management
│   ├── tools/                    # Tool installation
│   └── service/                  # High-level services
│
├── scripts/                      # Utility scripts
│   ├── setup-network.sh         # Network setup
│   ├── setup-rootfs-once.sh     # Rootfs template setup
│   └── start-worker.sh          # Worker startup
│
├── migrations/                   # Database migrations
│   └── 000001_initial_schema.up.sql
│
├── docs/                         # Documentation
│   ├── SETUP_GUIDE.md           # This file
│   ├── QUICKSTART.md
│   ├── API-GATEWAY.md
│   ├── PRODUCTION-ARCHITECTURE.md
│   └── ...
│
├── docker-compose.yml            # Infrastructure
├── Makefile                      # Build automation
├── README.md                     # Project overview
├── CLAUDE.md                     # AI guidelines (keep in root)
├── AGENTS.md                     # Agent guidelines (keep in root)
└── go.mod / go.sum              # Go dependencies
```

---

## Common Commands

### Building

```bash
make build              # Build all binaries
make clean              # Clean build artifacts
make deps               # Download and tidy modules
make fmt               # Format code
make lint              # Run linter
```

### Testing

```bash
make test              # Run all tests
make test-coverage     # Generate coverage report
go test ./pkg/...      # Run unit tests only
```

### Running Services

```bash
# Infrastructure
docker-compose up -d   # Start PostgreSQL + Redis
docker-compose down    # Stop PostgreSQL + Redis

# Worker
sudo ./bin/worker      # Start worker

# API Gateway
./bin/api-gateway      # Start API server
```

### VM Management

```bash
# Create
./bin/aether-cli -type vm:create -name my-vm

# List
./bin/aether-cli -type vm:list

# Execute
./bin/aether-cli -type vm:execute -vm-id <id> -cmd ls

# Delete
./bin/aether-cli -type vm:delete -vm-id <id>
```

---

## Performance Characteristics

- **VM Creation**: ~10-15 seconds (with tool installation)
- **Command Execution**: <1 second (local vsock)
- **Task Latency**: <500ms (Redis queue overhead)
- **Concurrent VMs**: Limited by host CPU/memory
- **Network Throughput**: ~1 Gbps (TAP bridge)

---

## Security Considerations

1. **VM Isolation**: Hardware-level via KVM
2. **Network Isolation**: Bridge + iptables
3. **Vsock**: No network exposure
4. **Sudo Required**: Worker needs network privileges
5. **Credentials**: Use environment variables, never commit secrets
6. **API Authentication**: Implement JWT or API keys (in progress)

---

## Next Steps

1. ✅ Run one-time setup (steps 1-4)
2. ✅ Start infrastructure (docker-compose)
3. ✅ Start worker (daily)
4. ✅ Start API Gateway (optional)
5. ✅ Create and manage VMs
6. 📖 Read specific guides in `docs/` folder
7. 🧪 Run integration tests: `cd tests/integration && sudo go test -v`

---

**Last Updated:** December 12, 2025  
**Status:** Ready for production use
