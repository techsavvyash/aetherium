# Production Architecture

This document describes the production-ready architecture of Aetherium.

## Overview

Aetherium has been refactored into a production-ready distributed task execution system with clean separation of concerns:

```
┌─────────────────┐      ┌─────────────────┐
│  aether-cli     │──────▶│   Redis Queue   │
│  (Task Submit)  │      └─────────────────┘
└─────────────────┘               │
                                  ▼
                         ┌─────────────────┐
                         │     Worker      │
                         │   (Processor)   │
                         └─────────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
           ┌─────────────┐ ┌─────────┐ ┌──────────────┐
           │ Firecracker │ │  Vsock  │ │  PostgreSQL  │
           │     VMM     │ │  Agent  │ │   Storage    │
           └─────────────┘ └─────────┘ └──────────────┘
```

## Package Structure

### Core Packages (`pkg/`)

**`pkg/service/`** - Service Layer
- `TaskService` - High-level task submission and querying API
- Abstracts away queue and storage details
- Used by CLI, API Gateway, and integrations

**`pkg/worker/`** - Worker Package
- `Worker` - Task processor implementation
- Registers handlers with queue
- Orchestrates VMs and executes commands
- Reusable across different deployment configurations

**`pkg/queue/`** - Queue Abstraction
- Interface: `Queue`
- Implementations: `pkg/queue/asynq/` (Redis-backed)
- Task enqueueing, handler registration, processing

**`pkg/storage/`** - Storage Abstraction
- Interface: `Store`, `VMRepository`, `ExecutionRepository`, etc.
- Implementations: `pkg/storage/postgres/`
- State persistence for VMs, tasks, executions

**`pkg/vmm/`** - VMM Orchestration
- Interface: `VMOrchestrator`
- Implementations: `pkg/vmm/firecracker/`, `pkg/vmm/docker/`
- VM lifecycle, command execution via vsock

### Commands (`cmd/`)

**`cmd/worker/`** - Worker Daemon
- Standalone worker process
- Consumes tasks from Redis queue
- Manages Firecracker VMs
- Stores results in PostgreSQL

**`cmd/aether-cli/`** - Task Submission CLI
- Submit tasks to queue
- Query VM and execution state
- Production-ready CLI tool

### Tests (`tests/`)

**`tests/integration/`** - Integration Tests
- End-to-end testing of distributed execution
- `TestThreeVMClone` - Demonstrates 3 VMs cloning repos in parallel
- Uses real infrastructure (PostgreSQL, Redis, Firecracker)

## Usage

### Starting the Worker

```bash
# Build the worker
go build -o bin/worker ./cmd/worker

# Run the worker (requires sudo for Firecracker)
sudo ./bin/worker
```

The worker:
- Connects to PostgreSQL (localhost:5432)
- Connects to Redis (localhost:6379)
- Initializes Firecracker orchestrator
- Registers handlers for vm:create, vm:execute, vm:delete
- Starts processing tasks from the queue

### Submitting Tasks

```bash
# Build the CLI
go build -o bin/aether-cli ./cmd/aether-cli

# Create a VM
./bin/aether-cli -type vm:create -name my-vm -vcpus 1 -memory 256

# Execute a command (get VM ID from database or logs)
./bin/aether-cli -type vm:execute -vm-id <uuid> -cmd echo -args "Hello World"

# Delete a VM
./bin/aether-cli -type vm:delete -vm-id <uuid>
```

### Running Integration Tests

```bash
# Quick setup and run (requires sudo)
sudo ./scripts/run-integration-test.sh
```

This will:
1. Check prerequisites (Docker, Firecracker, kernel, rootfs)
2. Start PostgreSQL and Redis containers
3. Run database migrations
4. Build and run the integration test
5. Display results

See `tests/integration/README.md` for detailed testing documentation.

## Service Layer API

The `TaskService` provides a clean API for task operations:

```go
import (
    "context"
    "github.com/aetherium/aetherium/pkg/service"
    "github.com/aetherium/aetherium/pkg/queue/asynq"
    "github.com/aetherium/aetherium/pkg/storage/postgres"
)

// Initialize infrastructure
store, _ := postgres.NewStore(postgres.Config{...})
queue, _ := asynq.NewQueue(asynq.Config{...})

// Create service
taskService := service.NewTaskService(queue, store)

// Create a VM
taskID, err := taskService.CreateVMTask(ctx, "my-vm", 1, 256)

// Execute command
taskID, err := taskService.ExecuteCommandTask(ctx, vmID, "git", []string{"clone", "https://..."})

// Query VMs
vm, err := taskService.GetVMByName(ctx, "my-vm")
vms, err := taskService.ListVMs(ctx)

// Get execution history
executions, err := taskService.GetExecutions(ctx, vmID)
```

## Worker API

The `Worker` package is reusable and can be embedded in different applications:

```go
import (
    "github.com/aetherium/aetherium/pkg/worker"
    "github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

// Create worker
orchestrator, _ := firecracker.NewFirecrackerOrchestrator(...)
w := worker.New(store, orchestrator)

// Register handlers
w.RegisterHandlers(queue)

// Start processing (handled by queue.Start)
queue.Start(ctx)
```

## Configuration

### Worker Configuration

Currently hardcoded in `cmd/worker/main.go`. Future versions will support:

