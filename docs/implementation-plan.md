# Aetherium Implementation Plan

**Version**: 1.0
**Date**: 2025-10-04
**Status**: In Progress

## Executive Summary

Aetherium is a plugin-based, interface-driven orchestration platform for running autonomous AI agents in isolated Firecracker microVMs. The system prioritizes **flexibility**, **security**, and **extensibility** through a clean separation of concerns between the Control Plane and Execution Plane.

### Core Design Principles

1. **Dependency Inversion**: All major components depend on abstractions, not concrete implementations
2. **Plugin Architecture**: Hot-swappable components via configuration
3. **Event-Driven Integration**: Decoupled integrations via pub/sub event bus
4. **Configuration-Driven**: Runtime selection of providers without code changes

---

## Architecture Overview

### High-Level Components

```
┌─────────────────────────────────────────────────────────────┐
│                      Control Plane                          │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ API Gateway  │  │ Orchestrator │  │ GitHub Service  │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
│         │                  │                    │           │
│         └──────────────────┴────────────────────┘           │
│                            │                                 │
│              ┌─────────────┴─────────────┐                  │
│              │   Event Bus + State DB    │                  │
│              └───────────────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                     Execution Plane                         │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ Agent Worker │  │ Agent Worker │  │ Agent Worker    │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
│         │                  │                    │           │
│         ▼                  ▼                    ▼           │
│  ┌──────────┐      ┌──────────┐        ┌──────────┐       │
│  │ FC MicroVM│      │ FC MicroVM│        │ FC MicroVM│       │
│  └──────────┘      └──────────┘        └──────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### Pluggable Interfaces

| Component | Interface | Implementations |
|-----------|-----------|-----------------|
| **Task Queue** | `TaskQueue` | Redis (Asynq), RabbitMQ, Kafka, Memory |
| **State Storage** | `StateStore` | PostgreSQL, MySQL, MongoDB, DynamoDB |
| **Logging** | `Logger` | Loki, Elasticsearch, CloudWatch, Stdout |
| **VMM** | `VMOrchestrator` | Firecracker, Kata, Docker, QEMU |
| **Event Bus** | `EventBus` | Memory, Redis Pub/Sub, NATS, Kafka |
| **Integrations** | `Integration` | GitHub, Slack, Discord, Linear, Jira |

---

## Implementation Phases

### **Phase 1: Core Abstractions** ✅ COMPLETED

**Timeline**: Week 1-2
**Status**: ✅ Done

#### Deliverables
- [x] Define all core interfaces
  - `pkg/queue/interface.go`
  - `pkg/storage/interface.go`
  - `pkg/logging/interface.go`
  - `pkg/vmm/interface.go`
  - `pkg/events/interface.go`
  - `pkg/integrations/interface.go`
- [x] Configuration system (YAML + env vars)
  - `pkg/config/config.go`
  - `config/config.yaml`
  - `config/config.dev.yaml`
- [x] Project structure
  - Go modules + Zig build system
  - Makefile for build automation

---

### **Phase 2: MVP Implementations**

**Timeline**: Week 3-6
**Status**: 🚧 In Progress

#### 2.1 In-Memory Providers (Testing)
**Files**:
- `pkg/queue/memory/memory.go`
- `pkg/storage/memory/memory.go`
- `pkg/logging/stdout/stdout.go`
- `pkg/events/memory/memory.go`

**Purpose**: Enable testing without external dependencies

#### 2.2 Production Providers
**Redis Task Queue** (`pkg/queue/redis/`)
```go
type RedisQueue struct {
    client *asynq.Client
    server *asynq.Server
}
```

**PostgreSQL Storage** (`pkg/storage/postgres/`)
```go
type PostgresStore struct {
    db *pgxpool.Pool
}
```
- Schema: `projects`, `tasks`, `vms`, `agent_history`
- Migrations with `golang-migrate`

**Loki Logger** (`pkg/logging/loki/`)
```go
type LokiLogger struct {
    client *loki.Client
}
```

**Firecracker VMM** (`pkg/vmm/firecracker/`)
```go
// #cgo LDFLAGS: -L../../internal/firecracker/zig-out/lib -lfirecracker_vmm
// #include "../../internal/firecracker/cgo/firecracker.h"
import "C"

