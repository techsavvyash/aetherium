# Aetherium Distributed Task Processing Demo

This demo showcases the distributed task processing capabilities of Aetherium, including VM creation, command execution, and result tracking.

## Architecture

```
┌─────────────────┐      ┌─────────────────┐
│  Task Submitter │──────▶│   Redis Queue   │
└─────────────────┘      └─────────────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │     Worker      │
                         └─────────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
           ┌─────────────┐ ┌─────────┐ ┌──────────────┐
           │ Firecracker │ │  Vsock  │ │  PostgreSQL  │
           │     VMM     │ │  Agent  │ │   Storage    │
           └─────────────┘ └─────────┘ └──────────────┘
```

## Components

### 1. **Infrastructure**
- **PostgreSQL**: Stores VM state, tasks, jobs, and execution history
- **Redis**: Distributed task queue with Asynq
- **Firecracker**: Lightweight VM runtime with vsock communication

### 2. **Worker** (`cmd/worker-demo`)
- Consumes tasks from Redis queue
- Creates and manages Firecracker VMs
- Executes commands via vsock
- Stores results in PostgreSQL

### 3. **Task Submitter** (`cmd/task-submit`)
- CLI tool to submit tasks to the queue
- Supports VM creation, execution, and deletion

## Quick Start

### Prerequisites

- Docker (for PostgreSQL and Redis)
- Firecracker installed
- Rootfs with fc-agent deployed
- Vsock kernel support (vhost-vsock module)

### Setup

Run the automated demo setup:

```bash
chmod +x scripts/demo-distributed-tasks.sh
./scripts/demo-distributed-tasks.sh
```

This will:
1. Start PostgreSQL and Redis containers
2. Run database migrations
3. Build all binaries
4. Configure vsock
5. Deploy agent to rootfs

### Running the Demo

#### Terminal 1: Start the Worker

```bash
./bin/worker-demo
```

Expected output:
```
Worker starting...
Registered handlers for: vm:create, vm:execute, vm:delete
Listening for tasks on Redis queue...
```

#### Terminal 2: Submit Tasks

**Create a VM:**
```bash
./bin/task-submit -type vm:create -name demo-vm
```

Output:
```
✓ VM creation task submitted: <task-id>
  Name: demo-vm
  Task will create a Firecracker VM with 1 vCPU and 256MB RAM
```

Worker will show:
```
Creating VM: demo-vm (vcpu=1, mem=256MB)
✓ VM created: <vm-id>
✓ VM started
Task <task-id> completed in 3.2s
```

**Execute Commands:**

Simple echo:
```bash
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd echo -args "Hello from VM"
```

Clone a repository:
```bash
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd git -args "clone https://github.com/torvalds/linux"
```

List files:
```bash
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd ls -args "-la"
```

**Delete VM:**
```bash
./bin/task-submit -type vm:delete -vm-id <vm-id>
```

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

### `vm:delete`
Stops and deletes a VM.

**Payload:**
```json
{
  "vm_id": "uuid"
}
```

## Database Schema

### VMs Table
```sql
CREATE TABLE vms (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    orchestrator VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    kernel_path VARCHAR(500),
    rootfs_path VARCHAR(500),
    socket_path VARCHAR(500),
    vcpu_count INTEGER,
    memory_mb INTEGER,
    created_at TIMESTAMP,
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    metadata JSONB
);
```

### Tasks Table
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    priority INTEGER,
    payload JSONB NOT NULL,
    result JSONB,
    error TEXT,
    vm_id UUID REFERENCES vms(id),
    worker_id VARCHAR(255),
    max_retries INTEGER,
    retry_count INTEGER,
    created_at TIMESTAMP,
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    metadata JSONB
);
```

### Executions Table
```sql
CREATE TABLE executions (
    id UUID PRIMARY KEY,
    job_id UUID,
    vm_id UUID REFERENCES vms(id),
    command VARCHAR(500) NOT NULL,
    args JSONB,
    env JSONB,
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    error TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    metadata JSONB
);
```

## Monitoring

### View VMs in Database

```bash
docker exec -it aetherium-postgres psql -U aetherium -c "SELECT id, name, status, created_at FROM vms;"
```

### View Tasks

```bash
docker exec -it aetherium-postgres psql -U aetherium -c "SELECT id, type, status, created_at FROM tasks ORDER BY created_at DESC LIMIT 10;"
```

### View Executions

```bash
docker exec -it aetherium-postgres psql -U aetherium -c "SELECT vm_id, command, exit_code, started_at FROM executions ORDER BY started_at DESC LIMIT 10;"
```

### Redis Queue Stats

```bash
docker exec -it aetherium-redis redis-cli INFO
```

## Example Workflow: Git Clone

This example demonstrates cloning a public GitHub repository inside a VM:

```bash
# 1. Create VM
./bin/task-submit -type vm:create -name git-demo
# Note the VM ID from output

# 2. Install git (if not in rootfs)
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd apt-get -args "update"
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd apt-get -args "install -y git"

# 3. Clone repository
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd git -args "clone https://github.com/torvalds/linux"

# 4. List cloned files
./bin/task-submit -type vm:execute -vm-id <vm-id> -cmd ls -args "-la linux"

# 5. Clean up
./bin/task-submit -type vm:delete -vm-id <vm-id>
```

## Troubleshooting

### Worker not processing tasks

1. Check Redis is running:
   ```bash
   docker ps | grep aetherium-redis
   ```

2. Check PostgreSQL is running:
   ```bash
   docker ps | grep aetherium-postgres
   ```

3. Check worker logs for errors

### VM creation fails

1. Verify Firecracker is installed:
   ```bash
   which firecracker
   ```

2. Check vsock module:
   ```bash
   lsmod | grep vhost_vsock
   ```

3. Verify rootfs exists:
   ```bash
   ls -lh /var/firecracker/rootfs.ext4
   ```

### Command execution timeout

1. Check agent is deployed:
   ```bash
   sudo ./scripts/check-agent-in-rootfs.sh
   ```

2. Verify vsock communication:
   ```bash
   ./scripts/diagnose-vsock.sh
   ```

3. Check VM logs:
   ```bash
   cat /tmp/aetherium-vm-*.sock.log
   ```

## Cleanup

Stop infrastructure:
```bash
docker stop aetherium-postgres aetherium-redis
```

Clean up VM sockets:
```bash
rm -f /tmp/aetherium-vm-*.sock*
```

## Next Steps

- Add API Gateway for REST/GraphQL endpoints
- Implement job orchestration for multi-step workflows
- Add GitHub/Slack integrations
- Build monitoring dashboard
- Implement resource quotas and limits

## Configuration

Edit `config/example.yaml` to customize:
- Database connection
- Redis connection
- Queue concurrency
- VM defaults (CPU, memory)
- Logging options
