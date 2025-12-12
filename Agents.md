# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Last Updated:** 2025-11-01

## What is Aetherium?

Aetherium is a **distributed task execution platform** that runs isolated workloads in **Firecracker microVMs** or Docker containers. Think of it as a programmable VM orchestration engine with:

- Async task queue (Redis/Asynq)
- VM lifecycle management (create, execute, delete)
- Command execution via vsock (host-VM communication)
- PostgreSQL state persistence
- Tool auto-installation (nodejs, bun, claude-code, go, python, etc.)
- REST API Gateway
- Integration framework (GitHub, Slack)
- Centralized logging (Loki)

## Common Commands

### Build
```bash
make build              # Build all binaries to bin/
make clean              # Clean build artifacts
make deps               # Download and tidy Go modules
```

### Testing
```bash
make test               # Run all tests with race detection
make test-coverage      # Generate coverage report (coverage.html)
go test ./pkg/...       # Run unit tests only
cd tests/integration && sudo go test -v -timeout 30m  # Integration tests (needs sudo)
```

### Development
```bash
make fmt                # Format Go code
make lint               # Run golangci-lint
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/worker ./cmd/worker
go build -o bin/aether-cli ./cmd/aether-cli
```

### Running Services
```bash
# Start infrastructure (PostgreSQL + Redis)
docker run -d --name aetherium-postgres \
  -e POSTGRES_USER=aetherium -e POSTGRES_PASSWORD=aetherium -e POSTGRES_DB=aetherium \
  -p 5432:5432 postgres:15-alpine

docker run -d --name aetherium-redis -p 6379:6379 redis:7-alpine

# Run database migrations
cd migrations
migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" -path . up

# One-time: Prepare Firecracker rootfs with tools
sudo ./scripts/prepare-rootfs-with-tools.sh

# Start worker (needs sudo for Firecracker/network)
sudo ./bin/worker

# Start API Gateway (separate terminal)
./bin/api-gateway

# Submit task via CLI
./bin/aether-cli -type vm:create -name my-vm -vcpus 2 -memory 2048
```


## High-Level Architecture

```
┌──────────────┐
│   Client     │
│ (REST/CLI)   │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│           API Gateway (port 8080)               │
│  - REST endpoints for VM/task operations        │
│  - Integration webhooks                         │
└──────┬──────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│             Redis Queue (Asynq)                 │
│  - Task distribution                            │
│  - Priority queues (critical/high/default/low)  │
└──────┬──────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│                Worker Process                    │
│  - Pulls tasks from queue                       │
│  - Delegates to VMM orchestrator                │
│  - Stores results in PostgreSQL                 │
└──────┬──────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────┐
│        Firecracker Orchestrator                 │
│  - VM lifecycle (create/start/stop/delete)      │
│  - Command execution via vsock                  │
│  - Network setup (TAP devices + bridge)         │
└─────────────────────────────────────────────────┘
```

## Task Flow: API Gateway → Queue → Worker → VM

### 1. User submits task via API or CLI

```bash
# Via CLI
./bin/aether-cli -type vm:create -name my-vm -vcpus 2 -memory 2048

# Via REST API
POST /api/v1/vms
{
  "name": "my-vm",
  "vcpus": 2,
  "memory_mb": 2048,
  "additional_tools": ["go", "python"]
}
```

### 2. API Gateway enqueues task

File: `/cmd/api-gateway/main.go`

```go
taskID, err := taskService.CreateVMTaskWithTools(ctx, name, vcpus, memoryMB, tools, versions)
```

### 3. TaskService creates task object

File: `/pkg/service/task_service.go`

```go
task := &queue.Task{
    ID:   uuid.New(),
    Type: queue.TaskTypeVMCreate,
    Payload: map[string]interface{}{
        "name": name,
        "vcpus": vcpus,
        "memory_mb": memoryMB,
        "additional_tools": tools,
        "tool_versions": versions,
    },
}
queue.Enqueue(ctx, task, &queue.TaskOptions{...})
```

### 4. Asynq queue persists task in Redis

File: `/pkg/queue/asynq/asynq.go`

- Task serialized to JSON
- Stored in Redis with priority/timeout metadata
- Available for worker consumption

### 5. Worker pulls and processes task

File: `/pkg/worker/worker.go`

```go
// Worker has registered handlers
worker.RegisterHandlers(queue)
  -> queue.RegisterHandler(TaskTypeVMCreate, worker.HandleVMCreate)
  -> queue.RegisterHandler(TaskTypeVMExecute, worker.HandleVMExecute)
  -> queue.RegisterHandler(TaskTypeVMDelete, worker.HandleVMDelete)
```