type FirecrackerOrchestrator struct {
    socketDir string
}
```

#### 2.3 Zig Firecracker Implementation
**Files**:
- `internal/firecracker/src/vmm.zig` - VM lifecycle
- `internal/firecracker/src/api.zig` - Firecracker HTTP API client
- `internal/firecracker/cgo/firecracker.h` - C bindings

**Key Functions**:
- `fc_create_vm(config_json, len) -> *VM`
- `fc_start_vm(vm) -> bool`
- `fc_stop_vm(vm, force) -> bool`
- `fc_destroy_vm(vm)`

---

### **Phase 3: Integration Framework**

**Timeline**: Week 7-9
**Status**: 📋 Planned

#### 3.1 Event Bus Implementation
**Memory Event Bus** (`pkg/events/memory/`)
```go
type MemoryEventBus struct {
    subscribers map[string][]subscriber
    mu          sync.RWMutex
}
```

#### 3.2 Integration SDK
**Base Integration** (`pkg/integrations/sdk/`)
```go
type BaseIntegration struct {
    name     string
    eventBus events.EventBus
    logger   logging.Logger
}

func (b *BaseIntegration) SubscribeToEvents(topics ...string) error
func (b *BaseIntegration) EmitEvent(event *Event) error
```

#### 3.3 GitHub Integration
**Files**: `pkg/integrations/github/`
- `github.go` - Main integration
- `pr.go` - Pull request creation
- `webhook.go` - Webhook handler

**Features**:
- Create PRs from agent output
- Handle `/aetherium run` commands in PR comments
- Webhook signature validation

#### 3.4 Slack Integration
**Files**: `pkg/integrations/slack/`
- `slack.go` - Main integration
- `commands.go` - Slash command handler
- `notifications.go` - Message posting

**Features**:
- Post task status updates to channels
- Handle `/aetherium` slash commands
- Interactive buttons for task control

---

### **Phase 4: Control Plane Services**

**Timeline**: Week 10-12
**Status**: 📋 Planned

#### 4.1 DI Container & Factory
**File**: `pkg/container/container.go`

```go
type Container struct {
    Config       *config.Config
    TaskQueue    queue.TaskQueue
    StateStore   storage.StateStore
    Logger       logging.Logger
    VMOrch       vmm.VMOrchestrator
    EventBus     events.EventBus
    Integrations *integrations.Registry
}

