# Aetherium Core Service

**VM provisioning and lifecycle management using Firecracker microVMs.**

Core service is the heart of Aetherium—managing VM orchestration, command execution, networking, and all supporting infrastructure.

## Features

- **VM Orchestration**: Firecracker microVMs with Docker fallback for testing
- **Command Execution**: Execute commands in VMs via vsock
- **Network Management**: TAP devices, bridge setup, NAT configuration
- **Tool Installation**: Automatic tool provisioning with mise
- **Database Persistence**: PostgreSQL state storage
- **Task Queue**: Redis/Asynq task distribution
- **Worker Daemon**: Scalable task processing

## Building

```bash
# From root
cd services/core
make build

# Binaries output to ../../bin/
ls ../../bin/
  worker          # Task worker daemon
  fc-agent        # Agent running inside VM
  aether-cli      # CLI tool
  migrate         # Database migrations
```

## Testing

```bash
# Run all tests
make test

# With coverage
make test-coverage
```

## Running

### Local Development

```bash
# Start infrastructure (PostgreSQL, Redis)
make docker-up

# Run migrations
make migrate-up

# Start worker (requires sudo for Firecracker/network)
sudo make run-worker
```

### Prerequisites

- Go 1.25+
- PostgreSQL 15+
- Redis 7+
- Firecracker binary
- Linux kernel with vsock support

## Architecture

### Packages

- **pkg/vmm/** - VM orchestration (Firecracker, Docker)
- **pkg/network/** - Network setup (TAP, bridge, NAT)
- **pkg/storage/** - Database layer (PostgreSQL)
- **pkg/queue/** - Task queue abstractions
- **pkg/tools/** - Tool installation (mise)
- **pkg/service/** - Public API (TaskService)
- **pkg/worker/** - Task handlers
- **pkg/types/** - Domain types (VM, Task, Execution)

### Key Components

```
TaskService (Public API)
  ├─ CreateVMTask()
  ├─ ExecuteCommandTask()
  ├─ DeleteVMTask()
  └─ GetVMStatus()
         ↓
    Worker Queue (Asynq/Redis)
         ↓
    Worker Process
         ├─ HandleVMCreate → VMOrchestrator
         ├─ HandleVMExecute → VSock
         └─ HandleVMDelete → Cleanup
         ↓
    Persistent Storage (PostgreSQL)
```

## API

### TaskService Interface

```go
type TaskService interface {
    CreateVMTask(ctx, name, vcpus, memoryMB) (taskID, error)
    ExecuteCommandTask(ctx, vmID, command, args) (taskID, error)
    DeleteVMTask(ctx, vmID) (taskID, error)
    GetVM(ctx, vmID) (*VM, error)
    ListVMs(ctx) ([]*VM, error)
    GetExecutions(ctx, vmID) ([]*Execution, error)
}
```

### VMOrchestrator Interface

```go
type VMOrchestrator interface {
    CreateVM(ctx, config) (*VM, error)
    StartVM(ctx, vmID) error
    StopVM(ctx, vmID, force bool) error
    DeleteVM(ctx, vmID) error
    ExecuteCommand(ctx, vmID, cmd) (*ExecResult, error)
    GetVMStatus(ctx, vmID) (*VM, error)
    ListVMs(ctx) ([]*VM, error)
    Health(ctx) error
}
```

## Database Schema

See `migrations/` for SQL schema definitions.

### Key Tables

- `vms` - Virtual machine records
- `tasks` - Task queue entries
- `executions` - Command execution history
- `jobs` - Multi-command job groups

## Configuration

```yaml
vmm:
  provider: firecracker  # or: docker
  firecracker:
    kernel_path: /var/firecracker/vmlinux
    rootfs_path: /var/firecracker/rootfs.ext4
    socket_dir: /tmp

queue:
  provider: asynq
  redis_addr: localhost:6379

storage:
  provider: postgres
  host: localhost
  port: 5432
  database: aetherium
```

## Exports to Gateway

This service exports the following to `services/gateway`:

- `TaskService` interface for REST API wrapping
- `VMOrchestrator` interface for orchestration
- Domain types (VM, Task, Execution)

See `services/gateway/` for how it's used.

## Development

### Adding a New Task Type

1. Define task type in `pkg/queue/queue.go`
2. Create handler in `pkg/worker/worker.go`
3. Register handler in worker initialization
4. Add service method in `pkg/service/task_service.go`

### Adding a New Integration

Use the VSock interface in `pkg/vmm/firecracker/exec.go` to communicate with the VM agent.

## Troubleshooting

### Worker can't create VMs

```
Error: Permission denied (network operations)
```

Solution: Worker needs network privileges
```bash
# Option 1: Run with sudo
sudo ./bin/worker

# Option 2: Grant capability
sudo setcap cap_net_admin+ep ./bin/worker

# Option 3: Setup network upfront
sudo scripts/setup-network.sh
./bin/worker
```

### VSock connection timeout

```
Error: Cannot connect to VM agent via vsock
```

Check:
1. Kernel has vsock support: `lsmod | grep vhost_vsock`
2. Agent is running in VM: `ss -lnp | grep 9999` (from inside VM)
3. Firecracker binary path is correct

### Database connection failed

```
Error: failed to connect to database
```

Check:
1. PostgreSQL is running: `docker ps | grep postgres`
2. Connection string is correct
3. Database exists: `createdb aetherium`
4. Migrations have run: `make migrate-up`

## Next Steps

- See [../../docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md) for system design
- See [../../docs/DEVELOPMENT.md](../../docs/DEVELOPMENT.md) for development guide
- See [../../MONOREPO_QUICK_REFERENCE.md](../../MONOREPO_QUICK_REFERENCE.md) for quick lookup