### 6. HandleVMCreate executes

```go
func (w *Worker) HandleVMCreate(ctx, task) (*TaskResult, error) {
    // 1. Create VM via orchestrator
    vm, err := w.orchestrator.CreateVM(ctx, vmConfig)
    
    // 2. Start VM
    w.orchestrator.StartVM(ctx, vm.ID)
    
    // 3. Wait for boot
    time.Sleep(5 * time.Second)
    
    // 4. Install tools (default + additional)
    w.toolInstaller.InstallToolsWithTimeout(ctx, vm.ID, tools, versions, 20*time.Minute)
    
    // 5. Store VM in PostgreSQL
    w.store.VMs().Create(ctx, dbVM)
    
    return &TaskResult{Success: true, Result: {...}}
}
```

### 7. Firecracker Orchestrator manages VM

File: `/pkg/vmm/firecracker/firecracker.go`

```go
func (f *FirecrackerOrchestrator) CreateVM(ctx, config) (*VM, error) {
    // 1. Create TAP network device
    tapDevice := f.networkManager.CreateTAPDevice(vmID)
    
    // 2. Build Firecracker config
    fcConfig := firecracker.Config{
        KernelImagePath: config.KernelPath,
        Drives: [...], // rootfs
        MachineCfg: {VcpuCount, MemSizeMib},
        NetworkInterfaces: [tapDevice],
        VsockDevices: [{CID: 3}], // For host-VM communication
    }
    
    // 3. Create machine via official SDK
    machine, err := firecracker.NewMachine(ctx, fcConfig)
    
    return vm
}
```

### 8. Command Execution via vsock

File: `/pkg/vmm/firecracker/exec.go`

```go
func (f *FirecrackerOrchestrator) ExecuteCommand(ctx, vmID, cmd) (*ExecResult, error) {
    // 1. Connect to VM agent via vsock (CID 3, port 9999)
    conn, err := vsock.Dial(3, 9999, nil)
    
    // 2. Send command JSON
    json.NewEncoder(conn).Encode(cmd)
    
    // 3. Receive result JSON
    json.NewDecoder(conn).Decode(&result)
    
    return &ExecResult{ExitCode, Stdout, Stderr}
}
```

## Key Abstractions & Interfaces

### 1. VMOrchestrator Interface

File: `/pkg/vmm/interface.go`

Defines contract for VM management:

```go
type VMOrchestrator interface {
    CreateVM(ctx, config) (*VM, error)
    StartVM(ctx, vmID) error
    StopVM(ctx, vmID, force bool) error
    GetVMStatus(ctx, vmID) (*VM, error)
    ExecuteCommand(ctx, vmID, cmd) (*ExecResult, error)
    DeleteVM(ctx, vmID) error
    ListVMs(ctx) ([]*VM, error)
    Health(ctx) error
}
```

Implementations:
- `/pkg/vmm/firecracker/firecracker.go` - Firecracker orchestrator (production)
- `/pkg/vmm/docker/docker.go` - Docker orchestrator (testing)

### 2. Queue Interface

File: `/pkg/queue/queue.go`

Defines contract for task queue:

```go
type Queue interface {
    Enqueue(ctx, task, opts) error
    RegisterHandler(taskType, handler) error
    Start(ctx) error
    Stop(ctx) error
    Stats(ctx) (*QueueStats, error)
}

type TaskHandler func(ctx context.Context, task *Task) (*TaskResult, error)
```

Implementation:
- `/pkg/queue/asynq/asynq.go` - Redis-backed queue using Asynq library

Task types:
- `vm:create` - Create and start VM
- `vm:execute` - Execute command in VM
- `vm:delete` - Stop and delete VM

### 3. Storage Interface

File: `/pkg/storage/interface.go`

Old interface (deprecated, replaced by repository pattern):

```go
type StateStore interface {
    CreateProject/GetProject/UpdateProject/DeleteProject/ListProjects
    CreateTask/GetTask/UpdateTask/DeleteTask/ListTasks
    CreateVM/GetVM/UpdateVM/DeleteVM/ListVMs
    Health() error
    Close() error
}
```

New repository pattern (current):

File: `/pkg/storage/storage.go`