func NewContainer(cfg *config.Config) (*Container, error)
```

**Factory Pattern** (`pkg/factory/`)
```go
func CreateTaskQueue(cfg config.ProviderConfig) (queue.TaskQueue, error)
func CreateStateStore(cfg config.ProviderConfig) (storage.StateStore, error)
func CreateLogger(cfg config.ProviderConfig) (logging.Logger, error)
func CreateVMOrchestrator(cfg config.ProviderConfig) (vmm.VMOrchestrator, error)
```

#### 4.2 API Gateway
**File**: `cmd/api-gateway/main.go`

**Endpoints**:
```
POST   /projects              - Create project
GET    /projects/:id          - Get project
POST   /tasks                 - Start agent task
GET    /tasks/:id             - Get task status
DELETE /tasks/:id             - Terminate task
GET    /tasks/:id/logs        - Stream logs
POST   /webhooks/github       - GitHub webhook
GET    /health                - Health check
```

**Tech Stack**:
- Gin/Echo for HTTP routing
- Middleware: Auth, CORS, rate limiting
- OpenAPI/Swagger docs

#### 4.3 Task Orchestrator
**File**: `cmd/orchestrator/main.go`

**Responsibilities**:
1. Receive task requests from API Gateway
2. Create task record in StateStore
3. Publish task to TaskQueue
4. Emit `task.created` event

**Workflow**:
```go
func (o *Orchestrator) CreateTask(ctx context.Context, req *TaskRequest) error {
    task := &types.Task{
        ID:        uuid.New().String(),
        Status:    types.TaskStatusPending,
        CreatedAt: time.Now(),
    }

    if err := o.store.CreateTask(ctx, task); err != nil {
        return err
    }

    if err := o.queue.Enqueue(ctx, task); err != nil {
        return err
    }

    o.eventBus.Publish(ctx, events.TopicTaskCreated, &types.Event{
        Type: "task.created",
        Data: map[string]interface{}{"task_id": task.ID},
    })

    return nil
}
```

---

### **Phase 5: Execution Plane**

**Timeline**: Week 13-14
**Status**: 📋 Planned

#### 5.1 Agent Worker
**File**: `cmd/worker/main.go`

**Workflow**:
1. Dequeue task from TaskQueue
2. Update status to `RUNNING`
3. Create VM via VMOrchestrator
4. Start VM and monitor
5. Collect output
6. Send to GitHub Integration
7. Update status to `COMPLETED`
8. Emit `task.completed` event

```go
func (w *Worker) ProcessTask(ctx context.Context, task *types.Task) error {
    // Update status
    task.Status = types.TaskStatusRunning
    w.store.UpdateTask(ctx, task)

    // Create VM
    vmConfig := &types.VMConfig{
        ID:         fmt.Sprintf("vm-%s", task.ID),
        VCPUCount:  2,
        MemoryMB:   512,
    }

    vm, err := w.vmm.CreateVM(ctx, vmConfig)
    if err != nil {
        return err
    }

    // Start VM
    if err := w.vmm.StartVM(ctx, vm.ID); err != nil {
        return err
    }

    // Stream logs
    logs, _ := w.vmm.StreamLogs(ctx, vm.ID)
    for log := range logs {
        w.logger.Info(ctx, log, map[string]interface{}{
            "task_id": task.ID,
            "vm_id": vm.ID,
        })
    }

    // Collect output and create PR
    // ...

    task.Status = types.TaskStatusCompleted
    w.store.UpdateTask(ctx, task)

    return nil
}
```

---

## Technical Specifications

### Database Schema

**projects**
```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repo_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

**tasks**
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    parent_task_id UUID REFERENCES tasks(id),
    status VARCHAR(50) NOT NULL,
    agent_type VARCHAR(100) NOT NULL,
    prompt TEXT NOT NULL,
    container_id VARCHAR(255),
    vm_id VARCHAR(255),
    pull_request TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP
);
```

**vms**
```sql
CREATE TABLE vms (
    id VARCHAR(255) PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    metadata JSONB
);
```

---

## Configuration Examples

### Swap Redis → RabbitMQ
```yaml
task_queue:
  provider: rabbitmq  # Changed from: redis
  config:
    rabbitmq:
      url: amqp://guest:guest@localhost:5672/
```

### Switch Docker → Firecracker
```yaml
vmm:
  provider: firecracker  # Changed from: docker
  config:
    firecracker:
      kernel_path: /var/firecracker/vmlinux
      socket_dir: /var/run/firecracker
```

### Add Discord Integration
```yaml
integrations:
  enabled:
    - github
    - slack
    - discord  # Added

  discord:
    token: ${DISCORD_BOT_TOKEN}
    guild_id: ${DISCORD_GUILD_ID}
```

Then implement:
```go
// pkg/integrations/discord/discord.go
type DiscordIntegration struct {
    *sdk.BaseIntegration
    client *discordgo.Session
}

