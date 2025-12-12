# Documentation

## Quick Navigation

**Getting Started:**
- [Quick Start](quickstart.md) - 5-minute setup
- [Setup Guide](setup.md) - Detailed installation
- [Architecture Overview](architecture.md) - System design

**Development:**
- [Monorepo Guide](monorepo.md) - Project structure
- [API Reference](api-gateway.md) - REST API docs
- [Tool Installation](tools-provisioning.md) - Tool system

**Integration & Deployment:**
- [Integrations](integrations.md) - GitHub, Slack plugins
- [Kubernetes](kubernetes.md) - K8s deployment
- [Production Architecture](production-architecture.md) - Enterprise setup

**Troubleshooting:**
- [Vsock Connection](troubleshooting/vsock-connection.md)
- [VM Creation](troubleshooting/vm-creation.md)
- [Tool Installation](troubleshooting/tools.md)

## Core Concepts

**What is Aetherium?**

Distributed task execution platform for running isolated workloads in Firecracker microVMs. Think of it as a programmable orchestration engine with:
- VM lifecycle management (create, start, stop, delete)
- Command execution via vsock
- Task queue (Redis/Asynq)
- State persistence (PostgreSQL)
- REST API gateway
- Integration framework

**How It Works:**

```
Client Request
    ↓
API Gateway (port 8080)
    ↓
Task Queue (Redis)
    ↓
Worker Process
    ↓
Firecracker Orchestrator
    ↓
Isolated microVMs
```

## Common Tasks

**Create a VM:**
```bash
./bin/aether-cli -type vm:create -name my-vm
```

**Execute a command:**
```bash
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd git -args "clone,https://github.com/user/repo"
```

**Delete a VM:**
```bash
./bin/aether-cli -type vm:delete -vm-id $VM_ID
```

**List all VMs:**
```bash
curl http://localhost:8080/api/v1/vms | jq
```

## Architecture Layers

**API Layer:**
- REST endpoints for VM operations
- Integration webhooks (GitHub, Slack)
- Request validation and routing

**Queue Layer:**
- Redis-backed task distribution
- Task types: vm:create, vm:execute, vm:delete
- Priority-based processing

**Worker Layer:**
- Processes tasks from queue
- Manages VM lifecycle
- Executes commands via vsock

**VM Layer:**
- Firecracker microVMs (production)
- Docker containers (testing)
- Network via TAP devices + bridge

## Deployment Options

**Docker Compose (Development):**
- Single host with all services
- PostgreSQL, Redis, Loki included
- Minimal setup required

**Kubernetes (Production):**
- Distributed across cluster
- Multiple API Gateway replicas
- Multiple Worker replicas
- Auto-scaling support

**Pulumi (Infrastructure as Code):**
- Define cloud infrastructure
- Automated deployment
- Multi-environment support

## Key Files

| Path | Purpose |
|------|---------|
| `services/core/` | VM orchestration & lifecycle |
| `services/gateway/` | REST API & integrations |
| `services/dashboard/` | Web UI (optional) |
| `libs/common/` | Shared utilities |
| `libs/types/` | Shared domain types |
| `infrastructure/` | Deployment configs |
| `migrations/` | Database schemas |

## Performance

- VM Creation: 10-15 seconds (with tools)
- Command Execution: <1 second (vsock)
- Task Processing: <500ms (queue overhead)
- Concurrent VMs: Limited by host resources

## Security

- Hardware isolation via Firecracker
- Network isolation (bridge + iptables)
- Vsock communication (no network exposure)
- API authentication (JWT/API keys)
- Secrets via environment variables

## Getting Help

1. Check troubleshooting guides
2. Review relevant documentation files
3. Check logs in `/tmp/` or `/var/log/`
4. Run diagnostic scripts in `scripts/`

## Useful Links

- [GitHub Repository](https://github.com/aetherium/aetherium)
- [Firecracker Documentation](https://github.com/firecracker-microvm/firecracker)
- [Asynq Task Queue](https://github.com/hibiken/asynq)