```go
type Store interface {
    VMs() VMRepository
    Tasks() TaskRepository
    Jobs() JobRepository
    Executions() ExecutionRepository
    Close() error
}

type VMRepository interface {
    Create(ctx, vm) error
    Get(ctx, id) (*VM, error)
    GetByName(ctx, name) (*VM, error)
    List(ctx, filters) ([]*VM, error)
    Update(ctx, vm) error
    Delete(ctx, id) error
}
```

Implementation:
- `/pkg/storage/postgres/postgres.go` - PostgreSQL store
- `/pkg/storage/postgres/vms.go` - VM repository
- `/pkg/storage/postgres/executions.go` - Execution repository

### 4. Logging Interface

File: `/pkg/logging/interface.go`

```go
type Logger interface {
    Log(ctx, entry) error
    Query(ctx, filter) ([]*LogEntry, error)
    Close() error
    Health(ctx) error
}
```

Implementation:
- `/pkg/logging/loki/loki.go` - Grafana Loki logger (batched)
- `/pkg/logging/stdout/stdout.go` - Console logger

### 5. EventBus Interface

File: `/pkg/events/interface.go`

```go
type EventBus interface {
    Publish(ctx, topic, event) error
    Subscribe(ctx, topic, handler) (Subscription, error)
    Close() error
    Health(ctx) error
}
```

Implementation:
- `/pkg/events/redis/redis.go` - Redis pub/sub

### 6. Integration Interface

File: `/pkg/integrations/interface.go`

```go
type Integration interface {
    Name() string
    Initialize(ctx, config) error
    SendNotification(ctx, notification) error
    CreateArtifact(ctx, artifact) error
    HandleEvent(ctx, event) error
    Health(ctx) error
    Close() error
}
```

Implementations:
- `/pkg/integrations/github/github.go` - GitHub integration (PRs, webhooks)
- `/pkg/integrations/slack/slack.go` - Slack integration (messages, slash commands)

## Service Layer

File: `/pkg/service/task_service.go`

High-level API that abstracts queue and storage:

```go
type TaskService struct {
    queue queue.Queue
    store storage.Store
}

// Public methods
CreateVMTask(ctx, name, vcpus, memoryMB) (taskID, error)
CreateVMTaskWithTools(ctx, name, vcpus, memoryMB, tools, versions) (taskID, error)
ExecuteCommandTask(ctx, vmID, command, args) (taskID, error)
DeleteVMTask(ctx, vmID) (taskID, error)
GetVM(ctx, vmID) (*VM, error)
GetVMByName(ctx, name) (*VM, error)
ListVMs(ctx) ([]*VM, error)
GetExecutions(ctx, vmID) ([]*Execution, error)
```

Used by:
- API Gateway (`/cmd/api-gateway/main.go`)
- CLI (`/cmd/aether-cli/main.go`)

## Worker System

File: `/pkg/worker/worker.go`

```go
type Worker struct {
    store        storage.Store
    orchestrator vmm.VMOrchestrator
    toolInstaller *tools.Installer
}

// Task handlers
HandleVMCreate(ctx, task) (*TaskResult, error)
HandleVMExecute(ctx, task) (*TaskResult, error)
HandleVMDelete(ctx, task) (*TaskResult, error)
```

Lifecycle:
1. Worker created with dependencies (store, orchestrator)
2. Handlers registered with queue: `worker.RegisterHandlers(queue)`
3. Queue started: `queue.Start(ctx)`
4. Worker processes tasks in background goroutines

## Database Schema

File: `/migrations/000001_initial_schema.up.sql`

### VMs Table

```sql
CREATE TABLE vms (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    orchestrator VARCHAR(50) NOT NULL, -- 'firecracker' or 'docker'
    status VARCHAR(50) NOT NULL, -- 'created', 'running', 'stopped', etc.
    
    -- Config
    kernel_path VARCHAR(500),
    rootfs_path VARCHAR(500),
    socket_path VARCHAR(500),
    vcpu_count INTEGER,
    memory_mb INTEGER,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    stopped_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);
```

### Tasks Table

```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    type VARCHAR(100) NOT NULL, -- 'vm:create', 'vm:execute', etc.
    status VARCHAR(50) DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    
    payload JSONB NOT NULL,
    result JSONB,
    error TEXT,
    
    vm_id UUID REFERENCES vms(id),
    worker_id VARCHAR(255),
    max_retries INTEGER DEFAULT 3,
    retry_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);
```

### Executions Table

```sql
CREATE TABLE executions (
    id UUID PRIMARY KEY,
    job_id UUID REFERENCES jobs(id),
    vm_id UUID REFERENCES vms(id),
    
    command VARCHAR(500) NOT NULL,
    args JSONB,
    env JSONB,
    
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    error TEXT,
    
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    metadata JSONB DEFAULT '{}'
);
```

