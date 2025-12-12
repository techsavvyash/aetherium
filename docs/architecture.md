# Architecture Overview

## High-Level Design

```
┌──────────────┐
│   Client     │
│ (REST/CLI)   │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────┐
│      API Gateway (port 8080)            │
│  - REST endpoints                       │
│  - Integration webhooks                 │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│        Redis Queue (Asynq)              │
│  - Task distribution                    │
│  - Priority queues                      │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│       Worker Process                    │
│  - Pulls tasks from queue               │
│  - Delegates to VM orchestrator         │
│  - Stores results in PostgreSQL         │
└──────┬──────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│   Firecracker Orchestrator              │
│  - VM lifecycle (create/start/stop)     │
│  - Command execution via vsock          │
│  - Network setup (TAP + bridge)         │
└─────────────────────────────────────────┘
```

## Components

### API Gateway

**Location:** `services/gateway/`

Entry point for all operations:
- REST API on port 8080
- Integration webhooks (GitHub, Slack)
- Request validation and routing

Endpoints:
- `POST /api/v1/vms` - Create VM
- `GET /api/v1/vms` - List VMs
- `DELETE /api/v1/vms/{id}` - Delete VM
- `POST /api/v1/vms/{id}/execute` - Execute command
- `POST /api/v1/webhooks/{integration}` - Integration webhooks

### Task Queue

**Location:** `services/core/pkg/queue/`

Redis-backed async task processing using Asynq:
- Task types: `vm:create`, `vm:execute`, `vm:delete`
- Priority queues: critical, high, default, low
- Task persistence and retry logic

### Worker

**Location:** `services/core/pkg/worker/`

Processes tasks from queue:
- Handles `vm:create` - Creates and starts VM
- Handles `vm:execute` - Executes commands in VM
- Handles `vm:delete` - Stops and deletes VM
- Stores results in PostgreSQL

### VM Orchestrator

**Location:** `services/core/pkg/vmm/`

VM lifecycle management:
- Firecracker implementation (production)
- Docker implementation (testing)
- Network configuration (TAP devices, bridge)
- Tool installation

### Storage

**Location:** `services/core/pkg/storage/`

PostgreSQL persistence:
- VMs table - VM metadata and status
- Tasks table - Task queue state
- Executions table - Command execution history
- Jobs table - Multi-command jobs

### Logging

**Location:** `libs/common/pkg/logging/`

Centralized logging:
- Loki implementation (production)
- Stdout implementation (development)

### Integrations

**Location:** `services/gateway/pkg/integrations/`

Plugin system for external integrations:
- GitHub - PR automation, webhooks
- Slack - Notifications, slash commands
- Discord - Notifications (future)

## Task Flow

### Create VM

```
1. Client POST /api/v1/vms
2. API Gateway validates request
3. TaskService.CreateVMTask()
4. Task enqueued to Redis
5. Worker pulls task
6. VMOrchestrator.CreateVM()
   - Create TAP device
   - Configure network
   - Boot VM
7. ToolInstaller.InstallTools()
   - Install git, nodejs, bun, claude-code
   - Install additional tools if requested
8. Store VM in PostgreSQL
9. Return success
```

### Execute Command

```
1. Client POST /api/v1/vms/{id}/execute
2. API Gateway validates request
3. TaskService.ExecuteCommandTask()
4. Task enqueued to Redis
5. Worker pulls task
6. VMOrchestrator.ExecuteCommand()
   - Connect via vsock
   - Send command JSON
   - Receive result JSON
7. Store execution in PostgreSQL
8. Return result (exit code, stdout, stderr)
```

### Delete VM

```
1. Client DELETE /api/v1/vms/{id}
2. API Gateway validates request
3. TaskService.DeleteVMTask()
4. Task enqueued to Redis
5. Worker pulls task
6. VMOrchestrator.DeleteVM()
   - Stop VM
   - Delete TAP device
   - Clean up sockets
7. Delete VM from PostgreSQL
8. Return success
```

## Network Architecture

Each VM gets its own TAP device connected to a bridge:

```
Host System
├── Bridge: aetherium0 (172.16.0.1/24)
│   ├── TAP1 -> VM1 (172.16.0.2)
│   ├── TAP2 -> VM2 (172.16.0.3)
│   └── TAP3 -> VM3 (172.16.0.4)
└── NAT -> Internet
```

Network Manager responsibilities:
1. Create bridge interface
2. Setup NAT for internet access
3. Allocate IPs from subnet
4. Create TAP device per VM
5. Attach TAP to bridge
6. Generate unique MAC addresses

## Command Execution (Vsock)

Host-VM communication via virtio-vsock:

```
Host                    Guest VM
vsock.Dial(3, 9999)     vsock.Listen(9999)
    │
    ├─→ Send JSON ─────→ Parse command
    │                    exec.Command()
    │
    ←─ Send result ←──── Capture output
```

Constants:
- Host CID: 2
- Guest CID: 3
- Agent Port: 9999

## Data Persistence

**PostgreSQL Schema:**

VMs table:
- id (UUID)
- name (unique)
- status (created, running, stopped, etc.)
- vcpu_count, memory_mb
- kernel_path, rootfs_path, socket_path
- created_at, started_at, stopped_at
- metadata (JSONB)

Tasks table:
- id (UUID)
- type (vm:create, vm:execute, vm:delete)
- status (pending, processing, completed, failed)
- payload (JSONB)
- result (JSONB)
- vm_id (FK)
- worker_id
- max_retries, retry_count
- created_at, scheduled_at, started_at, completed_at

Executions table:
- id (UUID)
- job_id (FK)
- vm_id (FK)
- command, args, env (JSONB)
- exit_code, stdout, stderr
- started_at, completed_at
- duration_ms

## Interfaces

All major components are interface-driven:

**VMOrchestrator:**
- CreateVM, StartVM, StopVM, DeleteVM
- GetVMStatus, ExecuteCommand, ListVMs
- Health check

**Queue:**
- Enqueue, RegisterHandler
- Start, Stop, Stats

**Storage:**
- VMs(), Tasks(), Jobs(), Executions()
- Close()

**Logger:**
- Log, Query, Close, Health

**EventBus:**
- Publish, Subscribe, Close, Health

**Integration:**
- Initialize, SendNotification, HandleEvent
- Health check

## Deployment Options

### Docker Compose (Development)

Single-host deployment with Docker Compose:
- API Gateway
- Worker (single)
- PostgreSQL
- Redis
- Loki (optional)

### Kubernetes (Production)

Distributed deployment with Helm:
- API Gateway (multiple replicas)
- Workers (multiple replicas)
- PostgreSQL (managed service)
- Redis (managed service)
- Loki (managed service)
- Horizontal autoscaling

### Pulumi (Infrastructure as Code)

Define infrastructure with Pulumi:
- Create cloud infrastructure
- Deploy services
- Configure databases
- Setup networking

## Performance

- VM Creation: ~10-15 seconds (with tools)
- Command Execution: <1 second (vsock)
- Task Latency: <500ms (Redis queue)
- Concurrent VMs: Limited by host CPU/memory
- Network Throughput: ~1 Gbps (TAP bridge)

## Security

- Hardware isolation via Firecracker microVMs
- Network isolation (bridge + iptables)
- Vsock communication (no network exposure)
- API authentication (JWT/API keys)
- Integration secrets (environment variables)