func (d *DiscordIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
    // Handle events
}
```

---

## Testing Strategy

### Unit Tests
- Test each interface implementation in isolation
- Use in-memory providers for dependencies
- Mock external services (GitHub API, Firecracker)

### Integration Tests
- Test Control Plane → Execution Plane flow
- Test event bus pub/sub
- Test VMM lifecycle with Docker (faster than Firecracker)

### End-to-End Tests
- Full workflow: API → Queue → Worker → VM → Integration
- Test with real Firecracker VMs in CI/CD
- Test webhook flows

---

## Security Considerations

### Firecracker Isolation
- Each VM runs in isolated microVM with dedicated kernel
- jailer for additional sandboxing
- Resource limits (CPU, memory, I/O)
- Network isolation via TAP devices

### Credential Management
- Secrets injected via Firecracker MMDS (Metadata Service)
- API keys never stored in VM images
- Ephemeral credentials per task

### API Security
- JWT/OAuth2 authentication
- Rate limiting per client
- Webhook signature validation
- HTTPS only in production

---

## Deployment

### Docker Compose (Development)
```yaml
services:
  postgres:
    image: postgres:15
  redis:
    image: redis:7
  loki:
    image: grafana/loki:latest

  api-gateway:
    build: .
    command: /app/bin/api-gateway
    depends_on: [postgres, redis]

  orchestrator:
    build: .
    command: /app/bin/orchestrator
    depends_on: [postgres, redis]

  worker:
    build: .
    command: /app/bin/worker
    privileged: true  # For Firecracker
    depends_on: [postgres, redis]
```

### Kubernetes (Production)
- Control Plane: Standard Deployments
- Execution Plane: DaemonSet on bare-metal nodes (for KVM)
- Persistent volumes for VM images
- Horizontal Pod Autoscaling for workers

---

## Monitoring & Observability

### Metrics (Prometheus)
- Task queue depth
- Task processing time
- VM creation/startup time
- API request rate & latency
- Resource utilization per VM

### Logs (Loki)
- Centralized logs with labels: `task_id`, `vm_id`, `project_id`
- Real-time log streaming via WebSocket
- Log retention policies

### Tracing (Jaeger/Tempo)
- Distributed tracing across services
- Trace task flow: API → Queue → Worker → VM → Integration

---

## Success Metrics

### Phase 1-2 (MVP)
- [ ] All interfaces defined
- [ ] Configuration system working
- [ ] In-memory providers functional
- [ ] One production provider per interface (Redis, Postgres, Loki, Docker)

### Phase 3-4 (Control Plane)
- [ ] API Gateway serving requests
- [ ] Task Orchestrator enqueueing tasks
- [ ] GitHub integration creating PRs
- [ ] Event bus routing events

### Phase 5 (Execution Plane)
- [ ] Worker processing tasks end-to-end
- [ ] Docker VMs executing agent code
- [ ] Logs streaming to Loki
- [ ] Full task lifecycle: API → VM → PR

### Future (Production)
- [ ] Firecracker VMM operational
- [ ] Multi-worker horizontal scaling
- [ ] 5+ integrations (GitHub, Slack, Discord, Linear, Jira)
- [ ] Web dashboard for monitoring

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Firecracker setup complexity | High | Start with Docker, migrate to Firecracker in Phase 5 |
| Integration API changes | Medium | Version all integrations, use adapters |
| Task queue failures | High | Use persistent queues (Redis/RabbitMQ), retry logic |
| VM resource exhaustion | High | Resource quotas, worker autoscaling, monitoring |

---

## Next Steps (Immediate)

1. ✅ Complete configuration system
2. 🚧 Implement DI container + factory pattern
3. 🚧 Build in-memory providers for testing
4. 📋 Implement Redis task queue
5. 📋 Implement PostgreSQL state store
6. 📋 Build API Gateway skeleton

---

## References

- Original Design Doc: `docs/design.md`
- Firecracker Docs: https://github.com/firecracker-microvm/firecracker
- Asynq (Task Queue): https://github.com/hibiken/asynq
- Grafana Loki: https://grafana.com/docs/loki/
- Go-GitHub: https://github.com/google/go-github

---

**Last Updated**: 2025-10-04
**Maintainers**: Aetherium Team