### Jobs Table

```sql
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    
    vm_id UUID REFERENCES vms(id),
    commands JSONB NOT NULL, -- Array of commands
    
    results JSONB,
    error TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);
```

## Design Patterns

### 1. Factory Pattern

File: `/pkg/container/factory.go`

Used for creating pluggable components based on configuration:

```go
type VMOrchestratorFactory interface {
    Create(ctx, provider, config) (vmm.VMOrchestrator, error)
}
```

Implementations:
- `/pkg/container/factories/vmm.go` - Creates Firecracker or Docker orchestrator
- `/pkg/container/factories/queue.go` - Creates Asynq queue
- `/pkg/container/factories/storage.go` - Creates PostgreSQL store

### 2. Dependency Injection Container

File: `/pkg/container/container.go`

Manages component lifecycle and dependencies:

```go
type Container struct {
    config *config.Config
    
    // Singletons
    taskQueue    queue.TaskQueue
    stateStore   storage.StateStore
    logger       logging.Logger
    vmOrch       vmm.VMOrchestrator
    eventBus     events.EventBus
    integrations map[string]integrations.Integration
    
    // Factories
    taskQueueFactory TaskQueueFactory
    stateStoreFactory StateStoreFactory
    // ...
}

// Usage
container := New(config)
container.RegisterVMOrchestratorFactory(factory)
container.Initialize(ctx)
orch := container.GetVMOrchestrator()
```

**Note:** The DI container exists but is not actively used in production code. The current approach uses direct initialization in `main.go` files.

### 3. Repository Pattern

File: `/pkg/storage/postgres/*.go`

Separates data access logic from business logic:

```go
type VMRepository interface {
    Create/Get/Update/Delete/List
}

// Implementation
type vmRepository struct {
    db *sqlx.DB
}

func (r *vmRepository) Create(ctx, vm) error {
    query := `INSERT INTO vms (...) VALUES (...)`
    _, err := r.db.ExecContext(ctx, query, ...)
    return err
}
```

### 4. Interface-Driven Design

All major components are defined as interfaces:
- `vmm.VMOrchestrator` - Swap Firecracker for Docker/QEMU
- `queue.Queue` - Swap Asynq for RabbitMQ/Kafka
- `storage.Store` - Swap PostgreSQL for MySQL/MongoDB
- `logging.Logger` - Swap Loki for ELK/CloudWatch

Configuration-driven switching:

```yaml
vmm:
  provider: firecracker  # or: docker, qemu
  
queue:
  provider: asynq  # or: rabbitmq, kafka

storage:
  provider: postgres  # or: mysql, mongodb

logging:
  provider: loki  # or: elasticsearch, stdout
```

## Network Infrastructure

### TAP Device Architecture

File: `/pkg/network/network.go`

Each VM gets its own TAP device connected to a bridge:

```
┌─────────────────────────────────────┐
│           Host System               │
│                                     │
│  ┌─────────────────────────────┐   │
│  │   Bridge: aetherium0        │   │
│  │   IP: 172.16.0.1/24         │   │
│  └───┬─────────┬─────────┬─────┘   │
│      │         │         │         │
│  ┌───▼───┐ ┌──▼────┐ ┌──▼────┐    │
│  │ TAP1  │ │ TAP2  │ │ TAP3  │    │
│  └───┬───┘ └───┬───┘ └───┬───┘    │
└──────┼─────────┼─────────┼─────────┘
       │         │         │
   ┌───▼───┐ ┌──▼────┐ ┌──▼────┐
   │ VM 1  │ │ VM 2  │ │ VM 3  │
   │ eth0  │ │ eth0  │ │ eth0  │
   │.16.0.2│ │.16.0.3│ │.16.0.4│
   └───────┘ └───────┘ └───────┘
```

Network Manager responsibilities:
1. Create bridge interface (`aetherium0`)
2. Setup NAT for internet access
3. Allocate IPs from subnet (172.16.0.0/24)
4. Create TAP device per VM
5. Attach TAP to bridge
6. Generate unique MAC addresses

### Vsock Communication

File: `/pkg/vmm/firecracker/exec.go`

Vsock = virtio-vsock, direct host-VM socket communication:

```
Host                             Guest VM
────                             ────────
                                 
vsock.Dial(3, 9999) ────────────▶ vsock.Listen(9999)
                                 fc-agent listening
                                 
Send JSON command   ────────────▶ Parse command
                                 exec.Command(cmd, args)
                                 
                    ◀──────────── Send JSON result
Receive result                   {ExitCode, Stdout, Stderr}
```