```yaml
# config/worker.yaml
postgres:
  host: localhost
  port: 5432
  database: aetherium

redis:
  addr: localhost:6379

firecracker:
  kernel_path: /var/firecracker/vmlinux
  rootfs_template: /var/firecracker/rootfs.ext4
  socket_dir: /tmp
  default_vcpu: 1
  default_memory_mb: 256
```

### CLI Configuration

Environment variables:
- `POSTGRES_HOST` - PostgreSQL host (default: localhost)
- `POSTGRES_PORT` - PostgreSQL port (default: 5432)
- `REDIS_ADDR` - Redis address (default: localhost:6379)

## Task Types

### `vm:create`

Creates and starts a new Firecracker VM.

**Payload:**
```json
{
  "name": "my-vm",
  "vcpus": 1,
  "memory_mb": 256
}
```

**Result:**
```json
{
  "vm_id": "uuid",
  "name": "my-vm",
  "status": "running"
}
```

**Options:**
- `MaxRetry: 3`
- `Timeout: 5 minutes`
- `Queue: default`
- `Priority: 5`

### `vm:execute`

Executes a command in a running VM via vsock.

**Payload:**
```json
{
  "vm_id": "uuid",
  "command": "git",
  "args": ["clone", "https://github.com/user/repo"]
}
```

**Result:**
```json
{
  "vm_id": "uuid",
  "exit_code": 0,
  "stdout": "Cloning into 'repo'...",
  "stderr": ""
}
```

**Options:**
- `MaxRetry: 2`
- `Timeout: 10 minutes`
- `Queue: default`
- `Priority: 5`

### `vm:delete`

Stops and deletes a VM.

**Payload:**
```json
{
  "vm_id": "uuid"
}
```

**Result:**
```json
{
  "vm_id": "uuid"
}
```

**Options:**
- `MaxRetry: 2`
- `Timeout: 2 minutes`
- `Queue: default`
- `Priority: 5`

## Database Schema

See `DEMO.md` for detailed schema documentation.

Key tables:
- `vms` - VM state and configuration
- `tasks` - Task queue state (managed by Asynq)
- `executions` - Command execution history
- `jobs` - Future: Multi-step workflow definitions

## Architecture Principles

1. **Interface-Driven Design**: All major components (queue, storage, VMM) are interfaces
2. **Dependency Injection**: Components receive their dependencies
3. **Separation of Concerns**: Service, worker, storage, and orchestration are separate
4. **Testability**: Integration tests in separate directory
5. **Production-Ready**: No hardcoded demos in main codebase

## Extending the System

### Adding a New Task Type

1. Define task type in `pkg/queue/queue.go`:
   ```go
   const TaskTypeMyTask TaskType = "my:task"
   ```

2. Create payload struct in `pkg/worker/worker.go`:
   ```go
   type MyTaskPayload struct {
       Field string `json:"field"`
   }
   ```

3. Implement handler in `pkg/worker/worker.go`:
   ```go
   func (w *Worker) HandleMyTask(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
       // Implementation
   }
   ```

4. Register handler in `RegisterHandlers`:
   ```go
   q.RegisterHandler(queue.TaskTypeMyTask, w.HandleMyTask)
   ```

5. Add service method in `pkg/service/task_service.go`:
   ```go
   func (s *TaskService) CreateMyTask(ctx context.Context, ...) (uuid.UUID, error) {
       // Implementation
   }
   ```

### Adding a New Storage Backend

1. Implement `storage.Store` interface
2. Implement repository interfaces (`VMRepository`, `ExecutionRepository`, etc.)
3. Update worker and CLI to support new backend

### Adding a New VMM Backend

1. Implement `vmm.VMOrchestrator` interface
2. Update worker configuration to support new backend

## Monitoring

### Database Queries

```bash
# View VMs
docker exec -it aetherium-postgres psql -U aetherium -c \
  "SELECT id, name, status, created_at FROM vms ORDER BY created_at DESC;"

# View recent executions
docker exec -it aetherium-postgres psql -U aetherium -c \
  "SELECT vm_id, command, exit_code, started_at FROM executions ORDER BY started_at DESC LIMIT 20;"

# View failed executions
docker exec -it aetherium-postgres psql -U aetherium -c \
  "SELECT * FROM executions WHERE exit_code != 0;"
```

### Redis Queue Stats

```bash
# Redis CLI
docker exec -it aetherium-redis redis-cli

# View queue info
> INFO
> KEYS asynq:*
> LLEN asynq:queues:default
```

## Future Enhancements

See main README for roadmap:

- **API Gateway**: REST/GraphQL API for task submission
- **Job Orchestration**: Multi-step workflows (DAGs)
- **Integrations**: GitHub, Slack, Discord plugins
- **Observability**: Loki logging, Prometheus metrics
- **Security**: mTLS, authentication, authorization
- **Scaling**: Multiple workers, queue sharding

## Migration from Demo Code

Old demo code locations:
- `cmd/worker-demo/` → Removed, functionality in `pkg/worker/` + `cmd/worker/`
- `cmd/task-submit/` → Replaced by `cmd/aether-cli/`
- `DEMO.md` → Still valid, documents the distributed system architecture

The demo functionality is now available as integration tests in `tests/integration/`.