Constants:
- Host CID: 2 (always)
- Guest CID: 3 (configured in Firecracker)
- Agent Port: 9999

Vsock advantages:
- No network configuration needed
- Lower latency than TCP
- Isolated from network attacks

## API Gateway Routing

File: `/cmd/api-gateway/main.go`

REST API structure:

```
/api/v1
├── /vms
│   ├── POST   /                    Create VM (returns taskID)
│   ├── GET    /                    List all VMs
│   ├── GET    /{id}                Get VM by ID
│   ├── DELETE /{id}                Delete VM (returns taskID)
│   ├── POST   /{id}/execute        Execute command (returns taskID)
│   └── GET    /{id}/executions     List command history
├── /tasks
│   └── GET    /{id}                Get task status (TODO)
├── /logs
│   ├── POST   /query               Query logs (Loki)
│   └── GET    /stream              WebSocket log stream (TODO)
├── /webhooks
│   └── POST   /{integration}       Webhook endpoint (GitHub/Slack)
└── /health                         Health check
```

### Example Requests

Create VM:
```bash
POST /api/v1/vms
{
  "name": "dev-vm",
  "vcpus": 2,
  "memory_mb": 2048,
  "additional_tools": ["go", "python"],
  "tool_versions": {"go": "1.23.0"}
}

Response: 202 Accepted
{
  "task_id": "uuid",
  "status": "pending"
}
```

Execute Command:
```bash
POST /api/v1/vms/{vm-id}/execute
{
  "command": "git",
  "args": ["clone", "https://github.com/user/repo"]
}

Response: 202 Accepted
{
  "task_id": "uuid",
  "vm_id": "uuid",
  "status": "pending"
}
```

List VMs:
```bash
GET /api/v1/vms

Response: 200 OK
{
  "vms": [
    {
      "id": "uuid",
      "name": "dev-vm",
      "status": "running",
      "vcpu_count": 2,
      "memory_mb": 2048,
      "created_at": "2025-11-01T...",
      ...
    }
  ],
  "total": 1
}
```

## Important Patterns & Idioms

### 1. Context Propagation

All operations accept `context.Context`:

```go
func CreateVM(ctx context.Context, config *VMConfig) (*VM, error)
```

Allows:
- Cancellation propagation
- Deadline enforcement
- Request-scoped values

### 2. Error Wrapping

Use `fmt.Errorf` with `%w` for error chains:

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### 3. Pointer Receivers

All methods use pointer receivers:

```go
func (w *Worker) HandleVMCreate(ctx, task) (*TaskResult, error)
```

### 4. Graceful Shutdown

All services implement graceful shutdown:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
server.Shutdown(ctx)
```

### 5. UUID for IDs

All entities use UUIDs:

```go
import "github.com/google/uuid"

vmID := uuid.New().String()
taskID := uuid.New()
```

### 6. JSONB Metadata

All tables have JSONB metadata column for extensibility:

```sql
metadata JSONB DEFAULT '{}'
```

## Common Workflows

### Creating a VM with Tools

1. Client sends POST to `/api/v1/vms`
2. API Gateway calls `TaskService.CreateVMTaskWithTools`
3. TaskService enqueues `vm:create` task with payload
4. Worker picks up task
5. Worker calls `orchestrator.CreateVM`:
   - Creates TAP device
   - Builds Firecracker config
   - Creates VM (not started)
6. Worker calls `orchestrator.StartVM`:
   - Starts Firecracker process
   - VM boots
7. Worker waits 5s for boot
8. Worker calls `toolInstaller.InstallToolsWithTimeout`:
   - Installs default tools (git, nodejs, bun, claude-code)
   - Installs additional tools (go, python, etc.)
   - Each tool installed via `ExecuteCommand`
9. Worker stores VM in PostgreSQL
10. Worker returns success result

### Executing Command in VM

1. Client sends POST to `/api/v1/vms/{id}/execute`
2. API Gateway calls `TaskService.ExecuteCommandTask`
3. TaskService enqueues `vm:execute` task
4. Worker picks up task
5. Worker calls `orchestrator.ExecuteCommand`:
   - Connects to VM via vsock
   - Sends JSON command
   - Receives JSON result
6. Worker stores execution in PostgreSQL
7. Worker returns result (exit code, stdout, stderr)

### Deleting a VM

1. Client sends DELETE to `/api/v1/vms/{id}`
2. API Gateway calls `TaskService.DeleteVMTask`
3. TaskService enqueues `vm:delete` task
4. Worker picks up task
5. Worker calls `orchestrator.DeleteVM`:
   - Stops VM if running
   - Deletes TAP device
   - Cleans up sockets
6. Worker deletes VM from PostgreSQL
7. Worker returns success

## Tool Installation System

File: `/pkg/tools/installer.go`

Default tools (installed in all VMs):
- git
- nodejs@20
- bun@latest
- claude-code@latest

Additional tools (per-request):
- go
- python
- rust
- docker
- And more...

Installation method:
- Uses mise (version manager) in VM
- Executed via `ExecuteCommand`
- Supports version pinning: `go@1.23.0`

Example:
```go
tools := []string{"git", "nodejs", "bun", "go", "python"}
versions := map[string]string{
    "go": "1.23.0",
    "python": "3.11",
}

installer.InstallToolsWithTimeout(ctx, vmID, tools, versions, 20*time.Minute)
```

## Configuration

File: `/pkg/config/config.go`

YAML-based configuration:

```yaml
vmm:
  provider: firecracker
  firecracker:
    kernel_path: /var/firecracker/vmlinux
    rootfs_template: /var/firecracker/rootfs.ext4
    socket_dir: /tmp
    default_vcpu: 1
    default_memory_mb: 256

queue:
  provider: asynq
  asynq:
    redis_addr: localhost:6379

storage:
  provider: postgres
  postgres:
    host: localhost
    port: 5432
    database: aetherium

logging:
  provider: loki
  loki:
    url: http://localhost:3100

event_bus:
  provider: redis
  redis:
    addr: localhost:6379

integrations:
  enabled:
    - github
    - slack
  github:
    token: ${GITHUB_TOKEN}
  slack:
    bot_token: ${SLACK_BOT_TOKEN}
```

## File Structure Quick Reference

```
aetherium/
├── cmd/                           # Entry points
│   ├── api-gateway/main.go       # REST API server (port 8080)
│   ├── worker/main.go            # Task worker daemon
│   ├── aether-cli/main.go        # CLI for task submission
│   ├── fc-agent/main.go          # Agent running inside VM
│   └── migrate/main.go           # DB migration tool
│
├── pkg/                           # Packages
│   ├── vmm/                       # VM orchestration
│   │   ├── interface.go           # VMOrchestrator interface
│   │   ├── firecracker/
│   │   │   ├── firecracker.go     # Firecracker implementation
│   │   │   └── exec.go            # Command execution via vsock
│   │   └── docker/
│   │       └── docker.go          # Docker implementation
│   │
│   ├── queue/                     # Task queue
│   │   ├── interface.go           # Old interface (deprecated)
│   │   ├── queue.go               # Current interface + types
│   │   └── asynq/
│   │       └── asynq.go           # Redis-backed queue
│   │
│   ├── storage/                   # Persistence
│   │   ├── interface.go           # Old interface (deprecated)
│   │   ├── storage.go             # Repository interfaces
│   │   └── postgres/
│   │       ├── postgres.go        # Store implementation
│   │       ├── vms.go             # VM repository
│   │       ├── tasks.go           # Task repository
│   │       ├── jobs.go            # Job repository
│   │       └── executions.go      # Execution repository
│   │
│   ├── logging/                   # Logging
│   │   ├── interface.go           # Logger interface
│   │   ├── loki/
│   │   │   └── loki.go            # Loki implementation
│   │   └── stdout/
│   │       └── stdout.go          # Console logger
│   │
│   ├── events/                    # Event bus
│   │   ├── interface.go           # EventBus interface
│   │   └── redis/
│   │       └── redis.go           # Redis pub/sub
│   │
│   ├── integrations/              # Integration framework
│   │   ├── interface.go           # Integration interface
│   │   ├── registry.go            # Plugin registry
│   │   ├── github/
│   │   │   └── github.go          # GitHub integration
│   │   └── slack/
│   │       └── slack.go           # Slack integration
│   │
│   ├── service/                   # Service layer
│   │   └── task_service.go        # High-level task API
│   │
│   ├── worker/                    # Worker system
│   │   └── worker.go              # Task handlers
│   │
│   ├── tools/                     # Tool installation
│   │   └── installer.go           # mise-based installer
│   │
│   ├── network/                   # Network management
│   │   └── network.go             # TAP device + bridge
│   │
│   ├── api/                       # API models
│   │   └── models.go              # Request/response types
│   │
│   ├── types/                     # Domain types
│   │   └── types.go               # VM, Task, LogEntry, etc.
│   │
│   ├── config/                    # Configuration
│   │   └── config.go              # YAML config loader
│   │
│   └── container/                 # DI container (not actively used)
│       ├── container.go           # Container implementation
│       └── factories/             # Component factories
│
├── migrations/                    # Database migrations
│   └── 000001_initial_schema.up.sql
│
├── scripts/                       # Utility scripts
│   ├── setup-network.sh           # Network setup
│   ├── prepare-rootfs-with-tools.sh
│   └── run-integration-test.sh
│
├── tests/integration/             # Integration tests
│   ├── three_vm_clone_test.go
│   └── tools_test.go
│
└── docs/                          # Documentation
    ├── API-GATEWAY.md
    ├── TOOLS-AND-PROVISIONING.md
    ├── INTEGRATIONS.md
    └── PRODUCTION-ARCHITECTURE.md
```

## Quick Development Guide

### Building

```bash
# Build all binaries
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/worker ./cmd/worker
go build -o bin/aether-cli ./cmd/aether-cli
go build -o bin/fc-agent ./cmd/fc-agent
```

### Running Locally

```bash
# 1. Start infrastructure
docker run -d --name aetherium-postgres \
  -e POSTGRES_USER=aetherium \
  -e POSTGRES_PASSWORD=aetherium \
  -e POSTGRES_DB=aetherium \
  -p 5432:5432 postgres:15-alpine

docker run -d --name aetherium-redis \
  -p 6379:6379 redis:7-alpine

# 2. Run migrations
go run ./cmd/migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" -path ./migrations up

# 3. Prepare rootfs (one-time setup)
sudo ./scripts/prepare-rootfs-with-tools.sh

# 4. Start worker (needs sudo for Firecracker/network)
sudo ./bin/worker

# 5. Start API Gateway (different terminal)
./bin/api-gateway

# 6. Submit tasks
./bin/aether-cli -type vm:create -name test-vm -vcpus 1 -memory 256
```

### Testing

```bash
# Unit tests
go test ./pkg/...

# Integration tests (needs sudo)
cd tests/integration
sudo go test -v -timeout 30m

# Or use the integration test script
sudo ./scripts/run-integration-test.sh
```

### Setup Scripts

The `scripts/` directory contains production setup and utility scripts:

**Installation & Setup:**
```bash
# Install Firecracker binary (v1.7.0)
sudo ./scripts/install-firecracker.sh

# Download kernel with vsock support (v5.10.239)
sudo ./scripts/download-vsock-kernel.sh

# Setup Firecracker environment
sudo ./scripts/setup-firecracker.sh

# Setup FC agent in rootfs
sudo ./scripts/setup-fc-agent.sh

# Prepare rootfs with pre-installed tools (git, nodejs, bun, mise)
sudo ./scripts/prepare-rootfs-with-tools.sh
```

**Network Setup:**
```bash
# Setup aetherium0 bridge and NAT
sudo ./scripts/setup-network.sh

# Create pool of pre-configured TAP devices (allows worker to run with reduced privileges)
sudo ./scripts/create-tap-pool.sh
```

**Runtime:**
```bash
# Start worker with proper permissions
sudo ./scripts/start-worker.sh
```

## Debugging Tips

### Check VM logs

```bash
# Firecracker logs
cat /tmp/aetherium-vm-{vm-id}.sock.log

# VM console output (if configured)
cat /tmp/aetherium-vm-{vm-id}.sock.log
```

### Check database state

```bash
# List VMs
docker exec aetherium-postgres psql -U aetherium -c \
  "SELECT id, name, status FROM vms ORDER BY created_at DESC;"

# List executions
docker exec aetherium-postgres psql -U aetherium -c \
  "SELECT vm_id, command, exit_code FROM executions ORDER BY started_at DESC LIMIT 10;"
```

### Check Redis queue

```bash
docker exec aetherium-redis redis-cli
> KEYS asynq:*
> LLEN asynq:queues:default
```

### Network debugging

```bash
# Check bridge
ip addr show aetherium0

# Check TAP devices
ip link show | grep aether-

# Check NAT rules
sudo iptables -t nat -L -n -v
```

### Vsock debugging

```bash
# Check vhost-vsock module
lsmod | grep vhost_vsock
ls -l /dev/vhost-vsock

# Check if VM agent is listening
# (from inside VM)
ss -lnp | grep 9999
```

## Common Issues

### 1. Worker can't create VMs

**Symptom:** "Permission denied" or "failed to create TAP device"

**Solution:** Worker needs network privileges
```bash
# Option 1: Run with sudo
sudo ./bin/worker

# Option 2: Grant capability
sudo setcap cap_net_admin+ep ./bin/worker

# Option 3: Setup network beforehand
sudo ./scripts/setup-network.sh
./bin/worker
```

### 2. Vsock connection timeout

**Symptom:** "Cannot connect to VM agent via vsock"

**Solution:** 
- Check kernel has vsock support: `/var/firecracker/vmlinux`
- Check agent is running in VM: `ss -lnp | grep 9999`
- Check vhost-vsock module: `lsmod | grep vhost_vsock`

### 3. Database connection failed

**Symptom:** "failed to connect to database"

**Solution:**
- Check PostgreSQL is running: `docker ps`
- Check connection string in config
- Run migrations: `go run ./cmd/migrate ...`

### 4. Redis queue not processing

**Symptom:** Tasks enqueued but not processed

**Solution:**
- Check Redis is running: `docker ps`
- Check worker registered handlers: Look for logs
- Check worker is running: `ps aux | grep worker`

## Extending the System

### Adding a New Task Type

1. Define task type in `/pkg/queue/queue.go`:
```go
const TaskTypeMyTask TaskType = "my:task"
```

2. Create handler in `/pkg/worker/worker.go`:
```go
func (w *Worker) HandleMyTask(ctx, task) (*TaskResult, error) {
    var payload MyTaskPayload
    queue.UnmarshalPayload(task.Payload, &payload)
    
    // Do work
    
    return &TaskResult{Success: true, Result: {...}}, nil
}
```

3. Register handler:
```go
q.RegisterHandler(TaskTypeMyTask, w.HandleMyTask)
```

4. Add service method in `/pkg/service/task_service.go`:
```go
func (s *TaskService) CreateMyTask(ctx, ...) (uuid.UUID, error) {
    task := &queue.Task{
        ID:   uuid.New(),
        Type: TaskTypeMyTask,
        Payload: {...},
    }
    return task.ID, s.queue.Enqueue(ctx, task, opts)
}
```

### Adding a New Integration

1. Implement interface in `/pkg/integrations/{name}/{name}.go`:
```go
type MyIntegration struct {
    config Config
}

func (m *MyIntegration) Name() string { return "my-integration" }
func (m *MyIntegration) Initialize(ctx, config) error { ... }
func (m *MyIntegration) SendNotification(ctx, notif) error { ... }
// etc.
```

2. Register in API Gateway:
```go
myInt := myintegration.NewMyIntegration()
myInt.Initialize(ctx, config)
registry.Register(myInt)
```

## Performance Characteristics

- **VM Creation:** ~10-15 seconds (with tool installation)
- **Command Execution:** <1 second (local vsock)
- **Task Latency:** <500ms (Redis queue overhead)
- **Concurrent VMs:** Limited by host CPU/memory
- **Network Throughput:** ~1 Gbps (TAP bridge)

## Security Considerations

1. **VM Isolation:** Hardware-level via KVM
2. **Network Isolation:** Bridge + iptables
3. **Vsock:** No network exposure
4. **Sudo Required:** Worker needs network privileges
5. **API Authentication:** JWT/API keys (in API Gateway)
6. **Integration Secrets:** Environment variables

## Key Dependencies

```
github.com/firecracker-microvm/firecracker-go-sdk  # Official Firecracker SDK
github.com/hibiken/asynq                           # Redis task queue
github.com/jmoiron/sqlx                           # SQL extensions
github.com/lib/pq                                  # PostgreSQL driver
github.com/go-chi/chi/v5                          # HTTP router
github.com/google/uuid                             # UUID generation
github.com/mdlayher/vsock                         # Vsock communication
```

## Resources

- Firecracker: https://github.com/firecracker-microvm/firecracker
- Asynq: https://github.com/hibiken/asynq
- Vsock: https://github.com/firecracker-microvm/firecracker/blob/main/docs/vsock.md

---

**When in doubt, read the code. The interfaces are self-documenting.**

**Key files to understand the system:**
1. `/pkg/types/types.go` - Domain types
2. `/pkg/worker/worker.go` - Task execution logic
3. `/pkg/vmm/firecracker/firecracker.go` - VM lifecycle
4. `/pkg/vmm/firecracker/exec.go` - Command execution
5. `/cmd/api-gateway/main.go` - API routing
